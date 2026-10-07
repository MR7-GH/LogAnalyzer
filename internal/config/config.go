package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config contains the application configuration.
type Config struct {
	Elasticsearch ElasticsearchConfig `yaml:"elasticsearch"`
	Analyze       AnalyzeConfig       `yaml:"analyze"`
}

// AnalyzeConfig contains analysis defaults, policies, and runtime services.
type AnalyzeConfig struct {
	StatsIndex          string          `yaml:"stats_index"`
	DefaultWindow       string          `yaml:"default_window"`
	DefaultCondition    StatusCondition `yaml:"default_condition"`
	ExcludedStatusCodes []int           `yaml:"excluded_status_codes"`
	Services            []string        `yaml:"-"`
}

// StatusCondition defines the default HTTP status range.
type StatusCondition struct {
	GTE int `yaml:"gte"`
	LT  int `yaml:"lt"`
}

// ElasticsearchConfig contains Elasticsearch index definitions.
type ElasticsearchConfig struct {
	Indexes []IndexConfig `yaml:"indexes"`
}

// IndexConfig describes one logical Elasticsearch index or index pattern.
type IndexConfig struct {
	Name    string      `yaml:"name"`
	Pattern string      `yaml:"pattern"`
	Schema  IndexSchema `yaml:"schema"`
}

// IndexSchema describes the fields required for analysis on an index.
type IndexSchema struct {
	ServiceField string `yaml:"service_field"`
	StatusField  string `yaml:"status_field"`
	TimeField    string `yaml:"time_field"`
}

// Load reads YAML configuration, applies environment configuration, and validates the result.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config: read %q: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("config: decode %q: %w", path, err)
	}

	cfg.Analyze.Services = parseServices(os.Getenv("STATS_SERVICES"))

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Validate verifies the application configuration.
func (c Config) Validate() error {
	if len(c.Elasticsearch.Indexes) == 0 {
		return fmt.Errorf("config: at least one Elasticsearch index is required")
	}

	names := make(map[string]struct{}, len(c.Elasticsearch.Indexes))

	for i, index := range c.Elasticsearch.Indexes {
		if err := index.Validate(); err != nil {
			return fmt.Errorf("config: elasticsearch index %d: %w", i, err)
		}

		if _, exists := names[index.Name]; exists {
			return fmt.Errorf("config: duplicate Elasticsearch index name %q", index.Name)
		}

		names[index.Name] = struct{}{}
	}

	if strings.TrimSpace(c.Analyze.StatsIndex) == "" {
		return fmt.Errorf("config: analyze.stats_index is required")
	}

	if _, ok := c.FindIndex(c.Analyze.StatsIndex); !ok {
		return fmt.Errorf("config: analyze.stats_index %q is not configured", c.Analyze.StatsIndex)
	}

	if _, err := c.Analyze.Window(); err != nil {
		return err
	}

	return nil
}

// Validate verifies that an index definition contains the required fields.
func (i IndexConfig) Validate() error {
	if strings.TrimSpace(i.Name) == "" {
		return fmt.Errorf("name is required")
	}

	if strings.TrimSpace(i.Pattern) == "" {
		return fmt.Errorf("pattern is required")
	}

	if strings.TrimSpace(i.Schema.ServiceField) == "" {
		return fmt.Errorf("schema.service_field is required")
	}

	if strings.TrimSpace(i.Schema.StatusField) == "" {
		return fmt.Errorf("schema.status_field is required")
	}

	if strings.TrimSpace(i.Schema.TimeField) == "" {
		return fmt.Errorf("schema.time_field is required")
	}

	return nil
}

// FindIndex returns an Elasticsearch index definition by logical name.
func (c Config) FindIndex(name string) (IndexConfig, bool) {
	for _, index := range c.Elasticsearch.Indexes {
		if index.Name == name {
			return index, true
		}
	}

	return IndexConfig{}, false
}

// Window parses the configured default analysis window.
func (c AnalyzeConfig) Window() (time.Duration, error) {
	window, err := time.ParseDuration(c.DefaultWindow)
	if err != nil {
		return 0, fmt.Errorf("config: invalid analyze.default_window %q: %w", c.DefaultWindow, err)
	}

	if window <= 0 {
		return 0, fmt.Errorf("config: analyze.default_window must be greater than zero")
	}

	return window, nil
}

// parseServices parses, trims, and deduplicates a comma-separated service list.
func parseServices(value string) []string {
	parts := strings.Split(value, ",")
	services := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))

	for _, part := range parts {
		service := strings.TrimSpace(part)
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
