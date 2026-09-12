package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"jax-trading-assistant/internal/modules/hypevidence"
)

const (
	parentDatasetID   = "hyp-event-001a-sec-8k-alpaca-sip-2016-2025-v1"
	parentManifestSHA = "db2b454793e50f28c34c9c2a7d91f798741936528752de7bd27cb52072a4864d"
	datasetID         = "hyp-event-001a-sec-8k-alpaca-sip-2016-2025-evidence-v2"
	acquisitionRule   = "sec_accession_evidence_packet_v2_primary_and_same_accession_text_documents_excluding_duplicate_submission_text"
	maxResponseBytes  = 16 << 20
)

type acquisitionEvent struct {
	hypevidence.Event
	Year int
}

type fetcher struct {
	client  *http.Client
	product string
	contact string
	delay   time.Duration
	last    time.Time
}

type tokenStats struct {
	Count  int `json:"count"`
	Total  int `json:"total_tokens"`
	Median int `json:"median_tokens"`
	P75    int `json:"p75_tokens"`
	P90    int `json:"p90_tokens"`
	P95    int `json:"p95_tokens"`
	P99    int `json:"p99_tokens"`
	Max    int `json:"max_tokens"`
}

type tokenProfile struct {
	ContractVersion string     `json:"contract_version"`
	DatasetID       string     `json:"dataset_id"`
	Includes        string     `json:"includes"`
	Excludes        string     `json:"excludes"`
	Development     tokenStats `json:"development"`
	Validation      tokenStats `json:"validation"`
	OOS             tokenStats `json:"oos_2024"`
	All2016To2024   tokenStats `json:"all_2016_2024"`
}

func main() {
	eventsPath := flag.String("events", "data/datasets/hyp-event-001a/dataset-2016-2025-sip-sec-v1/normalized/events-qualified.json", "qualified event panel")
	outDir := flag.String("out", "data/datasets/hyp-event-001a/dataset-2016-2025-evidence-v1", "private evidence dataset output")
	maxEvents := flag.Int("max-events", 0, "bounded event limit; zero means all")
	eventID := flag.String("event-id", "", "optional event ID for bounded diagnosis")
	reuseRoot := flag.String("reuse-root", "", "optional prior private acquisition root for immutable raw/index reuse")
	delayMS := flag.Int("delay-ms", 150, "minimum milliseconds between SEC requests")
	flag.Parse()

	if err := run(*eventsPath, *outDir, *maxEvents, *eventID, *reuseRoot, time.Duration(*delayMS)*time.Millisecond); err != nil {
		fmt.Fprintln(os.Stderr, "evidence acquisition failed:", err)
		os.Exit(1)
	}
}

func run(eventsPath, outDir string, maxEvents int, eventID, reuseRoot string, delay time.Duration) error {
	if strings.TrimSpace(os.Getenv("SEC_USER_AGENT")) == "" || strings.TrimSpace(os.Getenv("SEC_CONTACT")) == "" {
		return errors.New("SEC_USER_AGENT and SEC_CONTACT must be loaded through the existing local configuration")
	}
	events, err := loadEvents(eventsPath)
	if err != nil {
		return err
	}
	if maxEvents > 0 && maxEvents < len(events) {
		events = events[:maxEvents]
	}
	if eventID != "" {
		filtered := events[:0]
		for _, event := range events {
			if event.EventID == eventID {
				filtered = append(filtered, event)
			}
		}
		events = filtered
	}
	if err := os.MkdirAll(filepath.Join(outDir, "packets"), 0o755); err != nil {
		return err
	}
	f := &fetcher{client: &http.Client{Timeout: 45 * time.Second}, product: os.Getenv("SEC_USER_AGENT"), contact: os.Getenv("SEC_CONTACT"), delay: delay}
	stats := make(map[int][]int)
	manifest := hypevidence.Manifest{ContractVersion: hypevidence.EvidenceDatasetContractV1, DatasetID: datasetID, ParentDatasetID: parentDatasetID, ParentManifestSHA256: parentManifestSHA, AcquisitionRule: acquisitionRule, NormalizerVersion: hypevidence.NormalizerVersion, StudyPeriod: "2016-01-01/2025-12-31", EventCount: len(events), Year2024Status: "SEMANTICALLY_SEALED_UNTIL_CLASSIFIER_FREEZE", Year2025Status: "HASH_AND_INVENTORY_ONLY", SemanticSeal2024: "SEALED_FOR_DESIGN_AND_OUTCOME_USE", HoldoutSeal2025: "SEALED_FOR_SEMANTIC_AND_OUTCOME_USE", TokenProfilePath: "token-profile.json"}
	var failures []string
	for index, event := range events {
		packetPath := filepath.Join(outDir, "packets", event.EventID+".json")
		if existing, readErr := os.ReadFile(packetPath); readErr == nil {
			var packet hypevidence.Packet
			if json.Unmarshal(existing, &packet) == nil && packet.Validate() == nil {
				manifest.PacketCount++
				manifest.DocumentCount += len(packet.Documents)
				eventTokens := 0
				for _, document := range packet.Documents {
					if event.Year <= 2024 && document.NormalizedAvailable {
						eventTokens += document.EstimatedTokens
					}
				}
				if event.Year <= 2024 {
					stats[event.Year] = append(stats[event.Year], eventTokens)
				}
				continue
			}
		}
		packet, documentCount, tokens, acquireErr := acquireEvent(f, event, outDir, reuseRoot)
		if acquireErr != nil {
			failures = append(failures, event.EventID+": "+acquireErr.Error())
			continue
		}
		data, err := json.MarshalIndent(packet, "", "  ")
		if err != nil {
			return err
		}
		if err := atomicWrite(packetPath, append(data, '\n')); err != nil {
			return err
		}
		manifest.PacketCount++
		manifest.DocumentCount += documentCount
		for year, values := range tokens {
			stats[year] = append(stats[year], values...)
		}
		if (index+1)%25 == 0 || index+1 == len(events) {
			fmt.Printf("processed %d/%d events; failures=%d\n", index+1, len(events), len(failures))
		}
	}
	manifest.RetrievalFailureCount = len(failures)
	manifest.MissingDocumentCount = len(failures)
	profile := buildProfile(stats)
	profileBytes, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return err
	}
	if err := atomicWrite(filepath.Join(outDir, "token-profile.json"), append(profileBytes, '\n')); err != nil {
		return err
	}
	if len(failures) > 0 {
		failureBytes, _ := json.MarshalIndent(failures, "", "  ")
		if err := atomicWrite(filepath.Join(outDir, "retrieval-failures.json"), append(failureBytes, '\n')); err != nil {
			return err
		}
	}
	manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	manifest.ManifestSHA256 = sha256Hex(manifestBytes)
	manifestBytes, err = json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	if err := atomicWrite(filepath.Join(outDir, "manifest.json"), append(manifestBytes, '\n')); err != nil {
		return err
	}
	return nil
}

