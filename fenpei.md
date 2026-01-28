# 校园活动平台 - 模块设计与人员分配
---

## 一、服务架构总览

```
┌─────────────────────────────────────────────────────────────┐
│                        客户端 (Web/App)                       │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│                     Gateway API (BFF层)                       │
│              统一鉴权、限流、路由、聚合                          │
└─────────────────────────────────────────────────────────────┘
           ↓                    ↓                    ↓
    ┌──────────┐         ┌──────────┐         ┌──────────┐
    │User RPC  │         │Activity  │         │ IM RPC   │
    │用户服务   │         │ RPC      │         │消息服务   │
    └──────────┘         └──────────┘         └──────────┘
           ↓                    ↓                    ↓
    ┌──────────┐         ┌──────────┐         ┌──────────┐
    │MySQL     │         │MySQL     │         │MySQL     │
    │activity_ │         │activity_ │         │activity_ │
    │user      │         │main      │         │chat      │
    └──────────┘         └──────────┘         └──────────┘
                              ↓
              ┌───────────────────────────────┐
              │         公共组件               │
              │ Redis | Kafka | ES | OSS     │
              └───────────────────────────────┘
```

---

## 二、模块详细设计


### 1. 用户服务 (user-service)

| 功能模块           | 技术实现             | 面试考点                |
| ------------------ | -------------------- | ----------------------- |
| 注册登录           | JWT双Token + bcrypt  | Token刷新机制、单点登录 |
| 验证码（激光验证） | 图形验证码 + 短信    | 60s间隔+IP限流          |
| 学生认证           | 阿里云OCR + 人工审核 | 异步处理、状态机        |
| 个人资料           | CRUD + OSS上传       | 手机号脱敏              |
| 兴趣标签           | 用户-标签多对多      | 用于推荐匹配            |
| 信用分             | 初始100分+扣分机制   | 防爽约设计              |



### 2. 活动服务 (activity-service)

| 功能模块 | 技术实现               | 面试考点             |
| -------- | ---------------------- | -------------------- |
| 活动管理 | CRUD + 状态机          | 状态流转、权限控制   |
| 活动列表 | 分页 + Redis缓存       | Cache-Aside模式      |
| 活动搜索 | ES + IK分词            | 倒排索引原理         |
| 活动推荐 | 标签匹配 + 热度        | 算法设计             |
| 活动报名 | **Redis预扣 + MQ异步** | **秒杀架构（重点）** |
| 签到核销 | TOTP动态码             | 时间同步攻击防护     |

### 3. IM服务 (im-service)

| 功能模块 | 技术实现          | 面试考点           |
| -------- | ----------------- | ------------------ |
| 长连接   | WebSocket + 心跳  | 连接保活、断线重连 |
| 系统通知 | 推送机制          | 消息分发策略       |
| 消息可靠 | ACK + 重传 + 去重 | 消息不丢失设计     |
| 离线消息 | MySQL存储 + 拉取  | 读扩散vs写扩散     |

### 4. 公共组件 (common)

| 组件     | 技术实现                         | 面试考点           |
| -------- | -------------------------------- | ------------------ |
| 缓存     | Redis Cache-Aside                | 穿透/击穿/雪崩     |
| 分布式锁 | Redis SetNX + 续期   **redlock** | 锁过期问题         |
| 消息队列 | Kafka/RabbitMQ                   | 幂等消费、死信队列 |
| 限流     | 令牌桶算法                       | 漏桶vs令牌桶       |
| 日志追踪 | 结构化日志 + TraceID             | 分布式追踪         |

分布式事务
---

## 三、人员分配（每人详细任务+面试亮点）

---

### 👤 学生认证 + 信用分 + 数据统计+ 后台（goadmin）

#### 负责模块
```
app/user/rpc/
├── internal/logic/
│   ├── studentverifylogic.go    # 学生认证
│   ├── creditscore/             # 信用分模块
│   │   ├── getcreditlogic.go
│   │   ├── deductcreditlogic.go
│   │   └── creditrulelogic.go
│   └── statistics/              # 数据统计
│       ├── userstatslogic.go
│       └── activitystatslogic.go
```

