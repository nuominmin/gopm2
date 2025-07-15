# GoPM2 测试示例

这个目录包含了用于测试GoPM2功能的示例应用和配置文件。

## 📁 文件说明

### 测试应用

#### `test-app.js` - Node.js 示例应用
一个简单的HTTP服务器，用于测试Node.js应用的进程管理。

**特性:**
- HTTP服务器（默认端口3000）
- 定期心跳日志输出
- 优雅关闭处理
- 模拟随机错误

**启动方式:**
```bash
# 单实例启动
../gopm2.exe start examples/test-app.js --name "node-app"

# 集群模式启动
../gopm2.exe start examples/test-app.js --name "node-cluster" --instances 4 --exec-mode cluster

# 启用文件监控
../gopm2.exe start examples/test-app.js --name "node-watch" --watch
```

#### `test-app.go` - Go 示例应用
一个Go语言编写的HTTP服务器，用于测试Go应用的进程管理。

**特性:**
- HTTP服务器（默认端口8080）
- JSON响应格式
- 信号处理
- 定期心跳日志

**启动方式:**
```bash
# 启动Go应用
../gopm2.exe start examples/test-app.go --name "go-app" --env "PORT=9090"

# 多实例启动
../gopm2.exe start examples/test-app.go --name "go-multi" --instances 3
```

#### `test-server.py` - Python 示例应用
一个Python编写的HTTP服务器，用于测试Python应用的进程管理。

**特性:**
- HTTP服务器（默认端口8080）
- JSON响应
- 信号处理
- 环境变量支持

**启动方式:**
```bash
# 启动Python应用
../gopm2.exe start examples/test-server.py --name "python-app" --env "PORT=8080"

# 启用文件监控
../gopm2.exe start examples/test-server.py --name "python-watch" --watch
```

### 配置文件

#### `ecosystem.config.json` - 完整配置示例
包含多个应用的完整配置文件，展示了所有可用的配置选项。

**包含的应用:**
- `web-server`: Node.js集群应用
- `api-server`: Python API服务
- `worker`: Go后台工作进程

**使用方式:**
```bash
# 从配置文件启动所有应用
../gopm2.exe start examples/ecosystem.config.json
```

#### `test-config.json` - 基础配置模板
由GoPM2生成的配置文件模板，包含常用配置项。

## 🚀 快速测试指南

### 1. 基础功能测试
```bash
# 进入项目根目录
cd /path/to/gopm2

# 启动Node.js应用
./gopm2.exe start examples/test-app.js --name "test"

# 查看进程列表
./gopm2.exe list

# 查看进程详情
./gopm2.exe describe test

# 查看日志
./gopm2.exe logs test

# 重启进程
./gopm2.exe restart test

# 停止进程
./gopm2.exe stop test
```

### 2. 配置文件测试
```bash
# 从配置文件启动
./gopm2.exe start examples/ecosystem.config.json

# 查看所有进程
./gopm2.exe list

# 停止所有进程
./gopm2.exe stop web-server
./gopm2.exe stop api-server
./gopm2.exe stop worker
```

### 3. 文件监控测试
```bash
# 启用文件监控
./gopm2.exe start examples/test-app.js --name "watch-test" --watch

# 修改test-app.js文件，观察自动重启
# 进程会自动检测文件变更并重启
```

### 4. 集群模式测试
```bash
# 启动多实例
./gopm2.exe start examples/test-app.js --name "cluster-test" --instances 4

# 查看多个进程实例
./gopm2.exe list
```

### 5. 日志管理测试
```bash
# 查看标准输出日志
./gopm2.exe logs test-app --lines 100

# 查看错误日志
./gopm2.exe logs test-app --error

# 实时跟踪日志
./gopm2.exe logs test-app --follow

# 清空日志
./gopm2.exe flush test-app
```

### 6. 监控功能测试
```bash
# 实时监控所有进程
./gopm2.exe monit

# 查看进程详细信息
./gopm2.exe describe test-app
```

## 🔧 配置说明

### 环境变量配置
```bash
# 设置端口
--env "PORT=3000"

# 设置多个环境变量
--env "PORT=3000" --env "NODE_ENV=production"
```

### 高级选项
```bash
# 自定义工作目录
--cwd "/path/to/app"

# 设置最大重启次数
--max-restarts 5

# 设置最小运行时间
--min-uptime "10s"

# 自定义日志文件
--log "./logs/custom.log" --error "./logs/custom-error.log"
```

## 📊 性能测试

### HTTP服务测试
```bash
# 启动服务后，可以使用curl测试
curl http://localhost:3000
curl http://localhost:8080
curl http://localhost:9090
```

### 负载测试
```bash
# 启动集群模式
./gopm2.exe start examples/test-app.js --name "load-test" --instances 4

# 使用curl或其他工具进行负载测试
for i in {1..100}; do curl http://localhost:3000 & done
```

## 🐛 故障排除

### 常见问题

1. **进程无法启动**
   - 检查脚本文件路径是否正确
   - 确认运行环境已安装（Node.js、Python等）
   - 查看错误日志：`./gopm2.exe logs <name> --error`

2. **端口冲突**
   - 使用不同的端口：`--env "PORT=8081"`
   - 检查端口是否被占用

3. **文件监控不工作**
   - 确认文件路径正确
   - 检查文件权限
   - 查看监控日志

4. **进程频繁重启**
   - 检查应用代码是否有错误
   - 调整最小运行时间：`--min-uptime "30s"`
   - 查看详细日志分析问题

## 📝 注意事项

1. 在Windows环境下，某些信号处理可能与Linux/macOS不同
2. Python和Node.js应用需要相应的运行环境
3. Go应用使用`go run`命令，需要Go环境
4. 日志文件保存在用户主目录的`.gopm2/logs/`下
5. 进程信息持久化保存在`.gopm2/processes.json`中 