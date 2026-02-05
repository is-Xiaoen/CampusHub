# 服务器部署指南

> **应用服务器**：192.168.10.9
> **基础设施服务器**：192.168.10.4

## 一、架构说明

```
┌─────────────────────────────────────────────────────────────────┐
│                      192.168.10.4 (基础设施)                     │
│  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐    │
│  │ MySQL   │ │ Redis   │ │  Etcd   │ │ Jaeger  │ │   DTM   │    │
│  │ :3308   │ │ :6379   │ │ :2379   │ │ :14268  │ │ :36790  │    │
│  └─────────┘ └─────────┘ └─────────┘ └─────────┘ └─────────┘    │
└─────────────────────────────────────────────────────────────────┘
                              ↑
                              │ TCP 连接
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│                      192.168.10.9 (应用服务)                     │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                    Nginx (:8888)                        │    │
│  │                    统一入口网关                          │    │
│  └─────────────────────────────────────────────────────────┘    │
│           ↓                    ↓                    ↓           │
│  ┌─────────────┐      ┌─────────────┐      ┌─────────────┐     │
│  │ user-api    │      │activity-api │      │  chat-api   │     │
│  │   :8001     │      │   :8002     │      │   :8003     │     │
│  └─────────────┘      └─────────────┘      └─────────────┘     │
│           ↓                    ↓                    ↓           │
│  ┌─────────────┐      ┌─────────────┐      ┌─────────────┐     │
│  │ user-rpc    │      │activity-rpc │      │  chat-rpc   │     │
│  │   :9001     │      │   :9002     │      │   :9003     │     │
│  └─────────────┘      └─────────────┘      └─────────────┘     │
└─────────────────────────────────────────────────────────────────┘
```

## 二、服务器初始化（首次部署）

### 2.1 安装 Go 环境

```bash
# SSH 登录到 192.168.10.9
ssh root@192.168.10.9

# 下载 Go 1.21+（根据实际版本调整）
wget https://go.dev/dl/go1.21.6.linux-amd64.tar.gz

# 解压到 /usr/local
tar -C /usr/local -xzf go1.21.6.linux-amd64.tar.gz

# 配置环境变量
cat >> /etc/profile << 'EOF'
# Go 环境变量
export GOROOT=/usr/local/go
export GOPATH=/root/go
export PATH=$PATH:$GOROOT/bin:$GOPATH/bin
export GOPROXY=https://goproxy.cn,direct
EOF

# 生效
source /etc/profile

# 验证
go version
```

### 2.2 安装 Git

```bash
yum install -y git   # CentOS/RHEL
# 或
apt install -y git   # Ubuntu/Debian
```

### 2.3 创建部署目录

```bash
mkdir -p /opt/campushub
cd /opt/campushub
```

### 2.4 克隆代码

```bash
# 首次克隆
git clone https://github.com/your-org/activity-platform.git

# 后续更新
cd /opt/campushub/activity-platform
git pull origin main
```

## 三、日常部署流程

### 3.1 一键部署

```bash
# 登录服务器
ssh root@192.168.10.9

# 进入项目目录
cd /opt/campushub/activity-platform

# 执行部署脚本
./deploy/server/deploy.sh deploy
```

### 3.2 部署单个服务

```bash
# 只部署活动服务
./deploy/server/deploy.sh deploy activity

# 只部署用户服务
./deploy/server/deploy.sh deploy user
```

### 3.3 查看服务状态

```bash
./deploy/server/deploy.sh status
```

### 3.4 查看日志

```bash
# 查看所有日志
./deploy/server/deploy.sh logs

# 查看指定服务日志
./deploy/server/deploy.sh logs activity-rpc
```

### 3.5 重启服务

```bash
# 重启所有服务
./deploy/server/deploy.sh restart

# 重启单个服务
./deploy/server/deploy.sh restart activity-rpc
```

### 3.6 停止服务

```bash
./deploy/server/deploy.sh stop
```

## 四、端口说明

| 服务 | 端口 | 说明 |
|------|------|------|
| Nginx | 8888 | 统一入口网关 |
| user-api | 8001 | 用户 HTTP 接口 |
| user-rpc | 9001 | 用户 gRPC 服务 |
| activity-api | 8002 | 活动 HTTP 接口 |
| activity-rpc | 9002 | 活动 gRPC 服务 |
| chat-api | 8003 | 聊天 HTTP 接口 |
| chat-rpc | 9003 | 聊天 gRPC 服务 |

## 五、配置文件位置

部署时会自动复制配置文件到：

```
/opt/campushub/config/
├── user-api.yaml
├── user-rpc.yaml
├── activity-api.yaml
├── activity-rpc.yaml
├── chat-api.yaml
└── chat-rpc.yaml
```

**重要**：如需修改配置，编辑 `/opt/campushub/config/` 下的文件，然后重启服务。

## 六、故障排查

### 6.1 服务启动失败

```bash
# 查看详细日志
tail -100 /opt/campushub/logs/activity-rpc.log

# 检查端口占用
netstat -tlnp | grep 9002

# 检查进程
ps aux | grep activity
```

### 6.2 无法连接基础设施

```bash
# 测试 MySQL 连接
nc -zv 192.168.10.4 3308

# 测试 Redis 连接
nc -zv 192.168.10.4 6379

# 测试 Etcd 连接
nc -zv 192.168.10.4 2379
```

### 6.3 查看 Etcd 注册信息

```bash
# 在基础设施服务器上执行
etcdctl get --prefix /
```

## 七、回滚

```bash
# 查看 Git 提交历史
cd /opt/campushub/activity-platform
git log --oneline -10

# 回滚到指定版本
git checkout <commit-hash>

# 重新部署
./deploy/server/deploy.sh deploy
```
