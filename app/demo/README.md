# Demo RPC 服务 - 开发规范示例

这是 **CampusHub 校园活动平台** 的标准 go-zero RPC 微服务示例，展示项目的目录结构、代码规范和最佳实践。

**所有同学请以此为模板开发自己负责的模块。**

---

## 快速启动

### 前置条件

| 依赖 | 版本 | 说明 |
|------|------|------|
| Go | 1.21+ | `go version` 查看 |
| protoc | 3.x+ | Proto 编译器 |
| 远程服务 | - | MySQL、Redis、Etcd 已配置好 |

### 启动服务

```bash
# 1. 进入项目根目录
cd F:\桌面\activity

# 2. 下载依赖
go mod tidy

# 3. 启动 Demo RPC 服务
cd app/demo/rpc
go run demo.go -f etc/demo.yaml
```

看到以下输出说明成功：
```
{"@timestamp":"...","content":"数据库连接成功","level":"info"}
Starting Demo RPC server at 0.0.0.0:9100...
```

### 验证服务

```bash
# 1. 查看 Etcd 注册（在服务器上执行）
docker exec -it etcd etcdctl get --prefix /demo.rpc

# 2. 使用 grpcurl 测试（需要先安装）
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest

# 列出服务方法
grpcurl -plaintext localhost:9100 list

# 调用 GetItem
grpcurl -plaintext -d '{"id": 1}' localhost:9100 demo.DemoService/GetItem

# 调用 ListItems
grpcurl -plaintext -d '{"page": 1, "page_size": 10}' localhost:9100 demo.DemoService/ListItems
```

---

## 当前环境配置

| 服务 | 地址 | 说明 |
|------|------|------|
| MySQL | 192.168.10.4:3308 | 用户 root，密码 123456 |
| Redis | 192.168.10.4:6379 | 密码 123456 |
| Etcd | 192.168.10.4:2379 | 服务注册发现 |

| 数据库 | 说明 |
|--------|------|
| campushub_user | 用户服务数据库 |
| campushub_main | 活动服务数据库（Demo 使用此库） |
| campushub_chat | 聊天服务数据库 |

---

## 目录结构

```
demo/rpc/
├── etc/
│   └── demo.yaml              # 配置文件（数据库、Etcd 等）
├── internal/
│   ├── config/
│   │   └── config.go          # 配置结构体映射
│   ├── logic/                 # 业务逻辑层（核心代码）
│   │   ├── getitemlogic.go    # 查询单个
│   │   ├── listitemslogic.go  # 查询列表
│   │   └── createitemlogic.go # 创建
│   ├── model/                 # 数据模型层（数据库操作）
│   │   └── item.go            # GORM Model + CRUD 方法
│   ├── server/                # gRPC 服务实现
│   │   └── demoserviceserver.go
│   └── svc/
│       └── servicecontext.go  # 服务上下文（依赖注入）
├── pb/                        # Proto 生成的代码（勿手动修改）
│   ├── demo.pb.go
│   └── demo_grpc.pb.go
├── demo.go                    # 入口文件
└── demo.proto                 # Proto 接口定义
```

---

## 核心概念

### 1. 服务上下文 (ServiceContext)

**位置**：`internal/svc/servicecontext.go`

**作用**：集中管理所有依赖，实现依赖注入

```go
type ServiceContext struct {
    Config    config.Config
    DB        *gorm.DB        // 数据库连接
    ItemModel *model.ItemModel // 数据模型
}
```

**为什么这样设计**：
- 所有依赖在启动时初始化，运行时直接使用
- 方便测试时 Mock 依赖
- 避免全局变量，代码更清晰

### 2. Logic 层

**位置**：`internal/logic/`

**作用**：处理业务逻辑，是代码的核心

```go
func (l *GetItemLogic) GetItem(in *pb.GetItemRequest) (*pb.GetItemResponse, error) {
    // 1. 参数校验
    if in.Id <= 0 {
        return nil, errorx.New(errorx.CodeInvalidParams, "ID 不能为空")
    }

    // 2. 调用 Model 层查询数据
    item, err := l.svcCtx.ItemModel.FindOne(l.ctx, in.Id)
    if err != nil {
        return nil, err
    }

    // 3. 转换并返回
    return &pb.GetItemResponse{
        Item: convertToProto(item),
    }, nil
}
```

### 3. Model 层

**位置**：`internal/model/`

**作用**：封装数据库操作

```go
type ItemModel struct {
    db *gorm.DB
}

func (m *ItemModel) FindOne(ctx context.Context, id int64) (*Item, error) {
    var item Item
    err := m.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&item).Error
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, errorx.New(errorx.CodeNotFound, "数据不存在")
        }
        return nil, errorx.Wrap(err, errorx.CodeDBError, "查询失败")
    }
    return &item, nil
}
```

### 4. 服务注册原理

服务启动时自动向 Etcd 注册，配置如下：

```yaml
# etc/demo.yaml
Etcd:
  Hosts:
    - 192.168.10.4:2379
  Key: demo.rpc
```

go-zero 框架会：
1. 启动时向 Etcd 写入 Key: `demo.rpc/{实例ID}` Value: `{IP}:{Port}`
2. 每 3 秒发送心跳续约
3. 停止时自动注销

---

## 开发自己的服务

### 步骤 1：定义 Proto 接口

