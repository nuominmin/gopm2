package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

var (
	version = "1.0.1"
	pm      *ProcessManager
	rootCmd = &cobra.Command{
		Use:     "gopm2",
		Version: version,
		Short:   "Go实现的进程管理器，类似PM2",
		Long: `GoPM2 是一个用Go语言实现的进程管理器，提供类似PM2的功能：
- 进程管理（启动、停止、重启、删除）
- 守护进程和自动重启
- 日志管理
- 文件监控
- 配置文件支持
- 集群模式`,
	}
)

func init() {
	pm = NewProcessManager()

	// start 命令
	var startCmd = &cobra.Command{
		Use:   "start [script|config]",
		Short: "启动应用或配置文件",
		Long:  "启动一个脚本文件或从配置文件启动多个应用",
		Args:  cobra.MinimumNArgs(1),
		Run:   runStart,
	}

	startCmd.Flags().StringP("name", "n", "", "应用名称")
	startCmd.Flags().StringArrayP("args", "a", []string{}, "传递给脚本的参数")
	startCmd.Flags().StringP("cwd", "c", "", "工作目录")
	startCmd.Flags().StringToStringP("env", "e", map[string]string{}, "环境变量 (key=value)")
	startCmd.Flags().IntP("instances", "i", 1, "实例数量")
	startCmd.Flags().StringP("exec-mode", "x", "fork", "执行模式 (fork|cluster)")
	startCmd.Flags().BoolP("watch", "w", false, "启用文件监控")
	startCmd.Flags().StringArrayP("ignore", "", []string{}, "监控时忽略的文件模式")
	startCmd.Flags().StringP("log", "l", "", "日志文件路径")
	startCmd.Flags().StringP("error", "", "", "错误日志文件路径")
	startCmd.Flags().IntP("max-restarts", "", 15, "最大重启次数")
	startCmd.Flags().StringP("min-uptime", "", "1s", "最小运行时间")

	// stop 命令
	var stopCmd = &cobra.Command{
		Use:   "stop <name|id>",
		Short: "停止应用",
		Args:  cobra.ExactArgs(1),
		Run:   runStop,
	}

	// restart 命令
	var restartCmd = &cobra.Command{
		Use:   "restart <name|id>",
		Short: "重启应用",
		Args:  cobra.ExactArgs(1),
		Run:   runRestart,
	}

	// delete/del 命令
	var deleteCmd = &cobra.Command{
		Use:     "delete <name|id>",
		Aliases: []string{"del"},
		Short:   "删除应用",
		Args:    cobra.ExactArgs(1),
		Run:     runDelete,
	}

	// list/ls 命令
	var listCmd = &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls", "status"},
		Short:   "显示所有应用状态",
		Run:     runList,
	}

	// logs 命令
	var logsCmd = &cobra.Command{
		Use:   "logs <name|id>",
		Short: "显示应用日志",
		Args:  cobra.ExactArgs(1),
		Run:   runLogs,
	}

	logsCmd.Flags().IntP("lines", "n", 50, "显示的行数")
	logsCmd.Flags().BoolP("follow", "f", false, "实时跟踪日志")
	logsCmd.Flags().BoolP("error", "e", false, "显示错误日志")

	// describe 命令
	var describeCmd = &cobra.Command{
		Use:   "describe <name|id>",
		Short: "显示应用详细信息",
		Args:  cobra.ExactArgs(1),
		Run:   runDescribe,
	}

	// monit 命令
	var monitCmd = &cobra.Command{
		Use:   "monit",
		Short: "实时监控所有应用",
		Run:   runMonit,
	}

	// flush 命令
	var flushCmd = &cobra.Command{
		Use:   "flush [name|id]",
		Short: "清空日志",
		Run:   runFlush,
	}

	// config 命令
	var configCmd = &cobra.Command{
		Use:   "config",
		Short: "配置文件相关操作",
	}

	var configGenerateCmd = &cobra.Command{
		Use:   "generate [file]",
		Short: "生成配置文件模板",
		Run:   runConfigGenerate,
	}

	var configExportCmd = &cobra.Command{
		Use:   "export [file]",
		Short: "导出当前配置",
		Run:   runConfigExport,
	}

	// startup 命令
	var startupCmd = &cobra.Command{
		Use:   "startup",
		Short: "生成系统启动脚本",
		Run:   runStartup,
	}

	// save 命令
	var saveCmd = &cobra.Command{
		Use:   "save",
		Short: "保存当前进程列表",
		Run:   runSave,
	}

	// resurrect 命令
	var resurrectCmd = &cobra.Command{
		Use:   "resurrect",
		Short: "恢复保存的进程列表",
		Run:   runResurrect,
	}

	// watch 相关命令
	var watchCmd = &cobra.Command{
		Use:   "watch",
		Short: "文件监控相关操作",
	}

	var watchEnableCmd = &cobra.Command{
		Use:   "enable <name|id>",
		Short: "启用文件监控",
		Args:  cobra.ExactArgs(1),
		Run:   runWatchEnable,
	}

	var watchDisableCmd = &cobra.Command{
		Use:   "disable <name|id>",
		Short: "禁用文件监控",
		Args:  cobra.ExactArgs(1),
		Run:   runWatchDisable,
	}

	// 添加子命令
	configCmd.AddCommand(configGenerateCmd, configExportCmd)
	watchCmd.AddCommand(watchEnableCmd, watchDisableCmd)

	rootCmd.AddCommand(
		startCmd, stopCmd, restartCmd, deleteCmd, listCmd,
		logsCmd, describeCmd, monitCmd, flushCmd,
		configCmd, startupCmd, saveCmd, resurrectCmd, watchCmd,
	)
}

