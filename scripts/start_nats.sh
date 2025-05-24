#!/bin/bash

# 检查NATS Server容器是否已存在
CONTAINER_EXISTS=$(docker ps -a | grep nats-jetstream | wc -l)

if [ "$CONTAINER_EXISTS" -gt "0" ]; then
  echo "NATS Server容器已存在，正在启动..."
  docker start nats-jetstream
else
  echo "创建并启动NATS Server (with JetStream)容器..."
  docker run -d --name nats-jetstream \
    -p 4222:4222 \
    -p 8222:8222 \
    nats:latest \
    --jetstream \
    --http_port 8222
fi

# 等待服务启动
sleep 2

# 检查服务状态
if [ "$(docker ps | grep nats-jetstream | wc -l)" -gt "0" ]; then
  echo "NATS Server (JetStream) 服务已成功启动"
  echo "监听端口: 4222 (NATS), 8222 (HTTP监控)"
  echo "监控界面: http://localhost:8222"
else
  echo "NATS Server服务启动失败"
  exit 1
fi