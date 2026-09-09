// Command hyp-event-classifier executes the authorised, bounded HYP-EVENT-001A
// direction-classification run. It reads the private evidence dataset locally,
// sends only event-time SEC text to the explicitly selected model, and writes
// metadata/results outside Git. It has no recommendation or execution path.
package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"jax-trading-assistant/internal/modules/hypevidence"
)

const (
	modelName       = "gpt-5.6-luna"
	providerName    = "openai"
	endpoint        = "https://api.openai.com/v1/responses"
	maxOutputTokens = 256
	maxRetries      = 1
	budgetMicros    = int64(4_500_000)
	inputPrice      = int64(200_000)   // $0.20 / million tokens
	cachePrice      = int64(20_000)    // $0.02 / million tokens
	outputPrice     = int64(1_200_000) // $1.20 / million tokens
	largeInputCut   = 272_000
	requestKind     = "hyp-event-001a-direction"
)

type event struct {
	EventID          string `json:"EventID"`
	Symbol           string `json:"Symbol"`
	CIK              string `json:"CIK"`
	Accession        string `json:"Accession"`
	Form             string `json:"Form"`
	AcceptanceRaw    string `json:"AcceptanceDateTime"`
	PrimaryDocument  string `json:"PrimaryDocument"`
	Amended          bool   `json:"Amended"`
	EventEligibility string `json:"EventEligibility"`
}

type packet struct {
	ContractVersion     string                 `json:"contract_version"`
	EventID             string                 `json:"event_id"`
	Accession           string                 `json:"accession"`
	AcceptanceTimestamp string                 `json:"acceptance_timestamp"`
	AvailabilityCutoff  string                 `json:"availability_cutoff"`
	PacketSHA256        string                 `json:"packet_sha256"`
	SemanticStatus      string                 `json:"semantic_status"`
	Documents           []hypevidence.Document `json:"documents"`
}

type classification struct {
	Direction       string   `json:"direction"`
	ReasonCode      string   `json:"reason_code"`
	EvidenceAnchors []string `json:"evidence_anchors"`
}

type usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
	InputDetails struct {
		CachedTokens int `json:"cached_tokens"`
	} `json:"input_tokens_details"`
	OutputDetails struct {
		ReasoningTokens int `json:"reasoning_tokens"`
	} `json:"output_tokens_details"`
}

type apiResponse struct {
	ID     string `json:"id"`
	Model  string `json:"model"`
	Status string `json:"status"`
	Output []struct {
		Type    string `json:"type"`
		Content []struct {
			Type    string `json:"type"`
			Text    string `json:"text"`
			Refusal string `json:"refusal"`
		} `json:"content"`
	} `json:"output"`
	Usage usage `json:"usage"`
}

type resultRecord struct {
	ContractVersion      string    `json:"contract_version"`
	HypothesisID         string    `json:"hypothesis_id"`
	EventID              string    `json:"event_id"`
	Symbol               string    `json:"symbol"`
	Accession            string    `json:"accession"`
	EvidencePacketSHA256 string    `json:"evidence_packet_sha256"`
	AvailabilityCutoff   time.Time `json:"availability_cutoff"`
	Direction            string    `json:"direction"`
	ReasonCode           string    `json:"reason_code"`
	EvidenceAnchors      []string  `json:"evidence_anchors"`
	ClassifierVersion    string    `json:"classifier_version"`
	PromptVersion        string    `json:"prompt_version"`
	Provider             string    `json:"provider"`
	Model                string    `json:"model"`
	InferenceTimestamp   time.Time `json:"inference_timestamp"`
	RawResponseSHA256    string    `json:"raw_response_sha256"`
	RequestID            string    `json:"request_id"`
	ResponseID           string    `json:"response_id"`
	AttemptNumber        int       `json:"attempt_number"`
	Usage                usage     `json:"usage"`
	AccountedCostMicros  int64     `json:"accounted_cost_micros"`
}

