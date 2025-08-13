package http

import (
	"testing"

	"github.com/alfin-efendy/helper-go/config"
	"github.com/alfin-efendy/helper-go/config/schema"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

// setupTestConfigMinimal sets up minimal configuration for testing
func setupTestConfigMinimal() {
	config.Data = &schema.Config{
		Server: schema.Server{
			RestAPI: schema.RestAPI{
				Host: "localhost",
				Port: 8080,
			},
		},
	}
}

func TestExtractRoutes(t *testing.T) {
	r := chi.NewRouter()

	// Add some test routes
	r.Get("/health", Handler(HealthCheck))
	r.Post("/users", Handler(func(c *Ctx) {}))
	r.Put("/users/{id}", Handler(func(c *Ctx) {}))
	r.Delete("/users/{id}", Handler(func(c *Ctx) {}))
	r.Get("/api/v1/products", Handler(func(c *Ctx) {}))
	r.Post("/api/v1/products", Handler(func(c *Ctx) {}))

	routes := extractRoutes(r)

	// Should have all the routes we added
	assert.Len(t, routes, 6)

	// Check that routes are sorted
	expectedRoutes := []RouteInfo{
		{Method: "GET", Path: "/api/v1/products"},
		{Method: "POST", Path: "/api/v1/products"},
		{Method: "GET", Path: "/health"},
		{Method: "POST", Path: "/users"},
		{Method: "DELETE", Path: "/users/{id}"},
		{Method: "PUT", Path: "/users/{id}"},
	}

	assert.Equal(t, expectedRoutes, routes)
}

func TestPrintRoutes(t *testing.T) {
	r := chi.NewRouter()

	// Add some test routes
	r.Get("/health", Handler(HealthCheck))
	r.Post("/users", Handler(func(c *Ctx) {}))
	r.Get("/api/v1/products", Handler(func(c *Ctx) {}))

	// This should not panic and should print route information
	// In a real scenario, you would capture log output to test this properly
	assert.NotPanics(t, func() {
		printRoutes(r, "localhost:8080")
	})
}

func TestNewHTTPServerRoutes(t *testing.T) {
	// Setup minimal config for testing
	setupTestConfigMinimal()

	// Test that NewHTTPServer creates a server and lists routes without panicking
	assert.NotPanics(t, func() {
		server := NewHTTPServer()
		assert.NotNil(t, server)
		assert.Equal(t, "localhost:8080", server.Addr)
	})
}

// TestRouteListingIntegration demonstrates the complete route listing functionality
func TestRouteListingIntegration(t *testing.T) {
	// Create a router with multiple routes
	r := chi.NewRouter()

	// Add middleware (to simulate real usage)
	r.Use(LoggingMiddleware)
	r.Use(TracingMiddleware)

	// Add routes
	r.Get("/health", Handler(HealthCheck))
	r.Route("/api", func(r chi.Router) {
		r.Route("/v1", func(r chi.Router) {
			r.Get("/users", Handler(func(c *Ctx) {}))
			r.Post("/users", Handler(func(c *Ctx) {}))
			r.Get("/users/{id}", Handler(func(c *Ctx) {}))
			r.Put("/users/{id}", Handler(func(c *Ctx) {}))
			r.Delete("/users/{id}", Handler(func(c *Ctx) {}))
		})
	})

	// Extract routes
	routes := extractRoutes(r)

	// Should have all nested routes
	assert.GreaterOrEqual(t, len(routes), 6) // At least 6 routes (health + 5 user routes)

	// Check that nested routes are properly extracted
	routePaths := make([]string, len(routes))
	for i, route := range routes {
		routePaths[i] = route.Path
	}

	assert.Contains(t, routePaths, "/health")
	assert.Contains(t, routePaths, "/api/v1/users")
	assert.Contains(t, routePaths, "/api/v1/users/{id}")
}
