package litellmconfig

import (
	"os"
	"strings"
	"testing"
)

func TestLiteLLMConfigIsLocalFirstAndUsesVirtualKeys(t *testing.T) {
	data, err := os.ReadFile("litellm-config.yaml")
	if err != nil {
		t.Fatalf("read litellm config: %v", err)
	}
	text := string(data)
	for _, want := range []string{
		"model_name: local-small",
		"api_base: http://ollama:11434",
		"JAX_LITELLM_MASTER_KEY",
		"JAX_OPENAI_API_KEY",
		"paid-strong",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("litellm config missing %q", want)
		}
	}
	if strings.Contains(text, "sk-") {
		t.Fatal("litellm config must not contain plaintext provider keys")
	}
}

func TestDockerComposeRunsPrivateGatewayAndOllama(t *testing.T) {
	data, err := os.ReadFile("docker-compose.yaml")
	if err != nil {
		t.Fatalf("read docker compose: %v", err)
	}
	text := string(data)
	for _, want := range []string{
		"litellm:",
		"ollama:",
		"4000:4000",
		"11434",
		"litellm-config.yaml",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("docker compose missing %q", want)
		}
	}
}
