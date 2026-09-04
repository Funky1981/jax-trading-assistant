package fred

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"jax-trading-assistant/libs/contracts/canonical"
	providercontract "jax-trading-assistant/libs/contracts/provider"
	"jax-trading-assistant/libs/releaseevidence"
)

type ReleaseCalendarRequest struct {
	RealtimeStart                 *Date
	RealtimeEnd                   *Date
	IncludeReleaseDatesWithNoData bool
	MetadataPayloadID             providercontract.RawPayloadID
	DatesPayloadID                providercontract.RawPayloadID
	Retention                     providercontract.RawPayloadRetentionPolicy
	PageSize                      int
	MaxPages                      int
}

type ReleaseCalendarResult struct {
	Executions   []providercontract.ExecutionResult
	RawPayloads  []providercontract.RawPayloadDescriptor
	Releases     []releaseevidence.EconomicRelease
	Completeness CompletenessState
}

type fredReleaseMetadataEnvelope struct {
	RealtimeStart string                      `json:"realtime_start"`
	RealtimeEnd   string                      `json:"realtime_end"`
	OrderBy       string                      `json:"order_by"`
	SortOrder     string                      `json:"sort_order"`
	Count         int                         `json:"count"`
	Offset        int                         `json:"offset"`
	Limit         int                         `json:"limit"`
	Releases      []fredReleaseMetadataRecord `json:"releases"`
}

type fredReleaseMetadataRecord struct {
	ID            int    `json:"id"`
	RealtimeStart string `json:"realtime_start"`
	RealtimeEnd   string `json:"realtime_end"`
	Name          string `json:"name"`
	PressRelease  *bool  `json:"press_release"`
	Link          string `json:"link"`
}

type fredReleaseDatesEnvelope struct {
	RealtimeStart string                  `json:"realtime_start"`
	RealtimeEnd   string                  `json:"realtime_end"`
	OrderBy       string                  `json:"order_by"`
	SortOrder     string                  `json:"sort_order"`
	Count         int                     `json:"count"`
	Offset        int                     `json:"offset"`
	Limit         int                     `json:"limit"`
	Dates         []fredReleaseDateRecord `json:"release_dates"`
}

type fredReleaseDateRecord struct {
	ReleaseID          int    `json:"release_id"`
	ReleaseName        string `json:"release_name"`
	Date               string `json:"date"`
	ReleaseLastUpdated string `json:"release_last_updated"`
}

type fredReleaseMetadataPage struct {
	Page     PageInfo
	Period   *RealtimePeriod
	Releases []fredReleaseMetadataRecord
}

type fredReleaseDatesPage struct {
	Page   PageInfo
	Period *RealtimePeriod
	Dates  []fredReleaseDateRecord
}

