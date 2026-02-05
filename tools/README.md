# 测试 Token 生成工具

## 问题背景
在开发阶段，登录接口还未完成，但需要测试需要认证的接口。

## 解决方案
使用 `generate_test_token.go` 工具生成测试用的 JWT Token。

## 使用方法

### 1. 生成默认测试 Token
```bash
cd tools
go run generate_test_token.go
```

### 2. 修改测试用户信息
编辑 `generate_test_token.go` 文件中的用户信息：
```go
// 测试用户信息
userID := int64(1)      // 修改为你需要的用户ID
phone := "13800138000"  // 修改为你需要的手机号
```

### 3. 在 Apifox 中使用

#### 方法一：在请求头中添加
1. 打开 Apifox
2. 选择需要测试的接口
3. 在 "Headers" 标签页添加：
   - Key: `Authorization`
   - Value: `Bearer <生成的token>`

#### 方法二：使用环境变量
1. 在 Apifox 中创建环境变量 `token`
2. 将生成的 token 值粘贴进去
3. 在请求头中使用：`Authorization: Bearer {{token}}`

### 4. 在 curl 中使用
```bash
curl -X GET http://localhost:8001/api/user/info \
  -H "Authorization: Bearer <生成的token>"
```

## Token 信息
- **密钥**: 与配置文件 `user-api.yaml` 中的 `Auth.AccessSecret` 一致
- **有效期**: 2小时（7200秒）
- **包含字段**:
  - `userId`: 用户ID
  - `phone`: 手机号
  - `exp`: 过期时间
  - `iat`: 签发时间
  - `nbf`: 生效时间

## 注意事项
1. 此工具仅用于开发测试，不要在生产环境使用
2. Token 有效期为 2 小时，过期后需要重新生成
3. 所有 API 服务（user-api、activity-api、chat-api）使用相同的密钥，因此生成的 token 在所有服务中都有效

## 验证 Token
你可以在 [jwt.io](https://jwt.io) 上验证和解析生成的 token：
1. 将 token 粘贴到左侧
2. 在右侧 "VERIFY SIGNATURE" 部分输入密钥：`k9#8G7&6F5%4D3$2S1@0P9*8O7!6N5^4M3+2L1=0`
3. 查看解析后的 payload 内容
