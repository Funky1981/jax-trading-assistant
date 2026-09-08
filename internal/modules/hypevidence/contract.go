// Package hypevidence defines the immutable SEC evidence extension used by
// HYP-EVENT-001A. It is deliberately separate from recommendation and
// execution domains.
package hypevidence

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"path"
	"strings"
	"time"
)

const (
	EvidenceDatasetContractV1 = "jax.hyp_event_evidence_dataset/v1"
	PacketContractV1          = "jax.hyp_event_evidence_packet/v1"
	NormalizerVersion         = "sec_html_text_normalizer/v1"
)

type Event struct {
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

type Document struct {
	Filename            string `json:"filename"`
	DocumentType        string `json:"document_type,omitempty"`
	Description         string `json:"description,omitempty"`
	SourceURL           string `json:"source_url"`
	RawSHA256           string `json:"raw_sha256"`
	NormalizedSHA256    string `json:"normalized_sha256,omitempty"`
	RawBytes            int    `json:"raw_bytes"`
	NormalizedChars     int    `json:"normalized_chars,omitempty"`
	EstimatedTokens     int    `json:"estimated_tokens,omitempty"`
	TextEligible        bool   `json:"text_eligible"`
	NormalizedAvailable bool   `json:"normalized_available"`
	ExclusionReason     string `json:"exclusion_reason,omitempty"`
}

type IndexDocument struct {
	Filename     string
	DocumentType string
	Description  string
	Href         string
}

type Packet struct {
	ContractVersion     string          `json:"contract_version"`
	EventID             string          `json:"event_id"`
	CIK                 string          `json:"cik"`
	Accession           string          `json:"accession"`
	Form                string          `json:"form"`
	AcceptanceTimestamp string          `json:"acceptance_timestamp"`
	AvailabilityCutoff  string          `json:"availability_cutoff"`
	IndexURL            string          `json:"index_url"`
	IndexSHA256         string          `json:"index_sha256"`
	Inventory           []IndexDocument `json:"document_inventory"`
	Documents           []Document      `json:"documents"`
	PacketSHA256        string          `json:"packet_sha256"`
	SemanticStatus      string          `json:"semantic_status"`
}

type Manifest struct {
	ContractVersion       string `json:"contract_version"`
	DatasetID             string `json:"dataset_id"`
	ParentDatasetID       string `json:"parent_dataset_id"`
	ParentManifestSHA256  string `json:"parent_manifest_sha256"`
	AcquisitionRule       string `json:"acquisition_rule"`
	NormalizerVersion     string `json:"normalizer_version"`
	StudyPeriod           string `json:"study_period"`
	EventCount            int    `json:"event_count"`
	PacketCount           int    `json:"packet_count"`
	DocumentCount         int    `json:"document_count"`
	MissingDocumentCount  int    `json:"missing_document_count"`
	RetrievalFailureCount int    `json:"retrieval_failure_count"`
	Year2024Status        string `json:"year_2024_status"`
	Year2025Status        string `json:"year_2025_status"`
	SemanticSeal2024      string `json:"semantic_seal_2024"`
	HoldoutSeal2025       string `json:"holdout_seal_2025"`
	TokenProfilePath      string `json:"token_profile_path"`
	ManifestSHA256        string `json:"manifest_sha256"`
}

func SHA256Hex(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

func ArchiveIndexURL(cik, accession string) (string, error) {
	cik = strings.TrimLeft(strings.TrimSpace(cik), "0")
	if cik == "" || strings.ContainsAny(cik, "/\\") {
		return "", errors.New("valid SEC CIK is required")
	}
	accession = strings.TrimSpace(accession)
	compact := strings.ReplaceAll(accession, "-", "")
	if len(compact) != 18 || strings.ContainsAny(compact, "/\\") {
		return "", errors.New("valid SEC accession is required")
	}
	return "https://www.sec.gov/Archives/edgar/data/" + cik + "/" + compact + "/" + accession + "-index.html", nil
}

func ArchiveDocumentURL(indexURL, href string) (string, error) {
	base, err := url.Parse(indexURL)
	if err != nil {
		return "", err
	}
	ref, err := url.Parse(strings.TrimSpace(href))
	if err != nil || ref.IsAbs() && ref.Host != base.Host {
		return "", errors.New("document URL must remain on the SEC archive host")
	}
	joined := *base
	if ref.IsAbs() {
		joined = *ref
	} else if strings.HasPrefix(ref.Path, "/") {
		joined.Path = ref.Path
		joined.RawQuery, joined.Fragment = "", ""
	} else {
		joined.Path = path.Join(path.Dir(base.Path), ref.Path)
		joined.RawQuery, joined.Fragment = "", ""
	}
	if joined.Scheme != "https" || joined.Host != "www.sec.gov" || strings.Contains(ref.Path, "..") {
		return "", errors.New("document URL is outside the SEC archive")
	}
	return joined.String(), nil
}

func (p Packet) Validate() error {
	if p.ContractVersion != PacketContractV1 || p.EventID == "" || p.CIK == "" || p.Accession == "" || p.Form != "8-K" {
		return errors.New("invalid SEC evidence packet identity")
	}
	if p.AvailabilityCutoff == "" || p.SemanticStatus == "" {
		return errors.New("SEC packet cutoff and semantic status are required")
	}
	cutoff, err := time.Parse(time.RFC3339, p.AvailabilityCutoff)
	if err != nil || cutoff.Location() != time.UTC {
		return errors.New("SEC packet cutoff must be RFC3339 UTC")
	}
	for _, doc := range p.Documents {
		if doc.Filename == "" || len(doc.RawSHA256) != 64 || doc.SourceURL == "" {
			return fmt.Errorf("invalid SEC packet document %q", doc.Filename)
		}
	}
	return nil
}
