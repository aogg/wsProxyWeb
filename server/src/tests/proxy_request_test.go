package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"nhooyr.io/websocket"
	"wsProxyWeb/server/src/libs"
	"wsProxyWeb/server/src/types"
)

// encodeMessageWS 编码WebSocket消息
func encodeMessageWS(msg types.Message, cryptoLib *libs.CryptoLib, compressLib *libs.CompressLib) ([]byte, error) {
	jsonData, _ := json.Marshal(msg)
	compressedData, _ := compressLib.Compress(jsonData)
	encryptedData, _ := cryptoLib.Encrypt(compressedData)
	return encryptedData, nil
}

// decodeMessageWS 解码WebSocket消息
func decodeMessageWS(data []byte, cryptoLib *libs.CryptoLib, compressLib *libs.CompressLib) (*types.Message, error) {
	decryptedData, _ := cryptoLib.Decrypt(data)
	decompressedData, _ := compressLib.Decompress(decryptedData)
	var msg types.Message
	json.Unmarshal(decompressedData, &msg)
	return &msg, nil
}

func createTestCryptoLibWS(t *testing.T) *libs.CryptoLib {
	config := &libs.CryptoConfig{Enabled: false, Algorithm: "aes256gcm"}
	cryptoLib, _ := libs.NewCryptoLib(config)
	return cryptoLib
}

func createTestCompressLibWS(t *testing.T) *libs.CompressLib {
	config := &libs.CompressConfig{Enabled: false, Level: 6, Algorithm: "gzip"}
	compressLib, _ := libs.NewCompressLib(config)
	return compressLib
}

func TestProxyRequest_WebSocket_Baidu(t *testing.T) {
	// 设置日志级别为 debug，以便能看到 Debug 日志
	libs.GetLogLib().SetLevel("debug")

	libs.GetConfig()
	cryptoLib := createTestCryptoLibWS(t)
	compressLib := createTestCompressLibWS(t)

	wsServer := libs.NewWebSocketServer("0")
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("创建监听失败: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	wsURL := fmt.Sprintf("ws://127.0.0.1:%d/ws", port)

	httpServer := &http.Server{
		Handler: http.HandlerFunc(wsServer.HandleWebSocket),
	}
	go httpServer.Serve(listener)
	defer httpServer.Close()

	time.Sleep(100 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, wsURL, &websocket.DialOptions{
		CompressionMode: websocket.CompressionDisabled,
	})
	if err != nil {
		t.Fatalf("WebSocket连接失败: %v", err)
	}
	// 注意：不在此处使用 defer 关闭连接，而是在测试结束时手动关闭

	// 发送代理请求
	requestMsg := types.Message{
		ID:   "test-proxy-baidu",
		Type: "request",
		Data: map[string]interface{}{
			"url":     "https://www.baidu.com",
			"method":  "GET",
			"headers": map[string]string{},
			"body":    "",
		},
	}

	encodedData, err := encodeMessageWS(requestMsg, cryptoLib, compressLib)
	if err != nil {
		t.Fatalf("编码消息失败: %v", err)
	}
	t.Logf("发送请求消息, 长度: %d", len(encodedData))

	err = conn.Write(ctx, websocket.MessageBinary, encodedData)
	if err != nil {
		t.Fatalf("发送消息失败: %v", err)
	}
	t.Log("消息已发送，等待响应...")
	time.Sleep(50 * time.Millisecond) // 等待消息传输

	respType, respData, err := conn.Read(ctx)
	t.Logf("收到响应: type=%v, len=%d, err=%v", respType, len(respData), err)

	if err != nil {
		t.Fatalf("读取响应失败: %v", err)
	}

	respMsg, err := decodeMessageWS(respData, cryptoLib, compressLib)
	if err != nil {
		t.Fatalf("解码响应失败: %v", err)
	}
	t.Logf("响应类型: %s", respMsg.Type)

	if respMsg.Type != "response" {
		t.Errorf("期望响应类型为response，实际: %s", respMsg.Type)
	} else {
		// 验证响应数据
		if respDataMap, ok := respMsg.Data.(map[string]interface{}); ok {
			status, _ := respDataMap["status"].(float64)
			body, _ := respDataMap["body"].(string)
			t.Logf("代理请求成功: 状态码=%d, 响应大小=%d", int(status), len(body))
		}
	}

	// 关闭连接
	conn.Close(websocket.StatusNormalClosure, "测试结束")

	wsServer.Stop()
}
