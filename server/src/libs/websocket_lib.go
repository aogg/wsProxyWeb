// WebSocket工具库
package libs

import (
	"context"
	"log"
	"net/http"
	"sync"

	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

// WebSocketServer WebSocket服务器结构
type WebSocketServer struct {
	port     string
	clients  map[*websocket.Conn]bool
	clientsMu sync.RWMutex
	ctx      context.Context
	cancel   context.CancelFunc
}

// Message 消息结构
type Message struct {
	ID   string      `json:"id"`
	Type string      `json:"type"`
	Data interface{} `json:"data"`
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
		var msg Message
		err := wsjson.Read(ctx, conn, &msg)
		if err != nil {
			log.Printf("读取消息失败: %v", err)
			return
		}

		log.Printf("收到消息: ID=%s, Type=%s", msg.ID, msg.Type)

		// 回显消息（后续会改为处理请求）
		response := Message{
			ID:   msg.ID,
			Type: "echo",
			Data: map[string]string{
				"message": "收到消息",
			},
		}

		if err := wsjson.Write(ctx, conn, response); err != nil {
			log.Printf("发送消息失败: %v", err)
			return
		}
	}
}

// Broadcast 广播消息给所有客户端
func (s *WebSocketServer) Broadcast(msg Message) error {
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
func (s *WebSocketServer) SendToClient(conn *websocket.Conn, msg Message) error {
	ctx := context.Background()
	return wsjson.Write(ctx, conn, msg)
}
