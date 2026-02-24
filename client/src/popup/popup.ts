// 弹窗界面脚本

import { StorageUtil, ClientConfig, RuleConfig } from '../libs/storage';

// 界面元素
let websocketUrlInput: HTMLInputElement;
let cryptoEnabledCheckbox: HTMLInputElement;
let cryptoKeyInput: HTMLInputElement;
let cryptoAlgorithmSelect: HTMLSelectElement;
let compressEnabledCheckbox: HTMLInputElement;
let compressLevelInput: HTMLInputElement;
let compressLevelValue: HTMLElement;
let compressAlgorithmSelect: HTMLSelectElement;
let rulesEnabledCheckbox: HTMLInputElement;
let whitelistTextarea: HTMLTextAreaElement;
let blacklistTextarea: HTMLTextAreaElement;
let urlPatternsTextarea: HTMLTextAreaElement;
let saveConfigBtn: HTMLButtonElement;
let startBtn: HTMLButtonElement;
let stopBtn: HTMLButtonElement;
let reconnectBtn: HTMLButtonElement;
let resetBtn: HTMLButtonElement;
let statusDot: HTMLElement;
let statusText: HTMLElement;
let connectionTime: HTMLElement;

// 初始化界面
async function init(): Promise<void> {
  // 获取所有界面元素
  websocketUrlInput = document.getElementById('websocketUrl') as HTMLInputElement;
  cryptoEnabledCheckbox = document.getElementById('cryptoEnabled') as HTMLInputElement;
  cryptoKeyInput = document.getElementById('cryptoKey') as HTMLInputElement;
  cryptoAlgorithmSelect = document.getElementById('cryptoAlgorithm') as HTMLSelectElement;
  compressEnabledCheckbox = document.getElementById('compressEnabled') as HTMLInputElement;
  compressLevelInput = document.getElementById('compressLevel') as HTMLInputElement;
  compressLevelValue = document.getElementById('compressLevelValue') as HTMLElement;
  compressAlgorithmSelect = document.getElementById('compressAlgorithm') as HTMLSelectElement;
  rulesEnabledCheckbox = document.getElementById('rulesEnabled') as HTMLInputElement;
  whitelistTextarea = document.getElementById('whitelist') as HTMLTextAreaElement;
  blacklistTextarea = document.getElementById('blacklist') as HTMLTextAreaElement;
  urlPatternsTextarea = document.getElementById('urlPatterns') as HTMLTextAreaElement;
  saveConfigBtn = document.getElementById('saveConfigBtn') as HTMLButtonElement;
  startBtn = document.getElementById('startBtn') as HTMLButtonElement;
  stopBtn = document.getElementById('stopBtn') as HTMLButtonElement;
  reconnectBtn = document.getElementById('reconnectBtn') as HTMLButtonElement;
  resetBtn = document.getElementById('resetBtn') as HTMLButtonElement;
  statusDot = document.getElementById('statusDot') as HTMLElement;
  statusText = document.getElementById('statusText') as HTMLElement;
  connectionTime = document.getElementById('connectionTime') as HTMLElement;

  // 加载配置
  await loadConfig();
  await loadRules();
  await updateConnectionStatus();

  // 绑定事件
  bindEvents();

  // 监听状态变化
  StorageUtil.onStatusChange((status, time) => {
    updateStatusDisplay(status, time);
  });
}

// 加载配置
async function loadConfig(): Promise<void> {
  try {
    const config = await StorageUtil.getConfig();
    
    // 填充表单
    websocketUrlInput.value = config.websocketUrl || 'ws://localhost:8080/ws';
    
    // 更新启动/停止按钮状态
    updateProxyButtonState(config.proxyEnabled || false);
    
    if (config.crypto) {
      cryptoEnabledCheckbox.checked = config.crypto.enabled || false;
      cryptoKeyInput.value = config.crypto.key || '';
      cryptoAlgorithmSelect.value = config.crypto.algorithm || 'aes256gcm';
      toggleCryptoConfig();
    }
    
    if (config.compress) {
      compressEnabledCheckbox.checked = config.compress.enabled || false;
      compressLevelInput.value = String(config.compress.level || 6);
      compressLevelValue.textContent = String(config.compress.level || 6);
      compressAlgorithmSelect.value = config.compress.algorithm || 'gzip';
      toggleCompressConfig();
    }
  } catch (error) {
    console.error('加载配置失败:', error);
    showMessage('加载配置失败', 'error');
  }
}

