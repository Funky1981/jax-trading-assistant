package eventdecisions

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type SubjectIdentity struct {
	Key               string
	PublicID          string
	SubjectType       string
	CanonicalLabel    string
	AssociationReason string
	EventScoped       bool
}

type SubjectObservation struct {
	EventID            uuid.UUID
	Decision           Decision
	CandidateID        *uuid.UUID
	AffectedAssets     []string
	UnknownAssets      bool
	Confidence         float64
	PublicationAt      time.Time
	ReceiptAt          time.Time
	Relationship       string
	Independence       string
	SourceGroupKey     string
	ContradictionState string
	MissingEvidence    []string
}

type SubjectEvaluation struct {
	Decision               Decision
	Reason                 string
	MissingEvidence        []string
	ResolvedAssets         []string
	UnknownAssets          bool
	CandidateID            *uuid.UUID
	LinkedEventCount       int
	SourceGroupCount       int
	IndependentSourceCount int
	PrimarySourceCount     int
	ContradictionCount     int
	FirstObservedAt        time.Time
	LatestEvidenceAt       time.Time
}

type SubjectPersistenceOutcome struct {
	SubjectPublicID   string   `json:"subjectId"`
	SubjectCreated    bool     `json:"subjectCreated"`
	LinkCreated       bool     `json:"linkCreated"`
	EvaluationCreated bool     `json:"evaluationCreated"`
	Decision          Decision `json:"decision"`
}

var (
	spacePattern      = regexp.MustCompile(`\s+`)
	nonTopicPattern   = regexp.MustCompile(`[^a-z0-9+]+`)
	companyPattern    = regexp.MustCompile(`\b(?:[A-Z][A-Za-z0-9&.-]*\s+){0,3}(?:Inc|Corp|Corporation|Ltd|PLC|Bank|Group|Holdings)\b`)
	proceedingPattern = regexp.MustCompile(`(?i)\b(?:case|docket|proceeding|order|regulation|act)\s+(?:no\.?\s*)?[a-z0-9][a-z0-9./-]{2,}\b`)
)

var institutionAliases = []struct {
	canonical string
	pattern   *regexp.Regexp
}{
	{"federal-reserve", regexp.MustCompile(`(?i)\b(?:federal reserve|the fed|fed)\b`)},
	{"european-central-bank", regexp.MustCompile(`(?i)\b(?:european central bank|ecb)\b`)},
	{"bank-of-england", regexp.MustCompile(`(?i)\b(?:bank of england|boe)\b`)},
	{"bank-of-japan", regexp.MustCompile(`(?i)\b(?:bank of japan|boj)\b`)},
	{"opec", regexp.MustCompile(`(?i)\bopec\+?\b`)},
	{"nato", regexp.MustCompile(`(?i)\bnato\b`)},
	{"us-sec", regexp.MustCompile(`(?i)\b(?:u\.?s\.? securities and exchange commission|securities and exchange commission|sec)\b`)},
	{"us-doj", regexp.MustCompile(`(?i)\b(?:u\.?s\.? department of justice|department of justice|doj)\b`)},
	{"european-union", regexp.MustCompile(`(?i)\b(?:european union|eu)\b`)},
}

var topicAnchors = map[string][]string{
	"macro_rates":      {"interest rate", "rate cut", "rate hike", "monetary policy", "policy meeting", "bond purchase"},
	"central_bank":     {"interest rate", "rate cut", "rate hike", "monetary policy", "policy meeting", "bond purchase"},
	"inflation":        {"inflation", "consumer price", "cpi", "pce", "producer price"},
	"energy_oil":       {"oil", "crude", "production cut", "supply", "refinery", "tanker", "hormuz"},
	"geopolitical":     {"sanction", "ceasefire", "missile", "invasion", "tariff", "export control"},
	"financial_credit": {"default", "bank failure", "credit", "liquidity", "capital requirement"},
	"cyber_outage":     {"cyber", "ransomware", "outage", "data breach"},
	"supply_chain":     {"supply chain", "port closure", "shipping", "factory closure"},
	"commodity_shock":  {"commodity", "shortage", "production", "export ban"},
	"semiconductor_ai": {"semiconductor", "chip", "artificial intelligence", "ai model", "export control"},
	"market_panic":     {"market crash", "volatility halt", "bank run", "liquidity"},
}

