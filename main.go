package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/shirou/gopsutil/v3/process"
)

func main() {
	// 检查是否需要以守护进程模式运行
	if len(os.Args) > 1 && os.Args[1] == "daemon" {
		runDaemon()
		return
	}

	// 对于所有其他命令，确保守护进程正在运行
	ensureDaemonRunning()

	// 执行CLI命令
	Execute()
}

// runDaemon 运行守护进程模式
func runDaemon() {
	fmt.Println("启动 GoPM2 守护进程...")

	pm := NewProcessManager()

	// 创建锁文件
	lockFile := filepath.Join(pm.dataDir, "daemon.lock")
	pid := fmt.Sprintf("%d", os.Getpid())
	if err := os.WriteFile(lockFile, []byte(pid), 0644); err != nil {
		fmt.Printf("创建锁文件失败: %v\n", err)
		return
	}

	// 确保退出时删除锁文件
	defer os.Remove(lockFile)

	// 恢复之前的进程
	pm.loadProcesses()

	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 启动命令处理循环
	go pm.commandLoop()

	// 启动定期保存进程状态
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				pm.saveProcesses()
			case <-sigChan:
				return
			}
		}
	}()

	fmt.Printf("GoPM2 守护进程已启动 (PID: %d)\n", os.Getpid())

	// 等待退出信号
	<-sigChan
	fmt.Println("正在关闭 GoPM2 守护进程...")

	// 优雅关闭所有进程
	processes := pm.GetProcessList()
	for _, p := range processes {
		if p.Status == StatusOnline {
			fmt.Printf("停止进程: %s\n", p.Name)
			pm.stopProcessInstance(p)
		}
	}

	fmt.Println("GoPM2 守护进程已关闭")
}

// ensureDaemonRunning 确保守护进程在运行
func ensureDaemonRunning() {
	// 检查守护进程是否已在运行
	if isDaemonRunning() {
		return
	}

	fmt.Println("启动 GoPM2 守护进程...")

	// 重新执行自己作为守护进程
	executable, err := os.Executable()
	if err != nil {
		fmt.Printf("获取可执行文件路径失败: %v\n", err)
		return
	}

	// 在 Windows 上启动守护进程
	cmd := exec.Command(executable, "daemon")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
	err = cmd.Start()
	if err != nil {
		fmt.Printf("启动守护进程失败: %v\n", err)
		return
	}

	// 等待守护进程启动
	for i := 0; i < 10; i++ {
		time.Sleep(500 * time.Millisecond)
		if isDaemonRunning() {
			fmt.Println("✓ GoPM2 守护进程已启动")
			return
		}
	}

	fmt.Println("✗ 启动 GoPM2 守护进程失败")
}

// isDaemonRunning 检查守护进程是否在运行
func isDaemonRunning() bool {
	pm := NewProcessManager()
	lockFile := filepath.Join(pm.dataDir, "daemon.lock")

	// 尝试读取现有锁文件
	if data, err := os.ReadFile(lockFile); err == nil {
		if pid, err := strconv.Atoi(string(data)); err == nil {
			// 检查进程是否存在
			if proc, err := process.NewProcess(int32(pid)); err == nil {
				if exists, _ := proc.IsRunning(); exists {
					return true
				}
			}
		}
	}

	return false
}
