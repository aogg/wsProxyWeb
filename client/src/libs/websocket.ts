// WebSocket工具库

import { CryptoUtil, CryptoConfig } from './crypto';
import { CompressUtil, CompressConfig } from './compress';

// 连接状态枚举
export enum ConnectionStatus {
  Disconnected = 'disconnected',
  Connecting = 'connecting',
  Connected = 'connected',
  Reconnecting = 'reconnecting',
  Error = 'error'
}

// 消息类型
export interface Message {
  id: string;
  type: string;
  data?: any;
}

// 待处理请求
interface PendingMessage {
  message: Message;
  resolve: (success: boolean) => void;
  reject: (error: Error) => void;
  timestamp: number;
}

// 连接状态变化回调
export type StatusChangeCallback = (status: ConnectionStatus) => void;

// 消息接收回调
export type MessageCallback = (message: Message) => void;

// 认证结果
export interface AuthResult {
  success: boolean;
  isAdmin: boolean;
  username: string;
  token?: string;
  message?: string;
}

// 默认配置
const DEFAULT_CONFIG = {
  requestQueueSize: 100,      // 请求队列最大长度
  requestTimeout: 60000,      // 请求超时时间（毫秒）
  maxRetryAttempts: 3,        // 发送失败最大重试次数
  retryDelay: 1000,           // 重试延迟（毫秒）
};

// WebSocket客户端类
export class WebSocketClient {
  private ws: WebSocket | null = null;
  private url: string;
  private status: ConnectionStatus = ConnectionStatus.Disconnected;
  private reconnectAttempts: number = 0;
  private maxReconnectAttempts: number = 10;
  private reconnectDelay: number = 1000; // 初始重连延迟（毫秒）
  private maxReconnectDelay: number = 30000; // 最大重连延迟（毫秒）
  private heartbeatInterval: number = 30000; // 心跳间隔（毫秒）
  private heartbeatTimer: ReturnType<typeof setInterval> | null = null;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private statusChangeCallbacks: StatusChangeCallback[] = [];
  private messageCallbacks: MessageCallback[] = [];
  private shouldReconnect: boolean = true;
  private cryptoUtil: CryptoUtil | null = null;
  private compressUtil: CompressUtil | null = null;

  // 请求队列和超时处理
  private requestQueue: PendingMessage[] = [];
  private requestQueueSize: number = DEFAULT_CONFIG.requestQueueSize;
  private requestTimeout: number = DEFAULT_CONFIG.requestTimeout;
  private maxRetryAttempts: number = DEFAULT_CONFIG.maxRetryAttempts;
  private retryDelay: number = DEFAULT_CONFIG.retryDelay;
  private queueCheckTimer: ReturnType<typeof setInterval> | null = null;

  // 认证状态
  private authenticated: boolean = false;
  private authUsername: string = '';
  private authIsAdmin: boolean = false;

  constructor(url: string, cryptoConfig?: CryptoConfig, compressConfig?: CompressConfig) {
    this.url = url;
    
    // 初始化加密工具
    if (cryptoConfig) {
      try {
        this.cryptoUtil = new CryptoUtil(cryptoConfig);
        console.log(`加密功能: ${cryptoConfig.enabled ? '已启用' : '已禁用'} (算法: ${cryptoConfig.algorithm})`);
      } catch (error) {
        console.error('初始化加密工具失败:', error);
        this.cryptoUtil = null;
      }
    }
    
    // 初始化压缩工具
    if (compressConfig) {
      try {
        this.compressUtil = new CompressUtil(compressConfig);
        console.log(`压缩功能: ${compressConfig.enabled ? '已启用' : '已禁用'} (算法: ${compressConfig.algorithm}, 级别: ${compressConfig.level})`);
      } catch (error) {
        console.error('初始化压缩工具失败:', error);
        this.compressUtil = null;
      }
    }
  }

