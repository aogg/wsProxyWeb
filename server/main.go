// WebSocket代理服务端入口文件
package main

import (
	"os"
	"os/signal"
	"syscall"

	"wsProxyWeb/server/src/libs"
)

const Version = "1.0.0"

func main() {
	libs.Info("WebSocket代理服务端 v%s", Version)
	// 加载配置文件
	config, err := libs.LoadConfig("")
	if err != nil {
		// 配置加载失败，使用默认日志输出到控制台
		libs.Error("加载配置文件失败: %v", err)
		os.Exit(1)
	}

	// 初始化日志库
	if err := libs.InitLog(&config.Log); err != nil {
		libs.Error("初始化日志库失败: %v", err)
		os.Exit(1)
	}
	defer libs.GetLogLib().Close()

	// 从配置文件读取端口，如果命令行参数有指定则优先使用命令行参数
	port := config.Server.Port
	if len(os.Args) > 1 {
		port = os.Args[1]
		libs.Info("使用命令行参数指定的端口: %s", port)
	} else {
		libs.Info("使用配置文件中的端口: %s", port)
	}

	server := libs.NewWebSocketServer(port)

	// 处理优雅关闭
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// 在goroutine中启动服务器
	go func() {
		if err := server.Start(); err != nil {
			libs.Fatal("服务器启动失败: %v", err)
		}
	}()

	// 等待中断信号
	<-sigChan
	libs.Info("收到关闭信号，正在关闭服务器...")
	server.Stop()
	libs.Info("服务器已关闭")
}
