package server

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"sslplatform/internal/platform/database"
)

func RegisterHealth(engine *gin.Engine, db *gorm.DB, _ database.Ready) {
	engine.GET("/health/live", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "live"})
	})
	engine.GET("/health/ready", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		sqlDB, err := db.DB()
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not_ready"})
			return
		}
		if err := sqlDB.PingContext(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not_ready"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})
}
