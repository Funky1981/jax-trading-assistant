package exploratorypaper

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

// VersionedSessionCalendar is the production configuration wrapper around the
// PAPER-01 SessionCalendar. It generates every covered date deterministically;
// dates outside coverage remain unknown and therefore fail closed.
type VersionedSessionCalendar struct {
	Version        string                   `json:"version"`
	Market         string                   `json:"market"`
	Timezone       string                   `json:"timezone"`
	OpenTime       string                   `json:"openTime"`
	CloseTime      string                   `json:"closeTime"`
	CoverageStart  string                   `json:"coverageStart"`
	CoverageEnd    string                   `json:"coverageEnd"`
	WeekendDays    []int                    `json:"weekendDays"`
	Holidays       []string                 `json:"holidays"`
	SpecialWindows map[string]SessionWindow `json:"specialWindows,omitempty"`
	Provenance     string                   `json:"provenance"`
	ApplicableMode string                   `json:"applicableMode"`
}

func LoadVersionedSessionCalendar(path string) (VersionedSessionCalendar, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return VersionedSessionCalendar{}, fmt.Errorf("read session calendar %q: %w", path, err)
	}
	var manifest VersionedSessionCalendar
	if err := json.Unmarshal(data, &manifest); err != nil {
		return VersionedSessionCalendar{}, fmt.Errorf("parse session calendar %q: %w", path, err)
	}
	if err := manifest.Validate(); err != nil {
		return VersionedSessionCalendar{}, err
	}
	return manifest, nil
}

func (m VersionedSessionCalendar) Validate() error {
	if m.Version == "" || m.Market != "US_EQUITIES_REGULAR" || m.Timezone == "" || m.OpenTime == "" || m.CloseTime == "" || m.CoverageStart == "" || m.CoverageEnd == "" || m.Provenance == "" || m.ApplicableMode != "PAPER" {
		return fmt.Errorf("versioned PAPER session calendar identity is incomplete")
	}
	location, err := time.LoadLocation(m.Timezone)
	if err != nil {
		return fmt.Errorf("session calendar timezone is unsupported: %w", err)
	}
	start, err := time.ParseInLocation("2006-01-02", m.CoverageStart, location)
	if err != nil {
		return fmt.Errorf("invalid calendar coverage start: %w", err)
	}
	end, err := time.ParseInLocation("2006-01-02", m.CoverageEnd, location)
	if err != nil || end.Before(start) {
		return fmt.Errorf("invalid calendar coverage end")
	}
	base := SessionCalendar{Sessions: map[string]bool{"probe": true}, Timezone: m.Timezone, OpenTime: m.OpenTime, CloseTime: m.CloseTime}
	if err := base.Validate(); err != nil {
		return err
	}
	weekends := map[int]bool{}
	for _, day := range m.WeekendDays {
		if day < 0 || day > 6 {
			return fmt.Errorf("calendar weekend day is invalid")
		}
		weekends[day] = true
	}
	holidaySet := map[string]bool{}
	for _, raw := range m.Holidays {
		if _, err := time.ParseInLocation("2006-01-02", raw, location); err != nil {
			return fmt.Errorf("invalid calendar holiday %q", raw)
		}
		holidaySet[raw] = true
	}
	for day := start; !day.After(end); day = day.AddDate(0, 0, 1) {
		key := day.Format("2006-01-02")
		if holidaySet[key] || weekends[int(day.Weekday())] {
			continue
		}
		if override, ok := m.SpecialWindows[key]; ok {
			if _, _, err := (SessionCalendar{Sessions: map[string]bool{"probe": true}, Timezone: m.Timezone, OpenTime: override.OpenTime, CloseTime: override.CloseTime}).sessionWindow(); err != nil {
				return fmt.Errorf("invalid special session %q: %w", key, err)
			}
		}
	}
	return nil
}