// AcquireReleaseCalendar obtains FRED release identity metadata and
// source-published release dates. A date remains date-only; FRED does not
// supply an authoritative intraday release time through these endpoints.
func (provider *Provider) AcquireReleaseCalendar(ctx context.Context, deps Dependencies, request ReleaseCalendarRequest) (ReleaseCalendarResult, error) {
	result := ReleaseCalendarResult{Completeness: CompletenessIncomplete}
	if err := validateReleaseCalendarRequest(request); err != nil {
		return result, err
	}
	if err := provider.validateDependencies(deps); err != nil {
		return result, err
	}
	pageSize := request.PageSize
	if pageSize == 0 {
		pageSize = DefaultPageSize
	}
	maxPages := request.MaxPages
	if maxPages == 0 {
		maxPages = provider.config.MaxPages
	}

	metadata, metadataRefs, metadataExec, err := provider.acquireReleaseMetadata(ctx, deps, request, pageSize, maxPages)
	result.Executions = append(result.Executions, metadataExec...)
	result.RawPayloads = append(result.RawPayloads, metadataRefs...)
	if err != nil {
		return result, err
	}
	dates, dateRefs, dateExec, err := provider.acquireReleaseDates(ctx, deps, request, pageSize, maxPages)
	result.Executions = append(result.Executions, dateExec...)
	result.RawPayloads = append(result.RawPayloads, dateRefs...)
	if err != nil {
		return result, err
	}

	metadataByID := make(map[int]fredReleaseMetadataRecord, len(metadata))
	metadataRefByID := make(map[int]providercontract.RawPayloadRef, len(metadata))
	for pageIndex, page := range metadata {
		ref := metadataRefs[pageIndex].Ref
		for _, release := range page.Releases {
			metadataByID[release.ID] = release
			metadataRefByID[release.ID] = ref
		}
	}
	seen := make(map[string]struct{}, len(dates))
	for pageIndex, page := range dates {
		dateRef := dateRefs[pageIndex].Ref
		for _, dateRecord := range page.Dates {
			date, err := parseDate(dateRecord.Date)
			if err != nil {
				return result, fmt.Errorf("FRED release date: %w", err)
			}
			key := fmt.Sprintf("%d\x00%s", dateRecord.ReleaseID, date)
			if _, exists := seen[key]; exists {
				return result, fmt.Errorf("FRED release dates contain duplicate occurrence %q", key)
			}
			seen[key] = struct{}{}

			metadataRecord, metadataOK := metadataByID[dateRecord.ReleaseID]
			name := strings.TrimSpace(dateRecord.ReleaseName)
			if name == "" && metadataOK {
				name = strings.TrimSpace(metadataRecord.Name)
			}
			if name == "" {
				return result, fmt.Errorf("FRED release %d has no authoritative name", dateRecord.ReleaseID)
			}
			dateCopy := releaseevidence.Date(date)
			period, err := parseOptionalReleasePeriod(page.Period)
			if err != nil {
				return result, err
			}
			refs := []providercontract.RawPayloadRef{dateRef}
			acquiredAt := dateRef.ReceivedAt
			if metadataOK {
				metadataRef := metadataRefByID[dateRecord.ReleaseID]
				refs = append(refs, metadataRef)
				if metadataRef.ReceivedAt.After(acquiredAt) {
					acquiredAt = metadataRef.ReceivedAt
				}
			}
			provider := ProviderIdentity
			source := SourceIdentity
			id := releaseevidence.DeterministicID(provider.ID, source.ID, strconv.Itoa(dateRecord.ReleaseID), string(dateCopy), "")
			provenance, err := releaseevidence.MakeProvenance(refs, canonical.ComponentIdentity{
				ID: "cmp_fred_release_normalizer", Kind: canonical.ComponentKindNormalizer,
				Name: "FRED release deterministic normalizer", Version: canonical.VersionIdentity{Namespace: "jax.fred.release_normalizer", Value: AdapterVersion}, Provider: &provider,
			}, id)
			if err != nil {
				return result, err
			}
			availability := releaseevidence.DataAvailabilityIndicated
			var lastUpdated *releaseevidence.Date
			if request.IncludeReleaseDatesWithNoData {
				if strings.TrimSpace(dateRecord.ReleaseLastUpdated) == "" {
					availability = releaseevidence.DataAvailabilityNoData
				}
				if strings.TrimSpace(dateRecord.ReleaseLastUpdated) != "" {
					updated := releaseevidence.Date(strings.TrimSpace(dateRecord.ReleaseLastUpdated))
					if err := updated.Validate(); err != nil {
						return result, fmt.Errorf("FRED release last updated date: %w", err)
					}
					lastUpdated = &updated
				}
			}
			release := releaseevidence.EconomicRelease{
				ContractVersion: releaseevidence.EconomicReleaseContractV1, ID: id, Provider: provider, Source: source,
				SourceReleaseID: strconv.Itoa(dateRecord.ReleaseID), Name: name, ScheduledDate: &dateCopy,
				TimingAuthority: releaseevidence.TimingDateOnlySchedule, Status: releaseevidence.ReleaseStatusUnknown,
				DataAvailability: availability, RealtimePeriod: period, DataLastUpdatedDate: lastUpdated,
				AcquiredAt: acquiredAt, SourcePayloads: refs, Provenance: provenance,
			}
			if metadataOK {
				release.PressRelease = metadataRecord.PressRelease
				release.SourceLink = metadataRecord.Link
			}
			if err := release.Validate(); err != nil {
				return result, err
			}
			result.Releases = append(result.Releases, release)
		}
	}
	result.Completeness = CompletenessComplete
	return result, nil
}

