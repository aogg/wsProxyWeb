// HTTP请求处理逻辑
package logic

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"wsProxyWeb/server/src/types"
)

// RequestLogic HTTP请求处理逻辑类
type RequestLogic struct {
	// 可以添加配置、统计等字段
}

// NewRequestLogic 创建新的请求处理逻辑实例
func NewRequestLogic() *RequestLogic {
	return &RequestLogic{}
}

// ProcessRequest 处理HTTP请求
// 接收WebSocket消息数据，解析请求，执行HTTP请求，返回响应消息
func (rl *RequestLogic) ProcessRequest(msgData interface{}) (*types.Message, error) {
	// 解析请求数据
	reqData, err := rl.parseRequestData(msgData)
	if err != nil {
		return nil, fmt.Errorf("解析请求数据失败: %v", err)
	}

	// 验证请求数据
	if err := rl.validateRequest(reqData); err != nil {
		return nil, fmt.Errorf("请求验证失败: %v", err)
	}

	// 转换为HTTP库需要的格式
	httpReqData := types.HTTPRequestData{
		URL:          reqData.URL,
		Method:       reqData.Method,
		Headers:      reqData.Headers,
		Body:         reqData.Body,
		BodyEncoding: reqData.BodyEncoding,
	}

	// 执行HTTP请求
	log.Printf("执行HTTP请求: %s %s", reqData.Method, reqData.URL)
	httpResp, err := executeHTTPRequest(httpReqData)
	if err != nil {
		// 返回错误响应
		return rl.buildErrorResponse(err), fmt.Errorf("请求执行失败: %v", err)
	}

	// 构建成功响应
	responseMsg := rl.buildSuccessResponse(httpResp, reqData)
	log.Printf("HTTP请求完成: %s %s, 状态码: %d", reqData.Method, reqData.URL, httpResp.Status)

	return responseMsg, nil
}

// parseRequestData 解析请求数据
func (rl *RequestLogic) parseRequestData(msgData interface{}) (*types.RequestData, error) {
	// 将消息数据转换为JSON，再解析为RequestData
	jsonData, err := json.Marshal(msgData)
	if err != nil {
		return nil, fmt.Errorf("序列化请求数据失败: %v", err)
	}

	var reqData types.RequestData
	if err := json.Unmarshal(jsonData, &reqData); err != nil {
		return nil, fmt.Errorf("解析请求数据失败: %v", err)
	}

	return &reqData, nil
}

// validateRequest 验证请求数据
func (rl *RequestLogic) validateRequest(reqData *types.RequestData) error {
	// 验证必需字段
	if reqData.URL == "" {
		return fmt.Errorf("URL不能为空")
	}

	// 设置默认值
	if reqData.Method == "" {
		reqData.Method = "GET" // 默认为GET
	}
	if reqData.BodyEncoding == "" {
		reqData.BodyEncoding = "text" // 默认为text
	}

	// 验证HTTP方法
	validMethods := map[string]bool{
		"GET":     true,
		"POST":    true,
		"PUT":     true,
		"DELETE":  true,
		"PATCH":   true,
		"HEAD":    true,
		"OPTIONS": true,
	}
	if !validMethods[reqData.Method] {
		return fmt.Errorf("不支持的HTTP方法: %s", reqData.Method)
	}

	return nil
}

// buildSuccessResponse 构建成功响应
func (rl *RequestLogic) buildSuccessResponse(httpResp *types.HTTPResponseData, reqData *types.RequestData) *types.Message {
	// 构建响应数据
	responseData := types.ResponseData{
		Status:       httpResp.Status,
		StatusText:   httpResp.StatusText,
		Headers:      httpResp.Headers,
		Body:         httpResp.Body,
		BodyEncoding: httpResp.BodyEncoding,
	}

	// 封装为WebSocket消息格式
	return &types.Message{
		Type: "response",
		Data: map[string]interface{}{
			"status":       responseData.Status,
			"statusText":   responseData.StatusText,
			"headers":      responseData.Headers,
			"body":         responseData.Body,
			"bodyEncoding": responseData.BodyEncoding,
		},
	}
}

