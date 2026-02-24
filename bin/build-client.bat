@echo off
chcp 65001 >nul
echo ========================================
echo   构建客户端
echo ========================================

cd /d "%~dp0..\client"
call npm run build
if errorlevel 1 (
    echo 客户端构建失败！
    exit /b 1
)

echo.
echo 构建完成！输出目录: client\dist\
