// 浏览器插件后台脚本

import { WebSocketClient, ConnectionStatus, Message, AuthResult } from '../libs/websocket';
import { CryptoConfig } from '../libs/crypto';
import { CompressConfig } from '../libs/compress';
import { StorageUtil, ClientConfig, RuleConfig, AuthState } from '../libs/storage';
import { RuleLib } from '../libs/rule_lib';
import { DEFAULT_CONFIG } from '../configs/defaults';

// 初始化状态键名
const INIT_KEY = 'ws_proxy_initialized';

// WebSocket客户端实例
let wsClient: WebSocketClient | null = null;

// 代理是否启用
let proxyEnabled: boolean = false;

// 请求拦截器是否已初始化
let interceptorInitialized = false;

// 全局标记（用于Service Worker重启时检测）
declare global {
  var __WS_INITIALIZED__: boolean;
  var __WS_CLIENT__: WebSocketClient | null;
}

// 规则匹配实例（用于决定是否代理）
let ruleLib: RuleLib | null = null;

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
  chromeRequestId: string; // Chrome的requestId
  requestDetails: chrome.webRequest.OnBeforeRequestDetails;
  headers?: chrome.webRequest.HttpHeader[];
  body?: string;
  bodyEncoding?: 'text' | 'base64';
  tabId?: number; // 标签页ID，用于返回响应
}

// 存储待处理的请求
const pendingRequests = new Map<string, PendingRequest>(); // key是chrome的requestId
const pendingRequestsById = new Map<string, PendingRequest>(); // key是我们生成的requestId，用于响应关联

// 检查是否已初始化且连接活跃
function isConnectionActive(): boolean {
  // 检查全局变量（Service Worker未重启时可复用）
  if (globalThis.__WS_CLIENT__ && globalThis.__WS_INITIALIZED__) {
    const client = globalThis.__WS_CLIENT__;
    if (client && client.getStatus() === ConnectionStatus.Connected) {
      console.log('复用已存在的WebSocket连接');
      wsClient = client;
      return true;
    }
  }
  return false;
}

