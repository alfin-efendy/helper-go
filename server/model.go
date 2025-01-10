package server

type Responses struct {
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   interface{} `json:"error,omitempty"`
}

type CommonError struct {
	Errors map[string]interface{}
}

type PageRequest struct {
	Page     int    `form:"page"`
	PageSize int    `form:"pageSize"`
	Search   string `form:"search"`
}

type PageResponse struct {
	TotalPage   int   `json:"totalPage"`
	TotalRecord int64 `json:"totalRecord"`
}

type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Response is a struct for standard response format
type Response struct {
	Message string        `json:"message"`
	Errors  interface{}   `json:"errors,omitempty"`
	Data    interface{}   `json:"data,omitempty"`
	Page    *PageResponse `json:"page,omitempty"`
}