// 更新启动/停止按钮显示状态
function updateProxyButtonState(enabled: boolean): void {
  if (enabled) {
    startBtn.style.display = 'none';
    stopBtn.style.display = 'block';
  } else {
    startBtn.style.display = 'block';
    stopBtn.style.display = 'none';
  }
}

// 加载规则配置
async function loadRules(): Promise<void> {
  try {
    const rules = await StorageUtil.getRules();
    
    rulesEnabledCheckbox.checked = rules.enabled || false;
    whitelistTextarea.value = rules.whitelist?.join('\n') || '';
    blacklistTextarea.value = rules.blacklist?.join('\n') || '';
    urlPatternsTextarea.value = rules.urlPatterns?.join('\n') || '';
    
    toggleRulesConfig();
  } catch (error) {
    console.error('加载规则配置失败:', error);
    showMessage('加载规则配置失败', 'error');
  }
}

// 更新连接状态
async function updateConnectionStatus(): Promise<void> {
  try {
    const statusInfo = await StorageUtil.getConnectionStatus();
    if (statusInfo) {
      updateStatusDisplay(statusInfo.status, statusInfo.time);
    } else {
      updateStatusDisplay('disconnected', 0);
    }
  } catch (error) {
    console.error('获取连接状态失败:', error);
  }
}

// 更新状态显示
function updateStatusDisplay(status: string, time: number): void {
  // 更新状态文本和颜色
  const statusMap: Record<string, { text: string; color: string }> = {
    'disconnected': { text: '未连接', color: '#999' },
    'connecting': { text: '连接中', color: '#ffa500' },
    'connected': { text: '已连接', color: '#4caf50' },
    'reconnecting': { text: '重连中', color: '#ffa500' },
    'error': { text: '连接错误', color: '#f44336' }
  };

  const statusInfo = statusMap[status] || statusMap['disconnected'];
  statusText.textContent = statusInfo.text;
  statusDot.style.backgroundColor = statusInfo.color;

  // 更新连接时间
  if (time > 0) {
    const date = new Date(time);
    connectionTime.textContent = date.toLocaleString('zh-CN');
  } else {
    connectionTime.textContent = '-';
  }
}

// 绑定事件
function bindEvents(): void {
  // 保存配置
  saveConfigBtn.addEventListener('click', async () => {
    await saveConfig();
  });

  // 启动代理
  startBtn.addEventListener('click', async () => {
    await startProxy();
  });

  // 停止代理
  stopBtn.addEventListener('click', async () => {
    await stopProxy();
  });

  // 重新连接
  reconnectBtn.addEventListener('click', async () => {
    await reconnect();
  });

  // 重置配置
  resetBtn.addEventListener('click', async () => {
    if (confirm('确定要重置所有配置吗？')) {
      await resetConfig();
    }
  });

  // 加密配置切换
  cryptoEnabledCheckbox.addEventListener('change', toggleCryptoConfig);

  // 压缩配置切换
  compressEnabledCheckbox.addEventListener('change', toggleCompressConfig);

  // 规则配置切换
  rulesEnabledCheckbox.addEventListener('change', toggleRulesConfig);

  // 压缩级别显示
  compressLevelInput.addEventListener('input', () => {
    compressLevelValue.textContent = compressLevelInput.value;
  });
}

// 切换加密配置显示
function toggleCryptoConfig(): void {
  const cryptoConfigGroup = document.getElementById('cryptoConfigGroup') as HTMLElement;
  if (cryptoEnabledCheckbox.checked) {
    cryptoConfigGroup.style.display = 'block';
  } else {
    cryptoConfigGroup.style.display = 'none';
  }
}

// 切换压缩配置显示
function toggleCompressConfig(): void {
  const compressConfigGroup = document.getElementById('compressConfigGroup') as HTMLElement;
  if (compressEnabledCheckbox.checked) {
    compressConfigGroup.style.display = 'block';
  } else {
    compressConfigGroup.style.display = 'none';
  }
}

// 切换规则配置显示
function toggleRulesConfig(): void {
  const rulesConfigGroup = document.getElementById('rulesConfigGroup') as HTMLElement;
  if (rulesEnabledCheckbox.checked) {
    rulesConfigGroup.style.display = 'block';
  } else {
    rulesConfigGroup.style.display = 'none';
  }
}

