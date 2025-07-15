package main

import (
	"fmt"
	"os"
)

func main() {
	// 执行根命令
	if err := rootCmd.Execute(); err != nil {
		fmt.Printf("错误: %v\n", err)
		os.Exit(1)
	}
}
