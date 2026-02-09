# ============================================================================
# CampusHub 多阶段构建 Dockerfile（v3.3 离线构建版）
# ============================================================================
#
# 优化历史：
#   v3.0 - 基础多阶段构建
#   v3.1 - 零 apk 依赖，适配网络受限环境
#   v3.2 - ARG 后移至 COPY 之后，go mod download 层跨服务复用
#   v3.3 - 基础镜像锁定 digest，BuildKit 不再联网检查镜像元数据
#          彻底消除 Docker Hub 镜像源不可用导致的构建失败
#
# 升级基础镜像步骤：
#   1. 在网络正常时执行：docker pull golang:1.24-alpine
#   2. 查看新 digest：docker inspect --format='{{index .RepoDigests 0}}' golang:1.24-alpine
#   3. 更新下方 FROM 行的 @sha256:... 部分
#   4. alpine:3.20 同理
#
# ============================================================================

# ==================== 构建阶段 ====================
# 锁定 digest：本地已缓存时 BuildKit 直接使用，完全不联网
FROM golang:1.24-alpine@sha256:8bee1901f1e530bfb4a7850aa7a479d17ae3a18beb6e09064ed54cfd245b7191 AS builder

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
FROM alpine:3.20@sha256:a4f4213abb84c497377b8544c81b3564f313746700372ec4fe84653e4fb03805

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