#### 详细任务清单

**1. 学生认证模块**
- [ ] 设计认证表：`student_verifications`（user_id, id_card_front, id_card_back, student_card, status, reject_reason）
- [ ] 对接阿里云OCR API识别学生证
- [ ] 实现认证状态机：`未认证 → 审核中 → 已认证/已拒绝`
- [ ] 人工审核后台接口
- [ ] 认证过期（每学年重新认证）

**2. 信用分模块**
- [ ] 设计信用分表：`user_credits`（user_id, score, level）
- [ ] 设计信用记录表：`credit_logs`（user_id, change_type, change_value, reason）
- [ ] 信用规则引擎：
  - 注册初始100分
  - 爽约扣10分
  - 正常签到+2分
  - 被举报核实扣20分
- [ ] 信用等级：优秀(90+)/良好(70-89)/一般(60-69)/限制(<60)
- [ ] 低信用限制报名逻辑

**3. 数据统计模块**
- [ ] 用户统计：日活、新增、留存率
- [ ] 活动统计：发布数、报名数、签到率
- [ ] 定时任务生成统计报表

#### 🎯 面试亮点（可深挖）

| 亮点 | 面试问法 | 回答要点 |
|------|---------|---------|
| **OCR异步处理** | "认证很慢怎么办？" | MQ异步、轮询/回调通知 |
| **状态机设计** | "状态怎么管理？" | 有限状态机、状态流转图 |
| **信用分规则引擎** | "规则怎么配置？" | 策略模式、规则可配置化 |
| **防刷设计** | "恶意认证怎么防？" | 次数限制、人工审核 |

#### 预计技术栈
- 阿里云OSS（上传证件照）
- 阿里云OCR API
- 定时任务（go-zero scheduler 或 cron）
- 规则引擎设计

---

### 👤 B - 注册登录 + 验证码 + 个人资料 + 图片上传

#### 负责模块
```
app/user/rpc/
├── internal/logic/
│   ├── registerlogic.go         # 注册
│   ├── loginlogic.go            # 密码登录
│   ├── loginbysmslogic.go       # 短信登录
│   ├── sendsmscodelogic.go      # 发送验证码
│   ├── refreshtokenlogic.go     # Token刷新
│   ├── getuserinfologic.go      # 获取用户信息
│   ├── updateuserinfologic.go   # 更新用户信息
│   └── changepasswordlogic.go   # 修改密码
common/utils/
├── jwt/jwt.go                   # JWT工具
├── sms/sms.go                   # 短信工具
├── captcha/captcha.go           # 图形验证码
└── oss/oss.go                   # OSS上传
```

#### 详细任务清单

**1. 注册登录模块**
- [ ] 双Token实现：AccessToken(15min) + RefreshToken(7天)
- [ ] bcrypt密码加密（cost=10）
- [ ] 单设备登录：新登录踢出旧设备（Redis存储Token）
- [ ] 登录失败次数限制：5次锁定30分钟
- [ ] 登录日志记录（IP、设备、时间）

**2. 验证码模块**
- [ ] 图形验证码：base64图片，支持点击刷新
- [ ] 短信验证码：对接阿里云/腾讯云短信API
- [ ] 验证码限流：
  - 同一手机号60秒间隔
  - 同一IP每小时最多20次
  - 同一手机号每天最多10次
- [ ] 验证码存储：Redis 5分钟过期

**3. 个人资料模块**
- [ ] 基础信息CRUD
- [ ] 头像上传（对接OSS，限制大小2MB，格式jpg/png）
- [ ] 手机号脱敏展示（138****8888）
- [ ] 用户信息缓存（Cache-Aside）

**4. 图片上传公共模块**
- [ ] 前端直传OSS（生成临时签名URL）
- [ ] 图片压缩（可选）
- [ ] 图片格式、大小校验

#### 🎯 面试亮点（可深挖）

