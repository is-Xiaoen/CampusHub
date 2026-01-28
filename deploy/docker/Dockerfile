# ============================================================================
# CampusHub 多阶段构建 Dockerfile
# ============================================================================
# 使用方法：
#   docker build --build-arg SERVICE=gateway -t campushub-gateway .
#   docker build --build-arg SERVICE=user -t campushub-user .
#   docker build --build-arg SERVICE=activity -t campushub-activity .
#   docker build --build-arg SERVICE=chat -t campushub-chat .
# ============================================================================

# ==================== 构建阶段 ====================
FROM golang:1.21-alpine AS builder

# 设置 Go 环境
ENV GO111MODULE=on \
    GOPROXY=https://goproxy.cn,direct \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

# 安装基本工具
RUN apk add --no-cache git

WORKDIR /build

# 复制依赖文件并下载
COPY go.mod go.sum ./
RUN go mod download

# 复制源码
COPY . .

# 构建参数：指定要构建的服务
ARG SERVICE=gateway

# 根据 SERVICE 参数构建不同的服务
RUN if [ "$SERVICE" = "gateway" ]; then \
        go build -ldflags="-s -w" -o /app/server ./app/gateway/api/gateway.go; \
    elif [ "$SERVICE" = "user" ]; then \
        go build -ldflags="-s -w" -o /app/server ./app/user/rpc/user.go; \
    elif [ "$SERVICE" = "activity" ]; then \
        go build -ldflags="-s -w" -o /app/server ./app/activity/rpc/activity.go; \
    elif [ "$SERVICE" = "chat" ]; then \
        go build -ldflags="-s -w" -o /app/server ./app/chat/rpc/chat.go; \
    elif [ "$SERVICE" = "demo" ]; then \
        go build -ldflags="-s -w" -o /app/server ./app/demo/rpc/demo.go; \
    fi

# ==================== 运行阶段 ====================
FROM alpine:3.18

# 安装时区数据和证书
RUN apk add --no-cache tzdata ca-certificates && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone

WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/server .

# 创建配置目录
RUN mkdir -p /app/etc

# 暴露端口（根据服务不同而不同）
# Gateway: 8080, User: 9001, Activity: 9002, Chat: 9003, Demo: 9100
EXPOSE 8080 9001 9002 9003 9100

# 启动命令
ENTRYPOINT ["./server"]
CMD ["-f", "/app/etc/config.yaml"]
