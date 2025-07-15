#!/usr/bin/env python3

import http.server
import socketserver
import json
import os
import time
import signal
import sys
from datetime import datetime

PORT = int(os.environ.get('PORT', 8080))
PID = os.getpid()

print(f"启动时间: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")
print(f"进程ID: {PID}")
print(f"端口: {PORT}")

class MyHandler(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        timestamp = datetime.now().strftime('%Y-%m-%d %H:%M:%S')
        response = {
            "timestamp": timestamp,
            "pid": PID,
            "message": "Hello from GoPM2 Python test app!",
            "url": self.path,
            "method": "GET"
        }
        
        print(f"[{timestamp}] GET {self.path} - PID: {PID}")
        
        self.send_response(200)
        self.send_header('Content-type', 'application/json')
        self.end_headers()
        self.wfile.write(json.dumps(response, indent=2).encode())

def signal_handler(sig, frame):
    print(f'\n收到信号 {sig}，正在关闭服务器...')
    sys.exit(0)

# 注册信号处理器
signal.signal(signal.SIGINT, signal_handler)
signal.signal(signal.SIGTERM, signal_handler)

# 启动服务器
with socketserver.TCPServer(("", PORT), MyHandler) as httpd:
    print(f"✓ 服务器启动成功，监听端口 {PORT}")
    
    try:
        httpd.serve_forever()
    except KeyboardInterrupt:
        print("\n服务器已关闭")
        pass 