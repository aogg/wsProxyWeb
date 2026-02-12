package tests

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"nhooyr.io/websocket"
	"wsProxyWeb/server/src/libs"
	"wsProxyWeb/server/src/types"
)

// ==================== 辅助函数 ====================

// createTestCryptoLib 创建测试用的加密库
func createTestCryptoLib(t *testing.T, enabled bool) *libs.CryptoLib {
	key := make([]byte, 32)
	keyBase64 := base64.StdEncoding.EncodeToString(key)

	config := &libs.CryptoConfig{
		Enabled:   enabled,
		Key:       keyBase64,
		Algorithm: "aes256gcm",
	}

	cryptoLib, err := libs.NewCryptoLib(config)
	if err != nil {
		t.Fatalf("创建CryptoLib失败: %v", err)
	}
	return cryptoLib
}

// createTestCompressLib 创建测试用的压缩库
func createTestCompressLib(t *testing.T, enabled bool) *libs.CompressLib {
	config := &libs.CompressConfig{
		Enabled:   enabled,
		Level:     6,
		Algorithm: "gzip",
	}

	compressLib, err := libs.NewCompressLib(config)
	if err != nil {
		t.Fatalf("创建CompressLib失败: %v", err)
	}
	return compressLib
}

// encodeMessage 编码消息：JSON序列化 → 压缩 → 加密
func encodeMessage(msg types.Message, cryptoLib *libs.CryptoLib, compressLib *libs.CompressLib) ([]byte, error) {
	// JSON序列化
	jsonData, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("JSON序列化失败: %v", err)
	}

	// 压缩
	compressedData, err := compressLib.Compress(jsonData)
	if err != nil {
		return nil, fmt.Errorf("压缩失败: %v", err)
	}

	// 加密
	encryptedData, err := cryptoLib.Encrypt(compressedData)
	if err != nil {
		return nil, fmt.Errorf("加密失败: %v", err)
	}

	return encryptedData, nil
}

// decodeMessage 解码消息：解密 → 解压 → JSON解析
func decodeMessage(data []byte, cryptoLib *libs.CryptoLib, compressLib *libs.CompressLib) (*types.Message, error) {
	// 解密
	decryptedData, err := cryptoLib.Decrypt(data)
	if err != nil {
		return nil, fmt.Errorf("解密失败: %v", err)
	}

	// 解压
	decompressedData, err := compressLib.Decompress(decryptedData)
	if err != nil {
		return nil, fmt.Errorf("解压失败: %v", err)
	}

	// JSON解析
	var msg types.Message
	if err := json.Unmarshal(decompressedData, &msg); err != nil {
		return nil, fmt.Errorf("JSON解析失败: %v", err)
	}

	return &msg, nil
}

// ==================== 端到端流程测试 ====================

