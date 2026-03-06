// WebSocket工具库
package libs

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"nhooyr.io/websocket"
	"wsProxyWeb/server/src/logic"
	"wsProxyWeb/server/src/types"
)

// clientInfo 客户端连接信息（含认证状态）
type clientInfo struct {
	authenticated bool
	username      string
	role          string
	isSuperAdmin  bool
	isAdmin       bool
}

// WebSocketServer WebSocket服务器结构
type WebSocketServer struct {
	host           string
	port           string
	clients        map[*websocket.Conn]*clientInfo
	clientsMu      sync.RWMutex
	ctx            context.Context
	cancel         context.CancelFunc
	cryptoLib      *CryptoLib
	compressLib    *CompressLib
	securityLogic  *logic.SecurityLogic
	performanceLib *PerformanceLib
	authLogic      *logic.AuthLogic
	authEnabled    bool
}

// NewWebSocketServer 创建新的WebSocket服务器
func NewWebSocketServer(port string) *WebSocketServer {
	ctx, cancel := context.WithCancel(context.Background())

	// 加载配置
	config, err := LoadConfig("")
	if err != nil {
		Warn("加载配置失败: %v，使用默认配置", err)
		config = GetConfig()
	}

	// 初始化加密库
	cryptoLib, err := NewCryptoLib(&config.Crypto)
	if err != nil {
		Warn("初始化加密库失败: %v，加密功能将被禁用", err)
		cryptoLib, _ = NewCryptoLib(&CryptoConfig{Enabled: false})
	}

	// 初始化压缩库
	compressLib, err := NewCompressLib(&config.Compress)
	if err != nil {
		Warn("初始化压缩库失败: %v，压缩功能将被禁用", err)
		compressLib, _ = NewCompressLib(&CompressConfig{Enabled: false})
	}

	Info("加密功能: %v (算法: %s)", cryptoLib.IsEnabled(), config.Crypto.Algorithm)
	Info("压缩功能: %v (算法: %s, 级别: %d)", compressLib.IsEnabled(), config.Compress.Algorithm, config.Compress.Level)

	// 初始化安全控制逻辑（默认启用，但规则为空时基本不影响）
	securityLogic := logic.NewSecurityLogic(logic.SecurityLogicConfig{
		Enabled:             config.Security.Enabled,
		MaxConnections:      config.Security.MaxConnections,
		MaxMessageBytes:     config.Security.MaxMessageBytes,
		MaxRequestBodyBytes: config.Security.MaxRequestBodyBytes,
		RateLimitPerSecond:  config.Security.RateLimitPerSecond,
		RateBurst:           config.Security.RateBurst,
		AllowIPs:            config.Security.AllowIPs,
		DenyIPs:             config.Security.DenyIPs,
		AllowDomains:        config.Security.AllowDomains,
		DenyDomains:         config.Security.DenyDomains,
	})
	Info("安全控制: %v (最大连接:%d, 最大消息:%dB, 最大请求体:%dB, 限流:%g/s, burst:%d)",
		securityLogic.IsEnabled(),
		config.Security.MaxConnections,
		config.Security.MaxMessageBytes,
		config.Security.MaxRequestBodyBytes,
		config.Security.RateLimitPerSecond,
		config.Security.RateBurst,
	)

	// 初始化性能优化库
	performanceLib := NewPerformanceLib(&config.Performance)
	Info("性能优化: workerPool=%d, chunkSize=%d, metrics=%v",
		config.Performance.WorkerPoolSize, config.Performance.ChunkSize, config.Performance.EnableMetrics)

	// 初始化认证逻辑
	var authLogic *logic.AuthLogic
	authEnabled := config.Auth.Enabled
	if authEnabled {
		authLogic = logic.GetAuthLogic(config.Auth.UserDataDir, config.Auth.AdminUsername)
		if err := authLogic.InitAdmin(config.Auth.AdminUsername, config.Auth.AdminPassword); err != nil {
			Warn("初始化管理员账号失败: %v", err)
		}
		Info("认证功能: 已启用 (管理员: %s)", config.Auth.AdminUsername)
	} else {
		Info("认证功能: 已禁用")
	}

	return &WebSocketServer{
		host:           config.Server.Host,
		port:           port,
		clients:        make(map[*websocket.Conn]*clientInfo),
		ctx:            ctx,
		cancel:         cancel,
		cryptoLib:      cryptoLib,
		compressLib:    compressLib,
		securityLogic:  securityLogic,
		performanceLib: performanceLib,
		authLogic:      authLogic,
		authEnabled:    authEnabled,
	}
}

