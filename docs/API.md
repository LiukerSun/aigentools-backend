# aigentools-backend API 接口文档

## 基础信息

- **Base URL**: `/api/v1`
- **认证方式**: Bearer Token (JWT)
- **Token 有效期**: 72小时

## 通用响应格式

```json
{
  "status": 200,
  "message": "Success message",
  "data": { ... }
}
```

**错误响应**:
```json
{
  "status": 400,
  "message": "Error message",
  "data": null
}
```

---

## 一、认证模块 `/auth`

### 1.1 用户注册

```
POST /auth/register
```

**请求体**:
```json
{
  "username": "string (必填)",
  "password": "string (必填)"
}
```

**响应** (201):
```json
{
  "status": 200,
  "message": "User registered successfully",
  "data": {
    "id": 1,
    "username": "user1",
    "role": "user",
    "token": "eyJhbGciOiJIUzI1NiIs..."
  }
}
```

**错误码**: 400 (参数错误), 409 (用户名已存在), 500 (服务器错误)

---

### 1.2 用户登录

```
POST /auth/login
```

**请求体**:
```json
{
  "username": "string (必填)",
  "password": "string (必填)"
}
```

**响应** (200):
```json
{
  "status": 200,
  "message": "Logged in successfully",
  "data": {
    "id": 1,
    "username": "user1",
    "role": "user",
    "token": "eyJhbGciOiJIUzI1NiIs..."
  }
}
```

**错误码**: 400 (参数错误), 401 (用户名或密码错误)

---

### 1.3 用户登出

```
POST /auth/logout
```

**Header**: `Authorization: Bearer <token>`

**响应** (200):
```json
{
  "status": 200,
  "message": "Logged out successfully",
  "data": null
}
```

---

### 1.4 获取当前用户信息

```
GET /auth/user
```

**Header**: `Authorization: Bearer <token>`

**响应** (200):
```json
{
  "status": 200,
  "message": "User information retrieved successfully",
  "data": {
    "id": 1,
    "username": "user1",
    "role": "user",
    "is_active": true,
    "activated_at": "2024-01-01T00:00:00Z",
    "deactivated_at": null,
    "creditLimit": 100.00,
    "total_consumed": 50.00,
    "credit": {
      "total": 200.00,
      "used": 50.00,
      "available": 150.00,
      "usagePercentage": 25.00
    },
    "token": "eyJhbGciOiJIUzI1NiIs..."
  }
}
```

---

## 二、AI模型管理 `/models`

> 所有接口需要认证

### 2.1 获取模型列表

```
GET /models
```

**Query 参数**:
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码，默认 1 |
| limit | int | 否 | 每页数量，默认 10 |
| name | string | 否 | 按名称过滤 |
| status | string | 否 | 按状态过滤: `open`, `closed`, `draft` |

> 普通用户只能看到 `status=open` 的模型

**响应** (200):
```json
{
  "status": 200,
  "message": "Success",
  "data": {
    "models": [
      {
        "id": 1,
        "name": "Model Name",
        "description": "Description",
        "status": "open",
        "url": "https://api.example.com",
        "price": 0.01,
        "parameters": { ... },
        "created_at": "2024-01-01T00:00:00Z",
        "updated_at": "2024-01-01T00:00:00Z"
      }
    ],
    "total": 100,
    "page": 1,
    "limit": 10
  }
}
```

---

### 2.2 获取模型名称列表（简化版）

```
GET /models/names
```

**响应** (200):
```json
{
  "status": 200,
  "message": "Success",
  "data": [
    {
      "id": 1,
      "name": "Model Name",
      "description": "Description",
      "status": "open",
      "price": 0.01,
      "url": "https://api.example.com",
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z"
    }
  ]
}
```

---

### 2.3 获取模型参数

```
GET /models/:id/parameters
```

**响应** (200):
```json
{
  "status": 200,
  "message": "Success",
  "data": {
    "request_header": [...],
    "request_body": [...],
    "response_parameters": [...]
  }
}
```

**错误码**: 400 (无效ID), 404 (模型不存在)

---

### 2.4 创建模型 (仅管理员)

```
POST /models/create
```

**请求体**:
```json
{
  "name": "string (必填)",
  "description": "string",
  "status": "open|closed|draft (必填)",
  "url": "string",
  "price": 0.01,
  "parameters": {
    "request_header": [],
    "request_body": [],
    "response_parameters": []
  }
}
```

**响应** (201): 返回创建的模型对象

**错误码**: 400 (参数错误), 403 (无权限)

---

### 2.5 更新模型 (仅管理员)

```
PUT /models/:id
```

**请求体** (所有字段可选):
```json
{
  "name": "string",
  "description": "string",
  "status": "open|closed|draft",
  "url": "string",
  "price": 0.01,
  "parameters": { ... }
}
```

