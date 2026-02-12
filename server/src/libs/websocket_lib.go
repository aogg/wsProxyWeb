// WebSocket工具库
package libs

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"nhooyr.io/websocket"
	"wsProxyWeb/server/src/logic"
	"wsProxyWeb/server/src/types"
)

// WebSocketServer WebSocket服务器结构
type WebSocketServer struct {
	port        string
	clients     map[*websocket.Conn]bool
	clientsMu   sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
	cryptoLib   *CryptoLib
	compressLib *CompressLib
}


// NewWebSocketServer 创建新的WebSocket服务器
func NewWebSocketServer(port string) *WebSocketServer {
	ctx, cancel := context.WithCancel(context.Background())

	// 加载配置
	config, err := LoadConfig("")
	if err != nil {
		log.Printf("警告: 加载配置失败: %v，使用默认配置", err)
		config = GetConfig()
	}

	// 初始化加密库
	cryptoLib, err := NewCryptoLib(&config.Crypto)
	if err != nil {
		log.Printf("警告: 初始化加密库失败: %v，加密功能将被禁用", err)
		cryptoLib, _ = NewCryptoLib(&CryptoConfig{Enabled: false})
	}

	// 初始化压缩库
	compressLib, err := NewCompressLib(&config.Compress)
	if err != nil {
		log.Printf("警告: 初始化压缩库失败: %v，压缩功能将被禁用", err)
		compressLib, _ = NewCompressLib(&CompressConfig{Enabled: false})
	}

	log.Printf("加密功能: %v (算法: %s)", cryptoLib.IsEnabled(), config.Crypto.Algorithm)
	log.Printf("压缩功能: %v (算法: %s, 级别: %d)", compressLib.IsEnabled(), config.Compress.Algorithm, config.Compress.Level)

	return &WebSocketServer{
		port:        port,
		clients:     make(map[*websocket.Conn]bool),
		ctx:         ctx,
		cancel:      cancel,
		cryptoLib:   cryptoLib,
		compressLib: compressLib,
	}
}

// Start 启动WebSocket服务器
func (s *WebSocketServer) Start() error {
	http.HandleFunc("/ws", s.handleWebSocket)

	addr := ":" + s.port
	log.Printf("WebSocket服务器启动，监听端口: %s", addr)
	log.Printf("WebSocket连接地址: ws://localhost%s/ws", addr)

	return http.ListenAndServe(addr, nil)
}

// Stop 停止WebSocket服务器
func (s *WebSocketServer) Stop() {
	s.cancel()
	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()
	
	// 关闭所有连接
	for conn := range s.clients {
		conn.Close(websocket.StatusNormalClosure, "服务器关闭")
	}
	s.clients = make(map[*websocket.Conn]bool)
}

// handleWebSocket 处理WebSocket连接
func (s *WebSocketServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// 升级HTTP连接为WebSocket连接
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{"*"}, // 允许所有来源，生产环境应该限制
	})
	if err != nil {
		log.Printf("WebSocket连接失败: %v", err)
		return
	}
	defer conn.Close(websocket.StatusInternalError, "连接关闭")

	// 添加客户端
	s.addClient(conn)
	defer s.removeClient(conn)

	log.Printf("新客户端连接，当前连接数: %d", s.getClientCount())

	// 处理消息
	s.handleMessages(conn)
}

// addClient 添加客户端
func (s *WebSocketServer) addClient(conn *websocket.Conn) {
	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()
	s.clients[conn] = true
}

// removeClient 移除客户端
func (s *WebSocketServer) removeClient(conn *websocket.Conn) {
	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()
	delete(s.clients, conn)
	log.Printf("客户端断开连接，当前连接数: %d", len(s.clients))
}

// getClientCount 获取客户端数量
func (s *WebSocketServer) getClientCount() int {
	s.clientsMu.RLock()
	defer s.clientsMu.RUnlock()
	return len(s.clients)
}

