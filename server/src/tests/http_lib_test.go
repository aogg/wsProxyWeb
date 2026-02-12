package tests

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"wsProxyWeb/server/src/libs"
	"wsProxyWeb/server/src/types"
)

// 初始化测试配置
func init() {
	// 重置全局配置
	libs.GetConfig()
}

// 测试GET请求
func TestExecuteHTTPRequest_Get(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证请求方法
		if r.Method != "GET" {
			t.Errorf("期望GET请求，实际: %s", r.Method)
		}

		// 验证请求头
		if r.Header.Get("X-Custom-Header") != "test-value" {
			t.Errorf("请求头未正确传递")
		}

		// 返回JSON响应
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "success"})
	}))
	defer server.Close()

	// 准备请求数据
	reqData := types.HTTPRequestData{
		URL:    server.URL + "/test",
		Method: "GET",
		Headers: map[string]string{
			"X-Custom-Header": "test-value",
		},
	}

	// 执行请求
	resp, err := libs.ExecuteHTTPRequest(reqData)
	if err != nil {
		t.Fatalf("请求执行失败: %v", err)
	}

	// 验证响应
	if resp.Status != http.StatusOK {
		t.Errorf("期望状态码200，实际: %d", resp.Status)
	}

	if resp.BodyEncoding != "text" {
		t.Errorf("期望BodyEncoding为text，实际: %s", resp.BodyEncoding)
	}

	// 验证响应体
	var result map[string]string
	if err := json.Unmarshal([]byte(resp.Body), &result); err != nil {
		t.Fatalf("解析响应体失败: %v", err)
	}

	if result["message"] != "success" {
		t.Errorf("响应体内容不正确")
	}
}

// 测试POST请求
func TestExecuteHTTPRequest_Post(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证请求方法
		if r.Method != "POST" {
			t.Errorf("期望POST请求，实际: %s", r.Method)
		}

		// 验证Content-Type
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type未正确设置")
		}

		// 读取请求体
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("解析请求体失败: %v", err)
		}

		// 返回响应
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"result": "created", "name": body["name"]})
	}))
	defer server.Close()

	// 准备请求数据
	reqBody := `{"name":"test"}`
	reqData := types.HTTPRequestData{
		URL:    server.URL + "/create",
		Method: "POST",
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body:         reqBody,
		BodyEncoding: "text",
	}

	// 执行请求
	resp, err := libs.ExecuteHTTPRequest(reqData)
	if err != nil {
		t.Fatalf("请求执行失败: %v", err)
	}

	// 验证响应
	if resp.Status != http.StatusCreated {
		t.Errorf("期望状态码201，实际: %d", resp.Status)
	}

	// 验证响应体
	var result map[string]string
	if err := json.Unmarshal([]byte(resp.Body), &result); err != nil {
		t.Fatalf("解析响应体失败: %v", err)
	}

	if result["name"] != "test" {
		t.Errorf("响应体内容不正确，期望name=test，实际: %s", result["name"])
	}
}

// 测试PUT请求
func TestExecuteHTTPRequest_Put(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("期望PUT请求，实际: %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("updated"))
	}))
	defer server.Close()

	reqData := types.HTTPRequestData{
		URL:    server.URL + "/update",
		Method: "PUT",
		Body:   `{"id":1}`,
	}

	resp, err := libs.ExecuteHTTPRequest(reqData)
	if err != nil {
		t.Fatalf("请求执行失败: %v", err)
	}

	if resp.Status != http.StatusOK {
		t.Errorf("期望状态码200，实际: %d", resp.Status)
	}
}

// 测试DELETE请求
func TestExecuteHTTPRequest_Delete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("期望DELETE请求，实际: %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	reqData := types.HTTPRequestData{
		URL:    server.URL + "/delete",
		Method: "DELETE",
	}

	resp, err := libs.ExecuteHTTPRequest(reqData)
	if err != nil {
		t.Fatalf("请求执行失败: %v", err)
	}

	if resp.Status != http.StatusNoContent {
		t.Errorf("期望状态码204，实际: %d", resp.Status)
	}
}

