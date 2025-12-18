package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"jax-trading-assistant/libs/utcp"
)

func main() {
	var providersPath string
	var postgresDSN string
	var skipStorage bool

	flag.StringVar(&providersPath, "providers", "config/providers.json", "Path to providers.json")
	flag.StringVar(&postgresDSN, "postgres", "", "Postgres DSN (e.g. postgres://user:pass@localhost:5432/jax?sslmode=disable)")
	flag.BoolVar(&skipStorage, "skip-storage", false, "Skip Postgres-backed storage smoke tests")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cfg, err := utcp.LoadProvidersConfig(providersPath)
	if err != nil {
		log.Fatal(err)
	}

	registry := utcp.NewLocalRegistry()
	if err := utcp.RegisterRiskTools(registry); err != nil {
		log.Fatal(err)
	}
	if err := utcp.RegisterBacktestTools(registry, utcp.NewBacktestEngine()); err != nil {
		log.Fatal(err)
	}

	var db *sql.DB
	if !skipStorage {
		if postgresDSN == "" {
			postgresDSN = os.Getenv("JAX_POSTGRES_DSN")
		}
		if postgresDSN == "" {
			log.Fatal("missing Postgres DSN: pass -postgres or set JAX_POSTGRES_DSN")
		}

		db, err = sql.Open("pgx", postgresDSN)
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

	client, err := utcp.NewUTCPClient(cfg, utcp.WithLocalRegistry(registry))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("== risk.position_size ==")
	riskSvc := utcp.NewRiskService(client)
	ps, err := riskSvc.PositionSize(ctx, utcp.PositionSizeInput{
		AccountSize: 10000,
		RiskPercent: 3,
		Entry:       100,
		Stop:        95,
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("positionSize=%d totalRisk=%.2f\n", ps.PositionSize, ps.TotalRisk)

	fmt.Println("== backtest.run_strategy ==")
	btSvc := utcp.NewBacktestService(client)
	run, err := btSvc.RunStrategy(ctx, utcp.RunStrategyInput{
		StrategyConfigID: "earnings_gap_v1",
		Symbols:          []string{"AAPL", "MSFT"},
		From:             time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		To:               time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("runId=%s sharpe=%.2f\n", run.RunID, run.Stats.Sharpe)

	if !skipStorage {
		fmt.Println("== storage.save_trade + storage.get_trade ==")
		storageSvc := utcp.NewStorageService(client)
		if err := storageSvc.SaveTrade(ctx, utcp.SaveTradeInput{
			Trade: utcp.StoredTrade{
				ID:         "ts_smoke_001",
				Symbol:     "AAPL",
				Direction:  "long",
				Entry:      100,
				Stop:       95,
				Targets:    []float64{110, 115},
				StrategyID: "earnings_gap_v1",
				Notes:      "smoke test",
			},
			Risk: &utcp.StoredRisk{
				PositionSize: ps.PositionSize,
				TotalRisk:    ps.TotalRisk,
			},
		}); err != nil {
			log.Fatal(err)
		}

		got, err := storageSvc.GetTrade(ctx, "ts_smoke_001")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("tradeId=%s symbol=%s\n", got.Trade.ID, got.Trade.Symbol)
	}

	fmt.Println("OK")
}
