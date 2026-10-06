package elastic

import (
	"context"
	"fmt"
)

// Source represents one Elasticsearch data source.
type Source struct {
	name   string
	client *Client
}

// NewSource creates a new Elasticsearch source.
func NewSource(name string, cfg Config) (*Source, error) {
	if name == "" {
		return nil, fmt.Errorf("elasticsearch: source name is required")
	}

	client, err := NewClient(cfg)
	if err != nil {
		return nil, err
	}

	return &Source{
		name:   name,
		client: client,
	}, nil
}

// Name returns the unique source name.
func (s *Source) Name() string {
	return s.name
}

// Check verifies connectivity to the source.
func (s *Source) Check(ctx context.Context) (*ClusterInfo, error) {
	return s.client.Check(ctx)
}

// Client returns the Elasticsearch client used by this source.
func (s *Source) Client() *Client {
	return s.client
}

// Search executes a query against this Elasticsearch source.
func (s *Source) Search(ctx context.Context, indexes []string, query []byte) (SearchResult, error) {
	return s.client.Search(ctx, indexes, query)
}
