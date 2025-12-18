package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

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
		db, err := sql.Open("pgx", coreCfg.PostgresDSN)
		if err != nil {
			log.Fatal(err)
		}
		defer db.Close()
		if err := db.PingContext(ctx); err != nil {
			log.Fatal(err)
		}
		store, err := utcp.NewPostgresStorage(db)
		if err != nil {
			log.Fatal(err)
		}
		if err := utcp.RegisterStorageTools(registry, store); err != nil {
			log.Fatal(err)
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
	riskEngine := app.NewRiskEngine(riskAdapter)

	var tradeStore app.TradeStore
	var tradeSaver app.Storage
	if coreCfg.PostgresDSN != "" {
		storeAdapter := adapters.NewUTCPStorageAdapter(storageSvc)
		tradeStore = storeAdapter
		tradeSaver = storeAdapter
	}

	var dexter app.Dexter
	if coreCfg.UseDexter {
		dexter = adapters.NewUTCPDexterAdapter(dexterSvc)
	}

	detector := app.NewEventDetector(marketAdapter)
	generator := app.NewTradeGenerator(marketAdapter, strategyMap)
	orchestrator := app.NewOrchestrator(detector, generator, riskEngine, tradeSaver, dexter)

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
