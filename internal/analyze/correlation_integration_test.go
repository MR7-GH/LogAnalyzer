package analyze

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"LogAnalyzer/internal/config"
	"LogAnalyzer/internal/elastic"
)

// correlationIntegrationConfig contains configuration required by the correlation integration test.
type correlationIntegrationConfig struct {
	ElasticURL      string
	ElasticUsername string
	ElasticPassword string
	ConfigPath      string
	Service         string
}

// TestIntegrationCorrelation verifies matched and unmatched correlation against real Elasticsearch indexes.
func TestIntegrationCorrelation(t *testing.T) {
	testCfg := loadCorrelationIntegrationConfig(t)

	appCfg, err := config.Load(testCfg.ConfigPath)
	if err != nil {
		t.Fatalf("load application config %q: %v", testCfg.ConfigPath, err)
	}

	t.Logf("config path: %s", testCfg.ConfigPath)
	t.Logf("correlation enabled: %v", appCfg.Analyze.Correlation.Enabled)
	t.Logf("primary index: %s", appCfg.Analyze.Correlation.PrimaryIndex)
	t.Logf("secondary index: %s", appCfg.Analyze.Correlation.SecondaryIndex)
	t.Logf("primary field: %s", appCfg.Analyze.Correlation.PrimaryField)
	t.Logf("secondary field: %s", appCfg.Analyze.Correlation.SecondaryField)

	if !appCfg.Analyze.Correlation.Enabled {
		t.Fatalf("correlation is disabled in config %q", testCfg.ConfigPath)
	}

	client, err := elastic.NewClient(elastic.Config{
		Addresses: []string{testCfg.ElasticURL},
		Username:  testCfg.ElasticUsername,
		Password:  testCfg.ElasticPassword,
		Timeout:   15 * time.Second,
	})
	if err != nil {
		t.Fatalf("create Elasticsearch client: %v", err)
	}

	service, err := NewService(client, appCfg, appCfg.Analyze.StatsIndex)
	if err != nil {
		t.Fatalf("create analyze service: %v", err)
	}

	window, err := appCfg.Analyze.Window()
	if err != nil {
		t.Fatalf("load analyze window: %v", err)
	}

	to := time.Now().UTC()
	from := to.Add(-window)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := service.Correlate(ctx, StatusRequest{
		Service: testCfg.Service,
		Condition: StatusCondition{
			GTE: appCfg.Analyze.DefaultCondition.GTE,
			LT:  appCfg.Analyze.DefaultCondition.LT,
		},
		TimeRange: TimeRange{
			From: from,
			To:   to,
		},
	})
	if err != nil {
		t.Fatalf("execute correlation: %v", err)
	}

	t.Logf("service=%s", testCfg.Service)
	t.Logf("from=%s", from.Format(time.RFC3339))
	t.Logf("to=%s", to.Format(time.RFC3339))
	t.Logf("documents=%d", result.Documents)
	t.Logf("correlatable=%d", result.Correlatable)
	t.Logf("matched=%d", result.Matched)
	t.Logf("unmatched=%d", result.Unmatched)
	t.Logf("missing_request_id=%d", result.MissingRequestID)
	t.Logf("matched_percent=%.2f", result.MatchedPercent)
	t.Logf("unmatched_percent=%.2f", result.UnmatchedPercent)

	if result.Matched+result.Unmatched != result.Correlatable {
		t.Fatalf(
			"invalid correlation result: matched=%d unmatched=%d correlatable=%d",
			result.Matched,
			result.Unmatched,
			result.Correlatable,
		)
	}

	if result.Correlatable+result.MissingRequestID != result.Documents {
		t.Fatalf(
			"invalid document accounting: correlatable=%d missing_request_id=%d documents=%d",
			result.Correlatable,
			result.MissingRequestID,
			result.Documents,
		)
	}

	for status, statusResult := range result.ByStatus {
		t.Logf(
			"status=%d documents=%d correlatable=%d matched=%d unmatched=%d missing_request_id=%d matched_percent=%.2f unmatched_percent=%.2f unmatched_ids=%d",
			status,
			statusResult.Documents,
			statusResult.Correlatable,
			statusResult.Matched,
			statusResult.Unmatched,
			statusResult.MissingRequestID,
			statusResult.MatchedPercent,
			statusResult.UnmatchedPercent,
			len(statusResult.UnmatchedRequestIDs),
		)

		if statusResult.Matched+statusResult.Unmatched != statusResult.Correlatable {
			t.Fatalf(
				"invalid status correlation result for status=%d: matched=%d unmatched=%d correlatable=%d",
				status,
				statusResult.Matched,
				statusResult.Unmatched,
				statusResult.Correlatable,
			)
		}

		if statusResult.Correlatable+statusResult.MissingRequestID != statusResult.Documents {
			t.Fatalf(
				"invalid status document accounting for status=%d: correlatable=%d missing_request_id=%d documents=%d",
				status,
				statusResult.Correlatable,
				statusResult.MissingRequestID,
				statusResult.Documents,
			)
		}
	}
}

// loadCorrelationIntegrationConfig loads environment and config-path settings for the correlation integration test.
func loadCorrelationIntegrationConfig(t *testing.T) correlationIntegrationConfig {
	t.Helper()

	cfg := correlationIntegrationConfig{
		ElasticURL:      strings.TrimSpace(os.Getenv("ELASTIC_URL")),
		ElasticUsername: strings.TrimSpace(os.Getenv("ELASTIC_USERNAME")),
		ElasticPassword: strings.TrimSpace(os.Getenv("ELASTIC_PASSWORD")),
		Service:         strings.TrimSpace(os.Getenv("ANALYZE_TEST_SERVICE")),
		ConfigPath:      resolveCorrelationConfigPath(t),
	}

	if cfg.ElasticURL == "" {
		t.Fatal("ELASTIC_URL is required")
	}

	if cfg.ElasticUsername == "" {
		t.Fatal("ELASTIC_USERNAME is required")
	}

	if cfg.ElasticPassword == "" {
		t.Fatal("ELASTIC_PASSWORD is required")
	}

	if cfg.Service == "" {
		t.Fatal("ANALYZE_TEST_SERVICE is required")
	}

	return cfg
}

// resolveCorrelationConfigPath uses CONFIG_PATH when set or falls back to the project configs/config.yaml file.
func resolveCorrelationConfigPath(t *testing.T) string {
	t.Helper()

	if value := strings.TrimSpace(os.Getenv("CONFIG_PATH")); value != "" {
		path, err := filepath.Abs(value)
		if err != nil {
			t.Fatalf("resolve CONFIG_PATH %q: %v", value, err)
		}

		if _, err := os.Stat(path); err != nil {
			t.Fatalf("CONFIG_PATH does not exist %q: %v", path, err)
		}

		return path
	}

	candidates := []string{
		"../../configs/config.yaml",
		"configs/config.yaml",
		"../configs/config.yaml",
	}

	for _, candidate := range candidates {
		path, err := filepath.Abs(candidate)
		if err != nil {
			continue
		}

		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	t.Fatal("could not find configs/config.yaml and CONFIG_PATH is empty")
	return ""
}