func (provider *Provider) acquireReleaseMetadata(ctx context.Context, deps Dependencies, request ReleaseCalendarRequest, pageSize, maxPages int) ([]fredReleaseMetadataPage, []providercontract.RawPayloadDescriptor, []providercontract.ExecutionResult, error) {
	endpoint, err := provider.endpoint("releases")
	if err != nil {
		return nil, nil, nil, err
	}
	var pages []fredReleaseMetadataPage
	var raws []providercontract.RawPayloadDescriptor
	var executions []providercontract.ExecutionResult
	seenIDs := map[int]struct{}{}
	lastReleaseID := -1
	expectedOffset, expectedCount := 0, -1
	for pageNumber := 0; pageNumber < maxPages; pageNumber++ {
		query := releaseMetadataQuery(request, pageSize, expectedOffset)
		execution, err := provider.acquireCalendar(ctx, deps, providercontract.OperationPaginatedRead, endpoint, query)
		executions = append(executions, execution)
		if err != nil {
			return pages, raws, executions, err
		}
		payloadID := releasePagePayloadID(request.MetadataPayloadID, "metadata", pageNumber)
		raw, err := provider.persistCalendar(ctx, deps, execution, payloadID, request.Retention)
		if err != nil {
			return pages, raws, executions, err
		}
		raws = append(raws, raw)
		payload, err := providercontract.RetrieveRawPayload(ctx, deps.Store, raw.Ref)
		if err != nil {
			return pages, raws, executions, err
		}
		page, err := parseReleaseMetadataPage(payload)
		if err != nil {
			return pages, raws, executions, err
		}
		if expectedCount < 0 {
			expectedCount = page.Page.Count
		} else if page.Page.Count != expectedCount || page.Page.Offset != expectedOffset {
			return pages, raws, executions, fmt.Errorf("FRED release metadata pagination boundary is inconsistent")
		}
		for _, item := range page.Releases {
			if item.ID <= lastReleaseID {
				return pages, raws, executions, fmt.Errorf("FRED release metadata is not strictly ordered by release ID")
			}
			lastReleaseID = item.ID
			if _, exists := seenIDs[item.ID]; exists {
				return pages, raws, executions, fmt.Errorf("FRED release metadata contains duplicate release ID %d", item.ID)
			}
			seenIDs[item.ID] = struct{}{}
		}
		pages = append(pages, page)
		if page.Page.Offset+len(page.Releases) >= page.Page.Count {
			return pages, raws, executions, nil
		}
		if len(page.Releases) == 0 {
			return pages, raws, executions, fmt.Errorf("FRED release metadata pagination made no progress")
		}
		expectedOffset += len(page.Releases)
	}
	return pages, raws, executions, fmt.Errorf("FRED release metadata pagination exceeded maximum pages")
}

func (provider *Provider) acquireReleaseDates(ctx context.Context, deps Dependencies, request ReleaseCalendarRequest, pageSize, maxPages int) ([]fredReleaseDatesPage, []providercontract.RawPayloadDescriptor, []providercontract.ExecutionResult, error) {
	endpoint, err := provider.endpoint("releases", "dates")
	if err != nil {
		return nil, nil, nil, err
	}
	var pages []fredReleaseDatesPage
	var raws []providercontract.RawPayloadDescriptor
	var executions []providercontract.ExecutionResult
	expectedOffset, expectedCount := 0, -1
	lastDate := ""
	for pageNumber := 0; pageNumber < maxPages; pageNumber++ {
		query := releaseDatesQuery(request, pageSize, expectedOffset)
		execution, err := provider.acquireCalendar(ctx, deps, providercontract.OperationPaginatedRead, endpoint, query)
		executions = append(executions, execution)
		if err != nil {
			return pages, raws, executions, err
		}
		payloadID := releasePagePayloadID(request.DatesPayloadID, "dates", pageNumber)
		raw, err := provider.persistCalendar(ctx, deps, execution, payloadID, request.Retention)
		if err != nil {
			return pages, raws, executions, err
		}
		raws = append(raws, raw)
		payload, err := providercontract.RetrieveRawPayload(ctx, deps.Store, raw.Ref)
		if err != nil {
			return pages, raws, executions, err
		}
		page, err := parseReleaseDatesPage(payload)
		if err != nil {
			return pages, raws, executions, err
		}
		if expectedCount < 0 {
			expectedCount = page.Page.Count
		} else if page.Page.Count != expectedCount || page.Page.Offset != expectedOffset {
			return pages, raws, executions, fmt.Errorf("FRED release dates pagination boundary is inconsistent")
		}
		for _, item := range page.Dates {
			if _, err := parseDate(item.Date); err != nil {
				return pages, raws, executions, fmt.Errorf("FRED release date: %w", err)
			}
			if lastDate != "" && item.Date < lastDate {
				return pages, raws, executions, fmt.Errorf("FRED release dates are not ordered by release date")
			}
			lastDate = item.Date
		}
		pages = append(pages, page)
		if page.Page.Offset+len(page.Dates) >= page.Page.Count {
			return pages, raws, executions, nil
		}
		if len(page.Dates) == 0 {
			return pages, raws, executions, fmt.Errorf("FRED release dates pagination made no progress")
		}
		expectedOffset += len(page.Dates)
	}
	return pages, raws, executions, fmt.Errorf("FRED release dates pagination exceeded maximum pages")
}

