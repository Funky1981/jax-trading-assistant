package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

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
	if coreCfg.PostgresDSN != "" {
		storeAdapter := adapters.NewUTCPStorageAdapter(storageSvc)
		tradeStore = storeAdapter
		tradeSaver = storeAdapter
		auditLogger = app.NewAuditLogger(storeAdapter)
	}

	var dexter app.Dexter
	if coreCfg.UseDexter {
		dexter = adapters.NewUTCPDexterAdapter(dexterSvc)
	}

	riskEngine := app.NewRiskEngine(riskAdapter, auditLogger)
	detector := app.NewEventDetector(marketAdapter, auditLogger)
	generator := app.NewTradeGenerator(marketAdapter, strategyMap, auditLogger)
	orchestrator := app.NewOrchestrator(detector, generator, riskEngine, tradeSaver, dexter, auditLogger)

	server := httpapi.NewServer()
	server.RegisterHealth()
	server.RegisterRisk(riskEngine)
	server.RegisterStrategies(strategyRegistry)
	server.RegisterProcess(orchestrator, coreCfg.AccountSize, coreCfg.RiskPercent, 3)
	server.RegisterTrades(tradeStore)

	addr := fmt.Sprintf(":%d", coreCfg.HTTPPort)
	log.Printf("jax-core listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, server.Handler()))
}
