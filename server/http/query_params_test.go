package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alfin-efendy/helper-go/database/util"
)

func TestQueryParamsMiddleware(t *testing.T) {
	// Create a test handler that checks the query parameters
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := GetCtx(r)
		queryParams := ctx.GetQuery()

		// Test pagination parameters
		if queryParams.GetPageNumber() != 2 {
			t.Errorf("Expected page 2, got %d", queryParams.GetPageNumber())
		}

		if queryParams.GetPageSize() != 20 {
			t.Errorf("Expected limit 20, got %d", queryParams.GetPageSize())
		}

		// Test search parameter
		if queryParams.GetSearch() != "test search" {
			t.Errorf("Expected search 'test search', got '%s'", queryParams.GetSearch())
		}

		// Test order parameters
		if queryParams.Order != "asc" {
			t.Errorf("Expected order 'asc', got '%s'", queryParams.Order)
		}

		if queryParams.OrderBy != "name" {
			t.Errorf("Expected order_by 'name', got '%s'", queryParams.OrderBy)
		}

		// Test filters
		filters := queryParams.GetFilters()
		if filters["status"] != "active" {
			t.Errorf("Expected filter status 'active', got '%s'", filters["status"])
		}

		if filters["role"] != "admin" {
			t.Errorf("Expected filter role 'admin', got '%s'", filters["role"])
		}

		w.WriteHeader(http.StatusOK)
	})

	// Create middleware chain
	middlewareChain := NewCtx(QueryParamsMiddleware(testHandler))

	// Create test request with query parameters
	req := httptest.NewRequest("GET", "/test?page=2&limit=20&search=test+search&order=asc&order_by=name&filter_status=active&filter_role=admin", nil)
	w := httptest.NewRecorder()

	// Execute the request
	middlewareChain.ServeHTTP(w, req)

	// Check response status
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestQueryParamsDefaults(t *testing.T) {
	// Create a test handler that checks default values
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := GetCtx(r)
		queryParams := ctx.GetQuery()

		// Test default values
		if queryParams.GetPageNumber() != 1 {
			t.Errorf("Expected default page 1, got %d", queryParams.GetPageNumber())
		}

		if queryParams.GetPageSize() != 10 {
			t.Errorf("Expected default limit 10, got %d", queryParams.GetPageSize())
		}

		if queryParams.GetSearch() != "" {
			t.Errorf("Expected empty search, got '%s'", queryParams.GetSearch())
		}

		orderClause := queryParams.GetOrderClause()
		if orderClause != "id desc" {
			t.Errorf("Expected default order clause 'id desc', got '%s'", orderClause)
		}

		w.WriteHeader(http.StatusOK)
	})

	// Create middleware chain
	middlewareChain := NewCtx(QueryParamsMiddleware(testHandler))

	// Create test request without query parameters
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	// Execute the request
	middlewareChain.ServeHTTP(w, req)

	// Check response status
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestQueryParamsLimits(t *testing.T) {
	// Create a test handler that checks limits
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := GetCtx(r)
		queryParams := ctx.GetQuery()

		// Test page size limit (should be capped at 100)
		if queryParams.GetPageSize() != 100 {
			t.Errorf("Expected page size to be capped at 100, got %d", queryParams.GetPageSize())
		}

		// Test negative page (should default to 1)
		if queryParams.GetPageNumber() != 1 {
			t.Errorf("Expected negative page to default to 1, got %d", queryParams.GetPageNumber())
		}

		w.WriteHeader(http.StatusOK)
	})

	// Create middleware chain
	middlewareChain := NewCtx(QueryParamsMiddleware(testHandler))

	// Create test request with extreme values
	req := httptest.NewRequest("GET", "/test?page=-1&limit=500", nil)
	w := httptest.NewRecorder()

	// Execute the request
	middlewareChain.ServeHTTP(w, req)

	// Check response status
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// Example test for scope functions
func TestScopeInteraction(t *testing.T) {
	// Mock query params
	queryParams := &QueryParams{
		Page:    2,
		Limit:   15,
		Search:  "test",
		Order:   "asc",
		OrderBy: "name",
		Filters: map[string]string{
			"status": "active",
		},
	}

	// Test that the interface methods work correctly
	if queryParams.GetPageNumber() != 2 {
		t.Errorf("Expected page 2, got %d", queryParams.GetPageNumber())
	}

	if queryParams.GetPageSize() != 15 {
		t.Errorf("Expected limit 15, got %d", queryParams.GetPageSize())
	}

	if queryParams.GetSearch() != "test" {
		t.Errorf("Expected search 'test', got '%s'", queryParams.GetSearch())
	}

	if queryParams.GetOrderClause() != "name asc" {
		t.Errorf("Expected order clause 'name asc', got '%s'", queryParams.GetOrderClause())
	}

	filters := queryParams.GetFilters()
	if filters["status"] != "active" {
		t.Errorf("Expected filter status 'active', got '%s'", filters["status"])
	}

	// Test that it implements the interface
	var _ util.QueryParamsInterface = queryParams
}
