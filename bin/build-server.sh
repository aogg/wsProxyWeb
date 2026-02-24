#!/bin/bash
echo "========================================"
echo "  构建服务端"
echo "========================================"

cd "$(dirname "$0")/../server"
go build -o ../runtime/wsproxy-server .
if [ $? -ne 0 ]; then
    echo "服务端构建失败！"
    exit 1
fi

echo ""
echo "构建完成！输出文件: runtime/wsproxy-server"
