package restapi

var (
	dataStr string = "data"
	pageStr string = "page"
)

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
	Search   string `form:"Search"`
}

type PageResponse struct {
	TotalPage   int   `json:"totalPage"`
	TotalRecord int64 `json:"totalRecord"`
}
