package analyze

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// CorrelationSummary contains lightweight correlation statistics returned in normal API responses.
type CorrelationSummary struct {
	Total     int `json:"total"`
	Matched   int `json:"matched"`
	Unmatched int `json:"unmatched"`
}

// CorrelationDetail contains detailed correlation statistics returned when detail mode is enabled.
type CorrelationDetail struct {
	Documents        int                              `json:"documents"`
	Correlatable     int                              `json:"correlatable"`
	Matched          int                              `json:"matched"`
	Unmatched        int                              `json:"unmatched"`
	MissingRequestID int                              `json:"missing_request_id"`
	MatchedPercent   float64                          `json:"matched_percent"`
	UnmatchedPercent float64                          `json:"unmatched_percent"`
	ByStatus         map[int]*StatusCorrelationResult `json:"by_status"`
}

// StatusCorrelationResult contains correlation statistics for one HTTP status code.
type StatusCorrelationResult struct {
	Documents           int      `json:"documents"`
	Correlatable        int      `json:"correlatable"`
	Matched             int      `json:"matched"`
	Unmatched           int      `json:"unmatched"`
	MissingRequestID    int      `json:"missing_request_id"`
	MatchedPercent      float64  `json:"matched_percent"`
	UnmatchedPercent    float64  `json:"unmatched_percent"`
	UnmatchedRequestIDs []string `json:"unmatched_request_ids"`
}

// CorrelationResult contains the complete internal correlation analysis result.
type CorrelationResult struct {
	Documents        int
	Correlatable     int
	Matched          int
	Unmatched        int
	MissingRequestID int
	MatchedPercent   float64
	UnmatchedPercent float64
	ByStatus         map[int]*StatusCorrelationResult
}

// correlationRecord contains one primary Elasticsearch document used for correlation.
type correlationRecord struct {
	RequestID string
	Status    int
}

