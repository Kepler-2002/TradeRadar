#!/bin/bash

echo "编译LLM验证程序..."
go build -o bin/verify_llm cmd/verify_llm/main.go

if [ $? -ne 0 ]; then
  echo "编译验证程序失败！"
  exit 1
fi

echo "运行LLM验证程序..."
./bin/verify_llm

echo "验证完成"