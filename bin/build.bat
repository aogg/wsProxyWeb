@echo off
chcp 65001 >nul
echo ========================================
echo   构建所有组件
echo ========================================

echo.
echo [1/2] 构建客户端...
cd /d "%~dp0..\client"
call npm run build
if errorlevel 1 (
    echo 客户端构建失败！
    exit /b 1
)

echo.
echo [2/2] 构建服务端...
cd /d "%~dp0..\server"
go build -o ..\runtime\wsproxy-server.exe .
if errorlevel 1 (
    echo 服务端构建失败！
    exit /b 1
)

echo.
echo ========================================
echo   构建完成！
echo   客户端: client\dist\
echo   服务端: runtime\wsproxy-server.exe
echo ========================================
