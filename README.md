# Joggbe 開發與部署

## 本機前端

需求：Node.js 24、npm。

```powershell
npm ci
npm run dev
```

預設網址為 `http://localhost:5173/`。若要連接後端，可從 `.env.example` 建立
`.env.local`，並設定 `VITE_API_BASE_URL` 與 `VITE_USE_MOCK_API=false`。

### 本機串接 Go API

前端 request 固定呼叫同源 `/api/*` 並攜帶 Session Cookie。開發時可在 `.env.local` 設定：

```dotenv
VITE_API_BASE_URL=
VITE_DEV_API_TARGET=http://127.0.0.1:5000
VITE_USE_MOCK_API=false
```

Vite 會代理 `/api` 與 WebSocket 至 `VITE_DEV_API_TARGET`。後端在 `APP_ENV=development` 或
`APP_ENV=test` 時允許非 Secure 的本機 Session Cookie；production 一律使用 Secure Cookie。

API 欄位、狀態碼與限制請以 [`API_CONTRACT.md`](./API_CONTRACT.md) 為準。

## 品質檢查

```powershell
npm run verify
npm run test:e2e
```

`verify` 會依序執行 ESLint、Vitest 與正式建置。端對端測試第一次執行前需安裝
Chromium：

```powershell
npx playwright install chromium
```

Go 服務各自是獨立 module：

```powershell
cd Type-WSP-deploy/api
go test ./...
go vet ./...

cd ../worker
go test ./...
go vet ./...

cd ../shared
go test ./...
go vet ./...
```

Pull Request 與主要分支 push 會由 `.github/workflows/ci.yml` 執行上述檢查。

## Docker 部署

需求：Docker Engine 與 Docker Compose。

```powershell
cd Type-WSP-deploy
Copy-Item .env.example .env
docker compose up --build
```

本機完整環境使用 development overlay；`.env.local` 需提供非機密的本機服務設定，
包含 `TLS_CERTS_DIR=./nginx/certs`：

```powershell
docker compose --env-file .env --env-file .env.local `
  -f docker-compose.yaml -f docker-compose.dev.yaml up --build
```

啟動前必須替換 `.env` 內的 PostgreSQL、Redis、MinIO 密碼與 `SECRET_KEY`。
前端由 nginx 的 multi-stage image 直接從原始碼建置，不需也不應提交
`Type-WSP-deploy/frontend/dist`。

服務包含：

- `nginx`：WAF、TLS、前端靜態檔與 API reverse proxy
- `api`：Go HTTP API
- `worker`：Redis Stream 非同步工作
- `postgres`：user/system database
- `redis`：session、cache、task stream 與 Pub/Sub
- `minio`：原圖與處理後圖片
- `mailpit`（僅 development overlay）：攔截本機驗證信並提供 Web UI

健康檢查端點：

- `/health/live`：API process 存活
- `/health/ready`：PostgreSQL 與 Redis 均可用
- `/api/health`：保留給舊客戶端的 liveness 相容路徑

每個 HTTP response 都包含 `X-Request-ID`；API 會以 JSON 記錄 method、route、
status、response bytes 與 duration，方便本機除錯。

## 資料庫 migration

API 啟動時會在 advisory lock 保護下執行：

- `Type-WSP-deploy/api/migrations/user/*.sql`
- `Type-WSP-deploy/api/migrations/system/*.sql`

每個檔案以不可重複的數字版本開頭，例如 `002_add_profile.sql`。已執行版本記錄
在各 database 的 `schema_migrations`，因此既有永久 volume 也會套用新版本。
不要修改已部署過的 migration；新增下一個版本。

`postgres/init` 只負責全新 volume 建立第二個 database，名稱來自
`POSTGRES_DB_SYSTEM` 與 `POSTGRES_DB_USER`。

## 驗證碼郵件

Worker 透過 SMTP 寄送驗證碼。開發環境預設連到 Mailpit：

- SMTP：Docker 內部的 `mailpit:1025`
- 收件匣：`http://localhost:8025/`

可透過 `SMTP_HOST`、`SMTP_PORT`、`SMTP_FROM`、`SMTP_USERNAME`、
`SMTP_PASSWORD` 與 `SMTP_SECURE` 切換 SMTP 服務。`SMTP_SECURE=true` 代表要求
STARTTLS；production 環境禁止設為 `false`。驗證碼與 SMTP 密碼不會寫入 Log，
收件地址只記錄遮罩版本。

## 版本控制

以下內容不應提交：

- `.env*` 中的私密設定
- `node_modules/`
- `.go-build-cache/`
- `coverage/`、`playwright-report/`、`test-results/`
- `dist/` 與 `Type-WSP-deploy/frontend/dist/`