// correlationSearchResponse contains correlation fields returned by Elasticsearch.
type correlationSearchResponse struct {
	Hits struct {
		Total struct {
			Value int `json:"value"`
		} `json:"total"`

		Hits []struct {
			Source json.RawMessage `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
}

// Correlate compares primary index documents against the secondary index.
func (s *Service) Correlate(ctx context.Context, req StatusRequest) (CorrelationResult, error) {
	cfg := s.config.Analyze.Correlation

	if !cfg.Enabled {
		return CorrelationResult{}, nil
	}

	primary, ok := s.config.FindIndex(cfg.PrimaryIndex)
	if !ok {
		return CorrelationResult{}, fmt.Errorf("analyze: primary correlation index %q is not configured", cfg.PrimaryIndex)
	}

	secondary, ok := s.config.FindIndex(cfg.SecondaryIndex)
	if !ok {
		return CorrelationResult{}, fmt.Errorf("analyze: secondary correlation index %q is not configured", cfg.SecondaryIndex)
	}

	body, err := BuildCorrelationPrimaryQuery(primary, req, cfg.PrimaryField, s.config.Analyze.ExcludedStatusCodes, cfg.MaxDocuments)
	if err != nil {
		return CorrelationResult{}, err
	}

	response, err := s.client.SearchRaw(ctx, []string{primary.Pattern}, body)
	if err != nil {
		return CorrelationResult{}, err
	}

	var parsed correlationSearchResponse
	if err := json.Unmarshal(response, &parsed); err != nil {
		return CorrelationResult{}, fmt.Errorf("analyze: decode primary correlation response: %w", err)
	}

	if parsed.Hits.Total.Value > cfg.MaxDocuments {
		return CorrelationResult{}, fmt.Errorf("analyze: correlation result exceeds max_documents=%d", cfg.MaxDocuments)
	}

	records, missingByStatus := extractPrimaryCorrelationRecords(parsed.Hits.Hits, cfg.PrimaryField, primary.Schema.StatusField)

	result := CorrelationResult{
		Documents:        len(parsed.Hits.Hits),
		Correlatable:     len(records),
		MissingRequestID: len(parsed.Hits.Hits) - len(records),
		ByStatus:         make(map[int]*StatusCorrelationResult),
	}

	for status, missing := range missingByStatus {
		statusResult := ensureStatusCorrelationResult(result.ByStatus, status)
		statusResult.Documents += missing
		statusResult.MissingRequestID += missing
	}

	if len(records) == 0 {
		return result, nil
	}

	primaryIDs := uniqueCorrelationIDs(records)
	matchedIDs := make(map[string]struct{}, len(primaryIDs))

	for start := 0; start < len(primaryIDs); start += cfg.BatchSize {
		end := start + cfg.BatchSize
		if end > len(primaryIDs) {
			end = len(primaryIDs)
		}

		query, err := BuildCorrelationSecondaryQuery(cfg.SecondaryField, primaryIDs[start:end])
		if err != nil {
			return CorrelationResult{}, err
		}

		response, err := s.client.SearchRaw(ctx, []string{secondary.Pattern}, query)
		if err != nil {
			return CorrelationResult{}, err
		}

		var secondaryResponse correlationSearchResponse
		if err := json.Unmarshal(response, &secondaryResponse); err != nil {
			return CorrelationResult{}, fmt.Errorf("analyze: decode secondary correlation response: %w", err)
		}

		for _, id := range extractSecondaryCorrelationIDs(secondaryResponse.Hits.Hits, cfg.SecondaryField) {
			matchedIDs[id] = struct{}{}
		}
	}

	unmatchedIDsByStatus := make(map[int]map[string]struct{})

	for _, record := range records {
		statusResult := ensureStatusCorrelationResult(result.ByStatus, record.Status)

		statusResult.Documents++
		statusResult.Correlatable++

		if _, matched := matchedIDs[record.RequestID]; matched {
			statusResult.Matched++
			result.Matched++
			continue
		}

		statusResult.Unmatched++
		result.Unmatched++

		if _, exists := unmatchedIDsByStatus[record.Status]; !exists {
			unmatchedIDsByStatus[record.Status] = make(map[string]struct{})
		}

		unmatchedIDsByStatus[record.Status][record.RequestID] = struct{}{}
	}

	for status, statusResult := range result.ByStatus {
		statusResult.MatchedPercent = percentage(statusResult.Matched, statusResult.Correlatable)
		statusResult.UnmatchedPercent = percentage(statusResult.Unmatched, statusResult.Correlatable)

		ids := unmatchedIDsByStatus[status]
		statusResult.UnmatchedRequestIDs = make([]string, 0, len(ids))

		for id := range ids {
			statusResult.UnmatchedRequestIDs = append(statusResult.UnmatchedRequestIDs, id)
		}

		sort.Strings(statusResult.UnmatchedRequestIDs)
	}

	result.MatchedPercent = percentage(result.Matched, result.Correlatable)
	result.UnmatchedPercent = percentage(result.Unmatched, result.Correlatable)

	return result, nil
}

// extractPrimaryCorrelationRecords extracts status and request ID information from primary hits.
func extractPrimaryCorrelationRecords(hits []struct {
	Source json.RawMessage `json:"_source"`
}, requestIDField, statusField string) ([]correlationRecord, map[int]int) {
	records := make([]correlationRecord, 0, len(hits))
	missingByStatus := make(map[int]int)

	for _, hit := range hits {
		status, ok := readIntField(hit.Source, statusField)
		if !ok {
			continue
		}

		requestID, ok := readStringField(hit.Source, requestIDField)
		if !ok || strings.TrimSpace(requestID) == "" {
			missingByStatus[status]++
			continue
		}

		records = append(records, correlationRecord{
			RequestID: requestID,
			Status:    status,
		})
	}

	return records, missingByStatus
}

// extractSecondaryCorrelationIDs extracts unique correlation IDs from secondary hits.
func extractSecondaryCorrelationIDs(hits []struct {
	Source json.RawMessage `json:"_source"`
}, field string) []string {
	ids := make([]string, 0, len(hits))
	seen := make(map[string]struct{}, len(hits))

	for _, hit := range hits {
		id, ok := readStringField(hit.Source, field)
		if !ok || strings.TrimSpace(id) == "" {
			continue
		}

		if _, exists := seen[id]; exists {
			continue
		}

		seen[id] = struct{}{}
		ids = append(ids, id)
	}

	return ids
}

// uniqueCorrelationIDs returns unique request IDs used for the secondary terms query.
func uniqueCorrelationIDs(records []correlationRecord) []string {
	ids := make([]string, 0, len(records))
	seen := make(map[string]struct{}, len(records))

	for _, record := range records {
		if _, exists := seen[record.RequestID]; exists {
			continue
		}

		seen[record.RequestID] = struct{}{}
		ids = append(ids, record.RequestID)
	}

	return ids
}

// ensureStatusCorrelationResult returns an initialized result for a status code.
func ensureStatusCorrelationResult(results map[int]*StatusCorrelationResult, status int) *StatusCorrelationResult {
	result, exists := results[status]
	if exists {
		return result
	}

	result = &StatusCorrelationResult{
		UnmatchedRequestIDs: []string{},
	}

	results[status] = result
	return result
}

// percentage calculates a percentage while safely handling zero totals.
func percentage(value, total int) float64 {
	if total == 0 {
		return 0
	}

	return float64(value) * 100 / float64(total)
}

// readStringField reads a string value from a JSON source using a dotted field path.
func readStringField(source json.RawMessage, field string) (string, bool) {
	value, ok := readField(source, field)
	if !ok {
		return "", false
	}

	stringValue, ok := value.(string)
	return stringValue, ok
}

// readIntField reads an integer value from a JSON source using a dotted field path.
func readIntField(source json.RawMessage, field string) (int, bool) {
	value, ok := readField(source, field)
	if !ok {
		return 0, false
	}

	switch typed := value.(type) {
	case float64:
		return int(typed), true

	case int:
		return typed, true

	case string:
		var status int
		if _, err := fmt.Sscanf(typed, "%d", &status); err != nil {
			return 0, false
		}

		return status, true

	default:
		return 0, false
	}
}

// readField reads a value from a JSON source using a dotted field path.
func readField(source json.RawMessage, field string) (any, bool) {
	var data map[string]any

	if err := json.Unmarshal(source, &data); err != nil {
		return nil, false
	}

	var current any = data

	for _, part := range strings.Split(field, ".") {
		object, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}

		current, ok = object[part]
		if !ok {
			return nil, false
		}
	}

	return current, true
}
