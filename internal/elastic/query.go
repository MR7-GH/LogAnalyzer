package elastic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
)

// Search executes a raw Elasticsearch JSON query against one or more indexes.
func (c *Client) Search(ctx context.Context, indexes []string, query []byte) (SearchResult, error) {
	if len(indexes) == 0 {
		return SearchResult{}, fmt.Errorf("elasticsearch: at least one index is required")
	}

	if len(query) == 0 {
		return SearchResult{}, fmt.Errorf("elasticsearch: query body is required")
	}

	res, err := c.raw.Search(
		c.raw.Search.WithContext(ctx),
		c.raw.Search.WithIndex(indexes...),
		c.raw.Search.WithBody(bytes.NewReader(query)),
		c.raw.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		return SearchResult{}, fmt.Errorf("elasticsearch: search request failed: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)

		return SearchResult{}, fmt.Errorf(
			"elasticsearch: search failed: status=%s body=%s",
			res.Status(),
			string(body),
		)
	}

	var response searchResponse

	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return SearchResult{}, fmt.Errorf("elasticsearch: decode search response: %w", err)
	}

	return mapSearchResponse(response), nil
}