type runSummary struct {
	ContractVersion       string `json:"contract_version"`
	HypothesisID          string `json:"hypothesis_id"`
	DatasetID             string `json:"dataset_id"`
	DatasetManifestSHA256 string `json:"dataset_manifest_sha256"`
	Stage                 string `json:"stage"`
	Model                 string `json:"model"`
	ReasoningEffort       string `json:"reasoning_effort"`
	PromptVersion         string `json:"prompt_version"`
	StartedAt             string `json:"started_at"`
	CompletedAt           string `json:"completed_at"`
	Requested             int    `json:"requested"`
	Completed             int    `json:"completed"`
	Retries               int    `json:"retries"`
	Failures              int    `json:"failures"`
	InputTokens           int    `json:"input_tokens"`
	CachedTokens          int    `json:"cached_tokens"`
	OutputTokens          int    `json:"output_tokens"`
	ReasoningTokens       int    `json:"reasoning_tokens"`
	TotalTokens           int    `json:"total_tokens"`
	ActualCostMicros      int64  `json:"actual_cost_micros"`
	PriorCostMicros       int64  `json:"prior_cost_micros"`
	BudgetMicros          int64  `json:"budget_micros"`
	StopReason            string `json:"stop_reason,omitempty"`
}

type classifier struct {
	client      *http.Client
	apiKey      string
	outputPath  string
	summaryPath string
	resultPath  string
	spentMicros int64
	usage       usage
	retries     int
	failures    int
}

func main() {
	stage := flag.String("stage", "qualification", "qualification or full")
	outDir := flag.String("out", "data/datasets/hyp-event-001a/classifications-v1", "private output directory")
	evidenceDir := flag.String("evidence", "data/datasets/hyp-event-001a/evidence-v2", "private evidence dataset")
	eventsPath := flag.String("events", "data/datasets/hyp-event-001a/dataset-2016-2025-sip-sec-v1/normalized/events-qualified.json", "private parent event panel")
	flag.Parse()
	if *stage != "qualification" && *stage != "full" {
		fatal("stage must be qualification or full")
	}
	if err := run(*stage, *outDir, *evidenceDir, *eventsPath); err != nil {
		fatal(err.Error())
	}
}

