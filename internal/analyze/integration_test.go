package analyze

import (
	"context"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"LogAnalyzer/internal/config"
	"LogAnalyzer/internal/elastic"
)

// integrationConfig contains runtime configuration required by analyze integration tests.
type integrationConfig struct {
	ElasticURL      string
	ElasticUsername string
	ElasticPassword string
	ConfigPath      string
	Service         string
}

// loadIntegrationConfig loads analyze integration-test settings.
func loadIntegrationConfig(t *testing.T) integrationConfig {
	t.Helper()

	cfg := integrationConfig{
		ElasticURL:      strings.TrimSpace(os.Getenv("ELASTIC_URL")),
		ElasticUsername: strings.TrimSpace(os.Getenv("ELASTIC_USERNAME")),
		ElasticPassword: strings.TrimSpace(os.Getenv("ELASTIC_PASSWORD")),
		ConfigPath:      envOrDefault("CONFIG_PATH", "../../configs/config.yaml"),
		Service:         strings.TrimSpace(os.Getenv("ANALYZE_TEST_SERVICE")),
	}

	if cfg.ElasticURL == "" || cfg.ElasticUsername == "" || cfg.ElasticPassword == "" || cfg.Service == "" {
		t.Skip("analyze integration-test environment variables are not configured")
	}

	return cfg
}

// envOrDefault returns an environment variable or fallback value when empty.
func envOrDefault(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	return value
}

// TestIntegrationStatusAnalysis verifies dynamic status analysis against real Elasticsearch data.
func TestIntegrationStatusAnalysis(t *testing.T) {
	testCfg := loadIntegrationConfig(t)

	appCfg, err := config.Load(testCfg.ConfigPath)
	if err != nil {
		t.Fatalf("load application config: %v", err)
	}

	client, err := elastic.NewClient(elastic.Config{
		Addresses: []string{testCfg.ElasticURL},
		Username:  testCfg.ElasticUsername,
		Password:  testCfg.ElasticPassword,
		Timeout:   10 * time.Second,
	})
	if err != nil {
		t.Fatalf("create Elasticsearch client: %v", err)
	}

	service, err := NewService(client, appCfg, appCfg.Analyze.StatsIndex)
	if err != nil {
		t.Fatalf("create analyze service: %v", err)
	}

	to := time.Now().UTC()
	from := to.Add(-5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	result, err := service.Status(ctx, StatusRequest{
		Service: testCfg.Service,
		Condition: StatusCondition{
			GTE: 400,
			LT:  600,
		},
		TimeRange: TimeRange{
			From: from,
			To:   to,
		},
	})
	if err != nil {
		t.Fatalf("execute status analysis: %v", err)
	}

	t.Logf("index=%s service=%s", result.Index, result.Service)
	t.Logf("from=%s to=%s", result.From.Format(time.RFC3339), result.To.Format(time.RFC3339))
	t.Logf("total=%d 4xx=%d 5xx=%d", result.Total, result.Total4xx, result.Total5xx)

	codes := make([]int, 0, len(result.Codes))
	for code := range result.Codes {
		codes = append(codes, code)
	}

	sort.Ints(codes)

	for _, code := range codes {
		t.Logf("status=%d count=%d", code, result.Codes[code])
	}
}
