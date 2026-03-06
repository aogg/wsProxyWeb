---
name: git推送
description: git推送
allowed-tools: Read, Write, Bash, WebFetch
---



# 说明
- 自动自增版本号


## 需要先修改前后端的版本号
- client/public/manifest.json:4
- server/main.go:12

## 然后推送git，并且给git打标签
- 打额tag会应用到github releases，所以也要维护好github releases的说明
- 需要符合.github/workflows/里的标签要求