**响应** (200): 返回更新后的模型对象

---

### 2.6 更新模型状态 (仅管理员)

```
PATCH /models/:id/status
```

**请求体**:
```json
{
  "status": "open|closed|draft (必填)"
}
```

---

## 三、任务管理 `/tasks`

> 所有接口需要认证

### 3.1 提交任务

```
POST /tasks
```

**请求体**:
```json
{
  "body": {
    "key": "value"
  },
  "user": {
    "creatorId": 1,
    "creatorName": "username"
  }
}
```

**响应** (200):
```json
{
  "status": 200,
  "message": "Task submitted successfully",
  "data": {
    "id": 1,
    "input_data": { ... },
    "creator_id": 1,
    "creator_name": "username",
    "status": 1,
    "result_url": "",
    "retry_count": 0,
    "max_retries": 3,
    "error_log": "",
    "remote_task_id": "",
    "cost": 0,
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
  }
}
```

**任务状态枚举**:
| 值 | 含义 |
|----|------|
| 1 | 待审核 (PendingAudit) |
| 2 | 待执行 (PendingExecution) |
| 3 | 处理中 (Processing) |
| 4 | 已完成 (Completed) |
| 5 | 失败 (Failed) |
| 6 | 已取消 (Cancelled) |

---

### 3.2 获取任务列表

```
GET /tasks
```

**Query 参数**:
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码，默认 1 |
| page_size | int | 否 | 每页数量，默认 10 |
| creator_id | int | 否 | 按创建者过滤 (仅管理员可用) |
| status | int | 否 | 按状态过滤 (1-6) |

> 普通用户只能看到自己的任务

**响应** (200):
```json
{
  "status": 200,
  "message": "Tasks retrieved successfully",
  "data": {
    "total": 100,
    "items": [ ... ]
  }
}
```

---

### 3.3 获取任务详情

```
GET /tasks/:id
```

**响应** (200): 返回任务对象

---

### 3.4 审批任务

```
PATCH /tasks/:id/approve
```

**响应** (200): 返回更新后的任务对象

---

### 3.5 更新任务

```
PUT /tasks/:id
```

**请求体**:
```json
{
  "body": {
    "key": "value"
  }
}
```

> 仅当任务未开始处理时可更新

---

### 3.6 重试任务

```
POST /tasks/:id/retry
```

> 仅对失败状态的任务有效

**错误码**: 400 (任务不是失败状态), 403 (无权限)

---

### 3.7 取消任务

```
POST /tasks/:id/cancel
```

> 仅对待处理状态的任务有效

---

## 四、支付模块 `/payment`

### 4.1 获取支付方式

```
GET /payment/methods
```

**Header**: `Authorization: Bearer <token>`

**响应** (200):
```json
{
  "status": 200,
  "message": "success",
  "data": [
    {
      "uuid": "abc-123",
      "type": "epay",
      "name": "支付宝"
    }
  ]
}
```

---

### 4.2 创建支付订单

```
POST /payment/create
```

**Header**: `Authorization: Bearer <token>`

**请求体**:
```json
{
  "amount": 100.00,
  "payment_method_uuid": "abc-123",
  "payment_channel": "alipay|wxpay",
  "return_url": "https://your-site.com/callback"
}
```

**响应** (200):
```json
{
  "status": 200,
  "message": "success",
  "data": {
    "jump_url": "https://payment.example.com/pay?...",
    "order_id": "order_123456"
  }
}
```

---

### 4.3 支付回调 (公开)

```
ANY /payment/notify/:uuid
```

> 此接口供支付平台回调使用，无需认证

---

## 五、文件上传 `/common/upload`

### 5.1 获取 OSS 上传凭证

```
GET /common/upload/token
```

**响应** (200):
```json
{
  "status": 200,
  "message": "OSS token retrieved successfully",
  "data": {
    "accessKeyId": "STS.xxxxx",
    "accessKeySecret": "xxxxx",
    "securityToken": "xxxxx",
    "expiration": "2024-01-01T01:00:00Z",
    "region": "oss-cn-beijing",
    "bucket": "your-bucket-name"
  }
}
```

---

## 六、AI 助手 `/ai-assistant`

> 需要认证

### 6.1 分析图片生成提示词

```
POST /ai-assistant/analyze
```

**请求体**:
```json
{
  "imageUrl": "https://example.com/image.jpg",
  "template": "nsfw|ecommerce"
}
```

**模板说明**:
- `nsfw` - 生成性感风格的 AI 图像生成提示词
- `ecommerce` - 生成电商带货视频的提示词

**响应** (200):
```json
{
  "status": 200,
  "message": "Success",
  "data": {
    "result": "生成的提示词内容..."
  }
}
```

---

## 七、管理员接口 `/admin`

> 所有接口需要管理员权限 (role = "admin")

