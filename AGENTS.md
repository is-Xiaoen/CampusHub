# CampusHub Codex Rules (Condensed)

## 工作路径（必须）
- 所有代码必须位于 `d:\workspace_go\CampusHub\CampusHub`

## 技术栈（关键约束）
- go-zero `v1.6.0`
- **GORM `v1.25.5`（必须，禁止 sqlx）**
- MySQL `8.0`
- Redis `go-redis/v8`
- Etcd `v3.5.10`
- gRPC `v1.64.0`
- protobuf `v1.33.0`
- JWT `v4.5.0`（双 Token）
- OpenTelemetry `v1.19.0` + Jaeger
- google/uuid `v1.6.0`

## 项目结构（简版）
- `app/<service>/api`：HTTP API
- `app/<service>/rpc`：gRPC
- `app/<service>/rpc/internal/model`：GORM Model
- `common/`：公共组件（constants/errorx/response/utils）
- `deploy/`、`docs/`

## API 规范
- 使用 `.api`（go-zero DSL），配合 `@handler`、`@doc`、`@server`
- 生成命令：`goctl api go -api service.api -dir . -style go_zero`

## RPC 规范
- `.proto` 使用 `proto3`，设置 `option go_package`
- 字段名使用 `snake_case`
- 生成命令：  
  `goctl rpc protoc service.proto --go_out=./pb --go-grpc_out=./pb --zrpc_out=. -style go_zero`

## Logic 规范
- 业务逻辑仅放在 `logic/`
- 每个 handler 对应一个 logic 文件
- 通过 `ServiceContext` 注入依赖
- 显式错误处理并包装；单函数不超 80 行
- 导出函数必须有注释

## 错误与响应
- 错误码放在 `common/errorx/`
- 统一响应格式：
```json
{"code":0,"message":"success","data":{}}
```

## 配置
- `etc/` 目录下 YAML
- 区分 dev/test/prod
- 敏感信息不入库，使用 `.yaml.example`

## GORM Model（强制）
- Model 必须位于 `internal/model/`
- 通过 `ServiceContext` 初始化 Model
- **Proto 与 Model 必须显式转换**

## 中间件
- 放在 `middleware/`
- 常用：Auth、CORS、RateLimit、RequestID

## 命名规范（核心）
- 服务名：小写+连字符（`user-service`）
- Go 文件：小写+下划线（`get_user_logic.go`）
- Handler：`XxxHandler`

## Redis 规范
- Cache-Aside
- 防穿透/击穿/雪崩（空值缓存、singleflight/锁、随机过期）

## JWT 双 Token
- AccessToken：15 分钟
- RefreshToken：7 天

## 可观测性 & 测试
- logx、metrics、OpenTelemetry
- 表驱动测试，`go test -v ./...`

