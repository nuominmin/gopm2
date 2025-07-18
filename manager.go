package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"bufio"

	"github.com/shirou/gopsutil/v3/process"
)

// NewProcessManager 创建新的进程管理器
func NewProcessManager() *ProcessManager {
	homeDir, _ := os.UserHomeDir()
	dataDir := filepath.Join(homeDir, ".gopm2")

	// 确保数据目录存在
	os.MkdirAll(dataDir, 0755)
	os.MkdirAll(filepath.Join(dataDir, "logs"), 0755)
	os.MkdirAll(filepath.Join(dataDir, "pids"), 0755)

	pm := &ProcessManager{
		processes: make(map[int]*Process),
		nextID:    1,
		dataDir:   dataDir,
	}

	// 加载已保存的进程信息
	pm.loadProcesses()

	return pm
}

// StartProcess 启动进程
func (pm *ProcessManager) StartProcess(config AppConfig) (*Process, error) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	// 检查进程名是否已存在
	for _, p := range pm.processes {
		if p.Name == config.Name && p.Status != StatusStopped {
			return nil, fmt.Errorf("进程 '%s' 已经在运行", config.Name)
		}
	}

	// 创建进程实例
	process := &Process{
		ID:          pm.nextID,
		Name:        config.Name,
		Script:      config.Script,
		Args:        config.Args,
		Cwd:         config.Cwd,
		Env:         config.Env,
		Instances:   config.Instances,
		Status:      StatusStopped,
		Watch:       config.Watch,
		WatchIgnore: config.WatchIgnore,
		MaxRestarts: config.MaxRestarts,
		LogFile:     config.LogFile,
		ErrorFile:   config.ErrorFile,
		watcherStop: make(chan bool, 1),
	}

	// 设置默认值
	if process.Instances == 0 {
		process.Instances = 1
	}
	if process.MaxRestarts == 0 {
		process.MaxRestarts = 15
	}
	if process.Cwd == "" {
		process.Cwd, _ = os.Getwd()
	}
	if process.LogFile == "" {
		process.LogFile = filepath.Join(pm.dataDir, "logs", fmt.Sprintf("%s.log", process.Name))
	}
	if process.ErrorFile == "" {
		process.ErrorFile = filepath.Join(pm.dataDir, "logs", fmt.Sprintf("%s-error.log", process.Name))
	}

	// 设置执行模式
	if config.ExecMode == "cluster" {
		process.ExecMode = ExecModeCluster
	} else {
		process.ExecMode = ExecModeFork
	}

	// 解析最小运行时间
	if config.MinUptime != "" {
		duration, err := time.ParseDuration(config.MinUptime)
		if err == nil {
			process.MinUptime = duration
		}
	}
	if process.MinUptime == 0 {
		process.MinUptime = 1 * time.Second
	}

	pm.processes[pm.nextID] = process
	pm.nextID++

	// 启动进程
	err := pm.startProcessInstance(process)
	if err != nil {
		process.Status = StatusErrored
		return process, fmt.Errorf("启动进程失败: %v", err)
	}

	// 保存进程信息
	pm.saveProcesses()

	// 如果启用了文件监控，启动文件监控器
	if process.Watch {
		go pm.startFileWatcher(process)
	}

	return process, nil
}

// startProcessInstance 启动单个进程实例
func (pm *ProcessManager) startProcessInstance(p *Process) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// 创建日志文件
	logFile, err := os.OpenFile(p.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("创建日志文件失败: %v", err)
	}
	p.logWriter = logFile

	errorFile, err := os.OpenFile(p.ErrorFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		logFile.Close()
		return fmt.Errorf("创建错误日志文件失败: %v", err)
	}
	p.errorWriter = errorFile

	// 创建命令
	ctx, cancel := context.WithCancel(context.Background())
	p.cancelFunc = cancel

	var cmd *exec.Cmd
	if strings.HasSuffix(p.Script, ".js") || strings.HasSuffix(p.Script, ".ts") {
		// Node.js 脚本
		args := append([]string{p.Script}, p.Args...)
		cmd = exec.CommandContext(ctx, "node", args...)
	} else if strings.HasSuffix(p.Script, ".py") {
		// Python 脚本
		args := append([]string{p.Script}, p.Args...)
		cmd = exec.CommandContext(ctx, "python", args...)
	} else if strings.HasSuffix(p.Script, ".go") {
		// Go 脚本
		args := append([]string{"run", p.Script}, p.Args...)
		cmd = exec.CommandContext(ctx, "go", args...)
	} else {
		// 其他可执行文件
		cmd = exec.CommandContext(ctx, p.Script, p.Args...)
	}

	// 设置工作目录
	cmd.Dir = p.Cwd

	// 设置环境变量
	cmd.Env = os.Environ()
	for key, value := range p.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	// 设置标准输出和错误输出
	cmd.Stdout = p.logWriter
	cmd.Stderr = p.errorWriter

	// 启动进程
	err = cmd.Start()
	if err != nil {
		p.logWriter.Close()
		p.errorWriter.Close()
		return fmt.Errorf("启动命令失败: %v", err)
	}

	p.cmd = cmd
	p.PID = cmd.Process.Pid
	p.Status = StatusOnline
	p.StartTime = time.Now()

	// 保存PID文件
	pidFile := filepath.Join(pm.dataDir, "pids", fmt.Sprintf("%s.pid", p.Name))
	os.WriteFile(pidFile, []byte(strconv.Itoa(p.PID)), 0644)

	// 启动守护协程
	go pm.watchProcess(p)

	return nil
}

