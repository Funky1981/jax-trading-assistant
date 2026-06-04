package llmcontext

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

type CompressionZone string

const (
	CompressionZoneNever         CompressionZone = "A"
	CompressionZoneDeterministic CompressionZone = "B"
	CompressionZoneSafe          CompressionZone = "C"
)

type CompressionPolicy struct {
	zones map[string]CompressionZone
}

func DefaultCompressionPolicy() CompressionPolicy {
	p := CompressionPolicy{zones: map[string]CompressionZone{}}
	for _, field := range []string{
		"symbol", "asset_type", "event_id", "candidate_id", "strategy_id", "timestamp",
		"source_timestamp", "quote_timestamp", "entry", "stop_loss", "take_profit",
		"risk_amount", "position_size", "spread_bps", "quote_freshness_seconds",
		"priced_in_verdict", "priced_in_score", "guardrail_results", "approval_token",
		"approval_expiry", "paper_mode", "live_mode", "broker_order_id", "fill_status",
	} {
		p.zones[field] = CompressionZoneNever
	}
	for _, field := range []string{"candles", "quotes", "price_windows", "confounder_scores", "provider_health", "strategy_eligibility_checks"} {
		p.zones[field] = CompressionZoneDeterministic
	}
	for _, field := range []string{"article_body", "duplicated_news_text", "long_logs", "test_failures", "search_results", "documentation_excerpts", "historical_research_prose", "post_trade_narrative_notes"} {
		p.zones[field] = CompressionZoneSafe
	}
	return p
}

func (p CompressionPolicy) ZoneForField(field string) CompressionZone {
	field = strings.ToLower(strings.TrimSpace(field))
	if zone, ok := p.zones[field]; ok {
		return zone
	}
	return CompressionZoneSafe
}

type CompressRequest struct {
	FieldName    string
	Text         string
	SourceIDs    []string
	RetrievalKey string
}

type CompressedEnvelope struct {
	CompressionAllowed bool            `json:"compression_allowed"`
	CompressionZone    CompressionZone `json:"compression_zone"`
	SourceIDs          []string        `json:"source_ids"`
	OriginalAvailable  bool            `json:"original_available"`
	RetrievalKey       string          `json:"retrieval_key"`
	ContentHash        string          `json:"content_hash"`
	CompressedText     string          `json:"compressed_text"`
}

type Compressor interface {
	Compress(text string) (string, error)
}

type NoopCompressor struct{}

func (NoopCompressor) Compress(text string) (string, error) {
	return strings.TrimSpace(text), nil
}

type CompressionService struct {
	policy     CompressionPolicy
	compressor Compressor
}

func NewCompressionService(policy CompressionPolicy, compressor Compressor) CompressionService {
	return CompressionService{policy: policy, compressor: compressor}
}

func (s CompressionService) Compress(req CompressRequest) (CompressedEnvelope, error) {
	zone := s.policy.ZoneForField(req.FieldName)
	if zone == CompressionZoneNever {
		return CompressedEnvelope{}, fmt.Errorf("field %s is Zone A and cannot be compressed", req.FieldName)
	}
	if zone == CompressionZoneDeterministic {
		return CompressedEnvelope{}, fmt.Errorf("field %s requires deterministic compaction", req.FieldName)
	}
	text, err := s.compressor.Compress(req.Text)
	if err != nil {
		return CompressedEnvelope{}, err
	}
	sum := sha256.Sum256([]byte(req.Text))
	return CompressedEnvelope{
		CompressionAllowed: true,
		CompressionZone:    zone,
		SourceIDs:          append([]string(nil), req.SourceIDs...),
		OriginalAvailable:  true,
		RetrievalKey:       req.RetrievalKey,
		ContentHash:        hex.EncodeToString(sum[:]),
		CompressedText:     text,
	}, nil
}

type ApprovalPacket struct {
	Symbol            string
	Entry             string
	StopLoss          string
	TakeProfit        string
	RiskAmount        string
	PricedInVerdict   PricedInVerdict
	GuardrailResults  string
	SupportingContext []CompressedEnvelope
}

func ValidateApprovalPacket(packet ApprovalPacket) error {
	if packet.Symbol == "" || packet.Entry == "" || packet.StopLoss == "" || packet.TakeProfit == "" ||
		packet.RiskAmount == "" || packet.PricedInVerdict == "" || packet.GuardrailResults == "" {
		return fmt.Errorf("approval packet missing uncompressed trading truth")
	}
	for _, env := range packet.SupportingContext {
		if !env.OriginalAvailable || len(env.SourceIDs) == 0 {
			return fmt.Errorf("compressed support missing source audit data")
		}
	}
	return nil
}
