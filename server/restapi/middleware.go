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
func requestIDMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Writer.Header().Set("X-Request-Id", uuid.New().String())
		ctx.Next()
	}
}

// LoggerMiddleware logs the request and response details.
// It is a middleware function for Gin framework.
func loggerMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// Start timer
		start := time.Now()

		// Process request
		ctx.Next()

		// Stop timer
		end := time.Now()
		latency := end.Sub(start)

		// Log details
		logger.Info(
			ctx.Request.Context(),
			"Request",
			logrus.Fields{
				"method":  ctx.Request.Method,
				"path":    ctx.Request.URL.Path,
				"status":  ctx.Writer.Status(),
				"latency": latency.String(),
			},
		)
	}
}

// CORSMiddleware adds CORS headers to the response.
// It is a middleware function for Gin framework.
func corsMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if config.Config.Server.RestAPI.Cors == nil {
			ctx.Next()
		}

		config := config.Config.Server.RestAPI.Cors
		for _, origin := range config.AllowOrigins {
			ctx.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		}
		for _, method := range config.AllowMethods {
			ctx.Writer.Header().Set("Access-Control-Allow-Methods", method)
		}
		for _, header := range config.AllowHeaders {
			ctx.Writer.Header().Set("Access-Control-Allow-Headers", header)
		}
		for _, exposeHeaders := range config.ExposeHeaders {
			ctx.Writer.Header().Set("Access-Control-Expose-Headers", exposeHeaders)
		}
		ctx.Writer.Header().Set("Access-Control-Max-Age", strconv.Itoa(config.MaxAge))
		ctx.Writer.Header().Set("Access-Control-Allow-Credentials", strconv.FormatBool(config.AllowCredentials))

		if ctx.Request.Method == "OPTIONS" {
			ctx.AbortWithStatus(204)
			return
		}

		ctx.Next()
	}
}

// HelmetMiddleware adds security headers to the response.
func helmetMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Writer.Header().Set("X-Content-Type-Options", "nosniff")
		ctx.Writer.Header().Set("X-Frame-Options", "DENY always")
		ctx.Writer.Header().Set("X-XSS-Protection", "1; mode=block")
		ctx.Writer.Header().Set("X-DNS-Prefetch-Control", "off")
		ctx.Writer.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		ctx.Writer.Header().Set("Referrer-Policy", "no-referrer")
		ctx.Writer.Header().Set("Cache-control", "no-store")
		ctx.Writer.Header().Set("Content-Security-Policy", "default-src 'self'")
		ctx.Writer.Header().Set("Permissions-Policy", "geolocation=(), midi=(), notifications=(), push=(), sync-xhr=(), microphone=(), camera=(), magnetometer=(), gyroscope=(), speaker=(), vibrate=(), fullscreen=(self), payment=()")
		ctx.Next()
	}
}

// AuthMiddleware is a middleware function that checks if the request is authorized.
func AuthMiddleware(permission string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		accessToken, err := ctx.Cookie("access_token")

		if err != nil {
			// Get authorization header
			authorization := ctx.GetHeader("Authorization")

			// Check authorization header
			if !strings.HasPrefix(authorization, "Bearer ") {
				ctx.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
				ctx.Abort()
				return
			}

			// Get access token
			accessToken = strings.TrimPrefix(authorization, "Bearer ")
		}

		dataAccess, err := token.TokenValidation(ctx, accessToken, "access", true)

		if err != nil {
			ctx.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
			ctx.Abort()
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
				ctx.JSON(http.StatusForbidden, gin.H{"message": "Access Denied"})
				ctx.Abort()
				return
			}
		}

		ctx.Set("issuer", dataAccess.Issuer)
		ctx.Set("subject", dataAccess.Subject)
		ctx.Next()
	}
}