// StopProcess 停止进程
func (pm *ProcessManager) StopProcess(nameOrID string) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	process := pm.findProcess(nameOrID)
	if process == nil {
		return fmt.Errorf("未找到进程: %s", nameOrID)
	}

	return pm.stopProcessInstance(process)
}

// stopProcessInstance 停止单个进程实例
func (pm *ProcessManager) stopProcessInstance(p *Process) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.Status != StatusOnline {
		return fmt.Errorf("进程 '%s' 当前状态为 %s，无法停止", p.Name, p.Status)
	}

	p.Status = StatusStopping

	// 停止文件监控
	if p.Watch && p.watcherStop != nil {
		select {
		case p.watcherStop <- true:
		default:
		}
	}

	// 取消上下文
	if p.cancelFunc != nil {
		p.cancelFunc()
	}

	// 尝试优雅关闭
	if p.cmd != nil && p.cmd.Process != nil {
		// 发送SIGTERM信号
		err := p.cmd.Process.Signal(syscall.SIGTERM)
		if err == nil {
			// 等待5秒让进程优雅退出
			done := make(chan error, 1)
			go func() {
				done <- p.cmd.Wait()
			}()

			select {
			case <-time.After(5 * time.Second):
				// 强制杀死进程
				p.cmd.Process.Kill()
				<-done
			case <-done:
				// 进程已优雅退出
			}
		} else {
			// 直接杀死进程
			p.cmd.Process.Kill()
			p.cmd.Wait()
		}
	}

	p.Status = StatusStopped
	p.PID = 0

	// 关闭日志文件
	if p.logWriter != nil {
		p.logWriter.Close()
		p.logWriter = nil
	}
	if p.errorWriter != nil {
		p.errorWriter.Close()
		p.errorWriter = nil
	}

	// 删除PID文件
	pidFile := filepath.Join(pm.dataDir, "pids", fmt.Sprintf("%s.pid", p.Name))
	os.Remove(pidFile)

	pm.saveProcesses()
	return nil
}

// RestartProcess 重启进程
func (pm *ProcessManager) RestartProcess(nameOrID string) error {
	process := pm.findProcess(nameOrID)
	if process == nil {
		return fmt.Errorf("未找到进程: %s", nameOrID)
	}

	if process.Status == StatusOnline {
		err := pm.stopProcessInstance(process)
		if err != nil {
			return fmt.Errorf("停止进程失败: %v", err)
		}
	}

	// 等待一小段时间确保进程完全停止
	time.Sleep(500 * time.Millisecond)

	err := pm.startProcessInstance(process)
	if err != nil {
		return fmt.Errorf("重启进程失败: %v", err)
	}

	process.Restarts++
	pm.saveProcesses()
	return nil
}

// DeleteProcess 删除进程
func (pm *ProcessManager) DeleteProcess(nameOrID string) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	process := pm.findProcess(nameOrID)
	if process == nil {
		return fmt.Errorf("未找到进程: %s", nameOrID)
	}

	// 如果进程在运行，先停止它
	if process.Status == StatusOnline {
		pm.stopProcessInstance(process)
	}

	// 从进程列表中删除
	delete(pm.processes, process.ID)

	// 删除相关文件
	pidFile := filepath.Join(pm.dataDir, "pids", fmt.Sprintf("%s.pid", process.Name))
	os.Remove(pidFile)

	pm.saveProcesses()
	return nil
}

