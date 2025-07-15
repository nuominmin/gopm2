package main

import (
	"os"
	"os/exec"
	"sync"
	"time"
)

// ProcessStatus 进程状态枚举
type ProcessStatus string

const (
	StatusOnline   ProcessStatus = "online"
	StatusStopped  ProcessStatus = "stopped"
	StatusStopping ProcessStatus = "stopping"
	StatusErrored  ProcessStatus = "errored"
	StatusOneTime  ProcessStatus = "one-time"
)

// ExecMode 执行模式
type ExecMode string

const (
	ExecModeFork    ExecMode = "fork"
	ExecModeCluster ExecMode = "cluster"
)

// Process 进程信息结构
type Process struct {
	ID          int               `json:"id"`
	Name        string            `json:"name"`
	Script      string            `json:"script"`
	Args        []string          `json:"args"`
	Cwd         string            `json:"cwd"`
	Env         map[string]string `json:"env"`
	Instances   int               `json:"instances"`
	ExecMode    ExecMode          `json:"exec_mode"`
	Status      ProcessStatus     `json:"status"`
	PID         int               `json:"pid"`
	CPUUsage    float64           `json:"cpu_usage"`
	MemoryUsage uint64            `json:"memory_usage"`
	Uptime      time.Duration     `json:"uptime"`
	Restarts    int               `json:"restarts"`
	StartTime   time.Time         `json:"start_time"`
	LogFile     string            `json:"log_file"`
	ErrorFile   string            `json:"error_file"`
	Watch       bool              `json:"watch"`
	WatchIgnore []string          `json:"watch_ignore"`
	MaxRestarts int               `json:"max_restarts"`
	MinUptime   time.Duration     `json:"min_uptime"`

	// 内部字段
	cmd         *exec.Cmd    `json:"-"`
	mutex       sync.RWMutex `json:"-"`
	logWriter   *os.File     `json:"-"`
	errorWriter *os.File     `json:"-"`
	cancelFunc  func()       `json:"-"`
	watcherStop chan bool    `json:"-"`
}

// Config 配置文件结构
type Config struct {
	Apps []AppConfig `json:"apps" yaml:"apps"`
}

// AppConfig 应用配置
type AppConfig struct {
	Name        string            `json:"name" yaml:"name"`
	Script      string            `json:"script" yaml:"script"`
	Args        []string          `json:"args,omitempty" yaml:"args,omitempty"`
	Cwd         string            `json:"cwd,omitempty" yaml:"cwd,omitempty"`
	Env         map[string]string `json:"env,omitempty" yaml:"env,omitempty"`
	Instances   int               `json:"instances,omitempty" yaml:"instances,omitempty"`
	ExecMode    string            `json:"exec_mode,omitempty" yaml:"exec_mode,omitempty"`
	Watch       bool              `json:"watch,omitempty" yaml:"watch,omitempty"`
	WatchIgnore []string          `json:"watch_ignore,omitempty" yaml:"watch_ignore,omitempty"`
	LogFile     string            `json:"log_file,omitempty" yaml:"log_file,omitempty"`
	ErrorFile   string            `json:"error_file,omitempty" yaml:"error_file,omitempty"`
	MaxRestarts int               `json:"max_restarts,omitempty" yaml:"max_restarts,omitempty"`
	MinUptime   string            `json:"min_uptime,omitempty" yaml:"min_uptime,omitempty"`
}

// ProcessManager 进程管理器
type ProcessManager struct {
	processes map[int]*Process
	nextID    int
	mutex     sync.RWMutex
	dataDir   string
}

// LogEntry 日志条目
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	ProcessID int       `json:"process_id"`
	Message   string    `json:"message"`
}

// Stats 进程统计信息
type Stats struct {
	CPU    float64 `json:"cpu"`
	Memory uint64  `json:"memory"`
	Uptime string  `json:"uptime"`
}
