#!/bin/bash

echo "启动AKShare服务..."

# 使用AKShare官方Docker镜像
echo "拉取AKShare官方Docker镜像..."
docker pull registry.cn-shanghai.aliyuncs.com/akfamily/aktools:jupyter

# 检查容器是否已存在
CONTAINER_EXISTS=$(docker ps -a | grep akshare-service | wc -l)

if [ "$CONTAINER_EXISTS" -gt "0" ]; then
  echo "AKShare服务容器已存在，正在启动..."
  docker start akshare-service
else
  echo "创建并启动AKShare服务容器..."
  # 使用官方镜像启动容器
  docker run -d --name akshare-service -p 5000:5000 -p 5001:5001 registry.cn-shanghai.aliyuncs.com/akfamily/aktools:jupyter

fi

# 等待服务启动
echo "等待服务启动..."
sleep 10

echo "AKShare服务已启动"
echo "Jupyter界面: http://localhost:5000"
echo "API服务将在你通过Jupyter创建并运行后可用"