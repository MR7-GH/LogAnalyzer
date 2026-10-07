package analyze

import (
	"encoding/json"
	"fmt"
	"time"

	"LogAnalyzer/internal/config"
)

// StatusCondition defines the HTTP status condition used by a query.
type StatusCondition struct {
	GTE int
	LT  int
}

// TimeRange defines the time window used by a query.
type TimeRange struct {
	From time.Time
	To   time.Time
}

// StatusRequest contains all dynamic inputs required to build an analysis query.
type StatusRequest struct {
	Service   string
	Condition StatusCondition
	TimeRange TimeRange
}

// BuildStatusQuery builds an Elasticsearch status aggregation query.
func BuildStatusQuery(index config.IndexConfig, req StatusRequest, excludedStatusCodes []int) ([]byte, error) {
	filters, err := buildFilters(index, req)
	if err != nil {
		return nil, err
	}

	boolQuery := map[string]any{"filter": filters}
	addExcludedStatusCodes(boolQuery, index.Schema.StatusField, excludedStatusCodes)

	query := map[string]any{
		"size": 0,
		"query": map[string]any{
			"bool": boolQuery,
		},
		"aggs": map[string]any{
			"status_codes": map[string]any{
				"terms": map[string]any{
					"field": index.Schema.StatusField,
					"size":  200,
				},
			},
		},
	}

	return marshalQuery(query)
}

// BuildCorrelationPrimaryQuery builds the query used to collect correlation IDs and status codes from the primary index.
func BuildCorrelationPrimaryQuery(index config.IndexConfig, req StatusRequest, correlationField string, excludedStatusCodes []int, size int) ([]byte, error) {
	filters, err := buildFilters(index, req)
	if err != nil {
		return nil, err
	}

	boolQuery := map[string]any{"filter": filters}
	addExcludedStatusCodes(boolQuery, index.Schema.StatusField, excludedStatusCodes)

	query := map[string]any{
		"size":             size,
		"track_total_hits": true,
		"_source": []string{
			correlationField,
			index.Schema.StatusField,
		},
		"query": map[string]any{
			"bool": boolQuery,
		},
	}

	return marshalQuery(query)
}

// BuildCorrelationSecondaryQuery builds a batched lookup query for correlation IDs.
func BuildCorrelationSecondaryQuery(field string, ids []string) ([]byte, error) {
	if field == "" {
		return nil, fmt.Errorf("analyze: secondary correlation field is required")
	}

	if len(ids) == 0 {
		return nil, fmt.Errorf("analyze: correlation IDs are required")
	}

	query := map[string]any{
		"size":    len(ids),
		"_source": []string{field},
		"query": map[string]any{
			"terms": map[string]any{
				field: ids,
			},
		},
	}

	return marshalQuery(query)
}

// buildFilters builds the service, condition, and timeline filters shared by analysis queries.
func buildFilters(index config.IndexConfig, req StatusRequest) ([]any, error) {
	if req.Service == "" {
		return nil, fmt.Errorf("analyze: service is required")
	}

	if req.TimeRange.From.IsZero() || req.TimeRange.To.IsZero() {
		return nil, fmt.Errorf("analyze: time range is required")
	}

	if !req.TimeRange.From.Before(req.TimeRange.To) {
		return nil, fmt.Errorf("analyze: from must be before to")
	}

	filters := []any{
		map[string]any{
			"term": map[string]any{
				index.Schema.ServiceField: req.Service,
			},
		},
		map[string]any{
			"range": map[string]any{
				index.Schema.TimeField: map[string]any{
					"gte": req.TimeRange.From.UTC().Format(time.RFC3339Nano),
					"lt":  req.TimeRange.To.UTC().Format(time.RFC3339Nano),
				},
			},
		},
	}

	statusRange := map[string]any{}

	if req.Condition.GTE > 0 {
		statusRange["gte"] = req.Condition.GTE
	}

	if req.Condition.LT > 0 {
		statusRange["lt"] = req.Condition.LT
	}

	if len(statusRange) > 0 {
		filters = append(filters, map[string]any{
			"range": map[string]any{
				index.Schema.StatusField: statusRange,
			},
		})
	}

	return filters, nil
}

// addExcludedStatusCodes adds status-code exclusions to an Elasticsearch bool query.
func addExcludedStatusCodes(boolQuery map[string]any, field string, excluded []int) {
	if len(excluded) == 0 {
		return
	}

	boolQuery["must_not"] = []any{
		map[string]any{
			"terms": map[string]any{
				field: excluded,
			},
		},
	}
}

// marshalQuery serializes an Elasticsearch query.
func marshalQuery(query map[string]any) ([]byte, error) {
	body, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("analyze: marshal query: %w", err)
	}

	return body, nil
}
