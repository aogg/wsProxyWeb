// HTTP请求工具库
package libs

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"wsProxyWeb/server/src/types"
)

var (
	httpClientOnce sync.Once
	httpClient     *http.Client
)

// 大响应阈值：超过此大小的响应体使用分块传输
const largeResponseThreshold = 1024 * 1024 // 1MB

// getHTTPClient 获取全局复用的HTTP客户端（连接池复用，提升性能）
func getHTTPClient() *http.Client {
	httpClientOnce.Do(func() {
		cfg := GetConfig()

		timeout := time.Duration(cfg.HTTP.TimeoutSeconds) * time.Second
		if timeout <= 0 {
			timeout = 30 * time.Second
		}

		// 配置代理
		var proxyFunc func(*http.Request) (*url.URL, error)
		if cfg.HTTP.ProxyEnabled && cfg.HTTP.ProxyURL != "" {
			proxyURL, err := url.Parse(cfg.HTTP.ProxyURL)
			if err != nil {
				Warn("解析代理URL失败: %v，使用环境变量代理", err)
				proxyFunc = http.ProxyFromEnvironment
			} else {
				proxyFunc = http.ProxyURL(proxyURL)
				Info("HTTP客户端使用正向代理: %s", cfg.HTTP.ProxyURL)
			}
		} else {
			proxyFunc = http.ProxyFromEnvironment
		}

		transport := &http.Transport{
			Proxy: proxyFunc,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          maxInt(cfg.HTTP.MaxIdleConns, 0),
			MaxIdleConnsPerHost:   maxInt(cfg.HTTP.MaxIdleConnsPerHost, 0),
			IdleConnTimeout:       time.Duration(maxInt(cfg.HTTP.IdleConnTimeoutSeconds, 0)) * time.Second,
			TLSHandshakeTimeout:   time.Duration(maxInt(cfg.HTTP.TLSHandshakeTimeoutSeconds, 0)) * time.Second,
			ExpectContinueTimeout: time.Duration(maxInt(cfg.HTTP.ExpectContinueTimeoutSeconds, 0)) * time.Second,
		}

		httpClient = &http.Client{
			Timeout:   timeout,
			Transport: transport,
		}
	})

	return httpClient
}

func maxInt(v int, min int) int {
	if v < min {
		return min
	}
	return v
}

// ExecuteHTTPRequest 执行HTTP请求
func ExecuteHTTPRequest(reqData types.HTTPRequestData) (*types.HTTPResponseData, error) {
	return ExecuteHTTPRequestWithChunk(reqData, 0)
}

// ExecuteHTTPRequestWithChunk 执行HTTP请求，支持分块传输
// chunkSize > 0 时，大响应体会被分块编码
func ExecuteHTTPRequestWithChunk(reqData types.HTTPRequestData, chunkSize int) (*types.HTTPResponseData, error) {
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
	startTime := time.Now()
	resp, err := getHTTPClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求执行失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应体
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应体失败: %v", err)
	}

	// 记录请求耗时
	elapsed := time.Since(startTime)
	Info("HTTP请求完成: %s %s, 状态码: %d, 响应大小: %d bytes, 耗时: %v",
		reqData.Method, reqData.URL, resp.StatusCode, len(bodyBytes), elapsed)

	// 判断响应体是否需要Base64编码
	// 如果Content-Type是二进制类型（如图片、视频等），使用Base64编码
	bodyEncoding := "text"
	bodyStr := string(bodyBytes)
	contentType := resp.Header.Get("Content-Type")
	
	// 判断是否为二进制内容
	isBinary := isBinaryContent(contentType) || len(bodyBytes) > 0 && !isTextContent(bodyBytes)
	if isBinary {
		bodyEncoding = "base64"
		bodyStr = base64.StdEncoding.EncodeToString(bodyBytes)
	}

	// 判断是否需要分块传输
	chunked := false
	var chunks []string
	if chunkSize > 0 && len(bodyBytes) > largeResponseThreshold {
		chunked = true
		chunks = splitIntoChunks(bodyBytes, chunkSize, isBinary)
		Debug("大响应体分块传输: 总大小=%d, 块数=%d, 块大小=%d", len(bodyBytes), len(chunks), chunkSize)
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
		Chunked:      chunked,
		Chunks:       chunks,
		TotalSize:    len(bodyBytes),
	}

	return responseData, nil
}

// splitIntoChunks 将数据分割成块
func splitIntoChunks(data []byte, chunkSize int, isBinary bool) []string {
	var chunks []string
	totalChunks := (len(data) + chunkSize - 1) / chunkSize

	for i := 0; i < totalChunks; i++ {
		start := i * chunkSize
		end := start + chunkSize
		if end > len(data) {
			end = len(data)
		}

		chunk := data[start:end]
		if isBinary {
			chunks = append(chunks, base64.StdEncoding.EncodeToString(chunk))
		} else {
			chunks = append(chunks, string(chunk))
		}
	}

	return chunks
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