func loadEvents(path string) ([]acquisitionEvent, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var raw []hypevidence.Event
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	result := make([]acquisitionEvent, 0, len(raw))
	for _, event := range raw {
		if event.EventEligibility != "QUALIFYING" || event.Form != "8-K" || event.Amended {
			continue
		}
		acceptance, err := time.ParseInLocation("01/02/2006 15:04:05", event.AcceptanceRaw, time.UTC)
		if err != nil {
			return nil, fmt.Errorf("event %s acceptance timestamp: %w", event.EventID, err)
		}
		result = append(result, acquisitionEvent{Event: event, Year: acceptance.Year()})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].EventID < result[j].EventID })
	return result, nil
}

func acquireEvent(f *fetcher, event acquisitionEvent, outDir, reuseRoot string) (hypevidence.Packet, int, map[int][]int, error) {
	indexURL, err := hypevidence.ArchiveIndexURL(event.CIK, event.Accession)
	if err != nil {
		return hypevidence.Packet{}, 0, nil, err
	}
	indexPath := filepath.Join(reuseRoot, "raw", event.EventID, "-index.html")
	indexBytes, err := readOrFetch(indexPath, func() ([]byte, error) { return f.get(indexURL, "text/html") })
	if err != nil {
		return hypevidence.Packet{}, 0, nil, fmt.Errorf("index: %w", err)
	}
	packetDir := filepath.Join(outDir, "raw", event.EventID)
	if err := os.MkdirAll(packetDir, 0o755); err != nil {
		return hypevidence.Packet{}, 0, nil, err
	}
	if err := atomicWrite(filepath.Join(packetDir, "-index.html"), indexBytes); err != nil {
		return hypevidence.Packet{}, 0, nil, err
	}
	inventory, err := hypevidence.ParseIndex(indexBytes)
	if err != nil {
		return hypevidence.Packet{}, 0, nil, err
	}
	selected := hypevidence.SelectTextDocuments(inventory, event.PrimaryDocument)
	primaryFound := false
	for _, document := range selected {
		if strings.EqualFold(document.Filename, event.PrimaryDocument) {
			primaryFound = true
		}
	}
	if !primaryFound {
		return hypevidence.Packet{}, 0, nil, errors.New("primary document missing from SEC index")
	}
	acceptance, _ := time.ParseInLocation("01/02/2006 15:04:05", event.AcceptanceRaw, time.UTC)
	packet := hypevidence.Packet{ContractVersion: hypevidence.PacketContractV1, EventID: event.EventID, CIK: event.CIK, Accession: event.Accession, Form: event.Form, AcceptanceTimestamp: acceptance.Format(time.RFC3339), AvailabilityCutoff: acceptance.Format(time.RFC3339), IndexURL: indexURL, IndexSHA256: hypevidence.SHA256Hex(indexBytes), Inventory: inventory, SemanticStatus: semanticStatus(event.Year)}
	eventTokens := 0
	for _, entry := range selected {
		documentURL, err := hypevidence.ArchiveDocumentURL(indexURL, entry.Href)
		if err != nil {
			return hypevidence.Packet{}, 0, nil, err
		}
		rawPath := filepath.Join(reuseRoot, "raw", event.EventID, entry.Filename)
		raw, err := readOrFetch(rawPath, func() ([]byte, error) { return f.get(documentURL, "text/html, text/plain") })
		if err != nil {
			return hypevidence.Packet{}, 0, nil, fmt.Errorf("%s: %w", entry.Filename, err)
		}
		document := hypevidence.Document{Filename: entry.Filename, DocumentType: entry.DocumentType, Description: entry.Description, SourceURL: documentURL, RawSHA256: hypevidence.SHA256Hex(raw), RawBytes: len(raw), TextEligible: true}
		if event.Year <= 2024 {
			var normalized string
			if strings.HasSuffix(strings.ToLower(entry.Filename), ".txt") {
				normalized = hypevidence.NormalizeText(raw)
			} else {
				normalized, err = hypevidence.NormalizeHTML(raw)
				if err != nil {
					return hypevidence.Packet{}, 0, nil, fmt.Errorf("%s normalize: %w", entry.Filename, err)
				}
			}
			document.NormalizedSHA256 = hypevidence.SHA256Hex([]byte(normalized))
			document.NormalizedChars = len([]rune(normalized))
			document.EstimatedTokens = hypevidence.EstimateTokens(normalized)
			document.NormalizedAvailable = true
			eventTokens += document.EstimatedTokens
		}
		if err := atomicWrite(filepath.Join(packetDir, entry.Filename), raw); err != nil {
			return hypevidence.Packet{}, 0, nil, err
		}
		packet.Documents = append(packet.Documents, document)
	}
	packetBytes, _ := json.Marshal(packet)
	packet.PacketSHA256 = sha256Hex(packetBytes)
	if err := packet.Validate(); err != nil {
		return hypevidence.Packet{}, 0, nil, err
	}
	tokens := map[int][]int{}
	if event.Year <= 2024 {
		tokens[event.Year] = []int{eventTokens}
	}
	return packet, len(packet.Documents), tokens, nil
}