func deriveSubjectIdentity(event Event) SubjectIdentity {
	label := strings.TrimSpace(event.Headline)
	if label == "" {
		label = "Unresolved evidence subject"
	}
	eventType := strings.ToLower(strings.TrimSpace(event.EventType))
	if eventType == "" {
		eventType = "unknown"
	}
	if eventType != "unknown" {
		entities := canonicalSubjectEntities(event)
		anchors := canonicalTopicAnchors(eventType, event.Headline+" "+event.Summary)
		proceeding := normalizeTopicText(proceedingPattern.FindString(event.Headline + " " + event.Summary))
		if len(entities) > 0 && (len(anchors) > 0 || proceeding != "") {
			bucket := event.PublicationAt.UTC().Format("2006-01-02")
			parts := []string{"topic", eventType, strings.Join(entities, ","), strings.Join(anchors, ","), proceeding, canonicalJurisdiction(event), bucket}
			key := strings.Join(parts, "|")
			return subjectIdentity(key, subjectType(eventType, entities), label, "same event category, canonical entity, specific topic anchor, jurisdiction, and UTC observation date", false)
		}
		if canonical := canonicalEvidenceURL(firstNonEmpty(event.ArticleURL, firstString(event.SourceURLs))); canonical != "" {
			return subjectIdentity("canonical-url|"+canonical, "source_topic", label, "same canonical article URL and event category", false)
		}
	}
	key := "event|" + event.InboxID.String()
	return subjectIdentity(key, "unresolved_topic", label, "ambiguous evidence remains event-scoped to prevent a false merge", true)
}

func subjectIdentity(key, subjectType, label, reason string, eventScoped bool) SubjectIdentity {
	digest := sha256.Sum256([]byte(key))
	return SubjectIdentity{Key: key, PublicID: "es_" + hex.EncodeToString(digest[:12]), SubjectType: subjectType, CanonicalLabel: label, AssociationReason: reason, EventScoped: eventScoped}
}

func subjectType(eventType string, entities []string) string {
	for _, entity := range entities {
		if strings.HasPrefix(entity, "asset:") || strings.HasPrefix(entity, "company:") {
			return "company_catalyst"
		}
	}
	switch eventType {
	case "macro_rates", "central_bank", "inflation":
		return "macroeconomic_release"
	case "geopolitical":
		return "geopolitical_development"
	case "semiconductor_ai", "energy_oil", "commodity_shock", "supply_chain":
		return "sector_event"
	default:
		return "market_relevant_topic"
	}
}

