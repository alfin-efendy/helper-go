package http

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/alfin-efendy/helper-go/config"
	"github.com/alfin-efendy/helper-go/logger"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// RouteInfo represents information about a registered route
type RouteInfo struct {
	Method string
	Path   string
}

// extractRoutes walks the chi router to extract all registered routes
func extractRoutes(r chi.Router) []RouteInfo {
	var routes []RouteInfo

	// Walk through all routes
	walkFunc := func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		// Clean up the route path
		cleanPath := strings.ReplaceAll(route, "*", "{*}")
		routes = append(routes, RouteInfo{
			Method: method,
			Path:   cleanPath,
		})
		return nil
	}

	err := chi.Walk(r, walkFunc)
	if err != nil {
		logger.Error(context.Background(), err, "Failed to walk routes")
	}

	// Sort routes for consistent output
	sort.Slice(routes, func(i, j int) bool {
		if routes[i].Path == routes[j].Path {
			return routes[i].Method < routes[j].Method
		}
		return routes[i].Path < routes[j].Path
	})

	return routes
}

// printRoutes logs all registered routes similar to Gin's route listing
func printRoutes(r chi.Router, serverAddr string) {
	routes := extractRoutes(r)

	if len(routes) == 0 {
		logger.Info(context.Background(), "No routes registered")
		return
	}

	logger.Info(context.Background(), "Server starting...",
		"addr", serverAddr,
		"routes_count", len(routes))

	// Print header
	logger.Info(context.Background(), "Registered routes:")

	// Group routes by method for better readability
	methodGroups := make(map[string][]string)
	for _, route := range routes {
		methodGroups[route.Method] = append(methodGroups[route.Method], route.Path)
	}

	// Print routes grouped by method
	for method, paths := range methodGroups {
		for _, path := range paths {
			logger.Info(context.Background(), fmt.Sprintf("%-7s %s", method, path))
		}
	}
}

func NewHTTPServer() *http.Server {
	// Server configuration
	config := config.Data.Server.RestAPI
	if config.Host == "" {
		config.Host = "localhost"
	}
	if config.Port == 0 {
		config.Port = 8080
	}

	r := chi.NewRouter()
	r.Use(LoggingMiddleware) // Use our custom logging middleware integrated with zerolog
	r.Use(middleware.Recoverer)
	r.Use(middleware.CleanPath)
	r.Use(TracingMiddleware) // Add tracing middleware to extract trace ID and set response header
	r.Use(NewCtx)
	r.Use(ResponseMiddleware) // Add the response middleware

	// Health check endpoint
	r.Get("/health", Handler(HealthCheck))

	// Print registered routes
	serverAddr := fmt.Sprintf("%s:%d", config.Host, config.Port)
	printRoutes(r, serverAddr)

	return &http.Server{
		Addr:              fmt.Sprintf("%s:%d", config.Host, config.Port),
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,  // Prevent Slowloris attacks
		ReadTimeout:       30 * time.Second,  // Total time to read request
		WriteTimeout:      30 * time.Second,  // Total time to write response
		IdleTimeout:       120 * time.Second, // Keep-alive timeout
	}
}
