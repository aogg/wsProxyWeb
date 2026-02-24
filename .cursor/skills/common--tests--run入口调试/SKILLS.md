---
name: common--tests--run入口调试
description: 编写tests/测试代码时必看
allowed-tools: Read, Write, Bash, WebFetch
---

# 目录结构
- tests测试目录包含

  - run入口文件夹

  - import文件夹

# 说明
- 必须有个run文件夹放测试脚本
- run文件夹的代码每个测试方法里每行只有简单的调用入口代码，不允许有复杂的逻辑
- 调用入口代码调用的可以是主代码，也可以是import文件夹里的代码

## import文件夹
- 放run里的每次调用前的准备参数和调用后的断言代码