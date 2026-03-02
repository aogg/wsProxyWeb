// 存储工具库

// 客户端配置接口
export interface ClientConfig {
  websocketUrl: string;
  proxyEnabled?: boolean; // 代理是否启用
  crypto?: {
    enabled: boolean;
    key: string;
    algorithm: string;
  };
  compress?: {
    enabled: boolean;
    level: number;
    algorithm: string;
  };
  auth?: {
    username: string;
    password: string;
  };
}

// 认证状态接口
export interface AuthState {
  authenticated: boolean;
  username: string;
  isAdmin: boolean;
  token: string;
}

// 拦截规则配置接口
export interface RuleConfig {
  enabled: boolean;
  whitelist: string[]; // 域名白名单
  blacklist: string[]; // 域名黑名单
  urlPatterns: string[]; // URL模式匹配
}

// 存储工具类
export class StorageUtil {
  // 存储键名常量
  private static readonly KEY_CONFIG = 'clientConfig';
  private static readonly KEY_RULES = 'ruleConfig';
  private static readonly KEY_CONNECTION_STATUS = 'wsConnectionStatus';
  private static readonly KEY_CONNECTION_TIME = 'wsConnectionTime';
  private static readonly KEY_AUTH_STATE = 'authState';

  /**
   * 从chrome.storage.local读取数据的Promise封装
   */
  private static getFromStorage<T>(
    keys: string | string[],
  ): Promise<Record<string, T>> {
    return new Promise((resolve, reject) => {
      try {
        chrome.storage.local.get(keys, (items) => {
          const error = chrome.runtime.lastError;
          if (error) {
            reject(error);
            return;
          }
          resolve(items as Record<string, T>);
        });
      } catch (error) {
        reject(error);
      }
    });
  }

  /**
   * 写入chrome.storage.local的Promise封装
   */
  private static setToStorage(items: Record<string, unknown>): Promise<void> {
    return new Promise((resolve, reject) => {
      try {
        chrome.storage.local.set(items, () => {
          const error = chrome.runtime.lastError;
          if (error) {
            reject(error);
            return;
          }
          resolve();
        });
      } catch (error) {
        reject(error);
      }
    });
  }

  /**
   * 保存客户端配置
   */
  static async saveConfig(config: Partial<ClientConfig>): Promise<void> {
    try {
      // 读取现有配置
      const existing = await this.getConfig();
      const merged = { ...existing, ...config };
      
      await this.setToStorage({
        [this.KEY_CONFIG]: merged,
      });
    } catch (error) {
      console.error('保存配置失败:', error);
      throw error;
    }
  }

  /**
   * 获取客户端配置
   */
  static async getConfig(): Promise<ClientConfig> {
    try {
      const result = await this.getFromStorage<ClientConfig>(this.KEY_CONFIG);
      const stored = result[this.KEY_CONFIG];
      if (stored) {
        return stored;
      }
      // 默认配置
      return {
        websocketUrl: 'ws://localhost:8080/ws',
        proxyEnabled: false,
        crypto: {
          enabled: false,
          key: '',
          algorithm: 'aes256gcm',
        },
        compress: {
          enabled: false,
          level: 6,
          algorithm: 'gzip',
        },
      };
    } catch (error) {
      console.error('读取配置失败:', error);
      throw error;
    }
  }

  /**
   * 保存拦截规则配置
   */
  static async saveRules(rules: Partial<RuleConfig>): Promise<void> {
    try {
      const existing = await this.getRules();
      const merged = { ...existing, ...rules };
      
      await this.setToStorage({
        [this.KEY_RULES]: merged,
      });
    } catch (error) {
      console.error('保存规则配置失败:', error);
      throw error;
    }
  }

  /**
   * 获取拦截规则配置
   */
  static async getRules(): Promise<RuleConfig> {
    try {
      const result = await this.getFromStorage<RuleConfig>(this.KEY_RULES);
      const stored = result[this.KEY_RULES];
      if (stored) {
        return stored;
      }
      // 默认规则配置
      return {
        enabled: true,
        whitelist: [],
        blacklist: [],
        urlPatterns: [],
      };
    } catch (error) {
      console.error('读取规则配置失败:', error);
      throw error;
    }
  }

  /**
   * 获取连接状态
   */
  static async getConnectionStatus(): Promise<{ status: string; time: number } | null> {
    try {
      const result = await this.getFromStorage<string | number>([
        this.KEY_CONNECTION_STATUS,
        this.KEY_CONNECTION_TIME,
      ]);
      
      if (result[this.KEY_CONNECTION_STATUS]) {
        return {
          status: String(result[this.KEY_CONNECTION_STATUS]),
          time: Number(result[this.KEY_CONNECTION_TIME] || 0),
        };
      }
      return null;
    } catch (error) {
      console.error('读取连接状态失败:', error);
      return null;
    }
  }

  /**
   * 监听配置变化
   */
  static onConfigChange(callback: (config: ClientConfig) => void): void {
    chrome.storage.onChanged.addListener((changes, areaName) => {
      if (areaName === 'local' && changes[this.KEY_CONFIG]) {
        callback(changes[this.KEY_CONFIG].newValue as ClientConfig);
      }
    });
  }

  /**
   * 监听规则配置变化
   */
  static onRulesChange(callback: (rules: RuleConfig) => void): void {
    chrome.storage.onChanged.addListener((changes, areaName) => {
      if (areaName === 'local' && changes[this.KEY_RULES]) {
        callback(changes[this.KEY_RULES].newValue as RuleConfig);
      }
    });
  }

  /**
   * 保存认证状态
   */
  static async saveAuthState(state: AuthState): Promise<void> {
    await this.setToStorage({ [this.KEY_AUTH_STATE]: state });
  }

  /**
   * 获取认证状态
   */
  static async getAuthState(): Promise<AuthState | null> {
    const result = await this.getFromStorage<AuthState>(this.KEY_AUTH_STATE);
    return result[this.KEY_AUTH_STATE] || null;
  }

  /**
   * 清除认证状态
   */
  static async clearAuthState(): Promise<void> {
    await this.setToStorage({ [this.KEY_AUTH_STATE]: null });
  }

  /**
   * 监听认证状态变化
   */
  static onAuthStateChange(callback: (state: AuthState | null) => void): void {
    chrome.storage.onChanged.addListener((changes, areaName) => {
      if (areaName === 'local' && changes[this.KEY_AUTH_STATE]) {
        callback(changes[this.KEY_AUTH_STATE].newValue as AuthState | null);
      }
    });
  }

  /**
   * 监听连接状态变化
   */
  static onStatusChange(callback: (status: string, time: number) => void): void {
    chrome.storage.onChanged.addListener((changes, areaName) => {
      if (areaName === 'local' && changes[this.KEY_CONNECTION_STATUS]) {
        const rawStatus = changes[this.KEY_CONNECTION_STATUS].newValue;
        const rawTime = changes[this.KEY_CONNECTION_TIME]?.newValue;
        const status = String(rawStatus ?? 'disconnected');
        const time = typeof rawTime === 'number' ? rawTime : Date.now();
        callback(status, time);
      }
    });
  }
}
