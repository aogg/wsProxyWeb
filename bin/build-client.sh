#!/bin/bash
echo "========================================"
echo "  构建客户端"
echo "========================================"

cd "$(dirname "$0")/../client"
npm run build
if [ $? -ne 0 ]; then
    echo "客户端构建失败！"
    exit 1
fi

echo ""
echo "构建完成！输出目录: client/dist/"
