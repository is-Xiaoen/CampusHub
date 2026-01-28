#!/bin/bash
# ============================================================================
# CampusHub 部署脚本
# ============================================================================
# 使用方法：
#   ./deploy.sh build    # 构建镜像
#   ./deploy.sh push     # 推送镜像到服务器
#   ./deploy.sh start    # 启动服务
#   ./deploy.sh stop     # 停止服务
#   ./deploy.sh restart  # 重启服务
#   ./deploy.sh logs     # 查看日志
#   ./deploy.sh status   # 查看状态
# ============================================================================

set -e

# 配置
APP_SERVER="192.168.10.9"
APP_USER="root"
DEPLOY_DIR="/opt/campushub"
COMPOSE_FILE="docker-compose-app.yaml"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# 构建镜像
build() {
    log_info "开始构建镜像..."
    cd "$(dirname "$0")/docker"

    for service in gateway user activity chat demo; do
        log_info "构建 $service 服务..."
        docker build --build-arg SERVICE=$service -t campushub/$service:latest -f Dockerfile ../..
    done

    log_info "镜像构建完成！"
    docker images | grep campushub
}

# 推送到服务器
push() {
    log_info "推送到服务器 $APP_SERVER..."

    # 保存镜像
    log_info "打包镜像..."
    docker save campushub/gateway campushub/user campushub/activity campushub/chat campushub/demo \
        | gzip > /tmp/campushub-images.tar.gz

    # 传输到服务器
    log_info "传输镜像到服务器..."
    scp /tmp/campushub-images.tar.gz $APP_USER@$APP_SERVER:/tmp/

    # 在服务器上加载镜像
    log_info "加载镜像..."
    ssh $APP_USER@$APP_SERVER "gunzip -c /tmp/campushub-images.tar.gz | docker load"

    # 清理
    rm -f /tmp/campushub-images.tar.gz
    ssh $APP_USER@$APP_SERVER "rm -f /tmp/campushub-images.tar.gz"

    log_info "镜像推送完成！"
}

# 初始化服务器目录
init() {
    log_info "初始化服务器目录..."
    ssh $APP_USER@$APP_SERVER "mkdir -p $DEPLOY_DIR/config"

    # 上传 docker-compose 文件
    scp "$(dirname "$0")/docker/$COMPOSE_FILE" $APP_USER@$APP_SERVER:$DEPLOY_DIR/

    log_warn "请确保已上传配置文件到 $APP_SERVER:$DEPLOY_DIR/config/"
    log_info "初始化完成！"
}

# 启动服务
start() {
    log_info "启动服务..."
    ssh $APP_USER@$APP_SERVER "cd $DEPLOY_DIR && docker-compose -f $COMPOSE_FILE up -d"
    log_info "服务启动完成！"
    status
}

# 停止服务
stop() {
    log_info "停止服务..."
    ssh $APP_USER@$APP_SERVER "cd $DEPLOY_DIR && docker-compose -f $COMPOSE_FILE down"
    log_info "服务已停止"
}

# 重启服务
restart() {
    stop
    start
}

# 查看日志
logs() {
    service=${1:-""}
    if [ -n "$service" ]; then
        ssh $APP_USER@$APP_SERVER "cd $DEPLOY_DIR && docker-compose -f $COMPOSE_FILE logs -f $service"
    else
        ssh $APP_USER@$APP_SERVER "cd $DEPLOY_DIR && docker-compose -f $COMPOSE_FILE logs -f"
    fi
}

# 查看状态
status() {
    log_info "服务状态："
    ssh $APP_USER@$APP_SERVER "cd $DEPLOY_DIR && docker-compose -f $COMPOSE_FILE ps"
}

# 主函数
case "$1" in
    build)
        build
        ;;
    push)
        push
        ;;
    init)
        init
        ;;
    start)
        start
        ;;
    stop)
        stop
        ;;
    restart)
        restart
        ;;
    logs)
        logs $2
        ;;
    status)
        status
        ;;
    *)
        echo "用法: $0 {build|push|init|start|stop|restart|logs|status}"
        echo ""
        echo "命令说明："
        echo "  build   - 构建 Docker 镜像"
        echo "  push    - 推送镜像到服务器"
        echo "  init    - 初始化服务器目录"
        echo "  start   - 启动服务"
        echo "  stop    - 停止服务"
        echo "  restart - 重启服务"
        echo "  logs    - 查看日志 (可指定服务名)"
        echo "  status  - 查看服务状态"
        exit 1
        ;;
esac
