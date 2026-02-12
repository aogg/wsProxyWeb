// WebSocket代理服务端入口文件
package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"wsProxyWeb/server/src/libs"
)

func main() {
	// 创建WebSocket服务器，默认端口8080
	port := "8080"
	if len(os.Args) > 1 {
		port = os.Args[1]
	}

	server := libs.NewWebSocketServer(port)

	// 处理优雅关闭
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// 在goroutine中启动服务器
	go func() {
		if err := server.Start(); err != nil {
			log.Fatalf("服务器启动失败: %v", err)
		}
	}()

	// 等待中断信号
	<-sigChan
	log.Println("收到关闭信号，正在关闭服务器...")
	server.Stop()
	log.Println("服务器已关闭")
}
