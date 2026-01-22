package config

import (
	"flag"
	"io"
)

type Config struct {
	ProvidersPath string
	Bank          string
	Symbol        string
	Strategy      string
	UserContext   string
	TagsCSV       string
}

func DefaultConfig() Config {
	return Config{
		ProvidersPath: "config/providers.json",
		Bank:          "trade_decisions",
		Symbol:        "",
		Strategy:      "",
		UserContext:   "",
		TagsCSV:       "",
	}
}

func Parse(args []string) (Config, error) {
	cfg := DefaultConfig()
	fs := flag.NewFlagSet("jax-orchestrator", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	fs.StringVar(&cfg.ProvidersPath, "providers", cfg.ProvidersPath, "Path to providers.json")
	fs.StringVar(&cfg.Bank, "bank", cfg.Bank, "Memory bank to use")
	fs.StringVar(&cfg.Symbol, "symbol", cfg.Symbol, "Symbol to orchestrate")
	fs.StringVar(&cfg.Strategy, "strategy", cfg.Strategy, "Strategy identifier for tagging")
	fs.StringVar(&cfg.UserContext, "context", cfg.UserContext, "User context for planning")
	fs.StringVar(&cfg.TagsCSV, "tags", cfg.TagsCSV, "Comma-separated tags")

	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}
	return cfg, nil
}
