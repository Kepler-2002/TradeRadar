#!/bin/bash

echo "停止所有服务..."

# 停止主要服务
echo "停止主要服务进程..."
pkill -f bin/api
pkill -f bin/collector
pkill -f bin/engine
pkill -f bin/monitor

# 检查服务是否已停止
sleep 2
if pgrep -f "bin/(api|collector|engine|monitor)" > /dev/null; then
  echo "部分服务未能正常停止，尝试强制终止..."
  pkill -9 -f bin/api
  pkill -9 -f bin/collector
  pkill -9 -f bin/engine
  pkill -9 -f bin/monitor
fi

# 停止AKTools服务
echo "停止AKTools服务..."
./scripts/stop_aktools.sh

# 停止NATS服务
echo "停止NATS服务..."
./scripts/stop_nats.sh

echo "所有服务已停止"
echo "可以通过 'ps aux | grep -E \"bin/|aktools|nats\"' 检查是否有残留进程"