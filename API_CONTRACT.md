# Type WSP API 合約

本文件是前後端交接的 API 基準。修改 endpoint、欄位、狀態碼、驗證規則或 enum 時，必須同步更新本文件、Go handler、前端 `src/api/contracts.ts` 與測試。

## 共通規則

- Base path：`/api`
- 正式環境採同源 HTTPS，由 Nginx 將 `/api/*` 反向代理至 Go API。
- 驗證方式：名稱為 `session` 的 HttpOnly Cookie；前端 request 必須使用 `credentials: include`。
- JSON response 的 `Content-Type` 必須包含 `application/json`。
- 所有瀏覽器 API 請求必須帶 `X-Type-WSP-Request: 1`；所有 JSON request 必須使用 `Content-Type: application/json`，且 body 只能包含單一 JSON value。
- JSON 欄位使用 `snake_case`。
- 一般錯誤格式：`{ "error": string }`。
- 日期時間使用 RFC 3339／ISO 8601 字串。
- `401` 代表登入狀態不存在或已失效。

## 共用資料型別

### User

```json
{
  "id": 1,
  "username": "Amy",
  "email": "amy@example.com"
}
```

### Post

```json
{
  "id": 10,
  "user_id": 1,
  "username": "Amy",
  "visibility": "public",
  "content": "Hello",
  "image_urls": ["/api/images/processed/example.jpg"],
  "image_status": "ready",
  "created_at": "2026-08-04T10:00:00Z"
}
```

- `visibility`：`public | private`
- `image_status`：`none | processing | ready | failed`
- `content`、`image_urls` 沒有內容時可能省略。
- 私人貼文只會回傳給貼文擁有者。

## 驗證 API

### `POST /api/auth/send-code`

Request：`{ "email": string }`

Success `200`：`{ "message": string, "challenge_id": string }`

可能錯誤：`400`、`429`、`500`、`503`。

規則：註冊驗證碼有效 24 小時、每個來源最多錯誤嘗試 5 次、每個 challenge 全域最多錯誤嘗試 20 次、重新寄送冷卻 1 分鐘。收件信箱寄送上限為每小時 5 次／每日 10 次，來源端上限為每小時 20 次／每日 50 次。每次寄送會產生獨立的 `challenge_id` 與高熵驗證碼；新 challenge 不會使已寄達的舊 challenge 失效，錯誤嘗試額度耗盡也不會阻擋正確的高熵驗證碼。

### `POST /api/auth/register`

Registration challenges use a 16-character uppercase Base32 code and expire after 24 hours. Each `challenge_id` remains independently usable until consumed or expired; sending a newer code does not invalidate an earlier delivered challenge. Recipient and source send quotas still apply. If a recipient quota or cooldown is reached while an unexpired challenge exists, `/api/auth/send-code` returns the latest challenge instead of replacing it.

Request：

```json
{
  "username": "Amy",
  "email": "amy@example.com",
  "password": "password1",
  "code": "ABCDEFGHJKLMNPQR",
  "challenge_id": "adf04b8e-9ae7-4dd5-a924-0b299a5aa865"
}
```

Success `201`：`{ "message": "registered", "user_id": 1 }`

規則：

- username 為 2–20 個 Unicode 字元，不得包含空白或控制字元。
- password 至少為 8 個 Unicode 字元、UTF-8 編碼不得超過 72 bytes，且至少包含一個字母與一個數字。
- code 必須是 16 位大寫 Base32 字元（排除易混淆的 `I`、`O`、`0`、`1`）。
- challenge_id 必須是最近一次寄送驗證碼所回傳的 UUID，並與 email、code 一起驗證。

### `POST /api/auth/login`

Request：`{ "email": string, "password": string, "remember": boolean }`

Success `202`：

```json
{
  "message": "login verification code sent",
  "challenge_id": "2cd53940-fc0d-4972-921b-086061dde6e5",
  "requires_verification": true,
  "expires_in_seconds": 300
}
```

密碼正確後只寄送登入驗證碼，不建立 Session，也不設定 Cookie。登入與註冊 challenge 使用不同 Redis namespace 與寄送額度。

### `POST /api/auth/login/verify`

Request：

