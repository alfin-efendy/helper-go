# Chi Middleware (Interceptors) Implementation

This implementation provides Gin-like middleware functionality for Chi router, including automatic error handling and success response formatting.

## Features

### 1. Enhanced Context (`Ctx`)
The `Ctx` struct has been enhanced to support:
- Error collection (`AddError`)
- Data storage (`SetData`)
- Pagination (`SetPage`)
- JSON response (`JSON`)
- Response status tracking (`IsWritten`)

### 2. Response Structures
- `Response`: Standard API response format
- `ValidationError`: Field validation error details
- `PageResponse`: Pagination information

### 3. Middleware Functions

#### `ResponseMiddleware`
Combines both error and success response handling. This is the main middleware you'll use.

#### `ErrorResponseMiddleware`
Handles various types of errors:
- **Validation Errors**: Automatically formats validation errors from `go-playground/validator`
- **General Errors**: Handles common errors like "record not found", "EOF", etc.
- **Custom Errors**: You can add your own error handling logic

#### `SuccessResponseMiddleware`
Automatically formats success responses when no errors occur.

## Usage

### 1. Basic Setup
```go
r := chi.NewRouter()
r.Use(middleware.Logger)
r.Use(middleware.Recoverer)
r.Use(NewCtx)
r.Use(ResponseMiddleware) // Add the response middleware
```

### 2. Handler Examples

#### Success Response
```go
func GetUsersHandler(c *Ctx) {
    users := []User{...} // Your data
    
    c.SetData(users)
    c.SetPage(PageResponse{
        Page:       1,
        Limit:      10,
        Total:      100,
        TotalPages: 10,
    })
    // Middleware will automatically return success response
}
```

#### Validation Error
```go
type CreateUserRequest struct {
    Name  string `json:"name" validate:"required"`
    Email string `json:"email" validate:"required,email"`
}

func CreateUserHandler(c *Ctx) {
    var req CreateUserRequest
    if err := c.Bind(&req); err != nil {
        c.AddError(err) // Middleware will handle validation errors
        return
    }
    
    // Process the request...
    c.SetData(map[string]interface{}{
        "message": "User created successfully",
        "user":    req,
    })
}
```

#### General Error
```go
func GetUserHandler(c *Ctx) {
    user, err := userService.GetByID(id)
    if err != nil {
        c.AddError(err) // Middleware will handle the error
        return
    }
    
    c.SetData(user)
}
```

### 3. Response Formats

#### Success Response
```json
{
    "message": "Success",
    "data": {...},
    "page": {
        "page": 1,
        "limit": 10,
        "total": 100,
        "totalPages": 10
    }
}
```

#### Validation Error Response
```json
{
    "message": "Validation failed",
    "errors": [
        {
            "field": "name",
            "message": "name is required"
        },
        {
            "field": "email",
            "message": "email is not valid email"
        }
    ]
}
```

#### General Error Response
```json
{
    "message": "Data not found"
}
```

## Testing the Implementation

You can test the middleware with the provided example routes:

1. **Success Response**: `GET /example/success`
2. **Validation Error**: `POST /example/validation` (with invalid JSON)
3. **General Error**: `GET /example/error`

## Key Differences from Gin

1. **No `c.Next()`**: Chi middleware executes in a wrapper pattern, so we handle responses after the handler completes.
2. **Context Enhancement**: We enhanced the `Ctx` struct to store errors, data, and pagination info.
3. **Automatic Response**: The middleware automatically sends responses based on the context state.

## Migration from Gin

To migrate from Gin to this Chi implementation:

1. Replace `c.Next()` with direct handler logic
2. Use `c.AddError(err)` instead of `c.Error(err)`
3. Use `c.SetData(data)` instead of `c.Set("data", data)`
4. Use `c.SetPage(page)` instead of `c.Set("page", page)`
5. Remove manual `c.JSON()` calls for standard responses (let middleware handle them)

This implementation provides the same convenience and automatic response handling as your Gin middleware while working seamlessly with Chi router.
