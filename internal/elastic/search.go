package elastic

import (
	"bytes"
	"context"
	"fmt"
	"io"
)

// SearchRaw executes a raw JSON search query against the given Elasticsearch indexes.
func (c *Client) SearchRaw(ctx context.Context, indexes []string, body []byte) ([]byte, error) {
	if len(indexes) == 0 {
		return nil, fmt.Errorf("elasticsearch: at least one index is required")
	}

	if len(body) == 0 {
		return nil, fmt.Errorf("elasticsearch: query body is required")
	}

	res, err := c.raw.Search(c.raw.Search.WithContext(ctx), c.raw.Search.WithIndex(indexes...), c.raw.Search.WithBody(bytes.NewReader(body)))
	if err != nil {
		return nil, fmt.Errorf("elasticsearch: search request failed: %w", err)
	}
	defer res.Body.Close()

	response, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("elasticsearch: read search response: %w", err)
	}

	if res.IsError() {
		return nil, fmt.Errorf("elasticsearch: search failed: status=%s body=%s", res.Status(), string(response))
	}

	return response, nil
}