// 初始化WebSocket连接
async function initWebSocket(): Promise<void> {
  // 检查连接是否已活跃
  if (isConnectionActive()) {
    return;
  }

  try {
    // 优先从storage读取配置，如果没有则使用默认配置
    let config: ClientConfig;
    try {
      config = await StorageUtil.getConfig();
      console.info('读取到的配置', config);
    } catch (error) {
      console.warn('从storage读取配置失败，使用默认配置:', error);
      config = {
        ...DEFAULT_CONFIG,
        crypto: DEFAULT_CONFIG.crypto,
        compress: DEFAULT_CONFIG.compress
      };
    }

    const wsUrl = config.websocketUrl || DEFAULT_CONFIG.websocketUrl;

    // 检查账号密码是否已配置
    if (!config.auth?.username || !config.auth?.password) {
      console.warn('未配置账号密码，无法连接WebSocket');
      throw new Error('请先配置账号密码');
    }

    console.log('初始化WebSocket连接:', wsUrl);

    // 读取加密和压缩配置（使用类型断言确保类型正确）
    const cryptoConfig: CryptoConfig = {
      enabled: config.crypto?.enabled ?? false,
      key: config.crypto?.key ?? '',
      algorithm: (config.crypto?.algorithm as CryptoConfig['algorithm']) ?? 'aes256gcm',
    };

    const compressConfig: CompressConfig = {
      enabled: config.compress?.enabled ?? false,
      level: config.compress?.level ?? 6,
      algorithm: (config.compress?.algorithm as CompressConfig['algorithm']) ?? 'gzip',
    };

    // 如果已有连接，先断开
    if (wsClient && wsClient !== globalThis.__WS_CLIENT__) {
      wsClient.disconnect();
    }

    // 创建WebSocket客户端（传入加密和压缩配置）
    wsClient = new WebSocketClient(wsUrl, cryptoConfig, compressConfig);

    // 保存到全局变量
    globalThis.__WS_CLIENT__ = wsClient;

    // 监听连接状态变化
    wsClient.onStatusChange((status: ConnectionStatus) => {
      console.log('WebSocket连接状态:', status);

      // 连接成功时标记已初始化并自动认证
      if (status === ConnectionStatus.Connected) {
        globalThis.__WS_INITIALIZED__ = true;
        autoAuth();
      }

      // 通过storage记录状态，供popup读取
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
  } catch (error) {
    console.error('初始化WebSocket连接失败:', error);
  }
}

// 更新连接状态（可以发送到popup界面）
function updateConnectionStatus(status: ConnectionStatus): void {
  // 使用chrome.storage存储连接状态，供popup界面读取
  try {
    chrome.storage.local.set(
      {
        wsConnectionStatus: status,
        wsConnectionTime: Date.now(),
      },
      () => {
        const error = chrome.runtime.lastError;
        if (error) {
          console.error('保存连接状态失败:', error);
        }
      },
    );
  } catch (error) {
    console.error('保存连接状态异常:', error);
  }
}

// 监听配置变化，自动重新初始化WebSocket连接
StorageUtil.onConfigChange((config: ClientConfig) => {
  console.log('检测到客户端配置变更，重新初始化WebSocket连接');
  // 使用最新配置重新建立连接
  initWebSocket().catch((error) => {
    console.error('因配置变更重新初始化WebSocket失败:', error);
  });
});

// 监听规则变化，实时更新规则匹配器
StorageUtil.onRulesChange((rules: RuleConfig) => {
  try {
    ruleLib = new RuleLib(rules);
    console.log('检测到规则配置变更，已更新规则匹配器');
  } catch (error) {
    console.error('更新规则匹配器失败:', error);
  }
});

// 插件安装或启动时初始化
chrome.runtime.onInstalled.addListener(async () => {
  console.log('插件已安装');
  // 清除会话标记，允许重新初始化
  try {
    await chrome.storage.session.remove(INIT_KEY);
  } catch {}
  globalThis.__WS_INITIALIZED__ = false;
  await initialize();
});

// Service Worker启动时初始化（Manifest V3）
chrome.runtime.onStartup.addListener(async () => {
  console.log('插件已启动');
  await initialize();
});

// 检查是否已在当前会话中初始化（使用 session storage）
async function checkSessionInit(): Promise<boolean> {
  try {
    const result = await chrome.storage.session.get(INIT_KEY);
    return !!result[INIT_KEY];
  } catch {
    // session storage 可能不被支持（Chrome < 102）
    return false;
  }
}

// 标记当前会话已初始化
async function markSessionInit(): Promise<void> {
  try {
    await chrome.storage.session.set({ [INIT_KEY]: true });
  } catch {
    // 忽略不支持的情况
  }
}

// 初始化保活机制
function initKeepAlive(): void {
  // 创建定期闹钟保持Service Worker活跃（每25秒，Chrome限制最小1分钟）
  chrome.alarms.create('keepAlive', { periodInMinutes: 1 });

  chrome.alarms.onAlarm.addListener((alarm) => {
    if (alarm.name === 'keepAlive') {
      // 检查连接状态
      if (wsClient) {
        const status = wsClient.getStatus();
        console.log('Service Worker保活检查, 连接状态:', status);
      }
    }
  });
}

// 统一初始化入口
async function initialize(): Promise<void> {
  console.log('开始初始化WebSocket代理插件...');

  // 读取代理启用状态
  try {
    const config = await StorageUtil.getConfig();
    proxyEnabled = config.proxyEnabled ?? false;
    console.log('代理启用状态:', proxyEnabled);
  } catch (error) {
    console.warn('读取代理启用状态失败，默认禁用:', error);
    proxyEnabled = false;
  }

  // 标记会话已初始化
  await markSessionInit();

  // 只有在代理启用时才初始化WebSocket连接
  if (proxyEnabled) {
    await initWebSocket();
  }

  // 初始化规则
  await initRules();

  // 初始化请求拦截（只注册一次）
  initRequestInterceptor();

  // 初始化保活机制
  initKeepAlive();
}

// 如果Service Worker已经运行，执行初始化
if (chrome.runtime.id) {
  console.log('WebSocket代理插件已加载');
  initialize().catch((error) => {
    console.error('初始化失败:', error);
  });
}

// 监听来自popup的消息（用于手动重连等操作）
chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
  if (message.type === 'reconnect') {
    console.log('收到重连请求');
    initWebSocket().then(() => {
      sendResponse({ success: true });
    }).catch((error) => {
      console.error('重连失败:', error);
      sendResponse({ success: false, error: error.message });
    });
    return true; // 保持消息通道开放以支持异步响应
  } else if (message.type === 'reloadConfig') {
    console.log('收到重新加载配置请求');
    initWebSocket().then(() => {
      sendResponse({ success: true });
    }).catch((error) => {
      console.error('重新加载配置失败:', error);
      sendResponse({ success: false, error: error.message });
    });
    return true; // 保持消息通道开放以支持异步响应
  } else if (message.type === 'getStatus') {
    const status = wsClient ? wsClient.getStatus() : ConnectionStatus.Disconnected;
    sendResponse({ status });
  } else if (message.type === 'startProxy') {
    console.log('收到启动代理请求');
    startProxy().then(() => {
      sendResponse({ success: true });
    }).catch((error) => {
      console.error('启动代理失败:', error);
      sendResponse({ success: false, error: error.message });
    });
    return true; // 保持消息通道开放以支持异步响应
  } else if (message.type === 'stopProxy') {
    console.log('收到停止代理请求');
    stopProxy().then(() => {
      sendResponse({ success: true });
    }).catch((error) => {
      console.error('停止代理失败:', error);
      sendResponse({ success: false, error: error.message });
    });
    return true; // 保持消息通道开放以支持异步响应
  } else if (message.type === 'connect') {
    console.log('收到启用连接请求');
    connectWebSocket().then(() => {
      sendResponse({ success: true });
    }).catch((error) => {
      console.error('启用连接失败:', error);
      sendResponse({ success: false, error: error.message });
    });
    return true;
  } else if (message.type === 'disconnect') {
    console.log('收到停止连接请求');
    disconnectWebSocket().then(() => {
      sendResponse({ success: true });
    }).catch((error) => {
      console.error('停止连接失败:', error);
      sendResponse({ success: false, error: error.message });
    });
    return true;
  } else if (message.type === 'login') {
    // 登录认证
    handleLogin(message.data).then((result) => {
      sendResponse(result);
    }).catch((error) => {
      sendResponse({ success: false, message: error.message });
    });
    return true;
  } else if (message.type === 'logout') {
    StorageUtil.clearAuthState().then(() => {
      sendResponse({ success: true });
    });
    return true;
  } else if (message.type === 'wsMessage') {
    // 通用WS消息转发（用于changePassword、userList、userCreate、userDelete、userUpdate）
    handleWsMessage(message.data).then((result) => {
      sendResponse(result);
    }).catch((error) => {
      sendResponse({ success: false, message: error.message });
    });
    return true;
  }
  
  return true; // 保持消息通道开放以支持异步响应
});

