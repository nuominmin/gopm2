# GoPM2 - Go语言实现的进程管理器

GoPM2 是一个用Go语言实现的生产级进程管理器，提供类似PM2的功能，支持管理任何类型的进程（不仅限于Node.js）

## 🚀 特性

### ✅ 核心功能
- **进程管理**: 启动、停止、重启、删除进程
- **守护进程**: 自动检测崩溃并重启
- **配置文件**: 支持JSON/YAML格式的配置文件
- **日志管理**: 自动日志记录、查看和清理
- **文件监控**: 自动检测文件变更并重启进程
- **多语言支持**: Node.js、Python、Go等多种脚本语言
- **状态监控**: 实时查看CPU、内存使用情况

### 🔄 进程管理功能
| 功能 | 描述 |
|------|------|
| start | 启动应用，支持配置文件批量启动 |
| stop | 停止指定应用 |
| restart | 重启应用 |
| delete | 删除进程记录 |
| list | 查看所有运行中的进程状态 |
| describe | 查看某一进程的详细信息 |
| logs | 实时查看日志（支持跟踪模式） |

### 🛠 高级功能
- **自动重启**: 可配置最大重启次数和最小运行时间
- **环境变量**: 支持自定义环境变量
- **工作目录**: 可指定进程运行目录
- **集群模式**: 支持多实例运行
- **持久化**: 进程信息自动保存和恢复


## 🎯 快速开始

### 1. 启动单个应用
```bash
# 启动Node.js应用
./gopm2 start app.js --name "my-app" --instances 2

# 启动Python应用
./gopm2 start script.py --name "python-app" --watch

# 启动Go应用
./gopm2 start main.go --name "go-app" --env "PORT=8080"
```

### 2. 使用配置文件
```bash
# 生成配置文件模板
./gopm2 config generate ecosystem.config.json

# 从配置文件启动所有应用
./gopm2 start examples/ecosystem.config.json
```

### 3. 管理进程
```bash
# 查看所有进程
./gopm2 list

# 查看进程详情
./gopm2 describe my-app

# 查看日志
./gopm2 logs my-app --follow

# 重启进程
./gopm2 restart my-app

# 停止进程
./gopm2 stop my-app

# 删除进程
./gopm2 delete my-app
```

### 4. 文件监控
```bash
# 启用文件监控（自动重启）
./gopm2 watch enable my-app

# 禁用文件监控
./gopm2 watch disable my-app
```

## 📋 命令参考

### 启动命令选项
```bash
./gopm2 start <script> [选项]

选项:
  -n, --name string          应用名称
  -a, --args stringArray     传递给脚本的参数
  -c, --cwd string           工作目录
  -e, --env stringToString   环境变量 (key=value)
  -i, --instances int        实例数量 (默认: 1)
  -x, --exec-mode string     执行模式 (fork|cluster) (默认: "fork")
  -w, --watch                启用文件监控
      --ignore stringArray   监控时忽略的文件模式
  -l, --log string           日志文件路径
      --error string         错误日志文件路径
      --max-restarts int     最大重启次数 (默认: 15)
      --min-uptime string    最小运行时间 (默认: "1s")
```

### 日志命令选项
```bash
./gopm2 logs <name|id> [选项]

选项:
  -n, --lines int   显示的行数 (默认: 50)
  -f, --follow      实时跟踪日志
  -e, --error       显示错误日志
```

## 📝 配置文件格式

### JSON格式 (ecosystem.config.json)
```json
{
  "apps": [
    {
      "name": "web-server",
      "script": "./server.js",
      "args": ["--port", "3000"],
      "cwd": "/path/to/app",
      "env": {
        "NODE_ENV": "production",
        "PORT": "3000"
      },
      "instances": 2,
      "exec_mode": "cluster",
      "watch": true,
      "watch_ignore": ["node_modules", "logs"],
      "log_file": "./logs/app.log",
      "error_file": "./logs/app-error.log",
      "max_restarts": 10,
      "min_uptime": "10s"
    }
  ]
}
```

### YAML格式 (ecosystem.config.yml)
```yaml
apps:
  - name: web-server
    script: ./server.js
    args: ["--port", "3000"]
    cwd: /path/to/app
    env:
      NODE_ENV: production
      PORT: "3000"
    instances: 2
    exec_mode: cluster
    watch: true
    watch_ignore: ["node_modules", "logs"]
    log_file: ./logs/app.log
    error_file: ./logs/app-error.log
    max_restarts: 10
    min_uptime: 10s
```

