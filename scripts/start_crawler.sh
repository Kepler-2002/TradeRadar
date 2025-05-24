#!/bin/bash

echo "启动爬虫数据发布服务..."

# 检查Python环境
if ! command -v python3 &> /dev/null; then
    echo "错误: 未找到python3命令"
    exit 1
fi

# 检查依赖库
if ! python3 -c "import stan.aio" &> /dev/null; then
    echo "安装NATS Streaming客户端..."
    pip3 install asyncio-nats-streaming
fi

# 确保日志目录存在
mkdir -p logs

# 启动爬虫发布服务
echo "启动爬虫发布服务..."
nohup python3 scripts/crawler_integration.py > logs/crawler.log 2>&1 &
CRAWLER_PID=$!

# 检查服务是否成功启动
sleep 2
if ! ps -p $CRAWLER_PID > /dev/null; then
    echo "爬虫发布服务启动失败，请检查日志: logs/crawler.log"
    exit 1
fi

echo "爬虫发布服务已启动，PID: $CRAWLER_PID"