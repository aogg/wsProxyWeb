# bin 目录使用说明

## 概述

`bin/` 目录包含项目的构建和启动脚本，支持 Windows (.bat) 和 Unix/Linux (.sh) 双平台。

## 脚本列表

| 脚本 | Windows | Unix/Linux | 功能说明 |
|------|---------|------------|----------|
| 构建所有 | `build.bat` | `build.sh` | 构建客户端和服务端 |
| 构建客户端 | `build-client.bat` | `build-client.sh` | 只构建客户端 |
| 构建服务端 | `build-server.bat` | `build-server.sh` | 只构建服务端 |
| 启动服务端 | `start-server.bat` | `start-server.sh` | 启动服务端（自动构建） |
| 客户端开发 | `dev-client.bat` | `dev-client.sh` | 客户端开发模式（watch） |
| 服务端开发 | `dev-server.bat` | `dev-server.sh` | 服务端开发模式（go run） |

## 使用方式

### Windows (PowerowerShell/CMD)

```powershell
# 构建所有组件
.\bin\build.bat

# 只构建客户端
.\bin\build-client.bat

# 只构建服务端
.\bin\build-server.bat

# 启动服务端（可指定端口）
.\bin\start-server.bat
.\bin\start-server.bat 9090

# 客户端开发模式（自动监听文件变化）
.\bin\dev-client.bat

# 服务端开发模式（直接运行源码）
.\bin\dev-server.bat
```

### Unix/Linux/macOS

```bash
# 添加执行权限（首次使用）
chmod +x bin/*.sh

# 构建所有组件
./bin/build.sh

# 只构建客户端
./bin/build-client.sh

# 只构建服务端
./bin/build-server.sh

# 启动服务端（可指定端口）
./bin/start-server.sh
./bin/start-server.sh 9090

# 客户端开发模式
./bin/dev-client.sh

# 服务端开发模式
./bin/dev-server.sh
```

## 输出目录

| 组件 | 输出位置 |
|------|----------|
| 客户端 | `client/dist/` |
| 服务端 | `runtime/wsproxy-server.exe` (Windows) 或 `runtime/wsproxy-server` (Unix) |

## 开发流程建议

### 首次使用

```bash
# 1. 安装客户端依赖
cd client && npm install && cd ..

# 2. 下载服务端依赖
cd server && go mod download && cd ..

# 3. 构建所有组件
.\bin\build.bat   # Windows
./bin/build.sh    # Unix
```

### 日常开发

```bash
# 终端1：客户端开发模式
.\bin\dev-client.bat

# 终端2：服务端开发模式
.\bin\dev-server.bat
```

### 生产部署

```bash
# 构建所有
.\bin\build.bat

# 启动服务
.\bin\start-server.bat
```
