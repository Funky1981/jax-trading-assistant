package configuration_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(1)
	if !ok {
		t.Fatal("cannot locate configuration test")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func readRepositoryFile(t *testing.T, relative string) string {
	t.Helper()
	root := repositoryRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, relative))
	if err != nil {
		t.Fatalf("read %s: %v", relative, err)
	}
	return string(raw)
}

func TestCanonicalEnvironmentExampleRetainsSafeOwnership(t *testing.T) {
	env := readRepositoryFile(t, ".env.example")
	for _, required := range []string{
		"DATABASE_URL=",
		"JAX_TRADER_RUNTIME_MODE=paper",
		"JAX_RESEARCH_RUNTIME_MODE=research",
		"JAX_REQUIRE_EXPLICIT_RUNTIME_MODE=true",
		"ALLOW_LIVE_TRADING=false",
		"BROKER_EXECUTION_ALLOWED=false",
		"EXECUTION_ENABLED=false",
		"EXECUTION_INSTRUCTION_WORKER_ENABLED=false",
		"MAX_LEVERAGE=1",
		"WORLD_MONITOR_PULL_ENABLED=false",
		"VITE_JAX_API_URL=",
		"VITE_IB_BRIDGE_URL=",
	} {
		if !strings.Contains(env, required) {
			t.Errorf(".env.example is missing canonical setting %q", required)
		}
	}
	for _, removed := range []string{
		"KNOWLEDGE_DATABASE_URL=",
		"REDIS_URL=",
		"REDIS_PASSWORD=",
		"MARKET_DATA_PROVIDER=",
		"VITE_API_URL=",
		"VITE_MEMORY_API_URL=",
	} {
		if strings.Contains(env, removed) {
			t.Errorf(".env.example still exposes retired setting %q", removed)
		}
	}
}

func TestComposeKeepsOptionalServicesOutOfCore(t *testing.T) {
	compose := readRepositoryFile(t, "docker-compose.yml")
	for _, required := range []string{
		"profiles: [\"observability\"]",
		"profiles: [\"world-monitor\"]",
		"WORLD_MONITOR_PULL_ENABLED: ${WORLD_MONITOR_PULL_ENABLED:-false}",
	} {
		if !strings.Contains(compose, required) {
			t.Errorf("docker-compose.yml is missing optional-service guard %q", required)
		}
	}
	traderStart := strings.Index(compose, "  jax-trader:")
	traderEnd := strings.Index(compose[traderStart:], "\n  jax-research:")
	if traderStart < 0 || traderEnd < 0 {
		t.Fatal("cannot isolate jax-trader service")
	}
	trader := compose[traderStart : traderStart+traderEnd]
	if strings.Contains(trader, "worldmonitor-events:") {
		t.Fatal("core jax-trader must not depend on optional World Monitor")
	}
}

func TestFrontendExampleUsesCurrentReaders(t *testing.T) {
	env := readRepositoryFile(t, "frontend/.env.example")
	for _, required := range []string{"VITE_JAX_API_URL=", "VITE_IB_BRIDGE_URL="} {
		if !strings.Contains(env, required) {
			t.Errorf("frontend/.env.example is missing %q", required)
		}
	}
	for _, removed := range []string{"VITE_API_URL=", "VITE_MEMORY_API_URL="} {
		if strings.Contains(env, removed) {
			t.Errorf("frontend/.env.example still exposes retired reader %q", removed)
		}
	}
}
