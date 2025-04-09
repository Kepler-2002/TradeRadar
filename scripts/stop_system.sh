#!/bin/bash

echo "停止所有服务..."

# 查找并终止所有服务进程
pkill -f bin/api
pkill -f bin/collector
pkill -f bin/engine
pkill -f bin/monitor

echo "所有服务已停止"