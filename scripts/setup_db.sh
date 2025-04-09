#!/bin/bash

# 使用Docker容器ID
CONTAINER_ID="6e14b90e9ca3"

# 检查数据库是否存在
echo "检查数据库是否存在..."
DB_EXISTS=$(docker exec $CONTAINER_ID psql -U postgres -tAc "SELECT 1 FROM pg_database WHERE datname='traderadar'")

if [ -z "$DB_EXISTS" ]; then
  echo "创建数据库 traderadar..."
  docker exec $CONTAINER_ID psql -U postgres -c "CREATE DATABASE traderadar"
else
  echo "数据库 traderadar 已存在"
fi

# 复制SQL脚本到容器
echo "复制初始化脚本到容器..."
docker cp /Users/dewei/GolandProjects/TradeRadar/scripts/init_db.sql $CONTAINER_ID:/tmp/init_db.sql

# 执行初始化脚本
echo "执行初始化脚本..."
docker exec $CONTAINER_ID psql -U postgres -d traderadar -f /tmp/init_db.sql

echo "数据库设置完成"