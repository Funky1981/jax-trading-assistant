// Command phase03-exit-gate performs a bounded, fail-closed preflight for the
// real-source Phase 03 exit demonstration. It never substitutes fixtures or
// model output for an acquisition and never creates trading state.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type sourceStatus struct {
	Family string `json:"family"`
	Status string `json:"status"`
	Detail string `json:"detail"`
}

type report struct {
	Representative string         `json:"representative"`
	InstrumentID   string         `json:"instrument_id"`
	IssuerID       string         `json:"issuer_id"`
	Statuses       []sourceStatus `json:"statuses"`
	Result         string         `json:"result"`
}

func main() {
	r := report{
		Representative: "AAPL / Apple Inc.",
		InstrumentID:   "ins_aapl_common",
		IssuerID:       "iss_apple",
		Statuses: []sourceStatus{
			marketStatus(),
			secStatus(),
			{Family: "MACRO_CONTEXT", Status: "LIVE_SMOKE_SEPARATE", Detail: "Use the accepted keyless Treasury/Cboe live smoke command; this preflight does not claim a packet item without its raw acquisition reference."},
		},
		Result: "PHASE 03 EXIT CONDITION NOT YET DEMONSTRATED",
	}

	encoded, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(string(encoded))
	for _, status := range r.Statuses {
		if status.Status == "BLOCKED" {
			os.Exit(2)
		}
	}
	os.Exit(2)
}

func marketStatus() sourceStatus {
	if strings.TrimSpace(os.Getenv("ALPACA_API_KEY")) != "" && strings.TrimSpace(os.Getenv("ALPACA_API_SECRET")) != "" {
		return sourceStatus{Family: "MARKET", Status: "CONFIGURED", Detail: "zero-cost Alpaca Basic development source is configured; use an explicit SIP or IEX feed and provide its raw provenance to the packet"}
	}
	if strings.TrimSpace(os.Getenv("FINANCIAL_DATASETS_API_KEY")) != "" {
		return sourceStatus{Family: "MARKET", Status: "BLOCKED", Detail: "Alpaca Basic development credentials are not configured; Financial Datasets remains accepted but its development access is externally blocked"}
	}
	return sourceStatus{Family: "MARKET", Status: "BLOCKED", Detail: "Alpaca Basic development credentials are not configured; no persisted real AAPL market raw evidence was found"}
}

func secStatus() sourceStatus {
	if strings.TrimSpace(os.Getenv("SEC_USER_AGENT")) == "" || strings.TrimSpace(os.Getenv("SEC_CONTACT")) == "" {
		return sourceStatus{Family: "COMPANY", Status: "BLOCKED", Detail: "approved SEC_USER_AGENT and SEC_CONTACT are not configured; no persisted real Apple SEC raw evidence was found"}
	}
	return sourceStatus{Family: "COMPANY", Status: "CONFIGURED", Detail: "approved SEC identity is configured; run the bounded live acquisition and provide its raw provenance to the packet"}
}