// buildErrorResponse 构建错误响应
func (rl *RequestLogic) buildErrorResponse(err error) *types.Message {
	return &types.Message{
		Type: "response",
		Data: map[string]interface{}{
			"status":       0,
			"statusText":   "Error",
			"headers":      make(map[string]string),
			"body":         "",
			"bodyEncoding": "text",
		},
	}
}

// executeHTTPRequest 执行HTTP请求（内部函数，避免循环导入）
func executeHTTPRequest(reqData types.HTTPRequestData) (*types.HTTPResponseData, error) {
	// 创建HTTP客户端，设置超时时间30秒
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// 准备请求体
	var bodyReader io.Reader
	if reqData.Body != "" {
		if reqData.BodyEncoding == "base64" {
			// Base64解码
			decoded, err := base64.StdEncoding.DecodeString(reqData.Body)
			if err != nil {
				return nil, fmt.Errorf("Base64解码失败: %v", err)
			}
			bodyReader = bytes.NewReader(decoded)
		} else {
			// 文本内容
			bodyReader = strings.NewReader(reqData.Body)
		}
	}

	// 创建HTTP请求
	req, err := http.NewRequest(reqData.Method, reqData.URL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	// 设置请求头
	for key, value := range reqData.Headers {
		// 跳过一些浏览器自动设置的请求头，避免冲突
		if strings.ToLower(key) == "host" {
			continue
		}
		req.Header.Set(key, value)
	}

	// 执行请求
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求执行失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应体
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应体失败: %v", err)
	}

	// 判断响应体是否需要Base64编码
	// 如果Content-Type是二进制类型（如图片、视频等），使用Base64编码
	bodyEncoding := "text"
	bodyStr := string(bodyBytes)
	contentType := resp.Header.Get("Content-Type")
	
	// 判断是否为二进制内容
	if isBinaryContent(contentType) || len(bodyBytes) > 0 && !isTextContent(bodyBytes) {
		bodyEncoding = "base64"
		bodyStr = base64.StdEncoding.EncodeToString(bodyBytes)
	}

	// 构建响应头映射
	headers := make(map[string]string)
	for key, values := range resp.Header {
		// 合并多个值
		headers[key] = strings.Join(values, ", ")
	}

	// 构建响应数据
	responseData := &types.HTTPResponseData{
		Status:       resp.StatusCode,
		StatusText:   resp.Status,
		Headers:      headers,
		Body:         bodyStr,
		BodyEncoding: bodyEncoding,
	}

	return responseData, nil
}

// isBinaryContent 判断Content-Type是否为二进制类型
func isBinaryContent(contentType string) bool {
	if contentType == "" {
		return false
	}
	
	contentType = strings.ToLower(contentType)
	
	// 常见的二进制类型
	binaryTypes := []string{
		"image/",
		"video/",
		"audio/",
		"application/octet-stream",
		"application/pdf",
		"application/zip",
		"application/x-",
	}
	
	for _, binaryType := range binaryTypes {
		if strings.HasPrefix(contentType, binaryType) {
			return true
		}
	}
	
	return false
}

// isTextContent 判断字节内容是否为文本
func isTextContent(data []byte) bool {
	if len(data) == 0 {
		return true
	}
	
	// 检查是否包含NULL字节（二进制数据的特征）
	for _, b := range data {
		if b == 0 {
			return false
		}
	}
	
	// 检查是否大部分是可打印字符
	printableCount := 0
	for _, b := range data {
		if (b >= 32 && b <= 126) || b == 9 || b == 10 || b == 13 {
			printableCount++
		}
	}
	
	// 如果80%以上是可打印字符，认为是文本
	return float64(printableCount)/float64(len(data)) >= 0.8
}
