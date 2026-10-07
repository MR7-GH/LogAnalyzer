package analyze

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"LogAnalyzer/internal/config"
	"LogAnalyzer/internal/elastic"
)

// Service executes log analysis operations.
type Service struct {
	client    *elastic.Client
	config    *config.Config
	indexName string
}

// StatsResponse contains ordered statistics for the requested service and configured services with errors.
type StatsResponse struct {
	Services []StatusResult `json:"services"`
}

// StatusResult contains HTTP status statistics for one service and time window.
type StatusResult struct {
	Service  string        `json:"service"`
	Index    string        `json:"index"`
	From     time.Time     `json:"from"`
	To       time.Time     `json:"to"`
	Total    int64         `json:"total"`
	Codes    map[int]int64 `json:"codes"`
	Total4xx int64         `json:"total_4xx"`
	Total5xx int64         `json:"total_5xx"`
}

// statusResponse contains the Elasticsearch fields required by status analysis.
type statusResponse struct {
	Hits struct {
		Total struct {
			Value int64 `json:"value"`
		} `json:"total"`
	} `json:"hits"`

	Aggregations struct {
		StatusCodes struct {
			Buckets []struct {
				Key      int   `json:"key"`
				DocCount int64 `json:"doc_count"`
			} `json:"buckets"`
		} `json:"status_codes"`
	} `json:"aggregations"`
}

// NewService creates a new analysis service.
func NewService(client *elastic.Client, cfg *config.Config, indexName string) (*Service, error) {
	if client == nil {
		return nil, fmt.Errorf("analyze: Elasticsearch client is required")
	}

	if cfg == nil {
		return nil, fmt.Errorf("analyze: config is required")
	}

	if _, ok := cfg.FindIndex(indexName); !ok {
		return nil, fmt.Errorf("analyze: index %q is not configured", indexName)
	}

	return &Service{client: client, config: cfg, indexName: indexName}, nil
}

// Stats returns the requested service first and only configured services that have errors.
func (s *Service) Stats(ctx context.Context, requestedService string) (StatsResponse, error) {
	services := mergeServices(requestedService, s.config.Analyze.Services)

	window, err := s.config.Analyze.Window()
	if err != nil {
		return StatsResponse{}, err
	}

	to := time.Now().UTC()
	from := to.Add(-window)

	results := make([]StatusResult, 0, len(services))

	for i, service := range services {
		result, err := s.Status(ctx, StatusRequest{
			Service: service,
			Condition: StatusCondition{
				GTE: s.config.Analyze.DefaultCondition.GTE,
				LT:  s.config.Analyze.DefaultCondition.LT,
			},
			TimeRange: TimeRange{
				From: from,
				To:   to,
			},
		})
		if err != nil {
			return StatsResponse{}, fmt.Errorf("analyze service %q: %w", service, err)
		}

		if i != 0 && result.Total == 0 {
			continue
		}

		results = append(results, result)
	}

	return StatsResponse{Services: results}, nil
}

// Status executes a fully dynamic status analysis request.
func (s *Service) Status(ctx context.Context, req StatusRequest) (StatusResult, error) {
	index, ok := s.config.FindIndex(s.indexName)
	if !ok {
		return StatusResult{}, fmt.Errorf("analyze: index %q is not configured", s.indexName)
	}

	body, err := BuildStatusQuery(index, req, s.config.Analyze.ExcludedStatusCodes)
	if err != nil {
		return StatusResult{}, err
	}

	response, err := s.client.SearchRaw(ctx, []string{index.Pattern}, body)
	if err != nil {
		return StatusResult{}, err
	}

	var parsed statusResponse
	if err := json.Unmarshal(response, &parsed); err != nil {
		return StatusResult{}, fmt.Errorf("analyze: decode Elasticsearch response: %w", err)
	}

	result := StatusResult{
		Service: req.Service,
		Index:   index.Name,
		From:    req.TimeRange.From,
		To:      req.TimeRange.To,
		Total:   parsed.Hits.Total.Value,
		Codes:   make(map[int]int64, len(parsed.Aggregations.StatusCodes.Buckets)),
	}

	for _, bucket := range parsed.Aggregations.StatusCodes.Buckets {
		result.Codes[bucket.Key] = bucket.DocCount

		if bucket.Key >= 400 && bucket.Key < 500 {
			result.Total4xx += bucket.DocCount
		}

		if bucket.Key >= 500 && bucket.Key < 600 {
			result.Total5xx += bucket.DocCount
		}
	}

	return result, nil
}

// mergeServices places the requested service first and appends configured services without duplicates.
func mergeServices(requested string, configured []string) []string {
	requested = strings.TrimSpace(requested)

	services := make([]string, 0, len(configured)+1)
	seen := make(map[string]struct{}, len(configured)+1)

	if requested != "" {
		services = append(services, requested)
		seen[requested] = struct{}{}
	}

	for _, service := range configured {
		service = strings.TrimSpace(service)
		if service == "" {
			continue
		}

		if _, exists := seen[service]; exists {
			continue
		}

		seen[service] = struct{}{}
		services = append(services, service)
	}

	return services
}
