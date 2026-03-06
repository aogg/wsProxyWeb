# 部署客户端到 Microsoft Edge 插件市场

本文档说明如何将 WebSocket代理 浏览器插件发布到 Microsoft Edge Add-ons 市场。

## 前置要求

1. **Microsoft 合作伙伴中心账号**
   - 访问 [Microsoft 合作伙伴中心](https://partner.microsoft.com/dashboard/microsoftedge/overview)
   - 注册开发者账号（需要 Microsoft 账号）
   - 完成开发者注册（个人或公司）

2. **构建工具**
   - Node.js 20+
   - npm

## 构建插件包

### 1. 本地构建

```bash
cd client
npm ci
npm run build
```

构建完成后，`dist/` 目录包含所有插件文件。

### 2. 打包为 ZIP

将 `dist/` 目录下的所有文件打包为 ZIP 文件：

```bash
cd dist
zip -r ../wsproxy-client.zip .
```

或在 Windows 上：
- 进入 `client/dist` 目录
- 选择所有文件（不是 dist 文件夹本身）
- 右键 → 发送到 → 压缩(zipped)文件夹

**重要**: ZIP 文件的根目录必须直接包含 `manifest.json`，而不是包含一个文件夹。

## 发布到 Edge Add-ons

### 1. 登录合作伙伴中心

访问 [Edge 扩展管理](https://partner.microsoft.com/dashboard/microsoftedge/overview)

### 2. 创建新扩展（首次发布）

1. 点击 "新建扩展"
2. 上传 ZIP 包
3. 填写扩展信息：
   - **名称**: WebSocket代理
   - **简短描述**: 通过WebSocket代理网页请求
   - **详细描述**: 提供完整的功能说明
   - **类别**: 开发者工具
   - **语言**: 中文（简体）

4. 上传资源：
   - **图标**: 使用 `client/public/icons/icon.svg`（需转换为 PNG）
   - **截图**: 至少 1 张，展示插件界面
   - **宣传图片**（可选）

5. 隐私设置：
   - 提供隐私政策 URL（如果收集用户数据）
   - 说明权限用途

6. 提交审核

### 3. 更新现有扩展

1. 在合作伙伴中心找到现有扩展
2. 点击 "更新"
3. 上传新的 ZIP 包
4. 更新版本说明
5. 提交审核

## 版本管理

### 更新版本号

编辑 `client/public/manifest.json`:

```json
{
  "version": "1.1.2"
}
```

版本号格式: `major.minor.patch`

### 使用 GitHub Actions 自动构建

项目已配置自动构建流程（`.github/workflows/release-client.yml`）：

1. 创建 tag:
```bash
git tag client-v1.1.2
git push origin client-v1.1.2
```

2. GitHub Actions 自动构建并创建 Release
3. 从 Release 页面下载 `client-dist.tar.gz`
4. 解压并打包为 ZIP 上传到 Edge Add-ons

## 审核流程

- **审核时间**: 通常 1-3 个工作日
- **审核标准**:
  - 符合 [Edge Add-ons 政策](https://docs.microsoft.com/microsoft-edge/extensions-chromium/store-policies/developer-policies)
  - 无恶意代码
  - 权限使用合理
  - 功能描述准确

## 权限说明

插件请求的权限及用途：

- `webRequest`: 拦截和修改网页请求
- `storage`: 保存配置信息
- `tabs`: 管理标签页状态
- `scripting`: 注入脚本
- `alarms`: 定时任务
- `<all_urls>`: 代理所有网站请求

在提交时需要详细说明每个权限的使用场景。

## 常见问题

### 1. 审核被拒

- 检查是否违反政策
- 补充权限说明
- 更新隐私政策
- 修改后重新提交

### 2. 版本号冲突

确保每次提交的版本号大于当前版本。

### 3. ZIP 格式错误

确保 ZIP 根目录直接包含 `manifest.json`，而不是嵌套文件夹。

## 参考资源

- [Edge 扩展开发文档](https://docs.microsoft.com/microsoft-edge/extensions-chromium/)
- [发布到 Edge Add-ons](https://docs.microsoft.com/microsoft-edge/extensions-chromium/publish/publish-extension)
- [合作伙伴中心](https://partner.microsoft.com/dashboard/microsoftedge/overview)