// GetProcessList 获取进程列表
func (pm *ProcessManager) GetProcessList() []*Process {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	processes := make([]*Process, 0, len(pm.processes))
	for _, p := range pm.processes {
		// 更新进程统计信息
		pm.updateProcessStats(p)
		processes = append(processes, p)
	}

	return processes
}

// findProcess 查找进程（通过名称或ID）
func (pm *ProcessManager) findProcess(nameOrID string) *Process {
	// 尝试按ID查找
	if id, err := strconv.Atoi(nameOrID); err == nil {
		if process, exists := pm.processes[id]; exists {
			return process
		}
	}

	// 按名称查找
	for _, process := range pm.processes {
		if process.Name == nameOrID {
			return process
		}
	}

	return nil
}

// watchProcess 守护进程，监控进程状态并处理自动重启
func (pm *ProcessManager) watchProcess(p *Process) {
	// 添加调试日志
	if p.logWriter != nil {
		logMsg := fmt.Sprintf("[%s] 开始监控进程 '%s' (PID: %d)",
			time.Now().Format("2006-01-02 15:04:05"), p.Name, p.PID)
		p.logWriter.WriteString(logMsg + "\n")
	}

	for {
		if p.cmd == nil {
			if p.logWriter != nil {
				logMsg := fmt.Sprintf("[%s] watchProcess: cmd 为空，退出监控",
					time.Now().Format("2006-01-02 15:04:05"))
				p.logWriter.WriteString(logMsg + "\n")
			}
			break
		}

		// 添加调试日志
		if p.logWriter != nil {
			logMsg := fmt.Sprintf("[%s] 等待进程退出...",
				time.Now().Format("2006-01-02 15:04:05"))
			p.logWriter.WriteString(logMsg + "\n")
		}

		err := p.cmd.Wait()

		// 添加调试日志
		if p.logWriter != nil {
			logMsg := fmt.Sprintf("[%s] 进程退出，错误: %v",
				time.Now().Format("2006-01-02 15:04:05"), err)
			p.logWriter.WriteString(logMsg + "\n")
		}

		p.mutex.Lock()
		if p.Status == StatusStopping || p.Status == StatusStopped {
			if p.logWriter != nil {
				logMsg := fmt.Sprintf("[%s] 进程状态为停止中或已停止，退出监控",
					time.Now().Format("2006-01-02 15:04:05"))
				p.logWriter.WriteString(logMsg + "\n")
			}
			p.mutex.Unlock()
			break
		}

		// 进程意外退出
		p.Status = StatusErrored
		p.PID = 0

		// 记录调试信息
		if p.logWriter != nil {
			logMsg := fmt.Sprintf("[%s] 进程意外退出，当前重启次数: %d，最大重启次数: %d",
				time.Now().Format("2006-01-02 15:04:05"), p.Restarts, p.MaxRestarts)
			p.logWriter.WriteString(logMsg + "\n")
		}

		// 检查是否应该重启
		if p.Restarts < p.MaxRestarts {
			// 检查最小运行时间
			if time.Since(p.StartTime) < p.MinUptime {
				// 如果运行时间太短，等待一段时间再重启
				if p.logWriter != nil {
					logMsg := fmt.Sprintf("[%s] 运行时间太短 (%v < %v)，等待1秒后重启",
						time.Now().Format("2006-01-02 15:04:05"), time.Since(p.StartTime), p.MinUptime)
					p.logWriter.WriteString(logMsg + "\n")
				}
				p.mutex.Unlock()
				time.Sleep(1 * time.Second)
				p.mutex.Lock()
			}

			p.Restarts++
			p.mutex.Unlock()

			// 记录重启日志
			logMsg := fmt.Sprintf("[%s] 进程意外退出 (错误: %v)，正在重启... (第 %d 次)",
				time.Now().Format("2006-01-02 15:04:05"), err, p.Restarts)
			if p.logWriter != nil {
				p.logWriter.WriteString(logMsg + "\n")
			}

			// 保存进程状态
			pm.saveProcesses()

			// 重启进程
			time.Sleep(1 * time.Second)
			restartErr := pm.startProcessInstance(p)
			if restartErr != nil {
				p.mutex.Lock()
				p.Status = StatusErrored
				if p.logWriter != nil {
					logMsg := fmt.Sprintf("[%s] 重启失败: %v",
						time.Now().Format("2006-01-02 15:04:05"), restartErr)
					p.logWriter.WriteString(logMsg + "\n")
				}
				p.mutex.Unlock()
				break
			} else {
				if p.logWriter != nil {
					logMsg := fmt.Sprintf("[%s] 重启成功 (PID: %d)",
						time.Now().Format("2006-01-02 15:04:05"), p.PID)
					p.logWriter.WriteString(logMsg + "\n")
				}
			}
		} else {
			// 达到最大重启次数
			p.Status = StatusErrored
			logMsg := fmt.Sprintf("[%s] 进程 '%s' 达到最大重启次数 (%d)，停止自动重启",
				time.Now().Format("2006-01-02 15:04:05"), p.Name, p.MaxRestarts)
			if p.logWriter != nil {
				p.logWriter.WriteString(logMsg + "\n")
			}
			p.mutex.Unlock()
			break
		}
	}

	// 监控退出日志
	if p.logWriter != nil {
		logMsg := fmt.Sprintf("[%s] 退出进程监控 '%s'",
			time.Now().Format("2006-01-02 15:04:05"), p.Name)
		p.logWriter.WriteString(logMsg + "\n")
	}
}

