package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"slices"
	"strings"
)

const defaultETFPolicyPath = "config/etf-instrument-policy.json"

var excludedETFClasses = map[string]struct{}{
	"leveraged":  {},
	"inverse":    {},
	"volatility": {},
}

type etfInstrumentPolicy struct {
	Phase       string          `json:"phase"`
	Version     string          `json:"version"`
	Instruments []etfInstrument `json:"instruments"`
}

type etfInstrument struct {
	Symbol           string   `json:"symbol"`
	AssetClass       string   `json:"assetClass"`
	InstrumentType   string   `json:"instrumentType"`
	ETFClass         string   `json:"etfClass"`
	TradableModes    []string `json:"tradableModes"`
	EligibilityState string   `json:"eligibilityState"`
	EffectiveDate    string   `json:"effectiveDate"`
	ChangeOwner      string   `json:"changeOwner"`
}

type etfEligibilityDecision struct {
	Symbol     string
	Known      bool
	IsETF      bool
	Allowed    bool
	ReasonCode string
	Reason     string
}

type tradingETFPolicyResponse struct {
	Phase         string   `json:"phase"`
	Version       string   `json:"version"`
	PolicyPath    string   `json:"policyPath"`
	ApprovedETFs  []string `json:"approvedEtfs"`
	ExcludedETFs  []string `json:"excludedEtfs"`
	LoadedDefault bool     `json:"loadedDefault"`
}

func loadActiveETFInstrumentPolicy() (*etfInstrumentPolicy, string, bool) {
	policyPath := strings.TrimSpace(os.Getenv("ETF_INSTRUMENT_POLICY_PATH"))
	if policyPath == "" {
		policyPath = defaultETFPolicyPath
	}
	policy, err := loadETFInstrumentPolicy(policyPath)
	if err == nil {
		return policy, policyPath, false
	}
	return defaultETFInstrumentPolicy(), policyPath, true
}

func loadETFInstrumentPolicy(path string) (*etfInstrumentPolicy, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read etf policy: %w", err)
	}
	var policy etfInstrumentPolicy
	if err := json.Unmarshal(data, &policy); err != nil {
		return nil, fmt.Errorf("decode etf policy: %w", err)
	}
	if len(policy.Instruments) == 0 {
		return nil, fmt.Errorf("etf policy has no instruments")
	}
	return &policy, nil
}

func evaluateETFPhase1Eligibility(policy *etfInstrumentPolicy, symbol, mode string) etfEligibilityDecision {
	normalizedSymbol := strings.ToUpper(strings.TrimSpace(symbol))
	if normalizedSymbol == "" || policy == nil {
		return etfEligibilityDecision{Symbol: normalizedSymbol, Allowed: true}
	}
	normalizedMode := strings.ToLower(strings.TrimSpace(mode))
	for _, instrument := range policy.Instruments {
		if strings.ToUpper(strings.TrimSpace(instrument.Symbol)) != normalizedSymbol {
			continue
		}
		isETF := strings.EqualFold(strings.TrimSpace(instrument.AssetClass), "etf") ||
			strings.EqualFold(strings.TrimSpace(instrument.InstrumentType), "etf")
		if !isETF {
			return etfEligibilityDecision{
				Symbol:  normalizedSymbol,
				Known:   true,
				IsETF:   false,
				Allowed: true,
			}
		}
		if _, excluded := excludedETFClasses[strings.ToLower(strings.TrimSpace(instrument.ETFClass))]; excluded {
			return etfEligibilityDecision{
				Symbol:     normalizedSymbol,
				Known:      true,
				IsETF:      true,
				Allowed:    false,
				ReasonCode: "etf_class_excluded",
				Reason:     fmt.Sprintf("ETF class %q is excluded for phase-1 trading", instrument.ETFClass),
			}
		}
		if !strings.EqualFold(strings.TrimSpace(instrument.EligibilityState), "approved") {
			return etfEligibilityDecision{
				Symbol:     normalizedSymbol,
				Known:      true,
				IsETF:      true,
				Allowed:    false,
				ReasonCode: "etf_not_approved",
				Reason:     fmt.Sprintf("ETF eligibility state %q is not approved", instrument.EligibilityState),
			}
		}
		if !containsFold(instrument.TradableModes, normalizedMode) {
			return etfEligibilityDecision{
				Symbol:     normalizedSymbol,
				Known:      true,
				IsETF:      true,
				Allowed:    false,
				ReasonCode: "etf_mode_not_permitted",
				Reason:     fmt.Sprintf("ETF %s is not tradable in %s mode", normalizedSymbol, normalizedMode),
			}
		}
		return etfEligibilityDecision{
			Symbol:  normalizedSymbol,
			Known:   true,
			IsETF:   true,
			Allowed: true,
		}
	}
	return etfEligibilityDecision{
		Symbol:  normalizedSymbol,
		Known:   false,
		IsETF:   false,
		Allowed: true,
	}
}

func tradingETFPolicyHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		policy, path, loadedDefault := loadActiveETFInstrumentPolicy()
		resp := tradingETFPolicyResponse{
			Phase:         policy.Phase,
			Version:       policy.Version,
			PolicyPath:    path,
			ApprovedETFs:  []string{},
			ExcludedETFs:  []string{},
			LoadedDefault: loadedDefault,
		}
		for _, instrument := range policy.Instruments {
			if !strings.EqualFold(strings.TrimSpace(instrument.AssetClass), "etf") &&
				!strings.EqualFold(strings.TrimSpace(instrument.InstrumentType), "etf") {
				continue
			}
			symbol := strings.ToUpper(strings.TrimSpace(instrument.Symbol))
			if symbol == "" {
				continue
			}
			decision := evaluateETFPhase1Eligibility(policy, symbol, "paper")
			if decision.Allowed {
				resp.ApprovedETFs = append(resp.ApprovedETFs, symbol)
				continue
			}
			resp.ExcludedETFs = append(resp.ExcludedETFs, symbol)
		}
		slices.Sort(resp.ApprovedETFs)
		slices.Sort(resp.ExcludedETFs)
		jsonOK(w, resp)
	}
}

func containsFold(items []string, target string) bool {
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item), strings.TrimSpace(target)) {
			return true
		}
	}
	return false
}

func defaultETFInstrumentPolicy() *etfInstrumentPolicy {
	return &etfInstrumentPolicy{
		Phase:   "phase-1",
		Version: "fallback-v1",
		Instruments: []etfInstrument{
			{Symbol: "SPY", AssetClass: "ETF", InstrumentType: "ETF", ETFClass: "plain_vanilla", TradableModes: []string{"paper"}, EligibilityState: "approved", EffectiveDate: "2026-05-11", ChangeOwner: "trading-risk"},
			{Symbol: "QQQ", AssetClass: "ETF", InstrumentType: "ETF", ETFClass: "plain_vanilla", TradableModes: []string{"paper"}, EligibilityState: "approved", EffectiveDate: "2026-05-11", ChangeOwner: "trading-risk"},
			{Symbol: "DIA", AssetClass: "ETF", InstrumentType: "ETF", ETFClass: "plain_vanilla", TradableModes: []string{"paper"}, EligibilityState: "approved", EffectiveDate: "2026-05-11", ChangeOwner: "trading-risk"},
			{Symbol: "IWM", AssetClass: "ETF", InstrumentType: "ETF", ETFClass: "plain_vanilla", TradableModes: []string{"paper"}, EligibilityState: "approved", EffectiveDate: "2026-05-11", ChangeOwner: "trading-risk"},
			{Symbol: "XLK", AssetClass: "ETF", InstrumentType: "ETF", ETFClass: "plain_vanilla", TradableModes: []string{"paper"}, EligibilityState: "approved", EffectiveDate: "2026-05-11", ChangeOwner: "trading-risk"},
			{Symbol: "XLF", AssetClass: "ETF", InstrumentType: "ETF", ETFClass: "plain_vanilla", TradableModes: []string{"paper"}, EligibilityState: "approved", EffectiveDate: "2026-05-11", ChangeOwner: "trading-risk"},
			{Symbol: "XLE", AssetClass: "ETF", InstrumentType: "ETF", ETFClass: "plain_vanilla", TradableModes: []string{"paper"}, EligibilityState: "approved", EffectiveDate: "2026-05-11", ChangeOwner: "trading-risk"},
			{Symbol: "SMH", AssetClass: "ETF", InstrumentType: "ETF", ETFClass: "plain_vanilla", TradableModes: []string{"paper"}, EligibilityState: "approved", EffectiveDate: "2026-05-11", ChangeOwner: "trading-risk"},
			{Symbol: "SOXX", AssetClass: "ETF", InstrumentType: "ETF", ETFClass: "plain_vanilla", TradableModes: []string{"paper"}, EligibilityState: "approved", EffectiveDate: "2026-05-11", ChangeOwner: "trading-risk"},
			{Symbol: "TLT", AssetClass: "ETF", InstrumentType: "ETF", ETFClass: "plain_vanilla", TradableModes: []string{"paper"}, EligibilityState: "approved", EffectiveDate: "2026-05-11", ChangeOwner: "trading-risk"},
			{Symbol: "GLD", AssetClass: "ETF", InstrumentType: "ETF", ETFClass: "plain_vanilla", TradableModes: []string{"paper"}, EligibilityState: "approved", EffectiveDate: "2026-05-11", ChangeOwner: "trading-risk"},
			{Symbol: "TQQQ", AssetClass: "ETF", InstrumentType: "ETF", ETFClass: "leveraged", TradableModes: []string{}, EligibilityState: "excluded", EffectiveDate: "2026-05-11", ChangeOwner: "trading-risk"},
			{Symbol: "SQQQ", AssetClass: "ETF", InstrumentType: "ETF", ETFClass: "inverse", TradableModes: []string{}, EligibilityState: "excluded", EffectiveDate: "2026-05-11", ChangeOwner: "trading-risk"},
			{Symbol: "UVXY", AssetClass: "ETF", InstrumentType: "ETF", ETFClass: "volatility", TradableModes: []string{}, EligibilityState: "excluded", EffectiveDate: "2026-05-11", ChangeOwner: "trading-risk"},
		},
	}
}
