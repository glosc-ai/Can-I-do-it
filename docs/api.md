# API 接口文档

本文档以当前 Go 服务实际注册的路由为准。

## 通用约定

- API 基础路径：`/api/v1`
- 健康检查路径：`/health`
- 数据格式：除文件上传和重定向接口外，均使用 `application/json`
- 登录态：OAuth 回调成功后由服务端设置 `can_i_session` HttpOnly Cookie；浏览器请求需携带 Cookie
- 写入、更新和删除接口需要携带 `X-Requested-With: XMLHttpRequest`；前端客户端会自动添加，用于防止跨站表单伪造请求
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
| POST | `/api/v1/plans` | 登录 | 上传计划书或图片 |
| GET | `/api/v1/plans/{id}` | 登录 | 获取自己的单个计划书 |
| POST | `/api/v1/plans/{id}/analyze` | 登录 | 创建异步分析任务 |
| GET | `/api/v1/plans/{id}/analysis` | 登录 | 获取计划书的最新分析任务 |
| POST | `/api/v1/plans/{id}/analysis/retry` | 登录 | 重新入队失败的分析任务 |
| PATCH | `/api/v1/plans/{id}/visibility` | 登录 | 切换计划书公开/私有 |
| GET | `/api/v1/gallery/plans` | 公开 | 分页获取已公开且分析成功的项目 |
| GET | `/api/v1/gallery/plans/{id}` | 公开 | 获取单个公开项目的完整分析报告 |
| GET | `/api/v1/gallery/similar` | 公开 | 按标题模糊匹配已公开的相似项目 |
| GET | `/api/v1/assets` | 登录 | 获取当前用户的素材对象 |
| POST | `/api/v1/assets` | 登录 | 上传或登记素材（upload / ai_generated / fetched） |
| GET | `/api/v1/assets/{id}` | 登录 | 获取素材元数据和下载链接 |
| GET | `/api/v1/assets/{id}/download` | 登录 | 下载素材或跳转到 R2 链接 |
| DELETE | `/api/v1/assets/{id}` | 登录 | 删除自己的素材 |
| GET | `/api/v1/admin/users` | Owner | 获取全部用户 |
| PATCH | `/api/v1/admin/users/{id}` | Owner | 启用或禁用普通用户 |
| GET | `/api/v1/admin/plans` | Owner | 获取全部计划书 |
| GET | `/api/v1/admin/analysis` | Owner | 分页获取全部分析任务 |
| GET | `/api/v1/admin/settings/ai` | Owner | 获取 AI 服务配置 |
| PATCH | `/api/v1/admin/settings/ai` | Owner | 更新 AI 服务配置 |
| GET | `/api/v1/admin/settings/storage` | Owner | 获取 R2 存储配置（不返回密钥） |
| PATCH | `/api/v1/admin/settings/storage` | Owner | 更新 R2 存储配置 |
| POST | `/api/v1/admin/settings/storage/test` | Owner | 测试 R2 Bucket 连接 |
| GET | `/api/v1/admin/assets` | Owner | 分页获取全部素材对象 |
| DELETE | `/api/v1/admin/assets/{id}` | Owner | 删除素材对象和记录 |

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
  "visibility": "private",
  "asset_id": 18,
  "download_url": "/api/v1/assets/18/download",
  "created_at": "2026-08-08T12:00:00Z",
  "updated_at": "2026-08-08T12:00:00Z"
}
```

`status` 取值：`uploaded`（待分析）、`queued`（排队中）、`processing`（分析中）、`completed`（已完成）、`failed`（失败）。`visibility` 取值：`private`（默认，仅本人可见）、`public`（分析成功后会出现在项目广场）。

### `GET /api/v1/plans`

只返回当前用户拥有的计划书，按创建时间倒序排列：

```json
{"data": []}
```

### `POST /api/v1/plans`

请求类型：`multipart/form-data`。

| 字段 | 必填 | 说明 |
| --- | --- | --- |
| `file` | 是 | 计划书文件或图片，当前最大 20 MiB；支持 PDF、DOC/DOCX、TXT/Markdown、PNG/JPG/WEBP |
| `title` | 否 | 标题；为空时使用原始文件名 |
| `visibility` | 否 | `public` 或 `private`；为空时默认为 `private` |

成功返回 `201` 和计划书对象。可能错误：`400 file_required`、`400 unsupported_file_type`、`413 file_too_large`、`500 storage_error`。

示例：

```bash
curl -X POST http://localhost:5173/api/v1/plans \
  -b cookies.txt \
  -F 'title=创业计划书' \
  -F 'file=@plan.pdf'