// updateProcessStats 更新进程统计信息
func (pm *ProcessManager) updateProcessStats(p *Process) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.Status != StatusOnline || p.PID == 0 {
		p.CPUUsage = 0
		p.MemoryUsage = 0
		p.Uptime = 0
		return
	}

	// 使用gopsutil获取进程信息
	proc, err := process.NewProcess(int32(p.PID))
	if err != nil {
		return
	}

	// 获取CPU使用率
	if cpu, err := proc.CPUPercent(); err == nil {
		p.CPUUsage = cpu
	}

	// 获取内存使用量
	if memInfo, err := proc.MemoryInfo(); err == nil {
		p.MemoryUsage = memInfo.RSS
	}

	// 计算运行时间
	if !p.StartTime.IsZero() {
		p.Uptime = time.Since(p.StartTime)
	}
}

// saveProcesses 保存进程信息到文件
func (pm *ProcessManager) saveProcesses() {
	data, err := json.MarshalIndent(pm.processes, "", "  ")
	if err != nil {
		return
	}

	processFile := filepath.Join(pm.dataDir, "processes.json")
	os.WriteFile(processFile, data, 0644)
}

// commandLoop 守护进程命令处理循环
func (pm *ProcessManager) commandLoop() {
	commandDir := filepath.Join(pm.dataDir, "commands")
	os.MkdirAll(commandDir, 0755)

	for {
		// 检查命令文件
		files, err := os.ReadDir(commandDir)
		if err != nil {
			time.Sleep(1 * time.Second)
			continue
		}

		for _, file := range files {
			if !strings.HasSuffix(file.Name(), ".cmd") {
				continue
			}

			cmdFile := filepath.Join(commandDir, file.Name())
			pm.processCommand(cmdFile)
		}

		time.Sleep(1 * time.Second)
	}
}