// Start 启动WebSocket服务器
func (s *WebSocketServer) Start() error {
	http.HandleFunc("/ws", s.handleWebSocket)

	addr := s.host + ":" + s.port
	Info("WebSocket服务器启动，监听地址: %s", addr)
	Info("WebSocket连接地址: ws://%s/ws", addr)

	return http.ListenAndServe(addr, nil)
}

// Stop 停止WebSocket服务器
func (s *WebSocketServer) Stop() {
	s.cancel()

	// 停止性能优化库
	if s.performanceLib != nil {
		s.performanceLib.Stop()
	}

	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()

	// 关闭所有连接
	for conn := range s.clients {
		conn.Close(websocket.StatusNormalClosure, "服务器关闭")
	}
	s.clients = make(map[*websocket.Conn]*clientInfo)
}

// HandleWebSocket 处理WebSocket连接（导出的HTTP处理函数）
func (s *WebSocketServer) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	s.handleWebSocket(w, r)
}

// handleWebSocket 处理WebSocket连接
func (s *WebSocketServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[DEBUG] handleWebSocket 开始处理连接\n")
	// 升级HTTP连接为WebSocket连接
	fmt.Printf("[DEBUG] 开始Accept\n")
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns:  []string{"*"},                 // 允许所有来源，生产环境应该限制
		CompressionMode: websocket.CompressionDisabled, // 禁用压缩，避免 RSV bits 错误
	})
	if err != nil {
		Error("WebSocket连接失败: %v", err)
		fmt.Printf("[DEBUG] Accept失败: %v\n", err)
		return
	}
	fmt.Printf("[DEBUG] Accept成功\n")
	defer conn.Close(websocket.StatusInternalError, "连接关闭")

	clientIP := getClientIP(r)
	// 安全控制：连接数限制、IP白黑名单
	if err := s.securityLogic.CheckNewConnection(clientIP, s.getClientCount()); err != nil {
		Warn("拒绝连接: ip=%s, err=%v", clientIP, err)
		conn.Close(websocket.StatusPolicyViolation, "连接被拒绝: "+err.Error())
		return
	}

	// 添加客户端
	s.addClient(conn)
	defer s.removeClient(conn)

	// 性能指标：增加连接计数
	if s.performanceLib != nil {
		s.performanceLib.IncConnection()
		defer s.performanceLib.DecConnection()
	}

	Info("新客户端连接: ip=%s, 当前连接数: %d", clientIP, s.getClientCount())

	// 处理消息
	s.handleMessages(conn, clientIP)
}

// addClient 添加客户端
func (s *WebSocketServer) addClient(conn *websocket.Conn) {
	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()
	s.clients[conn] = &clientInfo{}
}

// removeClient 移除客户端
func (s *WebSocketServer) removeClient(conn *websocket.Conn) {
	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()
	delete(s.clients, conn)
	Info("客户端断开连接，当前连接数: %d", len(s.clients))
}

// getClientInfo 获取客户端认证信息
func (s *WebSocketServer) getClientInfo(conn *websocket.Conn) *clientInfo {
	s.clientsMu.RLock()
	defer s.clientsMu.RUnlock()
	return s.clients[conn]
}

// getClientCount 获取客户端数量
func (s *WebSocketServer) getClientCount() int {
	s.clientsMu.RLock()
	defer s.clientsMu.RUnlock()
	return len(s.clients)
}

