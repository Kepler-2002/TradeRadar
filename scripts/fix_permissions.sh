#!/bin/bash

echo "为所有脚本添加执行权限..."

# 为scripts目录下的所有.sh文件添加执行权限
chmod +x scripts/*.sh

echo "权限已更新，以下是所有可执行脚本："
ls -la scripts/*.sh

echo "完成"