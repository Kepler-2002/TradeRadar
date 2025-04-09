#!/bin/bash

echo "编译验证程序..."
go build -o bin/verify cmd/verify/main.go

if [ $? -ne 0 ]; then
  echo "编译验证程序失败！"
  exit 1
fi

echo "运行验证程序..."
./bin/verify

echo "验证完成"