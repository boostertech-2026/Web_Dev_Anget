@echo off
chcp 65001 >nul
echo ============================================
echo    Eino 智能运维 Agent - 安装依赖脚本
echo ============================================
echo.

echo [1/2] 安装 Node.js 依赖...
cd /d %~dp0web
npm install
echo Node.js 依赖安装完成！
echo.

echo [2/2] 下载 Go 依赖...
cd /d %~dp0server-go
go env -w GOPROXY=https://goproxy.cn,direct
go env -w GOSUMDB=off
go mod tidy
echo Go 依赖下载完成！
echo.

echo ============================================
echo    所有依赖安装完成！
echo ============================================
echo.
echo 现在可以运行 start.bat 启动服务！
echo.
echo 按任意键关闭此窗口...
pause >nul