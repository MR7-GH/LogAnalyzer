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
	Stats(ctx context.Context, requestedBy string, options analyze.StatsOptions) (analyze.StatsResponse, error)
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
func (s *Service) Submit(requestedBy string, options analyze.StatsOptions) error {
	requestedBy = strings.TrimSpace(requestedBy)

	if requestedBy == "" {
		return fmt.Errorf("report: requestedBy is required")
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
		defer cancel()

		if _, err := s.Process(ctx, requestedBy, options); err != nil {
			log.Printf("report: asynchronous processing requested by %q failed: %v", requestedBy, err)
		}
	}()

	return nil
}

// Process executes analysis synchronously, sends Telegram output, and returns the result.
func (s *Service) Process(ctx context.Context, requestedBy string, options analyze.StatsOptions) (analyze.StatsResponse, error) {
	requestedBy = strings.TrimSpace(requestedBy)

	if requestedBy == "" {
		return analyze.StatsResponse{}, fmt.Errorf("report: requestedBy is required")
	}

	result, err := s.analyzer.Stats(ctx, requestedBy, options)
	if err != nil {
		return analyze.StatsResponse{}, fmt.Errorf("report: analyze request from %q: %w", requestedBy, err)
	}

	message := telegram.FormatStats(result, options.Detailed)

	if err := s.telegram.Send(ctx, message); err != nil {
		return analyze.StatsResponse{}, fmt.Errorf("report: Telegram delivery for request from %q: %w", requestedBy, err)
	}

	log.Printf("report: request from %q successfully analyzed and sent to Telegram", requestedBy)

	return result, nil
}
