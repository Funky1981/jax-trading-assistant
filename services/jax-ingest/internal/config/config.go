package config

import (
	"flag"
	"io"
	"strings"
)

type Config struct {
	ProvidersPath  string
	InputPath      string
	Threshold      float64
	IngestInterval int      // seconds between ingestion runs
	Symbols        []string // list of symbols to ingest
}

func DefaultConfig() Config {
	return Config{
		ProvidersPath:  "config/providers.json",
		InputPath:      "",
		Threshold:      0.7,
		IngestInterval: 60,                                     // default 1 minute
		Symbols:        []string{"SPY", "QQQ", "AAPL", "MSFT"}, // default watchlist
	}
}

func Parse(args []string) (Config, error) {
	cfg := DefaultConfig()
	fs := flag.NewFlagSet("jax-ingest", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var symbolsStr string

	fs.StringVar(&cfg.ProvidersPath, "providers", cfg.ProvidersPath, "Path to providers.json")
	fs.StringVar(&cfg.InputPath, "input", cfg.InputPath, "Path to Dexter JSON payload (defaults to stdin)")
	fs.Float64Var(&cfg.Threshold, "threshold", cfg.Threshold, "Significance threshold for retention")
	fs.IntVar(&cfg.IngestInterval, "interval", cfg.IngestInterval, "Seconds between ingestion runs")
	fs.StringVar(&symbolsStr, "symbols", "", "Comma-separated list of symbols to ingest")

	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}

	// Parse symbols if provided
	if symbolsStr != "" {
		cfg.Symbols = strings.Split(symbolsStr, ",")
		for i := range cfg.Symbols {
			cfg.Symbols[i] = strings.TrimSpace(cfg.Symbols[i])
		}
	}

	return cfg, nil
}