| 亮点 | 面试问法 | 回答要点 |
|------|---------|---------|
| **JWT双Token** | "Token过期怎么办？" | AccessToken短期+RefreshToken长期，无感刷新 |
| **单设备登录** | "多端登录怎么处理？" | Redis存token，新登录覆盖，旧token失效 |
| **验证码限流** | "防刷怎么做？" | 多维度限流：手机号+IP+每日总量 |
| **密码安全** | "为什么用bcrypt？" | 自带盐值、计算慢防暴力破解、可调cost |
| **OSS直传** | "大文件上传卡顿？" | 前端直传OSS，后端只签名，不经过服务器 |

#### 预计技术栈
- JWT (github.com/golang-jwt/jwt/v5)
- bcrypt (golang.org/x/crypto/bcrypt)
- 阿里云短信 SDK
- 阿里云OSS SDK
- 图形验证码库

---

### 👤 C - 活动CRUD + 状态机 + 列表/详情 + 缓存+ 搜索 敏感词

#### 负责模块
```
app/activity/rpc/
├── internal/logic/
│   ├── createactivitylogic.go   # 创建活动
│   ├── getactivitylogic.go      # 活动详情
│   ├── updateactivitylogic.go   # 更新活动
│   ├── deleteactivitylogic.go   # 删除活动
│   ├── listactivitieslogic.go   # 活动列表
│   ├── listcategorieslogic.go   # 分类列表
│   └── statemachine/            # 状态机
│       └── activitystate.go
common/
├── cache/                       # 缓存封装
│   ├── activity_cache.go
│   └── cache_aside.go
```

#### 详细任务清单

**1. 活动CRUD**
- [ ] 创建活动（草稿保存、直接发布）
- [ ] 活动详情查询（关联查询分类、组织者信息）
- [ ] 更新活动（仅组织者/管理员可操作）
- [ ] 删除活动（软删除，有报名不可删）
- [ ] 分类管理（CRUD，缓存分类列表）

**2. 活动状态机**
- [ ] 状态定义：
  ```
  草稿(0) → 待审核(1) → 已发布(2) → 进行中(3) → 已结束(4)
                ↓
            已拒绝(5)
  任意状态 → 已取消(6)
  ```
- [ ] 状态流转校验（不能跳跃状态）
- [ ] 状态变更日志
- [ ] 定时任务：自动更新活动状态（报名截止→进行中→已结束）

**3. 活动列表**
- [ ] 分页查询（Cursor分页 vs Offset分页）
- [ ] 多条件筛选（分类、状态、时间范围）
- [ ] 排序（时间、热度、距离）
- [ ] 列表接口性能优化

**4. 缓存策略**
- [ ] 活动详情缓存（Cache-Aside模式）
- [ ] 活动列表缓存（按条件Hash）
- [ ] 缓存穿透防护（布隆过滤器）
- [ ] 缓存击穿防护（singleflight）
- [ ] 缓存雪崩防护（随机过期时间）
- [ ] 缓存更新（更新/删除时失效缓存）

#### 🎯 面试亮点（可深挖）

| 亮点 | 面试问法 | 回答要点 |
|------|---------|---------|
| **状态机设计** | "状态怎么管理的？" | 状态模式、流转图、校验机制 |
| **缓存三连** | "缓存穿透/击穿/雪崩？" | 布隆过滤器、singleflight、随机过期 |
| **Cache-Aside** | "缓存一致性？" | 先更新DB再删缓存，延迟双删 |
| **分页优化** | "深度分页问题？" | Cursor游标分页、避免offset |
| **乐观锁** | "并发更新冲突？" | version字段、CAS更新 |

#### 预计技术栈
- Redis缓存
- 布隆过滤器 (github.com/bits-and-blooms/bloom)
- singleflight (golang.org/x/sync/singleflight)
- 定时任务

---

### 👤 D - 活动报名(高并发) + 签到核销 + 推荐算法

