#!/bin/bash
echo "========================================"
echo "  启动服务端"
echo "========================================"

cd "$(dirname "$0")/.."

# 检查是否已构建
if [ ! -f "runtime/wsproxy-server" ]; then
    echo "服务端未构建，正在构建..."
    ./bin/build-server.sh
    if [ $? -ne 0 ]; then exit 1; fi
fi

cd runtime
./wsproxy-server "$@"
