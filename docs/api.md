# API 接口文档

本文档以当前 Go 服务实际注册的路由为准。

## 通用约定

- API 基础路径：`/api/v1`
- 健康检查路径：`/health`
- 数据格式：除文件上传和重定向接口外，均使用 `application/json`
- 登录态：OAuth 回调成功后由服务端设置 `can_i_session` HttpOnly Cookie；浏览器请求需携带 Cookie
- 角色：`owner`（最高管理员）和 `user`（普通用户）
- 普通受保护接口返回 `401 unauthorized`；禁用用户返回 `403 disabled`
- Owner 接口权限不足时返回 `403 owner_required`

成功响应通常使用：

```json
{"data": {}}
```

错误响应统一为：

```json
{
  "error": {
    "code": "error_code",
    "message": "error message"
  }
}
```

## 接口总览

| 方法 | 路径 | 权限 | 说明 |
| --- | --- | --- | --- |
| GET | `/health/live` | 公开 | 进程存活检查 |
| GET | `/health/ready` | 公开 | 数据库和 Redis 就绪检查 |
| GET | `/api/v1/auth/login` | 公开 | 发起 SSO 登录 |
| GET | `/api/v1/auth/callback` | 公开 | SSO OAuth 回调 |
| GET | `/api/v1/auth/me` | 登录 | 获取当前用户 |
| POST | `/api/v1/auth/logout` | 登录可选 | 撤销当前应用会话 |
| GET | `/api/v1/tasks` | 登录 | 获取任务列表 |
| POST | `/api/v1/tasks` | 登录 | 创建任务 |
| PATCH | `/api/v1/tasks/{id}` | 登录 | 更新任务完成状态 |
| DELETE | `/api/v1/tasks/{id}` | 登录 | 删除任务 |
| GET | `/api/v1/plans` | 登录 | 获取自己的计划书 |
| POST | `/api/v1/plans` | 登录 | 上传计划书 |
| GET | `/api/v1/plans/{id}` | 登录 | 获取自己的单个计划书 |
| POST | `/api/v1/plans/{id}/analyze` | 登录 | 创建异步分析任务 |
| GET | `/api/v1/plans/{id}/analysis` | 登录 | 获取计划书的最新分析任务 |
| POST | `/api/v1/plans/{id}/analysis/retry` | 登录 | 重新入队失败的分析任务 |
| GET | `/api/v1/admin/users` | Owner | 获取全部用户 |
| PATCH | `/api/v1/admin/users/{id}` | Owner | 启用或禁用普通用户 |
| GET | `/api/v1/admin/plans` | Owner | 获取全部计划书 |
| GET | `/api/v1/admin/analysis` | Owner | 分页获取全部分析任务 |
| GET | `/api/v1/admin/settings/ai` | Owner | 获取 AI 服务配置 |
| PATCH | `/api/v1/admin/settings/ai` | Owner | 更新 AI 服务配置 |

## 健康检查

### `GET /health/live`

成功响应 `200`：

```json
{"status":"ok"}
```

### `GET /health/ready`

数据库和 Redis 都可用时返回 `200`，否则返回 `503`：

```json
{
  "status": "ok",
  "checks": {"database":"ok","redis":"ok"}
}
```

## SSO 与会话

### `GET /api/v1/auth/login`

从 SSO Discovery 获取授权地址，生成 OAuth `state` Cookie，然后以 `302` 重定向到 Gloscai SSO。无请求体。

可能错误：`503 sso_not_configured`、`502 sso_unavailable`。

### `GET /api/v1/auth/callback`

查询参数：

| 参数 | 必填 | 说明 |
| --- | --- | --- |
| `code` | 是 | SSO 返回的一次性授权码 |
| `state` | 是 | 必须与登录时的 `oauth_state` Cookie 一致 |
| `error_description` | 否 | SSO 拒绝授权时的错误说明 |

成功后创建或更新本地用户；数据库中的首个用户成为 `owner`。服务端设置应用会话 Cookie，并以 `302` 重定向到 `/`。

可能错误：`400 invalid_state`、`400 oauth_error`、`502 token_exchange_failed`、`502 userinfo_failed`、`500 user_error`、`500 session_error`。

### `GET /api/v1/auth/me`

成功响应 `200`：

```json
{
  "data": {
    "id": 1,
    "name": "user-name",
    "nickname": "显示名称",
    "email": "user@example.com",
    "avatar": "https://example.com/avatar.png",
    "role": "owner",
    "status": "active"
  }
}
```

### `POST /api/v1/auth/logout`

撤销数据库中的当前会话并删除 Cookie。无请求体，成功返回 `204 No Content`。未携带 Cookie 时同样返回 `204`。

## 任务接口

任务接口是现有模板示例，所有登录用户共享同一任务列表。

任务对象：

```json
{
  "id": 1,
  "title": "验证登录流程",
  "completed": false,
  "created_at": "2026-08-08T12:00:00Z"
}
```

### `GET /api/v1/tasks`

返回最多 100 条任务，按创建时间倒序排列：

```json
{"data": []}
```

### `POST /api/v1/tasks`

请求体：

```json
{"title":"验证登录流程"}
```

`title` 去除首尾空白后必须为 1 至 160 个字符。成功返回 `201` 和任务对象。

### `PATCH /api/v1/tasks/{id}`

请求体：

```json
{"completed":true}
```

成功返回 `200` 和更新后的任务对象。ID 非正整数返回 `400 invalid_id`，不存在返回 `404 not_found`。

### `DELETE /api/v1/tasks/{id}`

成功返回 `204 No Content`。ID 非正整数返回 `400 invalid_id`，不存在返回 `404 not_found`。

