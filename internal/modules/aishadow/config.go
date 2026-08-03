package aishadow

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Enabled     bool
	Provider    string
	Model       string
	BaseURL     string
	Timeout     time.Duration
	Temperature float64
	Seed        int64
	MaxEvents   int
}

func LoadConfig(lookup func(string) (string, bool)) (Config, error) {
	required := []string{
		"JAX_AI_SHADOW_ENABLED", "JAX_AI_PROVIDER", "JAX_AI_MODEL", "JAX_AI_BASE_URL",
		"JAX_AI_TIMEOUT_SECONDS", "JAX_AI_TEMPERATURE", "JAX_AI_SEED", "JAX_AI_MAX_EVENTS",
	}
	values := map[string]string{}
	for _, key := range required {
		value, ok := lookup(key)
		if !ok || strings.TrimSpace(value) == "" {
			return Config{}, fmt.Errorf("missing required AI shadow configuration: %s", key)
		}
		values[key] = strings.TrimSpace(value)
	}
	enabled, err := strconv.ParseBool(values["JAX_AI_SHADOW_ENABLED"])
	if err != nil || !enabled {
		return Config{}, fmt.Errorf("JAX_AI_SHADOW_ENABLED must explicitly be true")
	}
	provider := strings.ToLower(values["JAX_AI_PROVIDER"])
	if provider != "ollama" {
		return Config{}, fmt.Errorf("unsupported JAX_AI_PROVIDER %q; only local ollama is allowed", provider)
	}
	parsedURL, err := url.Parse(values["JAX_AI_BASE_URL"])
	if err != nil || parsedURL.Scheme != "http" || parsedURL.Host == "" || parsedURL.User != nil ||
		parsedURL.RawQuery != "" || parsedURL.Fragment != "" || parsedURL.Path != "" && parsedURL.Path != "/" {
		return Config{}, fmt.Errorf("JAX_AI_BASE_URL must be an HTTP origin")
	}
	host := strings.ToLower(parsedURL.Hostname())
	if host != "localhost" && host != "127.0.0.1" && host != "::1" && host != "host.docker.internal" {
		return Config{}, fmt.Errorf("JAX_AI_BASE_URL must target local Ollama, not %q", host)
	}
	timeoutSeconds, err := strconv.Atoi(values["JAX_AI_TIMEOUT_SECONDS"])
	if err != nil || timeoutSeconds < 1 || timeoutSeconds > 600 {
		return Config{}, fmt.Errorf("JAX_AI_TIMEOUT_SECONDS must be between 1 and 600")
	}
	temperature, err := strconv.ParseFloat(values["JAX_AI_TEMPERATURE"], 64)
	if err != nil || temperature < 0 || temperature > 1 {
		return Config{}, fmt.Errorf("JAX_AI_TEMPERATURE must be between 0 and 1")
	}
	seed, err := strconv.ParseInt(values["JAX_AI_SEED"], 10, 64)
	if err != nil {
		return Config{}, fmt.Errorf("JAX_AI_SEED must be an integer")
	}
	maxEvents, err := strconv.Atoi(values["JAX_AI_MAX_EVENTS"])
	if err != nil || maxEvents < 1 || maxEvents > 60 {
		return Config{}, fmt.Errorf("JAX_AI_MAX_EVENTS must be between 1 and 60")
	}
	model := strings.TrimSpace(values["JAX_AI_MODEL"])
	if model == "" {
		return Config{}, fmt.Errorf("JAX_AI_MODEL must not be empty")
	}
	return Config{
		Enabled: true, Provider: provider, Model: model, BaseURL: strings.TrimRight(values["JAX_AI_BASE_URL"], "/"),
		Timeout: time.Duration(timeoutSeconds) * time.Second, Temperature: temperature, Seed: seed, MaxEvents: maxEvents,
	}, nil
}