func (provider *Provider) acquireCalendar(ctx context.Context, deps Dependencies, kind providercontract.OperationKind, endpoint string, query url.Values) (providercontract.ExecutionResult, error) {
	operation := providercontract.Operation{ContractVersion: providercontract.OperationContractV1, Provider: ProviderIdentity, CapabilityID: providercontract.CapabilityEconomicCalendar, Kind: kind, RetrySafety: providercontract.RetrySafetyRepeatable}
	return deps.Executor.Execute(ctx, operation, func(attemptCtx context.Context, _ providercontract.AttemptContext) providercontract.ProviderAttemptResult {
		return provider.fetch(attemptCtx, endpoint, query)
	})
}

func (provider *Provider) persistCalendar(ctx context.Context, deps Dependencies, execution providercontract.ExecutionResult, payloadID providercontract.RawPayloadID, retention providercontract.RawPayloadRetentionPolicy) (providercontract.RawPayloadDescriptor, error) {
	revision := canonical.RevisionIdentity{Namespace: "fred.release.response.sha256", Value: canonical.DigestBytes(execution.RawBytes).Value}
	return providercontract.PersistRawPayload(ctx, deps.Registry, deps.Store, providercontract.RawPayloadPersistenceRequest{
		ID: payloadID, Provider: ProviderIdentity, Capability: providercontract.CapabilityEconomicCalendar, Raw: fredCalendarRawRepresentation(),
		Capture: providercontract.RawPayloadCapture{ByteForm: providercontract.RawPayloadByteFormEntityBody, ContentCodingState: providercontract.ContentCodingIdentity, CharacterEncoding: "utf-8"},
		Source:  &SourceIdentity, Revision: &revision, ReceivedAt: execution.CompletedAt, Retention: retention, Complete: true,
	}, execution.RawBytes)
}

func parseReleaseMetadataPage(raw []byte) (fredReleaseMetadataPage, error) {
	var envelope fredReleaseMetadataEnvelope
	if err := decodeOneDocument(raw, &envelope); err != nil {
		return fredReleaseMetadataPage{}, err
	}
	page := PageInfo{Count: envelope.Count, Offset: envelope.Offset, Limit: envelope.Limit}
	if err := page.Validate(1000); err != nil {
		return fredReleaseMetadataPage{}, err
	}
	if envelope.OrderBy != "release_id" || envelope.SortOrder != "asc" || len(envelope.Releases) > page.Limit {
		return fredReleaseMetadataPage{}, fmt.Errorf("FRED release metadata ordering or page shape is invalid: order_by=%q sort_order=%q count=%d offset=%d limit=%d rows=%d", envelope.OrderBy, envelope.SortOrder, envelope.Count, envelope.Offset, envelope.Limit, len(envelope.Releases))
	}
	period, err := parseOptionalReleasePeriodStrings(envelope.RealtimeStart, envelope.RealtimeEnd)
	if err != nil {
		return fredReleaseMetadataPage{}, err
	}
	return fredReleaseMetadataPage{Page: page, Period: period, Releases: envelope.Releases}, nil
}

