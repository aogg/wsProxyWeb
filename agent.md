# Agent Context: wsProxyWeb

## 1) 项目概览
- 项目名：`wsProxyWeb`
- 目标：通过浏览器插件将网页 HTTP/HTTPS 请求经 WebSocket 转发到代理服务端执行。
- 架构：
  - `server/`：Go 实现的 WebSocket 代理后端
  - `client/`：TypeScript 实现的浏览器扩展（Manifest V3）
  - `docs/`：使用、协议与开发文档

## 2) 快速开始
- 安装依赖：
  - `cd client && npm install`
  - `cd ../server && go mod download`
- 构建：
  - `bin/build.bat`
- 启动服务端：
  - `bin/start-server.bat`
  - 指定端口：`bin/start-server.bat 9090`
- 默认 WebSocket 地址：
  - `ws://localhost:8080/ws`

## 3) 常用开发命令
- 服务端：
  - `cd server`
  - `go run main.go`
  - `go test ./src/tests/...`
- 客户端：
  - `cd client`
  - `npm run dev`
  - `npm run build`
  - `npm test`
  - `npm run test:coverage`

## 4) 关键路径
- 服务端配置：`server/src/configs/server.yaml`
- 客户端源码：`client/src/`
- 扩展清单：`client/public/manifest.json`
- 扩展构建输出：`client/dist/`

## 5) 协作约定
- 优先进行最小化、目标明确的修改。
- 涉及消息结构调整时，必须同时关注前后端协议兼容性。
- 行为变更需补充或更新测试：
  - `server/src/tests/`
  - `client/src/tests/`

## 6) 本地产物
- 构建产物（`*.exe`、`client/node_modules/`、`client/dist/`）不应提交。

## 7) 建议阅读顺序
1. `docs/使用说明.md`
2. `docs/协议说明.md`
3. `docs/开发文档.md`
