package restapi

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/alfin-efendy/helper-go/config"
	"github.com/alfin-efendy/helper-go/logger"
	"github.com/alfin-efendy/helper-go/token"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap/zapcore"
)

// ResponseWriter is a custom response writer to capture the response body
type ResponseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w ResponseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// LoggerMiddleware logs the request and response details.
// It is a middleware function for Gin framework.
func loggerMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// Start timer
		start := time.Now()

		// Read the request body
		payload, err := io.ReadAll(ctx.Request.Body)
		if err == nil {
			ctx.Request.Body = io.NopCloser(bytes.NewBuffer(payload))
		}

		// Create a new body buffer
		responseWriter := &ResponseWriter{body: bytes.NewBufferString(""), ResponseWriter: ctx.Writer}
		ctx.Writer = responseWriter

		// Process request
		ctx.Next()

		// Stop timer
		end := time.Now()
		latency := end.Sub(start)

		msg := fmt.Sprintf("[%v] [%v] %s %s", latency, ctx.Writer.Status(), ctx.Request.Method, ctx.Request.RequestURI)

		// Log details
		logger.Info(
			ctx.Request.Context(),
			msg,
			zapcore.Field{Key: "Method", Type: zapcore.StringType, String: ctx.Request.Method},
			zapcore.Field{Key: "Query", Type: zapcore.StringType, String: ctx.Request.URL.RawQuery},
			zapcore.Field{Key: "Payload", Type: zapcore.ByteStringType, Interface: payload},
			zapcore.Field{Key: "Status", Type: zapcore.Int64Type, Integer: int64(ctx.Writer.Status())},
			zapcore.Field{Key: "Response", Type: zapcore.ByteStringType, Interface: responseWriter.body.Bytes()},
			zapcore.Field{Key: "Latency", Type: zapcore.StringType, String: latency.String()},
			zapcore.Field{Key: "ClientIP", Type: zapcore.StringType, String: ctx.ClientIP()},
			zapcore.Field{Key: "UserAgent", Type: zapcore.StringType, String: ctx.GetHeader("User-Agent")},
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
