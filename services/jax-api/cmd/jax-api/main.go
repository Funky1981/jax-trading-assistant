package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"jax-trading-assistant/internal/strategyregistry"
	"jax-trading-assistant/libs/database"
	"jax-trading-assistant/libs/utcp"
	"jax-trading-assistant/services/jax-api/internal/app"
	"jax-trading-assistant/services/jax-api/internal/config"
	"jax-trading-assistant/services/jax-api/internal/infra/adapters"
	httpapi "jax-trading-assistant/services/jax-api/internal/infra/http"
)

func main() {
	var providersPath string
	var coreConfigPath string
	var strategiesDir string

	flag.StringVar(&providersPath, "providers", "config/providers.json", "Path to providers.json")
	flag.StringVar(&coreConfigPath, "config", "config/jax-core.json", "Path to jax-core.json")
	flag.StringVar(&strategiesDir, "strategies", "config/strategies", "Directory of strategy JSON files")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	coreCfg, err := config.LoadJaxCoreConfig(coreConfigPath)
	if err != nil {
		log.Fatal(err)
	}

	strategyMap, err := config.LoadStrategyConfigs(strategiesDir)
	if err != nil {
		log.Fatal(err)
	}
	strategyRegistry := app.NewStrategyRegistry(strategyMap)

	registry := utcp.NewLocalRegistry()
	if err := utcp.RegisterRiskTools(registry); err != nil {
		log.Fatal(err)
	}
	if err := utcp.RegisterBacktestTools(registry, utcp.NewBacktestEngine()); err != nil {
		log.Fatal(err)
	}

	var dbConn *database.DB
	if coreCfg.PostgresDSN != "" {
		// Configure database connection with pooling and retry logic
		dbConfig := database.DefaultConfig()
		dbConfig.DSN = coreCfg.PostgresDSN

		// Connect with migrations
		db, err := database.ConnectWithMigrations(ctx, dbConfig, "file://db/postgres/migrations")
		if err != nil {
			log.Fatalf("failed to connect to database: %v", err)
		}
		defer db.Close()
		dbConn = db

		log.Printf("database connected: max_open=%d, max_idle=%d, max_lifetime=%v",
			dbConfig.MaxOpenConns, dbConfig.MaxIdleConns, dbConfig.ConnMaxLifetime)

		// Create UTCP storage provider
		store, err := utcp.NewPostgresStorage(db.DB)
		if err != nil {
			log.Fatalf("failed to create storage: %v", err)
		}
		if err := utcp.RegisterStorageTools(registry, store); err != nil {
			log.Fatalf("failed to register storage tools: %v", err)
		}
	}

	// Initialize Knowledge Base (Strategy Registry from Postgres)
	// This connects to jax_knowledge database and loads approved documents.
	knowledgeRegistry, err := strategyregistry.NewFromDSN(ctx, coreCfg.KnowledgeDSN)
	if err != nil {
		log.Printf("WARNING: knowledge registry not available: %v", err)
		log.Printf("Run 'make knowledge-up && make knowledge-schema && make knowledge-ingest' to set up")
	} else {
		defer knowledgeRegistry.Close()

		// Health check
		if err := knowledgeRegistry.HealthCheck(ctx); err != nil {
			log.Printf("WARNING: knowledge registry health check failed: %v", err)
		} else {
			// Load and log counts ("it's alive" proof)
			counts, err := knowledgeRegistry.CountsByType(ctx)
			if err != nil {
				log.Printf("WARNING: failed to load knowledge counts: %v", err)
			} else {
				log.Printf("knowledge registry connected:")
				log.Printf("  strategies:    %d", counts[strategyregistry.DocTypeStrategy])
				log.Printf("  anti-patterns: %d", counts[strategyregistry.DocTypeAntiPattern])
				log.Printf("  patterns:      %d", counts[strategyregistry.DocTypePattern])
				log.Printf("  meta docs:     %d", counts[strategyregistry.DocTypeMeta])
				log.Printf("  risk docs:     %d", counts[strategyregistry.DocTypeRisk])
				log.Printf("  evaluation:    %d", counts[strategyregistry.DocTypeEvaluation])
			}
		}
	}

	client, err := utcp.NewUTCPClientFromFile(providersPath, utcp.WithLocalRegistry(registry))
	if err != nil {
		log.Fatal(err)
	}

	marketSvc := utcp.NewMarketDataService(client)
	riskSvc := utcp.NewRiskService(client)
	storageSvc := utcp.NewStorageService(client)
	dexterSvc := utcp.NewDexterService(client)

	marketAdapter := adapters.NewUTCPMarketAdapter(marketSvc)
	riskAdapter := adapters.NewUTCPRiskAdapter(riskSvc)
	var auditLogger *app.AuditLogger

	var tradeStore app.TradeStore
	var tradeSaver app.Storage
	var signalStore app.SignalStore
	if coreCfg.PostgresDSN != "" {
		storeAdapter := adapters.NewUTCPStorageAdapter(storageSvc)
		tradeStore = storeAdapter
		tradeSaver = storeAdapter
		auditLogger = app.NewAuditLogger(storeAdapter)

		// Initialize signal store with direct database access
		if dbConn != nil {
			signalStore = adapters.NewPostgresSignalStore(dbConn.DB)
		}
	}

	var dexter app.Dexter
	if coreCfg.UseDexter {
		dexter = adapters.NewUTCPDexterAdapter(dexterSvc)
	}

	riskEngine := app.NewRiskEngine(riskAdapter, auditLogger)
	detector := app.NewEventDetector(marketAdapter, auditLogger)
	generator := app.NewTradeGenerator(marketAdapter, strategyMap, auditLogger)
	tradingGuard := app.NewTradingGuard(app.TradingGuardConfig{MaxConsecutiveLosses: coreCfg.MaxConsecutiveLosses}, auditLogger)
	orchestrator := app.NewOrchestrator(detector, generator, riskEngine, tradeSaver, dexter, auditLogger, tradingGuard)

	server := httpapi.NewServer()
	server.RegisterHealth()
	server.RegisterAuth()
	server.RegisterMetrics()
	server.RegisterRisk(riskEngine)
	server.RegisterStrategies(strategyRegistry)
	server.RegisterProcess(orchestrator, coreCfg.AccountSize, coreCfg.RiskPercent, 3)
	server.RegisterTrades(tradeStore)
	server.RegisterTradingGuard(tradingGuard)
	server.RegisterSignals(signalStore)

	addr := fmt.Sprintf(":%d", coreCfg.HTTPPort)
	log.Printf("jax-core listening on %s", addr)
	log.Printf("Public endpoints: /health, /auth/login, /auth/refresh")
	log.Printf("Protected endpoints: /risk/*, /strategies, /trades, /symbols/*, /trading/*, /api/v1/metrics*, /api/v1/signals*")
	log.Fatal(http.ListenAndServe(addr, server.Handler()))
}
