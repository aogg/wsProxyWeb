// 存储工具库

// 客户端配置接口
export interface ClientConfig {
  websocketUrl: string;
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

  /**
   * 保存客户端配置
   */
  static async saveConfig(config: Partial<ClientConfig>): Promise<void> {
    try {
      // 读取现有配置
      const existing = await this.getConfig();
      const merged = { ...existing, ...config };
      
      await chrome.storage.local.set({
        [this.KEY_CONFIG]: merged
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
      const result = await chrome.storage.local.get(this.KEY_CONFIG);
      return result[this.KEY_CONFIG] || {
        websocketUrl: 'ws://localhost:8080/ws',
        crypto: {
          enabled: false,
          key: '',
          algorithm: 'aes256gcm'
        },
        compress: {
          enabled: false,
          level: 6,
          algorithm: 'gzip'
        }
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
      
      await chrome.storage.local.set({
        [this.KEY_RULES]: merged
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
      const result = await chrome.storage.local.get(this.KEY_RULES);
      return result[this.KEY_RULES] || {
        enabled: true,
        whitelist: [],
        blacklist: [],
        urlPatterns: []
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
      const result = await chrome.storage.local.get([
        this.KEY_CONNECTION_STATUS,
        this.KEY_CONNECTION_TIME
      ]);
      
      if (result[this.KEY_CONNECTION_STATUS]) {
        return {
          status: result[this.KEY_CONNECTION_STATUS],
          time: result[this.KEY_CONNECTION_TIME] || 0
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
        callback(changes[this.KEY_CONFIG].newValue);
      }
    });
  }

  /**
   * 监听连接状态变化
   */
  static onStatusChange(callback: (status: string, time: number) => void): void {
    chrome.storage.onChanged.addListener((changes, areaName) => {
      if (areaName === 'local') {
        if (changes[this.KEY_CONNECTION_STATUS]) {
          const status = changes[this.KEY_CONNECTION_STATUS].newValue;
          const time = changes[this.KEY_CONNECTION_TIME]?.newValue || Date.now();
          callback(status, time);
        }
      }
    });
  }
}