  /**
   * 连接到WebSocket服务器
   */
  public connect(): void {
    if (this.status === ConnectionStatus.Connected || 
        this.status === ConnectionStatus.Connecting) {
      console.log('WebSocket已经连接或正在连接中');
      return;
    }

    this.setStatus(ConnectionStatus.Connecting);
    this.shouldReconnect = true;

    try {
      this.ws = new WebSocket(this.url);

      this.ws.onopen = () => {
        console.log('WebSocket连接成功');
        this.setStatus(ConnectionStatus.Connected);
        this.reconnectAttempts = 0;
        this.reconnectDelay = 1000;
        this.startHeartbeat();
        this.startQueueProcessor();
        // 如果启用加密，推送密钥给服务端
        this.sendCryptoKeyToServer();
        // 处理队列中的待发送消息
        this.flushQueue();
      };

      this.ws.onmessage = async (event) => {
        try {
          // 处理接收到的消息：解密 → 解压 → 解析JSON
          let messageData: Message;
          
          if (event.data instanceof ArrayBuffer || event.data instanceof Blob) {
            // 二进制数据，需要解密和解压
            const arrayBuffer = event.data instanceof Blob 
              ? await event.data.arrayBuffer() 
              : event.data;
            const uint8Array = new Uint8Array(arrayBuffer);
            
            // 解密 → 解压 → 解析JSON
            const decryptedData = await this.processIncomingMessage(uint8Array);
            messageData = JSON.parse(new TextDecoder().decode(decryptedData));
          } else {
            // 文本数据，直接解析JSON（兼容未启用加密压缩的情况）
            messageData = JSON.parse(event.data);
          }
          
          // 处理心跳响应
          if (messageData.type === 'pong' || messageData.type === 'heartbeat') {
            return;
          }

          // 触发消息回调
          this.messageCallbacks.forEach(callback => {
            try {
              callback(messageData);
            } catch (error) {
              console.error('消息回调执行错误:', error);
            }
          });
        } catch (error) {
          console.error('解析消息失败:', error, event.data);
        }
      };

      this.ws.onerror = (error) => {
        console.error('WebSocket错误:', error);
        this.setStatus(ConnectionStatus.Error);
      };

      this.ws.onclose = (event) => {
        // 详细记录关闭原因
        const closeReason = this.getCloseReasonDescription(event.code);
        console.error('WebSocket连接关闭:', {
          code: event.code,
          reason: event.reason || closeReason,
          wasClean: event.wasClean,
          description: closeReason
        });
        this.stopHeartbeat();
        this.setStatus(ConnectionStatus.Disconnected);

        // 如果不是主动关闭，则尝试重连
        if (this.shouldReconnect && this.reconnectAttempts < this.maxReconnectAttempts) {
          this.scheduleReconnect();
        } else if (this.reconnectAttempts >= this.maxReconnectAttempts) {
          console.error('达到最大重连次数，停止重连');
          this.setStatus(ConnectionStatus.Error);
        }
      };
    } catch (error) {
      console.error('创建WebSocket连接失败:', error);
      this.setStatus(ConnectionStatus.Error);
      if (this.shouldReconnect) {
        this.scheduleReconnect();
      }
    }
  }

  /**
   * 断开连接
   */
  public disconnect(): void {
    this.shouldReconnect = false;
    this.stopHeartbeat();
    this.stopQueueProcessor();
    this.clearReconnectTimer();

    // 拒绝所有待处理的请求
    this.rejectAllPending(new Error('连接已断开'));

    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }

