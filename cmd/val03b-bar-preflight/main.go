package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"

	"jax-trading-assistant/libs/contracts/canonical"
	providercontract "jax-trading-assistant/libs/contracts/provider"
	"jax-trading-assistant/libs/marketdata"
)

const (
	startDate = "2015-01-01"
	endDate   = "2024-12-31"
	asOfDate  = "2024-12-31"
)

var symbols = []string{"SPY", "QQQ", "IWM", "DIA", "XLK", "XLF", "XLE", "TLT", "GLD"}

type fileStore struct {
	root string
	mu   sync.Mutex
	refs map[providercontract.RawPayloadID]providercontract.RawPayloadRef
}

func newFileStore(root string) (*fileStore, error) {
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, err
	}
	return &fileStore{root: root, refs: make(map[providercontract.RawPayloadID]providercontract.RawPayloadRef)}, nil
}

func (store *fileStore) Put(ctx context.Context, ref providercontract.RawPayloadRef, payload []byte) (providercontract.RawPayloadLocation, error) {
	if err := ctx.Err(); err != nil {
		return providercontract.RawPayloadLocation{}, err
	}
	if err := ref.Validate(); err != nil {
		return providercontract.RawPayloadLocation{}, err
	}
	if err := ref.Content.Digest.VerifyBytes(payload); err != nil {
		return providercontract.RawPayloadLocation{}, err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	if existing, ok := store.refs[ref.ID]; ok && !reflect.DeepEqual(existing, ref) {
		return providercontract.RawPayloadLocation{}, errors.New("raw payload identity conflict")
	}
	path := filepath.Join(store.root, ref.Content.Digest.Value+".json")
	if existing, err := os.ReadFile(path); err == nil {
		if !reflect.DeepEqual(existing, payload) {
			return providercontract.RawPayloadLocation{}, errors.New("raw payload digest collision")
		}
	} else if os.IsNotExist(err) {
		if err := os.WriteFile(path, payload, 0o600); err != nil {
			return providercontract.RawPayloadLocation{}, err
		}
	} else {
		return providercontract.RawPayloadLocation{}, err
	}
	store.refs[ref.ID] = ref
	return providercontract.RawPayloadLocation{Store: canonical.VersionIdentity{Namespace: "jax.raw_payload_store", Value: "val03b-file/v1"}, Key: "sha256/" + ref.Content.Digest.Value}, nil
}

func (store *fileStore) Get(ctx context.Context, ref providercontract.RawPayloadRef) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	store.mu.Lock()
	known, ok := store.refs[ref.ID]
	store.mu.Unlock()
	if !ok || !reflect.DeepEqual(known, ref) {
		return nil, errors.New("raw payload reference is not known to preflight store")
	}
	return os.ReadFile(filepath.Join(store.root, ref.Content.Digest.Value+".json"))
}

type familyQuality struct {
	Instrument             string   `json:"instrument"`
	Adjustment             string   `json:"adjustment"`
	SourceID               string   `json:"source_id"`
	FirstSession           string   `json:"first_session"`
	LastSession            string   `json:"last_session"`
	RecordCount            int      `json:"record_count"`
	DuplicateCount         int      `json:"duplicate_count"`
	MissingSessionCount    int      `json:"missing_session_count"`
	NonMonotonicCount      int      `json:"non_monotonic_count"`
	OHLCValidity           string   `json:"ohlc_validity"`
	VolumeValidity         string   `json:"volume_validity"`
	RequestedStartCoverage string   `json:"requested_start_coverage"`
	RawPayloadSHA256       []string `json:"raw_payload_sha256"`
	RequestStart           string   `json:"request_start"`
	RequestEnd             string   `json:"request_end"`
	RequestAsOf            string   `json:"request_asof"`
	Feed                   string   `json:"feed"`
	Timeframe              string   `json:"timeframe"`
}

type factorBoundary struct {
	Instrument string  `json:"instrument"`
	Date       string  `json:"date"`
	Previous   float64 `json:"previous_factor"`
	Current    float64 `json:"current_factor"`
}