// processCommand 处理单个命令文件
func (pm *ProcessManager) processCommand(cmdFile string) {
	// 读取命令内容
	data, err := os.ReadFile(cmdFile)
	if err != nil {
		return
	}

	// 删除命令文件
	defer os.Remove(cmdFile)

	// 解析命令
	parts := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(parts) < 1 {
		return
	}

	command := parts[0]
	responseFile := strings.Replace(cmdFile, ".cmd", ".resp", 1)

	switch command {
	case "START":
		if len(parts) >= 2 {
			// 解析配置（简化版）
			var config AppConfig
			err := json.Unmarshal([]byte(parts[1]), &config)
			if err != nil {
				os.WriteFile(responseFile, []byte("ERROR: "+err.Error()), 0644)
				return
			}

			process, err := pm.StartProcess(config)
			if err != nil {
				os.WriteFile(responseFile, []byte("ERROR: "+err.Error()), 0644)
				return
			}

			response := fmt.Sprintf("SUCCESS: 启动 '%s' (ID: %d)", process.Name, process.ID)
			os.WriteFile(responseFile, []byte(response), 0644)
		}

	case "STOP":
		if len(parts) >= 2 {
			nameOrID := parts[1]
			err := pm.StopProcess(nameOrID)
			if err != nil {
				os.WriteFile(responseFile, []byte("ERROR: "+err.Error()), 0644)
				return
			}
			response := fmt.Sprintf("SUCCESS: 停止 '%s'", nameOrID)
			os.WriteFile(responseFile, []byte(response), 0644)
		}

	case "RESTART":
		if len(parts) >= 2 {
			nameOrID := parts[1]
			err := pm.RestartProcess(nameOrID)
			if err != nil {
				os.WriteFile(responseFile, []byte("ERROR: "+err.Error()), 0644)
				return
			}
			response := fmt.Sprintf("SUCCESS: 重启 '%s'", nameOrID)
			os.WriteFile(responseFile, []byte(response), 0644)
		}

	case "DELETE":
		if len(parts) >= 2 {
			nameOrID := parts[1]
			err := pm.DeleteProcess(nameOrID)
			if err != nil {
				os.WriteFile(responseFile, []byte("ERROR: "+err.Error()), 0644)
				return
			}
			response := fmt.Sprintf("SUCCESS: 删除 '%s'", nameOrID)
			os.WriteFile(responseFile, []byte(response), 0644)
		}

	case "LIST":
		processes := pm.GetProcessList()
		data, err := json.Marshal(processes)
		if err != nil {
			os.WriteFile(responseFile, []byte("ERROR: "+err.Error()), 0644)
			return
		}
		os.WriteFile(responseFile, data, 0644)

	case "LOGS":
		if len(parts) >= 4 {
			nameOrID := parts[1]
			linesStr := parts[2]
			followStr := parts[3]
			showErrorStr := ""
			if len(parts) >= 5 {
				showErrorStr = parts[4]
			}

			lines, _ := strconv.Atoi(linesStr)
			follow := followStr == "true"
			showError := showErrorStr == "true"

			var err error
			if follow {
				// 对于follow模式，我们需要特殊处理
				err = pm.handleFollowLogs(responseFile, nameOrID, lines, showError)
			} else {
				if showError {
					err = pm.GetErrorLogs(nameOrID, lines, false)
				} else {
					err = pm.GetLogs(nameOrID, lines, false)
				}
			}

			if err != nil {
				os.WriteFile(responseFile, []byte("ERROR: "+err.Error()), 0644)
				return
			}

			if !follow {
				os.WriteFile(responseFile, []byte("SUCCESS: 日志显示完成"), 0644)
			}
		}

	default:
		os.WriteFile(responseFile, []byte("ERROR: 未知命令"), 0644)
	}
}

// handleFollowLogs 处理实时日志跟踪
func (pm *ProcessManager) handleFollowLogs(responseFile, nameOrID string, lines int, showError bool) error {
	process := pm.findProcess(nameOrID)
	if process == nil {
		return fmt.Errorf("未找到进程: %s", nameOrID)
	}

	var logFile string
	if showError {
		logFile = process.ErrorFile
		if logFile == "" {
			logFile = filepath.Join(pm.dataDir, "logs", fmt.Sprintf("%s-error.log", process.Name))
		}
	} else {
		logFile = process.LogFile
		if logFile == "" {
			logFile = filepath.Join(pm.dataDir, "logs", fmt.Sprintf("%s.log", process.Name))
		}
	}

	// 创建follow日志的协程
	go func() {
		defer func() {
			// 发送完成信号
			os.WriteFile(responseFile, []byte("SUCCESS: 日志跟踪结束"), 0644)
		}()

		// 首先显示最后N行（如果指定了）
		if lines > 0 {
			if _, err := os.Stat(logFile); err == nil {
				file, err := os.Open(logFile)
				if err == nil {
					tailLines, err := pm.readLastLines(file, lines)
					file.Close()
					if err == nil {
						for _, line := range tailLines {
							fmt.Println(line)
						}
					}
				}
			}
		}

		fmt.Printf("\n==> 正在跟踪日志文件: %s (按 Ctrl+C 退出)\n", logFile)

		// 开始跟踪新内容
		pm.followLogFile(logFile)
	}()

	return nil
}

// followLogFile 跟踪日志文件的新内容
func (pm *ProcessManager) followLogFile(logFile string) {
	var file *os.File
	var lastSize int64

	for {
		// 检查文件是否存在
		fileInfo, err := os.Stat(logFile)
		if err != nil {
			if !os.IsNotExist(err) {
				time.Sleep(1 * time.Second)
				continue
			}
			// 文件不存在，等待创建
			time.Sleep(1 * time.Second)
			continue
		}

		// 如果文件大小发生变化或文件刚打开
		if file == nil || fileInfo.Size() != lastSize {
			if file != nil {
				file.Close()
			}

			file, err = os.Open(logFile)
			if err != nil {
				time.Sleep(1 * time.Second)
				continue
			}

			// 如果文件变小了（可能被轮转），从头开始读
			if fileInfo.Size() < lastSize {
				lastSize = 0
			}

			// 移动到上次读取的位置
			file.Seek(lastSize, 0)

			// 读取新内容
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				fmt.Println(scanner.Text())
			}

			// 更新文件位置
			newPos, _ := file.Seek(0, 1) // 获取当前位置
			lastSize = newPos
		}

		time.Sleep(100 * time.Millisecond)
	}
}

