// 浏览器插件后台脚本

import { WebSocketClient, ConnectionStatus, Message } from '../libs/websocket';
import { CryptoConfig } from '../libs/crypto';
import { CompressConfig } from '../libs/compress';
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
  requestId: string; // 我们生成的请求ID
  chromeRequestId: number; // Chrome的requestId
  requestDetails: chrome.webRequest.WebRequestDetails;
  headers?: chrome.webRequest.HttpHeader[];
  body?: string;
  bodyEncoding?: 'text' | 'base64';
  tabId?: number; // 标签页ID，用于返回响应
}

// 存储待处理的请求
const pendingRequests = new Map<number, PendingRequest>(); // key是chrome的requestId
const pendingRequestsById = new Map<string, PendingRequest>(); // key是我们生成的requestId，用于响应关联

// 初始化WebSocket连接
function initWebSocket(): void {
  const wsUrl = clientConfig.websocketUrl || 'ws://localhost:8080/ws';
  
  console.log('初始化WebSocket连接:', wsUrl);
  
  // 读取加密和压缩配置
  const cryptoConfig: CryptoConfig = (clientConfig as any).crypto || {
    enabled: false,
    key: '',
    algorithm: 'aes256gcm'
  };
  
  const compressConfig: CompressConfig = (clientConfig as any).compress || {
    enabled: false,
    level: 6,
    algorithm: 'gzip'
  };
  
  // 创建WebSocket客户端（传入加密和压缩配置）
  wsClient = new WebSocketClient(wsUrl, cryptoConfig, compressConfig);

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
    // 处理服务端返回的响应
    handleWebSocketResponse(message);
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

      // 检查是否有缓存的响应（用于返回代理响应）
      // 查找所有可能的缓存key
      for (const [key, cachedResponse] of responseCache.entries()) {
        if (cachedResponse.originalUrl === details.url || key.endsWith(`_${details.url}`)) {
          console.log('返回缓存的响应:', details.url);
          // 返回data URL作为响应
          responseCache.delete(key);
          return { redirectUrl: cachedResponse.dataUrl };
        }
      }

      // 检查WebSocket连接状态，如果未连接则让请求正常进行
      if (!wsClient || wsClient.getStatus() !== ConnectionStatus.Connected) {
        // WebSocket未连接，不拦截请求，让请求正常进行
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
      pendingRequest.chromeRequestId = details.requestId;
      pendingRequest.tabId = details.tabId;
      
      // 存储到两个Map中，方便查找
      pendingRequests.set(details.requestId, pendingRequest);
      pendingRequestsById.set(requestId, pendingRequest);
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

      // 检查WebSocket连接状态，如果已连接则取消原始请求
      if (wsClient && wsClient.getStatus() === ConnectionStatus.Connected) {
        // 取消原始请求，等待代理响应
        // 注意：这里不能直接返回cancel，需要在onBeforeRequest中处理
        // 所以我们需要标记这个请求需要代理
        pendingRequest.requestDetails = { ...pendingRequest.requestDetails, ...details };
      }

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

  // 添加onBeforeRequest监听器来处理取消请求和返回响应
  // 注意：这个监听器需要在获取请求体之后执行，所以使用异步方式
  // 实际上，我们需要在收到响应后触发重定向

  console.log('请求拦截器初始化完成');
}

/**
 * 将请求信息发送到服务端
 */
async function sendRequestToServer(
  requestId: string,
  details: chrome.webRequest.WebRequestDetails,
  requestBody: string,
  bodyEncoding: 'text' | 'base64'
): Promise<void> {
  // 检查WebSocket连接状态
  if (!wsClient || wsClient.getStatus() !== ConnectionStatus.Connected) {
    console.warn('WebSocket未连接，无法发送请求:', details.url);
    // 清理待处理请求
    const pendingRequest = pendingRequestsById.get(requestId);
    if (pendingRequest) {
      pendingRequests.delete(pendingRequest.chromeRequestId);
      pendingRequestsById.delete(requestId);
    }
    return;
  }

  // 获取请求头
  const headers: Record<string, string> = {};
  const pendingRequest = pendingRequestsById.get(requestId);
  if (pendingRequest && pendingRequest.headers) {
    for (const header of pendingRequest.headers) {
      headers[header.name] = header.value || '';
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

  // 发送消息（现在是异步的）
  try {
    const success = await wsClient.send(message);
    if (success) {
      console.log('请求已发送到服务端:', requestId, details.url);
      // 注意：不在这里删除pendingRequest，等待响应返回后再删除
    } else {
      console.error('发送请求失败:', requestId, details.url);
      // 清理待处理请求
      const pendingRequest = pendingRequestsById.get(requestId);
      if (pendingRequest) {
        pendingRequests.delete(pendingRequest.chromeRequestId);
        pendingRequestsById.delete(requestId);
      }
    }
  } catch (error) {
    console.error('发送请求异常:', error);
    // 清理待处理请求
    const pendingRequest = pendingRequestsById.get(requestId);
    if (pendingRequest) {
      pendingRequests.delete(pendingRequest.chromeRequestId);
      pendingRequestsById.delete(requestId);
    }
  }
}

/**
 * 处理WebSocket响应消息
 */
function handleWebSocketResponse(message: Message): void {
  // 只处理响应类型的消息
  if (message.type !== 'response') {
    return;
  }

  // 根据消息ID查找对应的待处理请求
  const pendingRequest = pendingRequestsById.get(message.id);
  if (!pendingRequest) {
    console.warn('未找到对应的请求:', message.id);
    return;
  }

  console.log('处理响应:', message.id, pendingRequest.requestDetails.url);

  // 获取响应数据
  const responseData = message.data || {};
  const error = (message as any).error;

  // 如果有错误，返回错误响应
  if (error) {
    console.error('请求错误:', error);
    returnErrorResponse(pendingRequest, error);
    // 清理待处理请求
    cleanupPendingRequest(message.id);
    return;
  }

  // 返回正常响应
  returnResponse(pendingRequest, responseData);
  
  // 清理待处理请求
  cleanupPendingRequest(message.id);
}

/**
 * 返回正常响应
 * 使用chrome.webRequest.onBeforeRequest返回data URL的方式
 * 注意：data URL有大小限制（通常几MB），对于大数据需要使用其他方法
 */
function returnResponse(
  pendingRequest: PendingRequest,
  responseData: any
): void {
  const { status, statusText, headers, body, bodyEncoding } = responseData;

  // 构建响应数据
  const responseBody = body || '';
  const responseHeaders = headers || {};
  const responseStatus = status || 200;
  const responseStatusText = statusText || 'OK';

  // 处理响应体编码
  let finalBody: string = responseBody;
  let contentType = responseHeaders['Content-Type'] || responseHeaders['content-type'] || 'text/plain';
  
  // 构建data URL
  let dataUrl: string;
  if (bodyEncoding === 'base64' && responseBody) {
    // Base64编码的数据，直接构建data URL
    dataUrl = `data:${contentType};base64,${responseBody}`;
  } else {
    // 文本响应，对响应体进行URL编码
    const encodedBody = encodeURIComponent(finalBody);
    dataUrl = `data:${contentType};charset=utf-8,${encodedBody}`;
  }
  
  // 存储响应数据，使用请求ID作为key，因为URL可能重复
  const cacheKey = `${pendingRequest.requestId}_${pendingRequest.requestDetails.url}`;
  storeResponseForRequest(cacheKey, {
    status: responseStatus,
    statusText: responseStatusText,
    headers: responseHeaders,
    dataUrl: dataUrl,
    originalUrl: pendingRequest.requestDetails.url
  });

  // 如果有tabId，尝试触发页面重新请求（通过注入脚本）
  if (pendingRequest.tabId !== undefined && pendingRequest.tabId >= 0) {
    triggerResponseInjection(pendingRequest.tabId, pendingRequest.requestDetails.url, dataUrl);
  }
}

/**
 * 返回错误响应
 */
function returnErrorResponse(
  pendingRequest: PendingRequest,
  error: string
): void {
  // 构建错误响应的data URL
  const errorBody = `代理错误: ${error}`;
  const encodedBody = encodeURIComponent(errorBody);
  const dataUrl = `data:text/plain;charset=utf-8,${encodedBody}`;
  
  // 存储错误响应
  storeResponseForRequest(pendingRequest.requestDetails.url, {
    status: 500,
    statusText: 'Internal Server Error',
    headers: { 'Content-Type': 'text/plain' },
    dataUrl: dataUrl
  });
}

// 存储响应数据，用于在onBeforeRequest中返回
interface StoredResponse {
  status: number;
  statusText: string;
  headers: Record<string, string>;
  dataUrl: string;
  originalUrl?: string; // 原始URL，用于匹配
}

const responseCache = new Map<string, StoredResponse>();

/**
 * 存储响应数据
 */
function storeResponseForRequest(key: string, response: StoredResponse): void {
  responseCache.set(key, response);
  console.log('响应已存储:', key, response.status);
  
  // 设置超时清理（30秒后自动清理）
  setTimeout(() => {
    responseCache.delete(key);
  }, 30000);
}

/**
 * 触发响应注入（通过注入脚本返回响应）
 */
function triggerResponseInjection(tabId: number, originalUrl: string, dataUrl: string): void {
  // 注入脚本，将响应数据注入到页面
  const script = `
    (function() {
      // 创建一个隐藏的iframe来加载data URL，然后替换原始资源
      // 或者直接修改页面的fetch/XMLHttpRequest
      // 这里使用简单的方法：通过postMessage通知content script
      window.postMessage({
        type: 'WS_PROXY_RESPONSE',
        url: ${JSON.stringify(originalUrl)},
        dataUrl: ${JSON.stringify(dataUrl)}
      }, '*');
    })();
  `;

  chrome.scripting.executeScript({
    target: { tabId: tabId },
    func: new Function(script),
    world: 'MAIN'
  }).catch(err => {
    console.error('注入响应脚本失败:', err);
    // 如果注入失败，尝试使用重定向方式
    // 注意：这需要页面重新请求，可能不是最佳方案
  });
}

/**
 * 清理待处理请求
 */
function cleanupPendingRequest(requestId: string): void {
  const pendingRequest = pendingRequestsById.get(requestId);
  if (pendingRequest) {
    pendingRequests.delete(pendingRequest.chromeRequestId);
    pendingRequestsById.delete(requestId);
  }
}