// 测试Base64编码的请求体
func TestExecuteHTTPRequest_Base64Body(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 读取请求体
		buf := make([]byte, 1024)
		n, _ := r.Body.Read(buf)
		receivedBody := buf[:n]

		// 验证接收到的数据
		expectedBody := []byte("binary data")
		if string(receivedBody) != string(expectedBody) {
			t.Errorf("请求体不匹配，期望: %s, 实际: %s", expectedBody, receivedBody)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// 使用Base64编码的请求体
	body := []byte("binary data")
	reqData := types.HTTPRequestData{
		URL:          server.URL + "/binary",
		Method:       "POST",
		Body:         base64.StdEncoding.EncodeToString(body),
		BodyEncoding: "base64",
	}

	resp, err := libs.ExecuteHTTPRequest(reqData)
	if err != nil {
		t.Fatalf("请求执行失败: %v", err)
	}

	if resp.Status != http.StatusOK {
		t.Errorf("期望状态码200，实际: %d", resp.Status)
	}
}

// 测试二进制响应
func TestExecuteHTTPRequest_BinaryResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		// 模拟PNG文件头
		w.Write([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A})
	}))
	defer server.Close()

	reqData := types.HTTPRequestData{
		URL:    server.URL + "/image.png",
		Method: "GET",
	}

	resp, err := libs.ExecuteHTTPRequest(reqData)
	if err != nil {
		t.Fatalf("请求执行失败: %v", err)
	}

	// 验证二进制内容使用Base64编码
	if resp.BodyEncoding != "base64" {
		t.Errorf("期望二进制响应使用Base64编码，实际: %s", resp.BodyEncoding)
	}

	// 验证可以解码
	_, err = base64.StdEncoding.DecodeString(resp.Body)
	if err != nil {
		t.Errorf("Base64解码失败: %v", err)
	}
}

// 测试响应头
func TestExecuteHTTPRequest_ResponseHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom-Response", "response-value")
		w.Header().Set("X-Multiple", "value1")
		w.Header().Add("X-Multiple", "value2")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	reqData := types.HTTPRequestData{
		URL:    server.URL + "/headers",
		Method: "GET",
	}

	resp, err := libs.ExecuteHTTPRequest(reqData)
	if err != nil {
		t.Fatalf("请求执行失败: %v", err)
	}

	// 验证响应头
	if resp.Headers["X-Custom-Response"] != "response-value" {
		t.Errorf("响应头未正确获取")
	}

	// 验证多值响应头
	if resp.Headers["X-Multiple"] == "" {
		t.Error("多值响应头未正确获取")
	}
}

// 测试错误状态码
func TestExecuteHTTPRequest_ErrorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("not found"))
	}))
	defer server.Close()

	reqData := types.HTTPRequestData{
		URL:    server.URL + "/notfound",
		Method: "GET",
	}

	resp, err := libs.ExecuteHTTPRequest(reqData)
	if err != nil {
		t.Fatalf("请求执行失败: %v", err)
	}

	if resp.Status != http.StatusNotFound {
		t.Errorf("期望状态码404，实际: %d", resp.Status)
	}
}

// 测试服务器错误
func TestExecuteHTTPRequest_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal server error"))
	}))
	defer server.Close()

	reqData := types.HTTPRequestData{
		URL:    server.URL + "/error",
		Method: "GET",
	}

	resp, err := libs.ExecuteHTTPRequest(reqData)
	if err != nil {
		t.Fatalf("请求执行失败: %v", err)
	}

	if resp.Status != http.StatusInternalServerError {
		t.Errorf("期望状态码500，实际: %d", resp.Status)
	}
}

// 测试无效URL
func TestExecuteHTTPRequest_InvalidURL(t *testing.T) {
	reqData := types.HTTPRequestData{
		URL:    "://invalid-url",
		Method: "GET",
	}

	_, err := libs.ExecuteHTTPRequest(reqData)
	if err == nil {
		t.Error("无效URL应该返回错误")
	}
}