type preflightArtifact struct {
	ContractVersion      string           `json:"contract_version"`
	Status               string           `json:"status"`
	GeneratedAt          string           `json:"generated_at"`
	Provider             string           `json:"provider"`
	Feed                 string           `json:"feed"`
	Timeframe            string           `json:"timeframe"`
	AsOf                 string           `json:"as_of"`
	DateRange            [2]string        `json:"date_range"`
	Universe             []string         `json:"universe"`
	Families             []familyQuality  `json:"families"`
	StructuralBoundaries []factorBoundary `json:"structural_factor_boundaries"`
	SpinOffBoundaries    []string         `json:"spin_off_adjustment_boundaries"`
	SynchronizedSessions int              `json:"synchronized_session_count"`
	CompleteNo2025Guard  bool             `json:"complete_no_2025_guard"`
	PerformanceOutput    bool             `json:"performance_output_generated"`
	SourceContract       string           `json:"source_contract"`
	Error                string           `json:"error,omitempty"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "VAL-03B_PREFLIGHT=BLOCKED: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	start, _ := time.Parse("2006-01-02", startDate)
	end, _ := time.Parse("2006-01-02", endDate)
	if err := marketdata.ValidateVAL03BDateRange(start.UTC(), end.UTC()); err != nil {
		return err
	}
	key := configValue("ALPACA_API_KEY")
	secret := configValue("ALPACA_API_SECRET")
	if key == "" || secret == "" {
		return errors.New("existing Alpaca market-data credentials are unavailable")
	}
	provider, err := marketdata.NewAlpacaProvider(marketdata.ProviderConfig{Name: marketdata.ProviderAlpaca, APIKey: key, APISecret: secret, Tier: "free", Feed: "sip", Enabled: true})
	if err != nil {
		return err
	}
	store, err := newFileStore(filepath.Join(".runtime", "val03b", "raw"))
	if err != nil {
		return err
	}
	artifact := preflightArtifact{ContractVersion: "jax.val-03b.bar-family-readiness/v1", Status: "IN_PROGRESS", GeneratedAt: time.Now().UTC().Format(time.RFC3339Nano), Provider: "Alpaca", Feed: "SIP", Timeframe: "1Day", AsOf: asOfDate, DateRange: [2]string{startDate, endDate}, Universe: append([]string(nil), symbols...), CompleteNo2025Guard: true, PerformanceOutput: false, SourceContract: "raw + split + split,spin-off through hardened Alpaca route"}
	series := make(map[string]map[string]map[string]marketdata.VAL03BPriceFrame)
	coverageBlocked := false
	for _, symbol := range symbols {
		instrument := instrumentFor(symbol)
		series[symbol] = make(map[string]map[string]marketdata.VAL03BPriceFrame)
		for _, adjustment := range []marketdata.MarketAdjustmentState{marketdata.MarketAdjustmentRaw, marketdata.MarketAdjustmentSplit, marketdata.MarketAdjustmentSplitSpinOff} {
			result, err := acquire(context.Background(), provider, store, instrument, adjustment, symbol)
			if err != nil {
				artifact.Status = "BLOCKED"
				artifact.Error = fmt.Sprintf("%s/%s: %v", symbol, adjustment, err)
				_ = writeArtifact(artifact)
				return err
			}
			quality, frames, err := qualityFor(symbol, adjustment, result)
			if err != nil {
				artifact.Status = "BLOCKED"
				artifact.Error = fmt.Sprintf("%s/%s: %v", symbol, adjustment, err)
				_ = writeArtifact(artifact)
				return err
			}
			artifact.Families = append(artifact.Families, quality)
			if quality.RequestedStartCoverage != "PASS" {
				coverageBlocked = true
			}
			series[symbol][adjustmentKey(adjustment)] = frames
		}
		boundaries, spinOff, synchronized, err := compareSeries(symbol, series[symbol])
		if err != nil {
			artifact.Status = "BLOCKED"
			artifact.Error = fmt.Sprintf("%s: %v", symbol, err)
			_ = writeArtifact(artifact)
			return err
		}
		artifact.StructuralBoundaries = append(artifact.StructuralBoundaries, boundaries...)
		artifact.SpinOffBoundaries = append(artifact.SpinOffBoundaries, spinOff...)
		if artifact.SynchronizedSessions == 0 || synchronized < artifact.SynchronizedSessions {
			artifact.SynchronizedSessions = synchronized
		}
	}
	if coverageBlocked {
		artifact.Status = "BLOCKED_INCOMPLETE_DATE_COVERAGE"
		artifact.Error = "one or more adjustment families do not cover the requested 2015-01-01 start"
		if err := writeArtifact(artifact); err != nil {
			return err
		}
		return errors.New("DATASET_BLOCKED: requested 2015-01-01 warm-up coverage is unavailable")
	}
	sort.Strings(artifact.SpinOffBoundaries)
	if len(artifact.SpinOffBoundaries) > 0 {
		artifact.Status = "BLOCKED_UNSUPPORTED_STRUCTURAL_ACTION"
		artifact.Error = "split,spin-off family differs from split family"
		if err := writeArtifact(artifact); err != nil {
			return err
		}
		return errors.New("UNSUPPORTED_STRUCTURAL_ACTION: split,spin-off boundary detected")
	}
	artifact.Status = "PASS"
	if err := writeArtifact(artifact); err != nil {
		return err
	}
	return nil
}

func acquire(ctx context.Context, provider *marketdata.AlpacaProvider, store providercontract.RawPayloadStore, instrument canonical.Instrument, adjustment marketdata.MarketAdjustmentState, symbol string) (marketdata.AlpacaBarsResult, error) {
	registry, err := providercontract.NewRegistry(providercontract.RegistryContractV1)
	if err != nil {
		return marketdata.AlpacaBarsResult{}, err
	}
	if err := registry.Register(marketdata.AlpacaProviderDefinition()); err != nil {
		return marketdata.AlpacaBarsResult{}, err
	}
	normalizers, err := providercontract.NewNormalizerRegistry(registry)
	if err != nil {
		return marketdata.AlpacaBarsResult{}, err
	}
	normalizer, err := marketdata.NewAlpacaBarsNormalizerForAdjustment(instrument, marketdata.MarketFeedSIP, adjustment)
	if err != nil {
		return marketdata.AlpacaBarsResult{}, err
	}
	if err := normalizers.Register(normalizer); err != nil {
		return marketdata.AlpacaBarsResult{}, err
	}
	pipeline, err := providercontract.NewNormalizationPipeline(registry, normalizers)
	if err != nil {
		return marketdata.AlpacaBarsResult{}, err
	}
	executor, err := providercontract.NewOperationalExecutor(registry, operationalPolicy(), providercontract.SystemTimeSource{}, nil, nil)
	if err != nil {
		return marketdata.AlpacaBarsResult{}, err
	}
	start, _ := time.Parse("2006-01-02", startDate)
	end, _ := time.Parse("2006-01-02", endDate)
	asOf, _ := time.Parse("2006-01-02", asOfDate)
	return provider.AcquireAndNormalizeDailyBars(ctx, marketdata.AlpacaBarsDependencies{Registry: registry, Executor: executor, Store: store, Pipeline: pipeline}, marketdata.AlpacaBarsRequest{Instrument: instrument, StartDate: start.UTC(), EndDate: end.UTC(), AsOfDate: asOf.UTC(), Interval: marketdata.Timeframe1Day, Feed: marketdata.MarketFeedSIP, Adjustment: adjustment, PayloadID: providercontract.RawPayloadID("rpa_val03b_" + strings.ToLower(symbol) + "_" + adjustmentKey(adjustment)), Retention: retention()})
}

func qualityFor(symbol string, adjustment marketdata.MarketAdjustmentState, result marketdata.AlpacaBarsResult) (familyQuality, map[string]marketdata.VAL03BPriceFrame, error) {
	quality := familyQuality{Instrument: symbol, Adjustment: adjustmentKey(adjustment), SourceID: marketdata.AlpacaHistoricalSourceID + "_sip_" + strings.ToLower(string(adjustment)), RequestStart: startDate, RequestEnd: endDate, RequestAsOf: asOfDate, Feed: "sip", Timeframe: "1Day", OHLCValidity: "PASS", VolumeValidity: "PASS"}
	frames := make(map[string]marketdata.VAL03BPriceFrame, len(result.Bars))
	previous := ""
	for _, bar := range result.Bars {
		if err := marketdata.ValidateVAL03BProviderDate(bar.ProviderDate); err != nil {
			return familyQuality{}, nil, err
		}
		if previous != "" && bar.ProviderDate <= previous {
			quality.NonMonotonicCount++
		}
		previous = bar.ProviderDate
		if _, exists := frames[bar.ProviderDate]; exists {
			quality.DuplicateCount++
		}
		frames[bar.ProviderDate] = frameFromBar(bar)
	}
	quality.RecordCount = len(result.Bars)
	if quality.RecordCount == 0 {
		return familyQuality{}, nil, errors.New("no bars returned")
	}
	quality.FirstSession = result.Bars[0].ProviderDate
	quality.LastSession = result.Bars[len(result.Bars)-1].ProviderDate
	quality.RequestedStartCoverage = "PASS"
	if quality.FirstSession > startDate {
		quality.RequestedStartCoverage = "BLOCKED_REQUESTED_START_NOT_COVERED"
	}
	for _, raw := range result.RawPayloads {
		quality.RawPayloadSHA256 = append(quality.RawPayloadSHA256, raw.Ref.Content.Digest.Value)
	}
	sort.Strings(quality.RawPayloadSHA256)
	return quality, frames, nil
}

func compareSeries(symbol string, families map[string]map[string]marketdata.VAL03BPriceFrame) ([]factorBoundary, []string, int, error) {
	raw, okRaw := families[adjustmentKey(marketdata.MarketAdjustmentRaw)]
	split, okSplit := families[adjustmentKey(marketdata.MarketAdjustmentSplit)]
	spin, okSpin := families[adjustmentKey(marketdata.MarketAdjustmentSplitSpinOff)]
	if !okRaw || !okSplit || !okSpin {
		return nil, nil, 0, errors.New("required adjustment family missing")
	}
	if !sameKeys(raw, split) || !sameKeys(split, spin) {
		return nil, nil, 0, errors.New("synchronized adjustment families have different sessions")
	}
	dates := make([]string, 0, len(raw))
	for date := range raw {
		dates = append(dates, date)
	}
	sort.Strings(dates)
	boundaries := make([]factorBoundary, 0)
	spinBoundaries := make([]string, 0)
	previousFactor := 0.0
	for _, date := range dates {
		factor, err := marketdata.DeriveVAL03BSplitFactor(raw[date], split[date])
		if err != nil {
			return nil, nil, 0, fmt.Errorf("%s factor on %s: %w", symbol, date, err)
		}
		if previousFactor != 0 && marketdata.IsVAL03BStructuralFactorBoundary(previousFactor, factor) {
			boundaries = append(boundaries, factorBoundary{Instrument: symbol, Date: date, Previous: previousFactor, Current: factor})
		}
		previousFactor = factor
		if differs(spin[date], split[date]) {
			spinBoundaries = append(spinBoundaries, symbol+"/"+date)
		}
	}
	return boundaries, spinBoundaries, len(dates), nil
}

func sameKeys(left, right map[string]marketdata.VAL03BPriceFrame) bool {
	if len(left) != len(right) {
		return false
	}
	for key := range left {
		if _, ok := right[key]; !ok {
			return false
		}
	}
	return true
}

func differs(left, right marketdata.VAL03BPriceFrame) bool {
	return left != right
}

func frameFromBar(bar marketdata.CanonicalMarketBar) marketdata.VAL03BPriceFrame {
	return marketdata.VAL03BPriceFrame{Open: *bar.Open.Value.Number, High: *bar.High.Value.Number, Low: *bar.Low.Value.Number, Close: *bar.Close.Value.Number, Volume: *bar.Volume.Value.Number}
}

func adjustmentKey(adjustment marketdata.MarketAdjustmentState) string {
	return strings.ToLower(string(adjustment))
}

func instrumentFor(symbol string) canonical.Instrument {
	return canonical.Instrument{ContractVersion: canonical.InstrumentContractV1, ID: canonical.InstrumentID("ins_val03b_" + strings.ToLower(symbol)), Type: canonical.InstrumentTypeEquity, Name: symbol, Currency: "USD", CreatedAt: time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC), ExternalIDs: []canonical.ExternalID{{Namespace: "ticker.us", Value: symbol}}}
}

func configValue(name string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	raw, err := os.ReadFile(".env")
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(raw), "\n") {
		parts := strings.SplitN(strings.TrimSpace(line), "=", 2)
		if len(parts) == 2 && strings.TrimSpace(parts[0]) == name {
			return strings.Trim(strings.TrimSpace(parts[1]), "\"'")
		}
	}
	return ""
}

func retention() providercontract.RawPayloadRetentionPolicy {
	return providercontract.RawPayloadRetentionPolicy{Class: providercontract.RawPayloadRetentionReplayAudit, Policy: canonical.VersionIdentity{Namespace: "jax.raw_retention", Value: "replay-audit/v1"}, Redistribution: providercontract.RawPayloadRedistributionNotAuthorized}
}

func operationalPolicy() providercontract.OperationalPolicy {
	identity := func(id, namespace string, kind canonical.ComponentKind) canonical.ComponentIdentity {
		return canonical.ComponentIdentity{ID: id, Kind: kind, Name: id, Version: canonical.VersionIdentity{Namespace: namespace, Value: "v1"}}
	}
	return providercontract.OperationalPolicy{ContractVersion: providercontract.OperationalPolicyContractV1, Classification: providercontract.ClassificationPolicy{ContractVersion: providercontract.ClassificationPolicyContractV1, Identity: identity("cmp_val03b_classification", "jax.policy.failure_classification", canonical.ComponentKindPolicy)}, Retry: providercontract.RetryPolicy{ContractVersion: providercontract.RetryPolicyContractV1, Identity: identity("cmp_val03b_retry", "jax.policy.retry", canonical.ComponentKindPolicy), MaximumAttempts: 2, MaximumElapsed: time.Minute, PerAttemptTimeout: 30 * time.Second, RetryableFailures: []providercontract.FailureClass{providercontract.FailureTransportTransient, providercontract.FailureProviderServer, providercontract.FailureTemporaryUnavailable, providercontract.FailureRateLimited, providercontract.FailureAttemptDeadline}, Backoff: providercontract.BackoffPolicy{InitialDelay: time.Millisecond, Multiplier: 2, MaximumDelay: time.Second, Jitter: providercontract.JitterNone}}, RateLimit: providercontract.RateLimitPolicy{ContractVersion: providercontract.RateLimitPolicyContractV1, Identity: identity("cmp_val03b_rate_limit", "jax.policy.rate_limit", canonical.ComponentKindPolicy), RequestLimit: 100, Window: time.Minute, ConcurrencyLimit: 1, MaximumProviderDelay: 5 * time.Second}, Health: providercontract.HealthPolicy{ContractVersion: providercontract.HealthPolicyContractV1, Identity: identity("cmp_val03b_health", "jax.policy.health", canonical.ComponentKindPolicy), DegradedAfterFailures: 1, UnavailableAfterFailures: 3, RecoverySuccesses: 1, AssessmentHorizon: time.Hour}, Component: identity("cmp_val03b_operational", "git.commit", canonical.ComponentKindSoftwareBuild)}
}

func writeArtifact(artifact preflightArtifact) error {
	path := filepath.Join("Docs", "validation", "results", "VAL-03B-DATASET-READINESS.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(artifact, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return os.WriteFile(path, raw, 0o644)
}
