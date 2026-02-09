# ============================================================================
# CampusHub 多阶段构建 Dockerfile（加固版）
# ============================================================================
#
# 安全改进（v3.1）：
#   - 非 root 用户运行（appuser:appgroup）
#   - Alpine 3.20（最新稳定版）
#   - 零 apk 依赖：CA 证书和时区数据从 builder 阶段复制
#   - 适配网络受限环境（无需访问 Alpine 仓库）
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
    GOPROXY=https://goproxy.cn,https://goproxy.io,direct \
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

# 从 builder 阶段复制 CA 证书和时区数据（避免 apk add，服务器无法访问 Alpine 仓库）
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/local/go/lib/time/zoneinfo.zip /usr/local/zoneinfo.zip
ENV ZONEINFO=/usr/local/zoneinfo.zip
ENV TZ=Asia/Shanghai

RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app
COPY --from=builder /app/server .
RUN mkdir -p /app/etc && chown -R appuser:appgroup /app

USER appuser

ENTRYPOINT ["./server"]
CMD ["-f", "/app/etc/config.yaml"]