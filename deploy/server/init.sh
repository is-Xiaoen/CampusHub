#!/bin/bash
# ============================================================================
# CampusHub 服务器初始化脚本
# ============================================================================
#
# 在 192.168.10.9 服务器上首次部署时执行此脚本
#
# 使用方法：
#   curl -sSL https://raw.githubusercontent.com/your-org/activity-platform/main/deploy/server/init.sh | bash
#
# 或者手动执行：
#   chmod +x init.sh && ./init.sh
#
# ============================================================================

set -e

# 配置
GO_VERSION="1.21.6"
PROJECT_REPO="https://github.com/your-org/activity-platform.git"  # 请替换为实际仓库地址
PROJECT_DIR="/opt/campushub/activity-platform"
CONFIG_DIR="/opt/campushub/config"
LOG_DIR="/opt/campushub/logs"
PID_DIR="/opt/campushub/pids"
BIN_DIR="/opt/campushub/bin"

# 颜色
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info()  { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# ==================== 检查系统 ====================
check_system() {
    log_info "检查系统..."

    # 检查是否为 root
    if [[ $EUID -ne 0 ]]; then
        log_error "请以 root 用户运行此脚本"
        exit 1
    fi

    # 检查系统类型
    if [[ -f /etc/redhat-release ]]; then
        OS="centos"
        PKG_MGR="yum"
    elif [[ -f /etc/debian_version ]]; then
        OS="debian"
        PKG_MGR="apt"
    else
        log_warn "未知的操作系统，可能需要手动安装依赖"
        OS="unknown"
    fi

    log_info "操作系统: $OS"
}

# ==================== 安装基础工具 ====================
install_base() {
    log_info "安装基础工具..."

    if [[ "$PKG_MGR" == "yum" ]]; then
        yum install -y git wget curl net-tools
    elif [[ "$PKG_MGR" == "apt" ]]; then
        apt update
        apt install -y git wget curl net-tools
    fi
}

# ==================== 安装 Go ====================
install_go() {
    log_info "检查 Go 环境..."

    if command -v go &> /dev/null; then
        local current_version=$(go version | awk '{print $3}' | sed 's/go//')
        log_info "已安装 Go $current_version"

        # 检查版本是否满足要求 (>= 1.21)
        if [[ "$(printf '%s\n' "1.21" "$current_version" | sort -V | head -n1)" == "1.21" ]]; then
            log_info "Go 版本满足要求"
            return 0
        fi
        log_warn "Go 版本过低，将升级..."
    fi

    log_info "安装 Go $GO_VERSION..."

    # 下载
    cd /tmp
    wget -q "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz"

    # 安装
    rm -rf /usr/local/go
    tar -C /usr/local -xzf "go${GO_VERSION}.linux-amd64.tar.gz"
    rm -f "go${GO_VERSION}.linux-amd64.tar.gz"

    # 配置环境变量
    if ! grep -q "GOROOT" /etc/profile; then
        cat >> /etc/profile << 'EOF'

# Go 环境变量
export GOROOT=/usr/local/go
export GOPATH=/root/go
export PATH=$PATH:$GOROOT/bin:$GOPATH/bin
export GOPROXY=https://goproxy.cn,direct
EOF
    fi

    # 生效
    export GOROOT=/usr/local/go
    export GOPATH=/root/go
    export PATH=$PATH:$GOROOT/bin:$GOPATH/bin
    export GOPROXY=https://goproxy.cn,direct

    log_info "Go 安装完成: $(go version)"
}

# ==================== 创建目录 ====================
create_dirs() {
    log_info "创建目录结构..."

    mkdir -p "$CONFIG_DIR"
    mkdir -p "$LOG_DIR"
    mkdir -p "$PID_DIR"
    mkdir -p "$BIN_DIR"
    mkdir -p "$(dirname $PROJECT_DIR)"

    log_info "目录结构:"
    echo "  $PROJECT_DIR  - 项目代码"
    echo "  $CONFIG_DIR   - 配置文件"
    echo "  $LOG_DIR      - 日志文件"
    echo "  $PID_DIR      - PID 文件"
    echo "  $BIN_DIR      - 二进制文件"
}

# ==================== 克隆代码 ====================
clone_repo() {
    log_info "克隆代码..."

    if [[ -d "$PROJECT_DIR" ]]; then
        log_info "项目已存在，更新代码..."
        cd "$PROJECT_DIR"
        git fetch origin
        git pull origin main
    else
        log_info "克隆仓库..."
        git clone "$PROJECT_REPO" "$PROJECT_DIR"
    fi

    cd "$PROJECT_DIR"
    log_info "当前版本: $(git log --oneline -1)"
}

# ==================== 复制配置文件 ====================
copy_configs() {
    log_info "复制配置文件..."

    # 检查是否有生产配置模板
    if [[ -d "$PROJECT_DIR/deploy/server/config" ]]; then
        cp "$PROJECT_DIR/deploy/server/config"/*.yaml "$CONFIG_DIR/" 2>/dev/null || true
        log_info "已复制生产配置模板"
    else
        # 复制开发配置
        cp "$PROJECT_DIR/app/user/rpc/etc/user.yaml" "$CONFIG_DIR/user-rpc.yaml" 2>/dev/null || true
        cp "$PROJECT_DIR/app/user/api/etc/user-api.yaml" "$CONFIG_DIR/user-api.yaml" 2>/dev/null || true
        cp "$PROJECT_DIR/app/activity/rpc/etc/activity.yaml" "$CONFIG_DIR/activity-rpc.yaml" 2>/dev/null || true
        cp "$PROJECT_DIR/app/activity/api/etc/activity-api.yaml" "$CONFIG_DIR/activity-api.yaml" 2>/dev/null || true
        log_warn "已复制开发配置，请检查并修改"
    fi

    log_info "配置文件位置: $CONFIG_DIR"
}

# ==================== 测试连接 ====================
test_connections() {
    log_info "测试基础设施连接..."

    local infra_host="192.168.10.4"
    local failed=0

    # MySQL
    if nc -zw3 "$infra_host" 3308 2>/dev/null; then
        log_info "MySQL ($infra_host:3308) ✓"
    else
        log_error "MySQL ($infra_host:3308) ✗"
        ((failed++))
    fi

    # Redis
    if nc -zw3 "$infra_host" 6379 2>/dev/null; then
        log_info "Redis ($infra_host:6379) ✓"
    else
        log_error "Redis ($infra_host:6379) ✗"
        ((failed++))
    fi

    # Etcd
    if nc -zw3 "$infra_host" 2379 2>/dev/null; then
        log_info "Etcd ($infra_host:2379) ✓"
    else
        log_error "Etcd ($infra_host:2379) ✗"
        ((failed++))
    fi

    if [[ $failed -gt 0 ]]; then
        log_warn "部分基础设施连接失败，请检查网络和服务状态"
    fi
}

# ==================== 设置部署脚本 ====================
setup_deploy_script() {
    log_info "设置部署脚本..."

    chmod +x "$PROJECT_DIR/deploy/server/deploy.sh"

    # 创建快捷方式
    ln -sf "$PROJECT_DIR/deploy/server/deploy.sh" /usr/local/bin/campushub

    log_info "已创建快捷命令: campushub"
    echo "  campushub deploy    - 部署服务"
    echo "  campushub status    - 查看状态"
    echo "  campushub logs      - 查看日志"
}

# ==================== 主函数 ====================
main() {
    echo ""
    echo "========================================"
    echo "   CampusHub 服务器初始化"
    echo "========================================"
    echo ""

    check_system
    install_base
    install_go
    create_dirs
    clone_repo
    copy_configs
    test_connections
    setup_deploy_script

    echo ""
    echo "========================================"
    echo "   初始化完成！"
    echo "========================================"
    echo ""
    echo "下一步："
    echo "  1. 检查配置文件: $CONFIG_DIR/"
    echo "  2. 部署服务: campushub deploy"
    echo "  3. 查看状态: campushub status"
    echo ""
}

main
