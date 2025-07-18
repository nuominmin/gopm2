package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// LogManager 日志管理器
type LogManager struct {
	dataDir string
}

// NewLogManager 创建日志管理器
func NewLogManager(dataDir string) *LogManager {
	return &LogManager{
		dataDir: dataDir,
	}
}

// GetLogs 获取进程日志
func (pm *ProcessManager) GetLogs(nameOrID string, lines int, follow bool) error {
	process := pm.findProcess(nameOrID)
	if process == nil {
		return fmt.Errorf("未找到进程: %s", nameOrID)
	}

	// 优先显示标准输出日志
	logFile := process.LogFile
	if logFile == "" {
		logFile = filepath.Join(pm.dataDir, "logs", fmt.Sprintf("%s.log", process.Name))
	}

	if follow {
		return pm.followLogs(logFile, lines)
	} else {
		return pm.showLogs(logFile, lines)
	}
}

// GetErrorLogs 获取进程错误日志
func (pm *ProcessManager) GetErrorLogs(nameOrID string, lines int, follow bool) error {
	process := pm.findProcess(nameOrID)
	if process == nil {
		return fmt.Errorf("未找到进程: %s", nameOrID)
	}

	errorFile := process.ErrorFile
	if errorFile == "" {
		errorFile = filepath.Join(pm.dataDir, "logs", fmt.Sprintf("%s-error.log", process.Name))
	}

	if follow {
		return pm.followLogs(errorFile, lines)
	} else {
		return pm.showLogs(errorFile, lines)
	}
}

// showLogs 显示日志文件的最后N行
func (pm *ProcessManager) showLogs(logFile string, lines int) error {
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		fmt.Printf("日志文件不存在: %s\n", logFile)
		return nil
	}

	file, err := os.Open(logFile)
	if err != nil {
		return fmt.Errorf("打开日志文件失败: %v", err)
	}
	defer file.Close()

	// 如果lines为0，显示所有内容
	if lines == 0 {
		_, err := io.Copy(os.Stdout, file)
		return err
	}

	// 读取最后N行
	tailLines, err := pm.readLastLines(file, lines)
	if err != nil {
		return fmt.Errorf("读取日志失败: %v", err)
	}

	for _, line := range tailLines {
		fmt.Println(line)
	}

	return nil
}

// followLogs 实时跟踪日志文件
func (pm *ProcessManager) followLogs(logFile string, lines int) error {
	// 首先显示最后N行
	if lines > 0 {
		pm.showLogs(logFile, lines)
	}

	// 然后开始跟踪新内容
	file, err := os.Open(logFile)
	if err != nil {
		return fmt.Errorf("打开日志文件失败: %v", err)
	}
	defer file.Close()

	// 移动到文件末尾
	file.Seek(0, io.SeekEnd)

	fmt.Printf("\n==> 正在跟踪日志文件: %s (按 Ctrl+C 退出)\n", logFile)

	scanner := bufio.NewScanner(file)
	for {
		for scanner.Scan() {
			fmt.Println(scanner.Text())
		}

		// 等待一小段时间再检查新内容
		time.Sleep(100 * time.Millisecond)

		// 检查文件是否被轮转或删除
		if _, err := os.Stat(logFile); os.IsNotExist(err) {
			break
		}
	}

	return scanner.Err()
}

// clientFollowLogs 客户端实时跟踪日志文件
func (pm *ProcessManager) clientFollowLogs(logFile string, lines int) error {
	// 首先显示最后N行
	if lines > 0 {
		if _, err := os.Stat(logFile); err == nil {
			pm.showLogs(logFile, lines)
		}
	}

	fmt.Printf("\n==> 正在跟踪日志文件: %s (按 Ctrl+C 退出)\n", logFile)

	var file *os.File
	var lastSize int64 = 0

	// 如果文件存在，获取当前大小并移动到末尾
	if fileInfo, err := os.Stat(logFile); err == nil {
		lastSize = fileInfo.Size()
	}

	for {
		// 检查文件是否存在
		fileInfo, err := os.Stat(logFile)
		if err != nil {
			if os.IsNotExist(err) {
				// 文件不存在，等待创建
				time.Sleep(500 * time.Millisecond)
				continue
			}
			return fmt.Errorf("访问日志文件失败: %v", err)
		}

		// 如果文件大小发生变化
		if fileInfo.Size() != lastSize {
			// 重新打开文件
			if file != nil {
				file.Close()
			}

			file, err = os.Open(logFile)
			if err != nil {
				time.Sleep(500 * time.Millisecond)
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

			// 更新文件大小
			lastSize = fileInfo.Size()
		}

		time.Sleep(100 * time.Millisecond)
	}
}

// readLastLines 读取文件的最后N行
func (pm *ProcessManager) readLastLines(file *os.File, lines int) ([]string, error) {
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, err
	}

	fileSize := fileInfo.Size()
	if fileSize == 0 {
		return []string{}, nil
	}

	// 从文件末尾开始逐步向前读取
	buffer := make([]byte, 1024)
	var result []string
	var currentLine strings.Builder
	position := fileSize

	for len(result) < lines && position > 0 {
		// 计算要读取的字节数
		readSize := int64(len(buffer))
		if position < readSize {
			readSize = position
		}

		position -= readSize
		file.Seek(position, io.SeekStart)

		n, err := file.Read(buffer[:readSize])
		if err != nil && err != io.EOF {
			return nil, err
		}

		// 从后往前处理字节
		for i := n - 1; i >= 0; i-- {
			char := buffer[i]
			if char == '\n' {
				if currentLine.Len() > 0 {
					result = append([]string{currentLine.String()}, result...)
					currentLine.Reset()
					if len(result) >= lines {
						break
					}
				}
			} else {
				// 字符需要插入到行的开头
				line := currentLine.String()
				currentLine.Reset()
				currentLine.WriteByte(char)
				currentLine.WriteString(line)
			}
		}
	}

	// 处理最后一行（如果有）
	if currentLine.Len() > 0 && len(result) < lines {
		result = append([]string{currentLine.String()}, result...)
	}

	return result, nil
}

