package telegram

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config contains Telegram Bot API configuration.
type Config struct {
	BotToken string
	ChatID   string
	ThreadID int64
	Timeout  time.Duration
}

// Client sends messages through the Telegram Bot API.
type Client struct {
	botToken   string
	chatID     string
	threadID   int64
	httpClient *http.Client
}

// NewClientFromEnv creates a Telegram client from environment variables.
func NewClientFromEnv(timeout time.Duration) (*Client, error) {
	botToken := strings.TrimSpace(os.Getenv("TELEGRAM_BOT_TOKEN"))
	chatID := strings.TrimSpace(os.Getenv("TELEGRAM_CHAT_ID"))
	threadIDValue := strings.TrimSpace(os.Getenv("TELEGRAM_THREAD_ID"))

	if botToken == "" {
		return nil, fmt.Errorf("telegram: TELEGRAM_BOT_TOKEN is required")
	}

	if chatID == "" {
		return nil, fmt.Errorf("telegram: TELEGRAM_CHAT_ID is required")
	}

	if threadIDValue == "" {
		return nil, fmt.Errorf("telegram: TELEGRAM_THREAD_ID is required")
	}

	threadID, err := strconv.ParseInt(threadIDValue, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("telegram: invalid TELEGRAM_THREAD_ID %q: %w", threadIDValue, err)
	}

	return NewClient(Config{
		BotToken: botToken,
		ChatID:   chatID,
		ThreadID: threadID,
		Timeout:  timeout,
	})
}

// NewClient creates a Telegram Bot API client.
func NewClient(cfg Config) (*Client, error) {
	if strings.TrimSpace(cfg.BotToken) == "" {
		return nil, fmt.Errorf("telegram: bot token is required")
	}

	if strings.TrimSpace(cfg.ChatID) == "" {
		return nil, fmt.Errorf("telegram: chat ID is required")
	}

	if cfg.ThreadID <= 0 {
		return nil, fmt.Errorf("telegram: thread ID must be greater than zero")
	}

	if cfg.Timeout <= 0 {
		cfg.Timeout = 10 * time.Second
	}

	return &Client{
		botToken: cfg.BotToken,
		chatID:   cfg.ChatID,
		threadID: cfg.ThreadID,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
	}, nil
}