// 启动代理
async function startProxy(): Promise<void> {
  console.log('启动代理...');
  proxyEnabled = true;
  await initWebSocket();
  console.log('代理已启动');
}

// 停止代理
async function stopProxy(): Promise<void> {
  console.log('停止代理...');
  proxyEnabled = false;
  if (wsClient) {
    wsClient.disconnect();
    wsClient = null;
    globalThis.__WS_CLIENT__ = null;
    globalThis.__WS_INITIALIZED__ = false;
    updateConnectionStatus(ConnectionStatus.Disconnected);
  }
  console.log('代理已停止');
}

// 启用WebSocket连接
async function connectWebSocket(): Promise<void> {
  console.log('启用WebSocket连接...');
  await initWebSocket();
  console.log('WebSocket连接已启用');
}

// 停止WebSocket连接
async function disconnectWebSocket(): Promise<void> {
  console.log('停止WebSocket连接...');

  // 停止代理
  proxyEnabled = false;
  const config = await StorageUtil.getConfig();
  await StorageUtil.saveConfig({ ...config, proxyEnabled: false });

  // 清除认证状态
  await StorageUtil.clearAuthState();

  // 断开WebSocket连接
  if (wsClient) {
    wsClient.disconnect();
    wsClient = null;
    globalThis.__WS_CLIENT__ = null;
    globalThis.__WS_INITIALIZED__ = false;
    updateConnectionStatus(ConnectionStatus.Disconnected);
  }
  console.log('WebSocket连接已停止，代理已停止，已退出登录');
}

