## 二、API 接口列表

### 2.1 群组管理 API

#### 2.1.1 查询群组信息

**接口**: `GET /api/groups/{group_id}`

**功能**: 获取指定群组的详细信息

**路径参数**:
- `group_id` (string, 必需): 群组ID

**请求示例**:
```http
GET /api/groups/group_abc123 HTTP/1.1
Host: localhost:8001
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

**成功响应** (200 OK):
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "group_id": "group_abc123",
    "activity_id": 1,
    "name": "周末登山群",
    "owner_id": 1,
    "member_count": 15,
    "status": 0,
    "created_at": "2026-01-28T10:00:00Z"
  }
}
```

**错误响应**:
```json
{
  "code": 1001,
  "message": "群组不存在",
  "data": null
}
```

---

#### 2.1.2 查询群成员列表

**接口**: `GET /api/groups/{group_id}/members`

**功能**: 分页获取群组的所有成员信息

**路径参数**:
- `group_id` (string, 必需): 群组ID

**查询参数**:
- `page` (integer, 可选): 页码，默认 1
- `page_size` (integer, 可选): 每页数量，默认 20，最大 100

**请求示例**:
```http
GET /api/groups/group_abc123/members?page=1&page_size=20 HTTP/1.1
Host: localhost:8001
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

**成功响应** (200 OK):
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "members": [
      {
        "user_id": 1,
        "username": "张三",
        "avatar": "https://cdn.example.com/avatar/789.jpg",
        "role": "owner",
        "joined_at": "2026-01-28T10:00:00Z"
      },
      {
        "user_id": 2,
        "username": "李四",
        "avatar": "https://cdn.example.com/avatar/999.jpg",
        "role": "member",
        "joined_at": "2026-01-28T11:00:00Z"
      }
    ],
    "total": 15,
    "page": 1,
    "page_size": 20
  }
}
```

---

#### 2.1.3 获取用户的群聊列表

**接口**: `GET /api/users/{user_id}/groups`

**功能**: 查询用户参与的所有群聊（分页）

**路径参数**:
- `user_id` (integer, 必需): 用户ID

**查询参数**:
- `page` (integer, 可选): 页码，默认 1
- `page_size` (integer, 可选): 每页数量，默认 20

**请求示例**:
```http
GET /api/users/999/groups?page=1&page_size=20 HTTP/1.1
Host: localhost:8001
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

**成功响应** (200 OK):
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "groups": [
      {
        "group_id": "group_abc123",
        "activity_id": 123456,
        "name": "周末登山群",
        "owner_id": 789,
        "member_count": 15,
        "status": 0,
        "role": "member",
        "joined_at": "2026-01-28T11:00:00Z",
        "last_message": "大家明天几点出发？",
        "last_message_at": "2026-01-29T08:30:00Z"
      }
    ],
    "total": 5,
    "page": 1,
    "page_size": 20
  }
}
```

---

### 2.2 消息管理 API

#### 2.2.1 查询消息历史

**接口**: `GET /api/messages`

**功能**: 查询指定群组的历史消息（分页）

**查询参数**:
- `group_id` (string, 必需): 群组ID
- `before_id` (string, 可选): 消息ID，查询此消息之前的历史消息
- `limit` (integer, 可选): 返回数量，默认 20，最大 100

**请求示例**:
```http
GET /api/messages?group_id=group_abc123&limit=20 HTTP/1.1
Host: localhost:8001
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

**成功响应** (200 OK):
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "messages": [
      {
        "message_id": "msg_xyz789",
        "group_id": "group_abc123",
        "sender_id": 999,
        "sender_name": "张三",
        "msg_type": 1,
        "content": "Hello",
        "created_at": "2026-01-28T10:00:00Z"
      }
    ],
    "has_more": true
  }
}
```

**字段说明**:
- `msg_type`: 消息类型（1=文本, 2=图片）
- `has_more`: 是否还有更多历史消息
- `before_id`: 用于分页，传入当前最早消息的 `message_id`

---

### 2.3 通知管理 API

#### 2.3.1 查询通知列表

**接口**: `GET /api/notifications`

**功能**: 查询用户的通知列表（分页）

**查询参数**:
- `user_id` (integer, 必需): 用户ID
- `page` (integer, 可选): 页码，默认 1
- `page_size` (integer, 可选): 每页数量，默认 20

**请求示例**:
```http
GET /api/notifications?user_id=999&page=1&page_size=20 HTTP/1.1
Host: localhost:8001
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

**成功响应** (200 OK):
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "total": 50,
    "unread_count": 5,
    "notifications": [
      {
        "notification_id": "notif_123",
        "type": "activity_joined",
        "title": "报名成功",
        "content": "您已成功报名《周末登山》",
        "is_read": false,
        "created_at": "2026-01-28T11:00:00Z"
      }
    ],
    "page": 1,
    "page_size": 20
  }
}
```

---

#### 2.3.2 获取未读数量

**接口**: `GET /api/notifications/unread-count`

**功能**: 获取用户的未读通知数量

**查询参数**:
- `user_id` (integer, 必需): 用户ID

**请求示例**:
```http
GET /api/notifications/unread-count?user_id=999 HTTP/1.1
Host: localhost:8001
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

**成功响应** (200 OK):
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "count": 5
  }
}
```

---

#### 2.3.3 标记已读

**接口**: `POST /api/notifications/read`

**功能**: 标记指定通知为已读

**请求体**:
```json
{
  "user_id": 999,
  "notification_ids": ["notif_123", "notif_456"]
}
```

**请求示例**:
```http
POST /api/notifications/read HTTP/1.1
Host: localhost:8001
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
Content-Type: application/json

{
  "user_id": 999,
  "notification_ids": ["notif_123", "notif_456"]
}
```

**成功响应** (200 OK):
```json
{
  "code": 0,
  "message": "标记成功",
  "data": {
    "updated_count": 2
  }
}
```

---

#### 2.3.4 标记全部已读

**接口**: `POST /api/notifications/read-all`

**功能**: 标记用户的所有通知为已读

**请求体**:
```json
{
  "user_id": 999
}
```

**请求示例**:
```http
POST /api/notifications/read-all HTTP/1.1
Host: localhost:8001
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
Content-Type: application/json

{
  "user_id": 999
}
```

**成功响应** (200 OK):
```json
{
  "code": 0,
  "message": "全部标记成功",
  "data": {
    "updated_count": 15
  }
}
```
