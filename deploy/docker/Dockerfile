# ============================================================================
# CampusHub 多阶段构建 Dockerfile（加固版）
# ============================================================================
#
# 安全改进（v3.0）：
#   - 非 root 用户运行（appuser:appgroup）
#   - Alpine 3.20（最新稳定版）
#   - 精简构建命令
#
# 使用方法：
#   docker build --build-arg SERVICE=user --build-arg TYPE=api \
#     -t ghcr.io/<owner>/campushub/user-api:latest \
#     -f deploy/docker/Dockerfile .
#
# ============================================================================

# ==================== 构建阶段 ====================
FROM golang:1.24-alpine AS builder

ARG SERVICE
ARG TYPE

ENV GO111MODULE=on \
    GOPROXY=https://goproxy.cn,direct \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

WORKDIR /build

# 利用层缓存：先复制依赖声明，下载依赖
COPY go.mod go.sum ./
RUN go mod download

# 复制源码并编译
COPY . .
RUN go build -ldflags="-s -w" -o /app/server ./app/${SERVICE}/${TYPE}/

# ==================== 运行阶段 ====================
FROM alpine:3.20

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories && \
    apk add --no-cache tzdata ca-certificates && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone && \
    addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app
COPY --from=builder /app/server .
RUN mkdir -p /app/etc && chown -R appuser:appgroup /app

USER appuser

ENTRYPOINT ["./server"]
CMD ["-f", "/app/etc/config.yaml"]