// sendCommand 发送命令给守护进程
func (pm *ProcessManager) sendCommand(command string, args ...string) (string, error) {
	commandDir := filepath.Join(pm.dataDir, "commands")
	os.MkdirAll(commandDir, 0755)

	// 创建唯一的命令文件名
	timestamp := time.Now().UnixNano()
	cmdFile := filepath.Join(commandDir, fmt.Sprintf("%d.cmd", timestamp))
	responseFile := strings.Replace(cmdFile, ".cmd", ".resp", 1)

	// 准备命令内容
	cmdContent := command
	for _, arg := range args {
		cmdContent += "\n" + arg
	}

	// 写入命令文件
	err := os.WriteFile(cmdFile, []byte(cmdContent), 0644)
	if err != nil {
		return "", err
	}

	// 等待响应文件
	for i := 0; i < 100; i++ { // 最多等待10秒
		time.Sleep(100 * time.Millisecond)
		if _, err := os.Stat(responseFile); err == nil {
			// 读取响应
			data, err := os.ReadFile(responseFile)
			os.Remove(responseFile)
			if err != nil {
				return "", err
			}
			return string(data), nil
		}
	}

	// 清理命令文件
	os.Remove(cmdFile)
	return "", fmt.Errorf("守护进程响应超时")
}

// checkStartSignals 检查并处理启动信号
func (pm *ProcessManager) checkStartSignals() {
	signalFile := filepath.Join(pm.dataDir, "start_signal")

	// 检查信号文件是否存在
	if _, err := os.Stat(signalFile); os.IsNotExist(err) {
		return
	}

	// 读取信号文件
	data, err := os.ReadFile(signalFile)
	if err != nil {
		return
	}

	// 解析进程ID
	idStr := strings.TrimSpace(string(data))
	id, err := strconv.Atoi(idStr)
	if err != nil {
		os.Remove(signalFile)
		return
	}

	// 删除信号文件
	os.Remove(signalFile)

	// 查找进程
	pm.mutex.Lock()
	process, exists := pm.processes[id]
	pm.mutex.Unlock()

	if !exists {
		return
	}

	// 启动进程
	if process.Status == StatusStopped {
		err := pm.startProcessInstance(process)
		if err != nil {
			process.mutex.Lock()
			process.Status = StatusErrored
			process.mutex.Unlock()
		}
	}
}

// loadProcesses 从文件加载进程信息
func (pm *ProcessManager) loadProcesses() {
	processFile := filepath.Join(pm.dataDir, "processes.json")

	data, err := os.ReadFile(processFile)
	if err != nil {
		return
	}

	var processes map[int]*Process
	err = json.Unmarshal(data, &processes)
	if err != nil {
		return
	}

	// 恢复进程状态
	for id, p := range processes {
		// 检查进程是否仍在运行
		if p.PID > 0 {
			if proc, err := process.NewProcess(int32(p.PID)); err == nil {
				if exists, _ := proc.IsRunning(); exists {
					p.Status = StatusOnline
					// 重新启动守护协程
					go pm.watchProcess(p)
				} else {
					p.Status = StatusStopped
					p.PID = 0
				}
			} else {
				p.Status = StatusStopped
				p.PID = 0
			}
		} else {
			p.Status = StatusStopped
		}

		p.watcherStop = make(chan bool, 1)
		pm.processes[id] = p

		if id >= pm.nextID {
			pm.nextID = id + 1
		}

		// 如果进程应该在运行但当前已停止，尝试重启
		// 这处理了系统重启后的自动恢复场景
		if p.Status == StatusStopped && !p.StartTime.IsZero() {
			// 延迟一小段时间后尝试重启
			go func(process *Process) {
				time.Sleep(2 * time.Second)
				err := pm.startProcessInstance(process)
				if err != nil {
					process.mutex.Lock()
					process.Status = StatusErrored
					process.mutex.Unlock()
				}
			}(p)
		}
	}
}