// handleMessages 处理客户端消息
func (s *WebSocketServer) handleMessages(conn *websocket.Conn, clientIP string) {
	defer func() {
		if r := recover(); r != nil {
			Error("handleMessages panic: %v", r)
			fmt.Printf("[DEBUG] handleMessages panic: %v\n", r)
		}
	}()

	fmt.Printf("[DEBUG] handleMessages 开始\n")
	ctx := s.ctx

	for {
		// 读取原始消息（二进制数据）
		fmt.Printf("[DEBUG] 等待读取消息...\n")
		msgType, rawData, err := conn.Read(ctx)
		fmt.Printf("[DEBUG] 读取到消息: type=%v, len=%d, err=%v\n", msgType, len(rawData), err)
		if err != nil {
			// 详细记录读取失败原因
			closeStatus := websocket.CloseStatus(err)
			if closeStatus == -1 {
				// 非WebSocket关闭错误，可能是网络问题
				Error("读取消息失败: ip=%s, 错误类型=网络错误, 详情: %v", clientIP, err)
			} else if closeStatus == websocket.StatusNormalClosure || closeStatus == websocket.StatusGoingAway {
				// 正常关闭，使用 Info 级别
				Info("客户端关闭连接: ip=%s, 关闭码=%d, 原因: %v", clientIP, closeStatus, err)
			} else {
				// WebSocket异常关闭帧
				Error("读取消息失败: ip=%s, 关闭码=%d, 详情: %v", clientIP, closeStatus, err)
			}
			return
		}

		// 记录接收到的消息类型
		if msgType == websocket.MessageText {
			Warn("收到文本消息（服务端期望二进制消息）: ip=%s, 数据长度=%d", clientIP, len(rawData))
		}

		// 安全控制：消息大小限制（解密前）
		if err := s.securityLogic.CheckRawMessageSize(len(rawData)); err != nil {
			Warn("消息被拒绝: ip=%s, err=%v", clientIP, err)
			conn.Close(websocket.StatusPolicyViolation, "消息被拒绝: "+err.Error())
			return
		}

		Debug("开始处理消息, 数据长度=%d", len(rawData))
		fmt.Printf("[DEBUG] 收到原始消息, 长度=%d\n", len(rawData))
		// 解密 → 解压 → 解析JSON
		msg, err := s.processIncomingMessage(rawData)
		if err != nil {
			Error("处理接收消息失败: %v", err)
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
				Error("发送错误响应失败: %v", err)
			}
			continue
		}

		Debug("收到消息: ID=%s, Type=%s", msg.ID, msg.Type)

		// 获取客户端认证信息
		ci := s.getClientInfo(conn)

		// 认证检查：未认证时只允许auth和update_config消息
		if s.authEnabled && !ci.authenticated && msg.Type != "auth" && msg.Type != "update_config" {
			response := types.Message{
				ID:   msg.ID,
				Type: "error",
				Data: map[string]interface{}{
					"code":    "AUTH_REQUIRED",
					"message": "请先登录认证",
				},
			}
			if err := s.sendMessage(ctx, conn, response); err != nil {
				Error("发送认证要求失败: %v", err)
			}
			continue
		}

		// 处理不同类型的消息
		var response types.Message
		response.ID = msg.ID

		switch msg.Type {
		case "auth":
			response = s.handleAuth(conn, msg, ci)
		case "update_crypto_key":
			response = s.handleUpdateCryptoKey(conn, msg, ci)
		case "update_config":
			response = s.handleUpdateConfig(conn, msg, ci)
		case "change_password":
			response = s.handleChangePassword(msg, ci)
		case "user_list":
			response = s.handleUserList(msg, ci)
		case "user_create":
			response = s.handleUserCreate(msg, ci)
		case "user_delete":
			response = s.handleUserDelete(msg, ci)
		case "user_update":
			response = s.handleUserUpdate(msg, ci)
		case "ping":
			// 心跳消息，回复pong
			response.Type = "pong"
			response.Data = msg.Data
		case "request":
			// 性能指标：记录请求开始
			if s.performanceLib != nil {
				s.performanceLib.IncRequest()
				defer s.performanceLib.DecRequest()
			}
			startTime := time.Now()

			// 安全控制：请求校验（IP/域名/请求体大小/限流）
			if err := s.securityLogic.CheckRequestMessage(clientIP, msg.Data); err != nil {
				Warn("请求被拒绝: ip=%s, err=%v", clientIP, err)
				response.Type = "response"
				response.Data = map[string]interface{}{
					"status":       0,
					"statusText":   "SecurityDenied",
					"headers":      make(map[string]string),
					"body":         err.Error(),
					"bodyEncoding": "text",
				}
				break
			}
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
				Error("处理请求失败: %v", err)
			} else {
				response = *responseMsg
				response.ID = msg.ID // 确保ID一致
			}

			// 性能指标：记录响应时间
			if s.performanceLib != nil {
				s.performanceLib.RecordResponseTime(time.Since(startTime))
			}
		case "close":
			// 关闭消息
			Info("收到关闭消息: %v", msg.Data)
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
			Error("发送消息失败: %v", err)
			return
		}
	}
}

