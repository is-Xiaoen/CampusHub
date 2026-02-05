#!/bin/bash
# ============================================================================
# CampusHub 服务器部署脚本
# ============================================================================
#
# 使用方法：
#   ./deploy.sh deploy [service]  # 部署服务（可选指定服务名）
#   ./deploy.sh start [service]   # 启动服务
#   ./deploy.sh stop [service]    # 停止服务
#   ./deploy.sh restart [service] # 重启服务
#   ./deploy.sh status            # 查看状态
#   ./deploy.sh logs [service]    # 查看日志
#   ./deploy.sh update            # 更新代码（不重启）
#
# 示例：
#   ./deploy.sh deploy            # 部署所有服务
#   ./deploy.sh deploy activity   # 只部署活动服务
#   ./deploy.sh restart user-rpc  # 重启 user-rpc
#   ./deploy.sh logs activity-rpc # 查看 activity-rpc 日志
#
# ============================================================================

set -e

# ==================== 配置 ====================
PROJECT_ROOT="/opt/campushub/activity-platform"
CONFIG_DIR="/opt/campushub/config"
LOG_DIR="/opt/campushub/logs"
PID_DIR="/opt/campushub/pids"
BIN_DIR="/opt/campushub/bin"

# 服务列表
SERVICES=(
    "user-rpc:app/user/rpc:user.go:9001"
    "user-api:app/user/api:user.go:8001"
    "activity-rpc:app/activity/rpc:activity.go:9002"
    "activity-api:app/activity/api:activity.go:8002"
    "chat-rpc:app/chat/rpc:chat.go:9003"
    "chat-api:app/chat/api:chat.go:8003"
)

# ==================== 颜色输出 ====================
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info()  { echo -e "${GREEN}[INFO]${NC} $(date '+%Y-%m-%d %H:%M:%S') $1"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC} $(date '+%Y-%m-%d %H:%M:%S') $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $(date '+%Y-%m-%d %H:%M:%S') $1"; }
log_title() { echo -e "\n${BLUE}========== $1 ==========${NC}\n"; }

# ==================== 初始化目录 ====================
init_dirs() {
    mkdir -p "$CONFIG_DIR" "$LOG_DIR" "$PID_DIR" "$BIN_DIR"
}

# ==================== 解析服务信息 ====================
# 参数: 服务名 (如 user-rpc)
# 返回: SVC_NAME, SVC_PATH, SVC_MAIN, SVC_PORT
parse_service() {
    local name=$1
    for svc in "${SERVICES[@]}"; do
        IFS=':' read -r svc_name svc_path svc_main svc_port <<< "$svc"
        if [[ "$svc_name" == "$name" ]]; then
            SVC_NAME=$svc_name
            SVC_PATH=$svc_path
            SVC_MAIN=$svc_main
            SVC_PORT=$svc_port
            return 0
        fi
    done
    return 1
}

# ==================== 获取服务列表 ====================
# 参数: 可选的服务名或服务组（user, activity, chat）
get_services() {
    local filter=$1
    local result=()

    if [[ -z "$filter" ]]; then
        # 返回所有服务
        for svc in "${SERVICES[@]}"; do
            IFS=':' read -r svc_name _ _ _ <<< "$svc"
            result+=("$svc_name")
        done
    elif [[ "$filter" == "user" || "$filter" == "activity" || "$filter" == "chat" ]]; then
        # 返回指定组的服务
        for svc in "${SERVICES[@]}"; do
            IFS=':' read -r svc_name _ _ _ <<< "$svc"
            if [[ "$svc_name" == "$filter"* ]]; then
                result+=("$svc_name")
            fi
        done
    else
        # 返回指定服务
        result+=("$filter")
    fi

    echo "${result[@]}"
}

# ==================== 复制配置文件 ====================
copy_configs() {
    log_info "复制配置文件..."

    # User 服务
    cp "$PROJECT_ROOT/app/user/rpc/etc/user.yaml" "$CONFIG_DIR/user-rpc.yaml"
    cp "$PROJECT_ROOT/app/user/api/etc/user-api.yaml" "$CONFIG_DIR/user-api.yaml"

    # Activity 服务
    cp "$PROJECT_ROOT/app/activity/rpc/etc/activity.yaml" "$CONFIG_DIR/activity-rpc.yaml"
    cp "$PROJECT_ROOT/app/activity/api/etc/activity-api.yaml" "$CONFIG_DIR/activity-api.yaml"

    # Chat 服务
    if [[ -f "$PROJECT_ROOT/app/chat/rpc/etc/chat.yaml" ]]; then
        cp "$PROJECT_ROOT/app/chat/rpc/etc/chat.yaml" "$CONFIG_DIR/chat-rpc.yaml"
    fi
    if [[ -f "$PROJECT_ROOT/app/chat/api/etc/chat-api.yaml" ]]; then
        cp "$PROJECT_ROOT/app/chat/api/etc/chat-api.yaml" "$CONFIG_DIR/chat-api.yaml"
    fi

    log_info "配置文件复制完成"
}

