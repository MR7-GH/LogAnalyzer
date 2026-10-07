package report

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"LogAnalyzer/internal/analyze"
	"LogAnalyzer/internal/notify/telegram"
)

// Analyzer defines the analysis behavior required by the report service.
type Analyzer interface {
	Stats(ctx context.Context, service string, options analyze.StatsOptions) (analyze.StatsResponse, error)
}

// TelegramSender defines the Telegram behavior required by the report service.
type TelegramSender interface {
	Send(ctx context.Context, text string) error
}

// Service executes analysis jobs and sends their results to Telegram.
type Service struct {
	analyzer Analyzer
	telegram TelegramSender
	timeout  time.Duration
}

// NewService creates a report processing service.
func NewService(analyzer Analyzer, telegramSender TelegramSender, timeout time.Duration) (*Service, error) {
	if analyzer == nil {
		return nil, fmt.Errorf("report: analyzer is required")
	}

	if telegramSender == nil {
		return nil, fmt.Errorf("report: Telegram sender is required")
	}

	if timeout <= 0 {
		timeout = 60 * time.Second
	}

	return &Service{
		analyzer: analyzer,
		telegram: telegramSender,
		timeout:  timeout,
	}, nil
}

// Submit accepts a report request and starts processing it asynchronously.
func (s *Service) Submit(service string, options analyze.StatsOptions) error {
	service = strings.TrimSpace(service)
	if service == "" {
		return fmt.Errorf("report: service is required")
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
		defer cancel()

		if _, err := s.Process(ctx, service, options); err != nil {
			log.Printf("report: asynchronous processing for service %q failed: %v", service, err)
		}
	}()

	return nil
}

// Process executes analysis synchronously, sends Telegram output, and returns the result.
func (s *Service) Process(ctx context.Context, service string, options analyze.StatsOptions) (analyze.StatsResponse, error) {
	service = strings.TrimSpace(service)
	if service == "" {
		return analyze.StatsResponse{}, fmt.Errorf("report: service is required")
	}

	result, err := s.analyzer.Stats(ctx, service, options)
	if err != nil {
		return analyze.StatsResponse{}, fmt.Errorf("report: analyze service %q: %w", service, err)
	}

	message := telegram.FormatStats(result, options.Detailed)

	if err := s.telegram.Send(ctx, message); err != nil {
		return analyze.StatsResponse{}, fmt.Errorf("report: Telegram delivery for service %q: %w", service, err)
	}

	log.Printf("report: service %q successfully analyzed and sent to Telegram", service)

	return result, nil
}
