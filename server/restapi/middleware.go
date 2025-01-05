package restapi

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/alfin-efendy/helper-go/config"
	"github.com/alfin-efendy/helper-go/logger"
	"github.com/alfin-efendy/helper-go/token"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// RequestIDMiddleware generates a unique request ID and sets it in the response header.
// It is a middleware function for Gin framework.
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("X-Request-Id", uuid.New().String())
		c.Next()
	}
}

// LoggerMiddleware logs the request and response details.
// It is a middleware function for Gin framework.
func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start timer
		start := time.Now()

		// Process request
		c.Next()

		// Stop timer
		end := time.Now()
		latency := end.Sub(start)

		// Log details
		logger.Info(
			c.Request.Context(),
			"Request",
			logrus.Fields{
				"method":  c.Request.Method,
				"path":    c.Request.URL.Path,
				"status":  c.Writer.Status(),
				"latency": latency.String(),
			},
		)
	}
}

// CORSMiddleware adds CORS headers to the response.
// It is a middleware function for Gin framework.
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if config.Config.Server.RestAPI.Cors == nil {
			c.Next()
		}

		config := config.Config.Server.RestAPI.Cors
		for _, origin := range config.AllowOrigins {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		}
		for _, method := range config.AllowMethods {
			c.Writer.Header().Set("Access-Control-Allow-Methods", method)
		}
		for _, header := range config.AllowHeaders {
			c.Writer.Header().Set("Access-Control-Allow-Headers", header)
		}
		for _, exposeHeaders := range config.ExposeHeaders {
			c.Writer.Header().Set("Access-Control-Expose-Headers", exposeHeaders)
		}
		c.Writer.Header().Set("Access-Control-Max-Age", strconv.Itoa(config.MaxAge))
		c.Writer.Header().Set("Access-Control-Allow-Credentials", strconv.FormatBool(config.AllowCredentials))

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// HelmetMiddleware adds security headers to the response.
func HelmetMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("X-Content-Type-Options", "nosniff")
		c.Writer.Header().Set("X-Frame-Options", "DENY always")
		c.Writer.Header().Set("X-XSS-Protection", "1; mode=block")
		c.Writer.Header().Set("X-DNS-Prefetch-Control", "off")
		c.Writer.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		c.Writer.Header().Set("Referrer-Policy", "no-referrer")
		c.Writer.Header().Set("Cache-control", "no-store")
		c.Writer.Header().Set("Content-Security-Policy", "default-src 'self'")
		c.Writer.Header().Set("Permissions-Policy", "geolocation=(), midi=(), notifications=(), push=(), sync-xhr=(), microphone=(), camera=(), magnetometer=(), gyroscope=(), speaker=(), vibrate=(), fullscreen=(self), payment=()")
		c.Next()
	}
}

// AuthMiddleware is a middleware function that checks if the request is authorized.
func AuthMiddleware(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		accessToken, err := c.Cookie("access_token")

		if err != nil {
			// Get authorization header
			authorization := c.GetHeader("Authorization")

			// Check authorization header
			if !strings.HasPrefix(authorization, "Bearer ") {
				APIResponse(c, "Need Bearer Authorization", http.StatusUnauthorized, nil)
				return
			}

			// Get access token
			accessToken = strings.TrimPrefix(authorization, "Bearer ")
		}

		dataAccess, err := token.TokenValidation(c, accessToken, "access", true)

		if err != nil {
			APIResponse(c, "Unauthorized", http.StatusUnauthorized, nil)
			return
		}

		// Check if token has the required permission
		if permission != "" {
			// Check if the Audience has the required permission
			ability := dataAccess.Audience
			hasPermission := false

			for _, v := range ability {
				if v == permission {
					hasPermission = true
					break
				}
			}

			if !hasPermission {
				APIResponse(c, "Forbidden", http.StatusForbidden, nil)
				return
			}
		}

		c.Set("issuer", dataAccess.Issuer)
		c.Set("subject", dataAccess.Subject)
		c.Next()
	}
}