// getClientIP 获取客户端IP（优先使用X-Forwarded-For，但本项目默认直连一般取RemoteAddr即可）
func getClientIP(r *http.Request) string {
	xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For"))
	if xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			ip := strings.TrimSpace(parts[0])
			if net.ParseIP(ip) != nil {
				return ip
			}
		}
	}
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil && net.ParseIP(host) != nil {
		return host
	}
	// RemoteAddr可能不带端口，兜底直接解析
	if net.ParseIP(strings.TrimSpace(r.RemoteAddr)) != nil {
		return strings.TrimSpace(r.RemoteAddr)
	}
	return ""
}

// processIncomingMessage 处理接收到的消息：解密 → 解压 → 解析JSON
func (s *WebSocketServer) processIncomingMessage(rawData []byte) (*types.Message, error) {
	// 先尝试直接解析JSON（检测是否为明文配置消息）
	var msg types.Message
	if err := json.Unmarshal(rawData, &msg); err == nil {
		// 解析成功，检查是否为配置消息
		if msg.Type == "update_config" {
			// 配置消息，直接返回（不解密）
			return &msg, nil
		}
	}

	// 不是配置消息，按正常流程处理：解密 → 解压 → 解析JSON
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
	responseMsg, logInfo, err := getRequestLogic().ProcessRequest(msg.Data)
	if err != nil {
		return responseMsg, err
	}

	// 记录日志
	if logInfo != nil {
		// info级别：记录URL、状态码、响应body大小
		Info("请求完成: %s %s, 状态码: %d, 响应大小: %d bytes",
			logInfo.Method, logInfo.URL, logInfo.RespStatus, logInfo.RespBodySize)

		// debug级别：记录请求body和响应body
		Debug("请求URL: %s %s, 请求Body: %s", logInfo.Method, logInfo.URL, truncateBody(logInfo.ReqBody))
		Debug("响应Body: %s", truncateBody(logInfo.RespBody))
	}

	return responseMsg, nil
}

// truncateBody 截断过长的body内容，避免日志过长
func truncateBody(body string) string {
	maxLen := 500
	if len(body) <= maxLen {
		return body
	}
	return body[:maxLen] + "...(截断)"
}

// handleAuth 处理认证消息
func (s *WebSocketServer) handleAuth(conn *websocket.Conn, msg *types.Message, ci *clientInfo) types.Message {
	response := types.Message{ID: msg.ID, Type: "auth_result"}

	data, ok := msg.Data.(map[string]interface{})
	if !ok {
		response.Data = map[string]interface{}{"success": false, "message": "无效的认证数据"}
		return response
	}

	username, _ := data["username"].(string)
	password, _ := data["password"].(string)

	session, err := s.authLogic.Authenticate(username, password)
	if err != nil {
		Warn("认证失败: username=%s, err=%v", username, err)
		response.Data = map[string]interface{}{"success": false, "message": err.Error()}
		// 发送失败响应后关闭连接
		s.sendMessage(context.Background(), conn, response)
		conn.Close(websocket.StatusPolicyViolation, "认证失败")
		return types.Message{} // 返回空消息，避免重复发送
	}

	// 更新客户端认证状态
	s.clientsMu.Lock()
	ci.authenticated = true
	ci.username = session.Username
	ci.role = session.Role
	ci.isSuperAdmin = session.IsSuperAdmin
	ci.isAdmin = session.IsAdmin
	s.clientsMu.Unlock()

	Info("用户认证成功: username=%s, role=%s", username, session.Role)
	response.Data = map[string]interface{}{
		"success":      true,
		"isAdmin":      session.IsAdmin,
		"isSuperAdmin": session.IsSuperAdmin,
		"role":         session.Role,
		"username":     session.Username,
		"token":        session.Token,
	}
	return response
}