func canonicalSubjectEntities(event Event) []string {
	seen := map[string]bool{}
	for _, asset := range normalizedAssets(event.AffectedAssets) {
		seen["asset:"+asset] = true
	}
	text := event.Headline + " " + event.Summary
	for _, alias := range institutionAliases {
		if alias.pattern.MatchString(text) {
			seen["org:"+alias.canonical] = true
		}
	}
	for _, company := range companyPattern.FindAllString(text, -1) {
		if normalized := normalizeTopicText(company); normalized != "" {
			seen["company:"+normalized] = true
		}
	}
	out := make([]string, 0, len(seen))
	for value := range seen {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func canonicalTopicAnchors(eventType, text string) []string {
	lower := strings.ToLower(text)
	out := []string{}
	for _, anchor := range topicAnchors[eventType] {
		if strings.Contains(lower, anchor) {
			out = append(out, anchor)
		}
	}
	sort.Strings(out)
	return out
}

func canonicalJurisdiction(event Event) string {
	if region := normalizeTopicText(event.Region); region != "" {
		return region
	}
	text := strings.ToLower(event.Headline + " " + event.Summary)
	if institutionAliases[0].pattern.MatchString(text) || institutionAliases[6].pattern.MatchString(text) || institutionAliases[7].pattern.MatchString(text) {
		return "us"
	}
	if institutionAliases[1].pattern.MatchString(text) || institutionAliases[8].pattern.MatchString(text) {
		return "eu"
	}
	if institutionAliases[2].pattern.MatchString(text) {
		return "uk"
	}
	if institutionAliases[3].pattern.MatchString(text) {
		return "japan"
	}
	for _, item := range []struct {
		value string
		terms []string
	}{
		{"us", []string{"united states", "u.s.", " us ", "federal reserve", "white house"}},
		{"uk", []string{"united kingdom", " u.k.", "bank of england", "britain"}},
		{"eu", []string{"european union", "european central bank", " eurozone"}},
		{"japan", []string{"japan", "bank of japan"}},
	} {
		for _, term := range item.terms {
			if strings.Contains(" "+text+" ", term) {
				return item.value
			}
		}
	}
	return "unknown"
}

func canonicalEvidenceURL(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Hostname() == "" {
		return ""
	}
	parsed.Fragment = ""
	parsed.Host = strings.ToLower(parsed.Host)
	for key := range parsed.Query() {
		lower := strings.ToLower(key)
		if strings.HasPrefix(lower, "utm_") || lower == "fbclid" || lower == "gclid" || lower == "mc_cid" || lower == "mc_eid" {
			query := parsed.Query()
			query.Del(key)
			parsed.RawQuery = query.Encode()
		}
	}
	parsed.Path = strings.TrimSuffix(parsed.Path, "/")
	return parsed.String()
}

func evidenceSourceGroup(event Event) (string, bool) {
	rawURL := firstNonEmpty(event.ArticleURL, firstString(event.SourceURLs))
	canonical := canonicalEvidenceURL(rawURL)
	host := ""
	if parsed, err := url.Parse(canonical); err == nil {
		host = strings.ToLower(parsed.Hostname())
	}
	primary := isOfficialPrimaryHost(host)
	if primary {
		identity := firstNonEmpty(event.SourceNativeID, canonical, event.SourceEventID)
		return stableDigest("primary|" + host + "|" + identity), true
	}
	textSignature := normalizeTopicText(event.Headline) + "\n" + normalizeTopicText(event.Summary)
	if len(textSignature) >= 80 {
		return stableDigest("syndication|" + textSignature), false
	}
	if canonical != "" {
		return stableDigest("url|" + canonical), false
	}
	return stableDigest("unknown|" + event.Source + "|" + event.SourceEventID), false
}

func isOfficialPrimaryHost(host string) bool {
	if strings.HasSuffix(host, ".gov") || strings.HasSuffix(host, ".gov.uk") || strings.HasSuffix(host, ".europa.eu") {
		return true
	}
	for _, official := range []string{"federalreserve.gov", "ecb.europa.eu", "bankofengland.co.uk", "boj.or.jp", "opec.org", "sec.gov"} {
		if host == official || strings.HasSuffix(host, "."+official) {
			return true
		}
	}
	return false
}

func contradictionState(event Event) string {
	text := strings.ToLower(event.Headline + " " + event.Summary)
	for _, phrase := range []string{"denies report", "report is false", "reports are false", "no plans to", "withdraws proposal", "proposal withdrawn", "deal cancelled", "deal canceled", "ceasefire collapsed", "decision reversed", "order vacated"} {
		if strings.Contains(text, phrase) {
			return "contradicts"
		}
	}
	return "corroborates"
}

func evaluateSubject(observations []SubjectObservation, rules Ruleset, now time.Time) SubjectEvaluation {
	result := SubjectEvaluation{Decision: DecisionNoTrade, Reason: "available evidence is not currently trade-relevant", MissingEvidence: []string{}}
	if len(observations) == 0 {
		result.MissingEvidence = []string{"genuine_event_evidence"}
		return result
	}
	groups := map[string]bool{}
	independent := map[string]bool{}
	primary := map[string]bool{}
	assets := []string{}
	anyWatch, anyCandidate := false, false
	var candidateID *uuid.UUID
	for index, item := range observations {
		if index == 0 || item.PublicationAt.Before(result.FirstObservedAt) {
			result.FirstObservedAt = item.PublicationAt
		}
		if item.PublicationAt.After(result.LatestEvidenceAt) {
			result.LatestEvidenceAt = item.PublicationAt
		}
		groups[item.SourceGroupKey] = true
		if item.Independence == "primary" || item.Independence == "independent" {
			independent[item.SourceGroupKey] = true
		}
		if item.Independence == "primary" {
			primary[item.SourceGroupKey] = true
		}
		if item.ContradictionState == "contradicts" {
			result.ContradictionCount++
		}
		assets = append(assets, item.AffectedAssets...)
		anyWatch = anyWatch || item.Decision == DecisionWatch
		if item.Decision == DecisionCandidate && item.CandidateID != nil {
			anyCandidate = true
			id := *item.CandidateID
			candidateID = &id
		}
	}
	result.LinkedEventCount = len(observations)
	result.SourceGroupCount = len(groups)
	result.IndependentSourceCount = len(independent)
	result.PrimarySourceCount = len(primary)
	result.ResolvedAssets = normalizedAssets(assets)
	result.UnknownAssets = len(result.ResolvedAssets) == 0
	fresh := !result.LatestEvidenceAt.IsZero() && !result.LatestEvidenceAt.After(now.Add(5*time.Minute)) && now.Sub(result.LatestEvidenceAt) <= time.Duration(rules.SubjectFreshnessHours)*time.Hour
	if result.ContradictionCount > 0 {
		result.Decision = DecisionNoTrade
		result.Reason = "explicit contradictory evidence prevents continued candidate readiness"
		result.MissingEvidence = []string{"resolved_contradiction"}
		return normalizeSubjectEvaluation(result)
	}
	if !fresh {
		result.Decision = DecisionNoTrade
		result.Reason = "the latest linked evidence is stale under the deterministic subject ruleset"
		result.MissingEvidence = []string{"fresh_market_context"}
		return normalizeSubjectEvaluation(result)
	}
	if anyCandidate && candidateID != nil && !result.UnknownAssets && result.IndependentSourceCount >= rules.SubjectCandidateIndependentMin && result.PrimarySourceCount > 0 {
		result.Decision = DecisionCandidate
		result.CandidateID = candidateID
		result.Reason = "an existing complete candidate is supported by fresh, resolved, independently grouped primary evidence"
		return normalizeSubjectEvaluation(result)
	}
	if anyWatch || anyCandidate {
		result.Decision = DecisionWatch
		result.Reason = "the subject is potentially relevant but still lacks required corroboration, specificity, asset resolution, or candidate context"
		if result.UnknownAssets {
			result.MissingEvidence = append(result.MissingEvidence, "truthful_asset_mapping")
		}
		if result.IndependentSourceCount < rules.SubjectCandidateIndependentMin {
			result.MissingEvidence = append(result.MissingEvidence, "independent_source_corroboration")
		}
		if result.PrimarySourceCount == 0 {
			result.MissingEvidence = append(result.MissingEvidence, "primary_source_evidence")
		}
		if !anyCandidate {
			result.MissingEvidence = append(result.MissingEvidence, "complete_structured_trade_candidate")
		}
	}
	return normalizeSubjectEvaluation(result)
}

func normalizeSubjectEvaluation(result SubjectEvaluation) SubjectEvaluation {
	result.MissingEvidence = uniqueSorted(result.MissingEvidence)
	result.ResolvedAssets = normalizedAssets(result.ResolvedAssets)
	if result.Decision != DecisionCandidate {
		result.CandidateID = nil
	}
	return result
}

func (s *Store) PersistSubjectEvaluation(ctx context.Context, tx pgx.Tx, event Event, rules Ruleset, evaluatedAt time.Time) (SubjectPersistenceOutcome, error) {
	if tx == nil {
		return SubjectPersistenceOutcome{}, fmt.Errorf("subject evaluation requires a transaction")
	}
	identity := deriveSubjectIdentity(event)
	var subjectID uuid.UUID
	var subjectPublicID, currentDecision string
	var projectionVersion int
	var existingSubjectID uuid.NullUUID
	err := tx.QueryRow(ctx, `SELECT subject_id FROM evidence_subject_events WHERE genuine_event_id=$1`, event.InboxID).Scan(&existingSubjectID)
	if err != nil && err != pgx.ErrNoRows {
		return SubjectPersistenceOutcome{}, fmt.Errorf("find existing event subject: %w", err)
	}
	createdSubject := false
	if existingSubjectID.Valid {
		subjectID = existingSubjectID.UUID
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, "evidence-subject:"+subjectID.String()); err != nil {
			return SubjectPersistenceOutcome{}, fmt.Errorf("lock evidence subject: %w", err)
		}
	} else {
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, "evidence-subject-key:"+identity.Key); err != nil {
			return SubjectPersistenceOutcome{}, fmt.Errorf("lock evidence subject identity: %w", err)
		}
		row := tx.QueryRow(ctx, `
			INSERT INTO evidence_subjects (
				public_id,deterministic_subject_key,subject_type,canonical_label,current_decision,
				current_decision_reason,first_observed_at,latest_evidence_at,ruleset_version,created_at,updated_at
			) VALUES ($1,$2,$3,$4,'NO_TRADE','awaiting deterministic evaluation',$5,$5,$6,$7,$7)
			ON CONFLICT (deterministic_subject_key) DO NOTHING
			RETURNING id
		`, identity.PublicID, identity.Key, identity.SubjectType, identity.CanonicalLabel, event.PublicationAt, rules.SubjectRulesetVersion, evaluatedAt)
		if err := row.Scan(&subjectID); err == nil {
			createdSubject = true
		} else if err != pgx.ErrNoRows {
			return SubjectPersistenceOutcome{}, fmt.Errorf("insert evidence subject: %w", err)
		}
		if !createdSubject {
			if err := tx.QueryRow(ctx, `SELECT id FROM evidence_subjects WHERE deterministic_subject_key=$1`, identity.Key).Scan(&subjectID); err != nil {
				return SubjectPersistenceOutcome{}, fmt.Errorf("load evidence subject: %w", err)
			}
		}
	}
	if err := tx.QueryRow(ctx, `SELECT public_id,current_decision,projection_version FROM evidence_subjects WHERE id=$1 FOR UPDATE`, subjectID).Scan(&subjectPublicID, &currentDecision, &projectionVersion); err != nil {
		return SubjectPersistenceOutcome{}, fmt.Errorf("lock evidence subject projection: %w", err)
	}
	groupKey, primary := evidenceSourceGroup(event)
	var groupExists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM evidence_subject_events WHERE subject_id=$1 AND source_group_key=$2)`, subjectID, groupKey).Scan(&groupExists); err != nil {
		return SubjectPersistenceOutcome{}, fmt.Errorf("inspect source group: %w", err)
	}
	independence := "unknown"
	if primary && !groupExists {
		independence = "primary"
	} else if groupExists {
		independence = "not_independent"
	}
	contradiction := contradictionState(event)
	relationship := "corroborating"
	if createdSubject {
		relationship = "originating"
	} else if contradiction == "contradicts" {
		relationship = "contradicting"
	} else if groupExists {
		relationship = "duplicate"
	} else if identity.EventScoped {
		relationship = "context"
	}
	contribution := "potentially relevant context"
	if contradiction == "contradicts" {
		contribution = "explicit contradiction"
	} else if independence == "primary" {
		contribution = "direct primary-source evidence"
	} else if independence == "not_independent" {
		contribution = "repeated report; readiness is not inflated"
	}
	command, err := tx.Exec(ctx, `
		INSERT INTO evidence_subject_events (
			subject_id,genuine_event_id,relationship_type,association_reason,source_independence,
			source_group_key,evidence_contribution,contradiction_state,publication_at,receipt_at,linked_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		ON CONFLICT (genuine_event_id) DO NOTHING
	`, subjectID, event.InboxID, relationship, identity.AssociationReason, independence, groupKey, contribution, contradiction, event.PublicationAt, event.ReceiptAt, evaluatedAt)
	if err != nil {
		return SubjectPersistenceOutcome{}, fmt.Errorf("link subject evidence: %w", err)
	}
	linkCreated := command.RowsAffected() == 1
	observations, err := loadSubjectObservations(ctx, tx, subjectID)
	if err != nil {
		return SubjectPersistenceOutcome{}, err
	}
	evaluation := evaluateSubject(observations, rules, evaluatedAt)
	snapshot := map[string]any{
		"linkedEventCount": evaluation.LinkedEventCount, "sourceGroupCount": evaluation.SourceGroupCount,
		"independentSourceCount": evaluation.IndependentSourceCount, "primarySourceCount": evaluation.PrimarySourceCount,
		"contradictionCount": evaluation.ContradictionCount, "resolvedAssets": evaluation.ResolvedAssets,
		"unknownAssets": evaluation.UnknownAssets, "inputEventIds": subjectObservationIDs(observations),
	}
	snapshotRaw, _ := json.Marshal(snapshot)
	fingerprintRaw, _ := json.Marshal(struct {
		Rules        string               `json:"rules"`
		Observations []SubjectObservation `json:"observations"`
	}{rules.SubjectRulesetVersion, observations})
	fingerprint := stableDigest(string(fingerprintRaw))
	idempotency := stableDigest(subjectID.String() + "|" + rules.SubjectRulesetVersion + "|" + fingerprint)
	var existingEvaluation uuid.UUID
	if err := tx.QueryRow(ctx, `SELECT id FROM evidence_subject_evaluations WHERE idempotency_identity=$1`, idempotency).Scan(&existingEvaluation); err == nil {
		return SubjectPersistenceOutcome{SubjectPublicID: subjectPublicID, SubjectCreated: createdSubject, LinkCreated: linkCreated, EvaluationCreated: false, Decision: evaluation.Decision}, nil
	} else if err != pgx.ErrNoRows {
		return SubjectPersistenceOutcome{}, fmt.Errorf("find subject evaluation replay: %w", err)
	}
	missing := evaluation.MissingEvidence
	if missing == nil {
		missing = []string{}
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO evidence_subject_evaluations (
			subject_id,previous_decision,new_decision,deterministic_reason,missing_evidence,evidence_snapshot,
			evidence_set_fingerprint,ruleset_version,evaluated_at,triggering_event_id,idempotency_identity
		) VALUES ($1,$2,$3,$4,$5,$6::jsonb,$7,$8,$9,$10,$11)
	`, subjectID, currentDecision, evaluation.Decision, evaluation.Reason, missing, string(snapshotRaw), fingerprint, rules.SubjectRulesetVersion, evaluatedAt, event.InboxID, idempotency); err != nil {
		return SubjectPersistenceOutcome{}, fmt.Errorf("insert subject evaluation: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE evidence_subjects SET
			current_decision=$2,current_decision_reason=$3,current_missing_evidence=$4,
			first_observed_at=$5,latest_evidence_at=$6,latest_evaluation_at=$7,ruleset_version=$8,
			resolved_assets=$9,unknown_assets=$10,candidate_id=$11,projection_version=$12,updated_at=$7
		WHERE id=$1
	`, subjectID, evaluation.Decision, evaluation.Reason, missing, evaluation.FirstObservedAt, evaluation.LatestEvidenceAt,
		evaluatedAt, rules.SubjectRulesetVersion, evaluation.ResolvedAssets, evaluation.UnknownAssets, evaluation.CandidateID, projectionVersion+1); err != nil {
		return SubjectPersistenceOutcome{}, fmt.Errorf("update subject projection: %w", err)
	}
	if evaluation.Decision == DecisionCandidate && evaluation.CandidateID != nil {
		if _, err := tx.Exec(ctx, `
			INSERT INTO evidence_subject_candidates(subject_id,candidate_id,linked_at,link_reason)
			VALUES($1,$2,$3,$4)
			ON CONFLICT (subject_id) DO UPDATE SET candidate_id=EXCLUDED.candidate_id,link_reason=EXCLUDED.link_reason
		`, subjectID, *evaluation.CandidateID, evaluatedAt, evaluation.Reason); err != nil {
			return SubjectPersistenceOutcome{}, fmt.Errorf("link subject candidate: %w", err)
		}
	}
	return SubjectPersistenceOutcome{SubjectPublicID: subjectPublicID, SubjectCreated: createdSubject, LinkCreated: linkCreated, EvaluationCreated: true, Decision: evaluation.Decision}, nil
}

func loadSubjectObservations(ctx context.Context, tx pgx.Tx, subjectID uuid.UUID) ([]SubjectObservation, error) {
	rows, err := tx.Query(ctx, `
		SELECT l.genuine_event_id,d.decision,d.candidate_id,d.affected_assets,d.unknown_assets,d.confidence::float8,
			l.publication_at,l.receipt_at,l.relationship_type,l.source_independence,l.source_group_key,
			l.contradiction_state,d.missing_evidence
		FROM evidence_subject_events l
		JOIN genuine_event_decisions d ON d.source_inbox_event_id=l.genuine_event_id AND d.is_current
		WHERE l.subject_id=$1
		ORDER BY l.publication_at,l.genuine_event_id
	`, subjectID)
	if err != nil {
		return nil, fmt.Errorf("load subject evidence set: %w", err)
	}
	defer rows.Close()
	observations := []SubjectObservation{}
	for rows.Next() {
		var item SubjectObservation
		var candidateID uuid.NullUUID
		if err := rows.Scan(&item.EventID, &item.Decision, &candidateID, &item.AffectedAssets, &item.UnknownAssets, &item.Confidence,
			&item.PublicationAt, &item.ReceiptAt, &item.Relationship, &item.Independence, &item.SourceGroupKey,
			&item.ContradictionState, &item.MissingEvidence); err != nil {
			return nil, fmt.Errorf("scan subject evidence set: %w", err)
		}
		if candidateID.Valid {
			id := candidateID.UUID
			item.CandidateID = &id
		}
		observations = append(observations, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("subject evidence rows: %w", err)
	}
	return observations, nil
}

func subjectObservationIDs(observations []SubjectObservation) []string {
	out := make([]string, len(observations))
	for index, item := range observations {
		out[index] = item.EventID.String()
	}
	return out
}

func stableDigest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func normalizeTopicText(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = nonTopicPattern.ReplaceAllString(value, " ")
	return spacePattern.ReplaceAllString(value, " ")
}

func firstString(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
