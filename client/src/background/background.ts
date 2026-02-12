// 浏览器插件后台脚本

import { WebSocketClient, ConnectionStatus, Message } from '../libs/websocket';
import clientConfig from '../configs/client.json';

// WebSocket客户端实例
let wsClient: WebSocketClient | null = null;

// 请求ID生成器（用于生成唯一的请求ID）
let requestIdCounter = 0;
function generateRequestId(): string {
  const timestamp = Date.now();
  const random = Math.random().toString(36).substring(2, 8);
  requestIdCounter++;
  return `req_${timestamp}_${random}_${requestIdCounter}`;
}

// 存储请求信息（用于关联请求和响应）
interface PendingRequest {
  requestId: string;
  requestDetails: chrome.webRequest.WebRequestDetails;
  headers?: chrome.webRequest.HttpHeader[];
  body?: string;
  bodyEncoding?: 'text' | 'base64';
}

const pendingRequests = new Map<number, PendingRequest>(); // key是details.requestId

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
    // 后续会在这里处理服务端返回的响应（任务3.3实现）
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
  // 初始化请求拦截
  initRequestInterceptor();
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

/**
 * 初始化请求拦截
 */
function initRequestInterceptor(): void {
  console.log('初始化请求拦截器');

  // 拦截所有HTTP/HTTPS请求（获取请求体）
  chrome.webRequest.onBeforeRequest.addListener(
    (details: chrome.webRequest.WebRequestDetails) => {
      // 只处理主框架和子资源请求，跳过扩展内部请求
      if (details.url.startsWith('chrome-extension://') || 
          details.url.startsWith('chrome://') ||
          details.url.startsWith('edge://')) {
        return;
      }

      // 生成请求ID
      const requestId = generateRequestId();

      // 存储请求信息
      const pendingRequest: PendingRequest = {
        requestId,
        requestDetails: details
      };

      // 获取请求体
      let requestBody = '';
      let bodyEncoding: 'text' | 'base64' = 'text';

      if (details.requestBody) {
        // 处理请求体
        if (details.requestBody.formData) {
          // 表单数据，转换为URL编码格式
          const formData = details.requestBody.formData;
          const formPairs: string[] = [];
          for (const [key, values] of Object.entries(formData)) {
            for (const value of values) {
              formPairs.push(`${encodeURIComponent(key)}=${encodeURIComponent(value)}`);
            }
          }
          requestBody = formPairs.join('&');
        } else if (details.requestBody.raw) {
          // 原始二进制数据
          const rawData = details.requestBody.raw;
          if (rawData && rawData.length > 0) {
            // 将ArrayBuffer转换为Base64
            const uint8Array = new Uint8Array(rawData[0].bytes!);
            const binaryString = Array.from(uint8Array)
              .map(byte => String.fromCharCode(byte))
              .join('');
            requestBody = btoa(binaryString);
            bodyEncoding = 'base64';
          }
        } else if (details.requestBody.error) {
          console.warn('获取请求体失败:', details.requestBody.error);
        }
      }

      pendingRequest.body = requestBody;
      pendingRequest.bodyEncoding = bodyEncoding;
      pendingRequests.set(details.requestId, pendingRequest);
    },
    {
      urls: ['<all_urls>']
    },
    ['requestBody']
  );

  // 监听请求头（获取完整的请求头信息后发送请求）
  chrome.webRequest.onBeforeSendHeaders.addListener(
    (details: chrome.webRequest.WebRequestHeadersDetails) => {
      // 查找对应的待处理请求
      const pendingRequest = pendingRequests.get(details.requestId);
      if (!pendingRequest) {
        return;
      }

      // 更新请求头信息
      pendingRequest.headers = details.requestHeaders;

      // 此时有了完整的请求信息，发送到服务端
      sendRequestToServer(
        pendingRequest.requestId,
        pendingRequest.requestDetails,
        pendingRequest.body || '',
        pendingRequest.bodyEncoding || 'text'
      );
    },
    {
      urls: ['<all_urls>']
    },
    ['requestHeaders']
  );

  console.log('请求拦截器初始化完成');
}

/**
 * 将请求信息发送到服务端
 */
function sendRequestToServer(
  requestId: string,
  details: chrome.webRequest.WebRequestDetails,
  requestBody: string,
  bodyEncoding: 'text' | 'base64'
): void {
  // 检查WebSocket连接状态
  if (!wsClient || wsClient.getStatus() !== ConnectionStatus.Connected) {
    console.warn('WebSocket未连接，无法发送请求:', details.url);
    pendingRequests.delete(requestId);
    return;
  }

  // 获取请求头
  const headers: Record<string, string> = {};
  // 查找对应的待处理请求（通过details.requestId）
  for (const [reqId, pendingRequest] of pendingRequests.entries()) {
    if (pendingRequest.requestId === requestId) {
      if (pendingRequest.headers) {
        for (const header of pendingRequest.headers) {
          headers[header.name] = header.value || '';
        }
      }
      // 发送后删除待处理请求
      pendingRequests.delete(reqId);
      break;
    }
  }

  // 构建协议消息
  const message: Message = {
    id: requestId,
    type: 'request',
    data: {
      url: details.url,
      method: details.method,
      headers: headers,
      body: requestBody,
      bodyEncoding: bodyEncoding
    }
  };

  // 发送消息
  const success = wsClient.send(message);
  if (success) {
    console.log('请求已发送到服务端:', requestId, details.url);
  } else {
    console.error('发送请求失败:', requestId, details.url);
    // 清理待处理请求
    for (const [reqId, pendingRequest] of pendingRequests.entries()) {
      if (pendingRequest.requestId === requestId) {
        pendingRequests.delete(reqId);
        break;
      }
    }
  }
}
