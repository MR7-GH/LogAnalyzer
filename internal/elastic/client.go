package elastic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	elasticsearch "github.com/elastic/go-elasticsearch/v8"
)

// Config contains the configuration required to connect to Elasticsearch.
type Config struct {
	Addresses []string
	Username  string
	Password  string
	APIKey    string
	Timeout   time.Duration
}

// Client wraps the official Elasticsearch client.
type Client struct {
	raw *elasticsearch.Client
}

// ClusterInfo contains basic information returned by Elasticsearch.
type ClusterInfo struct {
	Name        string `json:"name"`
	ClusterName string `json:"cluster_name"`
	ClusterUUID string `json:"cluster_uuid"`

	Version struct {
		Number string `json:"number"`
	} `json:"version"`

	Tagline string `json:"tagline"`
}

// NewClient creates a new Elasticsearch client.
func NewClient(cfg Config) (*Client, error) {
	if len(cfg.Addresses) == 0 {
		return nil, errors.New("elasticsearch: at least one address is required")
	}

	if cfg.Timeout <= 0 {
		cfg.Timeout = 10 * time.Second
	}

	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   cfg.Timeout,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ResponseHeaderTimeout: cfg.Timeout,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
	}

	esCfg := elasticsearch.Config{
		Addresses: cfg.Addresses,
		Username:  cfg.Username,
		Password:  cfg.Password,
		APIKey:    cfg.APIKey,
		Transport: transport,
	}

	raw, err := elasticsearch.NewClient(esCfg)
	if err != nil {
		return nil, fmt.Errorf("elasticsearch: create client: %w", err)
	}

	return &Client{
		raw: raw,
	}, nil
}

// Check verifies the Elasticsearch connection and returns cluster information.
func (c *Client) Check(ctx context.Context) (*ClusterInfo, error) {
	res, err := c.raw.Info(
		c.raw.Info.WithContext(ctx),
	)
	if err != nil {
		return nil, fmt.Errorf("elasticsearch: connection failed: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)

		return nil, fmt.Errorf(
			"elasticsearch: info request failed: status=%s body=%s",
			res.Status(),
			string(body),
		)
	}

	var info ClusterInfo

	if err := json.NewDecoder(res.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("elasticsearch: decode cluster info: %w", err)
	}

	return &info, nil
}

// Raw returns the underlying official Elasticsearch client.
func (c *Client) Raw() *elasticsearch.Client {
	return c.raw
}
