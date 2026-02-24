@echo off
chcp 65001 >nul
echo ========================================
echo   启动服务端
echo ========================================

cd /d "%~dp0.."

REM 检查是否已构建
if not exist "runtime\wsproxy-server.exe" (
    echo 服务端未构建，正在构建...
    call bin\build-server.bat
    if errorlevel 1 exit /b 1
)

cd runtime
wsproxy-server.exe %*