```json
{
  "email": "amy@example.com",
  "code": "123456",
  "challenge_id": "2cd53940-fc0d-4972-921b-086061dde6e5",
  "remember": false
}
```

Success `200`：`{ "user": User }`，並設定 `session` Cookie。

- 一般 Session 預設 1 天，可由後端環境變數調整。
- `remember=true` 時為 30 天。
- 每次新登入都必須完成 6 位數 Email 驗證碼；challenge 有效 5 分鐘且只能成功使用一次。
- 來源端在 5 分鐘內累積 10 次錯誤密碼後會被暫時限制；帳號總錯誤次數只作風險訊號，不會在驗證密碼前阻擋其他乾淨來源的正確登入。
- 登入驗證碼每個來源最多錯誤嘗試 5 次、每個 challenge 全域最多 20 次；全域額度耗盡後 challenge 立即失效。

### `GET /api/auth/session`

Success `200`：`{ "user": User }`

### `POST /api/auth/logout`

Success `200`：`{ "message": "logged out" }`，並清除 Session Cookie。

## 貼文 API

### `GET /api/feed?cursor={cursor}`

Success `200`：

```json
{
  "posts": [],
  "next_cursor": ""
}
```

- 每頁最多 20 筆。
- 第一頁不傳 cursor。
- `next_cursor` 為 opaque string，前端不得自行解析。
- 沒有下一頁時回傳空字串。

### `GET /api/posts/me?cursor={cursor}`

回傳格式與 `/api/feed` 相同，但只包含目前登入使用者擁有的貼文（包含公開與私人貼文）。
每頁最多 20 筆，並使用相同的 opaque cursor 分頁規則。

### `POST /api/posts`

純文字使用 JSON：

```json
{
  "content": "Hello",
  "visibility": "public"
}
```

圖片貼文使用 `multipart/form-data`：

- `content`：最多 5,000 個 Unicode 字元，可在有圖片時留空。
- `visibility`：`public | private`，省略時為 `public`。
- `images`：同名欄位最多 4 個檔案。
- 圖片格式：JPEG、PNG。
- 單檔上限：8 MiB。
- Request body 上限：25 MiB。
- 每位使用者的處理後圖片與處理中預留空間合計上限為 1 GiB；超過時回傳 `429`。

Success `201`：`{ "message": string, "post_id": number }`

圖片貼文建立後初始狀態為 `processing`；Worker 完成後改為 `ready`，失敗則改為 `failed`。

### `DELETE /api/posts/{id}`

只有貼文擁有者可刪除。

Success `200`：`{ "message": "post deleted" }`

### `GET /api/images/{key}`

回傳圖片 binary。只有公開圖片或目前使用者自己的私人圖片可讀取；未授權與不存在皆回傳 `404`。

## 私人資料

以下端點全部需要有效的 Session Cookie，後端只會依目前登入使用者的 `user_id` 讀寫資料。

### `GET /api/schedule`

取得目前使用者的任務與提醒。尚無資料時回傳空陣列：

```json
{ "tasks": [], "reminders": [] }
```

### `PUT /api/schedule`

以完整的 `{ "tasks": [...], "reminders": [...] }` 取代目前使用者的行程。後端會驗證日期、時間、優先級、長度及重複 ID。

### `GET /api/inspirations`

回傳目前使用者的靈感筆記：`{ "items": [...] }`。

### `POST /api/inspirations`

新增靈感筆記。Body：`{ "date": "YYYY-MM-DD", "text": "...", "imageLabel": "..." }`。

### `PATCH /api/inspirations/{id}`

更新目前使用者擁有的靈感文字。Body：`{ "text": "..." }`。

### `DELETE /api/inspirations/{id}`

刪除目前使用者擁有的靈感筆記。

## WebSocket

### `GET /api/ws/`

- 使用同一個 Session Cookie 驗證。
- 正式環境使用 `wss`。
- Worker 完成圖片處理時目前會送出：

```json
{
  "type": "post_ready",
  "post_id": 10
}
```

目前 `failed` 狀態沒有 WebSocket event，前端仍須保留輪詢或重新整理機制。

## 尚未提供正式 API 的功能

- 忘記密碼／重設密碼
- Explore
- Freq／設定摘要
- 個人資料預覽與編輯

這些功能目前使用 Mock API 或尚未完成，不得當成已完成的後端合約。
