// 弹窗界面脚本

import { StorageUtil, ClientConfig, RuleConfig, AuthState } from '../libs/storage';
import { DEFAULT_CONFIG } from '../configs/defaults';

// 界面元素
let websocketUrlInput: HTMLInputElement;
let cryptoEnabledCheckbox: HTMLInputElement;
let cryptoKeyInput: HTMLInputElement;
let cryptoAlgorithmSelect: HTMLSelectElement;
let regenerateKeyBtn: HTMLButtonElement;
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
let connectBtn: HTMLButtonElement;
let disconnectBtn: HTMLButtonElement;
let reconnectBtn: HTMLButtonElement;
let resetBtn: HTMLButtonElement;
let statusDot: HTMLElement;
let statusText: HTMLElement;
let connectionTime: HTMLElement;
let versionSpan: HTMLElement;

// 认证相关元素
let authUsernameInput: HTMLInputElement;
let authPasswordInput: HTMLInputElement;
let loginBtn: HTMLButtonElement;
let loginForm: HTMLElement;
let authInfo: HTMLElement;
let authUserDisplay: HTMLElement;
let adminBadge: HTMLElement;
let logoutBtn: HTMLButtonElement;
let changePwdToggle: HTMLButtonElement;
let userManageToggle: HTMLButtonElement;

// 初始化界面
async function init(): Promise<void> {
  // 获取所有界面元素
  websocketUrlInput = document.getElementById('websocketUrl') as HTMLInputElement;
  websocketUrlInput.placeholder = DEFAULT_CONFIG.websocketUrl;
  cryptoEnabledCheckbox = document.getElementById('cryptoEnabled') as HTMLInputElement;
  cryptoKeyInput = document.getElementById('cryptoKey') as HTMLInputElement;
  cryptoAlgorithmSelect = document.getElementById('cryptoAlgorithm') as HTMLSelectElement;
  regenerateKeyBtn = document.getElementById('regenerateKeyBtn') as HTMLButtonElement;
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
  connectBtn = document.getElementById('connectBtn') as HTMLButtonElement;
  disconnectBtn = document.getElementById('disconnectBtn') as HTMLButtonElement;
  reconnectBtn = document.getElementById('reconnectBtn') as HTMLButtonElement;
  resetBtn = document.getElementById('resetBtn') as HTMLButtonElement;
  statusDot = document.getElementById('statusDot') as HTMLElement;
  statusText = document.getElementById('statusText') as HTMLElement;
  connectionTime = document.getElementById('connectionTime') as HTMLElement;
  versionSpan = document.getElementById('version') as HTMLElement;

  // 认证相关元素
  authUsernameInput = document.getElementById('authUsername') as HTMLInputElement;
  authPasswordInput = document.getElementById('authPassword') as HTMLInputElement;
  loginBtn = document.getElementById('loginBtn') as HTMLButtonElement;
  loginForm = document.getElementById('loginForm') as HTMLElement;
  authInfo = document.getElementById('authInfo') as HTMLElement;
  authUserDisplay = document.getElementById('authUserDisplay') as HTMLElement;
  adminBadge = document.getElementById('adminBadge') as HTMLElement;
  logoutBtn = document.getElementById('logoutBtn') as HTMLButtonElement;
  changePwdToggle = document.getElementById('changePwdToggle') as HTMLButtonElement;
  userManageToggle = document.getElementById('userManageToggle') as HTMLButtonElement;

  // 加载配置
  await loadConfig();
  await loadRules();
  await updateConnectionStatus();
  await loadAuthState();

  // 显示版本号
  versionSpan.textContent = chrome.runtime.getManifest().version;

  // 绑定事件
  bindEvents();

  // 监听状态变化
  StorageUtil.onStatusChange((status, time) => {
    updateStatusDisplay(status, time);
  });

  // 监听认证状态变化
  StorageUtil.onAuthStateChange((state) => {
    updateAuthDisplay(state);
  });
}

