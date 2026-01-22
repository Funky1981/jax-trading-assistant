package config

import "testing"

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.ProvidersPath == "" {
		t.Fatalf("expected providers path")
	}
	if cfg.Bank == "" {
		t.Fatalf("expected default bank")
	}
}

func TestParseConfigOverrides(t *testing.T) {
	cfg, err := Parse([]string{
		"-providers", "config/custom.json",
		"-bank", "custom_bank",
		"-symbol", "AAPL",
		"-strategy", "gap_v1",
		"-context", "user context",
		"-tags", "earnings,high",
	})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if cfg.ProvidersPath != "config/custom.json" {
		t.Fatalf("expected providers override, got %q", cfg.ProvidersPath)
	}
	if cfg.Bank != "custom_bank" {
		t.Fatalf("expected bank override, got %q", cfg.Bank)
	}
	if cfg.Symbol != "AAPL" {
		t.Fatalf("expected symbol override, got %q", cfg.Symbol)
	}
	if cfg.Strategy != "gap_v1" {
		t.Fatalf("expected strategy override, got %q", cfg.Strategy)
	}
	if cfg.UserContext != "user context" {
		t.Fatalf("expected context override, got %q", cfg.UserContext)
	}
	if cfg.TagsCSV != "earnings,high" {
		t.Fatalf("expected tags override, got %q", cfg.TagsCSV)
	}
}
