# 部署前检查清单

在执行部署之前，请逐项确认以下内容。

## 一、基础设施确认（192.168.10.4）

### 1.1 服务运行状态

| 服务 | 端口 | 检查命令 | 状态 |
|------|------|---------|------|
| MySQL | 3308 | `nc -zv 192.168.10.4 3308` | ☐ |
| Redis | 6379 | `nc -zv 192.168.10.4 6379` | ☐ |
| Etcd | 2379 | `nc -zv 192.168.10.4 2379` | ☐ |
| Jaeger | 14268 | `nc -zv 192.168.10.4 14268` | ☐ |
| DTM | 36790 | `nc -zv 192.168.10.4 36790` | ☐ |

### 1.2 数据库初始化

```bash
# 在 MySQL 中执行
mysql -h 192.168.10.4 -P 3308 -u root -p

# 检查数据库是否存在
SHOW DATABASES LIKE 'campushub%';

# 如果不存在，执行初始化脚本
source deploy/sql/user.sql
source deploy/sql/activity.sql
source deploy/sql/dtm.sql
```

- ☐ `campushub_user` 数据库已创建
- ☐ `campushub_main` 数据库已创建
- ☐ `dtm_barrier` 表已创建（两个数据库都需要）

## 二、应用服务器确认（192.168.10.9）

### 2.1 环境检查

```bash
# 检查 Go 版本（需要 >= 1.21）
go version

# 检查 Git
git --version

# 检查网络连接
ping -c 3 192.168.10.4
```

- ☐ Go 版本 >= 1.21
- ☐ Git 已安装
- ☐ 能够访问基础设施服务器

### 2.2 目录结构

```bash
ls -la /opt/campushub/
```

- ☐ `/opt/campushub/activity-platform` - 项目代码
- ☐ `/opt/campushub/config` - 配置文件
- ☐ `/opt/campushub/logs` - 日志目录
- ☐ `/opt/campushub/pids` - PID 文件
- ☐ `/opt/campushub/bin` - 二进制文件

### 2.3 端口占用检查

```bash
# 确保端口未被占用
netstat -tlnp | grep -E "8001|8002|8003|9001|9002|9003"
```

- ☐ 8001 端口可用
- ☐ 8002 端口可用
- ☐ 8003 端口可用
- ☐ 9001 端口可用
- ☐ 9002 端口可用
- ☐ 9003 端口可用

## 三、配置文件确认

### 3.1 AccessSecret 一致性

**关键！** 所有 API 服务的 `Auth.AccessSecret` 必须完全一致：

```bash
# 检查 AccessSecret 是否一致
grep -h "AccessSecret" /opt/campushub/config/*-api.yaml
```

- ☐ user-api.yaml 的 AccessSecret
- ☐ activity-api.yaml 的 AccessSecret
- ☐ chat-api.yaml 的 AccessSecret
- ☐ 以上三个完全一致

### 3.2 基础设施地址

确认所有配置文件中的基础设施地址正确：

```yaml
# 应该是
MySQL:
  Host: 192.168.10.4
  Port: 3308

Redis:
  Host: 192.168.10.4:6379

Etcd:
  Hosts:
    - 192.168.10.4:2379
```

- ☐ MySQL 地址正确
- ☐ Redis 地址正确
- ☐ Etcd 地址正确

### 3.3 DTM 配置（如果启用）

```yaml
DTM:
  Enabled: true
  Server: "192.168.10.4:36790"
  HTTPServer: "192.168.10.4:36789"
  ActivityRpcURL: "192.168.10.9:9002"  # 注意：这是应用服务器地址
  UserRpcURL: "192.168.10.9:9001"      # 注意：这是应用服务器地址
```

- ☐ DTM Server 地址正确（基础设施服务器）
- ☐ RPC URL 地址正确（应用服务器）

## 四、部署执行

### 4.1 首次部署

```bash
# 1. 初始化服务器
cd /opt/campushub/activity-platform
chmod +x deploy/server/init.sh
./deploy/server/init.sh

# 2. 检查配置
vim /opt/campushub/config/activity-rpc.yaml

# 3. 部署
./deploy/server/deploy.sh deploy
```

### 4.2 日常更新

```bash
# 拉取最新代码并重新部署
./deploy/server/deploy.sh deploy
```

## 五、部署后验证

### 5.1 服务状态

```bash
./deploy/server/deploy.sh status
```

- ☐ user-rpc 运行中
- ☐ user-api 运行中
- ☐ activity-rpc 运行中
- ☐ activity-api 运行中

### 5.2 Etcd 注册

```bash
# 在基础设施服务器上检查
etcdctl get --prefix /
```

- ☐ user.rpc 已注册
- ☐ activity.rpc 已注册

### 5.3 接口测试

```bash
# 健康检查
curl http://192.168.10.9:8001/health
curl http://192.168.10.9:8002/health

# 测试接口
curl http://192.168.10.9:8002/v1/activity/categories
```

- ☐ 健康检查返回 200
- ☐ 接口正常返回数据

## 六、常见问题

### Q1: 服务启动后立即退出

检查日志：
```bash
tail -100 /opt/campushub/logs/activity-rpc.log
```

常见原因：
- 配置文件语法错误
- 无法连接数据库
- 端口被占用

### Q2: RPC 调用超时

检查 Etcd 注册：
```bash
# 确认服务已注册
etcdctl get --prefix /user.rpc
etcdctl get --prefix /activity.rpc
```

### Q3: DTM 事务失败

检查 DTM Server 状态：
```bash
curl http://192.168.10.4:36789/api/ping
```

确认 RPC URL 配置正确（应该是应用服务器地址，不是基础设施服务器）。