    this.setStatus(ConnectionStatus.Disconnected);
  }

  /**
   * 发送消息（支持队列和超时）
   */
  public async send(message: Message): Promise<boolean> {
    return new Promise((resolve, reject) => {
      // 如果已连接，直接发送
      if (this.status === ConnectionStatus.Connected && this.ws) {
        this.sendDirect(message, resolve, reject);
        return;
      }

      // 未连接时，加入队列等待
      if (this.requestQueue.length >= this.requestQueueSize) {
        // 队列已满，移除最旧的请求
        const oldest = this.requestQueue.shift();
        if (oldest) {
          oldest.reject(new Error('请求队列已满，旧请求被丢弃'));
        }
      }

      // 加入队列
      this.requestQueue.push({
        message,
        resolve,
        reject,
        timestamp: Date.now(),
      });

      console.log(`请求已加入队列，当前队列长度: ${this.requestQueue.length}`);
    });
  }

  /**
   * 直接发送消息（内部方法）
   */
  private async sendDirect(
    message: Message,
    resolve: (success: boolean) => void,
    reject: (error: Error) => void
  ): Promise<void> {
    if (!this.ws) {
      reject(new Error('WebSocket未连接'));
      return;
    }

    try {
      // 序列化JSON → 压缩 → 加密 → 发送二进制数据
      const jsonString = JSON.stringify(message);
      const jsonBytes = new TextEncoder().encode(jsonString);
      
      // 处理发送消息：压缩 → 加密 → 发送
      const processedData = await this.processOutgoingMessage(jsonBytes);
      
      // 发送二进制数据
      this.ws.send(processedData);
      resolve(true);
    } catch (error) {
      console.error('发送消息失败:', error);
      reject(error as Error);
    }
  }

  /**
   * 处理发送消息：压缩 → 加密
   * 始终返回二进制数据，确保与服务器协议一致
   */
  private async processOutgoingMessage(data: Uint8Array): Promise<Uint8Array> {
    let processedData: Uint8Array = data;
    
    // 步骤1: 压缩
    if (this.compressUtil && this.compressUtil.isEnabled()) {
      try {
        processedData = await this.compressUtil.compress(processedData);
      } catch (error) {
        console.error('压缩失败:', error);
        throw error;
      }
    }
    
    // 步骤2: 加密
    if (this.cryptoUtil && this.cryptoUtil.isEnabled()) {
      try {
        processedData = await this.cryptoUtil.encrypt(processedData);
      } catch (error) {
        console.error('加密失败:', error);
        throw error;
      }
    }
    
    // 始终返回二进制数据，确保与服务器协议一致
    // 服务器期望接收二进制格式的 WebSocket 消息
    return processedData;
  }

  /**
   * 处理接收消息：解密 → 解压
   */
  private async processIncomingMessage(data: Uint8Array): Promise<Uint8Array> {
    let processedData: Uint8Array = data;
    
    // 步骤1: 解密
    if (this.cryptoUtil && this.cryptoUtil.isEnabled()) {
      try {
        processedData = await this.cryptoUtil.decrypt(processedData);
      } catch (error) {
        console.error('解密失败:', error);
        throw error;
      }
    }
    
    // 步骤2: 解压
    if (this.compressUtil && this.compressUtil.isEnabled()) {
      try {
        processedData = await this.compressUtil.decompress(processedData);
      } catch (error) {
        console.error('解压失败:', error);
        throw error;
      }
    }
    
    return processedData;
  }

  /**
   * 获取 WebSocket 关闭码的描述
   */
  private getCloseReasonDescription(code: number): string {
    const reasons: Record<number, string> = {
      1000: '正常关闭',
      1001: '端点离开（服务器关闭或浏览器离开页面）',
      1002: '协议错误',
      1003: '不支持的数据类型',
      1005: '未收到关闭帧',
      1006: '连接异常关闭（无关闭帧）',
      1007: '数据类型不一致',
      1008: '违反策略',
      1009: '消息过大',
      1010: '缺少扩展',
      1011: '内部错误',
      1012: '服务重启',
      1013: '稍后重试',
      1015: 'TLS握手失败'
    };
    return reasons[code] || `未知关闭码: ${code}`;
  }

  /**
   * 获取当前连接状态
   */
  public getStatus(): ConnectionStatus {
    return this.status;
  }

  /**
   * 注册状态变化回调
   */
  public onStatusChange(callback: StatusChangeCallback): void {
    this.statusChangeCallbacks.push(callback);
  }

  /**
   * 移除状态变化回调
   */
  public offStatusChange(callback: StatusChangeCallback): void {
    const index = this.statusChangeCallbacks.indexOf(callback);
    if (index > -1) {
      this.statusChangeCallbacks.splice(index, 1);
    }
  }

  /**
   * 注册消息接收回调
   */
  public onMessage(callback: MessageCallback): void {
    this.messageCallbacks.push(callback);
  }

  /**
   * 移除消息接收回调
   */
  public offMessage(callback: MessageCallback): void {
    const index = this.messageCallbacks.indexOf(callback);
    if (index > -1) {
      this.messageCallbacks.splice(index, 1);
    }
  }

  /**
   * 设置连接状态
   */
  private setStatus(status: ConnectionStatus): void {
    if (this.status !== status) {
      this.status = status;
      console.log('连接状态变化:', status);
      
      // 触发状态变化回调
      this.statusChangeCallbacks.forEach(callback => {
        try {
          callback(status);
        } catch (error) {
          console.error('状态变化回调执行错误:', error);
        }
      });
    }
  }

  /**
   * 启动心跳
   */
  private startHeartbeat(): void {
    this.stopHeartbeat();
    
    this.heartbeatTimer = setInterval(async () => {
      if (this.status === ConnectionStatus.Connected && this.ws) {
        const heartbeatMessage: Message = {
          id: `heartbeat_${Date.now()}`,
          type: 'ping',
          data: { timestamp: Date.now() }
        };
        await this.send(heartbeatMessage);
      }
    }, this.heartbeatInterval);
  }

  /**
   * 停止心跳
   */
  private stopHeartbeat(): void {
    if (this.heartbeatTimer !== null) {
      clearInterval(this.heartbeatTimer);
      this.heartbeatTimer = null;
    }
  }

  /**
   * 安排重连（指数退避）
   */
  private scheduleReconnect(): void {
    this.clearReconnectTimer();
    
    this.reconnectAttempts++;
    const delay = Math.min(
      this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1),
      this.maxReconnectDelay
    );

    console.log(`将在 ${delay}ms 后尝试第 ${this.reconnectAttempts} 次重连`);

    this.setStatus(ConnectionStatus.Reconnecting);

    this.reconnectTimer = setTimeout(() => {
      this.connect();
    }, delay);
  }

  /**
   * 清除重连定时器
   */
  private clearReconnectTimer(): void {
    if (this.reconnectTimer !== null) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
  }

  /**
   * 启动队列处理器（检查超时）
   */
  private startQueueProcessor(): void {
    this.stopQueueProcessor();
    
    this.queueCheckTimer = setInterval(() => {
      const now = Date.now();
      const timeoutThreshold = this.requestTimeout;
      
      // 检查队列中超时的请求
      this.requestQueue = this.requestQueue.filter(item => {
        if (now - item.timestamp > timeoutThreshold) {
          // 请求超时
          item.reject(new Error('请求超时'));
          return false;
        }
        return true;
      });
    }, 5000); // 每5秒检查一次
  }

  /**
   * 停止队列处理器
   */
  private stopQueueProcessor(): void {
    if (this.queueCheckTimer !== null) {
      clearInterval(this.queueCheckTimer);
      this.queueCheckTimer = null;
    }
  }

  /**
   * 刷新队列（发送队列中的消息）
   */
  private async flushQueue(): Promise<void> {
    if (this.status !== ConnectionStatus.Connected || !this.ws) {
      return;
    }

    while (this.requestQueue.length > 0) {
      const item = this.requestQueue.shift();
      if (!item) break;

      try {
        await this.sendDirect(item.message, item.resolve, item.reject);
      } catch (error) {
        item.reject(error as Error);
      }
    }
  }

  /**
   * 拒绝所有待处理的请求
   */
  private rejectAllPending(error: Error): void {
    while (this.requestQueue.length > 0) {
      const item = this.requestQueue.shift();
      if (item) {
        item.reject(error);
      }
    }
  }

  /**
   * 推送配置给服务端（加密+压缩）
   */
  private async sendCryptoKeyToServer(): Promise<void> {
    try {
      const data: any = {};

      // 加密配置
      if (this.cryptoUtil && this.cryptoUtil.isEnabled()) {
        const cryptoConfig = (this.cryptoUtil as any).config;
        data.crypto = {
          enabled: true,
          key: cryptoConfig.key,
          algorithm: cryptoConfig.algorithm
        };
      }

      // 压缩配置
      if (this.compressUtil) {
        const compressConfig = (this.compressUtil as any).config;
        data.compress = {
          enabled: compressConfig.enabled,
          algorithm: compressConfig.algorithm,
          level: compressConfig.level
        };
      }

      const msg: Message = {
        id: `update_config_${Date.now()}`,
        type: 'update_config',
        data
      };
      await this.send(msg);
      console.log('配置已推送给服务端:', data);
    } catch (error) {
      console.error('推送配置失败:', error);
    }
  }

  /**
   * 发送认证请求
   */
  public async sendAuth(username: string, password: string): Promise<AuthResult> {
    const msg: Message = {
      id: `auth_${Date.now()}`,
      type: 'auth',
      data: { username, password }
    };

    return new Promise((resolve) => {
      // 临时监听auth_result
      const handler = (message: Message) => {
        if (message.type === 'auth_result') {
          this.offMessage(handler);
          const data = message.data || {};
          if (data.success) {
            this.authenticated = true;
            this.authUsername = data.username || username;
            this.authIsAdmin = data.isAdmin || false;
          }
          resolve({
            success: data.success,
            isAdmin: data.isAdmin || false,
            username: data.username || username,
            token: data.token,
            message: data.message,
          });
        }
      };
      this.onMessage(handler);
      this.send(msg).catch(() => {
        this.offMessage(handler);
        resolve({ success: false, isAdmin: false, username, message: '发送认证请求失败' });
      });
    });
  }

  /**
   * 获取认证状态
   */
  public isAuthenticated(): boolean {
    return this.authenticated;
  }

  /**
   * 获取是否管理员
   */
  public isAdmin(): boolean {
    return this.authIsAdmin;
  }

  /**
   * 获取认证用户名
   */
  public getAuthUsername(): string {
    return this.authUsername;
  }

  /**
   * 获取队列长度
   */
  public getQueueLength(): number {
    return this.requestQueue.length;
  }

  /**
   * 设置请求超时时间
   */
  public setRequestTimeout(timeout: number): void {
    this.requestTimeout = timeout;
  }

  /**
   * 设置请求队列大小
   */
  public setRequestQueueSize(size: number): void {
    this.requestQueueSize = size;
  }
}