func run(stage, outDir, evidenceDir, eventsPath string) error {
	if err := (hypevidence.DefaultDirectionClassifierContract()).Validate(); err != nil {
		return fmt.Errorf("direction contract: %w", err)
	}
	key, source, err := configuredAPIKey()
	if err != nil {
		return err
	}
	_ = source // source is intentionally not emitted; the key never enters output.
	events, err := loadEvents(eventsPath)
	if err != nil {
		return err
	}
	if err := verifyDataset(evidenceDir); err != nil {
		return err
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	selected, err := selectEvents(stage, events, evidenceDir, outDir)
	if err != nil {
		return err
	}
	if len(selected) == 0 {
		return errors.New("no eligible events selected")
	}
	c := &classifier{client: &http.Client{Timeout: 120 * time.Second}, apiKey: key,
		outputPath:  filepath.Join(outDir, "run-summary-"+stage+".json"),
		summaryPath: filepath.Join(outDir, "run-summary-"+stage+".json"),
		resultPath:  filepath.Join(outDir, "results-"+stage+".jsonl")}
	known, err := loadExistingResults(c.resultPath)
	if err != nil {
		return err
	}
	priorCost := loadResultCost(c.resultPath)
	if stage == "full" {
		qualificationPath := filepath.Join(outDir, "results-qualification.jsonl")
		qualificationKnown, qualificationErr := loadExistingResults(qualificationPath)
		if qualificationErr != nil {
			return qualificationErr
		}
		for eventID := range qualificationKnown {
			known[eventID] = true
		}
		priorCost += loadResultCost(qualificationPath)
	}
	c.spentMicros = priorCost
	started := time.Now().UTC()
	summary := runSummary{ContractVersion: "jax.hyp-event-001a.hosted-run/v1", HypothesisID: "HYP-EVENT-001A", DatasetID: "hyp-event-001a-sec-8k-alpaca-sip-2016-2025-evidence-v2", DatasetManifestSHA256: manifestHash(evidenceDir), Stage: stage, Model: modelName, ReasoningEffort: "none", PromptVersion: hypevidence.PromptIdentity(), StartedAt: started.Format(time.RFC3339), BudgetMicros: budgetMicros, PriorCostMicros: priorCost}
	for _, ev := range selected {
		if known[ev.EventID] {
			continue
		}
		summary.Requested++
		record, retry, callErr := c.classify(ev, evidenceDir)
		if callErr != nil {
			c.failures++
			if errors.Is(callErr, errBudget) {
				summary.StopReason = "HYP-EVENT-001A BUDGET LIMIT REACHED"
				break
			}
			return fmt.Errorf("HYP-EVENT-001A classification failed for %s: %w", ev.EventID, callErr)
		}
		c.retries += retry
		if err := appendJSONLine(c.resultPath, record); err != nil {
			return err
		}
		summary.Completed++
	}
	summary.CompletedAt = time.Now().UTC().Format(time.RFC3339)
	summary.Retries = c.retries
	summary.Failures = c.failures
	summary.InputTokens = c.usage.InputTokens
	summary.CachedTokens = c.usage.InputDetails.CachedTokens
	summary.OutputTokens = c.usage.OutputTokens
	summary.ReasoningTokens = c.usage.OutputDetails.ReasoningTokens
	summary.TotalTokens = c.usage.TotalTokens
	summary.ActualCostMicros = c.spentMicros
	if err := writeJSON(c.summaryPath, summary); err != nil {
		return err
	}
	fmt.Printf("stage=%s requested=%d completed=%d retries=%d failures=%d input_tokens=%d output_tokens=%d cost_usd=%.6f stop=%s\n", stage, summary.Requested, summary.Completed, summary.Retries, summary.Failures, summary.InputTokens, summary.OutputTokens, float64(summary.ActualCostMicros)/1_000_000, summary.StopReason)
	return nil
}

var errBudget = errors.New("budget ceiling")

func (c *classifier) classify(ev event, evidenceDir string) (resultRecord, int, error) {
	packet, text, err := loadPacketText(evidenceDir, ev)
	if err != nil {
		return resultRecord{}, 0, err
	}
	contract := hypevidence.DefaultDirectionClassifierContract()
	user := strings.ReplaceAll(contract.InputTemplate, "{{event_id}}", ev.EventID)
	user = strings.ReplaceAll(user, "{{accession}}", ev.Accession)
	user = strings.ReplaceAll(user, "{{evidence_packet_sha256}}", packet.PacketSHA256)
	user = strings.ReplaceAll(user, "{{availability_cutoff}}", packet.AvailabilityCutoff)
	user = strings.ReplaceAll(user, "{{normalized_packet_text}}", text)
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"direction":        map[string]any{"type": "string", "enum": []string{hypevidence.DirectionPositive, hypevidence.DirectionNegative, hypevidence.DirectionNeutral, hypevidence.DirectionInsufficientEvidence}},
			"reason_code":      map[string]any{"type": "string", "enum": []string{hypevidence.ReasonGroundedDirection, hypevidence.ReasonInsufficientEvidence, hypevidence.ReasonConflictingEvidence, hypevidence.ReasonInvalidGrounding}},
			"evidence_anchors": map[string]any{"type": "array", "items": map[string]any{"type": "string", "minLength": 1}, "minItems": 0, "maxItems": 8},
		},
		"required": []string{"direction", "reason_code", "evidence_anchors"}, "additionalProperties": false,
	}
	for attempt := 1; attempt <= 1+maxRetries; attempt++ {
		if !c.canReserve(user, contract.SystemPrompt, schema) {
			return resultRecord{}, attempt - 1, errBudget
		}
		response, raw, err := c.call(contract.SystemPrompt, user, schema, ev.EventID, attempt)
		if err != nil {
			return resultRecord{}, attempt - 1, err
		}
		parsed, parseErr := parseClassification(raw)
		validationErr := parseErr
		if parseErr == nil {
			validationErr = validateClassification(parsed, packet, text)
			if validationErr == nil {
				now := time.Now().UTC()
				return resultRecord{ContractVersion: hypevidence.DirectionContractVersion, HypothesisID: "HYP-EVENT-001A", EventID: ev.EventID, Symbol: ev.Symbol, Accession: ev.Accession, EvidencePacketSHA256: packet.PacketSHA256, AvailabilityCutoff: mustTime(packet.AvailabilityCutoff), Direction: parsed.Direction, ReasonCode: parsed.ReasonCode, EvidenceAnchors: parsed.EvidenceAnchors, ClassifierVersion: hypevidence.DirectionContractVersion, PromptVersion: hypevidence.PromptIdentity(), Provider: providerName, Model: modelName, InferenceTimestamp: now, RawResponseSHA256: sha256Hex([]byte(raw)), RequestID: response.requestID, ResponseID: response.id, AttemptNumber: attempt, Usage: response.usage, AccountedCostMicros: costMicros(response.usage)}, attempt - 1, nil
			}
		}
		if attempt == 1+maxRetries {
			return resultRecord{}, attempt - 1, fmt.Errorf("provider output failed frozen contract validation: %v", validationErr)
		}
	}
	return resultRecord{}, maxRetries, errors.New("classification exhausted")
}

