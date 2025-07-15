package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// LoadConfig 从文件加载配置
func LoadConfig(configPath string) (*Config, error) {
	// 检查文件是否存在
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("配置文件不存在: %s", configPath)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %v", err)
	}

	config := &Config{}

	// 根据文件扩展名选择解析器
	ext := strings.ToLower(filepath.Ext(configPath))
	switch ext {
	case ".json":
		err = json.Unmarshal(data, config)
	case ".yml", ".yaml":
		err = yaml.Unmarshal(data, config)
	default:
		// 尝试JSON解析
		err = json.Unmarshal(data, config)
		if err != nil {
			// 如果JSON解析失败，尝试YAML
			err = yaml.Unmarshal(data, config)
		}
	}

	if err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %v", err)
	}

	// 验证配置
	err = validateConfig(config)
	if err != nil {
		return nil, fmt.Errorf("配置验证失败: %v", err)
	}

	return config, nil
}

// SaveConfig 保存配置到文件
func SaveConfig(config *Config, configPath string) error {
	var data []byte
	var err error

	// 根据文件扩展名选择格式
	ext := strings.ToLower(filepath.Ext(configPath))
	switch ext {
	case ".yml", ".yaml":
		data, err = yaml.Marshal(config)
	default:
		data, err = json.MarshalIndent(config, "", "  ")
	}

	if err != nil {
		return fmt.Errorf("序列化配置失败: %v", err)
	}

	err = os.WriteFile(configPath, data, 0644)
	if err != nil {
		return fmt.Errorf("写入配置文件失败: %v", err)
	}

	return nil
}

// validateConfig 验证配置文件
func validateConfig(config *Config) error {
	if len(config.Apps) == 0 {
		return fmt.Errorf("配置文件中没有定义应用")
	}

	appNames := make(map[string]bool)
	for i, app := range config.Apps {
		// 检查必需字段
		if app.Name == "" {
			return fmt.Errorf("应用 %d: 名称不能为空", i)
		}
		if app.Script == "" {
			return fmt.Errorf("应用 '%s': 脚本路径不能为空", app.Name)
		}

		// 检查名称唯一性
		if appNames[app.Name] {
			return fmt.Errorf("应用名称 '%s' 重复", app.Name)
		}
		appNames[app.Name] = true

		// 检查脚本文件是否存在
		if _, err := os.Stat(app.Script); os.IsNotExist(err) {
			return fmt.Errorf("应用 '%s': 脚本文件不存在: %s", app.Name, app.Script)
		}

		// 验证instances数量
		if app.Instances < 0 {
			return fmt.Errorf("应用 '%s': instances 不能为负数", app.Name)
		}

		// 验证执行模式
		if app.ExecMode != "" && app.ExecMode != "fork" && app.ExecMode != "cluster" {
			return fmt.Errorf("应用 '%s': 不支持的执行模式: %s", app.Name, app.ExecMode)
		}
	}

	return nil
}

// GenerateConfigTemplate 生成配置文件模板
func GenerateConfigTemplate(configPath string) error {
	template := &Config{
		Apps: []AppConfig{
			{
				Name:   "example-app",
				Script: "./app.js",
				Args:   []string{"--port", "3000"},
				Cwd:    "/path/to/app",
				Env: map[string]string{
					"NODE_ENV": "production",
					"PORT":     "3000",
				},
				Instances:   2,
				ExecMode:    "cluster",
				Watch:       true,
				WatchIgnore: []string{"node_modules", "logs"},
				LogFile:     "./logs/app.log",
				ErrorFile:   "./logs/app-error.log",
				MaxRestarts: 10,
				MinUptime:   "10s",
			},
		},
	}

	return SaveConfig(template, configPath)
}

// FromProcessToAppConfig 将Process转换为AppConfig
func FromProcessToAppConfig(p *Process) AppConfig {
	minUptime := ""
	if p.MinUptime > 0 {
		minUptime = p.MinUptime.String()
	}

	return AppConfig{
		Name:        p.Name,
		Script:      p.Script,
		Args:        p.Args,
		Cwd:         p.Cwd,
		Env:         p.Env,
		Instances:   p.Instances,
		ExecMode:    string(p.ExecMode),
		Watch:       p.Watch,
		WatchIgnore: p.WatchIgnore,
		LogFile:     p.LogFile,
		ErrorFile:   p.ErrorFile,
		MaxRestarts: p.MaxRestarts,
		MinUptime:   minUptime,
	}
}

// ExportConfig 导出当前运行的进程为配置文件
func (pm *ProcessManager) ExportConfig(configPath string) error {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	config := &Config{
		Apps: make([]AppConfig, 0, len(pm.processes)),
	}

	for _, p := range pm.processes {
		config.Apps = append(config.Apps, FromProcessToAppConfig(p))
	}

	return SaveConfig(config, configPath)
}
