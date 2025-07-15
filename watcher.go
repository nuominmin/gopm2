package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

// startFileWatcher 启动文件监控器
func (pm *ProcessManager) startFileWatcher(p *Process) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return
	}
	defer watcher.Close()

	// 添加监控目录
	watchDirs := []string{p.Cwd}
	if p.Cwd == "" {
		workDir, _ := os.Getwd()
		watchDirs = []string{workDir}
	}

	// 递归添加子目录
	for _, dir := range watchDirs {
		err := pm.addWatchDirRecursive(watcher, dir, p.WatchIgnore)
		if err != nil {
			return
		}
	}

	// 防抖动机制
	var lastEvent time.Time
	debounceInterval := 1 * time.Second

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}

			// 检查是否应该忽略这个文件
			if pm.shouldIgnoreFile(event.Name, p.WatchIgnore) {
				continue
			}

			// 只关注写入和创建事件
			if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
				now := time.Now()

				// 防抖动：如果距离上次事件时间过短，则忽略
				if now.Sub(lastEvent) < debounceInterval {
					continue
				}
				lastEvent = now

				// 延迟一小段时间确保文件写入完成
				time.Sleep(100 * time.Millisecond)

				// 重启进程
				if p.Status == StatusOnline {
					logMsg := fmt.Sprintf("[%s] 检测到文件变更: %s，正在重启进程...",
						time.Now().Format("2006-01-02 15:04:05"), event.Name)
					if p.logWriter != nil {
						p.logWriter.WriteString(logMsg + "\n")
					}

					go func() {
						pm.RestartProcess(p.Name)
					}()
				}
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			if p.logWriter != nil {
				p.logWriter.WriteString(fmt.Sprintf("文件监控错误: %v\n", err))
			}

		case <-p.watcherStop:
			return
		}
	}
}

// addWatchDirRecursive 递归添加监控目录
func (pm *ProcessManager) addWatchDirRecursive(watcher *fsnotify.Watcher, dir string, ignorePatterns []string) error {
	// 检查是否应该忽略这个目录
	if pm.shouldIgnoreFile(dir, ignorePatterns) {
		return nil
	}

	err := watcher.Add(dir)
	if err != nil {
		return err
	}

	// 遍历子目录
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() && path != dir {
			// 检查是否应该忽略这个子目录
			if !pm.shouldIgnoreFile(path, ignorePatterns) {
				watcher.Add(path)
			}
		}
		return nil
	})

	return nil
}

// shouldIgnoreFile 检查文件是否应该被忽略
func (pm *ProcessManager) shouldIgnoreFile(filename string, ignorePatterns []string) bool {
	// 默认忽略的文件和目录
	defaultIgnore := []string{
		"node_modules",
		".git",
		".svn",
		".hg",
		".DS_Store",
		"*.log",
		"*.tmp",
		"*.temp",
		"*.swp",
		"*.swo",
		"*~",
		".gopm2",
	}

	allIgnorePatterns := append(defaultIgnore, ignorePatterns...)

	for _, pattern := range allIgnorePatterns {
		// 支持简单的通配符匹配
		if matched, _ := filepath.Match(pattern, filepath.Base(filename)); matched {
			return true
		}

		// 检查是否包含忽略的路径片段
		if strings.Contains(filename, pattern) {
			return true
		}
	}

	return false
}

// EnableWatch 为进程启用文件监控
func (pm *ProcessManager) EnableWatch(nameOrID string) error {
	process := pm.findProcess(nameOrID)
	if process == nil {
		return fmt.Errorf("未找到进程: %s", nameOrID)
	}

	process.mutex.Lock()
	defer process.mutex.Unlock()

	if process.Watch {
		return fmt.Errorf("进程 '%s' 已经启用文件监控", process.Name)
	}

	process.Watch = true

	if process.Status == StatusOnline {
		go pm.startFileWatcher(process)
	}

	pm.saveProcesses()
	return nil
}

// DisableWatch 为进程禁用文件监控
func (pm *ProcessManager) DisableWatch(nameOrID string) error {
	process := pm.findProcess(nameOrID)
	if process == nil {
		return fmt.Errorf("未找到进程: %s", nameOrID)
	}

	process.mutex.Lock()
	defer process.mutex.Unlock()

	if !process.Watch {
		return fmt.Errorf("进程 '%s' 未启用文件监控", process.Name)
	}

	process.Watch = false

	// 停止文件监控
	if process.watcherStop != nil {
		select {
		case process.watcherStop <- true:
		default:
		}
	}

	pm.saveProcesses()
	return nil
}
