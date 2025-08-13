package http

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/alfin-efendy/helper-go/config"
	"github.com/alfin-efendy/helper-go/database"
)

// HealthCheckResponse represents the structure of health check response
type HealthCheckResponse struct {
	Status    string                 `json:"status"`
	Timestamp string                 `json:"timestamp"`
	Services  map[string]ServiceInfo `json:"services"`
}

// ServiceInfo represents information about a service
type ServiceInfo struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// HealthCheck handles health check requests using the Handler pattern
func HealthCheck(c *Ctx) {
	ctx := c.Request.Context()

	// Initialize response
	response := HealthCheckResponse{
		Status:    "healthy",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Services:  make(map[string]ServiceInfo),
	}

	overallHealthy := true

	// Check SQL Database connection (only if configured)
	sqlStatus := ServiceInfo{Status: "healthy"}
	if config.Data != nil && config.Data.Database.SQL != nil {
		if !database.IsSQLConnected() {
			sqlStatus.Status = "unhealthy"
			sqlStatus.Message = "Database connection is not available"
			overallHealthy = false
		} else {
			// Additional ping test
			pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			if err := database.PingSQLDatabase(pingCtx); err != nil {
				sqlStatus.Status = "unhealthy"
				sqlStatus.Message = fmt.Sprintf("Database ping failed: %v", err)
				overallHealthy = false
			}
		}
		response.Services["database"] = sqlStatus
	} else {
		sqlStatus.Status = "disabled"
		sqlStatus.Message = "Database not configured"
		response.Services["database"] = sqlStatus
	}

	// Check Redis connection (only if configured)
	redisStatus := ServiceInfo{Status: "healthy"}
	if config.Data != nil && config.Data.Database.Redis != nil {
		redisCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		if !database.IsRedisConnected(redisCtx) {
			redisStatus.Status = "unhealthy"
			redisStatus.Message = "Redis connection is not available"
			overallHealthy = false
		}
		response.Services["redis"] = redisStatus
	} else {
		redisStatus.Status = "disabled"
		redisStatus.Message = "Redis not configured"
		response.Services["redis"] = redisStatus
	}

	// Set overall status and HTTP status code
	if !overallHealthy {
		response.Status = "unhealthy"
		c.ResponseWriter.WriteHeader(http.StatusServiceUnavailable)
	} else {
		c.ResponseWriter.WriteHeader(http.StatusOK)
	}

	// Set the response data for the middleware to handle
	c.Data = response
}
