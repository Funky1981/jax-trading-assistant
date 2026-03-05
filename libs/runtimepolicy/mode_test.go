package runtimepolicy

import "testing"

func TestParseMode(t *testing.T) {
	tests := []struct {
		in      string
		want    Mode
		wantErr bool
	}{
		{in: "", want: ModeDev},
		{in: "dev", want: ModeDev},
		{in: "test", want: ModeTest},
		{in: "research", want: ModeResearch},
		{in: "paper", want: ModePaper},
		{in: "live", want: ModeLive},
		{in: "production", want: ModeLive},
		{in: "prod", want: ModeLive},
		{in: "invalid", wantErr: true},
	}

	for _, tt := range tests {
		got, err := ParseMode(tt.in)
		if tt.wantErr {
			if err == nil {
				t.Fatalf("ParseMode(%q): expected error", tt.in)
			}
			continue
		}
		if err != nil {
			t.Fatalf("ParseMode(%q): %v", tt.in, err)
		}
		if got != tt.want {
			t.Fatalf("ParseMode(%q): got=%q want=%q", tt.in, got, tt.want)
		}
	}
}

func TestModePolicyFlags(t *testing.T) {
	if !ModeDev.AllowsSyntheticBacktest() || !ModeTest.AllowsSyntheticBacktest() {
		t.Fatalf("dev/test should allow synthetic backtests")
	}
	if ModePaper.AllowsSyntheticBacktest() || ModeLive.AllowsSyntheticBacktest() || ModeResearch.AllowsSyntheticBacktest() {
		t.Fatalf("paper/live/research must not allow synthetic backtests")
	}
}

func TestResolveModeFromEnv(t *testing.T) {
	t.Setenv("JAX_RUNTIME_MODE", "paper")
	mode, explicit, err := ResolveModeFromEnv()
	if err != nil {
		t.Fatalf("ResolveModeFromEnv error: %v", err)
	}
	if mode != ModePaper {
		t.Fatalf("ResolveModeFromEnv mode=%q want=%q", mode, ModePaper)
	}
	if !explicit {
		t.Fatalf("ResolveModeFromEnv explicit=false want=true")
	}
}

func TestResolveModeFromEnv_DefaultDev(t *testing.T) {
	t.Setenv("JAX_RUNTIME_MODE", "")
	t.Setenv("APP_RUNTIME_MODE", "")
	t.Setenv("APP_ENV", "")
	t.Setenv("ENV", "")

	mode, explicit, err := ResolveModeFromEnv()
	if err != nil {
		t.Fatalf("ResolveModeFromEnv error: %v", err)
	}
	if mode != ModeDev {
		t.Fatalf("ResolveModeFromEnv mode=%q want=%q", mode, ModeDev)
	}
	if explicit {
		t.Fatalf("ResolveModeFromEnv explicit=true want=false")
	}
}
