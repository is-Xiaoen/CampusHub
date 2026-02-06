#!/bin/bash
# ============================================================================
# CampusHub Service Management Script
# ============================================================================
#
# Usage:
#   ./run.sh start              # Start all services
#   ./run.sh stop               # Stop all services
#   ./run.sh restart            # Restart all services
#   ./run.sh restart user       # Restart user service only
#   ./run.sh restart activity   # Restart activity service only
#   ./run.sh status             # View service status
#   ./run.sh logs user-api      # View specific service logs
#
# ============================================================================

BASE_DIR="/opt/campushub"
BIN_DIR="$BASE_DIR/bin"
CONFIG_DIR="$BASE_DIR/config"
LOG_DIR="$BASE_DIR/logs"
PID_DIR="$BASE_DIR/pids"

# Service list (start order: RPC first, then API)
SERVICES="user-rpc user-api activity-rpc activity-api chat-rpc chat-api"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

# Create necessary directories
mkdir -p "$LOG_DIR" "$PID_DIR"

# ==================== Functions ====================

start_service() {
    local service=$1
    local pid_file="$PID_DIR/${service}.pid"
    local log_file="$LOG_DIR/${service}.log"
    local config_file="$CONFIG_DIR/${service}.yaml"
    local bin_file="$BIN_DIR/${service}"

    # Check if binary exists
    if [ ! -f "$bin_file" ]; then
        echo -e "  ${YELLOW}[SKIP]${NC} $service (binary not found)"
        return 1
    fi

    # Check if config exists
    if [ ! -f "$config_file" ]; then
        echo -e "  ${RED}[ERROR]${NC} $service (config not found: $config_file)"
        return 1
    fi

    # Check if already running
    if [ -f "$pid_file" ]; then
        local pid=$(cat "$pid_file")
        if kill -0 "$pid" 2>/dev/null; then
            echo -e "  ${YELLOW}[RUNNING]${NC} $service (PID: $pid)"
            return 0
        fi
    fi

    # Start service
    nohup "$bin_file" -f "$config_file" >> "$log_file" 2>&1 &
    local pid=$!
    echo $pid > "$pid_file"

    # Wait and verify
    sleep 1
    if kill -0 "$pid" 2>/dev/null; then
        echo -e "  ${GREEN}[START]${NC} $service (PID: $pid)"
        return 0
    else
        echo -e "  ${RED}[FAILED]${NC} $service"
        rm -f "$pid_file"
        return 1
    fi
}

stop_service() {
    local service=$1
    local pid_file="$PID_DIR/${service}.pid"

    if [ ! -f "$pid_file" ]; then
        echo -e "  ${YELLOW}[NOT RUNNING]${NC} $service"
        return 0
    fi

    local pid=$(cat "$pid_file")
    if kill -0 "$pid" 2>/dev/null; then
        kill "$pid" 2>/dev/null
        sleep 1
        # Force kill if still running
        if kill -0 "$pid" 2>/dev/null; then
            kill -9 "$pid" 2>/dev/null
        fi
        echo -e "  ${GREEN}[STOP]${NC} $service (PID: $pid)"
    else
        echo -e "  ${YELLOW}[ALREADY STOPPED]${NC} $service"
    fi
    rm -f "$pid_file"
}

status_service() {
    local service=$1
    local pid_file="$PID_DIR/${service}.pid"
    local bin_file="$BIN_DIR/${service}"

    # Check binary
    if [ ! -f "$bin_file" ]; then
        echo -e "  $service: ${YELLOW}NOT INSTALLED${NC}"
        return
    fi

    if [ -f "$pid_file" ]; then
        local pid=$(cat "$pid_file")
        if kill -0 "$pid" 2>/dev/null; then
            echo -e "  $service: ${GREEN}RUNNING${NC} (PID: $pid)"
        else
            echo -e "  $service: ${RED}STOPPED${NC} (stale PID file)"
        fi
    else
        echo -e "  $service: ${YELLOW}NOT STARTED${NC}"
    fi
}

get_services() {
    local filter=$1
    case $filter in
        user)
            echo "user-rpc user-api"
            ;;
        activity)
            echo "activity-rpc activity-api"
            ;;
        chat)
            echo "chat-rpc chat-api"
            ;;
        all|"")
            echo "$SERVICES"
            ;;
        *)
            # Single service name
            echo "$filter"
            ;;
    esac
}

# ==================== Command Handling ====================

case "$1" in
    start)
        echo -e "${CYAN}Starting services...${NC}"
        services=$(get_services "$2")
        for service in $services; do
            start_service "$service"
            # Wait after RPC starts before starting API
            if [[ "$service" == *"-rpc" ]]; then
                sleep 1
            fi
        done
        ;;

    stop)
        echo -e "${CYAN}Stopping services...${NC}"
        # Stop in reverse order: API first, then RPC
        services=$(get_services "$2" | tr ' ' '\n' | tac | tr '\n' ' ')
        for service in $services; do
            stop_service "$service"
        done
        ;;

    restart)
        echo -e "${CYAN}Restarting services...${NC}"
        services=$(get_services "$2")

        # Stop first
        echo ""
        echo "Stopping:"
        for service in $(echo $services | tr ' ' '\n' | tac | tr '\n' ' '); do
            stop_service "$service"
        done

        sleep 2

        # Then start
        echo ""
        echo "Starting:"
        for service in $services; do
            start_service "$service"
            if [[ "$service" == *"-rpc" ]]; then
                sleep 1
            fi
        done
        ;;

    status)
        echo -e "${CYAN}Service Status:${NC}"
        echo ""
        for service in $SERVICES; do
            status_service "$service"
        done
        echo ""
        ;;

    logs)
        if [ -z "$2" ]; then
            echo "Please specify service name, e.g.: ./run.sh logs user-api"
            exit 1
        fi
        log_file="$LOG_DIR/${2}.log"
        if [ -f "$log_file" ]; then
            tail -f "$log_file"
        else
            echo "Log file not found: $log_file"
            exit 1
        fi
        ;;

    *)
        echo ""
        echo "CampusHub Service Management"
        echo ""
        echo "Usage: $0 {start|stop|restart|status|logs} [service]"
        echo ""
        echo "Commands:"
        echo "  start   [service]  Start services (default: all)"
        echo "  stop    [service]  Stop services (default: all)"
        echo "  restart [service]  Restart services (default: all)"
        echo "  status             Show all service status"
        echo "  logs    <service>  Tail service log"
        echo ""
        echo "Services:"
        echo "  all       All services (default)"
        echo "  user      user-api + user-rpc"
        echo "  activity  activity-api + activity-rpc"
        echo "  chat      chat-api + chat-rpc"
        echo "  user-api  Single service"
        echo ""
        echo "Examples:"
        echo "  $0 start              # Start all"
        echo "  $0 restart user       # Restart user services"
        echo "  $0 logs activity-api  # View logs"
        echo ""
        exit 1
        ;;
esac