# ==================== 更新代码 ====================
update_code() {
    log_title "更新代码"
    cd "$PROJECT_ROOT"

    log_info "当前分支: $(git branch --show-current)"
    log_info "拉取最新代码..."

    git fetch origin
    git pull origin $(git branch --show-current)

    log_info "当前版本: $(git log --oneline -1)"
}

# ==================== 编译服务 ====================
build_service() {
    local name=$1
    parse_service "$name" || { log_error "未知服务: $name"; return 1; }

    log_info "编译 $SVC_NAME..."
    cd "$PROJECT_ROOT/$SVC_PATH"

    # 编译
    go build -o "$BIN_DIR/$SVC_NAME" "$SVC_MAIN"

    if [[ $? -eq 0 ]]; then
        log_info "$SVC_NAME 编译成功"
    else
        log_error "$SVC_NAME 编译失败"
        return 1
    fi
}

# ==================== 启动服务 ====================
start_service() {
    local name=$1
    parse_service "$name" || { log_error "未知服务: $name"; return 1; }

    local pid_file="$PID_DIR/$SVC_NAME.pid"
    local log_file="$LOG_DIR/$SVC_NAME.log"
    local config_file="$CONFIG_DIR/$SVC_NAME.yaml"
    local bin_file="$BIN_DIR/$SVC_NAME"

    # 检查是否已运行
    if [[ -f "$pid_file" ]]; then
        local pid=$(cat "$pid_file")
        if kill -0 "$pid" 2>/dev/null; then
            log_warn "$SVC_NAME 已在运行 (PID: $pid)"
            return 0
        fi
        rm -f "$pid_file"
    fi

    # 检查二进制文件
    if [[ ! -f "$bin_file" ]]; then
        log_error "$SVC_NAME 二进制文件不存在，请先编译"
        return 1
    fi

    # 检查配置文件
    if [[ ! -f "$config_file" ]]; then
        log_error "$SVC_NAME 配置文件不存在: $config_file"
        return 1
    fi

    log_info "启动 $SVC_NAME..."

    # 启动服务
    nohup "$bin_file" -f "$config_file" >> "$log_file" 2>&1 &
    local pid=$!
    echo $pid > "$pid_file"

    # 等待并检查
    sleep 2
    if kill -0 "$pid" 2>/dev/null; then
        log_info "$SVC_NAME 启动成功 (PID: $pid, Port: $SVC_PORT)"
    else
        log_error "$SVC_NAME 启动失败，查看日志: $log_file"
        rm -f "$pid_file"
        return 1
    fi
}

# ==================== 停止服务 ====================
stop_service() {
    local name=$1
    parse_service "$name" || { log_error "未知服务: $name"; return 1; }

    local pid_file="$PID_DIR/$SVC_NAME.pid"

    if [[ ! -f "$pid_file" ]]; then
        log_warn "$SVC_NAME 未运行"
        return 0
    fi

    local pid=$(cat "$pid_file")
    log_info "停止 $SVC_NAME (PID: $pid)..."

    # 优雅停止
    kill -SIGTERM "$pid" 2>/dev/null || true

    # 等待进程退出
    local count=0
    while kill -0 "$pid" 2>/dev/null && [[ $count -lt 10 ]]; do
        sleep 1
        ((count++))
    done

    # 强制停止
    if kill -0 "$pid" 2>/dev/null; then
        log_warn "优雅停止超时，强制终止..."
        kill -9 "$pid" 2>/dev/null || true
    fi

    rm -f "$pid_file"
    log_info "$SVC_NAME 已停止"
}

# ==================== 查看服务状态 ====================
show_status() {
    log_title "服务状态"

    printf "%-15s %-8s %-8s %-10s\n" "服务名" "状态" "端口" "PID"
    printf "%-15s %-8s %-8s %-10s\n" "-------" "----" "----" "---"

    for svc in "${SERVICES[@]}"; do
        IFS=':' read -r svc_name svc_path svc_main svc_port <<< "$svc"
        local pid_file="$PID_DIR/$svc_name.pid"
        local status="stopped"
        local pid="-"

        if [[ -f "$pid_file" ]]; then
            pid=$(cat "$pid_file")
            if kill -0 "$pid" 2>/dev/null; then
                status="${GREEN}running${NC}"
            else
                status="${RED}dead${NC}"
                pid="-"
            fi
        fi

        printf "%-15s %-18b %-8s %-10s\n" "$svc_name" "$status" "$svc_port" "$pid"
    done
}

