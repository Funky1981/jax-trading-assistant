package sec

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
)

const (
	SECBaseURL             = "https://data.sec.gov"
	SECUserAgentEnv        = "SEC_USER_AGENT"
	SECContactEnv          = "SEC_CONTACT"
	SECBaseURLEnv          = "SEC_BASE_URL"
	SECMaxResponseBytesEnv = "SEC_MAX_RESPONSE_BYTES"
	defaultSECMaxBytes     = 4 << 20
)

// RequestIdentity is the declared automated-client identity sent to SEC. It
// is configuration, not a secret, and is never persisted with raw evidence.
type RequestIdentity struct {
	Product string
	Contact string
}

func (identity RequestIdentity) userAgent() (string, error) {
	product := strings.TrimSpace(identity.Product)
	contact := strings.TrimSpace(identity.Contact)
	if product == "" {
		return "", errors.New("SEC user-agent product is required")
	}
	if strings.ContainsAny(product, "\r\n") || strings.ContainsAny(contact, "\r\n") {
		return "", errors.New("SEC user-agent identity must not contain newlines")
	}
	if contact == "" {
		return "", errors.New("SEC user-agent contact is required")
	}
	return product + " " + contact, nil
}

// Config contains only bounded SEC transport settings. No credential fields
// are supported because the selected read APIs are public and keyless.
type Config struct {
	BaseURL          string
	Identity         RequestIdentity
	MaxResponseBytes int64
}

func (config Config) Validate() error {
	base := strings.TrimSpace(config.BaseURL)
	parsed, err := url.Parse(base)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return errors.New("SEC base URL must be an HTTPS origin without credentials, query, or fragment")
	}
	if _, err := config.Identity.userAgent(); err != nil {
		return err
	}
	if config.MaxResponseBytes <= 0 || config.MaxResponseBytes > 16<<20 {
		return errors.New("SEC maximum response bytes must be between 1 and 16777216")
	}
	return nil
}

// LoadConfigFromEnv requires an operator-supplied product/contact identity;
// it intentionally has no fabricated production default.
func LoadConfigFromEnv() (Config, error) {
	product := strings.TrimSpace(os.Getenv(SECUserAgentEnv))
	contact := strings.TrimSpace(os.Getenv(SECContactEnv))
	if product == "" {
		return Config{}, fmt.Errorf("%s is required; declare an approved SEC automated-client identity", SECUserAgentEnv)
	}
	if contact == "" {
		return Config{}, fmt.Errorf("%s is required; do not invent SEC contact information", SECContactEnv)
	}
	config := Config{BaseURL: strings.TrimSpace(os.Getenv(SECBaseURLEnv)), Identity: RequestIdentity{Product: product, Contact: contact}, MaxResponseBytes: defaultSECMaxBytes}
	if config.BaseURL == "" {
		config.BaseURL = SECBaseURL
	}
	if raw := strings.TrimSpace(os.Getenv(SECMaxResponseBytesEnv)); raw != "" {
		var value int64
		if _, err := fmt.Sscan(raw, &value); err != nil {
			return Config{}, fmt.Errorf("%s must be an integer: %w", SECMaxResponseBytesEnv, err)
		}
		config.MaxResponseBytes = value
	}
	if err := config.Validate(); err != nil {
		return Config{}, err
	}
	return config, nil
}
