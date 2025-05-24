#!/bin/bash

echo "启动LLM服务..."

# 检查配置文件是否存在
CONFIG_PATH=${CONFIG_PATH:-"configs/dev/app.yaml"}
if [ ! -f "$CONFIG_PATH" ]; then
  echo "错误: 配置文件 $CONFIG_PATH 不存在"
  exit 1
fi

# 检查LLM服务是否已在运行
if pgrep -f "bin/llm" > /dev/null; then
  echo "LLM服务已在运行，跳过启动"
  exit 0
fi

# 确保bin目录存在
mkdir -p bin

# 编译LLM服务
echo "编译LLM服务..."
go build -o bin/llm cmd/llm/main.go

if [ $? -ne 0 ]; then
  echo "编译LLM服务失败"
  exit 1
fi

# 启动LLM服务
echo "启动LLM服务进程..."
nohup bin/llm > logs/llm.log 2>&1 &
LLM_PID=$!

# 检查服务是否成功启动
sleep 2
if ! ps -p $LLM_PID > /dev/null; then
  echo "LLM服务启动失败，请检查日志: logs/llm.log"
  exit 1
fi

echo "LLM服务已启动，PID: $LLM_PID"