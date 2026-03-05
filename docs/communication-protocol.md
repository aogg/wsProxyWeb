# 客户端与服务端通信协议

## 概述

本文档描述 WebSocket 代理系统中客户端和服务端之间的通信协议、配置同步机制和消息处理流程。

## 连接建立流程

```
客户端                                    服务端
  |                                        |
  |-------- WebSocket 连接请求 ----------->|
  |                                        |
  |<------- 连接建立成功 -------------------|
  |                                        |
  |-------- 推送配置 (update_config) ------>|
  |                                        |
  |<------- 配置更新结果 -------------------|
  |                                        |
  |-------- 认证请求 (auth) --------------->|
  |                                        |
  |<------- 认证结果 (auth_result) ---------|
  |                                        |
```

## 配置推送机制

### 客户端推送配置

客户端在连接建立后自动推送加密和压缩配置给服务端。

**消息类型**: `update_config`

**消息格式**:
```json
{
  "id": "update_config_1234567890",
  "type": "update_config",
  "data": {
    "crypto": {
      "enabled": true,
      "key": "base64编码的32字节密钥",
      "algorithm": "aes256gcm"
    },
    "compress": {
      "enabled": true,
      "algorithm": "gzip",
      "level": 6
    }
  }
}
```

**字段说明**:
- `crypto.enabled`: 是否启用加密
- `crypto.key`: 加密密钥（Base64编码）
- `crypto.algorithm`: 加密算法（`aes256gcm` 或 `chacha20poly1305`）
- `compress.enabled`: 是否启用压缩
- `compress.algorithm`: 压缩算法（`gzip` 或 `deflate`）
- `compress.level`: 压缩级别（1-9）

### 服务端响应

**消息类型**: `update_config_result`

**消息格式**:
```json
{
  "id": "update_config_1234567890",
  "type": "update_config_result",
  "data": {
    "success": true,
    "message": "配置更新成功"
  }
}
```

## 消息处理流程

### 发送消息流程

#### 客户端发送
```
原始数据 → JSON序列化 → 压缩 → 加密 → WebSocket发送（二进制）
```

**代码位置**: `client/src/libs/websocket.ts:297-323`

**处理步骤**:
1. 将消息对象序列化为 JSON 字符串
2. 转换为 UTF-8 字节数组
3. 如果启用压缩，使用 `CompressUtil.compress()` 压缩
4. 如果启用加密，使用 `CryptoUtil.encrypt()` 加密
5. 通过 WebSocket 发送二进制数据

#### 服务端发送
```
原始数据 → JSON序列化 → 压缩 → 加密 → WebSocket发送（二进制）
```

**代码位置**: `server/src/libs/websocket_lib.go:449-471`

**处理步骤**:
1. 将消息对象序列化为 JSON
2. 如果启用压缩，使用 `CompressLib.Compress()` 压缩
3. 如果启用加密，使用 `CryptoLib.Encrypt()` 加密
4. 通过 WebSocket 发送二进制数据

### 接收消息流程

#### 客户端接收
```
WebSocket接收（二进制） → 解密 → 解压 → JSON解析 → 原始数据
```

**代码位置**: `client/src/libs/websocket.ts:328-352`

**处理步骤**:
1. 接收 WebSocket 二进制数据
2. 如果启用加密，使用 `CryptoUtil.decrypt()` 解密
3. 如果启用压缩，使用 `CompressUtil.decompress()` 解压
4. 将 UTF-8 字节数组转换为字符串
5. 解析 JSON 得到消息对象

#### 服务端接收
```
WebSocket接收（二进制） → 解密 → 解压 → JSON解析 → 原始数据
```

**代码位置**: `server/src/libs/websocket_lib.go:427-447`

**处理步骤**:
1. 接收 WebSocket 二进制数据
2. 如果启用加密，使用 `CryptoLib.Decrypt()` 解密
3. 如果启用压缩，使用 `CompressLib.Decompress()` 解压
4. 解析 JSON 得到消息对象

## 消息类型

### 系统消息

| 消息类型 | 方向 | 说明 |
|---------|------|------|
| `ping` | 客户端→服务端 | 心跳请求 |
| `pong` | 服务端→客户端 | 心跳响应 |
| `update_config` | 客户端→服务端 | 更新配置（加密+压缩） |
| `update_config_result` | 服务端→客户端 | 配置更新结果 |
| `update_crypto_key` | 客户端→服务端 | 更新加密密钥（兼容旧版本） |
| `update_crypto_key_result` | 服务端→客户端 | 密钥更新结果 |

### 认证消息

| 消息类型 | 方向 | 说明 |
|---------|------|------|
| `auth` | 客户端→服务端 | 认证请求 |
| `auth_result` | 服务端→客户端 | 认证结果 |
| `change_password` | 客户端→服务端 | 修改密码 |
| `change_password_result` | 服务端→客户端 | 修改密码结果 |

### 代理消息

| 消息类型 | 方向 | 说明 |
|---------|------|------|
| `proxy_request` | 客户端→服务端 | HTTP代理请求 |
| `proxy_response` | 服务端→客户端 | HTTP代理响应 |

## 配置同步注意事项

1. **配置必须一致**: 客户端和服务端的加密、压缩配置必须完全一致，否则会导致解密或解压失败

2. **推送时机**: 客户端在 WebSocket 连接建立后立即推送配置，确保后续通信使用正确的配置

3. **处理顺序**:
   - 发送: 先压缩后加密
   - 接收: 先解密后解压

4. **兼容性**: 服务端同时支持 `update_config` 和 `update_crypto_key` 消息类型，保持向后兼容

5. **配置禁用**: 如果加密或压缩被禁用，对应的处理步骤会直接返回原始数据，不做任何处理

## 错误处理

当消息处理失败时，服务端会返回错误消息：

```json
{
  "id": "",
  "type": "error",
  "data": {
    "code": "MESSAGE_PROCESS_ERROR",
    "message": "处理消息失败: 具体错误信息"
  }
}
```

常见错误：
- 解密失败：密钥不匹配或算法不一致
- 解压失败：压缩配置不一致
- JSON解析失败：数据格式错误