# ==================== 查看日志 ====================
show_logs() {
    local name=$1

    if [[ -z "$name" ]]; then
        # 显示所有日志
        tail -f "$LOG_DIR"/*.log
    else
        local log_file="$LOG_DIR/$name.log"
        if [[ -f "$log_file" ]]; then
            tail -f "$log_file"
        else
            log_error "日志文件不存在: $log_file"
            return 1
        fi
    fi
}

# ==================== 部署 ====================
deploy() {
    local filter=$1
    local services=($(get_services "$filter"))

    log_title "开始部署"
    init_dirs

    # 1. 更新代码
    update_code

    # 2. 下载依赖
    log_info "下载依赖..."
    cd "$PROJECT_ROOT"
    go mod tidy

    # 3. 复制配置
    copy_configs

    # 4. 编译服务
    log_title "编译服务"
    for svc in "${services[@]}"; do
        build_service "$svc" || exit 1
    done

    # 5. 重启服务（先停后启，按依赖顺序）
    log_title "重启服务"

    # 停止顺序：API -> RPC
    for svc in "${services[@]}"; do
        if [[ "$svc" == *"-api" ]]; then
            stop_service "$svc"
        fi
    done
    for svc in "${services[@]}"; do
        if [[ "$svc" == *"-rpc" ]]; then
            stop_service "$svc"
        fi
    done

    sleep 2

    # 启动顺序：RPC -> API
    for svc in "${services[@]}"; do
        if [[ "$svc" == *"-rpc" ]]; then
            start_service "$svc"
        fi
    done
    sleep 2
    for svc in "${services[@]}"; do
        if [[ "$svc" == *"-api" ]]; then
            start_service "$svc"
        fi
    done

    # 6. 显示状态
    show_status

    log_title "部署完成"
    log_info "版本: $(cd $PROJECT_ROOT && git log --oneline -1)"
}

# ==================== 启动所有 ====================
start_all() {
    local filter=$1
    local services=($(get_services "$filter"))

    log_title "启动服务"

    # 启动顺序：RPC -> API
    for svc in "${services[@]}"; do
        if [[ "$svc" == *"-rpc" ]]; then
            start_service "$svc"
        fi
    done
    sleep 2
    for svc in "${services[@]}"; do
        if [[ "$svc" == *"-api" ]]; then
            start_service "$svc"
        fi
    done

    show_status
}

# ==================== 停止所有 ====================
stop_all() {
    local filter=$1
    local services=($(get_services "$filter"))

    log_title "停止服务"

    # 停止顺序：API -> RPC
    for svc in "${services[@]}"; do
        if [[ "$svc" == *"-api" ]]; then
            stop_service "$svc"
        fi
    done
    for svc in "${services[@]}"; do
        if [[ "$svc" == *"-rpc" ]]; then
            stop_service "$svc"
        fi
    done

    show_status
}

# ==================== 重启所有 ====================
restart_all() {
    local filter=$1
    stop_all "$filter"
    sleep 2
    start_all "$filter"
}

# ==================== 主函数 ====================
case "$1" in
    deploy)
        deploy "$2"
        ;;
    start)
        if [[ -n "$2" ]] && parse_service "$2"; then
            start_service "$2"
        else
            start_all "$2"
        fi
        ;;
    stop)
        if [[ -n "$2" ]] && parse_service "$2"; then
            stop_service "$2"
        else
            stop_all "$2"
        fi
        ;;
    restart)
        if [[ -n "$2" ]] && parse_service "$2"; then
            stop_service "$2"
            sleep 1
            start_service "$2"
        else
            restart_all "$2"
        fi
        ;;
    status)
        show_status
        ;;
    logs)
        show_logs "$2"
        ;;
    update)
        update_code
        ;;
    *)
        echo "CampusHub 服务器部署脚本"
        echo ""
        echo "用法: $0 <命令> [服务名]"
        echo ""
        echo "命令:"
        echo "  deploy [service]   部署服务（拉代码+编译+重启）"
        echo "  start [service]    启动服务"
        echo "  stop [service]     停止服务"
        echo "  restart [service]  重启服务"
        echo "  status             查看服务状态"
        echo "  logs [service]     查看日志"
        echo "  update             仅更新代码（不重启）"
        echo ""
        echo "服务名:"
        echo "  user               用户服务组（user-rpc + user-api）"
        echo "  activity           活动服务组（activity-rpc + activity-api）"
        echo "  chat               聊天服务组（chat-rpc + chat-api）"
        echo "  user-rpc           用户 RPC 服务"
        echo "  user-api           用户 API 服务"
        echo "  activity-rpc       活动 RPC 服务"
        echo "  activity-api       活动 API 服务"
        echo "  chat-rpc           聊天 RPC 服务"
        echo "  chat-api           聊天 API 服务"
        echo ""
        echo "示例:"
        echo "  $0 deploy              # 部署所有服务"
        echo "  $0 deploy activity     # 只部署活动服务"
        echo "  $0 restart user-rpc    # 重启 user-rpc"
        echo "  $0 logs activity-rpc   # 查看 activity-rpc 日志"
        exit 1
        ;;
esac
