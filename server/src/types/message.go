// WebSocket消息和HTTP请求/响应数据结构
package types

// Message WebSocket消息结构
type Message struct {
	ID   string      `json:"id"`
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// RequestData 请求数据（从WebSocket消息中解析）
type RequestData struct {
	URL          string            `json:"url"`
	Method       string            `json:"method"`
	Headers      map[string]string `json:"headers"`
	Body         string            `json:"body"`
	BodyEncoding string            `json:"bodyEncoding"`
}

// ResponseData 响应数据（封装为WebSocket消息格式）
type ResponseData struct {
	Status       int               `json:"status"`
	StatusText   string            `json:"statusText"`
	Headers      map[string]string `json:"headers"`
	Body         string            `json:"body"`
	BodyEncoding string            `json:"bodyEncoding"`
	Chunked      bool              `json:"chunked,omitempty"`      // 是否分块传输
	Chunks       []string          `json:"chunks,omitempty"`       // 分块数据
	TotalSize    int               `json:"totalSize,omitempty"`    // 总大小
	ChunkIndex   int               `json:"chunkIndex,omitempty"`   // 当前块索引（用于客户端重组）
	ChunkCount   int               `json:"chunkCount,omitempty"`   // 总块数
}

// HTTPRequestData HTTP请求数据
type HTTPRequestData struct {
	URL          string            `json:"url"`
	Method       string            `json:"method"`
	Headers      map[string]string `json:"headers"`
	Body         string            `json:"body"`
	BodyEncoding string            `json:"bodyEncoding"` // "text" | "base64"
}

// HTTPResponseData HTTP响应数据
type HTTPResponseData struct {
	Status       int               `json:"status"`
	StatusText   string            `json:"statusText"`
	Headers      map[string]string `json:"headers"`
	Body         string            `json:"body"`
	BodyEncoding string            `json:"bodyEncoding"` // "text" | "base64"
	Chunked      bool              `json:"chunked,omitempty"`    // 是否分块传输
	Chunks       []string          `json:"chunks,omitempty"`     // 分块数据
	TotalSize    int               `json:"totalSize,omitempty"`  // 总大小
}

// RequestLogInfo 请求日志信息（用于记录请求详情）
type RequestLogInfo struct {
	URL          string // 请求URL
	Method       string // 请求方法
	ReqBody      string // 请求body
	RespStatus   int    // 响应状态码
	RespBody     string // 响应body
	RespBodySize int    // 响应body大小(bytes)
}
