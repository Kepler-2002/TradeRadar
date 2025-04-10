#!/bin/bash

echo "停止AKTools服务..."

# 查找并终止AKTools进程
AKTOOLS_PID=$(ps aux | grep "python -m aktools" | grep -v grep | awk '{print $2}')
if [ -n "$AKTOOLS_PID" ]; then
  echo "终止AKTools进程，PID: $AKTOOLS_PID"
  kill -9 $AKTOOLS_PID
  sleep 1
  # 验证进程是否已终止
  if ps -p $AKTOOLS_PID > /dev/null; then
    echo "进程未能终止，尝试强制终止..."
    kill -9 $AKTOOLS_PID
  fi
  echo "AKTools服务已停止"
else
  echo "未找到运行中的AKTools服务"
fi

# 检查8081端口是否被占用
PORT_USED=$(lsof -i:8081 | grep LISTEN | wc -l)
if [ "$PORT_USED" -gt "0" ]; then
  echo "8081端口仍被占用，尝试释放..."
  lsof -i:8081 | grep LISTEN | awk '{print $2}' | xargs kill -9
  echo "端口已释放"
fi