## 商业计划书

计划书对象（JSON 字段统一为 snake_case）：

```json
{
  "id": 1,
  "user_id": 1,
  "title": "创业计划书",
  "filename": "plan.pdf",
  "mime_type": "application/pdf",
  "status": "uploaded",
  "size_bytes": 102400,
  "version": 1,
  "created_at": "2026-08-08T12:00:00Z",
  "updated_at": "2026-08-08T12:00:00Z"
}
```

`status` 取值：`uploaded`（待分析）、`queued`（排队中）、`processing`（分析中）、`completed`（已完成）、`failed`（失败）。

### `GET /api/v1/plans`

只返回当前用户拥有的计划书，按创建时间倒序排列：

```json
{"data": []}
```

### `POST /api/v1/plans`

请求类型：`multipart/form-data`。

| 字段 | 必填 | 说明 |
| --- | --- | --- |
| `file` | 是 | 计划书文件，当前最大 20 MiB |
| `title` | 否 | 标题；为空时使用原始文件名 |

成功返回 `201` 和计划书对象。可能错误：`400 file_required`、`413 file_too_large`、`500 storage_error`。

示例：

```bash
curl -X POST http://localhost:5173/api/v1/plans \
  -b cookies.txt \
  -F 'title=创业计划书' \
  -F 'file=@plan.pdf'
```

### `GET /api/v1/plans/{id}`

只允许读取当前用户拥有的计划书。成功返回 `200` 和计划书对象；不存在或不属于当前用户时返回 `404 not_found`。

### `POST /api/v1/plans/{id}/analyze`

为当前用户拥有的计划书创建异步分析任务。无请求体，成功返回 `202`：

```json
{"data":{"id":12,"status":"queued"}}
```

不存在返回 `404 not_found`，不属于当前用户返回 `403 forbidden`。

### `GET /api/v1/plans/{id}/analysis`

返回当前用户计划书的最新一条分析任务；从未分析过时 `data` 为 `null`：

```json
{
  "data": {
    "id": 12,
    "plan_id": 1,
    "status": "succeeded",
    "error": "",
    "summary": "AI analysis completed.",
    "result": {"feasibility": "..."},
    "created_at": "2026-08-08T12:00:00Z",
    "updated_at": "2026-08-08T12:01:00Z"
  }
}
```

任务 `status` 取值：`queued`、`running`、`succeeded`、`failed`。`result` 为 AI 返回的结构化 JSON，失败时为空。计划书不存在或不属于当前用户时统一返回 `404 not_found`。

### `POST /api/v1/plans/{id}/analysis/retry`

将最新一条 `failed` 分析任务重新置为 `queued`。仅计划书所属用户或 Owner 可操作。无请求体，成功返回 `202` 和任务 id；没有失败任务时返回 `409 not_retryable`，无权操作返回 `403 forbidden`。

## Owner 管理接口

### `GET /api/v1/admin/users`

返回全部用户：

```json
{"data":[]}
```

用户对象在 `/auth/me` 格式基础上增加 `updated_at`（每次 SSO 登录都会刷新，可视为最近活跃时间）；SSO `sub` 不会返回。

### `PATCH /api/v1/admin/users/{id}`

请求体：

```json
{"status":"disabled"}
```

`status` 只能为 `active` 或 `disabled`。成功响应：

```json
{"data":{"status":"disabled"}}
```

Owner 账号不可禁用，尝试操作时返回 `422 owner_protected`。

### `GET /api/v1/admin/plans`

返回所有用户最近的最多 200 条计划书：

```json
{"data":[]}
```

### `GET /api/v1/admin/analysis`

分页返回全部用户的分析任务，按更新时间倒序。查询参数：

| 参数 | 必填 | 说明 |
| --- | --- | --- |
| `status` | 否 | 按任务状态筛选：`queued`、`running`、`succeeded`、`failed` |
| `page` | 否 | 页码，默认 1 |
| `page_size` | 否 | 每页条数，默认 25，最大 100 |

```json
{
  "data": [
    {
      "id": 12,
      "plan_id": 1,
      "status": "failed",
      "error": "could not reach AI provider",
      "summary": "AI analysis failed.",
      "created_at": "2026-08-08T12:00:00Z",
      "updated_at": "2026-08-08T12:01:00Z",
      "user_id": 1,
      "plan_title": "创业计划书"
    }
  ],
  "meta": {"page": 1, "page_size": 25, "total": 1}
}
```

`status` 非法时返回 `400 invalid_status`。

### `GET /api/v1/admin/settings/ai`

API Key 永不返回明文：

```json
{
  "data": {
    "endpoint": "https://api.example.com/v1",
    "model": "gpt-4o-mini",
    "has_api_key": true
  }
}
```

### `PATCH /api/v1/admin/settings/ai`

请求体：

```json
{
  "Endpoint": "https://api.example.com/v1",
  "Model": "gpt-4o-mini",
  "APIKey": "sk-example"
}
```

- `Endpoint` 和 `Model` 必填。
- `APIKey` 为空时保留已保存的密钥。
- 保存新密钥要求 `APP_ENCRYPTION_KEY` 是 32 字节，否则返回 `503 encryption_not_configured`。开发环境未配置时会使用内置的 32 字节开发密钥；生产环境必须显式配置随机密钥。
- 成功返回 `200`；响应中的 `has_api_key` 当前为字符串 `"true"` 或 `"false"`。

## 浏览器调用示例

前端必须允许浏览器携带 HttpOnly Cookie：

```ts
const response = await fetch('/api/v1/auth/me', {
  credentials: 'include',
  headers: { Accept: 'application/json' },
})
```