#### 负责模块
```
app/activity/rpc/
├── internal/logic/
│   ├── registeractivitylogic.go   # 报名（核心高并发）
│   ├── cancelregistrationlogic.go # 取消报名
│   ├── getticketlogic.go          # 获取票据
│   ├── verifyticketlogic.go       # 核销票据
│   └── recommend/                  # 推荐模块
│       ├── recommendlogic.go
│       └── algorithm.go
app/activity/mq/                    # 消息队列消费者
├── consumer/
│   └── registrationconsumer.go
common/utils/
├── ticket/ticket.go               # TOTP票据
└── lock/redis_lock.go             # 分布式锁（已有）
```

#### 详细任务清单

**1. 活动报名（高并发核心）**
- [ ] 报名流程设计：
  ```
  1. 参数校验（活动存在、状态正确、未报名）
  2. 用户资格校验（信用分、学生认证）
  3. Redis库存预检（DECR原子操作）
  4. 发送MQ消息
  5. 返回排队中
  --- MQ消费者 ---
  6. 创建报名记录
  7. 数据库扣减库存（乐观锁）
  8. 生成票据
  9. 发送通知
  ```
- [ ] Redis库存预热（活动开始前Load到Redis）
- [ ] 防止重复报名（Redis Set存储已报名用户）
- [ ] 防止超卖（Redis DECR + 数据库乐观锁双保险）
- [ ] 报名失败回滚（MQ消费失败退还Redis库存）

**2. 取消报名**
- [ ] 取消时间限制（活动开始前24小时可取消）
- [ ] 释放库存（Redis INCR + 数据库更新）
- [ ] 信用分扣减（临近取消扣分多）

**3. 签到核销**
- [ ] TOTP动态码生成（30秒刷新，6位数字）
- [ ] 二维码生成（含票据码+用户ID+时间戳）
- [ ] 核销接口（组织者扫码）
- [ ] 防重复核销（幂等性设计）
- [ ] 核销时间窗口（活动开始前30分钟到结束后1小时）

**4. 推荐算法**
- [ ] 基于标签匹配（用户兴趣标签 ∩ 活动标签）
- [ ] 基于热度排序（报名人数/浏览量）
- [ ] 基于地理位置（可选，需要位置服务）
- [ ] 推荐结果缓存

#### 🎯 面试亮点（可深挖）⭐重点

| 亮点 | 面试问法 | 回答要点 |
|------|---------|---------|
| **秒杀架构** | "高并发报名怎么设计？" | Redis预扣+MQ异步+DB乐观锁 |
| **防超卖** | "怎么保证不超卖？" | Redis DECR原子操作，返回值<0说明库存不足 |
| **库存一致性** | "Redis和DB库存不一致？" | 最终一致性，MQ失败重试+回滚 |
| **幂等设计** | "重复请求怎么处理？" | 唯一索引、Redis Set去重 |
| **TOTP原理** | "动态码怎么实现？" | 时间戳+密钥→HMAC→6位数字 |
| **MQ削峰** | "为什么用MQ？" | 削峰填谷、解耦、异步 |

#### 预计技术栈
- Redis（库存、分布式锁、Set去重）
- Kafka/RabbitMQ（异步下单）
- TOTP (github.com/pquerna/otp)
- 二维码生成 (github.com/skip2/go-qrcode)

---

### 👤 E - WebSocket + 系统通知 + 消息中间件

#### 负责模块
```
app/chat/
├── ws/                           # WebSocket服务
│   ├── hub.go                    # 连接管理（已有）
│   ├── client.go                 # 客户端连接
│   ├── handler.go                # 消息处理
│   └── heartbeat.go              # 心跳保活
├── rpc/                          # RPC服务
│   └── internal/logic/
│       ├── sendnotifylogic.go    # 发送通知
│       ├── getunreadlogic.go     # 获取未读
│       └── markreadlogic.go      # 标记已读
├── mq/                           # MQ消费者
│   └── consumer/
│       └── notifyconsumer.go     # 通知消费
common/
├── mq/                           # MQ封装
│   ├── producer.go
│   └── consumer.go
```

#### 详细任务清单

