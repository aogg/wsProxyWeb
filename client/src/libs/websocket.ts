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

// 连接状态变化回调
export type StatusChangeCallback = (status: ConnectionStatus) => void;

// 消息接收回调
export type MessageCallback = (message: Message) => void;

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
        console.log('WebSocket连接关闭:', event.code, event.reason);
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
    this.clearReconnectTimer();

    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }

    this.setStatus(ConnectionStatus.Disconnected);
  }

  /**
   * 发送消息
   */
  public async send(message: Message): Promise<boolean> {
    if (this.status !== ConnectionStatus.Connected || !this.ws) {
      console.warn('WebSocket未连接，无法发送消息');
      return false;
    }

    try {
      // 序列化JSON → 压缩 → 加密 → 发送二进制数据
      const jsonString = JSON.stringify(message);
      const jsonBytes = new TextEncoder().encode(jsonString);
      
      // 处理发送消息：压缩 → 加密 → 发送
      const processedData = await this.processOutgoingMessage(jsonBytes);
      
      // 发送二进制数据
      this.ws.send(processedData);
      return true;
    } catch (error) {
      console.error('发送消息失败:', error);
      return false;
    }
  }

  /**
   * 处理发送消息：压缩 → 加密
   */
  private async processOutgoingMessage(data: Uint8Array): Promise<Uint8Array | string> {
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
    
    // 如果未启用加密和压缩，返回原始文本数据（兼容性）
    if (!this.cryptoUtil?.isEnabled() && !this.compressUtil?.isEnabled()) {
      return new TextDecoder().decode(data);
    }
    
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
}
