package llmcontext

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type HeadroomConfig struct {
	Enabled bool
	BaseURL string
	HTTP    *http.Client
}

type HeadroomRequest struct {
	FieldName            string
	TaskType             TaskType
	Text                 string
	SourceIDs            []string
	RetrievalKey         string
	LiveApprovalWorkflow bool
}

type CompressionTrialResult struct {
	Used             bool
	DisabledReason   string
	Envelope         CompressedEnvelope
	OriginalTokens   int
	CompressedTokens int
	SavingPercent    float64
	LatencyMillis    int64
}

type HeadroomTrial struct {
	config    HeadroomConfig
	policy    CompressionPolicy
	estimator SimpleTokenEstimator
}

func NewHeadroomTrial(config HeadroomConfig) HeadroomTrial {
	if config.HTTP == nil {
		config.HTTP = &http.Client{Timeout: 15 * time.Second}
	}
	return HeadroomTrial{
		config:    config,
		policy:    DefaultCompressionPolicy(),
		estimator: SimpleTokenEstimator{},
	}
}

func (t HeadroomTrial) Compress(ctx context.Context, req HeadroomRequest) (CompressionTrialResult, error) {
	if !t.config.Enabled {
		return CompressionTrialResult{DisabledReason: "headroom_disabled"}, nil
	}
	if req.LiveApprovalWorkflow {
		return CompressionTrialResult{DisabledReason: "live_approval_compression_disabled"}, nil
	}
	zone := t.policy.ZoneForField(req.FieldName)
	if zone != CompressionZoneSafe {
		return CompressionTrialResult{DisabledReason: "compression_zone_not_safe"}, nil
	}
	if strings.TrimSpace(t.config.BaseURL) == "" {
		return CompressionTrialResult{}, fmt.Errorf("headroom base URL required when enabled")
	}

	start := time.Now()
	compressed, err := t.call(ctx, req.Text)
	if err != nil {
		return CompressionTrialResult{}, err
	}
	service := NewCompressionService(t.policy, NoopCompressor{})
	envelope, err := service.Compress(CompressRequest{
		FieldName:    req.FieldName,
		Text:         req.Text,
		SourceIDs:    req.SourceIDs,
		RetrievalKey: req.RetrievalKey,
	})
	if err != nil {
		return CompressionTrialResult{}, err
	}
	envelope.CompressedText = compressed
	originalTokens := t.estimator.Estimate(req.Text)
	compressedTokens := t.estimator.Estimate(compressed)
	saving := 0.0
	if originalTokens > 0 {
		saving = float64(originalTokens-compressedTokens) / float64(originalTokens) * 100
	}
	return CompressionTrialResult{
		Used:             true,
		Envelope:         envelope,
		OriginalTokens:   originalTokens,
		CompressedTokens: compressedTokens,
		SavingPercent:    saving,
		LatencyMillis:    time.Since(start).Milliseconds(),
	}, nil
}

func (t HeadroomTrial) call(ctx context.Context, text string) (string, error) {
	type request struct {
		Text string `json:"text"`
	}
	type response struct {
		CompressedText string `json:"compressed_text"`
		Error          string `json:"error,omitempty"`
	}
	body, err := json.Marshal(request{Text: text})
	if err != nil {
		return "", fmt.Errorf("headroom request marshal: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(t.config.BaseURL, "/")+"/compress", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("headroom request create: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := t.config.HTTP.Do(req)
	if err != nil {
		return "", fmt.Errorf("headroom request: %w", err)
	}
	defer resp.Body.Close()
	respBytes, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("headroom response read: %w", err)
	}
	var decoded response
	if err := json.Unmarshal(respBytes, &decoded); err != nil {
		return "", fmt.Errorf("headroom response decode: %w", err)
	}
	if decoded.Error != "" {
		return "", fmt.Errorf("headroom API error: %s", decoded.Error)
	}
	if strings.TrimSpace(decoded.CompressedText) == "" {
		return "", fmt.Errorf("headroom response missing compressed_text")
	}
	return decoded.CompressedText, nil
}