// 自动认证（连接成功后使用存储的账号密码）
async function autoAuth(): Promise<void> {
  if (!wsClient) return;
  try {
    const config = await StorageUtil.getConfig();
    if (config.auth?.username && config.auth?.password) {
      console.log('自动认证中...');
      const result = await wsClient.sendAuth(config.auth.username, config.auth.password);
      if (result.success) {
        await StorageUtil.saveAuthState({
          authenticated: true,
          username: result.username,
          isAdmin: result.isAdmin,
          token: result.token || '',
        });
        console.log('自动认证成功:', result.username);
      } else {
        await StorageUtil.clearAuthState();
        console.warn('自动认证失败:', result.message);
      }
    }
  } catch (error) {
    console.error('自动认证异常:', error);
  }
}

// 处理登录请求
async function handleLogin(data: Partial<ClientConfig>): Promise<AuthResult> {
  // 如果WebSocket未连接，先建立连接
  if (!wsClient || wsClient.getStatus() !== ConnectionStatus.Connected) {
    try {
      // 先保存表单的完整配置，确保initWebSocket使用表单值
      const config = await StorageUtil.getConfig();
      await StorageUtil.saveConfig({ ...config, ...data });
      await initWebSocket();
      // 等待连接建立（最多5秒）
      const maxWait = 5000;
      const startTime = Date.now();
      while ((!wsClient || wsClient.getStatus() !== ConnectionStatus.Connected) && Date.now() - startTime < maxWait) {
        await new Promise(resolve => setTimeout(resolve, 100));
      }
      if (!wsClient || wsClient.getStatus() !== ConnectionStatus.Connected) {
        return { success: false, isAdmin: false, username: data.auth?.username || '', message: '连接服务器失败' };
      }
    } catch (error) {
      return { success: false, isAdmin: false, username: data.auth?.username || '', message: '连接服务器失败' };
    }
  }

  const result = await wsClient.sendAuth(data.auth!.username, data.auth!.password);
  if (result.success) {
    // 保存完整配置
    const config = await StorageUtil.getConfig();
    await StorageUtil.saveConfig({ ...config, ...data });
    // 保存认证状态
    await StorageUtil.saveAuthState({
      authenticated: true,
      username: result.username,
      isAdmin: result.isAdmin,
      token: result.token || '',
    });
  }
  return result;
}

// 通用WS消息转发（用于管理操作）
async function handleWsMessage(data: { type: string; data?: any }): Promise<any> {
  if (!wsClient || wsClient.getStatus() !== ConnectionStatus.Connected) {
    return { success: false, message: 'WebSocket未连接' };
  }

  const msgId = `msg_${Date.now()}_${Math.random().toString(36).substring(2, 8)}`;
  const expectedResponseType = getResponseType(data.type);

  return new Promise((resolve) => {
    const timeout = setTimeout(() => {
      wsClient?.offMessage(handler);
      resolve({ success: false, message: '请求超时' });
    }, 10000);

    const handler = (message: Message) => {
      if (message.type === expectedResponseType) {
        clearTimeout(timeout);
        wsClient?.offMessage(handler);
        resolve(message.data);
      }
    };

    wsClient!.onMessage(handler);
    wsClient!.send({ id: msgId, type: data.type, data: data.data }).catch((err) => {
      clearTimeout(timeout);
      wsClient?.offMessage(handler);
      resolve({ success: false, message: err.message });
    });
  });
}

