package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Health returns the current HTTP service health state.
func Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
