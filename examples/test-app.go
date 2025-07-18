package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("启动时间: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Printf("进程ID: %d\n", os.Getpid())
	fmt.Printf("端口: %s\n", port)

	// 简单的HTTP服务器
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		response := fmt.Sprintf(`{
  "timestamp": "%s",
  "pid": %d,
  "message": "Hello from GoPM2 Go test app!",
  "url": "%s",
  "method": "%s"
}`, timestamp, os.Getpid(), r.URL.Path, r.Method)

		fmt.Printf("[%s] %s %s - PID: %d\n", timestamp, r.Method, r.URL.Path, os.Getpid())

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(response))
	})

	http.HandleFunc("/panic", func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("[%s] 触发panic测试 - PID: %d\n", time.Now().Format("2006-01-02 15:04:05"), os.Getpid())
		// 在单独的goroutine中panic，这样不会被HTTP服务器恢复
		go func() {
			panic("test panic - 程序应该退出")
		}()
		// 给一点时间让panic发生
		time.Sleep(100 * time.Millisecond)
		// 如果panic没有生效，强制退出
		fmt.Println("强制退出程序")
		os.Exit(1)
	})

	// 启动服务器
	go func() {
		fmt.Printf("✓ 服务器启动成功，监听端口 %s\n", port)
		if err := http.ListenAndServe(":"+port, nil); err != nil {
			fmt.Printf("服务器启动失败: %v\n", err)
		}
	}()

	// 定期输出心跳日志
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				fmt.Printf("[心跳] %s - PID: %d\n", time.Now().Format("2006-01-02 15:04:05"), os.Getpid())
			}
		}
	}()

	// 等待信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigChan
	fmt.Printf("收到信号 %v，正在关闭服务器...\n", sig)
	fmt.Println("服务器已关闭")
}