```

### `GET /api/v1/plans/{id}`

只允许读取当前用户拥有的计划书。成功返回 `200` 和计划书对象；不存在或不属于当前用户时返回 `404 not_found`。

计划书上传会自动创建一个 `source=upload` 的素材对象；`asset_id` 和 `download_url` 可用于下载原始文件。

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
    "overall_score": 68.5,
    "verdict": "有条件可行",
    "dimensions": [
      {"key":"market","name":"市场空间","score":72,"weight":15,"confidence":64,"reasoning":"…","evidence":["…"],"gaps":["…"]}
    ],
    "analysis_process": [
      {"step":"market","title":"市场空间","status":"completed","summary":"检查了市场规模与增长证据","questions":["还需验证…"]}
    ],
    "result": {"overall_score": 68.5, "dimensions": [{"key":"market","score":72}], "analysis_process": []},
    "created_at": "2026-08-08T12:00:00Z",
    "updated_at": "2026-08-08T12:01:00Z"
  }
}
```

任务 `status` 取值：`queued`、`running`、`succeeded`、`failed`。Worker 会从对象存储读取并解析上传内容：DOCX/TXT/Markdown/PDF 提取文字，图片发送给视觉模型。`result` 为规范化后的结构化 JSON；九个维度的分数也会写入 `analysis_dimension_scores` 表，失败时为空。计划书不存在或不属于当前用户时统一返回 `404 not_found`。

### `POST /api/v1/plans/{id}/analysis/retry`

将最新一条 `failed` 分析任务重新置为 `queued`。仅计划书所属用户或 Owner 可操作。无请求体，成功返回 `202` 和任务 id；没有失败任务时返回 `409 not_retryable`，无权操作返回 `403 forbidden`。

### `PATCH /api/v1/plans/{id}/visibility`

仅计划书所属用户可操作。请求体：

```json
{"visibility":"public"}
```

`visibility` 必须为 `public` 或 `private`，否则返回 `422 invalid_visibility`。成功返回 `200`：

```json
{"data":{"visibility":"public"}}
```

设为 `public` 不要求分析已经完成；项目广场只展示 `visibility=public` 且 `status=completed` 的计划书，其余状态下切换公开只是预先设置好可见性。不存在返回 `404 not_found`，不属于当前用户返回 `403 forbidden`。

## 项目广场

项目广场展示所有用户主动公开、且分析已成功完成的计划书，任何人无需登录即可访问。这三个接口按客户端 IP 做了 Redis 限流（每分钟 30 次），超出后返回 `429 rate_limited`。

### `GET /api/v1/gallery/plans`

分页返回公开项目列表，按创建时间倒序排列。查询参数 `page`（默认 1）、`page_size`（默认 20，最大 50）。

```json
{
  "data": [
    {
      "id": 1,
      "title": "社区咖啡店计划书",
      "filename": "plan.pdf",
      "mime_type": "application/pdf",
      "overall_score": 74,
      "verdict": "有条件可行",
      "author_name": "小莫",
      "author_avatar": "https://example.com/avatar.png",
      "created_at": "2026-08-08T12:00:00Z"
    }
  ]
}
```

`author_name` 取用户昵称，昵称为空时回退到 SSO 姓名；不返回 `user_id`。

### `GET /api/v1/gallery/plans/{id}`

返回单个公开项目的计划书信息、作者信息与完整分析结果：

