# goctl 代码生成详解

## 目录

1. [什么是 goctl](#1-什么是-goctl)
2. [安装与验证](#2-安装与验证)
3. [API 服务代码生成](#3-api-服务代码生成)
4. [RPC 服务代码生成](#4-rpc-服务代码生成)
5. [生成文件详解](#5-生成文件详解)
6. [实战演示](#6-实战演示)
7. [常见问题](#7-常见问题)

---

## 1. 什么是 goctl

### 1.1 简介

goctl（读作 go-control）是 go-zero 框架的代码生成工具。它的核心作用是：

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        goctl 的作用                                      │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│   .api 文件（接口定义）  ──goctl──►  完整的 HTTP 服务代码                  │
│                                                                         │
│   .proto 文件（协议定义）──goctl──►  完整的 gRPC 服务代码                  │
│                                                                         │
│   你只需要写：                                                           │
│     1. 接口定义（.api / .proto）                                        │
│     2. 业务逻辑（logic 层）                                              │
│     3. 配置文件（yaml）                                                  │
│                                                                         │
│   goctl 自动生成：                                                       │
│     - 路由注册                                                          │
│     - HTTP 处理器                                                       │
│     - 请求/响应结构体                                                    │
│     - 服务启动代码                                                       │
│     - gRPC 服务端/客户端代码                                             │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

### 1.2 为什么要用 goctl

| 优势 | 说明 |
|------|------|
| **减少重复代码** | 路由、处理器、类型定义都自动生成 |
| **规范代码结构** | 强制使用统一的目录结构和命名规范 |
| **减少人为错误** | 避免手写路由时的拼写错误 |
| **提高开发效率** | 专注于业务逻辑，不用写样板代码 |

---

## 2. 安装与验证

### 2.1 安装 goctl

```bash
# 方法1：使用 go install（推荐）
go install github.com/zeromicro/go-zero/tools/goctl@latest

# 方法2：从源码编译
git clone https://github.com/zeromicro/go-zero.git
cd go-zero/tools/goctl
go build -o goctl .
```

### 2.2 验证安装

```bash
# 检查版本
goctl --version

# 期望输出类似：
# goctl version 1.6.0 darwin/amd64
```

### 2.3 安装 protoc（RPC 服务需要）

```bash
# Windows：
# 1. 下载 https://github.com/protocolbuffers/protobuf/releases
# 2. 解压，将 bin 目录添加到 PATH

# Mac：
brew install protobuf

# 验证
protoc --version
```

### 2.4 安装 protoc 插件

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

---

## 3. API 服务代码生成

### 3.1 核心命令

```bash
goctl api go -api <api文件路径> -dir <输出目录> -style go_zero
```

| 参数 | 说明 | 示例 |
|------|------|------|
| `-api` | .api 文件路径 | `desc/user.api` |
| `-dir` | 输出目录 | `.`（当前目录） |
| `-style` | 命名风格 | `go_zero`（下划线风格） |

### 3.2 .api 文件语法

#### 3.2.1 基础结构

```go
// 语法版本声明（必须）
syntax = "v1"

// 服务信息（可选，但推荐）
info (
    title:   "用户服务 API"
    desc:    "用户注册、登录、个人信息管理"
    author:  "杨春路"
    version: "v1"
)

// 类型定义区
type (
    // 请求结构
    LoginReq {
        Phone    string `json:"phone"`
        Password string `json:"password"`
    }

    // 响应结构
    LoginResp {
        UserId      int64  `json:"userId"`
        AccessToken string `json:"accessToken"`
    }
)

// 服务定义区
@server (
    prefix: /api/v1/user
    group:  public
)
service user-api {
    @handler Login
    post /login (LoginReq) returns (LoginResp)
}
```

#### 3.2.2 类型定义语法

```go
type 类型名 {
    字段名 类型 `标签`
}

// 示例
type UserProfile {
    UserId    int64  `json:"userId"`                    // 必填字段
    Nickname  string `json:"nickname,optional"`         // 可选字段
    Avatar    string `json:"avatar,optional,omitempty"` // 可选且空值不序列化
    Age       int    `json:"age,default=18"`            // 带默认值
    Phone     string `json:"phone" validate:"required"` // 带验证
}
```

#### 3.2.3 @server 注解

```go
@server (
    prefix:     /api/v1/user           // URL 前缀
    group:      user                   // 分组（生成到对应目录）
    jwt:        Auth                   // 开启 JWT 验证
    middleware: LogMiddleware          // 自定义中间件
    timeout:    3s                     // 超时时间
)
```

#### 3.2.4 接口定义语法

```go
@server (
    prefix: /api/v1/user
    group:  public
)
service user-api {
    @doc "接口描述"
    @handler 处理器名称
    请求方法 路径 (请求类型) returns (响应类型)
}

// 完整示例
@server (
    prefix: /api/v1/user
    group:  public
)
service user-api {
    @doc "用户注册"
    @handler Register
    post /register (RegisterReq) returns (RegisterResp)

    @doc "用户登录"
    @handler Login
    post /login (LoginReq) returns (LoginResp)
}

// 需要 JWT 的接口
@server (
    prefix: /api/v1/user
    group:  user
    jwt:    Auth              // 👈 关键：启用 JWT 验证
)
service user-api {
    @doc "获取个人信息"
    @handler GetProfile
    get /profile returns (GetProfileResp)

    @doc "更新个人信息"
    @handler UpdateProfile
    put /profile (UpdateProfileReq) returns (UpdateProfileResp)
}
```

### 3.3 生成代码命令演示

```bash
# 进入 API 目录
cd app/user/api

# 执行代码生成
goctl api go -api desc/user.api -dir . -style go_zero

# 输出示例：
# Done.
```

### 3.4 生成的目录结构

```
app/user/api/
│
├── desc/
│   └── user.api              # 📝 你写的
│
├── etc/
│   └── user_api.yaml         # ⚙️ 生成的配置模板
│
├── internal/
│   ├── config/
│   │   └── config.go         # 🔒 生成的（不要改）
│   │
│   ├── handler/
│   │   ├── routes.go         # 🔒 生成的（不要改）
│   │   ├── public/
│   │   │   ├── register_handler.go  # 🔒 生成的
│   │   │   └── login_handler.go     # 🔒 生成的
│   │   └── user/
│   │       ├── get_profile_handler.go    # 🔒 生成的
│   │       └── update_profile_handler.go # 🔒 生成的
│   │
│   ├── logic/
│   │   ├── public/
│   │   │   ├── register_logic.go    # ✏️ 你要写的（业务逻辑）
│   │   │   └── login_logic.go       # ✏️ 你要写的
│   │   └── user/
│   │       ├── get_profile_logic.go     # ✏️ 你要写的
│   │       └── update_profile_logic.go  # ✏️ 你要写的
│   │
│   ├── svc/
│   │   └── service_context.go  # ✏️ 你要改的（添加依赖）
│   │
│   └── types/
│       └── types.go          # 🔒 生成的（不要改）
│
└── user.go                   # 🔒 生成的（入口文件）
```

### 3.5 文件分类说明

| 图标 | 含义 | 处理方式 |
|------|------|----------|
| 📝 | 你编写的源文件 | 修改后需重新生成 |
| 🔒 | 自动生成，不要修改 | 每次生成会覆盖 |
| ✏️ | 生成骨架，需要实现 | 你的主战场 |
| ⚙️ | 生成的配置模板 | 需要根据实际情况修改 |

---

## 4. RPC 服务代码生成

### 4.1 核心命令

```bash
goctl rpc protoc <proto文件> --go_out=. --go-grpc_out=. --zrpc_out=. -style go_zero
```

### 4.2 .proto 文件语法

```protobuf
// 语法版本
syntax = "proto3";

// 包名
package user;

// Go 包路径
option go_package = "./pb";

// 服务定义
service UserService {
    rpc Login(LoginReq) returns (LoginResp);
    rpc Register(RegisterReq) returns (RegisterResp);
    rpc GetUserInfo(GetUserInfoReq) returns (GetUserInfoResp);
}

// 消息定义
message LoginReq {
    string phone = 1;
    string password = 2;
}

message LoginResp {
    int64 user_id = 1;
    string phone = 2;
    string nickname = 3;
}

message RegisterReq {
    string phone = 1;
    string password = 2;
    string nickname = 3;
}

message RegisterResp {
    int64 user_id = 1;
}

message GetUserInfoReq {
    int64 user_id = 1;
}

message GetUserInfoResp {
    int64 user_id = 1;
    string phone = 2;
    string nickname = 3;
    string avatar = 4;
    int64 created_at = 5;
}
```

### 4.3 生成代码命令演示

```bash
# 进入 RPC 目录
cd app/user/rpc

# 执行代码生成
goctl rpc protoc user.proto --go_out=. --go-grpc_out=. --zrpc_out=. -style go_zero

# 输出示例：
# Done.
```

### 4.4 生成的目录结构

```
app/user/rpc/
│
├── user.proto                # 📝 你写的
│
├── etc/
│   └── user.yaml             # ⚙️ 生成的配置模板
│
├── internal/
│   ├── config/
│   │   └── config.go         # 🔒 生成的
│   │
│   ├── logic/
│   │   ├── login_logic.go           # ✏️ 你要写的
│   │   ├── register_logic.go        # ✏️ 你要写的
│   │   └── get_user_info_logic.go   # ✏️ 你要写的
│   │
│   ├── server/
│   │   └── user_service_server.go   # 🔒 生成的
│   │
│   └── svc/
│       └── service_context.go       # ✏️ 你要改的
│
├── pb/
│   ├── user.pb.go            # 🔒 protoc 生成的消息定义
│   └── user_grpc.pb.go       # 🔒 protoc 生成的 gRPC 代码
│
├── userclient/
│   └── user.go               # 🔒 生成的 RPC 客户端
│
└── user.go                   # 🔒 生成的入口文件
```

---

## 5. 生成文件详解

### 5.1 API 服务生成文件

#### 5.1.1 routes.go（路由注册）

```go
// Code generated by goctl. DO NOT EDIT.
package handler

func RegisterHandlers(server *rest.Server, serverCtx *svc.ServiceContext) {
    // 公开接口（无 JWT）
    server.AddRoutes(
        []rest.Route{
            {
                Method:  http.MethodPost,
                Path:    "/register",
                Handler: public.RegisterHandler(serverCtx),
            },
            {
                Method:  http.MethodPost,
                Path:    "/login",
                Handler: public.LoginHandler(serverCtx),
            },
        },
        rest.WithPrefix("/api/v1/user"),
    )

    // 需要 JWT 的接口
    server.AddRoutes(
        []rest.Route{
            {
                Method:  http.MethodGet,
                Path:    "/profile",
                Handler: user.GetProfileHandler(serverCtx),
            },
        },
        rest.WithPrefix("/api/v1/user"),
        rest.WithJwt(serverCtx.Config.Auth.AccessSecret),  // 👈 自动添加 JWT 验证
    )
}
```

#### 5.1.2 handler（HTTP 处理器）

```go
// Code generated by goctl. DO NOT EDIT.
package public

func LoginHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // 1. 解析请求参数
        var req types.LoginReq
        if err := httpx.Parse(r, &req); err != nil {
            httpx.ErrorCtx(r.Context(), w, err)
            return
        }

        // 2. 调用业务逻辑
        l := logic.NewLoginLogic(r.Context(), svcCtx)
        resp, err := l.Login(&req)

        // 3. 返回响应
        if err != nil {
            httpx.ErrorCtx(r.Context(), w, err)
        } else {
            httpx.OkJsonCtx(r.Context(), w, resp)
        }
    }
}
```

#### 5.1.3 logic（业务逻辑骨架）

```go
package public

type LoginLogic struct {
    logx.Logger
    ctx    context.Context
    svcCtx *svc.ServiceContext
}

func NewLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginLogic {
    return &LoginLogic{
        Logger: logx.WithContext(ctx),
        ctx:    ctx,
        svcCtx: svcCtx,
    }
}

// TODO(human): 这里需要你实现业务逻辑
func (l *LoginLogic) Login(req *types.LoginReq) (resp *types.LoginResp, err error) {
    // 生成的只有这一行
    return
}
```

#### 5.1.4 types.go（请求/响应结构体）

```go
// Code generated by goctl. DO NOT EDIT.
package types

type LoginReq struct {
    Phone    string `json:"phone"`
    Password string `json:"password"`
}

type LoginResp struct {
    UserId      int64  `json:"userId"`
    AccessToken string `json:"accessToken"`
    ExpiresAt   int64  `json:"expiresAt"`
}
```

#### 5.1.5 service_context.go（服务上下文）

```go
package svc

type ServiceContext struct {
    Config config.Config
    // TODO(human): 添加你需要的依赖
    // UserRpc userclient.User
    // Redis   *redis.Redis
}

func NewServiceContext(c config.Config) *ServiceContext {
    return &ServiceContext{
        Config: c,
        // TODO(human): 初始化依赖
    }
}
```

### 5.2 RPC 服务生成文件

#### 5.2.1 logic（业务逻辑骨架）

```go
package logic

type LoginLogic struct {
    ctx    context.Context
    svcCtx *svc.ServiceContext
    logx.Logger
}

func NewLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginLogic {
    return &LoginLogic{
        ctx:    ctx,
        svcCtx: svcCtx,
        Logger: logx.WithContext(ctx),
    }
}

// TODO(human): 实现登录逻辑
func (l *LoginLogic) Login(in *pb.LoginReq) (*pb.LoginResp, error) {
    // 生成的只有这一行
    return &pb.LoginResp{}, nil
}
```

#### 5.2.2 userclient（RPC 客户端）

```go
// Code generated by goctl. DO NOT EDIT.
package userclient

type (
    LoginReq       = pb.LoginReq
    LoginResp      = pb.LoginResp
    // ... 其他类型别名
)

type User interface {
    Login(ctx context.Context, in *LoginReq, opts ...grpc.CallOption) (*LoginResp, error)
    Register(ctx context.Context, in *RegisterReq, opts ...grpc.CallOption) (*RegisterResp, error)
    GetUserInfo(ctx context.Context, in *GetUserInfoReq, opts ...grpc.CallOption) (*GetUserInfoResp, error)
}

// 创建 RPC 客户端
func NewUser(cli zrpc.Client) User {
    return &defaultUser{cli: cli}
}
```

---

## 6. 实战演示

### 6.1 从零开始创建 user-api

#### Step 1: 创建目录结构

```bash
mkdir -p app/user/api/desc
mkdir -p app/user/api/etc
```

#### Step 2: 编写 .api 文件

```bash
# 创建 app/user/api/desc/user.api
```

```go
syntax = "v1"

info (
    title:   "用户服务 API"
    author:  "杨春路"
    version: "v1"
)

// ==================== 类型定义 ====================

type (
    // 登录
    LoginReq {
        Phone    string `json:"phone"`
        Password string `json:"password"`
    }
    LoginResp {
        UserId      int64  `json:"userId"`
        AccessToken string `json:"accessToken"`
        ExpiresAt   int64  `json:"expiresAt"`
    }

    // 注册
    RegisterReq {
        Phone    string `json:"phone"`
        Password string `json:"password"`
        Nickname string `json:"nickname,optional"`
    }
    RegisterResp {
        UserId int64 `json:"userId"`
    }

    // 用户信息
    UserProfile {
        UserId    int64  `json:"userId"`
        Phone     string `json:"phone"`
        Nickname  string `json:"nickname"`
        Avatar    string `json:"avatar"`
    }
    GetProfileResp {
        Profile UserProfile `json:"profile"`
    }
)

// ==================== 公开接口 ====================

@server (
    prefix: /api/v1/user
    group:  public
)
service user-api {
    @doc "用户注册"
    @handler Register
    post /register (RegisterReq) returns (RegisterResp)

    @doc "用户登录"
    @handler Login
    post /login (LoginReq) returns (LoginResp)
}

// ==================== 需要登录的接口 ====================

@server (
    prefix: /api/v1/user
    group:  user
    jwt:    Auth
)
service user-api {
    @doc "获取个人信息"
    @handler GetProfile
    get /profile returns (GetProfileResp)
}
```

#### Step 3: 生成代码

```bash
cd app/user/api
goctl api go -api desc/user.api -dir . -style go_zero
```

#### Step 4: 查看生成结果

```bash
# Windows
dir /s /b

# Mac/Linux
find . -type f -name "*.go"
```

预期输出：
```
./internal/config/config.go
./internal/handler/routes.go
./internal/handler/public/login_handler.go
./internal/handler/public/register_handler.go
./internal/handler/user/get_profile_handler.go
./internal/logic/public/login_logic.go
./internal/logic/public/register_logic.go
./internal/logic/user/get_profile_logic.go
./internal/svc/service_context.go
./internal/types/types.go
./user.go
```

#### Step 5: 配置文件

创建 `app/user/api/etc/user-api.yaml`：

```yaml
Name: user-api
Host: 0.0.0.0
Port: 8001

# JWT 配置
Auth:
  AccessSecret: "your-secret-key-32-chars-long-xxx"
  AccessExpire: 7200

# RPC 配置（user-rpc 实现后启用）
# UserRpc:
#   Etcd:
#     Hosts:
#       - 127.0.0.1:2379
#     Key: user.rpc
```

#### Step 6: 实现业务逻辑

编辑 `internal/logic/public/login_logic.go`：

```go
func (l *LoginLogic) Login(req *types.LoginReq) (resp *types.LoginResp, err error) {
    // TODO(杨春路): 实现登录逻辑

    // 1. 调用 user-rpc 验证密码
    // user, err := l.svcCtx.UserRpc.Login(l.ctx, &userpb.LoginReq{
    //     Phone:    req.Phone,
    //     Password: req.Password,
    // })
    // if err != nil {
    //     return nil, errors.New("用户名或密码错误")
    // }

    // 2. 签发 JWT Token
    // now := time.Now().Unix()
    // token, _ := l.generateToken(user.UserId, now)

    // 3. 返回响应
    return &types.LoginResp{
        UserId:      1,  // 临时硬编码，用于测试
        AccessToken: "test-token",
        ExpiresAt:   time.Now().Unix() + 7200,
    }, nil
}
```

#### Step 7: 启动服务

```bash
cd app/user/api
go run user.go -f etc/user-api.yaml
```

#### Step 8: 测试接口

```bash
# 测试登录
curl -X POST http://localhost:8001/api/v1/user/login \
  -H "Content-Type: application/json" \
  -d '{"phone":"13800138000","password":"123456"}'

# 期望响应
# {"userId":1,"accessToken":"test-token","expiresAt":1706000000}
```

### 6.2 完整演示脚本

```bash
#!/bin/bash
# demo.sh - 完整演示脚本

echo "=== Step 1: 检查 goctl 版本 ==="
goctl --version

echo ""
echo "=== Step 2: 进入 user-api 目录 ==="
cd app/user/api

echo ""
echo "=== Step 3: 查看 .api 文件 ==="
cat desc/user.api

echo ""
echo "=== Step 4: 生成代码 ==="
goctl api go -api desc/user.api -dir . -style go_zero

echo ""
echo "=== Step 5: 查看生成的文件 ==="
find . -name "*.go" -type f

echo ""
echo "=== Step 6: 启动服务 ==="
echo "执行: go run user.go -f etc/user-api.yaml"

echo ""
echo "=== Step 7: 测试接口 ==="
echo "登录: curl -X POST http://localhost:8001/api/v1/user/login -H 'Content-Type: application/json' -d '{\"phone\":\"13800138000\",\"password\":\"123456\"}'"
```

---

## 7. 常见问题

### Q1: 修改了 .api 文件后怎么办？

```bash
# 重新生成代码
goctl api go -api desc/user.api -dir . -style go_zero

# 注意：
# - logic 文件不会被覆盖（你的业务代码安全）
# - handler、types、routes 会被覆盖
```

### Q2: 生成的代码报错 "undefined: xxx"

```bash
# 通常是缺少依赖，执行
go mod tidy
```

### Q3: 如何添加新接口？

```go
// 1. 在 .api 文件中添加新接口
@server (
    prefix: /api/v1/user
    group:  user
    jwt:    Auth
)
service user-api {
    // 新增接口
    @handler UpdateProfile
    put /profile (UpdateProfileReq) returns (UpdateProfileResp)
}

// 2. 添加对应的类型定义
type UpdateProfileReq {
    Nickname string `json:"nickname,optional"`
    Avatar   string `json:"avatar,optional"`
}

type UpdateProfileResp {
    Success bool `json:"success"`
}
```

```bash
# 3. 重新生成
goctl api go -api desc/user.api -dir . -style go_zero

# 4. 实现新的 logic 文件
```

### Q4: 如何自定义错误响应格式？

```go
// 在 main 函数中设置错误处理
func main() {
    // ...

    // 自定义错误处理
    httpx.SetErrorHandler(func(err error) (int, interface{}) {
        return http.StatusOK, map[string]interface{}{
            "code":    500,
            "message": err.Error(),
        }
    })

    // ...
}
```

### Q5: style 参数有哪些选项？

| 选项 | 说明 | 文件名示例 |
|------|------|-----------|
| `go_zero` | 下划线风格（推荐） | `login_handler.go` |
| `goZero` | 驼峰风格 | `loginHandler.go` |
| `gozero` | 小写风格 | `loginhandler.go` |

### Q6: 如何查看更多 goctl 命令？

```bash
# 查看所有命令
goctl --help

# 查看 api 子命令
goctl api --help

# 查看 rpc 子命令
goctl rpc --help

# 查看 api go 子命令详情
goctl api go --help
```

---

## 附录：命令速查表

### API 服务

```bash
# 生成 API 代码
goctl api go -api desc/xxx.api -dir . -style go_zero

# 格式化 .api 文件
goctl api format --dir ./desc

# 验证 .api 文件语法
goctl api validate --api desc/xxx.api

# 生成 API 文档（Markdown）
goctl api doc --dir ./desc --o ./docs
```

### RPC 服务

```bash
# 生成 RPC 代码
goctl rpc protoc xxx.proto --go_out=. --go-grpc_out=. --zrpc_out=. -style go_zero

# 仅生成 proto 文件的 Go 代码
protoc --go_out=. --go-grpc_out=. xxx.proto
```

### 其他常用命令

```bash
# 生成 model 代码（从数据库）
goctl model mysql datasource -url "user:pass@tcp(127.0.0.1:3306)/dbname" -table "user" -dir ./model

# 生成 Dockerfile
goctl docker -go user.go

# 生成 Kubernetes 部署文件
goctl kube deploy -name user-api -namespace default -image user-api:latest -o k8s.yaml
```
