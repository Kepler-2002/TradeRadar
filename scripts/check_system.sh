#!/bin/bash

echo "检查系统服务状态..."

# 检查主要服务
echo "主要服务状态:"
echo "API服务: $(pgrep -f "bin/api" > /dev/null && echo "运行中" || echo "未运行")"
echo "采集服务: $(pgrep -f "bin/collector" > /dev/null && echo "运行中" || echo "未运行")"
echo "引擎服务: $(pgrep -f "bin/engine" > /dev/null && echo "运行中" || echo "未运行")"
echo "监控服务: $(pgrep -f "bin/monitor" > /dev/null && echo "运行中" || echo "未运行")"
echo "LLM服务: $(pgrep -f "bin/llm" > /dev/null && echo "运行中" || echo "未运行")"

# 检查AKTools服务
echo -n "AKTools服务: "
if pgrep -f "python -m aktools" > /dev/null; then
  echo "运行中"
else
  echo "未运行"
fi

# 检查NATS服务
echo -n "NATS服务: "
if docker ps | grep nats-streaming > /dev/null; then
  echo "运行中"
else
  echo "未运行"
fi

# 检查日志文件
echo -e "\n日志文件状态:"
for log_file in logs/*.log; do
  if [ -f "$log_file" ]; then
    size=$(du -h "$log_file" | cut -f1)
    last_modified=$(stat -f "%Sm" -t "%Y-%m-%d %H:%M:%S" "$log_file")
    echo "$log_file - 大小: $size, 最后修改: $last_modified"
  fi
done

echo -e "\n系统状态检查完成"