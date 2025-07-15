# GoPM2 快速使用指南

## 🚀 快速开始

### 1. 编译项目
```bash
go build -o gopm2.exe
```

### 2. 查看帮助
```bash
./gopm2.exe --help
```

### 3. 基本命令

#### 启动应用
```bash
# 启动单个脚本
./gopm2.exe start examples/test-server.py --name "my-app"

# 启动并设置环境变量
./gopm2.exe start examples/test-app.js --name "web" --env "PORT=3000"

# 启动多实例（集群模式）
./gopm2.exe start examples/test-app.js --name "cluster" --instances 4

# 启用文件监控
./gopm2.exe start examples/test-app.js --name "watch" --watch

# 从配置文件启动
./gopm2.exe start examples/ecosystem.config.json
```

#### 管理进程
```bash
# 查看所有进程
./gopm2.exe list

# 查看进程详情
./gopm2.exe describe my-app

# 重启进程
./gopm2.exe restart my-app

# 停止进程
./gopm2.exe stop my-app

# 删除进程
./gopm2.exe delete my-app
```

#### 日志管理
```bash
# 查看日志（最后50行）
./gopm2.exe logs my-app

# 查看指定行数的日志
./gopm2.exe logs my-app --lines 100

# 实时跟踪日志
./gopm2.exe logs my-app --follow

# 查看错误日志
./gopm2.exe logs my-app --error

# 清空日志
./gopm2.exe flush my-app
```

#### 监控功能
```bash
# 实时监控所有进程
./gopm2.exe monit

# 查看进程详细信息
./gopm2.exe describe my-app
```

#### 配置管理
```bash
# 生成配置文件模板
./gopm2.exe config generate my-config.json

# 导出当前配置
./gopm2.exe config export current-config.json
```

#### 文件监控
```bash
# 启用文件监控
./gopm2.exe watch enable my-app

# 禁用文件监控
./gopm2.exe watch disable my-app
```

## 📝 配置文件示例

### 基础配置
```json
{
  "apps": [
    {
      "name": "my-app",
      "script": "./app.js",
      "instances": 1,
      "exec_mode": "fork",
      "watch": false
    }
  ]
}
```

### 完整配置
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

## 🔧 常用选项

### 启动选项
- `--name, -n`: 应用名称
- `--args, -a`: 命令行参数
- `--cwd, -c`: 工作目录
- `--env, -e`: 环境变量
- `--instances, -i`: 实例数量
- `--exec-mode, -x`: 执行模式 (fork/cluster)
- `--watch, -w`: 启用文件监控
- `--ignore`: 监控忽略模式
- `--log, -l`: 日志文件路径
- `--error`: 错误日志路径
- `--max-restarts`: 最大重启次数
- `--min-uptime`: 最小运行时间

### 日志选项
- `--lines, -n`: 显示行数
- `--follow, -f`: 实时跟踪
- `--error, -e`: 显示错误日志

## 📊 进程状态

- `online`: 正在运行
- `stopped`: 已停止
- `stopping`: 正在停止
- `errored`: 出错状态

## 🛠 故障排除

1. **进程无法启动**: 检查脚本路径和运行环境
2. **端口被占用**: 使用不同端口或检查占用进程
3. **权限问题**: 确保有文件读写权限
4. **内存不足**: 检查系统资源使用情况

## 📚 更多信息

详细文档请参考：
- [README.md](./README.md) - 完整功能说明
- [examples/README.md](./examples/README.md) - 测试示例指南 