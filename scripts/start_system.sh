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

# 下载依赖
echo "下载依赖..."
go mod tidy

# 编译所有服务
echo "编译服务..."
go build -o bin/api cmd/api/main.go
go build -o bin/collector cmd/collector/main.go
go build -o bin/engine cmd/engine/main.go
go build -o bin/monitor cmd/monitor/main.go

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