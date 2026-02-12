// WebSocket工具库

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

  constructor(url: string) {
    this.url = url;
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

      this.ws.onmessage = (event) => {
        try {
          const message: Message = JSON.parse(event.data);
          
          // 处理心跳响应
          if (message.type === 'pong' || message.type === 'heartbeat') {
            return;
          }

          // 触发消息回调
          this.messageCallbacks.forEach(callback => {
            try {
              callback(message);
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
  public send(message: Message): boolean {
    if (this.status !== ConnectionStatus.Connected || !this.ws) {
      console.warn('WebSocket未连接，无法发送消息');
      return false;
    }

    try {
      const json = JSON.stringify(message);
      this.ws.send(json);
      return true;
    } catch (error) {
      console.error('发送消息失败:', error);
      return false;
    }
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
    
    this.heartbeatTimer = setInterval(() => {
      if (this.status === ConnectionStatus.Connected && this.ws) {
        const heartbeatMessage: Message = {
          id: `heartbeat_${Date.now()}`,
          type: 'ping',
          data: { timestamp: Date.now() }
        };
        this.send(heartbeatMessage);
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