type callResponse struct {
	id, requestID string
	usage         usage
}

func (c *classifier) call(system, user string, schema map[string]any, eventID string, attempt int) (callResponse, string, error) {
	payload := map[string]any{"model": modelName, "input": []any{map[string]any{"role": "system", "content": system}, map[string]any{"role": "user", "content": user}}, "reasoning": map[string]string{"effort": "none"}, "max_output_tokens": maxOutputTokens, "store": false, "text": map[string]any{"format": map[string]any{"type": "json_schema", "name": "jax_hyp_event_direction_v1", "strict": true, "schema": schema}}}
	body, err := json.Marshal(payload)
	if err != nil {
		return callResponse{}, "", err
	}
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return callResponse{}, "", errors.New("create provider request")
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Jax-Hyp-Event", eventID)
	req.Header.Set("X-Jax-Hyp-Attempt", strconv.Itoa(attempt))
	resp, err := c.client.Do(req)
	if err != nil {
		return callResponse{}, "", errors.New("provider transport failure")
	}
	defer resp.Body.Close()
	rawBody, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return callResponse{}, "", errors.New("provider response read failure")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return callResponse{}, "", fmt.Errorf("provider returned HTTP status %d", resp.StatusCode)
	}
	var decoded apiResponse
	if err := json.Unmarshal(rawBody, &decoded); err != nil {
		return callResponse{}, "", errors.New("provider response decode failure")
	}
	if decoded.Status != "completed" || decoded.Model != modelName {
		return callResponse{}, "", errors.New("provider response identity/status failure")
	}
	if decoded.Usage.InputTokens <= 0 || decoded.Usage.OutputTokens <= 0 || decoded.Usage.TotalTokens < decoded.Usage.InputTokens+decoded.Usage.OutputTokens {
		return callResponse{}, "", errors.New("provider usage is incomplete")
	}
	text, err := responseText(decoded)
	if err != nil {
		return callResponse{}, "", err
	}
	actual := costMicros(decoded.Usage)
	if c.spentMicros+actual > budgetMicros {
		return callResponse{}, "", errBudget
	}
	c.spentMicros += actual
	c.usage.InputTokens += decoded.Usage.InputTokens
	c.usage.OutputTokens += decoded.Usage.OutputTokens
	c.usage.TotalTokens += decoded.Usage.TotalTokens
	c.usage.InputDetails.CachedTokens += decoded.Usage.InputDetails.CachedTokens
	c.usage.OutputDetails.ReasoningTokens += decoded.Usage.OutputDetails.ReasoningTokens
	return callResponse{id: decoded.ID, requestID: resp.Header.Get("x-request-id"), usage: decoded.Usage}, text, nil
}