// handleUpdateCryptoKey 处理更新加密密钥（兼容旧版本）
func (s *WebSocketServer) handleUpdateCryptoKey(conn *websocket.Conn, msg *types.Message, ci *clientInfo) types.Message {
	response := types.Message{ID: msg.ID, Type: "update_crypto_key_result"}

	data, ok := msg.Data.(map[string]interface{})
	if !ok {
		response.Data = map[string]interface{}{"success": false, "message": "无效的数据"}
		return response
	}

	key, _ := data["key"].(string)
	algorithm, _ := data["algorithm"].(string)

	if key == "" || algorithm == "" {
		response.Data = map[string]interface{}{"success": false, "message": "密钥或算法不能为空"}
		return response
	}

	// 创建新的加密库
	cryptoConfig := &CryptoConfig{
		Enabled:   true,
		Key:       key,
		Algorithm: algorithm,
	}

	newCryptoLib, err := NewCryptoLib(cryptoConfig)
	if err != nil {
		Warn("更新加密密钥失败: err=%v", err)
		response.Data = map[string]interface{}{"success": false, "message": fmt.Sprintf("更新失败: %v", err)}
		return response
	}

	// 更新服务器的加密库
	s.cryptoLib = newCryptoLib
	Info("加密密钥已更新: algorithm=%s", algorithm)

	response.Data = map[string]interface{}{"success": true, "message": "密钥更新成功"}
	return response
}

// handleUpdateConfig 处理更新配置（加密+压缩）
func (s *WebSocketServer) handleUpdateConfig(conn *websocket.Conn, msg *types.Message, ci *clientInfo) types.Message {
	response := types.Message{ID: msg.ID, Type: "config_result"}

	data, ok := msg.Data.(map[string]interface{})
	if !ok {
		response.Data = map[string]interface{}{"success": false, "message": "无效的数据"}
		return response
	}

	// 处理加密配置
	if cryptoData, ok := data["crypto"].(map[string]interface{}); ok {
		key, _ := cryptoData["key"].(string)
		algorithm, _ := cryptoData["algorithm"].(string)
		enabled, _ := cryptoData["enabled"].(bool)

		if enabled && (key == "" || algorithm == "") {
			response.Data = map[string]interface{}{"success": false, "message": "加密密钥或算法不能为空"}
			return response
		}

		cryptoConfig := &CryptoConfig{
			Enabled:   enabled,
			Key:       key,
			Algorithm: algorithm,
		}

		newCryptoLib, err := NewCryptoLib(cryptoConfig)
		if err != nil {
			Warn("更新加密配置失败: err=%v", err)
			response.Data = map[string]interface{}{"success": false, "message": fmt.Sprintf("更新加密配置失败: %v", err)}
			return response
		}

		s.cryptoLib = newCryptoLib
		Info("加密配置已更新: enabled=%v, algorithm=%s", enabled, algorithm)
	}

	// 处理压缩配置
	if compressData, ok := data["compress"].(map[string]interface{}); ok {
		algorithm, _ := compressData["algorithm"].(string)
		enabled, _ := compressData["enabled"].(bool)
		level := 6
		if lvl, ok := compressData["level"].(float64); ok {
			level = int(lvl)
		}

		compressConfig := &CompressConfig{
			Enabled:   enabled,
			Algorithm: algorithm,
			Level:     level,
		}

		newCompressLib, err := NewCompressLib(compressConfig)
		if err != nil {
			Warn("更新压缩配置失败: err=%v", err)
			response.Data = map[string]interface{}{"success": false, "message": fmt.Sprintf("更新压缩配置失败: %v", err)}
			return response
		}

		s.compressLib = newCompressLib
		Info("压缩配置已更新: enabled=%v, algorithm=%s, level=%d", enabled, algorithm, level)
	}

	response.Data = map[string]interface{}{"success": true, "message": "配置更新成功"}
	return response
}

// handleChangePassword 处理修改密码
func (s *WebSocketServer) handleChangePassword(msg *types.Message, ci *clientInfo) types.Message {
	response := types.Message{ID: msg.ID, Type: "change_password_result"}

	data, ok := msg.Data.(map[string]interface{})
	if !ok {
		response.Data = map[string]interface{}{"success": false, "message": "无效的数据"}
		return response
	}

	oldPassword, _ := data["oldPassword"].(string)
	newPassword, _ := data["newPassword"].(string)

	if err := s.authLogic.ChangePassword(ci.username, oldPassword, newPassword); err != nil {
		response.Data = map[string]interface{}{"success": false, "message": err.Error()}
		return response
	}

	Info("用户修改密码成功: username=%s", ci.username)
	response.Data = map[string]interface{}{"success": true, "message": "密码修改成功"}
	return response
}

