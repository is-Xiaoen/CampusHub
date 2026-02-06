# CampusHub 快速部署指南

## 一键部署流程

### 首次部署

```powershell
# 1. 初始化服务器（只需执行一次）
.\deploy\scripts\deploy.ps1 -Init

# 2. 完整部署
.\deploy\scripts\deploy.ps1
```

### 日常开发部署

```powershell
# 部署所有服务（编译 + 上传 + 重启）
.\deploy\scripts\deploy.ps1

# 只部署某个服务（更快）
.\deploy\scripts\deploy.ps1 -Service user
.\deploy\scripts\deploy.ps1 -Service activity
.\deploy\scripts\deploy.ps1 -Service chat

# 跳过编译（只上传已编译的文件）
.\deploy\scripts\deploy.ps1 -SkipBuild

# 同时更新配置文件
.\deploy\scripts\deploy.ps1 -UploadConfig
```

### 只编译不部署

```powershell
.\deploy\scripts\build.ps1           # 编译所有
.\deploy\scripts\build.ps1 user      # 只编译 user
.\deploy\scripts\build.ps1 activity  # 只编译 activity
```

## 服务器端管理

SSH 登录服务器后：

```bash
cd /opt/campushub

./run.sh start              # 启动所有服务
./run.sh stop               # 停止所有服务
./run.sh restart            # 重启所有服务
./run.sh restart user       # 只重启 user 服务
./run.sh status             # 查看服务状态
./run.sh logs user-api      # 查看日志（实时）
```

## 目录结构

```
本地项目:
deploy/
├── scripts/
│   ├── build.ps1       # 编译脚本
│   ├── deploy.ps1      # 部署脚本
│   └── README.md       # 本文档
├── bin/                # 编译输出（gitignore）
├── docker/config/      # 配置文件
└── server/
    └── run.sh          # 服务器运行脚本

服务器 192.168.10.9:
/opt/campushub/
├── bin/                # 二进制文件
│   ├── user-api
│   ├── user-rpc
│   ├── activity-api
│   ├── activity-rpc
│   ├── chat-api
│   └── chat-rpc
├── config/             # 配置文件
│   ├── user-api.yaml
│   ├── user-rpc.yaml
│   └── ...
├── logs/               # 日志目录
├── pids/               # PID 文件
└── run.sh              # 管理脚本
```

## 端口说明

| 服务 | 端口 |
|------|------|
| user-api | 8001 |
| user-rpc | 9001 |
| activity-api | 8002 |
| activity-rpc | 9002 |
| chat-api | 8003 |
| chat-rpc | 9003 |

## 常见问题

### Q: 上传很慢？
A: 使用 `-Service` 参数只部署修改的服务

### Q: 配置文件改了？
A: 加 `-UploadConfig` 参数上传配置

### Q: 编译失败？
A: 检查 Go 版本，需要 Go 1.24+