```protobuf
// xxx.proto
syntax = "proto3";

package xxx;
option go_package = "./pb";

service XxxService {
    rpc GetXxx(GetXxxRequest) returns (GetXxxResponse);
    rpc ListXxx(ListXxxRequest) returns (ListXxxResponse);
    rpc CreateXxx(CreateXxxRequest) returns (CreateXxxResponse);
}

message GetXxxRequest {
    int64 id = 1;
}

message GetXxxResponse {
    XxxInfo info = 1;
}
// ...
```

### 步骤 2：生成 pb 代码

```bash
cd app/xxx/rpc
protoc --go_out=. --go-grpc_out=. xxx.proto
```

### 步骤 3：实现各层代码

1. **config.go** - 添加配置字段
2. **servicecontext.go** - 初始化依赖
3. **model/xxx.go** - 定义数据模型和 CRUD
4. **logic/xxxlogic.go** - 实现业务逻辑
5. **server/xxxserver.go** - 实现 gRPC 接口

### 步骤 4：配置文件

```yaml
# etc/xxx.yaml
Name: xxx.rpc
ListenOn: 0.0.0.0:900x

Etcd:
  Hosts:
    - 192.168.10.4:2379
  Key: xxx.rpc

MySQL:
  DataSource: root:123456@tcp(192.168.10.4:3308)/campushub_xxx?charset=utf8mb4&parseTime=true&loc=Local

Redis:
  Host: 192.168.10.4:6379
  Pass: "123456"
```

---

## 服务间调用

如果你的服务需要调用其他 RPC 服务：

### 1. 配置文件添加 RPC 客户端

```yaml
# etc/xxx.yaml
UserRpc:
  Etcd:
    Hosts:
      - 192.168.10.4:2379
    Key: user.rpc
  NonBlock: true
  Timeout: 3000
```

### 2. Config 添加字段

```go
type Config struct {
    zrpc.RpcServerConf
    UserRpc zrpc.RpcClientConf
}
```

### 3. ServiceContext 初始化客户端

```go
type ServiceContext struct {
    Config  config.Config
    UserRpc userpb.UserClient
}

func NewServiceContext(c config.Config) *ServiceContext {
    return &ServiceContext{
        Config:  c,
        UserRpc: userpb.NewUserClient(zrpc.MustNewClient(c.UserRpc).Conn()),
    }
}
```

### 4. Logic 中调用

```go
func (l *XxxLogic) Xxx(in *pb.XxxRequest) (*pb.XxxResponse, error) {
    // 调用 User 服务
    userResp, err := l.svcCtx.UserRpc.GetUser(l.ctx, &userpb.GetUserRequest{
        UserId: in.UserId,
    })
    if err != nil {
        return nil, errorx.Wrap(err, errorx.CodeRPCError, "调用用户服务失败")
    }
    // ...
}
```

---

## 常见问题

### Q1: 启动报错 "连接数据库失败"

检查：
1. 配置文件中的 MySQL 地址和端口是否正确（192.168.10.4:3308）
2. 用户名密码是否正确（root/123456）
3. 数据库是否存在（campushub_main）

### Q2: 启动报错 "Etcd 连接失败"

检查：
1. Etcd 地址是否正确（192.168.10.4:2379）
2. 服务器上 Etcd 容器是否运行：`docker ps | grep etcd`

### Q3: grpcurl 调用失败

确保：
1. 服务已启动且监听 9100 端口
2. 开发模式下启用了 gRPC 反射
3. 防火墙未阻止端口

### Q4: 为什么没有 handler？

RPC 服务和 HTTP API 服务不同：
- **HTTP API**：需要 handler 解析 HTTP 请求
- **gRPC RPC**：框架自动反序列化 protobuf，直接到 server 层

---

## 代码规范

### 命名规范

| 类型 | 规范 | 示例 |
|------|------|------|
| 文件名 | 小写，下划线分隔 | `get_item_logic.go` |
| 结构体 | 大驼峰 | `ItemModel` |
| 方法名 | 大驼峰 | `FindOne` |
| 变量名 | 小驼峰 | `itemList` |
| 常量 | 大驼峰或全大写 | `StatusActive` |

### 错误处理

```go
// 使用 errorx 包装错误
if err != nil {
    return nil, errorx.Wrap(err, errorx.CodeDBError, "查询失败")
}

// 业务错误
if item == nil {
    return nil, errorx.New(errorx.CodeNotFound, "数据不存在")
}
```

### 日志规范

```go
import "github.com/zeromicro/go-zero/core/logx"

// 普通日志
logx.Infof("处理请求: id=%d", id)

// 错误日志
logx.Errorf("数据库查询失败: %v", err)

// 带上下文的日志（推荐）
logx.WithContext(ctx).Infof("用户 %d 查询活动", userId)
```

---

## 面试亮点

学习本项目后，你可以在面试中谈到：

1. **微服务架构**：服务拆分、服务发现、负载均衡
2. **go-zero 框架**：配置管理、中间件、拦截器
3. **gRPC 通信**：Proto 定义、序列化、流式传输
4. **Etcd 服务注册**：Key-Value 存储、Lease 租约、Watch 机制
5. **分层架构**：Config → Svc → Logic → Model
6. **错误处理**：统一错误码、错误包装、链路追踪

---

## 相关文档

- [go-zero 官方文档](https://go-zero.dev/)
- [gRPC 官方文档](https://grpc.io/docs/)
- [GORM 文档](https://gorm.io/zh_CN/docs/)
