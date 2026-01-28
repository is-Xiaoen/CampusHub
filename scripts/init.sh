#!/bin/bash
# 项目初始化脚本

set -e

echo "=== 校园活动平台 初始化脚本 ==="

# 1. 检查 Go 环境
echo "检查 Go 环境..."
go version || { echo "请先安装 Go 1.21+"; exit 1; }

# 2. 下载依赖
echo "下载依赖..."
go mod tidy

# 3. 检查 Docker
echo "检查 Docker 环境..."
docker --version || { echo "请先安装 Docker"; exit 1; }

# 4. 启动基础设施
echo "启动 MySQL 和 Redis..."
docker-compose -f deploy/docker/docker-compose.yaml up -d

# 5. 等待服务就绪
echo "等待服务就绪..."
sleep 10

# 6. 初始化数据库
echo "初始化数据库..."
docker exec -i activity-mysql mysql -uroot -proot123456 < deploy/sql/user.sql
docker exec -i activity-mysql mysql -uroot -proot123456 < deploy/sql/activity.sql
docker exec -i activity-mysql mysql -uroot -proot123456 < deploy/sql/chat.sql

echo "=== 初始化完成 ==="
echo "运行 'make run-gateway' 启动网关服务"