**1. WebSocket连接管理**
- [ ] 连接建立（鉴权、注册到Hub）
- [ ] 心跳保活（30秒ping/pong）
- [ ] 断线检测（超时踢出）
- [ ] 断线重连（客户端指数退避）
- [ ] 连接信息存储（userId→connId映射存Redis）

**2. 消息推送**
- [ ] 单点推送（指定用户推送）
- [ ] 广播推送（全体在线用户）
- [ ] 在线判断（先查Redis连接状态）
- [ ] 离线消息存储（MySQL，上线后拉取）

**3. 系统通知类型**
- [ ] 报名成功通知
- [ ] 活动提醒通知（活动开始前1小时）
- [ ] 审核结果通知
- [ ] 系统公告

**4. 消息中间件封装**
- [ ] Producer封装（统一发送接口）
- [ ] Consumer封装（统一消费接口）
- [ ] 消息可靠性：
  - 生产者确认（ACK）
  - 消费者手动ACK
  - 失败重试（指数退避）
  - 死信队列（3次失败后）
- [ ] 消息幂等（Redis存消息ID）

**5. 消息可靠性设计**
- [ ] 消息ACK机制（客户端收到后回复ACK）
- [ ] 超时重传（5秒未收到ACK则重发）
- [ ] 消息去重（消息ID + Redis）
- [ ] 消息有序（按seq排序展示）

#### 🎯 面试亮点（可深挖）

| 亮点 | 面试问法 | 回答要点 |
|------|---------|---------|
| **WebSocket原理** | "和HTTP区别？" | 全双工、持久连接、握手升级 |
| **心跳保活** | "为什么需要心跳？" | NAT超时、检测死连接、保活 |
| **消息可靠** | "消息丢失怎么办？" | ACK机制、超时重传、持久化 |
| **消息去重** | "重复推送怎么处理？" | 消息ID、Redis Set、客户端去重 |
| **MQ选型** | "Kafka vs RabbitMQ？" | Kafka高吞吐、RabbitMQ功能丰富 |
| **死信队列** | "消费失败怎么办？" | 重试3次、进入死信、人工处理 |

#### 预计技术栈
- WebSocket (gorilla/websocket 或 go-zero内置)
- Kafka/RabbitMQ
- Redis（在线状态、消息去重）
- 定时任务（活动提醒）

---

## 四、项目骨架优化建议

### 当前骨架分析

```
activity/
├── app/
│   ├── gateway/api/     ✅ BFF层，好
│   ├── user/rpc/        ✅ 用户RPC
│   ├── activity/rpc/    ⚠️ 只有proto，需补充
│   └── chat/            ⚠️ 只有ws/hub.go
├── common/              ✅ 公共组件
└── deploy/              ✅ 部署配置
```

### 建议优化的目录结构