## 🔧 配置字段说明

| 字段 | 类型 | 描述 | 默认值 |
|------|------|------|---------|
| name | string | 应用名称（必需） | - |
| script | string | 脚本路径（必需） | - |
| args | array | 命令行参数 | [] |
| cwd | string | 工作目录 | 当前目录 |
| env | object | 环境变量 | {} |
| instances | number | 实例数量 | 1 |
| exec_mode | string | 执行模式 (fork/cluster) | fork |
| watch | boolean | 启用文件监控 | false |
| watch_ignore | array | 监控忽略模式 | [] |
| log_file | string | 日志文件路径 | 自动生成 |
| error_file | string | 错误日志路径 | 自动生成 |
| max_restarts | number | 最大重启次数 | 15 |
| min_uptime | string | 最小运行时间 | "1s" |

## 📁 项目结构

```
gopm2/
├── main.go              # 主入口文件
├── types.go             # 数据类型定义  
├── manager.go           # 进程管理核心
├── config.go            # 配置文件处理
├── watcher.go           # 文件监控功能
├── logs.go              # 日志管理
├── cli.go               # 命令行界面
├── go.mod               # Go模块依赖
├── go.sum               # 依赖校验文件
├── .gitignore           # Git忽略文件配置
├── Makefile             # 构建脚本
├── README.md            # 项目文档
├── USAGE.md             # 快速使用指南
├── gopm2.exe            # 编译后的可执行文件
└── examples/            # 测试示例目录
    ├── README.md        # 示例说明文档
    ├── test-app.js      # Node.js测试应用
    ├── test-app.go      # Go测试应用
    ├── test-server.py   # Python测试应用
    ├── ecosystem.config.json # 完整配置示例
    └── test-config.json # 基础配置模板
```

### 数据文件结构

```
~/.gopm2/
├── logs/              # 日志文件目录
│   ├── app.log
│   └── app-error.log
├── pids/              # PID文件目录
│   └── app.pid
└── processes.json     # 进程信息持久化文件
```

## 🎨 监控界面

```bash
# 实时监控所有进程
./gopm2 monit
```

显示格式：
```
ID  名称       状态    PID    CPU   内存    运行时间  重启次数
--  ----       ----    ---    ---   ----    --------  --------
1   web-app    online  1234   2.5%  45.2MB  2h30m     0
2   api-app    online  1235   1.2%  32.1MB  1h45m     1
```

## 🚀 性能特点

- **轻量级**: 单个二进制文件，无需依赖
- **跨平台**: 支持Windows、Linux、macOS
- **高性能**: Go语言实现，低内存占用
- **并发安全**: 多进程并发管理
- **可靠性**: 进程守护和自动恢复

## 📖 使用示例

### 启动Node.js集群
```bash
./gopm2 start server.js --name "web" --instances 4 --exec-mode cluster --watch
```

### 启动Python后台服务
```bash
./gopm2 start worker.py --name "worker" --env "DEBUG=false" --max-restarts 5
```

### 批量管理
```bash
# 启动所有服务
./gopm2 start examples/ecosystem.config.json

# 停止所有服务
./gopm2 stop all

# 重启所有服务
./gopm2 restart all
```

## 📦 开发和构建

### 构建
```bash
# 安装依赖
go mod tidy

# 构建项目
go build -o gopm2

# 在Windows上构建
go build -o gopm2.exe

# 跨平台构建
GOOS=linux GOARCH=amd64 go build -o gopm2-linux
GOOS=darwin GOARCH=amd64 go build -o gopm2-darwin
GOOS=windows GOARCH=amd64 go build -o gopm2.exe
```

## 🆚 与PM2对比

| 功能 | GoPM2 | PM2 |
|------|-------|-----|
| 语言 | Go | Node.js |
| 安装 | 单二进制文件 | npm install |
| 内存占用 | 低 | 中等 |
| 启动速度 | 快 | 快 |
| 跨平台 | ✅ | ✅ |
| 配置文件 | JSON/YAML | JS/JSON/YAML |
| 进程管理 | ✅ | ✅ |
| 文件监控 | ✅ | ✅ |
| 集群模式 | ✅ | ✅ |
| 云端管理 | ❌ | ✅ |

GoPM2专注于本地进程管理，提供PM2的核心功能，同时具有更好的性能和更简单的部署方式。 