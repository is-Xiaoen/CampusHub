# CampusHub 部署指南

## 服务器架构

```
┌──────────────────────────────┐     ┌──────────────────────────────┐
│  192.168.10.4 (中间件服务器)  │     │  192.168.10.9 (应用服务器)    │
│                              │     │                              │
│  MySQL   :3308               │     │  Gateway   :18080  ◄─ Apifox │
│  Redis   :6379               │◄───►│  User RPC  :19001            │
│  Etcd    :2379               │     │  Activity  :19002            │
│                              │     │  Chat RPC  :19003            │
└──────────────────────────────┘     │  Demo RPC  :19100            │
                                     └──────────────────────────────┘
```

## 端口分配

| 服务 | 端口 | 说明 |
|------|------|------|
| Gateway API | 18080 | HTTP 入口，Apifox 测试用这个 |
| User RPC | 19001 | 用户服务（gRPC） |
| Activity RPC | 19002 | 活动服务（gRPC） |
| Chat RPC | 19003 | 聊天服务（gRPC） |
| Demo RPC | 19100 | 示例服务（gRPC） |

---

## GitHub CI 说明

`.github/workflows/ci.yml` 会在以下情况自动运行：
- push 到 `main` 或 `develop` 分支
- 提交 Pull Request

**CI 只做检查，不会自动部署到服务器！**

检查内容：
1. `go fmt` - 代码格式
2. `go vet` - 静态分析
3. `go build` - 编译检查
4. `go test` - 单元测试

---

## 部署到服务器的步骤

### 方式一：手动部署（推荐新手）

#### 第 1 步：在本地编译

```bash
# Windows 下交叉编译 Linux 二进制
set GOOS=linux
set GOARCH=amd64
set CGO_ENABLED=0

# 编译各服务
go build -o bin/gateway ./app/gateway/api/gateway.go
go build -o bin/user ./app/user/rpc/user.go
go build -o bin/activity ./app/activity/rpc/activity.go
go build -o bin/chat ./app/chat/rpc/chat.go
go build -o bin/demo ./app/demo/rpc/demo.go
```

#### 第 2 步：上传到服务器

```bash
# 上传二进制文件
scp bin/* root@192.168.10.9:/opt/campushub/

# 上传配置文件
scp app/gateway/api/etc/gateway.yaml root@192.168.10.9:/opt/campushub/etc/
scp app/demo/rpc/etc/demo.yaml root@192.168.10.9:/opt/campushub/etc/
# ... 其他配置文件
```

#### 第 3 步：在服务器上运行

```bash
# SSH 登录服务器
ssh root@192.168.10.9

# 创建目录
mkdir -p /opt/campushub/etc

# 运行服务（使用 nohup 后台运行）
cd /opt/campushub
nohup ./gateway -f etc/gateway.yaml > logs/gateway.log 2>&1 &
nohup ./demo -f etc/demo.yaml > logs/demo.log 2>&1 &
# ... 其他服务

# 查看日志
tail -f logs/gateway.log
```

---

### 方式二：Docker 部署（推荐生产环境）

#### 第 1 步：在本地构建镜像

```bash
cd deploy/docker

# 构建所有服务镜像
docker build --build-arg SERVICE=gateway -t campushub/gateway:latest -f Dockerfile ../..
docker build --build-arg SERVICE=demo -t campushub/demo:latest -f Dockerfile ../..
# ... 其他服务
```

#### 第 2 步：导出并上传镜像

```bash
# 导出镜像
docker save campushub/gateway campushub/demo | gzip > campushub-images.tar.gz

# 上传到服务器
scp campushub-images.tar.gz root@192.168.10.9:/tmp/

# 在服务器上加载
ssh root@192.168.10.9 "gunzip -c /tmp/campushub-images.tar.gz | docker load"
```

#### 第 3 步：在服务器上运行

```bash
# 上传 docker-compose 文件和配置
scp docker-compose-app.yaml root@192.168.10.9:/opt/campushub/
scp -r config/ root@192.168.10.9:/opt/campushub/

# 在服务器上启动
ssh root@192.168.10.9 "cd /opt/campushub && docker-compose -f docker-compose-app.yaml up -d"
```

---

## Apifox 测试

服务启动后，在 Apifox 中配置：

**Base URL**: `http://192.168.10.9:18080`

测试接口：
- `GET http://192.168.10.9:18080/health` - 健康检查
- `GET http://192.168.10.9:18080/` - 服务信息

---

## 常见问题

### Q1: 端口被占用怎么办？

检查占用：
```bash
ss -tlnp | grep 18080
```

如果被占用，修改配置文件中的端口号。

### Q2: 服务启动失败？

检查日志：
```bash
# 手动部署
tail -f /opt/campushub/logs/gateway.log

# Docker 部署
docker logs campushub-gateway
```

### Q3: 连不上数据库？

确认：
1. 192.168.10.4 的 MySQL 允许远程连接
2. 防火墙开放了 3308 端口
3. 配置文件中的连接信息正确

---

## 快速验证清单

- [ ] 数据库可连接：`mysql -h 192.168.10.4 -P 3308 -u root -p`
- [ ] Redis 可连接：`redis-cli -h 192.168.10.4 -a 123456 PING`
- [ ] Etcd 可连接：`etcdctl --endpoints=192.168.10.4:2379 endpoint health`
- [ ] Gateway 健康：`curl http://192.168.10.9:18080/health`
