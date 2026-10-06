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

// StatusRequest contains all dynamic inputs required to build a status query.
type StatusRequest struct {
	Service   string
	Condition StatusCondition
	TimeRange TimeRange
}

// BuildStatusQuery builds an Elasticsearch query from service, condition, timeline, and exclusions.
func BuildStatusQuery(index config.IndexConfig, req StatusRequest, excludedStatusCodes []int) ([]byte, error) {
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

	boolQuery := map[string]any{
		"filter": filters,
	}

	if len(excludedStatusCodes) > 0 {
		boolQuery["must_not"] = []any{
			map[string]any{
				"terms": map[string]any{
					index.Schema.StatusField: excludedStatusCodes,
				},
			},
		}
	}

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

	body, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("analyze: marshal query: %w", err)
	}

	return body, nil
}
