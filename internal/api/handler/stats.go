package handler

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"LogAnalyzer/internal/analyze"

	"github.com/gin-gonic/gin"
)

// StatsProcessor defines asynchronous and synchronous stats processing behavior.
type StatsProcessor interface {
	Submit(service string, options analyze.StatsOptions) error
	Process(ctx context.Context, service string, options analyze.StatsOptions) (analyze.StatsResponse, error)
}

// Stats handles stats API requests.
type Stats struct {
	processor StatsProcessor
}

// NewStats creates a new stats handler.
func NewStats(processor StatsProcessor) *Stats {
	return &Stats{
		processor: processor,
	}
}

// Get processes a stats request asynchronously or synchronously based on return_result.
func (h *Stats) Get(c *gin.Context) {
	service := strings.TrimSpace(c.Query("service"))
	if service == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "service is required",
		})
		return
	}

	detailed, err := parseBoolOption(c.Query("detail"), "detail")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	returnResult, err := parseBoolOption(c.Query("return_result"), "return_result")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	options := analyze.StatsOptions{
		Detailed: detailed,
	}

	if returnResult {
		result, err := h.processor.Process(c.Request.Context(), service, options)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, result)
		return
	}

	if err := h.processor.Submit(service, options); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"status": "accepted",
	})
}

// parseBoolOption parses an optional boolean query parameter.
func parseBoolOption(value, name string) (bool, error) {
	value = strings.TrimSpace(value)

	if value == "" {
		return false, nil
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("%s must be true or false", name)
	}

	return parsed, nil
}
