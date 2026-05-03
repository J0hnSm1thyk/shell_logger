@echo off
chcp 65001 > nul
title Shell Logger Session

cd /d "%~dp0"

if not exist shell_logger.exe (
    echo [ERROR] shell_logger.exe が見つかりません
    pause
    exit /b 1
)

shell_logger.exe cmd
pause