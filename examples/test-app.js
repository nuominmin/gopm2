#!/usr/bin/env node

const http = require('http');
const port = process.env.PORT || 3000;

console.log(`启动时间: ${new Date().toISOString()}`);
console.log(`进程ID: ${process.pid}`);
console.log(`端口: ${port}`);

// 简单的HTTP服务器
const server = http.createServer((req, res) => {
  const timestamp = new Date().toISOString();
  const response = {
    timestamp,
    pid: process.pid,
    message: 'Hello from GoPM2 test app!',
    url: req.url,
    method: req.method
  };

  console.log(`[${timestamp}] ${req.method} ${req.url} - PID: ${process.pid}`);

  res.writeHead(200, { 'Content-Type': 'application/json' });
  res.end(JSON.stringify(response, null, 2));
});

server.listen(port, () => {
  console.log(`✓ 服务器启动成功，监听端口 ${port}`);
});

// 优雅关闭
process.on('SIGTERM', () => {
  console.log('收到SIGTERM信号，正在关闭服务器...');
  server.close(() => {
    console.log('服务器已关闭');
    process.exit(0);
  });
});

process.on('SIGINT', () => {
  console.log('收到SIGINT信号，正在关闭服务器...');
  server.close(() => {
    console.log('服务器已关闭');
    process.exit(0);
  });
});

// 定期输出心跳日志
setInterval(() => {
  console.log(`[心跳] ${new Date().toISOString()} - PID: ${process.pid} - 内存使用: ${Math.round(process.memoryUsage().rss / 1024 / 1024)}MB`);
}, 30000);

// 模拟随机错误
setInterval(() => {
  if (Math.random() < 0.001) { // 0.1% 概率
    console.error(`[错误] 模拟随机错误 - PID: ${process.pid}`);
  }
}, 10000); 