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
	// 加载配置文件
	config, err := libs.LoadConfig("")
	if err != nil {
		log.Fatalf("加载配置文件失败: %v", err)
	}

	// 从配置文件读取端口，如果命令行参数有指定则优先使用命令行参数
	port := config.Server.Port
	if len(os.Args) > 1 {
		port = os.Args[1]
		log.Printf("使用命令行参数指定的端口: %s", port)
	} else {
		log.Printf("使用配置文件中的端口: %s", port)
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
