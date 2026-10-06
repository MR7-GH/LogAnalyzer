package handler

import (
	"context"
	"net/http"
	"strings"

	"LogAnalyzer/internal/analyze"

	"github.com/gin-gonic/gin"
)

// StatsService defines the behavior required by the stats HTTP handler.
type StatsService interface {
	Stats(ctx context.Context, service string) (analyze.StatusResult, error)
}

// Stats handles stats API requests.
type Stats struct {
	service StatsService
}

// NewStats creates a new stats handler.
func NewStats(service StatsService) *Stats {
	return &Stats{service: service}
}

// Get returns status statistics for the requested service.
func (h *Stats) Get(c *gin.Context) {
	service := strings.TrimSpace(c.Query("service"))

	if service == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "service is required"})
		return
	}

	result, err := h.service.Stats(c.Request.Context(), service)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}
