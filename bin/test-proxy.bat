@echo off
chcp 65001 >nul
echo ========================================
echo   运行代理测试
echo ========================================

cd /d "%~dp0.."
go test -v -run "TestProxyRequest" ./server/src/tests/
