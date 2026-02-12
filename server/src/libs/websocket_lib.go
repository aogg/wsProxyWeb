// WebSocket工具库
package libs

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"

	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
	"wsProxyWeb/server/src/logic"
	"wsProxyWeb/server/src/types"
)

// WebSocketServer WebSocket服务器结构
type WebSocketServer struct {
	port     string
	clients  map[*websocket.Conn]bool
	clientsMu sync.RWMutex
	ctx      context.Context
	cancel   context.CancelFunc
}


// NewWebSocketServer 创建新的WebSocket服务器
func NewWebSocketServer(port string) *WebSocketServer {
	ctx, cancel := context.WithCancel(context.Background())
	return &WebSocketServer{
		port:    port,
		clients: make(map[*websocket.Conn]bool),
		ctx:     ctx,
		cancel:  cancel,
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
		var msg types.Message
		err := wsjson.Read(ctx, conn, &msg)
		if err != nil {
			log.Printf("读取消息失败: %v", err)
			return
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
				// 尝试添加error字段（需要自定义结构）
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

		// 发送响应
		if err := wsjson.Write(ctx, conn, response); err != nil {
			log.Printf("发送消息失败: %v", err)
			return
		}
	}
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
func (s *WebSocketServer) handleHTTPRequest(msg types.Message) (*types.Message, error) {
	// 调用logic层处理请求
	return getRequestLogic().ProcessRequest(msg.Data)
}

// Broadcast 广播消息给所有客户端
func (s *WebSocketServer) Broadcast(msg types.Message) error {
	s.clientsMu.RLock()
	defer s.clientsMu.RUnlock()

	ctx := context.Background()
	for conn := range s.clients {
		if err := wsjson.Write(ctx, conn, msg); err != nil {
			log.Printf("广播消息失败: %v", err)
		}
	}
	return nil
}

// SendToClient 发送消息给指定客户端
func (s *WebSocketServer) SendToClient(conn *websocket.Conn, msg types.Message) error {
	ctx := context.Background()
	return wsjson.Write(ctx, conn, msg)
}
