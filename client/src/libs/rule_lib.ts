// 规则匹配工具库（高封装，供background调用）

import { RuleConfig } from './storage';

export interface RuleMatchResult {
  shouldProxy: boolean;
  reason: string;
}

/**
 * RuleLib 规则匹配类
 *
 * 规则语义（默认偏“安全”）：
 * - enabled=false：不代理任何请求
 * - blacklist 命中：永不代理（优先级最高）
 * - whitelist 非空：仅当命中白名单才代理（用于“只代理某些域名”）
 * - urlPatterns 非空：仅当命中 URL 模式才代理（用于更细粒度的 URL 过滤）
 *
 * 匹配支持通配符：
 * - 域名：`example.com`（含子域名）、`*.example.com`、`api-*.example.com`
 * - URL：`*://*.example.com/api/*`，无通配符时默认使用“包含”匹配（更符合输入习惯）
 */
export class RuleLib {
  private rules: RuleConfig;
  private whitelistPatterns: string[];
  private blacklistPatterns: string[];
  private urlPatterns: string[];

  constructor(rules: RuleConfig) {
    this.rules = RuleLib.normalizeRules(rules);
    this.whitelistPatterns = RuleLib.normalizePatterns(this.rules.whitelist);
    this.blacklistPatterns = RuleLib.normalizePatterns(this.rules.blacklist);
    this.urlPatterns = RuleLib.normalizePatterns(this.rules.urlPatterns);
  }

  /**
   * 判断某个URL是否应当走代理
   */
  public shouldProxy(urlStr: string): RuleMatchResult {
    if (!this.rules.enabled) {
      return { shouldProxy: false, reason: '规则未启用' };
    }

    if (!urlStr) {
      return { shouldProxy: false, reason: 'URL为空' };
    }

    let urlObj: URL;
    try {
      urlObj = new URL(urlStr);
    } catch {
      return { shouldProxy: false, reason: 'URL解析失败' };
    }

    // 只处理http/https，其他协议（data/blob/file/ws等）一律不代理
    const protocol = urlObj.protocol.toLowerCase();
    if (protocol !== 'http:' && protocol !== 'https:') {
      return { shouldProxy: false, reason: `不支持的协议: ${protocol}` };
    }

    const host = (urlObj.hostname || '').toLowerCase();
    if (!host) {
      return { shouldProxy: false, reason: '域名为空' };
    }

    // 黑名单优先
    if (this.blacklistPatterns.length > 0 && this.matchHostList(this.blacklistPatterns, host)) {
      return { shouldProxy: false, reason: '命中黑名单' };
    }

    // 白名单（非空则必须命中）
    if (this.whitelistPatterns.length > 0 && !this.matchHostList(this.whitelistPatterns, host)) {
      return { shouldProxy: false, reason: '未命中白名单' };
    }

    // URL 模式（非空则必须命中）
    if (this.urlPatterns.length > 0 && !this.matchUrlList(this.urlPatterns, urlStr)) {
      return { shouldProxy: false, reason: '未命中URL模式' };
    }

    return { shouldProxy: true, reason: '规则允许' };
  }

  private matchHostList(patterns: string[], host: string): boolean {
    for (const pattern of patterns) {
      if (RuleLib.matchHost(pattern, host)) {
        return true;
      }
    }
    return false;
  }

  private matchUrlList(patterns: string[], urlStr: string): boolean {
    for (const pattern of patterns) {
      if (RuleLib.matchUrl(pattern, urlStr)) {
        return true;
      }
    }
    return false;
  }

  /**
   * 域名匹配：
   * - `example.com`：匹配 `example.com` 以及其所有子域名（如 `a.example.com`）
   * - `*.example.com`：同上（显式子域名通配）
   * - 含通配符 `*`/`?`：按glob匹配host
   */
  public static matchHost(pattern: string, host: string): boolean {
    const p = (pattern || '').trim().toLowerCase();
    const h = (host || '').trim().toLowerCase();
    if (!p || !h) return false;

    // 显式子域名通配：*.example.com
    if (p.startsWith('*.') && p.length > 2) {
      const suffix = p.slice(2);
      return h === suffix || h.endsWith(`.${suffix}`);
    }

    // 通配符glob
    if (p.includes('*') || p.includes('?')) {
      const re = RuleLib.globToRegExp(p);
      return re.test(h);
    }

    // 无通配符：默认匹配根域及其子域名
    return h === p || h.endsWith(`.${p}`);
  }

  /**
   * URL模式匹配：
   * - 含通配符 `*`/`?`：按glob匹配整条URL
   * - 不含通配符：使用“包含”匹配（更符合手工输入）
   */
  public static matchUrl(pattern: string, urlStr: string): boolean {
    const p = (pattern || '').trim();
    const u = (urlStr || '').trim();
    if (!p || !u) return false;

    if (p.includes('*') || p.includes('?')) {
      const re = RuleLib.globToRegExp(p);
      return re.test(u);
    }

    return u.includes(p);
  }

  /**
   * glob转正则：仅支持 `*` 与 `?`，并对其他正则特殊字符做转义
   */
  private static globToRegExp(glob: string): RegExp {
    const escaped = glob
      .replace(/[.+^${}()|[\]\\]/g, '\\$&')
      .replace(/\*/g, '.*')
      .replace(/\?/g, '.');
    return new RegExp(`^${escaped}$`, 'i');
  }

  private static normalizePatterns(list: string[] | undefined | null): string[] {
    if (!Array.isArray(list)) return [];
    return list
      .map((s) => (typeof s === 'string' ? s.trim() : ''))
      .filter((s) => s.length > 0)
      .filter((s) => !s.startsWith('#'));
  }

  private static normalizeRules(rules: RuleConfig): RuleConfig {
    return {
      enabled: !!rules?.enabled,
      whitelist: Array.isArray(rules?.whitelist) ? rules.whitelist : [],
      blacklist: Array.isArray(rules?.blacklist) ? rules.blacklist : [],
      urlPatterns: Array.isArray(rules?.urlPatterns) ? rules.urlPatterns : [],
    };
  }
}

