#!/bin/bash

echo "停止爬虫发布服务..."

# 查找并终止爬虫发布服务进程
CRAWLER_PID=$(pgrep -f "python3 scripts/crawler_integration.py")
if [ -n "$CRAWLER_PID" ]; then
    echo "终止爬虫发布服务进程，PID: $CRAWLER_PID"
    kill $CRAWLER_PID
    
    # 等待进程终止
    sleep 2
    if ps -p $CRAWLER_PID > /dev/null; then
        echo "进程未能正常终止，尝试强制终止..."
        kill -9 $CRAWLER_PID
    fi
    
    echo "爬虫发布服务已停止"
else
    echo "未找到运行中的爬虫发布服务"
fi