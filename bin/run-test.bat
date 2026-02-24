@echo off
chcp 65001 >nul
cd /d d:\code\www\my\github\wsProxyWeb
go test -v -run TestProxyRequest_WebSocket ./server/src/tests/ > test_output.txt 2>&1
type test_output.txt
