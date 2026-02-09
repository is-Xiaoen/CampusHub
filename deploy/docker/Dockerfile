# ============================================================================
# CampusHub 多阶段构建 Dockerfile（v3.2 缓存优化版）
# ============================================================================
#
# 优化历史：
#   v3.0 - 基础多阶段构建
#   v3.1 - 零 apk 依赖，适配网络受限环境
#   v3.2 - ARG 后移至 COPY 之后，go mod download 层跨服务复用
#          连续构建 6 个服务时，2-6 号服务跳过依赖下载（节省 ~5min/个）
#
# 使用方法：
#   docker build --build-arg SERVICE=user --build-arg TYPE=api \
#     -t ghcr.io/<owner>/campushub/user-api:latest \
#     -f deploy/docker/Dockerfile .
#
# ============================================================================

# ==================== 构建阶段 ====================
FROM golang:1.24-alpine AS builder

ENV GO111MODULE=on \
    GOPROXY=https://goproxy.cn,https://goproxy.io,direct \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

WORKDIR /build

# 第 1 层：依赖下载（go.mod/go.sum 不变则缓存命中）
COPY go.mod go.sum ./
RUN go mod download

# 第 2 层：源码复制（同一次 CI 内容不变则缓存命中）
COPY . .

# 第 3 层：编译（ARG 放在这里，仅此层因服务不同而重建）
ARG SERVICE
ARG TYPE
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