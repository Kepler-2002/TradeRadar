#!/bin/bash

# 检查NATS Streaming容器是否已存在
CONTAINER_EXISTS=$(docker ps -a | grep nats-streaming | wc -l)

if [ "$CONTAINER_EXISTS" -gt "0" ]; then
  echo "NATS Streaming容器已存在，正在启动..."
  docker start nats-streaming
else
  echo "创建并启动NATS Streaming容器..."
  docker run -d --name nats-streaming -p 4222:4222 -p 8222:8222 nats-streaming -m 8222 -cid test-cluster
fi

# 等待服务启动
sleep 2

# 检查服务状态
if [ "$(docker ps | grep nats-streaming | wc -l)" -gt "0" ]; then
  echo "NATS Streaming服务已成功启动"
  echo "监听端口: 4222 (NATS), 8222 (HTTP监控)"
  echo "集群ID: test-cluster"
  echo "监控界面: http://localhost:8222/streaming"
else
  echo "NATS Streaming服务启动失败"
  exit 1
fi