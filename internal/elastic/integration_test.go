package elastic

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
)

// integrationConfig contains the environment configuration required by Elasticsearch integration tests.
type integrationConfig struct {
	URL      string
	Username string
	Password string
	Indexes  []string
}

// loadIntegrationConfig loads Elasticsearch integration-test configuration from environment variables.
func loadIntegrationConfig(t *testing.T) integrationConfig {
	t.Helper()

	cfg := integrationConfig{
		URL:      strings.TrimSpace(os.Getenv("ELASTIC_URL")),
		Username: strings.TrimSpace(os.Getenv("ELASTIC_USERNAME")),
		Password: strings.TrimSpace(os.Getenv("ELASTIC_PASSWORD")),
		Indexes:  parseIndexes(os.Getenv("ELASTIC_INDEXES")),
	}

	if cfg.URL == "" || cfg.Username == "" || cfg.Password == "" || len(cfg.Indexes) == 0 {
		t.Skip("Elasticsearch integration-test environment variables are not configured")
	}

	return cfg
}

// parseIndexes converts the comma-separated Elasticsearch index list into a clean slice.
func parseIndexes(value string) []string {
	parts := strings.Split(value, ",")
	indexes := make([]string, 0, len(parts))

	for _, part := range parts {
		index := strings.TrimSpace(part)
		if index != "" {
			indexes = append(indexes, index)
		}
	}

	return indexes
}

// newIntegrationSource creates an Elasticsearch source using the integration-test environment.
func newIntegrationSource(t *testing.T) *Source {
	t.Helper()

	cfg := loadIntegrationConfig(t)

	source, err := NewSource("integration-elastic", Config{
		Addresses: []string{cfg.URL},
		Username:  cfg.Username,
		Password:  cfg.Password,
		Timeout:   10 * time.Second,
	})
	if err != nil {
		t.Fatalf("create Elasticsearch source: %v", err)
	}

	return source
}

// TestIntegrationConnection verifies connectivity and authentication using an allowed index query.
func TestIntegrationConnection(t *testing.T) {
	cfg := loadIntegrationConfig(t)
	source := newIntegrationSource(t)

	query := json.RawMessage(`{
		"size": 0,
		"query": {
			"match_all": {}
		}
	}`)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := source.Search(ctx, []string{cfg.Indexes[0]}, query)
	if err != nil {
		t.Fatalf("check Elasticsearch connection using index %q: %v", cfg.Indexes[0], err)
	}

	t.Logf("connected successfully using index=%s total=%d took=%dms", cfg.Indexes[0], result.Total, result.Took)
}

// TestIntegrationConfiguredIndexes verifies that every configured Elasticsearch index can be queried.
func TestIntegrationConfiguredIndexes(t *testing.T) {
	cfg := loadIntegrationConfig(t)
	source := newIntegrationSource(t)

	query := json.RawMessage(`{
		"size": 1,
		"query": {
			"match_all": {}
		}
	}`)

	for _, index := range cfg.Indexes {
		t.Run(index, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			result, err := source.Search(ctx, []string{index}, query)
			if err != nil {
				t.Fatalf("query index %q: %v", index, err)
			}

			t.Logf("index=%s total=%d returned=%d took=%dms", index, result.Total, len(result.Documents), result.Took)
		})
	}
}

// TestIntegrationAllConfiguredIndexes verifies that all configured indexes can be queried in one search request.
func TestIntegrationAllConfiguredIndexes(t *testing.T) {
	cfg := loadIntegrationConfig(t)
	source := newIntegrationSource(t)

	query := json.RawMessage(`{
		"size": 10,
		"query": {
			"match_all": {}
		}
	}`)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := source.Search(ctx, cfg.Indexes, query)
	if err != nil {
		t.Fatalf("query configured indexes: %v", err)
	}

	t.Logf("indexes=%v total=%d returned=%d took=%dms", cfg.Indexes, result.Total, len(result.Documents), result.Took)

	for _, document := range result.Documents {
		t.Logf("document index=%s id=%s", document.Index, document.ID)
	}
}