func (c *classifier) canReserve(user, system string, schema map[string]any) bool {
	input := hypevidence.EstimateTokens(system+user) + hypevidence.EstimateTokens(mustJSON(schema)) + 1024
	output := maxOutputTokens
	return c.spentMicros+estimatedCost(input, output) <= budgetMicros
}

func estimatedCost(input, output int) int64 {
	inputRate := inputPrice
	outputRate := outputPrice
	if input > largeInputCut {
		inputRate *= 2
		outputRate = outputRate * 3 / 2
	}
	return int64(input)*inputRate/1_000_000 + int64(output)*outputRate/1_000_000
}

func costMicros(u usage) int64 {
	uncached := u.InputTokens - u.InputDetails.CachedTokens
	if uncached < 0 {
		uncached = 0
	}
	inputRate := inputPrice
	outputRate := outputPrice
	if u.InputTokens > largeInputCut {
		inputRate *= 2
		outputRate = outputRate * 3 / 2
	}
	return int64(uncached)*inputRate/1_000_000 + int64(u.InputDetails.CachedTokens)*cachePrice/1_000_000 + int64(u.OutputTokens)*outputRate/1_000_000
}

func responseText(r apiResponse) (string, error) {
	for _, item := range r.Output {
		for _, content := range item.Content {
			if content.Refusal != "" {
				return "", errors.New("provider refused classification")
			}
			if content.Type == "output_text" || content.Text != "" {
				return content.Text, nil
			}
		}
	}
	return "", errors.New("provider output text missing")
}

func parseClassification(raw string) (classification, error) {
	var out classification
	dec := json.NewDecoder(strings.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&out); err != nil {
		return classification{}, err
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		return classification{}, errors.New("trailing provider output")
	}
	return out, nil
}

func validateClassification(c classification, p packet, _ string) error {
	if c.Direction == hypevidence.DirectionPositive || c.Direction == hypevidence.DirectionNegative {
		if c.ReasonCode != hypevidence.ReasonGroundedDirection || len(c.EvidenceAnchors) == 0 {
			return errors.New("polarity is ungrounded")
		}
	}
	if c.Direction == hypevidence.DirectionNeutral && len(c.EvidenceAnchors) == 0 {
		return errors.New("neutral result is ungrounded")
	}
	if c.Direction == hypevidence.DirectionInsufficientEvidence && c.ReasonCode != hypevidence.ReasonInsufficientEvidence && c.ReasonCode != hypevidence.ReasonConflictingEvidence && c.ReasonCode != hypevidence.ReasonInvalidGrounding {
		return errors.New("invalid abstention reason")
	}
	allowed := map[string]bool{}
	for _, d := range p.Documents {
		allowed["document:"+d.Filename] = true
	}
	for _, a := range c.EvidenceAnchors {
		anchor := strings.TrimSpace(a)
		if anchor == "" || len(anchor) > 300 {
			return errors.New("anchor is empty or oversized")
		}
		if strings.HasPrefix(anchor, "document:") {
			parts := strings.SplitN(anchor, "#", 2)
			if !allowed[parts[0]] {
				return errors.New("anchor references unavailable document")
			}
			continue
		}
		// The frozen contract permits bounded textual anchors as well as
		// document-qualified anchors. Exact quote matching is intentionally not
		// imposed here because HTML normalization can alter whitespace and
		// punctuation; the qualification review checks anchors against the
		// supplied packet without using outcomes.
	}
	if c.Direction != hypevidence.DirectionPositive && c.Direction != hypevidence.DirectionNegative && c.Direction != hypevidence.DirectionNeutral && c.Direction != hypevidence.DirectionInsufficientEvidence {
		return errors.New("unsupported direction")
	}
	return nil
}

func loadEvents(path string) ([]event, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var events []event
	if err := json.Unmarshal(b, &events); err != nil {
		return nil, err
	}
	out := make([]event, 0, len(events))
	for _, e := range events {
		if e.EventEligibility == "QUALIFYING" && e.Form == "8-K" && !e.Amended {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].EventID < out[j].EventID })
	return out, nil
}

