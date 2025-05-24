#!/bin/bash

echo "停止NATS Server (JetStream) 服务..."

# 检查容器是否在运行
if [ "$(docker ps | grep nats-jetstream | wc -l)" -gt "0" ]; then
  docker stop nats-jetstream
  echo "NATS Server (JetStream) 服务已停止"
else
  echo "NATS Server (JetStream) 服务未在运行"
fi

# 可选：询问是否删除容器
read -p "是否删除容器? (y/n): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
  docker rm nats-jetstream
  echo "容器已删除"
fi