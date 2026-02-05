#!/bin/bash
# ============================================================================
# CampusHub 快捷操作脚本
# ============================================================================
#
# 上传到服务器后执行: chmod +x /opt/campushub/quick.sh
#
# 用法:
#   /opt/campushub/quick.sh rebuild   # 重新构建并启动
#   /opt/campushub/quick.sh restart   # 重启服务
#   /opt/campushub/quick.sh stop      # 停止服务
#   /opt/campushub/quick.sh logs      # 查看日志
#   /opt/campushub/quick.sh status    # 查看状态
#   /opt/campushub/quick.sh ps        # 查看容器
#
# ============================================================================

COMPOSE_DIR="/opt/campushub/activity/deploy/docker"
COMPOSE_FILE="docker-compose-app.yaml"

cd "$COMPOSE_DIR" || { echo "目录不存在: $COMPOSE_DIR"; exit 1; }

case "$1" in
    rebuild)
        echo ">>> 重新构建并启动..."
        docker-compose -f $COMPOSE_FILE build
        docker-compose -f $COMPOSE_FILE up -d
        docker-compose -f $COMPOSE_FILE ps
        ;;
    restart)
        echo ">>> 重启服务..."
        docker-compose -f $COMPOSE_FILE restart
        docker-compose -f $COMPOSE_FILE ps
        ;;
    stop)
        echo ">>> 停止服务..."
        docker-compose -f $COMPOSE_FILE down
        ;;
    start)
        echo ">>> 启动服务..."
        docker-compose -f $COMPOSE_FILE up -d
        docker-compose -f $COMPOSE_FILE ps
        ;;
    logs)
        echo ">>> 查看日志 (Ctrl+C 退出)..."
        docker-compose -f $COMPOSE_FILE logs -f --tail=100
        ;;
    status|ps)
        docker-compose -f $COMPOSE_FILE ps
        ;;
    *)
        echo "CampusHub 快捷操作"
        echo ""
        echo "用法: $0 <命令>"
        echo ""
        echo "命令:"
        echo "  rebuild  - 重新构建并启动（代码更新后用这个）"
        echo "  restart  - 重启服务（配置更新后用这个）"
        echo "  stop     - 停止所有服务"
        echo "  start    - 启动所有服务"
        echo "  logs     - 查看实时日志"
        echo "  status   - 查看服务状态"
        ;;
esac