// ClearLogs 清空进程日志
func (pm *ProcessManager) ClearLogs(nameOrID string) error {
	process := pm.findProcess(nameOrID)
	if process == nil {
		return fmt.Errorf("未找到进程: %s", nameOrID)
	}

	// 清空标准输出日志
	if process.LogFile != "" {
		err := pm.clearLogFile(process.LogFile)
		if err != nil {
			return fmt.Errorf("清空日志文件失败: %v", err)
		}
	}

	// 清空错误日志
	if process.ErrorFile != "" {
		err := pm.clearLogFile(process.ErrorFile)
		if err != nil {
			return fmt.Errorf("清空错误日志文件失败: %v", err)
		}
	}

	return nil
}

// clearLogFile 清空日志文件
func (pm *ProcessManager) clearLogFile(logFile string) error {
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		return nil // 文件不存在，无需清空
	}

	// 以截断模式打开文件
	file, err := os.OpenFile(logFile, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	return nil
}

// RotateLogs 轮转日志文件
func (pm *ProcessManager) RotateLogs(nameOrID string, maxSize int64) error {
	process := pm.findProcess(nameOrID)
	if process == nil {
		return fmt.Errorf("未找到进程: %s", nameOrID)
	}

	// 轮转标准输出日志
	if process.LogFile != "" {
		err := pm.rotateLogFile(process.LogFile, maxSize)
		if err != nil {
			return fmt.Errorf("轮转日志文件失败: %v", err)
		}
	}

	// 轮转错误日志
	if process.ErrorFile != "" {
		err := pm.rotateLogFile(process.ErrorFile, maxSize)
		if err != nil {
			return fmt.Errorf("轮转错误日志文件失败: %v", err)
		}
	}

	return nil
}

// rotateLogFile 轮转单个日志文件
func (pm *ProcessManager) rotateLogFile(logFile string, maxSize int64) error {
	fileInfo, err := os.Stat(logFile)
	if os.IsNotExist(err) {
		return nil // 文件不存在，无需轮转
	}
	if err != nil {
		return err
	}

	// 检查文件大小
	if fileInfo.Size() < maxSize {
		return nil // 文件大小未达到轮转阈值
	}

	// 生成轮转后的文件名
	timestamp := time.Now().Format("20060102150405")
	rotatedFile := fmt.Sprintf("%s.%s", logFile, timestamp)

	// 重命名当前日志文件
	err = os.Rename(logFile, rotatedFile)
	if err != nil {
		return err
	}

	// 创建新的日志文件
	newFile, err := os.Create(logFile)
	if err != nil {
		// 如果创建失败，尝试恢复原文件
		os.Rename(rotatedFile, logFile)
		return err
	}
	newFile.Close()

	return nil
}

// GetLogFiles 获取所有日志文件列表
func (pm *ProcessManager) GetLogFiles() ([]string, error) {
	logsDir := filepath.Join(pm.dataDir, "logs")

	if _, err := os.Stat(logsDir); os.IsNotExist(err) {
		return []string{}, nil
	}

	files, err := os.ReadDir(logsDir)
	if err != nil {
		return nil, fmt.Errorf("读取日志目录失败: %v", err)
	}

	var logFiles []string
	for _, file := range files {
		if !file.IsDir() && (strings.HasSuffix(file.Name(), ".log") ||
			strings.Contains(file.Name(), "-error.log")) {
			logFiles = append(logFiles, filepath.Join(logsDir, file.Name()))
		}
	}

	sort.Strings(logFiles)
	return logFiles, nil
}

// CleanOldLogs 清理旧的日志文件
func (pm *ProcessManager) CleanOldLogs(days int) error {
	logsDir := filepath.Join(pm.dataDir, "logs")

	if _, err := os.Stat(logsDir); os.IsNotExist(err) {
		return nil // 日志目录不存在
	}

	cutoff := time.Now().AddDate(0, 0, -days)

	return filepath.Walk(logsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 只处理日志文件
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".log") {
			// 检查文件修改时间
			if info.ModTime().Before(cutoff) {
				fmt.Printf("删除旧日志文件: %s\n", path)
				return os.Remove(path)
			}
		}

		return nil
	})
}