// 测试连接超时（使用一个不存在的地址）
func TestExecuteHTTPRequest_ConnectionError(t *testing.T) {
	reqData := types.HTTPRequestData{
		URL:    "http://127.0.0.1:1/test", // 使用一个不会响应的端口
		Method: "GET",
	}

	_, err := libs.ExecuteHTTPRequest(reqData)
	if err == nil {
		t.Error("连接失败应该返回错误")
	}
}

// 测试重定向
func TestExecuteHTTPRequest_Redirect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redirect" {
			http.Redirect(w, r, "/final", http.StatusFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("final destination"))
	}))
	defer server.Close()

	reqData := types.HTTPRequestData{
		URL:    server.URL + "/redirect",
		Method: "GET",
	}

	resp, err := libs.ExecuteHTTPRequest(reqData)
	if err != nil {
		t.Fatalf("请求执行失败: %v", err)
	}

	// Go的默认客户端会自动跟随重定向
	if resp.Status != http.StatusOK {
		t.Errorf("期望最终状态码200，实际: %d", resp.Status)
	}
}

// 测试空请求体
func TestExecuteHTTPRequest_EmptyBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	reqData := types.HTTPRequestData{
		URL:    server.URL + "/empty",
		Method: "POST",
		Body:   "",
	}

	resp, err := libs.ExecuteHTTPRequest(reqData)
	if err != nil {
		t.Fatalf("请求执行失败: %v", err)
	}

	if resp.Status != http.StatusOK {
		t.Errorf("期望状态码200，实际: %d", resp.Status)
	}
}

// 测试Host头被跳过
func TestExecuteHTTPRequest_HostHeaderSkipped(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Host头应该是服务器的地址，而不是我们设置的值
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	reqData := types.HTTPRequestData{
		URL:    server.URL + "/test",
		Method: "GET",
		Headers: map[string]string{
			"Host": "malicious-host.com",
		},
	}

	resp, err := libs.ExecuteHTTPRequest(reqData)
	if err != nil {
		t.Fatalf("请求执行失败: %v", err)
	}

	if resp.Status != http.StatusOK {
		t.Errorf("期望状态码200，实际: %d", resp.Status)
	}
}

// 测试分块传输
func TestExecuteHTTPRequestWithChunk_LargeResponse(t *testing.T) {
	// 创建一个大响应
	largeData := make([]byte, 2*1024*1024) // 2MB
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		w.Write(largeData)
	}))
	defer server.Close()

	reqData := types.HTTPRequestData{
		URL:    server.URL + "/large",
		Method: "GET",
	}

	// 使用分块传输，块大小为64KB
	resp, err := libs.ExecuteHTTPRequestWithChunk(reqData, 64*1024)
	if err != nil {
		t.Fatalf("请求执行失败: %v", err)
	}

	// 验证分块传输
	if !resp.Chunked {
		t.Error("大响应应该启用分块传输")
	}

	if len(resp.Chunks) == 0 {
		t.Error("应该有分块数据")
	}

	if resp.TotalSize != len(largeData) {
		t.Errorf("总大小不匹配，期望: %d, 实际: %d", len(largeData), resp.TotalSize)
	}

	// 验证可以重组数据
	var重组后的数据[]byte
	for _, chunk := range resp.Chunks {
		decoded, err := base64.StdEncoding.DecodeString(chunk)
		if err != nil {
			t.Fatalf("解码块失败: %v", err)
		}
		重组后的数据 = append(重组后的数据, decoded...)
	}

	if len(重组后的数据) != len(largeData) {
		t.Errorf("重组后数据大小不匹配")
	}
}

// 测试小响应不使用分块
func TestExecuteHTTPRequestWithChunk_SmallResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("small response"))
	}))
	defer server.Close()

	reqData := types.HTTPRequestData{
		URL:    server.URL + "/small",
		Method: "GET",
	}

	// 使用分块传输
	resp, err := libs.ExecuteHTTPRequestWithChunk(reqData, 64*1024)
	if err != nil {
		t.Fatalf("请求执行失败: %v", err)
	}

	// 小响应不应该使用分块
	if resp.Chunked {
		t.Error("小响应不应该启用分块传输")
	}
}
