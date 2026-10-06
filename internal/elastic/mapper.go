package elastic

import "encoding/json"

// Document represents one normalized Elasticsearch document.
type Document struct {
	Index  string
	ID     string
	Source json.RawMessage
}

// SearchResult represents a normalized Elasticsearch search response.
type SearchResult struct {
	Took      int64
	Total     int64
	Documents []Document
}

// searchResponse represents the Elasticsearch JSON response used internally.
type searchResponse struct {
	Took int64 `json:"took"`

	Hits struct {
		Total struct {
			Value int64 `json:"value"`
		} `json:"total"`

		Hits []struct {
			Index  string          `json:"_index"`
			ID     string          `json:"_id"`
			Source json.RawMessage `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
}

// mapSearchResponse converts an Elasticsearch response to the internal result model.
func mapSearchResponse(response searchResponse) SearchResult {
	documents := make([]Document, 0, len(response.Hits.Hits))

	for _, hit := range response.Hits.Hits {
		documents = append(documents, Document{
			Index:  hit.Index,
			ID:     hit.ID,
			Source: hit.Source,
		})
	}

	return SearchResult{
		Took:      response.Took,
		Total:     response.Hits.Total.Value,
		Documents: documents,
	}
}
