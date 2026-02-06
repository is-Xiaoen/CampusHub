# CampusHub 核心开发上下文手册

## 1. 项目架构概览

**CampusHub** 是一个基于 **go-zero v1.6.0** 的微服务架构系统，严格遵循 **Clean Architecture** 原则。项目根目录为 `d:\workspace_go\CampusHub\CampusHub`。

### 1.1 服务划分
系统核心业务拆分为以下独立微服务，通过 **gRPC** 进行内部通信，对外通过 **API Gateway** (go-zero API) 暴露 HTTP 接口：
*   **user**: 用户身份认证、个人信息管理、信用分体系、学生资质认证。
*   **activity**: 活动全生命周期管理（发布、审核、报名、核销）。
*   **chat**: 即时通讯服务，支持单聊、群聊及 WebSocket 消息推送。

### 1.2 目录结构规范
所有服务均遵循统一的 go-zero 标准目录结构：
*   `api/` & `rpc/`: 分别定义 HTTP 和 gRPC 接口层。
*   `internal/logic/`: **业务逻辑核心**。每个 API/RPC 接口对应一个独立的 Logic 文件（如 `apply_verify_logic.go`）。
*   `internal/svc/`: **依赖注入容器** (`ServiceContext`)，管理 Config、DB、Redis、RPC Clients 等资源。
*   `internal/handler/`: HTTP 请求解析与响应封装层。
*   `internal/model/`: **数据持久层**，严格使用 **GORM v1.25.5** 定义数据模型与操作。
*   `cache/`: **缓存服务层**。封装 Redis 操作，提供强类型的业务缓存接口（如 `VerifyCache`），隔离 Redis 细节，支持 Cache-Aside 模式。
*   `common/`: 全局公共库，包含 `errorx` (错误处理)、`response` (响应封装)、`utils` (工具函数)。

---

## 2. 核心开发规范与模式

### 2.1 错误处理与响应机制 (Standardized Error & Response)
系统实现了统一的错误处理和响应拦截机制，杜绝随意返回 HTTP 状态码或非结构化数据。

*   **错误定义 (`common/errorx`)**:
    *   核心结构体 `BizError`: 包含 `Code` (业务错误码) 和 `Message` (用户提示信息)。
    *   **严禁**直接返回 Go 原生 `error`。所有 Logic 层错误必须经过 `errorx.FromError(err)` 或 `errorx.New(code)` 包装。
    *   `FromError` 函数会自动识别 gRPC 的 `status.Status` 错误并还原为业务错误码，实现跨服务错误透传。

*   **响应格式 (`common/response`)**:
    *   所有 API 接口（无论成功或失败）统一返回 JSON 格式：
        ```json
        {
            "code": 0,          // 0 表示成功，非 0 为错误码
            "message": "success", // 错误时的提示信息
            "data": {}          // 成功时的业务数据
        }
        ```
    *   **Handler 实现**: 使用 `response.Success(w, data)` 或 `response.Fail(w, err)` 进行响应。`Fail` 方法会自动提取 `BizError` 中的 Code 和 Message。

### 2.2 数据持久层 (GORM Implementation)
*   **约束**: 严禁使用 `sqlx` 或原生 SQL 拼接，必须使用 **GORM**。
*   **Model 定义 (`internal/model`)**:
    *   结构体必须包含 `gorm` tag，如 `gorm:"primaryKey;autoIncrement;column:user_id"`。
    *   每个 Model 文件需提供接口定义（如 `IUserModel`）及其实现（`UserModel`），便于单元测试 Mock。
    *   **依赖注入**: Model 实例在 `ServiceContext` 中初始化，并注入到 Logic 层。
    *   **CRUD 规范**: 数据库操作必须传递 `context.Context` 以支持链路追踪和超时控制。

### 2.3 缓存层策略 (Redis Cache-Aside)
*   **客户端**: 使用 `go-redis/v8`。
*   **模式**: **Cache-Aside** (旁路缓存)。
    *   **读**: 先读 Cache -> Miss 则读 DB -> 回写 Cache。
    *   **写**: 先写 DB -> 成功后**删除** Cache (而非更新)。
*   **防护**: 必须处理缓存穿透（空值缓存）、击穿（SingleFlight/分布式锁）和雪崩（随机过期时间）。

### 2.4 业务逻辑编排 (Logic Layer)
*   **职责单一**: Logic 层只负责业务编排，不直接操作 HTTP Request/Response。
*   **上下文获取**: 使用 `common/ctxdata` 包中的 `GetUserIDFromCtx(ctx)` 从 JWT 中提取当前用户 ID。
*   **RPC 调用**:
    *   API Logic 不直接操作数据库，而是通过 `l.svcCtx.XxxRpc` 调用后端 RPC 服务。
    *   必须处理 RPC 调用失败的情况，并将其转换为 `errorx`。
*   **数据转换**:
    *   **API <-> RPC**: 需要显式转换 DTO (Data Transfer Object) 和 PB (Protobuf) 对象。
    *   **RPC <-> Model**: 需要显式转换 PB 对象和 GORM Model。
    *   **禁止**将 GORM Model 直接作为 API 或 RPC 的返回值。

---

## 3. 全局注意事项
1.  **环境感知**: 敏感配置（如 DB 密码）严禁硬编码，必须通过 `.yaml` 配置文件管理，生产环境使用 K8s ConfigMap/Secret。
2.  **链路追踪**: 所有跨层调用（API->RPC->DB/Redis）必须透传 `context.Context`，以确保 TraceID 贯穿全链路。
3.  **代码生成**: 修改 `.api` 或 `.proto` 文件后，必须执行 `goctl` 命令重新生成代码，严禁手动修改生成的 `*.pb.go` 文件。