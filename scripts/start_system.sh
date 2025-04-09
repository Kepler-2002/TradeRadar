#!/bin/bash

# 设置环境变量
export CONFIG_PATH="configs/dev/app.yaml"
export STOCK_CODES="000001.SZ,600000.SH,601318.SH"

# 确保目录存在
mkdir -p bin logs configs/dev

# 如果配置文件不存在，创建一个基本配置
if [ ! -f "$CONFIG_PATH" ]; then
  echo "创建默认配置文件..."
  cat > "$CONFIG_PATH" << EOF
app:
  name: TradeRadar
  env: development

data_sources:
  tushare:
    api_key: "your_api_key_here"
    base_url: "https://api.tushare.pro"
    timeout: 10s

database:
  timescaledb:
    host: localhost
    port: 5432
    user: postgres
    password: postgres
    dbname: traderadar
    sslmode: disable

nats:
  url: "nats://localhost:4222"
  cluster_id: "test-cluster"
  client_id: "traderadar"

api:
  port: "8080"
  read_timeout: 5s
  write_timeout: 10s
EOF
fi

# 使用本地模块
export GO111MODULE=on
export GOFLAGS=-mod=mod

# 编译所有服务
echo "编译服务..."

# 添加错误处理
compile_service() {
  service=$1
  echo "编译 $service 服务..."
  go build -o bin/$service cmd/$service/main.go
  if [ $? -ne 0 ]; then
    echo "编译 $service 服务失败！"
    exit 1
  fi
  echo "$service 服务编译成功"
}

# 逐个编译服务
compile_service "api"
compile_service "collector"
compile_service "engine"
compile_service "monitor"

# 检查二进制文件是否存在
for service in api collector engine monitor; do
  if [ ! -f "bin/$service" ]; then
    echo "错误: bin/$service 不存在，编译可能失败"
    exit 1
  fi
done

# 启动服务
echo "启动服务..."
nohup bin/api > logs/api.log 2>&1 &
echo "API服务已启动，PID: $!"

nohup bin/collector > logs/collector.log 2>&1 &
echo "数据采集服务已启动，PID: $!"

nohup bin/engine > logs/engine.log 2>&1 &
echo "异动检测引擎已启动，PID: $!"

nohup bin/monitor > logs/monitor.log 2>&1 &
echo "监控服务已启动，PID: $!"

echo "所有服务已启动"
echo "可以通过 'tail -f logs/*.log' 查看日志"