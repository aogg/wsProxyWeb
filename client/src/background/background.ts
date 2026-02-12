// 浏览器插件后台脚本

import { WebSocketClient, ConnectionStatus } from '../libs/websocket';
import clientConfig from '../configs/client.json';

// WebSocket客户端实例
let wsClient: WebSocketClient | null = null;

// 初始化WebSocket连接
function initWebSocket(): void {
  const wsUrl = clientConfig.websocketUrl || 'ws://localhost:8080/ws';
  
  console.log('初始化WebSocket连接:', wsUrl);
  
  // 创建WebSocket客户端
  wsClient = new WebSocketClient(wsUrl);

  // 监听连接状态变化
  wsClient.onStatusChange((status: ConnectionStatus) => {
    console.log('WebSocket连接状态:', status);
    
    // 可以通过chrome.storage或chrome.runtime发送状态更新
    // 这里先简单记录日志
    updateConnectionStatus(status);
  });

  // 监听消息
  wsClient.onMessage((message) => {
    console.log('收到WebSocket消息:', message);
    // 后续会在这里处理服务端返回的响应
  });

  // 连接到服务器
  wsClient.connect();
}

// 更新连接状态（可以发送到popup界面）
function updateConnectionStatus(status: ConnectionStatus): void {
  // 使用chrome.storage存储连接状态，供popup界面读取
  chrome.storage.local.set({ 
    wsConnectionStatus: status,
    wsConnectionTime: Date.now()
  }).catch(err => {
    console.error('保存连接状态失败:', err);
  });
}

// 插件安装或启动时初始化
chrome.runtime.onInstalled.addListener(() => {
  console.log('插件已安装，初始化WebSocket连接');
  initWebSocket();
});

// Service Worker启动时初始化（Manifest V3）
chrome.runtime.onStartup.addListener(() => {
  console.log('插件已启动，初始化WebSocket连接');
  initWebSocket();
});

// 如果Service Worker已经运行，直接初始化
if (chrome.runtime.id) {
  console.log('WebSocket代理插件已加载');
  initWebSocket();
}

// 监听来自popup的消息（用于手动重连等操作）
chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
  if (message.type === 'reconnect') {
    console.log('收到重连请求');
    if (wsClient) {
      wsClient.disconnect();
    }
    initWebSocket();
    sendResponse({ success: true });
  } else if (message.type === 'getStatus') {
    const status = wsClient ? wsClient.getStatus() : ConnectionStatus.Disconnected;
    sendResponse({ status });
  }
  
  return true; // 保持消息通道开放以支持异步响应
});
