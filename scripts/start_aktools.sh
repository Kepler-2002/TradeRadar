#!/bin/bash

echo "启动AKTools服务..."

# 检查是否已安装aktools
if ! pip show aktools > /dev/null 2>&1; then
  echo "安装AKTools..."
  pip install aktools
fi

# 卸载uvloop以避免与nest_asyncio的冲突
echo "卸载uvloop..."
pip uninstall -y uvloop

# 停止已有的AKTools进程
AKTOOLS_PID=$(ps aux | grep "python -m aktools" | grep -v grep | awk '{print $2}')
if [ -n "$AKTOOLS_PID" ]; then
  echo "停止已有的AKTools进程，PID: $AKTOOLS_PID"
  kill $AKTOOLS_PID
  sleep 2
fi

# 检查8081端口是否被占用
PORT_USED=$(lsof -i:8081 | grep LISTEN | wc -l)
if [ "$PORT_USED" -gt "0" ]; then
  echo "8081端口被占用，尝试释放..."
  lsof -i:8081 | grep LISTEN | awk '{print $2}' | xargs kill -9
  sleep 2
fi

# 使用不同的端口启动AKTools
echo "启动AKTools服务在8081端口..."
mkdir -p logs
nohup python -m aktools --port 8081 > logs/aktools.log 2>&1 &
AKTOOLS_NEW_PID=$!
echo "AKTools服务已启动，PID: $AKTOOLS_NEW_PID"

# 等待服务启动
echo "等待服务启动..."
for i in {1..30}; do  # 增加等待时间
  if curl -s --connect-timeout 5 http://127.0.0.1:8081/api/public/stock_zh_index_spot_em > /dev/null; then
    echo "AKTools服务已成功启动"
    echo "服务地址: http://127.0.0.1:8081"
    break
  fi
  
  if [ $i -eq 30 ]; then
    echo "AKTools服务启动超时，请检查日志"
    cat logs/aktools.log
    exit 1
  fi
  
  echo "等待中... $i/30"
  sleep 2
done

# 测试A股接口
echo "测试A股接口..."
if ! curl -s --connect-timeout 10 http://127.0.0.1:8081/api/public/stock_zh_a_spot_em > /dev/null; then
  echo "A股接口测试失败，请检查日志"
  cat logs/aktools.log
  exit 1
fi
echo "A股接口测试成功"