```
activity-platform/
├── app/                          # 应用服务
│   ├── gateway/                  # API网关（BFF）
│   │   └── api/
│   │       ├── internal/
│   │       │   ├── config/
│   │       │   ├── handler/
│   │       │   │   ├── user/
│   │       │   │   ├── activity/
│   │       │   │   └── chat/
│   │       │   ├── logic/
│   │       │   ├── middleware/
│   │       │   │   ├── auth.go        # JWT鉴权
│   │       │   │   ├── ratelimit.go   # 【新增】限流
│   │       │   │   ├── cors.go
│   │       │   │   └── requestid.go
│   │       │   └── svc/
│   │       └── etc/
│   │
│   ├── user/                     # 用户服务
│   │   └── rpc/
│   │       ├── internal/
│   │       │   ├── config/
│   │       │   ├── logic/
│   │       │   │   ├── auth/          # 【新增】认证相关
│   │       │   │   ├── profile/       # 【新增】资料相关
│   │       │   │   └── credit/        # 【新增】信用分
│   │       │   ├── model/             # 数据模型
│   │       │   ├── server/
│   │       │   └── svc/
│   │       ├── pb/
│   │       └── etc/
│   │
│   ├── activity/                 # 活动服务
│   │   ├── rpc/                  # RPC服务
│   │   │   ├── internal/
│   │   │   │   ├── logic/
│   │   │   │   │   ├── activity/      # 活动CRUD
│   │   │   │   │   ├── registration/  # 报名相关
│   │   │   │   │   └── recommend/     # 推荐相关
│   │   │   │   └── model/
│   │   │   └── pb/
│   │   └── mq/                   # 【新增】MQ消费者
│   │       └── consumer/
│   │           └── registration_consumer.go
│   │
│   ├── chat/                     # IM服务
│   │   ├── rpc/                  # RPC服务
│   │   ├── ws/                   # WebSocket服务
│   │   └── mq/                   # 【新增】MQ消费者
│   │
│   └── job/                      # 【新增】定时任务服务
│       ├── internal/
│       │   └── logic/
│       │       ├── activity_status.go  # 活动状态更新
│       │       ├── activity_remind.go  # 活动提醒
│       │       └── statistics.go       # 数据统计
│       └── etc/
│
├── common/                       # 公共组件
│   ├── constants/               ✅ 常量定义
│   ├── errorx/                  ✅ 错误处理
│   ├── response/                ✅ 响应封装
│   ├── ctxdata/                 ✅ 上下文数据
│   ├── utils/
│   │   ├── jwt/                 ✅ JWT
│   │   ├── encrypt/             ✅ 加密
│   │   ├── lock/                ✅ 分布式锁
│   │   ├── ticket/              ✅ 票据
│   │   ├── validate/            ✅ 验证
│   │   ├── sms/                 # 【新增】短信
│   │   ├── oss/                 # 【新增】OSS上传
│   │   ├── captcha/             # 【新增】验证码
│   │   └── snowflake/           # 【新增】雪花ID
│   ├── mq/                      # 【新增】消息队列
│   │   ├── producer.go
│   │   └── consumer.go
│   └── cache/                   # 【新增】缓存封装
│       └── cache.go
│
├── deploy/                      ✅ 部署配置
│   ├── docker/
│   ├── k8s/                     # 【新增】K8s配置
│   └── sql/
│
├── docs/                        # 【新增】文档
│   ├── api/                     # API文档
│   └── design/                  # 设计文档
│
├── scripts/                     # 【新增】脚本
│   ├── init.sh
│   └── gen_proto.sh
│
├── go.mod
├── go.sum
├── Makefile                     ✅
└── README.md
```

### 需要补充的关键文件

#### 1. 限流中间件 `common/middleware/ratelimit.go`
```go
// 令牌桶限流器
// 面试亮点：令牌桶 vs 漏桶算法
```

#### 2. MQ封装 `common/mq/producer.go`
```go
// 消息生产者封装
// 面试亮点：生产者确认、消息持久化
```

#### 3. 缓存封装 `common/cache/cache.go`
```go
// Cache-Aside模式封装
// 面试亮点：穿透/击穿/雪崩防护
```

#### 4. 雪花ID `common/utils/snowflake/snowflake.go`
```go
// 分布式唯一ID生成
// 面试亮点：时钟回拨问题
```

---

## 五、数据库表补充建议

### user库需要新增

```sql
-- 学生认证表（A负责）
CREATE TABLE `student_verifications` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `user_id` BIGINT UNSIGNED NOT NULL,
    `real_name` VARCHAR(50) NOT NULL COMMENT '真实姓名',
    `id_number` VARCHAR(18) NOT NULL COMMENT '身份证号（加密存储）',
    `student_card_url` VARCHAR(255) NOT NULL COMMENT '学生证照片',
    `school` VARCHAR(100) NOT NULL COMMENT '学校',
    `status` TINYINT NOT NULL DEFAULT 0 COMMENT '0待审核 1通过 2拒绝',
    `reject_reason` VARCHAR(200) DEFAULT '' COMMENT '拒绝原因',
    `verified_at` DATETIME DEFAULT NULL,
    `expire_at` DATETIME DEFAULT NULL COMMENT '认证过期时间',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_user_id` (`user_id`)
);

