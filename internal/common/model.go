package common

// SearchRequest 按关键字和过滤条件搜索白名单模型。
type SearchRequest struct {
	Model   string         `json:"model"`
	Page    int            `json:"page"`
	Size    int            `json:"size"`
	Keyword string         `json:"keyword"`
	Filters map[string]any `json:"filters"`
}

// ListRequest 按过滤条件分页列出白名单模型。
type ListRequest struct {
	Model   string         `json:"model"`
	Page    int            `json:"page"`
	Size    int            `json:"size"`
	Filters map[string]any `json:"filters"`
}

// OneRequest 按过滤条件返回一条记录。
type OneRequest struct {
	Model   string         `json:"model"`
	Filters map[string]any `json:"filters"`
}
