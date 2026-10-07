package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLoad verifies that a valid YAML configuration is loaded correctly.
func TestLoad(t *testing.T) {
	path := writeTestConfig(t, `
elasticsearch:
  indexes:
    - name: arvan-cdn
      pattern: "arvan-cdn*"
      schema:
        service_field: "domain"
        status_field: "status"
        time_field: "timestamp"

    - name: prod-nginx
      pattern: "prod-nginx*"
      schema:
        service_field: "host"
        status_field: "status"
        time_field: "@timestamp"

analyze:
  stats_index: "arvan-cdn"
  default_window: "5m"
  default_condition:
    gte: 400
    lt: 600
  excluded_status_codes:
    - 404
    - 429
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if len(cfg.Elasticsearch.Indexes) != 2 {
		t.Fatalf("expected 2 indexes, got %d", len(cfg.Elasticsearch.Indexes))
	}

	if cfg.Analyze.StatsIndex != "arvan-cdn" {
		t.Fatalf("expected stats index arvan-cdn, got %q", cfg.Analyze.StatsIndex)
	}

	if cfg.Analyze.DefaultWindow != "5m" {
		t.Fatalf("expected default window 5m, got %q", cfg.Analyze.DefaultWindow)
	}

	if cfg.Analyze.DefaultCondition.GTE != 400 {
		t.Fatalf("expected default gte 400, got %d", cfg.Analyze.DefaultCondition.GTE)
	}

	if cfg.Analyze.DefaultCondition.LT != 600 {
		t.Fatalf("expected default lt 600, got %d", cfg.Analyze.DefaultCondition.LT)
	}

	if len(cfg.Analyze.ExcludedStatusCodes) != 2 {
		t.Fatalf("expected 2 excluded status codes, got %d", len(cfg.Analyze.ExcludedStatusCodes))
	}
}

// TestFindIndex verifies that an index can be resolved by its logical name.
func TestFindIndex(t *testing.T) {
	cfg := validTestConfig()

	index, ok := cfg.FindIndex("arvan-cdn")
	if !ok {
		t.Fatal("expected arvan-cdn index to exist")
	}

	if index.Pattern != "arvan-cdn*" {
		t.Fatalf("expected pattern arvan-cdn*, got %q", index.Pattern)
	}

	if index.Schema.ServiceField != "domain" {
		t.Fatalf("expected service field domain, got %q", index.Schema.ServiceField)
	}
}

// TestFindIndexUnknown verifies that an unknown index name is not resolved.
func TestFindIndexUnknown(t *testing.T) {
	cfg := validTestConfig()

	if _, ok := cfg.FindIndex("unknown"); ok {
		t.Fatal("expected unknown index lookup to fail")
	}
}

// TestValidateRejectsDuplicateNames verifies that duplicate logical index names are rejected.
func TestValidateRejectsDuplicateNames(t *testing.T) {
	cfg := validTestConfig()

	cfg.Elasticsearch.Indexes = append(cfg.Elasticsearch.Indexes, IndexConfig{
		Name:    "arvan-cdn",
		Pattern: "another-pattern*",
		Schema: IndexSchema{
			ServiceField: "domain",
			StatusField:  "status",
			TimeField:    "timestamp",
		},
	})

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected duplicate index name validation to fail")
	}
}

// TestValidateRejectsIncompleteSchema verifies that incomplete index schemas are rejected.
func TestValidateRejectsIncompleteSchema(t *testing.T) {
	cfg := validTestConfig()
	cfg.Elasticsearch.Indexes[0].Schema.TimeField = ""

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected incomplete schema validation to fail")
	}
}

// TestValidateRejectsUnknownStatsIndex verifies that stats_index must reference a configured index.
func TestValidateRejectsUnknownStatsIndex(t *testing.T) {
	cfg := validTestConfig()
	cfg.Analyze.StatsIndex = "unknown"

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected unknown stats index validation to fail")
	}
}

// TestValidateRejectsInvalidWindow verifies that an invalid analysis window is rejected.
func TestValidateRejectsInvalidWindow(t *testing.T) {
	cfg := validTestConfig()
	cfg.Analyze.DefaultWindow = "wrong"

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected invalid analysis window validation to fail")
	}
}

// validTestConfig returns a valid base configuration for unit tests.
func validTestConfig() Config {
	return Config{
		Elasticsearch: ElasticsearchConfig{
			Indexes: []IndexConfig{
				{
					Name:    "arvan-cdn",
					Pattern: "arvan-cdn*",
					Schema: IndexSchema{
						ServiceField: "domain",
						StatusField:  "status",
						TimeField:    "timestamp",
					},
				},
			},
		},
		Analyze: AnalyzeConfig{
			StatsIndex:    "arvan-cdn",
			DefaultWindow: "5m",
			DefaultCondition: StatusCondition{
				GTE: 400,
				LT:  600,
			},
			ExcludedStatusCodes: []int{404, 429},
		},
	}
}

// writeTestConfig creates a temporary YAML configuration file for tests.
func writeTestConfig(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.yaml")

	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("write temporary config: %v", err)
	}

	return path
}

// TestLoadServicesFromEnvironment verifies STATS_SERVICES parsing and deduplication.
func TestLoadServicesFromEnvironment(t *testing.T) {
	t.Setenv("STATS_SERVICES", "api.example.com, panel.example.com,api.example.com,,cdn.example.com")

	path := writeTestConfig(t, `
elasticsearch:
  indexes:
    - name: arvan-cdn
      pattern: "arvan-cdn*"
      schema:
        service_field: "domain"
        status_field: "status"
        time_field: "timestamp"

analyze:
  stats_index: "arvan-cdn"
  default_window: "5m"
  default_condition:
    gte: 400
    lt: 600
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	want := []string{"api.example.com", "panel.example.com", "cdn.example.com"}

	if len(cfg.Analyze.Services) != len(want) {
		t.Fatalf("expected %d services, got %d", len(want), len(cfg.Analyze.Services))
	}

	for i := range want {
		if cfg.Analyze.Services[i] != want[i] {
			t.Fatalf("expected service %q at index %d, got %q", want[i], i, cfg.Analyze.Services[i])
		}
	}
}
