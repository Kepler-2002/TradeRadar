#!/bin/bash

echo "停止LLM服务..."

# 查找并终止LLM服务进程
LLM_PID=$(pgrep -f "bin/llm")
if [ -n "$LLM_PID" ]; then
  echo "终止LLM服务进程，PID: $LLM_PID"
  kill $LLM_PID
  
  # 等待进程终止
  sleep 2
  if ps -p $LLM_PID > /dev/null; then
    echo "进程未能正常终止，尝试强制终止..."
    kill -9 $LLM_PID
  fi
  
  echo "LLM服务已停止"
else
  echo "未找到运行中的LLM服务"
fi