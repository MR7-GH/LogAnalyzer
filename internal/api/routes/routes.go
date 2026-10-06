package routes

import (
	"LogAnalyzer/internal/api/handler"

	"github.com/gin-gonic/gin"
)

// New creates and configures the LogAnalyzer HTTP router.
func New(stats *handler.Stats) *gin.Engine {
	router := gin.Default()

	router.GET("/health", handler.Health)
	router.GET("/stats", stats.Get)

	return router
}
