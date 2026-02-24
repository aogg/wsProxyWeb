@echo off
chcp 65001 >nul
echo ========================================
echo   构建服务端
echo ========================================

cd /d "%~dp0..\server"
go build -o ..\runtime\wsproxy-server.exe .
if errorlevel 1 (
    echo 服务端构建失败！
    exit /b 1
)

echo.
echo 构建完成！输出文件: runtime\wsproxy-server.exe
