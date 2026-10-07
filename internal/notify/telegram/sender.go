package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// sendMessageRequest contains a Telegram sendMessage request body.
type sendMessageRequest struct {
	ChatID          string `json:"chat_id"`
	MessageThreadID int64  `json:"message_thread_id"`
	Text            string `json:"text"`
}

// sendMessageResponse contains the Telegram Bot API response fields required by the client.
type sendMessageResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description"`
}

// Send sends a text message to the configured Telegram chat and thread.
func (c *Client) Send(ctx context.Context, text string) error {
	chunks := splitMessage(text, 3900)

	for _, chunk := range chunks {
		if err := c.sendChunk(ctx, chunk); err != nil {
			return err
		}
	}

	return nil
}

// sendChunk sends one Telegram-safe message chunk.
func (c *Client) sendChunk(ctx context.Context, text string) error {
	body, err := json.Marshal(sendMessageRequest{
		ChatID:          c.chatID,
		MessageThreadID: c.threadID,
		Text:            text,
	})
	if err != nil {
		return fmt.Errorf("telegram: marshal message: %w", err)
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", c.botToken)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("telegram: create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	res, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("telegram: send message: %w", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("telegram: read response: %w", err)
	}

	var telegramResponse sendMessageResponse
	if err := json.Unmarshal(responseBody, &telegramResponse); err != nil {
		return fmt.Errorf("telegram: decode response: %w", err)
	}

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("telegram: request failed status=%s body=%s", res.Status, string(responseBody))
	}

	if !telegramResponse.OK {
		return fmt.Errorf("telegram: API rejected message: %s", telegramResponse.Description)
	}

	return nil
}

// splitMessage splits long messages into Telegram-safe chunks.
func splitMessage(text string, maxRunes int) []string {
	if maxRunes <= 0 {
		return []string{text}
	}

	runes := []rune(text)
	if len(runes) <= maxRunes {
		return []string{text}
	}

	chunks := make([]string, 0, (len(runes)/maxRunes)+1)

	for len(runes) > 0 {
		end := maxRunes
		if len(runes) < end {
			end = len(runes)
		}

		splitAt := end

		if end < len(runes) {
			for i := end - 1; i > 0; i-- {
				if runes[i] == '\n' {
					splitAt = i + 1
					break
				}
			}
		}

		chunks = append(chunks, string(runes[:splitAt]))
		runes = runes[splitAt:]
	}

	return chunks
}
