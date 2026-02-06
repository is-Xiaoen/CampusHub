# ============================================================================
# CampusHub 多阶段构建 Dockerfile
# ============================================================================
#
# 使用方法：
#   # 构建 API 服务
#   docker build --build-arg SERVICE=user --build-arg TYPE=api -t campushub/user-api .
#   docker build --build-arg SERVICE=activity --build-arg TYPE=api -t campushub/activity-api .
#   docker build --build-arg SERVICE=chat --build-arg TYPE=api -t campushub/chat-api .
#
#   # 构建 RPC 服务
#   docker build --build-arg SERVICE=user --build-arg TYPE=rpc -t campushub/user-rpc .
#   docker build --build-arg SERVICE=activity --build-arg TYPE=rpc -t campushub/activity-rpc .
#   docker build --build-arg SERVICE=chat --build-arg TYPE=rpc -t campushub/chat-rpc .
#
# ============================================================================

# ==================== 构建阶段 ====================
FROM golang:1.24-alpine AS builder

ENV GO111MODULE=on \
    GOPROXY=https://goproxy.cn,direct \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

RUN apk add --no-cache git

WORKDIR /build

# 复制依赖文件并下载
COPY go.mod go.sum ./
RUN go mod download

# 复制源码
COPY . .

# 构建参数
ARG SERVICE=user
ARG TYPE=rpc

# 根据参数构建不同的服务
RUN if [ "$TYPE" = "api" ]; then \
        go build -ldflags="-s -w" -o /app/server ./app/${SERVICE}/api/*.go; \
    elif [ "$TYPE" = "rpc" ]; then \
        go build -ldflags="-s -w" -o /app/server ./app/${SERVICE}/rpc/*.go; \
    fi

# ==================== 运行阶段 ====================
FROM alpine:3.18

RUN apk add --no-cache tzdata ca-certificates && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone

WORKDIR /app

COPY --from=builder /app/server .

RUN mkdir -p /app/etc

# API 端口: 8001-8003, RPC 端口: 9001-9003
EXPOSE 8001 8002 8003 9001 9002 9003

ENTRYPOINT ["./server"]
CMD ["-f", "/app/etc/config.yaml"]
