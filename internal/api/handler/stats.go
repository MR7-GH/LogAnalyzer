package handler

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"LogAnalyzer/internal/analyze"

	"github.com/gin-gonic/gin"
)

// StatsService defines the behavior required by the stats HTTP handler.
type StatsService interface {
	Stats(ctx context.Context, service string, options analyze.StatsOptions) (analyze.StatsResponse, error)
}

// Stats handles stats API requests.
type Stats struct {
	service StatsService
}

// NewStats creates a new stats handler.
func NewStats(service StatsService) *Stats {
	return &Stats{
		service: service,
	}
}

// Get returns statistics for the requested service and configured services.
func (h *Stats) Get(c *gin.Context) {
	service := strings.TrimSpace(c.Query("service"))
	if service == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "service is required",
		})
		return
	}

	detailed, err := parseDetailOption(c.Query("detail"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	result, err := h.service.Stats(c.Request.Context(), service, analyze.StatsOptions{
		Detailed: detailed,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// parseDetailOption parses the optional detail query parameter.
func parseDetailOption(value string) (bool, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return false, nil
	}

	detailed, err := strconv.ParseBool(value)
	if err != nil {
		return false, &invalidDetailError{}
	}

	return detailed, nil
}

// invalidDetailError represents an invalid detail query parameter.
type invalidDetailError struct{}

// Error returns the detail parameter validation message.
func (e *invalidDetailError) Error() string {
	return "detail must be true or false"
}