func readOrFetch(filename string, fetch func() ([]byte, error)) ([]byte, error) {
	if filename != "" {
		if data, err := os.ReadFile(filename); err == nil {
			return data, nil
		}
	}
	return fetch()
}

func (f *fetcher) get(endpoint, accept string) ([]byte, error) {
	if wait := f.delay - time.Since(f.last); wait > 0 {
		time.Sleep(wait)
	}
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		request, err := http.NewRequest(http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, err
		}
		request.Header.Set("User-Agent", f.product+" "+f.contact)
		request.Header.Set("Accept", accept)
		response, err := f.client.Do(request)
		f.last = time.Now()
		if err != nil {
			lastErr = err
			continue
		}
		body, readErr := hypevidence.ReadAllBounded(response.Body, maxResponseBytes)
		_ = response.Body.Close()
		if readErr == nil && response.StatusCode >= 200 && response.StatusCode < 300 {
			return body, nil
		}
		if readErr != nil {
			lastErr = readErr
		} else {
			lastErr = fmt.Errorf("SEC returned HTTP %d", response.StatusCode)
		}
		if response.StatusCode != http.StatusTooManyRequests && response.StatusCode < 500 {
			break
		}
		time.Sleep(time.Duration(attempt+1) * time.Second)
	}
	return nil, lastErr
}

func semanticStatus(year int) string {
	if year == 2025 {
		return "SEALED_HASH_AND_INVENTORY_ONLY"
	}
	if year == 2024 {
		return "SEMANTICALLY_SEALED_UNTIL_CLASSIFIER_FREEZE"
	}
	return "AVAILABLE_FOR_CLASSIFIER_DESIGN"
}

func buildProfile(values map[int][]int) tokenProfile {
	profile := tokenProfile{ContractVersion: "jax.hyp_event_token_profile/v1", DatasetID: datasetID, Includes: "normalized SEC primary documents and same-accession text documents for 2016-01-01 through 2024-12-31", Excludes: "2025 content normalization and all outcome/semantic use"}
	development := []int{}
	for year := 2016; year <= 2021; year++ {
		development = append(development, values[year]...)
	}
	validation := append([]int{}, values[2022]...)
	validation = append(validation, values[2023]...)
	oos := append([]int{}, values[2024]...)
	profile.Development = statsFor(development)
	profile.Validation = statsFor(validation)
	profile.OOS = statsFor(values[2024])
	all := append([]int{}, development...)
	all = append(all, validation...)
	all = append(all, oos...)
	profile.All2016To2024 = statsFor(all)
	return profile
}

func statsFor(values []int) tokenStats {
	if len(values) == 0 {
		return tokenStats{}
	}
	sorted := append([]int{}, values...)
	sort.Ints(sorted)
	return tokenStats{Count: len(sorted), Total: sum(sorted), Median: percentile(sorted, .50), P75: percentile(sorted, .75), P90: percentile(sorted, .90), P95: percentile(sorted, .95), P99: percentile(sorted, .99), Max: sorted[len(sorted)-1]}
}

func percentile(values []int, p float64) int {
	index := int(float64(len(values)-1) * p)
	return values[index]
}

func sum(values []int) int {
	total := 0
	for _, value := range values {
		total += value
	}
	return total
}

func sha256Hex(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

func atomicWrite(filename string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(filename), ".evidence-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, filename)
}