func (m VersionedSessionCalendar) ContentHash() string {
	data, _ := json.Marshal(m)
	digest := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func (m VersionedSessionCalendar) SessionCalendar() (SessionCalendar, error) {
	if err := m.Validate(); err != nil {
		return SessionCalendar{}, err
	}
	location, _ := time.LoadLocation(m.Timezone)
	start, _ := time.ParseInLocation("2006-01-02", m.CoverageStart, location)
	end, _ := time.ParseInLocation("2006-01-02", m.CoverageEnd, location)
	holidaySet := map[string]bool{}
	for _, holiday := range m.Holidays {
		holidaySet[holiday] = true
	}
	weekends := map[int]bool{}
	for _, day := range m.WeekendDays {
		weekends[day] = true
	}
	sessions := map[string]bool{}
	for day := start; !day.After(end); day = day.AddDate(0, 0, 1) {
		key := day.Format("2006-01-02")
		sessions[key] = !holidaySet[key] && !weekends[int(day.Weekday())]
	}
	return SessionCalendar{Sessions: sessions, Timezone: m.Timezone, OpenTime: m.OpenTime, CloseTime: m.CloseTime, Version: m.Version, Market: m.Market, CoverageStart: m.CoverageStart, CoverageEnd: m.CoverageEnd, SessionWindows: m.SpecialWindows}, nil
}

type EligibleInstrument struct {
	InstrumentID string `json:"instrumentId"`
	Symbol       string `json:"symbol"`
	Exchange     string `json:"exchange"`
	Market       string `json:"market"`
}

type EligibleUniverseManifest struct {
	Version        string               `json:"version"`
	EffectiveFrom  string               `json:"effectiveFrom"`
	Market         string               `json:"market"`
	ApplicableMode string               `json:"applicableMode"`
	Provenance     string               `json:"provenance"`
	Instruments    []EligibleInstrument `json:"instruments"`
}

func LoadEligibleUniverse(path string) (EligibleUniverseManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return EligibleUniverseManifest{}, fmt.Errorf("read eligible universe %q: %w", path, err)
	}
	var manifest EligibleUniverseManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return EligibleUniverseManifest{}, fmt.Errorf("parse eligible universe %q: %w", path, err)
	}
	if err := manifest.Validate(); err != nil {
		return EligibleUniverseManifest{}, err
	}
	return manifest, nil
}

func (m EligibleUniverseManifest) Validate() error {
	if m.Version == "" || m.EffectiveFrom == "" || m.Market != "US_EQUITIES_REGULAR" || m.ApplicableMode != "PAPER" || m.Provenance == "" || len(m.Instruments) == 0 {
		return fmt.Errorf("eligible PAPER universe identity is incomplete")
	}
	if _, err := time.Parse("2006-01-02", m.EffectiveFrom); err != nil {
		return fmt.Errorf("eligible universe effective date is invalid")
	}
	seen := map[string]bool{}
	for _, instrument := range m.Instruments {
		if instrument.InstrumentID == "" || instrument.Symbol == "" || instrument.Market != m.Market || instrument.Exchange == "" {
			return fmt.Errorf("eligible universe contains an invalid instrument")
		}
		key := strings.ToUpper(instrument.InstrumentID)
		if seen[key] {
			return fmt.Errorf("eligible universe contains duplicate instrument %q", instrument.InstrumentID)
		}
		seen[key] = true
		if instrument.Exchange != "NASDAQ" && instrument.Exchange != "NYSE" && instrument.Exchange != "NYSEARCA" {
			return fmt.Errorf("eligible universe contains unsupported exchange %q", instrument.Exchange)
		}
	}
	return nil
}

func (m EligibleUniverseManifest) ContentHash() string {
	data, _ := json.Marshal(m)
	digest := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func (m EligibleUniverseManifest) Symbols() []string {
	result := make([]string, 0, len(m.Instruments))
	for _, instrument := range m.Instruments {
		result = append(result, instrument.Symbol)
	}
	sort.Strings(result)
	return result
}
