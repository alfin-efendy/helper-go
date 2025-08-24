package util

import (
	"fmt"
	"strings"
	"sync"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// QueryParams interface for external query parameters
type QueryParamsInterface interface {
	GetPageNumber() int
	GetPageSize() int
	GetOrderClause() string
	GetSearch() string
	GetFilters() map[string]string
}

// For Scope of pagination
// https://gorm.io/docs/scopes.html#Pagination
func Paginate(page, pageSize int) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if page <= 0 {
			page = 1
		}

		switch {
		case pageSize > 100:
			pageSize = 100
		case pageSize <= 0:
			pageSize = 10
		}

		offset := (page - 1) * pageSize
		return db.Offset(offset).Limit(pageSize)
	}
}

// For Scope of search all fields
// https://gorm.io/docs/scopes.html#Search
func Search(search string, table interface{}) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if search != "" {
			// Loop through the fields and add them to the query
			s, err := schema.Parse(&table, &sync.Map{}, schema.NamingStrategy{})
			if err != nil {
				return db
			}
			query := ""
			values := []interface{}{}
			for _, field := range s.Fields {
				if field.Name != "" {
					query += field.Name + " LIKE ?"
					if field.Name != s.Fields[len(s.Fields)-1].Name {
						query += " OR "
					}
					values = append(values, "%"+search+"%")
				}
			}

			db = db.Where(query, values...)
		}
		return db
	}
}

// For Scope of search specific fields
func SearchCustomField(search string, fields []string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if search != "" {
			// Loop through the fields and add them to the query
			query := ""
			values := []interface{}{}
			for _, field := range fields {
				if field != "" {
					query += field + " LIKE ?"
					if field != fields[len(fields)-1] {
						query += " OR "
					}
					values = append(values, "%"+search+"%")
				}
			}

			db = db.Where(query, values...)
		}
		return db
	}
}

func IsActive(db *gorm.DB) *gorm.DB {
	return db.Where("deleted_at IS NULL")
}

// ApplyQuery applies query parameters to GORM DB
func ApplyQuery(params QueryParamsInterface, table interface{}, customSearchFields []string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		// Apply pagination
		db = Paginate(params.GetPageNumber(), params.GetPageSize())(db)

		// Apply search
		if params.GetSearch() != "" {
			if len(customSearchFields) > 0 {
				db = SearchCustomField(params.GetSearch(), customSearchFields)(db)
			} else if table != nil {
				db = Search(params.GetSearch(), table)(db)
			}
		}

		// Apply filters
		for field, value := range params.GetFilters() {
			if value != "" {
				// Convert snake_case to proper field name if needed
				fieldName := strings.ReplaceAll(field, "_", ".")
				db = db.Where(fmt.Sprintf("%s = ?", fieldName), value)
			}
		}

		// Apply ordering
		orderClause := params.GetOrderClause()
		if orderClause != "" {
			db = db.Order(orderClause)
		}

		return db
	}
}

// ApplyQueryWithTableName applies query parameters to GORM DB with explicit table name
func ApplyQueryWithTableName(params QueryParamsInterface, tableName string, customSearchFields []string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		// Apply pagination
		db = Paginate(params.GetPageNumber(), params.GetPageSize())(db)

		// Apply search
		if params.GetSearch() != "" && len(customSearchFields) > 0 {
			query := ""
			values := []interface{}{}
			for i, field := range customSearchFields {
				if field != "" {
					// Add table name prefix if not already present
					if !strings.Contains(field, ".") {
						field = tableName + "." + field
					}
					query += field + " LIKE ?"
					if i < len(customSearchFields)-1 {
						query += " OR "
					}
					values = append(values, "%"+params.GetSearch()+"%")
				}
			}
			if query != "" {
				db = db.Where(query, values...)
			}
		}

		// Apply filters
		for field, value := range params.GetFilters() {
			if value != "" {
				// Add table name prefix if not already present
				if !strings.Contains(field, ".") {
					field = tableName + "." + field
				}
				db = db.Where(fmt.Sprintf("%s = ?", field), value)
			}
		}

		// Apply ordering
		orderClause := params.GetOrderClause()
		if orderClause != "" {
			// Parse order clause to add table prefix if needed
			parts := strings.Split(orderClause, " ")
			if len(parts) >= 2 {
				orderBy := parts[0]
				order := parts[1]
				// Add table name prefix if not already present
				if !strings.Contains(orderBy, ".") {
					orderBy = tableName + "." + orderBy
				}
				db = db.Order(orderBy + " " + order)
			} else {
				db = db.Order(orderClause)
			}
		}

		return db
	}
}
