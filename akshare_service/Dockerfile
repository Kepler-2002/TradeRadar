FROM python:3.9-slim

WORKDIR /app

# 安装curl工具用于调试
RUN apt-get update && apt-get install -y curl

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY . .

EXPOSE 5000

# 使用Flask开发服务器
CMD ["python", "app.py"]