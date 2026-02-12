// HTTP请求工具库
package libs

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"wsProxyWeb/server/src/types"
)

// ExecuteHTTPRequest 执行HTTP请求
func ExecuteHTTPRequest(reqData types.HTTPRequestData) (*types.HTTPResponseData, error) {
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
