package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadJaxCoreConfig_Defaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "jax-core.json")
	if err := os.WriteFile(path, []byte(`{"accountSize":10000}`), 0600); err != nil {
		t.Fatalf("write: %v", err)
	}

	cfg, err := LoadJaxCoreConfig(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.HTTPPort != 8081 {
		t.Fatalf("expected default http port 8081, got %d", cfg.HTTPPort)
	}
	if cfg.RiskPercent != 3 {
		t.Fatalf("expected default risk percent 3, got %v", cfg.RiskPercent)
	}
	if cfg.MaxConsecutiveLosses != 3 {
		t.Fatalf("expected default max consecutive losses 3, got %d", cfg.MaxConsecutiveLosses)
	}
}

func TestLoadJaxCoreConfig_CustomValues(t *testing.T) {
	cases := []struct {
		name        string
		payload     string
		wantPort    int
		wantRisk    float64
		wantDexter  bool
		wantAccount float64
		wantMaxLoss int
	}{
		{
			name:        "custom values",
			payload:     `{"httpPort":9090,"riskPercent":2.5,"useDexter":true,"accountSize":20000,"maxConsecutiveLosses":4}`,
			wantPort:    9090,
			wantRisk:    2.5,
			wantDexter:  true,
			wantAccount: 20000,
			wantMaxLoss: 4,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "jax-core.json")
			if err := os.WriteFile(path, []byte(tc.payload), 0600); err != nil {
				t.Fatalf("write: %v", err)
			}

			cfg, err := LoadJaxCoreConfig(path)
			if err != nil {
				t.Fatalf("load: %v", err)
			}
			if cfg.HTTPPort != tc.wantPort {
				t.Fatalf("expected http port %d, got %d", tc.wantPort, cfg.HTTPPort)
			}
			if cfg.RiskPercent != tc.wantRisk {
				t.Fatalf("expected risk percent %v, got %v", tc.wantRisk, cfg.RiskPercent)
			}
			if cfg.UseDexter != tc.wantDexter {
				t.Fatalf("expected useDexter=%v, got %v", tc.wantDexter, cfg.UseDexter)
			}
			if cfg.AccountSize != tc.wantAccount {
				t.Fatalf("expected account size %v, got %v", tc.wantAccount, cfg.AccountSize)
			}
			if cfg.MaxConsecutiveLosses != tc.wantMaxLoss {
				t.Fatalf("expected max consecutive losses %d, got %d", tc.wantMaxLoss, cfg.MaxConsecutiveLosses)
			}
		})
	}
}