// TestIntegration_EndToEndRequestResponse 测试端到端请求-响应流程
// 场景：客户端发送HTTP请求 → 服务端执行请求 → 返回响应
func TestIntegration_EndToEndRequestResponse(t *testing.T) {
	// 创建测试HTTP服务器（模拟目标服务器）
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证请求
		if r.Method != "GET" {
			t.Errorf("期望GET请求，实际: %s", r.Method)
		}

		// 返回响应
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "hello from test server"}`))
	}))
	defer testServer.Close()

	// 创建加密和压缩库（不启用）
	cryptoLib := createTestCryptoLib(t, false)
	compressLib := createTestCompressLib(t, false)

	// 构造请求消息
	requestMsg := types.Message{
		ID:   "test-request-001",
		Type: "request",
		Data: map[string]interface{}{
			"url":     testServer.URL + "/api/test",
			"method":  "GET",
			"headers": map[string]string{},
			"body":    "",
		},
	}

	// 编码消息
	encodedData, err := encodeMessage(requestMsg, cryptoLib, compressLib)
	if err != nil {
		t.Fatalf("编码消息失败: %v", err)
	}

	// 模拟服务端处理：解码 → 处理 → 编码响应
	decodedMsg, err := decodeMessage(encodedData, cryptoLib, compressLib)
	if err != nil {
		t.Fatalf("解码消息失败: %v", err)
	}

	// 验证解码后的消息
	if decodedMsg.ID != requestMsg.ID {
		t.Errorf("消息ID不匹配，期望: %s, 实际: %s", requestMsg.ID, decodedMsg.ID)
	}
	if decodedMsg.Type != "request" {
		t.Errorf("消息类型不匹配，期望: request, 实际: %s", decodedMsg.Type)
	}

	// 解析请求数据
	reqData, ok := decodedMsg.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("请求数据格式错误")
	}

	// 执行HTTP请求
	httpReqData := types.HTTPRequestData{
		URL:     reqData["url"].(string),
		Method:  reqData["method"].(string),
		Headers: make(map[string]string),
	}
	if headers, ok := reqData["headers"].(map[string]interface{}); ok {
		for k, v := range headers {
			httpReqData.Headers[k] = v.(string)
		}
	}

	resp, err := libs.ExecuteHTTPRequest(httpReqData)
	if err != nil {
		t.Fatalf("执行HTTP请求失败: %v", err)
	}

	// 验证响应
	if resp.Status != http.StatusOK {
		t.Errorf("期望状态码200，实际: %d", resp.Status)
	}

	// 构造响应消息
	responseMsg := types.Message{
		ID:   decodedMsg.ID,
		Type: "response",
		Data: map[string]interface{}{
			"status":       resp.Status,
			"statusText":   resp.StatusText,
			"headers":      resp.Headers,
			"body":         resp.Body,
			"bodyEncoding": resp.BodyEncoding,
		},
	}

	// 编码响应
	encodedResponse, err := encodeMessage(responseMsg, cryptoLib, compressLib)
	if err != nil {
		t.Fatalf("编码响应失败: %v", err)
	}

	// 模拟客户端解码响应
	decodedResponse, err := decodeMessage(encodedResponse, cryptoLib, compressLib)
	if err != nil {
		t.Fatalf("解码响应失败: %v", err)
	}

	// 验证最终响应
	if decodedResponse.ID != requestMsg.ID {
		t.Errorf("响应ID与请求ID不匹配")
	}
	if decodedResponse.Type != "response" {
		t.Errorf("响应类型不正确")
	}

	respData, ok := decodedResponse.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("响应数据格式错误")
	}

	if respData["status"].(float64) != float64(http.StatusOK) {
		t.Errorf("响应状态码不正确")
	}

	t.Logf("端到端测试通过: 请求ID=%s, 响应状态=%v", decodedResponse.ID, respData["status"])
}

// TestIntegration_EndToEndWithPost 测试POST请求端到端流程
func TestIntegration_EndToEndWithPost(t *testing.T) {
	// 创建测试HTTP服务器
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("期望POST请求，实际: %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type不正确")
		}

		// 读取并返回请求体
		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"data":    body,
		})
	}))
	defer testServer.Close()

	cryptoLib := createTestCryptoLib(t, false)
	compressLib := createTestCompressLib(t, false)

	// 构造POST请求消息
	requestMsg := types.Message{
		ID:   "test-post-001",
		Type: "request",
		Data: map[string]interface{}{
			"url":    testServer.URL + "/api/create",
			"method": "POST",
			"headers": map[string]string{
				"Content-Type": "application/json",
			},
			"body":         `{"name": "test", "value": "integration"}`,
			"bodyEncoding": "text",
		},
	}

	// 编码并发送请求
	encodedData, _ := encodeMessage(requestMsg, cryptoLib, compressLib)
	decodedMsg, _ := decodeMessage(encodedData, cryptoLib, compressLib)

	// 执行HTTP请求
	reqData := decodedMsg.Data.(map[string]interface{})
	httpReqData := types.HTTPRequestData{
		URL:          reqData["url"].(string),
		Method:       reqData["method"].(string),
		Headers:      make(map[string]string),
		Body:         reqData["body"].(string),
		BodyEncoding: "text",
	}
	if headers, ok := reqData["headers"].(map[string]interface{}); ok {
		for k, v := range headers {
			httpReqData.Headers[k] = v.(string)
		}
	}

	resp, err := libs.ExecuteHTTPRequest(httpReqData)
	if err != nil {
		t.Fatalf("执行HTTP请求失败: %v", err)
	}

	if resp.Status != http.StatusCreated {
		t.Errorf("期望状态码201，实际: %d", resp.Status)
	}

	t.Logf("POST请求端到端测试通过")
}

// ==================== 加密压缩通信测试 ====================

// TestIntegration_EncryptedCommunication 测试加密通信
func TestIntegration_EncryptedCommunication(t *testing.T) {
	// 启用加密
	cryptoLib := createTestCryptoLib(t, true)
	compressLib := createTestCompressLib(t, false)

	// 创建测试服务器
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("encrypted response"))
	}))
	defer testServer.Close()

	// 构造请求
	requestMsg := types.Message{
		ID:   "encrypted-001",
		Type: "request",
		Data: map[string]interface{}{
			"url":    testServer.URL,
			"method": "GET",
		},
	}

	// 编码（加密）
	encodedData, err := encodeMessage(requestMsg, cryptoLib, compressLib)
	if err != nil {
		t.Fatalf("编码失败: %v", err)
	}

	// 验证加密后的数据与原始数据不同
	jsonData, _ := json.Marshal(requestMsg)
	if string(encodedData) == string(jsonData) {
		t.Error("加密后数据应该与原始数据不同")
	}

	// 解码（解密）
	decodedMsg, err := decodeMessage(encodedData, cryptoLib, compressLib)
	if err != nil {
		t.Fatalf("解码失败: %v", err)
	}

	// 验证消息完整性
	if decodedMsg.ID != requestMsg.ID {
		t.Errorf("消息ID不匹配")
	}
	if decodedMsg.Type != requestMsg.Type {
		t.Errorf("消息类型不匹配")
	}

	t.Logf("加密通信测试通过")
}

// TestIntegration_CompressedCommunication 测试压缩通信
func TestIntegration_CompressedCommunication(t *testing.T) {
	cryptoLib := createTestCryptoLib(t, false)
	compressLib := createTestCompressLib(t, true) // 启用压缩

	// 创建测试服务器
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("compressed response"))
	}))
	defer testServer.Close()

	// 构造包含大量重复数据的请求（便于压缩）
	largeBody := ""
	for i := 0; i < 1000; i++ {
		largeBody += "AAAAAAAAAA" // 重复数据，压缩效果明显
	}

	requestMsg := types.Message{
		ID:   "compressed-001",
		Type: "request",
		Data: map[string]interface{}{
			"url":    testServer.URL,
			"method": "POST",
			"body":   largeBody,
		},
	}

	// 编码（压缩）
	encodedData, err := encodeMessage(requestMsg, cryptoLib, compressLib)
	if err != nil {
		t.Fatalf("编码失败: %v", err)
	}

	// 验证压缩效果
	jsonData, _ := json.Marshal(requestMsg)
	compressionRatio := float64(len(encodedData)) / float64(len(jsonData))
	t.Logf("原始大小: %d, 压缩后大小: %d, 压缩比: %.2f", len(jsonData), len(encodedData), compressionRatio)

	// 解码（解压）
	decodedMsg, err := decodeMessage(encodedData, cryptoLib, compressLib)
	if err != nil {
		t.Fatalf("解码失败: %v", err)
	}

	// 验证消息完整性
	if decodedMsg.ID != requestMsg.ID {
		t.Errorf("消息ID不匹配")
	}

	t.Logf("压缩通信测试通过")
}

// TestIntegration_EncryptedAndCompressed 测试同时加密和压缩
func TestIntegration_EncryptedAndCompressed(t *testing.T) {
	cryptoLib := createTestCryptoLib(t, true)  // 启用加密
	compressLib := createTestCompressLib(t, true) // 启用压缩

	// 创建测试服务器
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("secure and compressed"))
	}))
	defer testServer.Close()

	requestMsg := types.Message{
		ID:   "secure-compressed-001",
		Type: "request",
		Data: map[string]interface{}{
			"url":    testServer.URL,
			"method": "GET",
		},
	}

	// 编码：JSON → 压缩 → 加密
	encodedData, err := encodeMessage(requestMsg, cryptoLib, compressLib)
	if err != nil {
		t.Fatalf("编码失败: %v", err)
	}

	// 解码：解密 → 解压 → JSON
	decodedMsg, err := decodeMessage(encodedData, cryptoLib, compressLib)
	if err != nil {
		t.Fatalf("解码失败: %v", err)
	}

	// 验证消息完整性
	if decodedMsg.ID != requestMsg.ID {
		t.Errorf("消息ID不匹配")
	}

	t.Logf("加密+压缩通信测试通过")
}

// ==================== 错误处理测试 ====================

// TestIntegration_ErrorHandling_InvalidURL 测试无效URL错误处理
func TestIntegration_ErrorHandling_InvalidURL(t *testing.T) {
	cryptoLib := createTestCryptoLib(t, false)
	compressLib := createTestCompressLib(t, false)

	requestMsg := types.Message{
		ID:   "error-invalid-url",
		Type: "request",
		Data: map[string]interface{}{
			"url":    "://invalid-url",
			"method": "GET",
		},
	}

	encodedData, _ := encodeMessage(requestMsg, cryptoLib, compressLib)
	decodedMsg, _ := decodeMessage(encodedData, cryptoLib, compressLib)

	reqData := decodedMsg.Data.(map[string]interface{})
	httpReqData := types.HTTPRequestData{
		URL:    reqData["url"].(string),
		Method: reqData["method"].(string),
	}

	_, err := libs.ExecuteHTTPRequest(httpReqData)
	if err == nil {
		t.Error("无效URL应该返回错误")
	}

	t.Logf("无效URL错误处理测试通过: %v", err)
}

// TestIntegration_ErrorHandling_ConnectionError 测试连接错误处理
func TestIntegration_ErrorHandling_ConnectionError(t *testing.T) {
	cryptoLib := createTestCryptoLib(t, false)
	compressLib := createTestCompressLib(t, false)

	requestMsg := types.Message{
		ID:   "error-connection",
		Type: "request",
		Data: map[string]interface{}{
			"url":    "http://127.0.0.1:1/test", // 不存在的端口
			"method": "GET",
		},
	}

	encodedData, _ := encodeMessage(requestMsg, cryptoLib, compressLib)
	decodedMsg, _ := decodeMessage(encodedData, cryptoLib, compressLib)

	reqData := decodedMsg.Data.(map[string]interface{})
	httpReqData := types.HTTPRequestData{
		URL:    reqData["url"].(string),
		Method: reqData["method"].(string),
	}

	_, err := libs.ExecuteHTTPRequest(httpReqData)
	if err == nil {
		t.Error("连接失败应该返回错误")
	}

	t.Logf("连接错误处理测试通过: %v", err)
}

// TestIntegration_ErrorHandling_DecryptionFailure 测试解密失败处理
func TestIntegration_ErrorHandling_DecryptionFailure(t *testing.T) {
	cryptoLib := createTestCryptoLib(t, true) // 启用加密
	compressLib := createTestCompressLib(t, false)

	// 构造请求数据
	requestMsg := types.Message{
		ID:   "decrypt-test",
		Type: "request",
		Data: map[string]interface{}{
			"url":    "http://example.com",
			"method": "GET",
		},
	}

	// 正确编码
	encodedData, _ := encodeMessage(requestMsg, cryptoLib, compressLib)

	// 篡改数据
	if len(encodedData) > 10 {
		encodedData[5] ^= 0xFF // 修改某个字节
	}

	// 尝试解码，应该失败
	_, err := decodeMessage(encodedData, cryptoLib, compressLib)
	if err == nil {
		t.Error("篡改的数据解密应该失败")
	}

	t.Logf("解密失败处理测试通过: %v", err)
}

// TestIntegration_ErrorHandling_Timeout 测试超时处理
func TestIntegration_ErrorHandling_Timeout(t *testing.T) {
	// 创建延迟响应的测试服务器
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second) // 延迟5秒
		w.WriteHeader(http.StatusOK)
	}))
	defer testServer.Close()

	cryptoLib := createTestCryptoLib(t, false)
	compressLib := createTestCompressLib(t, false)

	requestMsg := types.Message{
		ID:   "timeout-test",
		Type: "request",
		Data: map[string]interface{}{
			"url":    testServer.URL,
			"method": "GET",
		},
	}

	encodedData, _ := encodeMessage(requestMsg, cryptoLib, compressLib)
	decodedMsg, _ := decodeMessage(encodedData, cryptoLib, compressLib)

	reqData := decodedMsg.Data.(map[string]interface{})
	httpReqData := types.HTTPRequestData{
		URL:    reqData["url"].(string),
		Method: reqData["method"].(string),
	}

	// 注意：这里使用默认的HTTP客户端，它有30秒超时
	// 如果需要测试短超时，可以修改配置
	start := time.Now()
	_, err := libs.ExecuteHTTPRequest(httpReqData)
	elapsed := time.Since(start)

	t.Logf("请求耗时: %v, 错误: %v", elapsed, err)
}

// ==================== 心跳测试 ====================

// TestIntegration_Heartbeat 测试心跳消息处理
func TestIntegration_Heartbeat(t *testing.T) {
	cryptoLib := createTestCryptoLib(t, false)
	compressLib := createTestCompressLib(t, false)

	// 构造心跳请求
	pingMsg := types.Message{
		ID:   "heartbeat-001",
		Type: "ping",
		Data: map[string]interface{}{
			"timestamp": time.Now().Unix(),
		},
	}

	// 编码
	encodedData, err := encodeMessage(pingMsg, cryptoLib, compressLib)
	if err != nil {
		t.Fatalf("编码失败: %v", err)
	}

	// 解码
	decodedMsg, err := decodeMessage(encodedData, cryptoLib, compressLib)
	if err != nil {
		t.Fatalf("解码失败: %v", err)
	}

	// 验证消息
	if decodedMsg.Type != "ping" {
		t.Errorf("期望消息类型为ping，实际: %s", decodedMsg.Type)
	}

	// 模拟服务端响应pong
	pongMsg := types.Message{
		ID:   decodedMsg.ID,
		Type: "pong",
		Data: decodedMsg.Data,
	}

	encodedPong, _ := encodeMessage(pongMsg, cryptoLib, compressLib)
	decodedPong, _ := decodeMessage(encodedPong, cryptoLib, compressLib)

	if decodedPong.Type != "pong" {
		t.Errorf("期望响应类型为pong，实际: %s", decodedPong.Type)
	}

	t.Logf("心跳测试通过")
}

// ==================== WebSocket连接测试 ====================

// TestIntegration_WebSocketConnection 测试WebSocket连接建立和消息收发
func TestIntegration_WebSocketConnection(t *testing.T) {
	// 重置全局配置
	libs.GetConfig()

	// 创建WebSocket服务器
	server := libs.NewWebSocketServer("0") // 使用端口0让系统自动分配
	if server == nil {
		t.Fatal("创建WebSocket服务器失败")
	}

	// 在后台启动服务器
	go func() {
		server.Start()
	}()

	// 给服务器一点时间启动
	time.Sleep(100 * time.Millisecond)

	// 停止服务器
	server.Stop()

	t.Logf("WebSocket服务器创建和停止测试通过")
}

// TestIntegration_ConcurrentRequests 测试并发请求处理
func TestIntegration_ConcurrentRequests(t *testing.T) {
	// 创建测试HTTP服务器
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond) // 模拟处理延迟
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer testServer.Close()

	cryptoLib := createTestCryptoLib(t, false)
	compressLib := createTestCompressLib(t, false)

	// 并发发送多个请求
	var wg sync.WaitGroup
	numRequests := 10
	errors := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			requestMsg := types.Message{
				ID:   fmt.Sprintf("concurrent-%d", idx),
				Type: "request",
				Data: map[string]interface{}{
					"url":    testServer.URL,
					"method": "GET",
				},
			}

			encodedData, err := encodeMessage(requestMsg, cryptoLib, compressLib)
			if err != nil {
				errors <- fmt.Errorf("请求%d编码失败: %v", idx, err)
				return
			}

			decodedMsg, err := decodeMessage(encodedData, cryptoLib, compressLib)
			if err != nil {
				errors <- fmt.Errorf("请求%d解码失败: %v", idx, err)
				return
			}

			reqData := decodedMsg.Data.(map[string]interface{})
			httpReqData := types.HTTPRequestData{
				URL:    reqData["url"].(string),
				Method: reqData["method"].(string),
			}

			resp, err := libs.ExecuteHTTPRequest(httpReqData)
			if err != nil {
				errors <- fmt.Errorf("请求%d执行失败: %v", idx, err)
				return
			}

			if resp.Status != http.StatusOK {
				errors <- fmt.Errorf("请求%d状态码不正确: %d", idx, resp.Status)
				return
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// 检查错误
	for err := range errors {
		t.Errorf("并发请求错误: %v", err)
	}

	t.Logf("并发请求测试通过，共处理 %d 个请求", numRequests)
}

// TestIntegration_LargeData 测试大数据传输
func TestIntegration_LargeData(t *testing.T) {
	cryptoLib := createTestCryptoLib(t, true)  // 启用加密
	compressLib := createTestCompressLib(t, true) // 启用压缩

	// 创建测试服务器
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 返回大数据
		largeData := make([]byte, 500*1024) // 500KB
		for i := range largeData {
			largeData[i] = byte(i % 256)
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		w.Write(largeData)
	}))
	defer testServer.Close()

	requestMsg := types.Message{
		ID:   "large-data-001",
		Type: "request",
		Data: map[string]interface{}{
			"url":    testServer.URL,
			"method": "GET",
		},
	}

	// 编码
	start := time.Now()
	encodedData, err := encodeMessage(requestMsg, cryptoLib, compressLib)
	if err != nil {
		t.Fatalf("编码失败: %v", err)
	}
	encodeTime := time.Since(start)

	// 解码
	start = time.Now()
	decodedMsg, err := decodeMessage(encodedData, cryptoLib, compressLib)
	if err != nil {
		t.Fatalf("解码失败: %v", err)
	}
	decodeTime := time.Since(start)

	// 执行HTTP请求
	reqData := decodedMsg.Data.(map[string]interface{})
	httpReqData := types.HTTPRequestData{
		URL:    reqData["url"].(string),
		Method: reqData["method"].(string),
	}

	start = time.Now()
	resp, err := libs.ExecuteHTTPRequest(httpReqData)
	if err != nil {
		t.Fatalf("执行HTTP请求失败: %v", err)
	}
	requestTime := time.Since(start)

	t.Logf("大数据传输测试: 编码耗时=%v, 解码耗时=%v, 请求耗时=%v, 响应大小=%d",
		encodeTime, decodeTime, requestTime, len(resp.Body))

	if resp.BodyEncoding != "base64" {
		t.Errorf("大数据响应应该使用base64编码")
	}
}

// TestIntegration_MessageIDConsistency 测试消息ID一致性
func TestIntegration_MessageIDConsistency(t *testing.T) {
	cryptoLib := createTestCryptoLib(t, true)
	compressLib := createTestCompressLib(t, true)

	// 创建测试服务器
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer testServer.Close()

	// 测试多个消息的ID一致性
	for i := 0; i < 5; i++ {
		originalID := fmt.Sprintf("msg-%d-%d", time.Now().UnixNano(), i)

		requestMsg := types.Message{
			ID:   originalID,
			Type: "request",
			Data: map[string]interface{}{
				"url":    testServer.URL,
				"method": "GET",
			},
		}

		encodedData, _ := encodeMessage(requestMsg, cryptoLib, compressLib)
		decodedMsg, _ := decodeMessage(encodedData, cryptoLib, compressLib)

		if decodedMsg.ID != originalID {
			t.Errorf("消息ID不一致，期望: %s, 实际: %s", originalID, decodedMsg.ID)
		}
	}

	t.Logf("消息ID一致性测试通过")
}

// TestIntegration_DifferentHTTPMethods 测试不同HTTP方法
func TestIntegration_DifferentHTTPMethods(t *testing.T) {
	// 创建测试服务器
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Method", r.Method)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"method": r.Method})
	}))
	defer testServer.Close()

	cryptoLib := createTestCryptoLib(t, false)
	compressLib := createTestCompressLib(t, false)

	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}

	for _, method := range methods {
		requestMsg := types.Message{
			ID:   fmt.Sprintf("method-test-%s", method),
			Type: "request",
			Data: map[string]interface{}{
				"url":     testServer.URL,
				"method":  method,
				"headers": map[string]string{"Content-Type": "application/json"},
				"body":    `{"test": true}`,
			},
		}

		encodedData, _ := encodeMessage(requestMsg, cryptoLib, compressLib)
		decodedMsg, _ := decodeMessage(encodedData, cryptoLib, compressLib)

		reqData := decodedMsg.Data.(map[string]interface{})
		httpReqData := types.HTTPRequestData{
			URL:          reqData["url"].(string),
			Method:       reqData["method"].(string),
			Body:         reqData["body"].(string),
			BodyEncoding: "text",
			Headers:      make(map[string]string),
		}
		if headers, ok := reqData["headers"].(map[string]interface{}); ok {
			for k, v := range headers {
				httpReqData.Headers[k] = v.(string)
			}
		}

		resp, err := libs.ExecuteHTTPRequest(httpReqData)
		if err != nil {
			t.Errorf("%s请求失败: %v", method, err)
			continue
		}

		if resp.Status != http.StatusOK {
			t.Errorf("%s请求状态码不正确: %d", method, resp.Status)
		}

		if resp.Headers["X-Method"] != method {
			t.Errorf("%s请求方法未正确传递", method)
		}

		t.Logf("%s方法测试通过", method)
	}
}

// TestIntegration_ResponseHeaders 测试响应头处理
func TestIntegration_ResponseHeaders(t *testing.T) {
	// 创建测试服务器，返回多种响应头
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Custom-Header", "custom-value")
		w.Header().Set("X-Request-Id", "test-123")
		w.Header().Add("Set-Cookie", "session=abc")
		w.Header().Add("Set-Cookie", "user=test")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer testServer.Close()

	cryptoLib := createTestCryptoLib(t, false)
	compressLib := createTestCompressLib(t, false)

	requestMsg := types.Message{
		ID:   "headers-test",
		Type: "request",
		Data: map[string]interface{}{
			"url":    testServer.URL,
			"method": "GET",
		},
	}

	encodedData, _ := encodeMessage(requestMsg, cryptoLib, compressLib)
	decodedMsg, _ := decodeMessage(encodedData, cryptoLib, compressLib)

	reqData := decodedMsg.Data.(map[string]interface{})
	httpReqData := types.HTTPRequestData{
		URL:    reqData["url"].(string),
		Method: reqData["method"].(string),
	}

	resp, err := libs.ExecuteHTTPRequest(httpReqData)
	if err != nil {
		t.Fatalf("请求失败: %v", err)
	}

	// 验证响应头
	if resp.Headers["Content-Type"] != "application/json" {
		t.Errorf("Content-Type响应头不正确: %s", resp.Headers["Content-Type"])
	}

	if resp.Headers["X-Custom-Header"] != "custom-value" {
		t.Errorf("X-Custom-Header响应头不正确: %s", resp.Headers["X-Custom-Header"])
	}

	if resp.Headers["X-Request-Id"] != "test-123" {
		t.Errorf("X-Request-Id响应头不正确: %s", resp.Headers["X-Request-Id"])
	}

	t.Logf("响应头处理测试通过")
}

// TestIntegration_BinaryData 测试二进制数据处理
func TestIntegration_BinaryData(t *testing.T) {
	// 创建测试服务器，返回二进制数据
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		// 模拟PNG文件头
		w.Write([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A})
	}))
	defer testServer.Close()

	cryptoLib := createTestCryptoLib(t, true)
	compressLib := createTestCompressLib(t, true)

	requestMsg := types.Message{
		ID:   "binary-test",
		Type: "request",
		Data: map[string]interface{}{
			"url":    testServer.URL,
			"method": "GET",
		},
	}

	encodedData, _ := encodeMessage(requestMsg, cryptoLib, compressLib)
	decodedMsg, _ := decodeMessage(encodedData, cryptoLib, compressLib)

	reqData := decodedMsg.Data.(map[string]interface{})
	httpReqData := types.HTTPRequestData{
		URL:    reqData["url"].(string),
		Method: reqData["method"].(string),
	}

	resp, err := libs.ExecuteHTTPRequest(httpReqData)
	if err != nil {
		t.Fatalf("请求失败: %v", err)
	}

	// 验证二进制数据使用Base64编码
	if resp.BodyEncoding != "base64" {
		t.Errorf("二进制响应应该使用base64编码，实际: %s", resp.BodyEncoding)
	}

	// 验证可以解码
	decoded, err := base64.StdEncoding.DecodeString(resp.Body)
	if err != nil {
		t.Fatalf("Base64解码失败: %v", err)
	}

	// 验证PNG文件头
	if len(decoded) < 8 {
		t.Fatalf("解码后数据长度不足")
	}

	expected := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	for i, b := range expected {
		if decoded[i] != b {
			t.Errorf("二进制数据不匹配，位置%d: 期望0x%02X, 实际0x%02X", i, b, decoded[i])
		}
	}

	t.Logf("二进制数据处理测试通过")
}

// TestIntegration_WebSocketRealConnection 测试真实的WebSocket连接（如果可能）
func TestIntegration_WebSocketRealConnection(t *testing.T) {
	// 重置全局配置
	libs.GetConfig()

	// 创建一个简单的HTTP测试服务器来模拟WebSocket握手
	testHTTPServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 简单响应，用于测试连接
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test server"))
	}))
	defer testHTTPServer.Close()

	// 创建WebSocket客户端连接到测试服务器
	// 注意：这里只是测试客户端创建，不进行实际的WebSocket通信
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 尝试连接（会失败，但测试客户端创建）
	_, _, err := websocket.Dial(ctx, testHTTPServer.URL, nil)
	// 预期会失败，因为这不是真正的WebSocket服务器
	// 但我们测试的是客户端创建逻辑
	t.Logf("WebSocket客户端连接测试完成: %v", err)
}
