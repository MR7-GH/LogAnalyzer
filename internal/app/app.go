package app

import (
	"fmt"
	"os"
	"strings"
	"time"

	"LogAnalyzer/internal/analyze"
	"LogAnalyzer/internal/api/handler"
	"LogAnalyzer/internal/api/routes"
	"LogAnalyzer/internal/config"
	"LogAnalyzer/internal/elastic"
	"LogAnalyzer/internal/notify/telegram"
	"LogAnalyzer/internal/report"
)

// Run initializes and starts the LogAnalyzer application.
func Run() error {
	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		return err
	}

	elasticURL := strings.TrimSpace(os.Getenv("ELASTIC_URL"))
	if elasticURL == "" {
		return fmt.Errorf("app: ELASTIC_URL is required")
	}

	elasticClient, err := elastic.NewClient(elastic.Config{
		Addresses: []string{elasticURL},
		Username:  strings.TrimSpace(os.Getenv("ELASTIC_USERNAME")),
		Password:  strings.TrimSpace(os.Getenv("ELASTIC_PASSWORD")),
		Timeout:   10 * time.Second,
	})
	if err != nil {
		return err
	}

	analyzeService, err := analyze.NewService(elasticClient, cfg, cfg.Analyze.StatsIndex)
	if err != nil {
		return err
	}

	telegramClient, err := telegram.NewClientFromEnv(10 * time.Second)
	if err != nil {
		return err
	}

	reportService, err := report.NewService(analyzeService, telegramClient, 60*time.Second)
	if err != nil {
		return err
	}

	statsHandler := handler.NewStats(reportService)
	router := routes.New(statsHandler)

	address := envOrDefault("HTTP_ADDRESS", ":8080")

	fmt.Printf("LogAnalyzer listening on %s\n", address)

	return router.Run(address)
}

// envOrDefault returns an environment variable or fallback value when empty.
func envOrDefault(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	return value
}