### 7.1 用户管理

#### 获取用户列表

```
GET /admin/users
```

**Query 参数**:
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码，默认 1 |
| limit | int | 否 | 每页数量，默认 20 |
| is_active | bool | 否 | 按激活状态过滤 |
| created_after | string | 否 | 创建时间起始 (RFC3339) |
| created_before | string | 否 | 创建时间结束 (RFC3339) |

**响应** (200):
```json
{
  "status": 200,
  "message": "Users retrieved successfully",
  "data": {
    "users": [
      {
        "id": 1,
        "username": "user1",
        "role": "user",
        "is_active": true,
        "activated_at": "2024-01-01T00:00:00Z",
        "deactivated_at": null,
        "balance": 100.00,
        "creditLimit": 50.00,
        "created_at": "2024-01-01T00:00:00Z",
        "updated_at": "2024-01-01T00:00:00Z"
      }
    ],
    "total": 100,
    "page": 1,
    "limit": 20
  }
}
```

---

#### 更新用户

```
PATCH /admin/users/:id
```

**请求体** (所有字段可选):
```json
{
  "username": "string",
  "password": "string (最少6位)",
  "role": "admin|user",
  "is_active": true,
  "creditLimit": 100.00
}
```

---

#### 调整用户余额

```
POST /admin/users/:id/balance
```

**请求体**:
```json
{
  "amount": 100.00,
  "type": "credit|debit",
  "reason": "充值/扣款原因 (可选)"
}
```

- `credit` - 增加余额
- `debit` - 扣减余额

---

#### 删除用户

```
DELETE /admin/users/:id
```

> 不能删除自己

---

### 7.2 交易记录

#### 获取交易列表

```
GET /admin/transactions
```

**Query 参数**:
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码，默认 1 |
| limit | int | 否 | 每页数量，默认 20 |
| user_id | int | 否 | 按用户ID过滤 |
| type | string | 否 | 按类型过滤 |
| start_time | string | 否 | 开始时间 (RFC3339) |
| end_time | string | 否 | 结束时间 (RFC3339) |
| min_amount | float | 否 | 最小金额 |
| max_amount | float | 否 | 最大金额 |

**交易类型**:
- `admin_adjustment` - 管理员调整
- `system_auto` - 系统自动
- `user_consume` - 用户消费
- `user_refund` - 用户退款

**响应** (200):
```json
{
  "status": 200,
  "message": "Transactions retrieved successfully",
  "data": {
    "transactions": [
      {
        "id": 1,
        "created_at": "2024-01-01T00:00:00.000Z",
        "user_id": 1,
        "amount": 100.00,
        "balance_before": 0.00,
        "balance_after": 100.00,
        "reason": "充值",
        "operator": "admin",
        "type": "admin_adjustment",
        "ip_address": "127.0.0.1",
        "device_info": "Mozilla/5.0...",
        "hash": "abc123..."
      }
    ],
    "total": 100,
    "page": 1,
    "limit": 20
  }
}
```

---

#### 导出交易记录

```
GET /admin/transactions/export
```

**Query 参数**: 同上（除 page/limit 外）

**响应**: CSV 文件下载

---

### 7.3 支付配置

#### 获取支付配置列表

```
GET /admin/payment/config
```

**响应** (200):
```json
{
  "status": 200,
  "message": "success",
  "data": [
    {
      "id": 1,
      "uuid": "abc-123",
      "name": "支付宝",
      "payment_method": "epay",
      "config": {
        "pid": "xxx",
        "key": "xxx",
        "gateway": "https://..."
      },
      "enable": true,
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z"
    }
  ]
}
```

---

#### 创建支付配置

```
POST /admin/payment/config
```

**请求体**:
```json
{
  "name": "支付宝 (必填)",
  "payment_method": "epay (必填)",
  "config": {
    "pid": "xxx",
    "key": "xxx",
    "gateway": "https://..."
  },
  "enable": true
}
```

**响应** (200):
```json
{
  "status": 200,
  "message": "success",
  "data": {
    "id": 1,
    "uuid": "generated-uuid"
  }
}
```

---

#### 更新支付配置

```
PUT /admin/payment/config/:id
```

**请求体** (所有字段可选):
```json
{
  "name": "string",
  "config": { ... },
  "enable": true
}
```

---

#### 删除支付配置

```
DELETE /admin/payment/config/:id
```

---

## 八、HTTP 状态码参考

| 状态码 | 说明 |
|--------|------|
| 200 | 成功 |
| 201 | 创建成功 |
| 400 | 请求参数错误 |
| 401 | 未认证/Token无效 |
| 403 | 无权限 |
| 404 | 资源不存在 |
| 409 | 冲突（如用户名已存在、乐观锁冲突） |
| 500 | 服务器内部错误 |
