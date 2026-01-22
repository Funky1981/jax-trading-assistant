package config

import (
	"flag"
	"io"
)

type Config struct {
	ProvidersPath string
	InputPath     string
	Threshold     float64
}

func DefaultConfig() Config {
	return Config{
		ProvidersPath: "config/providers.json",
		InputPath:     "",
		Threshold:     0.7,
	}
}

func Parse(args []string) (Config, error) {
	cfg := DefaultConfig()
	fs := flag.NewFlagSet("jax-ingest", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	fs.StringVar(&cfg.ProvidersPath, "providers", cfg.ProvidersPath, "Path to providers.json")
	fs.StringVar(&cfg.InputPath, "input", cfg.InputPath, "Path to Dexter JSON payload (defaults to stdin)")
	fs.Float64Var(&cfg.Threshold, "threshold", cfg.Threshold, "Significance threshold for retention")

	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}
	return cfg, nil
}