// 加载配置
async function loadConfig(): Promise<void> {
  try {
    const config = await StorageUtil.getConfig();
    
    // 填充表单
    websocketUrlInput.value = config.websocketUrl || DEFAULT_CONFIG.websocketUrl;
    
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

// 更新连接按钮显示状态
function updateConnectButtonState(connected: boolean): void {
  if (connected) {
    connectBtn.style.display = 'none';
    disconnectBtn.style.display = 'block';
  } else {
    connectBtn.style.display = 'block';
    disconnectBtn.style.display = 'none';
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

  // 更新连接按钮状态（已连接时显示停止连接按钮）
  updateConnectButtonState(status === 'connected');

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

  // 启用连接
  connectBtn.addEventListener('click', async () => {
    await connect();
  });

  // 停止连接
  disconnectBtn.addEventListener('click', async () => {
    await disconnect();
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

  // 重新生成密钥
  regenerateKeyBtn.addEventListener('click', generateCryptoKey);

  // 登录
  loginBtn.addEventListener('click', handleLogin);

  // 退出登录
  logoutBtn.addEventListener('click', handleLogout);

  // 修改密码切换
  changePwdToggle.addEventListener('click', () => {
    const section = document.getElementById('changePwdSection') as HTMLElement;
    section.style.display = section.style.display === 'none' ? 'block' : 'none';
  });

  // 修改密码确认
  document.getElementById('changePwdBtn')?.addEventListener('click', handleChangePassword);
  document.getElementById('changePwdCancel')?.addEventListener('click', () => {
    (document.getElementById('changePwdSection') as HTMLElement).style.display = 'none';
  });

  // 用户管理切换
  userManageToggle.addEventListener('click', () => {
    const section = document.getElementById('userManageSection') as HTMLElement;
    if (section.style.display === 'none') {
      section.style.display = 'block';
      loadUserList();
    } else {
      section.style.display = 'none';
    }
  });

  // 添加用户
  document.getElementById('addUserBtn')?.addEventListener('click', handleAddUser);
  document.getElementById('userManageClose')?.addEventListener('click', () => {
    (document.getElementById('userManageSection') as HTMLElement).style.display = 'none';
  });
}

// 切换加密配置显示
function toggleCryptoConfig(): void {
  const cryptoConfigGroup = document.getElementById('cryptoConfigGroup') as HTMLElement;
  if (cryptoEnabledCheckbox.checked) {
    cryptoConfigGroup.style.display = 'block';
    // 如果没有密钥，自动生成
    if (!cryptoKeyInput.value) {
      generateCryptoKey();
    }
  } else {
    cryptoConfigGroup.style.display = 'none';
  }
}

// 生成随机加密密钥
function generateCryptoKey(): void {
  const keyBytes = new Uint8Array(32);
  crypto.getRandomValues(keyBytes);
  const keyBase64 = btoa(String.fromCharCode(...keyBytes));
  cryptoKeyInput.value = keyBase64;
  showMessage('密钥已生成', 'success');
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
      },
      auth: {
        username: authUsernameInput.value.trim(),
        password: authPasswordInput.value.trim() || (await StorageUtil.getConfig()).auth?.password || '',
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

// 启用连接
async function connect(): Promise<void> {
  try {
    // 检查账号密码（优先用表单值，自动保存）
    const username = authUsernameInput.value.trim();
    const password = authPasswordInput.value.trim();
    const config = await StorageUtil.getConfig();
    const authUser = username || config.auth?.username;
    const authPass = password || config.auth?.password;
    if (!authUser || !authPass) {
      showMessage('请先填写账号密码', 'error');
      return;
    }

    // 如果启用加密，生成新密钥
    if (cryptoEnabledCheckbox.checked) {
      generateCryptoKey();
    }

    // 自动保存配置（包括websocketUrl和账号密码）
    await StorageUtil.saveConfig({
      ...config,
      websocketUrl: websocketUrlInput.value.trim(),
      auth: { username: authUser, password: authPass },
      crypto: {
        enabled: cryptoEnabledCheckbox.checked,
        key: cryptoKeyInput.value.trim(),
        algorithm: cryptoAlgorithmSelect.value
      }
    });

    showMessage('正在启用连接...', 'info');

    const response = await chrome.runtime.sendMessage({ type: 'connect' });

    if (response && response.success) {
      connectBtn.style.display = 'none';
      disconnectBtn.style.display = 'block';
      showMessage('连接已启用', 'success');
    } else {
      showMessage(response?.error || '启用连接失败', 'error');
    }
  } catch (error) {
    console.error('启用连接失败:', error);
    showMessage('启用连接失败', 'error');
  }
}

// 停止连接
async function disconnect(): Promise<void> {
  try {
    showMessage('正在停止连接...', 'info');
    
    const response = await chrome.runtime.sendMessage({ type: 'disconnect' });
    
    if (response && response.success) {
      connectBtn.style.display = 'block';
      disconnectBtn.style.display = 'none';
      showMessage('连接已停止', 'success');
    } else {
      showMessage(response?.error || '停止连接失败', 'error');
    }
  } catch (error) {
    console.error('停止连接失败:', error);
    showMessage('停止连接失败', 'error');
  }
}

// 重新连接
async function reconnect(): Promise<void> {
  try {
    const username = authUsernameInput.value.trim();
    const password = authPasswordInput.value.trim();
    const config = await StorageUtil.getConfig();
    const authUser = username || config.auth?.username;
    const authPass = password || config.auth?.password;
    if (!authUser || !authPass) {
      showMessage('请先填写账号密码', 'error');
      return;
    }

    // 如果启用加密，生成新密钥
    if (cryptoEnabledCheckbox.checked) {
      generateCryptoKey();
    }

    // 自动保存配置（包括websocketUrl和账号密码）
    await StorageUtil.saveConfig({
      ...config,
      websocketUrl: websocketUrlInput.value.trim(),
      auth: { username: authUser, password: authPass },
      crypto: {
        enabled: cryptoEnabledCheckbox.checked,
        key: cryptoKeyInput.value.trim(),
        algorithm: cryptoAlgorithmSelect.value
      }
    });

    showMessage('正在重新连接...', 'info');

    // 先停止连接，再启用连接
    await chrome.runtime.sendMessage({ type: 'disconnect' });
    const response = await chrome.runtime.sendMessage({ type: 'connect' });

    if (response && response.success) {
      connectBtn.style.display = 'none';
      disconnectBtn.style.display = 'block';
      showMessage('重新连接成功', 'success');
    } else {
      showMessage(response?.error || '重新连接失败', 'error');
    }
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
    // 先停止连接
    await chrome.runtime.sendMessage({ type: 'disconnect' });

    // 重置为默认值
    const defaultConfig: ClientConfig = DEFAULT_CONFIG;

    const defaultRules: RuleConfig = {
      enabled: true,
      whitelist: [],
      blacklist: [],
      urlPatterns: []
    };

    await StorageUtil.saveConfig(defaultConfig);
    await StorageUtil.saveRules(defaultRules);
    // 清除登录状态和账号密码
    await StorageUtil.clearAuthState();

    // 更新按钮状态
    updateProxyButtonState(false);
    updateAuthDisplay(null);

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

// 加载认证状态
async function loadAuthState(): Promise<void> {
  const state = await StorageUtil.getAuthState();
  updateAuthDisplay(state);
  // 填充已保存的账号
  const config = await StorageUtil.getConfig();
  if (config.auth?.username) {
    authUsernameInput.value = config.auth.username;
  }
}

// 更新认证界面显示
function updateAuthDisplay(state: AuthState | null): void {
  if (state?.authenticated) {
    loginForm.style.display = 'none';
    authInfo.style.display = 'block';
    authUserDisplay.textContent = state.username;
    adminBadge.style.display = state.isAdmin ? 'inline' : 'none';
    userManageToggle.style.display = state.isAdmin ? 'inline-block' : 'none';
  } else {
    loginForm.style.display = 'block';
    authInfo.style.display = 'none';
    (document.getElementById('changePwdSection') as HTMLElement).style.display = 'none';
    (document.getElementById('userManageSection') as HTMLElement).style.display = 'none';
  }
}

// 处理登录
async function handleLogin(): Promise<void> {
  const username = authUsernameInput.value.trim();
  const password = authPasswordInput.value.trim();
  if (!username || !password) {
    showMessage('请输入用户名和密码', 'error');
    return;
  }

  showMessage('登录中...', 'info');
  try {
    const result = await chrome.runtime.sendMessage({
      type: 'login',
      data: {
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
        },
        auth: { username, password }
      }
    });
    if (result.success) {
      showMessage('登录成功', 'success');
      authPasswordInput.value = '';
    } else {
      showMessage(result.message || '登录失败', 'error');
    }
  } catch (error) {
    showMessage('登录失败', 'error');
  }
}

// 处理退出登录
async function handleLogout(): Promise<void> {
  await chrome.runtime.sendMessage({ type: 'logout' });
  updateAuthDisplay(null);
  showMessage('已退出登录', 'success');
}

// 处理修改密码
async function handleChangePassword(): Promise<void> {
  const oldPwd = (document.getElementById('oldPassword') as HTMLInputElement).value;
  const newPwd = (document.getElementById('newPassword') as HTMLInputElement).value;
  const confirmPwd = (document.getElementById('confirmPassword') as HTMLInputElement).value;

  if (!oldPwd || !newPwd) {
    showMessage('请填写完整', 'error');
    return;
  }
  if (newPwd !== confirmPwd) {
    showMessage('两次密码不一致', 'error');
    return;
  }

  try {
    const result = await chrome.runtime.sendMessage({
      type: 'wsMessage',
      data: { type: 'change_password', data: { oldPassword: oldPwd, newPassword: newPwd } }
    });
    if (result.success) {
      showMessage('密码修改成功', 'success');
      (document.getElementById('changePwdSection') as HTMLElement).style.display = 'none';
      // 更新存储的密码
      const config = await StorageUtil.getConfig();
      if (config.auth) {
        await StorageUtil.saveConfig({ ...config, auth: { ...config.auth, password: newPwd } });
      }
    } else {
      showMessage(result.message || '修改失败', 'error');
    }
  } catch (error) {
    showMessage('修改密码失败', 'error');
  }
}

// 加载用户列表
async function loadUserList(): Promise<void> {
  try {
    const result = await chrome.runtime.sendMessage({
      type: 'wsMessage',
      data: { type: 'user_list', data: {} }
    });
    if (result.success) {
      renderUserTable(result.users || []);
    } else {
      showMessage(result.message || '获取用户列表失败', 'error');
    }
  } catch (error) {
    showMessage('获取用户列表失败', 'error');
  }
}

// 渲染用户表格
function renderUserTable(users: Array<{ username: string; role: string; enabled: boolean }>): void {
  const tbody = document.getElementById('userTableBody') as HTMLElement;
  tbody.innerHTML = '';
  for (const user of users) {
    const tr = document.createElement('tr');
    const roleText = user.role === 'super_admin' ? '超级管理员' : user.role === 'admin' ? '管理员' : '普通用户';
    const statusText = user.enabled ? '启用' : '禁用';
    const canDelete = user.role !== 'super_admin';
    tr.innerHTML = `
      <td>${user.username}</td>
      <td>${roleText}</td>
      <td>${statusText}</td>
      <td>${canDelete ? `<button class="btn btn-danger btn-sm delete-user-btn" data-username="${user.username}">删除</button>` : '-'}</td>
    `;
    tbody.appendChild(tr);
  }
  // 绑定删除按钮
  tbody.querySelectorAll('.delete-user-btn').forEach(btn => {
    btn.addEventListener('click', async (e) => {
      const username = (e.target as HTMLElement).getAttribute('data-username');
      if (username && confirm(`确定删除用户 ${username}？`)) {
        await handleDeleteUser(username);
      }
    });
  });
}

// 处理添加用户
async function handleAddUser(): Promise<void> {
  const username = (document.getElementById('newUsername') as HTMLInputElement).value.trim();
  const password = (document.getElementById('newUserPassword') as HTMLInputElement).value.trim();
  const role = (document.getElementById('newUserRole') as HTMLSelectElement).value;
  const enabled = (document.getElementById('newUserEnabled') as HTMLInputElement).checked;

  if (!username || !password) {
    showMessage('请填写用户名和密码', 'error');
    return;
  }

  try {
    const result = await chrome.runtime.sendMessage({
      type: 'wsMessage',
      data: { type: 'user_create', data: { username, password, role, enabled } }
    });
    if (result.success) {
      showMessage('用户创建成功', 'success');
      (document.getElementById('newUsername') as HTMLInputElement).value = '';
      (document.getElementById('newUserPassword') as HTMLInputElement).value = '';
      (document.getElementById('newUserRole') as HTMLSelectElement).value = 'user';
      (document.getElementById('newUserEnabled') as HTMLInputElement).checked = true;
      loadUserList();
    } else {
      showMessage(result.message || '创建失败', 'error');
    }
  } catch (error) {
    showMessage('创建用户失败', 'error');
  }
}

// 处理删除用户
async function handleDeleteUser(username: string): Promise<void> {
  try {
    const result = await chrome.runtime.sendMessage({
      type: 'wsMessage',
      data: { type: 'user_delete', data: { username } }
    });
    if (result.success) {
      showMessage('用户已删除', 'success');
      loadUserList();
    } else {
      showMessage(result.message || '删除失败', 'error');
    }
  } catch (error) {
    showMessage('删除用户失败', 'error');
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
