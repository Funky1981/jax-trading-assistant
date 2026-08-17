package aishadow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestC1E2OfflineFreezeIsNoNetworkAndV2HashOnly(t *testing.T) {
	if _, _, err := GenerateC1E2OfflineFreeze(filepath.Clean("../../.."), t.TempDir(), true); err == nil {
		t.Fatal("hosted authorization=true must fail closed")
	}
	output := t.TempDir()
	path, hash, err := GenerateC1E2OfflineFreeze(filepath.Clean("../../.."), output, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(hash) != 64 {
		t.Fatalf("artifact hash is invalid: %q", hash)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var freeze C1E2OfflineFreeze
	if err := json.Unmarshal(raw, &freeze); err != nil {
		t.Fatal(err)
	}
	if freeze.ProviderContact || freeze.Inference || freeze.DatabaseMutation || freeze.TradingMutation || freeze.HostedInferenceAuthorized || freeze.C1DActive || freeze.C1E3ProfilesExecuted || freeze.V2ContentsInspected {
		t.Fatalf("offline safety proof is false: %+v", freeze)
	}
	if !freeze.C1E3ProfilesConfigured || len(freeze.Profiles) != 2 || len(freeze.V2Files) != 6 {
		t.Fatalf("offline freeze lacks registered metadata: %+v", freeze)
	}
	for _, file := range freeze.V2Files {
		if !file.Preserved || file.ActualSHA256 != file.ExpectedSHA256 {
			t.Fatalf("v2 hash was not preserved: %+v", file)
		}
	}
}
