#!/bin/bash
echo "========================================"
echo "  构建所有组件"
echo "========================================"

cd "$(dirname "$0")/.."

echo ""
echo "[1/2] 构建客户端..."
cd client
npm run build
if [ $? -ne 0 ]; then
    echo "客户端构建失败！"
    exit 1
fi

echo ""
echo "[2/2] 构建服务端..."
cd ../server
go build -o ../runtime/wsproxy-server .
if [ $? -ne 0 ]; then
    echo "服务端构建失败！"
    exit 1
fi

echo ""
echo "========================================"
echo "  构建完成！"
echo "  客户端: client/dist/"
echo "  服务端: runtime/wsproxy-server"
echo "========================================"