// 获取消息对应的响应类型
function getResponseType(type: string): string {
  const map: Record<string, string> = {
    'change_password': 'change_password_result',
    'user_list': 'user_list_result',
    'user_create': 'user_manage_result',
    'user_delete': 'user_manage_result',
    'user_update': 'user_manage_result',
  };
  return map[type] || `${type}_result`;
}

/**
 * 初始化请求拦截
 */
function initRequestInterceptor(): void {
  // 防止重复注册监听器
  if (interceptorInitialized) {
    console.log('请求拦截器已初始化，跳过');
    return;
  }
  interceptorInitialized = true;

  console.log('初始化请求拦截器');

  // 拦截所有HTTP/HTTPS请求（获取请求体）
  chrome.webRequest.onBeforeRequest.addListener(
    (details: chrome.webRequest.OnBeforeRequestDetails) => {
      // 只处理主框架和子资源请求，跳过扩展内部请求
      if (details.url.startsWith('chrome-extension://') || 
          details.url.startsWith('chrome://') ||
          details.url.startsWith('edge://')) {
        return;
      }

      // 检查代理是否启用
      if (!proxyEnabled) {
        return;
      }

      // 按规则决定是否需要代理（规则未初始化时默认放行，避免误拦截）
      if (ruleLib) {
        const match = ruleLib.shouldProxy(details.url);
        if (!match.shouldProxy) {
          // 规则未命中：直接放行，不进入后续代理流程
          console.log('规则未命中，不代理该请求:', match.reason, details.url);
          return;
        }
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
        chromeRequestId: details.requestId,
        requestDetails: details,
        tabId: details.tabId
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
            for (const value of values as string[]) {
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
    (details: chrome.webRequest.OnBeforeSendHeadersDetails) => {
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
        pendingRequest.requestDetails = { ...pendingRequest.requestDetails, ...details } as chrome.webRequest.OnBeforeRequestDetails;
      }

      // 此时有了完整的请求信息，发送到服务端
      sendRequestToServer(
        pendingRequest.requestId,
        pendingRequest.requestDetails,
        pendingRequest.body || '',
        pendingRequest.bodyEncoding || 'text'
      );
      
      return undefined;
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
 * 初始化规则配置（从storage读取，失败则使用默认值）
 */
async function initRules(): Promise<void> {
  try {
    const rules = await StorageUtil.getRules();
    ruleLib = new RuleLib(rules);
    console.log('规则匹配器初始化完成');
  } catch (error) {
    console.warn('读取规则配置失败，使用默认规则:', error);
    const defaultRules: RuleConfig = {
      enabled: true,
      whitelist: [],
      blacklist: [],
      urlPatterns: [],
    };
    ruleLib = new RuleLib(defaultRules);
  }
}

/**
 * 将请求信息发送到服务端
 */
async function sendRequestToServer(
  requestId: string,
  details: chrome.webRequest.OnBeforeRequestDetails,
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
  chrome.scripting.executeScript({
    target: { tabId: tabId },
    func: (url: string, data: string) => {
      // 创建一个隐藏的iframe来加载data URL，然后替换原始资源
      // 或者直接修改页面的fetch/XMLHttpRequest
      // 这里使用简单的方法：通过postMessage通知content script
      window.postMessage({
        type: 'WS_PROXY_RESPONSE',
        url: url,
        dataUrl: data
      }, '*');
    },
    args: [originalUrl, dataUrl],
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
