@echo off
chcp 65001 >nul
echo ============================================
echo    Eino 智能运维 Agent - 一键启动脚本
echo ============================================
echo.

echo [1/2] 启动 Go 后端服务 (端口 8080)...
start "Go Server" cmd /k "cd /d %~dp0server-go && go run main.go"
timeout /t 3 /nobreak >nul

echo [2/2] 启动 Vue3 前端 (端口 5173)...
start "Vue3 Web" cmd /k "cd /d %~dp0web && npm run dev"

echo.
echo ============================================
echo    所有服务已启动！
echo ============================================
echo.
echo 访问地址:
echo   Web UI:       http://localhost:5173
echo   后端 API:     http://localhost:8080
echo.
echo 登录账号:
echo   用户名: admin
echo   密码:   admin123
echo.
echo 按任意键关闭此窗口...
pause >nul