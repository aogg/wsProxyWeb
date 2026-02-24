@echo off
chcp 65001 >nul
echo ========================================
echo   客户端开发模式 (watch)
echo ========================================

cd /d "%~dp0..\client"
call npm run dev