// runStart 启动命令处理
func runStart(cmd *cobra.Command, args []string) {
	script := args[0]

	// 检查是否是配置文件
	if strings.HasSuffix(script, ".json") || strings.HasSuffix(script, ".yml") || strings.HasSuffix(script, ".yaml") {
		config, err := LoadConfig(script)
		if err != nil {
			fmt.Printf("错误: %v\n", err)
			os.Exit(1)
		}

		for _, appConfig := range config.Apps {
			process, err := pm.StartProcess(appConfig)
			if err != nil {
				fmt.Printf("启动 '%s' 失败: %v\n", appConfig.Name, err)
			} else {
				fmt.Printf("✓ 启动 '%s' (ID: %d)\n", process.Name, process.ID)
			}
		}
		return
	}

	// 解析参数
	name, _ := cmd.Flags().GetString("name")
	if name == "" {
		// 从脚本路径提取名称
		parts := strings.Split(script, "/")
		name = strings.TrimSuffix(parts[len(parts)-1], filepath.Ext(parts[len(parts)-1]))
	}

	args_list, _ := cmd.Flags().GetStringArray("args")
	cwd, _ := cmd.Flags().GetString("cwd")
	env, _ := cmd.Flags().GetStringToString("env")
	instances, _ := cmd.Flags().GetInt("instances")
	execMode, _ := cmd.Flags().GetString("exec-mode")
	watch, _ := cmd.Flags().GetBool("watch")
	ignore, _ := cmd.Flags().GetStringArray("ignore")
	logFile, _ := cmd.Flags().GetString("log")
	errorFile, _ := cmd.Flags().GetString("error")
	maxRestarts, _ := cmd.Flags().GetInt("max-restarts")
	minUptime, _ := cmd.Flags().GetString("min-uptime")

	config := AppConfig{
		Name:        name,
		Script:      script,
		Args:        args_list,
		Cwd:         cwd,
		Env:         env,
		Instances:   instances,
		ExecMode:    execMode,
		Watch:       watch,
		WatchIgnore: ignore,
		LogFile:     logFile,
		ErrorFile:   errorFile,
		MaxRestarts: maxRestarts,
		MinUptime:   minUptime,
	}

	process, err := pm.StartProcess(config)
	if err != nil {
		fmt.Printf("错误: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ 启动 '%s' (ID: %d)\n", process.Name, process.ID)
}

// runStop 停止命令处理
func runStop(cmd *cobra.Command, args []string) {
	nameOrID := args[0]
	err := pm.StopProcess(nameOrID)
	if err != nil {
		fmt.Printf("错误: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ 停止 '%s'\n", nameOrID)
}

// runRestart 重启命令处理
func runRestart(cmd *cobra.Command, args []string) {
	nameOrID := args[0]
	err := pm.RestartProcess(nameOrID)
	if err != nil {
		fmt.Printf("错误: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ 重启 '%s'\n", nameOrID)
}

// runDelete 删除命令处理
func runDelete(cmd *cobra.Command, args []string) {
	nameOrID := args[0]
	err := pm.DeleteProcess(nameOrID)
	if err != nil {
		fmt.Printf("错误: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ 删除 '%s'\n", nameOrID)
}

// runList 列表命令处理
func runList(cmd *cobra.Command, args []string) {
	processes := pm.GetProcessList()

	if len(processes) == 0 {
		fmt.Println("没有运行的进程")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\t名称\t状态\tPID\tCPU\t内存\t运行时间\t重启次数")
	fmt.Fprintln(w, "--\t----\t----\t---\t---\t----\t--------\t--------")

	for _, p := range processes {
		uptime := formatDuration(p.Uptime)
		memory := formatBytes(p.MemoryUsage)
		cpu := fmt.Sprintf("%.1f%%", p.CPUUsage)

		fmt.Fprintf(w, "%d\t%s\t%s\t%d\t%s\t%s\t%s\t%d\n",
			p.ID, p.Name, p.Status, p.PID, cpu, memory, uptime, p.Restarts)
	}

	w.Flush()
}

// runLogs 日志命令处理
func runLogs(cmd *cobra.Command, args []string) {
	nameOrID := args[0]
	lines, _ := cmd.Flags().GetInt("lines")
	follow, _ := cmd.Flags().GetBool("follow")
	showError, _ := cmd.Flags().GetBool("error")

	var err error
	if showError {
		err = pm.GetErrorLogs(nameOrID, lines, follow)
	} else {
		err = pm.GetLogs(nameOrID, lines, follow)
	}

	if err != nil {
		fmt.Printf("错误: %v\n", err)
		os.Exit(1)
	}
}

// runDescribe 详情命令处理
func runDescribe(cmd *cobra.Command, args []string) {
	nameOrID := args[0]
	process := pm.findProcess(nameOrID)
	if process == nil {
		fmt.Printf("未找到进程: %s\n", nameOrID)
		os.Exit(1)
	}

	// 更新统计信息
	pm.updateProcessStats(process)

	fmt.Printf("进程详情:\n")
	fmt.Printf("  ID: %d\n", process.ID)
	fmt.Printf("  名称: %s\n", process.Name)
	fmt.Printf("  脚本: %s\n", process.Script)
	fmt.Printf("  参数: %v\n", process.Args)
	fmt.Printf("  工作目录: %s\n", process.Cwd)
	fmt.Printf("  状态: %s\n", process.Status)
	fmt.Printf("  PID: %d\n", process.PID)
	fmt.Printf("  CPU 使用率: %.1f%%\n", process.CPUUsage)
	fmt.Printf("  内存使用: %s\n", formatBytes(process.MemoryUsage))
	fmt.Printf("  运行时间: %s\n", formatDuration(process.Uptime))
	fmt.Printf("  重启次数: %d\n", process.Restarts)
	fmt.Printf("  最大重启次数: %d\n", process.MaxRestarts)
	fmt.Printf("  启动时间: %s\n", process.StartTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("  执行模式: %s\n", process.ExecMode)
	fmt.Printf("  文件监控: %t\n", process.Watch)
	fmt.Printf("  日志文件: %s\n", process.LogFile)
	fmt.Printf("  错误日志: %s\n", process.ErrorFile)

	if len(process.Env) > 0 {
		fmt.Printf("  环境变量:\n")
		for k, v := range process.Env {
			fmt.Printf("    %s=%s\n", k, v)
		}
	}
}

// runMonit 监控命令处理
func runMonit(cmd *cobra.Command, args []string) {
	fmt.Println("实时监控模式 (按 Ctrl+C 退出)")
	fmt.Println()

	for {
		// 清屏
		fmt.Print("\033[H\033[2J")

		// 显示时间
		fmt.Printf("更新时间: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))

		// 显示进程列表
		runList(cmd, args)

		// 等待5秒
		time.Sleep(5 * time.Second)
	}
}

// runFlush 清空日志命令处理
func runFlush(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		// 清空所有日志
		processes := pm.GetProcessList()
		for _, p := range processes {
			err := pm.ClearLogs(p.Name)
			if err != nil {
				fmt.Printf("清空 '%s' 日志失败: %v\n", p.Name, err)
			}
		}
		fmt.Println("✓ 清空所有日志")
	} else {
		// 清空指定进程日志
		nameOrID := args[0]
		err := pm.ClearLogs(nameOrID)
		if err != nil {
			fmt.Printf("错误: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✓ 清空 '%s' 日志\n", nameOrID)
	}
}

// 其他命令处理函数...
func runConfigGenerate(cmd *cobra.Command, args []string) {
	configFile := "ecosystem.config.json"
	if len(args) > 0 {
		configFile = args[0]
	}

	err := GenerateConfigTemplate(configFile)
	if err != nil {
		fmt.Printf("错误: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ 生成配置文件模板: %s\n", configFile)
}

func runConfigExport(cmd *cobra.Command, args []string) {
	configFile := "ecosystem.config.json"
	if len(args) > 0 {
		configFile = args[0]
	}

	err := pm.ExportConfig(configFile)
	if err != nil {
		fmt.Printf("错误: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ 导出配置到: %s\n", configFile)
}

func runStartup(cmd *cobra.Command, args []string) {
	err := generateStartupScript()
	if err != nil {
		fmt.Printf("错误: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ 系统启动脚本已生成")
}

// generateStartupScript 生成系统启动脚本
func generateStartupScript() error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("获取可执行文件路径失败: %v", err)
	}

	switch runtime.GOOS {
	case "linux":
		return generateSystemdService(execPath)
	case "darwin":
		return generateLaunchdPlist(execPath)
	case "windows":
		return generateWindowsService(execPath)
	default:
		return fmt.Errorf("不支持的操作系统: %s", runtime.GOOS)
	}
}

// generateSystemdService 生成Linux systemd服务文件
func generateSystemdService(execPath string) error {
	serviceContent := fmt.Sprintf(`[Unit]
Description=GoPM2 Process Manager
After=network.target

[Service]
Type=simple
User=%s
ExecStart=%s resurrect
ExecReload=/bin/kill -USR2 $MAINPID
KillMode=mixed
Restart=always
RestartSec=5
WorkingDirectory=%s

[Install]
WantedBy=multi-user.target
`, os.Getenv("USER"), execPath, os.Getenv("HOME"))

	servicePath := "/etc/systemd/system/gopm2.service"

	fmt.Printf("请以root权限运行以下命令来安装服务:\n")
	fmt.Printf("sudo tee %s > /dev/null << 'EOF'\n%sEOF\n", servicePath, serviceContent)
	fmt.Printf("sudo systemctl daemon-reload\n")
	fmt.Printf("sudo systemctl enable gopm2\n")
	fmt.Printf("sudo systemctl start gopm2\n")

	return nil
}

// generateLaunchdPlist 生成macOS launchd配置文件
func generateLaunchdPlist(execPath string) error {
	homeDir := os.Getenv("HOME")
	plistContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.gopm2.agent</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
        <string>resurrect</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>WorkingDirectory</key>
    <string>%s</string>
</dict>
</plist>
`, execPath, homeDir)

	plistPath := filepath.Join(homeDir, "Library/LaunchAgents/com.gopm2.agent.plist")

	fmt.Printf("请运行以下命令来安装服务:\n")
	fmt.Printf("mkdir -p %s\n", filepath.Dir(plistPath))
	fmt.Printf("cat > %s << 'EOF'\n%sEOF\n", plistPath, plistContent)
	fmt.Printf("launchctl load %s\n", plistPath)

	return nil
}

// generateWindowsService 生成Windows服务脚本
func generateWindowsService(execPath string) error {
	// 提供两种选择：相对路径和完整路径
	relativePath := "gopm2.exe"

	fmt.Printf("请以管理员权限在PowerShell或CMD中运行以下命令:\n\n")
	fmt.Printf("# 方式1: 使用相对路径 (推荐，需要gopm2.exe在PATH中)\n")
	fmt.Printf("sc create \"GoPM2\" binPath= \"%s resurrect\" start= auto\n", relativePath)
	fmt.Printf("sc description \"GoPM2\" \"GoPM2 Process Manager - 进程管理器\"\n\n")

	fmt.Printf("# 方式2: 使用完整路径 (如果相对路径不工作)\n")
	fmt.Printf("sc create \"GoPM2\" binPath= \"%s resurrect\" start= auto\n", execPath)
	fmt.Printf("sc description \"GoPM2\" \"GoPM2 Process Manager - 进程管理器\"\n\n")

	fmt.Printf("# 启动服务\n")
	fmt.Printf("sc start \"GoPM2\"\n\n")

	fmt.Printf("# 服务管理命令:\n")
	fmt.Printf("# 启动服务: sc start \"GoPM2\"\n")
	fmt.Printf("# 停止服务: sc stop \"GoPM2\"\n")
	fmt.Printf("# 删除服务: sc delete \"GoPM2\"\n")

	return nil
}

func runSave(cmd *cobra.Command, args []string) {
	pm.saveProcesses()
	fmt.Println("✓ 保存进程列表")
}

func runResurrect(cmd *cobra.Command, args []string) {
	pm.loadProcesses()
	fmt.Println("✓ 恢复进程列表")
}

func runWatchEnable(cmd *cobra.Command, args []string) {
	nameOrID := args[0]
	err := pm.EnableWatch(nameOrID)
	if err != nil {
		fmt.Printf("错误: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ 启用 '%s' 文件监控\n", nameOrID)
}

func runWatchDisable(cmd *cobra.Command, args []string) {
	nameOrID := args[0]
	err := pm.DisableWatch(nameOrID)
	if err != nil {
		fmt.Printf("错误: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ 禁用 '%s' 文件监控\n", nameOrID)
}

// 辅助函数
func formatDuration(d time.Duration) string {
	if d == 0 {
		return "0s"
	}

	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd%dh%dm", days, hours, minutes)
	} else if hours > 0 {
		return fmt.Sprintf("%dh%dm%ds", hours, minutes, seconds)
	} else if minutes > 0 {
		return fmt.Sprintf("%dm%ds", minutes, seconds)
	} else {
		return fmt.Sprintf("%ds", seconds)
	}
}

func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