-- 信用分表（A负责）
CREATE TABLE `user_credits` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `user_id` BIGINT UNSIGNED NOT NULL,
    `score` INT NOT NULL DEFAULT 100 COMMENT '当前分数',
    `level` TINYINT NOT NULL DEFAULT 1 COMMENT '等级',
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_user_id` (`user_id`)
);

-- 信用记录表
CREATE TABLE `credit_logs` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `user_id` BIGINT UNSIGNED NOT NULL,
    `change_type` VARCHAR(50) NOT NULL COMMENT '变更类型',
    `change_value` INT NOT NULL COMMENT '变更值（正负）',
    `before_score` INT NOT NULL,
    `after_score` INT NOT NULL,
    `reason` VARCHAR(200) NOT NULL,
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `idx_user_id` (`user_id`)
);

-- 用户标签表（B负责）
CREATE TABLE `user_tags` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `user_id` BIGINT UNSIGNED NOT NULL,
    `tag_id` BIGINT UNSIGNED NOT NULL,
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_user_tag` (`user_id`, `tag_id`)
);

-- 标签字典表
CREATE TABLE `tags` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `name` VARCHAR(20) NOT NULL,
    `type` TINYINT NOT NULL COMMENT '1用户标签 2活动标签',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`)
);
```

### chat库需要新增

```sql
-- 消息表（E负责）
CREATE TABLE `messages` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `msg_id` VARCHAR(50) NOT NULL COMMENT '消息唯一ID',
    `user_id` BIGINT UNSIGNED NOT NULL COMMENT '接收用户',
    `type` TINYINT NOT NULL COMMENT '1系统通知 2活动提醒',
    `title` VARCHAR(100) NOT NULL,
    `content` TEXT NOT NULL,
    `extra` JSON DEFAULT NULL COMMENT '扩展数据',
    `is_read` TINYINT NOT NULL DEFAULT 0,
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_msg_id` (`msg_id`),
    KEY `idx_user_read` (`user_id`, `is_read`)
);
```

---

## 六、开发顺序建议

### 第一阶段：基础框架（Week 1）
1. **全员**：熟悉go-zero框架，搭建本地开发环境
2. **B**：完成注册登录基础功能（JWT、bcrypt）
3. **C**：完成活动CRUD基础功能

### 第二阶段：核心功能（Week 2-3）
1. **B**：验证码、个人资料、图片上传
2. **C**：活动列表、详情、缓存策略
3. **D**：报名功能（先不用MQ，直接同步）
4. **E**：WebSocket连接、心跳

### 第三阶段：进阶功能（Week 4-5）
1. **A**：学生认证、信用分
2. **D**：高并发报名（Redis+MQ）、签到核销
3. **E**：消息推送、MQ封装
4. **C**：搜索功能（对接ES）

### 第四阶段：完善优化（Week 6）
1. **全员**：联调测试
2. **A**：数据统计
3. **D**：推荐算法
4. **全员**：性能优化、文档完善

---

## 七、Git分支规范

```
main                 # 主分支，保护分支
├── develop          # 开发分支
│   ├── feature/user-auth        # A: 学生认证
│   ├── feature/user-credit      # A: 信用分
│   ├── feature/user-login       # B: 登录注册
│   ├── feature/user-profile     # B: 个人资料
│   ├── feature/activity-crud    # C: 活动CRUD
│   ├── feature/activity-cache   # C: 缓存
│   ├── feature/registration     # D: 报名
│   ├── feature/checkin          # D: 签到
│   ├── feature/websocket        # E: WebSocket
│   └── feature/notification     # E: 通知
```

---

## 八、面试重点总结

| 人员 | 核心亮点 | 必问问题 |
|------|---------|---------|
| A | OCR异步、信用分规则引擎 | "状态机怎么设计？" |
| B | JWT双Token、多维度限流 | "Token过期怎么处理？" |
| C | 缓存三连、状态机 | "缓存穿透怎么解决？" |
| D | **秒杀架构、MQ削峰** | **"高并发报名怎么设计？"** |
| E | 消息可靠性、心跳保活 | "消息丢失怎么处理？" |

---