// handleUserList 处理用户列表请求（管理员）
func (s *WebSocketServer) handleUserList(msg *types.Message, ci *clientInfo) types.Message {
	response := types.Message{ID: msg.ID, Type: "user_list_result"}
	if !ci.isAdmin {
		response.Data = map[string]interface{}{"success": false, "message": "权限不足"}
		return response
	}
	response.Data = map[string]interface{}{"success": true, "users": s.authLogic.ListUsers()}
	return response
}

// handleUserCreate 处理创建用户（管理员）
func (s *WebSocketServer) handleUserCreate(msg *types.Message, ci *clientInfo) types.Message {
	response := types.Message{ID: msg.ID, Type: "user_manage_result"}
	if !ci.isAdmin {
		response.Data = map[string]interface{}{"success": false, "message": "权限不足"}
		return response
	}

	data, ok := msg.Data.(map[string]interface{})
	if !ok {
		response.Data = map[string]interface{}{"success": false, "message": "无效的数据"}
		return response
	}

	username, _ := data["username"].(string)
	password, _ := data["password"].(string)
	role, _ := data["role"].(string)
	enabled := true
	if e, ok := data["enabled"].(bool); ok {
		enabled = e
	}

	if role == "" {
		role = "user"
	}

	if err := s.authLogic.CreateUser(username, password, role, enabled); err != nil {
		response.Data = map[string]interface{}{"success": false, "message": err.Error()}
		return response
	}

	Info("管理员%s创建用户: %s", ci.username, username)
	response.Data = map[string]interface{}{"success": true, "message": "用户创建成功"}
	return response
}

// handleUserDelete 处理删除用户（管理员）
func (s *WebSocketServer) handleUserDelete(msg *types.Message, ci *clientInfo) types.Message {
	response := types.Message{ID: msg.ID, Type: "user_manage_result"}
	if !ci.isAdmin {
		response.Data = map[string]interface{}{"success": false, "message": "权限不足"}
		return response
	}

	data, ok := msg.Data.(map[string]interface{})
	if !ok {
		response.Data = map[string]interface{}{"success": false, "message": "无效的数据"}
		return response
	}

	username, _ := data["username"].(string)
	if err := s.authLogic.DeleteUser(username, ci.username); err != nil {
		response.Data = map[string]interface{}{"success": false, "message": err.Error()}
		return response
	}

	Info("管理员%s删除用户: %s", ci.username, username)
	response.Data = map[string]interface{}{"success": true, "message": "用户删除成功"}
	return response
}

// handleUserUpdate 处理更新用户（管理员）
func (s *WebSocketServer) handleUserUpdate(msg *types.Message, ci *clientInfo) types.Message {
	response := types.Message{ID: msg.ID, Type: "user_manage_result"}
	if !ci.isAdmin {
		response.Data = map[string]interface{}{"success": false, "message": "权限不足"}
		return response
	}

	data, ok := msg.Data.(map[string]interface{})
	if !ok {
		response.Data = map[string]interface{}{"success": false, "message": "无效的数据"}
		return response
	}

	username, _ := data["username"].(string)
	newPassword, _ := data["password"].(string)
	var role *string
	if v, ok := data["role"].(string); ok {
		role = &v
	}
	var enabled *bool
	if v, ok := data["enabled"].(bool); ok {
		enabled = &v
	}

	if err := s.authLogic.UpdateUser(username, newPassword, role, enabled); err != nil {
		response.Data = map[string]interface{}{"success": false, "message": err.Error()}
		return response
	}

	Info("管理员%s更新用户: %s", ci.username, username)
	response.Data = map[string]interface{}{"success": true, "message": "用户更新成功"}
	return response
}

// Broadcast 广播消息给所有客户端
func (s *WebSocketServer) Broadcast(msg types.Message) error {
	s.clientsMu.RLock()
	defer s.clientsMu.RUnlock()

	ctx := context.Background()
	for conn := range s.clients {
		if err := s.sendMessage(ctx, conn, msg); err != nil {
			Error("广播消息失败: %v", err)
		}
	}
	return nil
}

// SendToClient 发送消息给指定客户端
func (s *WebSocketServer) SendToClient(conn *websocket.Conn, msg types.Message) error {
	ctx := context.Background()
	return s.sendMessage(ctx, conn, msg)
}