// 保存配置
async function saveConfig(): Promise<void> {
  try {
    // 收集配置
    const config: Partial<ClientConfig> = {
      websocketUrl: websocketUrlInput.value.trim(),
      crypto: {
        enabled: cryptoEnabledCheckbox.checked,
        key: cryptoKeyInput.value.trim(),
        algorithm: cryptoAlgorithmSelect.value
      },
      compress: {
        enabled: compressEnabledCheckbox.checked,
        level: parseInt(compressLevelInput.value),
        algorithm: compressAlgorithmSelect.value
      }
    };

    // 保存配置
    await StorageUtil.saveConfig(config);

    // 保存规则配置
    const rules: Partial<RuleConfig> = {
      enabled: rulesEnabledCheckbox.checked,
      whitelist: whitelistTextarea.value.split('\n').map(s => s.trim()).filter(s => s),
      blacklist: blacklistTextarea.value.split('\n').map(s => s.trim()).filter(s => s),
      urlPatterns: urlPatternsTextarea.value.split('\n').map(s => s.trim()).filter(s => s)
    };
    await StorageUtil.saveRules(rules);

    showMessage('配置已保存', 'success');

    // 通知background重新初始化
    chrome.runtime.sendMessage({ type: 'reloadConfig' }).catch(err => {
      console.error('通知background失败:', err);
    });
  } catch (error) {
    console.error('保存配置失败:', error);
    showMessage('保存配置失败', 'error');
  }
}

// 重新连接
async function reconnect(): Promise<void> {
  try {
    showMessage('正在重新连接...', 'info');
    
    // 通知background重新连接
    chrome.runtime.sendMessage({ type: 'reconnect' }).catch(err => {
      console.error('通知background失败:', err);
      showMessage('重新连接失败', 'error');
    });
  } catch (error) {
    console.error('重新连接失败:', error);
    showMessage('重新连接失败', 'error');
  }
}

// 启动代理
async function startProxy(): Promise<void> {
  try {
    showMessage('正在启动代理...', 'info');
    
    // 通知background启动代理
    const response = await chrome.runtime.sendMessage({ type: 'startProxy' });
    
    if (response && response.success) {
      // 保存启用状态
      const config = await StorageUtil.getConfig();
      await StorageUtil.saveConfig({ ...config, proxyEnabled: true });
      
      updateProxyButtonState(true);
      showMessage('代理已启动', 'success');
    } else {
      showMessage(response?.error || '启动代理失败', 'error');
    }
  } catch (error) {
    console.error('启动代理失败:', error);
    showMessage('启动代理失败', 'error');
  }
}

// 停止代理
async function stopProxy(): Promise<void> {
  try {
    showMessage('正在停止代理...', 'info');
    
    // 通知background停止代理
    const response = await chrome.runtime.sendMessage({ type: 'stopProxy' });
    
    if (response && response.success) {
      // 保存禁用状态
      const config = await StorageUtil.getConfig();
      await StorageUtil.saveConfig({ ...config, proxyEnabled: false });
      
      updateProxyButtonState(false);
      showMessage('代理已停止', 'success');
    } else {
      showMessage(response?.error || '停止代理失败', 'error');
    }
  } catch (error) {
    console.error('停止代理失败:', error);
    showMessage('停止代理失败', 'error');
  }
}

// 重置配置
async function resetConfig(): Promise<void> {
  try {
    // 重置为默认值
    const defaultConfig: ClientConfig = {
      websocketUrl: 'ws://localhost:8080/ws',
      proxyEnabled: false,
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

    const defaultRules: RuleConfig = {
      enabled: true,
      whitelist: [],
      blacklist: [],
      urlPatterns: []
    };

    await StorageUtil.saveConfig(defaultConfig);
    await StorageUtil.saveRules(defaultRules);

    // 更新按钮状态
    updateProxyButtonState(false);

    // 重新加载配置
    await loadConfig();
    await loadRules();

    showMessage('配置已重置', 'success');

    // 通知background重新初始化
    chrome.runtime.sendMessage({ type: 'reloadConfig' }).catch(err => {
      console.error('通知background失败:', err);
    });
  } catch (error) {
    console.error('重置配置失败:', error);
    showMessage('重置配置失败', 'error');
  }
}

// 显示消息
function showMessage(message: string, type: 'success' | 'error' | 'info'): void {
  // 创建消息元素
  const messageEl = document.createElement('div');
  messageEl.className = `message message-${type}`;
  messageEl.textContent = message;
  
  // 添加到页面
  document.body.appendChild(messageEl);
  
  // 3秒后移除
  setTimeout(() => {
    messageEl.remove();
  }, 3000);
}

// 页面加载完成后初始化
document.addEventListener('DOMContentLoaded', init);