func selectEvents(stage string, events []event, evidenceDir, outDir string) ([]event, error) {
	dev := make([]event, 0)
	all := make([]event, 0)
	for _, e := range events {
		y, err := eventYear(e)
		if err != nil {
			return nil, err
		}
		if y <= 2024 {
			all = append(all, e)
		}
		if y >= 2016 && y <= 2021 {
			dev = append(dev, e)
		}
	}
	if stage == "full" {
		return all, nil
	}
	// Deterministic qualification sample: one round-robin pass through year and
	// normalized-size strata. No market outcome or semantic polarity is used.
	type key struct{ year, bucket int }
	groups := map[key][]event{}
	for _, e := range dev {
		p, err := loadPacket(evidenceDir, e)
		if err != nil {
			return nil, err
		}
		total := 0
		for _, d := range p.Documents {
			total += d.EstimatedTokens
		}
		bucket := 0
		if total >= 10000 {
			bucket = 1
		}
		if total >= 30000 {
			bucket = 2
		}
		groups[key{mustEventYear(e), bucket}] = append(groups[key{mustEventYear(e), bucket}], e)
	}
	keys := make([]key, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].year != keys[j].year {
			return keys[i].year < keys[j].year
		}
		return keys[i].bucket < keys[j].bucket
	})
	for _, k := range keys {
		sort.Slice(groups[k], func(i, j int) bool { return groups[k][i].EventID < groups[k][j].EventID })
	}
	selected := make([]event, 0, 40)
	for round := 0; len(selected) < 40; round++ {
		added := false
		for _, k := range keys {
			if round < len(groups[k]) {
				selected = append(selected, groups[k][round])
				added = true
				if len(selected) == 40 {
					break
				}
			}
		}
		if !added {
			break
		}
	}
	if len(selected) != 40 {
		return nil, fmt.Errorf("qualification sample has %d events, want 40", len(selected))
	}
	selection := make([]string, len(selected))
	for i, e := range selected {
		selection[i] = e.EventID
	}
	sort.Strings(selection)
	return selected, writeJSON(filepath.Join(outDir, "qualification-selection.json"), map[string]any{"contract_version": "jax.hyp-event-001a.qualification-selection/v1", "dataset_id": "hyp-event-001a-sec-8k-alpaca-sip-2016-2025-evidence-v2", "hypothesis_id": "HYP-EVENT-001A", "count": len(selection), "event_ids": selection})
}

func loadPacket(evidenceDir string, e event) (packet, error) {
	b, err := os.ReadFile(filepath.Join(evidenceDir, "packets", e.EventID+".json"))
	if err != nil {
		return packet{}, err
	}
	var p packet
	if err = json.Unmarshal(b, &p); err != nil {
		return p, err
	}
	if p.EventID != e.EventID || p.Accession != e.Accession || p.SemanticStatus == "" {
		return p, errors.New("packet identity mismatch")
	}
	return p, nil
}

func loadPacketText(evidenceDir string, e event) (packet, string, error) {
	p, err := loadPacket(evidenceDir, e)
	if err != nil {
		return p, "", err
	}
	var b strings.Builder
	for _, d := range p.Documents {
		if !d.NormalizedAvailable || !d.TextEligible {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(evidenceDir, "raw", e.EventID, d.Filename))
		if err != nil {
			return p, "", err
		}
		var text string
		if strings.HasSuffix(strings.ToLower(d.Filename), ".txt") {
			text = hypevidence.NormalizeText(raw)
		} else {
			text, err = hypevidence.NormalizeHTML(raw)
			if err != nil {
				return p, "", err
			}
		}
		if hypevidence.SHA256Hex([]byte(text)) != d.NormalizedSHA256 {
			return p, "", errors.New("normalized evidence hash mismatch")
		}
		b.WriteString("DOCUMENT: ")
		b.WriteString(d.Filename)
		b.WriteString("\nDOCUMENT_TYPE: ")
		b.WriteString(d.DocumentType)
		b.WriteString("\n")
		b.WriteString(text)
		b.WriteString("\n\n")
	}
	if b.Len() == 0 {
		return p, "", errors.New("packet has no classifier-eligible text")
	}
	return p, b.String(), nil
}