// handleMessages 处理客户端消息
func (s *WebSocketServer) handleMessages(conn *websocket.Conn) {
	ctx := conn.CloseRead(s.ctx)

	for {
		// 读取原始消息（二进制数据）
		_, rawData, err := conn.Read(ctx)
		if err != nil {
			log.Printf("读取消息失败: %v", err)
			return
		}

		// 解密 → 解压 → 解析JSON
		msg, err := s.processIncomingMessage(rawData)
		if err != nil {
			log.Printf("处理接收消息失败: %v", err)
			// 发送错误响应
			errorResponse := types.Message{
				ID:   "",
				Type: "error",
				Data: map[string]interface{}{
					"code":    "MESSAGE_PROCESS_ERROR",
					"message": fmt.Sprintf("处理消息失败: %v", err),
				},
			}
			if err := s.sendMessage(ctx, conn, errorResponse); err != nil {
				log.Printf("发送错误响应失败: %v", err)
			}
			continue
		}

		log.Printf("收到消息: ID=%s, Type=%s", msg.ID, msg.Type)

		// 处理不同类型的消息
		var response types.Message
		response.ID = msg.ID

		switch msg.Type {
		case "ping":
			// 心跳消息，回复pong
			response.Type = "pong"
			response.Data = msg.Data
		case "request":
			// HTTP请求消息，调用请求处理逻辑
			responseMsg, err := s.handleHTTPRequest(msg)
			if err != nil {
				// 返回错误响应
				response.Type = "response"
				response.Data = map[string]interface{}{
					"status":       0,
					"statusText":   "Error",
					"headers":      make(map[string]string),
					"body":         "",
					"bodyEncoding": "text",
				}
				log.Printf("处理请求失败: %v", err)
			} else {
				response = *responseMsg
				response.ID = msg.ID // 确保ID一致
			}
		case "close":
			// 关闭消息
			log.Printf("收到关闭消息: %v", msg.Data)
			return
		default:
			// 未知消息类型，返回错误
			response.Type = "error"
			response.Data = map[string]interface{}{
				"code":    "UNKNOWN_MESSAGE_TYPE",
				"message": fmt.Sprintf("未知的消息类型: %s", msg.Type),
				"details": make(map[string]interface{}),
			}
		}

		// 发送响应（压缩 → 加密 → 发送）
		if err := s.sendMessage(ctx, conn, response); err != nil {
			log.Printf("发送消息失败: %v", err)
			return
		}
	}
}

// processIncomingMessage 处理接收到的消息：解密 → 解压 → 解析JSON
func (s *WebSocketServer) processIncomingMessage(rawData []byte) (*types.Message, error) {
	// 步骤1: 解密
	decryptedData, err := s.cryptoLib.Decrypt(rawData)
	if err != nil {
		return nil, fmt.Errorf("解密失败: %v", err)
	}

	// 步骤2: 解压
	decompressedData, err := s.compressLib.Decompress(decryptedData)
	if err != nil {
		return nil, fmt.Errorf("解压失败: %v", err)
	}

	// 步骤3: 解析JSON
	var msg types.Message
	if err := json.Unmarshal(decompressedData, &msg); err != nil {
		return nil, fmt.Errorf("解析JSON失败: %v", err)
	}

	return &msg, nil
}

// sendMessage 发送消息：序列化JSON → 压缩 → 加密 → 发送
func (s *WebSocketServer) sendMessage(ctx context.Context, conn *websocket.Conn, msg types.Message) error {
	// 步骤1: 序列化JSON
	jsonData, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("序列化JSON失败: %v", err)
	}

	// 步骤2: 压缩
	compressedData, err := s.compressLib.Compress(jsonData)
	if err != nil {
		return fmt.Errorf("压缩失败: %v", err)
	}

	// 步骤3: 加密
	encryptedData, err := s.cryptoLib.Encrypt(compressedData)
	if err != nil {
		return fmt.Errorf("加密失败: %v", err)
	}

	// 步骤4: 发送二进制数据
	return conn.Write(ctx, websocket.MessageBinary, encryptedData)
}

// requestLogicInstance 请求处理逻辑实例（单例）
var requestLogicInstance *logic.RequestLogic

// getRequestLogic 获取请求处理逻辑实例
func getRequestLogic() *logic.RequestLogic {
	if requestLogicInstance == nil {
		requestLogicInstance = logic.NewRequestLogic()
	}
	return requestLogicInstance
}

// handleHTTPRequest 处理HTTP请求
func (s *WebSocketServer) handleHTTPRequest(msg *types.Message) (*types.Message, error) {
	// 调用logic层处理请求
	return getRequestLogic().ProcessRequest(msg.Data)
}

// Broadcast 广播消息给所有客户端
func (s *WebSocketServer) Broadcast(msg types.Message) error {
	s.clientsMu.RLock()
	defer s.clientsMu.RUnlock()

	ctx := context.Background()
	for conn := range s.clients {
		if err := s.sendMessage(ctx, conn, msg); err != nil {
			log.Printf("广播消息失败: %v", err)
		}
	}
	return nil
}

// SendToClient 发送消息给指定客户端
func (s *WebSocketServer) SendToClient(conn *websocket.Conn, msg types.Message) error {
	ctx := context.Background()
	return s.sendMessage(ctx, conn, msg)
}
