# WebSocket 代理

通过浏览器扩展和 WebSocket 代理网页请求的系统，支持 Chrome 和 Edge 浏览器。

## 功能特性

- 🔄 自动拦截并代理网页 HTTP/HTTPS 请求
- 🔐 支持 AES-256-GCM 和 ChaCha20-Poly1305 加密
- 📦 支持 Gzip 和 Snappy 数据压缩
- 🛡️ 完善的安全控制（连接数限制、频率限制、IP/域名黑白名单）
- ⚡ HTTP 连接池复用，高性能并发处理
- 🎯 灵活的请求过滤规则（域名白名单/黑名单、URL 模式匹配）

## 快速开始

### 系统要求

- **服务端**: Go 1.24.0+
- **客户端**: Node.js 20.0+, TypeScript 5.9+
- **浏览器**: Chrome 88+ 或 Edge 88+

### 安装依赖

```bash
# 客户端
cd client && npm install && cd ..

# 服务端
cd server && go mod download && cd ..
```

### 构建项目

```bash
# Windows
.\bin\build.bat

# Linux/macOS
./bin/build.sh
```

### 启动服务端

**方式一：使用 Docker（推荐）**

```bash
# 拉取镜像
docker pull adockero/wsproxyweb-server:latest

# 运行容器
docker run -d \
  --name wsproxy-server \
  -p 8080:8080 \
  -v $(pwd)/server/src/configs:/app/src/configs \
  adockero/wsproxyweb-server:latest
```

**方式二：本地运行**

```bash
# Windows (默认端口 8080)
.\bin\start-server.bat

# Linux/macOS
./bin/start-server.sh
```

服务端将在 `ws://localhost:8080/ws` 监听 WebSocket 连接。

### 安装浏览器扩展

构建完成后，插件文件位于 `client/dist` 目录。

**Chrome:**
1. 访问 `chrome://extensions/`
2. 开启「开发者模式」
3. 点击「加载已解压的扩展程序」
4. 选择 `client/dist` 目录

**Edge:**
1. 访问 `edge://extensions/`
2. 开启「开发人员模式」
3. 点击「加载解压缩的扩展」
4. 选择 `client/dist` 目录

## 配置说明

### 服务端配置

配置文件: `server/src/configs/server.yaml`

```yaml
server:
  port: "8080"

crypto:
  enabled: false
  key: ""  # 32字节 Base64 编码
  algorithm: "aes256gcm"

compress:
  enabled: false
  level: 6
  algorithm: "gzip"

security:
  enabled: true
  maxConnections: 50
  rateLimitPerSecond: 50
```

### 客户端配置

点击浏览器工具栏的扩展图标，在弹窗中配置：

- WebSocket 服务器地址
- 加密设置（密钥、算法）
- 压缩设置（级别、算法）
- 拦截规则（域名白名单/黑名单、URL 模式）

## 开发调试

```bash
# 客户端开发模式（自动监听文件变化）
.\bin\dev-client.bat

# 服务端开发模式
.\bin\dev-server.bat
```

## 项目结构

```
wsProxyWeb/
├── client/          # 浏览器扩展（TypeScript）
│   ├── src/
│   │   ├── background/   # Service Worker
│   │   ├── popup/        # 配置界面
│   │   └── libs/         # 核心库
│   └── dist/        # 构建输出
├── server/          # 代理服务器（Go）
│   └── src/
│       ├── logic/        # 业务逻辑
│       ├── libs/         # 核心库
│       └── configs/      # 配置文件
├── bin/             # 构建和启动脚本
└── docs/            # 详细文档
```

## 安全建议

⚠️ **重要提示**

1. 生产环境务必启用加密
2. 定期更换加密密钥
3. 配置合适的 IP 白名单
4. 设置合理的请求频率限制
5. 不要在不可信网络中使用

## 文档

- [使用说明](docs/使用说明.md) - 详细的安装和配置指南
- [开发文档](docs/开发文档.md) - 开发者指南
- [协议说明](docs/协议说明.md) - 通信协议详解
- [构建脚本说明](docs/bin使用说明.md) - 脚本使用说明

## 许可证

MIT License

## 版本

当前版本: v1.1.4