func verifyDataset(dir string) error {
	b, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		return err
	}
	var m struct {
		DatasetID             string `json:"dataset_id"`
		ManifestSHA256        string `json:"manifest_sha256"`
		PacketCount           int    `json:"packet_count"`
		RetrievalFailureCount int    `json:"retrieval_failure_count"`
	}
	if err = json.Unmarshal(b, &m); err != nil {
		return err
	}
	if m.DatasetID != "hyp-event-001a-sec-8k-alpaca-sip-2016-2025-evidence-v2" || m.ManifestSHA256 != "967dfcb18eff8b6a3f4ec39fedd3898537446a284dc2a868631926dc1f41408c" || m.PacketCount != 1059 || m.RetrievalFailureCount != 0 {
		return errors.New("evidence dataset identity or coverage mismatch")
	}
	return nil
}

func configuredAPIKey() (string, string, error) {
	if v := strings.TrimSpace(os.Getenv("JAX_OPENAI_EXPERIMENT_API_KEY")); v != "" {
		return v, "process:JAX_OPENAI_EXPERIMENT_API_KEY", nil
	}
	if v := strings.TrimSpace(os.Getenv("OPENAI_API_KEY")); v != "" {
		return v, "process:OPENAI_API_KEY", nil
	}
	vals, err := readDotEnv(".env")
	if err != nil {
		return "", "", err
	}
	if v := strings.TrimSpace(vals["JAX_OPENAI_EXPERIMENT_API_KEY"]); v != "" {
		return v, "file:JAX_OPENAI_EXPERIMENT_API_KEY", nil
	}
	if v := strings.TrimSpace(vals["OPENAI_API_KEY"]); v != "" {
		return v, "file:OPENAI_API_KEY", nil
	}
	return "", "", errors.New("authorised OpenAI key is unavailable from process or repository .env")
}

func readDotEnv(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	out := map[string]string{}
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		value = strings.Trim(value, "\"")
		out[key] = value
	}
	return out, s.Err()
}

func loadExistingResults(path string) (map[string]bool, error) {
	out := map[string]bool{}
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return out, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	for s.Scan() {
		var r resultRecord
		if json.Unmarshal(s.Bytes(), &r) == nil && r.EventID != "" {
			out[r.EventID] = true
		}
	}
	return out, s.Err()
}

func loadResultCost(path string) int64 {
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer f.Close()
	var total int64
	s := bufio.NewScanner(f)
	for s.Scan() {
		var r resultRecord
		if json.Unmarshal(s.Bytes(), &r) == nil && r.AccountedCostMicros > 0 {
			total += r.AccountedCostMicros
		}
	}
	return total
}

func appendJSONLine(path string, v any) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = f.Write(append(b, '\n'))
	return err
}
func writeJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	tmp := path + ".tmp"
	if err = os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
func manifestHash(dir string) string {
	b, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		return ""
	}
	var m map[string]any
	if json.Unmarshal(b, &m) != nil {
		return ""
	}
	if v, ok := m["manifest_sha256"].(string); ok {
		return v
	}
	return sha256Hex(b)
}
func sha256Hex(b []byte) string   { d := sha256.Sum256(b); return hex.EncodeToString(d[:]) }
func mustJSON(v any) string       { b, _ := json.Marshal(v); return string(b) }
func mustTime(s string) time.Time { t, _ := time.Parse(time.RFC3339, s); return t }
func eventYear(e event) (int, error) {
	t, err := time.ParseInLocation("01/02/2006 15:04:05", e.AcceptanceRaw, time.UTC)
	if err != nil {
		return 0, err
	}
	return t.Year(), nil
}
func mustEventYear(e event) int { y, _ := eventYear(e); return y }
func fatal(s string)            { fmt.Fprintln(os.Stderr, s); os.Exit(1) }