```json
{
  "data": {
    "plan": { "id": 1, "title": "…", "visibility": "public", "status": "completed", "…": "…" },
    "author_name": "小莫",
    "author_avatar": "https://example.com/avatar.png",
    "analysis": { "id": 12, "overall_score": 74, "dimensions": [], "…": "…" }
  }
}
```

`plan.user_id` 会被清零，不对匿名访客暴露作者的内部用户 ID；作者身份只通过 `author_name`/`author_avatar` 暴露。计划书不存在、未公开或未分析成功时统一返回 `404 not_found`（不区分"不存在"和"存在但不可见"，避免泄露私有计划书的存在性）。

### `GET /api/v1/gallery/similar`

按标题对已公开且分析成功的项目做模糊匹配，用于提交分析前提醒用户是否已有相似项目。查询参数 `q`（必填，2–200 字符）。最多返回 5 条，按相似度排序：

```json
{
  "data": [
    {"id": 1, "title": "社区咖啡店计划书", "overall_score": 74, "verdict": "有条件可行", "created_at": "2026-08-08T12:00:00Z"}
  ]
}
```

`q` 长度不足 2 时直接返回空列表。Postgres 使用 `pg_trgm` 的 `similarity()` 函数（阈值 0.3），MySQL 使用 `FULLTEXT` 自然语言匹配；两者语义均为近似匹配，不保证跨语言/跨分词器结果完全一致。

## 素材对象

素材对象统一记录用户上传、AI 生成和外部获取的二进制资料。`source` 只能是 `upload`、`ai_generated` 或 `fetched`。对象内容写入 R2 后，默认通过 15 分钟有效的私有预签名 URL 下载；配置 `R2_PUBLIC_URL` 时使用公开 CDN URL。未启用 R2 时使用 `UPLOAD_DIR` 本地回退存储。

### `POST /api/v1/assets`

请求类型：`multipart/form-data`，字段如下：

| 字段 | 必填 | 说明 |
| --- | --- | --- |
| `file` | 是 | 素材文件，最大 20 MiB |
| `source` | 否 | `upload`（默认）、`ai_generated` 或 `fetched` |
| `name` | 否 | 展示名称，默认使用文件名 |
| `plan_id` | 否 | 关联当前用户的计划书 |
| `metadata` | 否 | JSON 对象字符串 |

服务端 AI/抓取任务也可以直接调用 `assets.Service.Save`，无需伪造 HTTP 上传请求。

### `GET /api/v1/assets`

返回当前用户最近 200 个素材；可用 `?source=ai_generated` 筛选来源。每项包含 `download_url`，对象 key 和凭据不会返回。

### `GET /api/v1/assets/{id}/download` 与 `DELETE /api/v1/assets/{id}`

只允许素材所属用户或 Owner 访问。下载在 R2 模式下跳转到签名 URL，本地模式由 API 鉴权后读取文件；删除会同时删除对象和数据库记录。

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

### R2 存储设置与素材管理

`GET /api/v1/admin/settings/storage` 返回 endpoint、bucket、公开 URL、region 和 `has_credentials`，不会返回 Access Key 或 Secret。`PATCH` 接受同名 JSON 字段（前端使用 snake_case），凭据为空时保留原值；`clear_credentials=true` 可清除凭据。启用 R2 时 endpoint 和 bucket 必填，并且要求 `http(s)` URL。

`POST /api/v1/admin/settings/storage/test` 会执行 Bucket `HeadBucket` 检查。`GET /api/v1/admin/assets` 支持 `source`、`page`、`page_size` 筛选并返回全部用户素材；`DELETE /api/v1/admin/assets/{id}` 同时删除 R2 对象和记录。

## 浏览器调用示例

前端必须允许浏览器携带 HttpOnly Cookie：

```ts
const response = await fetch('/api/v1/auth/me', {
  credentials: 'include',
  headers: { Accept: 'application/json' },
})

// POST/PATCH/DELETE additionally require this non-simple header.
const upload = await fetch('/api/v1/assets', {
  method: 'POST',
  credentials: 'include',
  headers: { 'X-Requested-With': 'XMLHttpRequest' },
  body: formData,
})
```
