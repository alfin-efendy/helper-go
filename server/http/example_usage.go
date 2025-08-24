package http

import (
	"net/http"

	"github.com/alfin-efendy/helper-go/database/util"
	"gorm.io/gorm"
)

// Example struct for demonstration
type User struct {
	ID    uint   `json:"id" gorm:"primaryKey"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Age   int    `json:"age"`
}

// ExampleHandler demonstrates how to use the QueryParamsMiddleware with GORM scopes
func ExampleHandler(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := GetCtx(r)
		queryParams := ctx.GetQuery()

		var users []User
		var total int64

		// Count total records with search and filters (without pagination)
		countQuery := db.Model(&User{})

		// Apply search and filters for counting
		if queryParams.Search != "" {
			// Use custom search fields
			searchFields := []string{"name", "email"}
			countQuery = countQuery.Scopes(util.SearchCustomField(queryParams.Search, searchFields))
		}

		// Apply filters
		for field, value := range queryParams.Filters {
			if value != "" {
				countQuery = countQuery.Where(field+" = ?", value)
			}
		}

		// Get total count
		countQuery.Count(&total)

		// Query with pagination, search, filters, and ordering
		query := db.Model(&User{})

		// Apply all query parameters using the utility function
		query = query.Scopes(util.ApplyQuery(queryParams, User{}, []string{"name", "email"}))

		// Execute query
		if err := query.Find(&users).Error; err != nil {
			ctx.AddError(err)
			return
		}

		// Calculate pagination info
		totalPages := int(total) / queryParams.GetPageSize()
		if int(total)%queryParams.GetPageSize() > 0 {
			totalPages++
		}

		// Set response data and pagination
		ctx.SetData(users)
		ctx.SetPage(PageResponse{
			Page:      queryParams.GetPageNumber(),
			Limit:     queryParams.GetPageSize(),
			Total:     total,
			TotalPage: totalPages,
		})
	}
}

// ExampleHandlerWithJoins demonstrates how to use QueryParamsMiddleware with table joins
func ExampleHandlerWithJoins(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := GetCtx(r)
		queryParams := ctx.GetQuery()

		var users []User
		var total int64

		// For joins, use the table name version
		tableName := "users"
		searchFields := []string{"users.name", "users.email"}

		// Count total records
		countQuery := db.Table(tableName).
			Joins("LEFT JOIN profiles ON profiles.user_id = users.id")

		countQuery = countQuery.Scopes(util.ApplyQueryWithTableName(queryParams, tableName, searchFields))
		countQuery.Count(&total)

		// Query with joins
		query := db.Table(tableName).
			Select("users.*, profiles.bio").
			Joins("LEFT JOIN profiles ON profiles.user_id = users.id")

		query = query.Scopes(util.ApplyQueryWithTableName(queryParams, tableName, searchFields))

		if err := query.Find(&users).Error; err != nil {
			ctx.AddError(err)
			return
		}

		// Calculate pagination info
		totalPages := int(total) / queryParams.GetPageSize()
		if int(total)%queryParams.GetPageSize() > 0 {
			totalPages++
		}

		// Set response data and pagination
		ctx.SetData(users)
		ctx.SetPage(PageResponse{
			Page:      queryParams.GetPageNumber(),
			Limit:     queryParams.GetPageSize(),
			Total:     total,
			TotalPage: totalPages,
		})
	}
}

/*
Usage example in your server setup:

func SetupRoutes(db *gorm.DB) http.Handler {
	mux := http.NewServeMux()

	// Create a middleware chain
	middlewareChain := func(next http.Handler) http.Handler {
		return NewCtx(                    // Initialize context
			QueryParamsMiddleware(        // Parse query parameters
				LoggingMiddleware(        // Add logging
					TracingMiddleware(    // Add tracing
						ResponseMiddleware(next), // Handle responses
					),
				),
			),
		)
	}

	// Register routes
	mux.Handle("/users", middlewareChain(ExampleHandler(db)))
	mux.Handle("/users-with-joins", middlewareChain(ExampleHandlerWithJoins(db)))

	return mux
}

Query Parameters Examples:

1. Basic pagination:
   GET /users?page=1&limit=10

2. Search with pagination:
   GET /users?search=john&page=1&limit=10

3. Filtering:
   GET /users?filter_status=active&filter_role=admin

4. Ordering:
   GET /users?order_by=name&order=asc

5. Combined:
   GET /users?search=john&filter_status=active&order_by=created_at&order=desc&page=1&limit=20

Available query parameters:
- page: Page number (default: 1)
- limit: Items per page (default: 10, max: 100)
- search: Search term (searches in specified fields)
- order_by: Field to order by (default: id)
- order: Order direction (asc/desc, default: desc)
- filter_[field]: Filter by field value (e.g., filter_status=active)

*/