func parseReleaseDatesPage(raw []byte) (fredReleaseDatesPage, error) {
	var envelope fredReleaseDatesEnvelope
	if err := decodeOneDocument(raw, &envelope); err != nil {
		return fredReleaseDatesPage{}, err
	}
	page := PageInfo{Count: envelope.Count, Offset: envelope.Offset, Limit: envelope.Limit}
	if err := page.Validate(10000); err != nil {
		return fredReleaseDatesPage{}, err
	}
	if envelope.OrderBy != "release_date" || envelope.SortOrder != "asc" || len(envelope.Dates) > page.Limit {
		return fredReleaseDatesPage{}, fmt.Errorf("FRED release dates ordering or page shape is invalid: order_by=%q sort_order=%q count=%d offset=%d limit=%d rows=%d", envelope.OrderBy, envelope.SortOrder, envelope.Count, envelope.Offset, envelope.Limit, len(envelope.Dates))
	}
	period, err := parseOptionalReleasePeriodStrings(envelope.RealtimeStart, envelope.RealtimeEnd)
	if err != nil {
		return fredReleaseDatesPage{}, err
	}
	return fredReleaseDatesPage{Page: page, Period: period, Dates: envelope.Dates}, nil
}

func parseOptionalReleasePeriod(page *RealtimePeriod) (*releaseevidence.RealtimePeriod, error) {
	if page == nil {
		return nil, nil
	}
	return &releaseevidence.RealtimePeriod{Start: releaseevidence.Date(page.Start), End: releaseevidence.Date(page.End)}, nil
}

func parseOptionalReleasePeriodStrings(start, end string) (*RealtimePeriod, error) {
	if strings.TrimSpace(start) == "" && strings.TrimSpace(end) == "" {
		return nil, nil
	}
	period, err := parseOptionalPeriod(start, end)
	if err != nil {
		return nil, err
	}
	return period, nil
}

func releaseMetadataQuery(request ReleaseCalendarRequest, limit, offset int) url.Values {
	query := url.Values{"file_type": []string{"json"}, "limit": []string{strconv.Itoa(limit)}, "offset": []string{strconv.Itoa(offset)}, "order_by": []string{"release_id"}, "sort_order": []string{"asc"}}
	if request.RealtimeStart != nil {
		query.Set("realtime_start", string(*request.RealtimeStart))
	}
	if request.RealtimeEnd != nil {
		query.Set("realtime_end", string(*request.RealtimeEnd))
	}
	return query
}

func releaseDatesQuery(request ReleaseCalendarRequest, limit, offset int) url.Values {
	query := url.Values{"file_type": []string{"json"}, "limit": []string{strconv.Itoa(limit)}, "offset": []string{strconv.Itoa(offset)}, "order_by": []string{"release_date"}, "sort_order": []string{"asc"}, "include_release_dates_with_no_data": []string{"false"}}
	if request.IncludeReleaseDatesWithNoData {
		query.Set("include_release_dates_with_no_data", "true")
	}
	if request.RealtimeStart != nil {
		query.Set("realtime_start", string(*request.RealtimeStart))
	}
	if request.RealtimeEnd != nil {
		query.Set("realtime_end", string(*request.RealtimeEnd))
	}
	return query
}

func releasePagePayloadID(base providercontract.RawPayloadID, endpoint string, page int) providercontract.RawPayloadID {
	return providercontract.RawPayloadID("rpa_" + canonical.DigestBytes([]byte(string(base) + "\x00" + endpoint + "\x00" + strconv.Itoa(page))).Value[:24])
}

func fredCalendarRawRepresentation() providercontract.RawRepresentation {
	return providercontract.RawRepresentation{Boundary: providercontract.RawBoundaryProvider, Format: providercontract.RawFormatJSONDocument, Schema: canonical.VersionIdentity{Namespace: "fred.release.api.json", Value: "v1"}, MediaType: "application/json"}
}

func validateReleaseCalendarRequest(request ReleaseCalendarRequest) error {
	if strings.TrimSpace(string(request.MetadataPayloadID)) == "" || strings.TrimSpace(string(request.DatesPayloadID)) == "" {
		return errors.New("FRED release metadata and dates payload IDs are required")
	}
	if request.PageSize < 0 || request.PageSize > 1000 {
		return errors.New("FRED release page size is outside the documented bound")
	}
	if request.MaxPages < 0 {
		return errors.New("FRED release maximum page count must not be negative")
	}
	if request.RealtimeStart != nil {
		if _, err := parseDate(string(*request.RealtimeStart)); err != nil {
			return err
		}
	}
	if request.RealtimeEnd != nil {
		if _, err := parseDate(string(*request.RealtimeEnd)); err != nil {
			return err
		}
	}
	if request.RealtimeStart != nil && request.RealtimeEnd != nil && *request.RealtimeEnd < *request.RealtimeStart {
		return errors.New("FRED release realtime end precedes realtime start")
	}
	return request.Retention.Validate()
}
