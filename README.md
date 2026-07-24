# Joggbe 開發與部署

## 本機前端

需求：Node.js 24、npm。

```powershell
npm ci
npm run dev
```

預設網址為 `http://localhost:5173/`。若要連接後端，可從 `.env.example` 建立
`.env.local`，並設定 `VITE_API_BASE_URL` 與 `VITE_USE_MOCK_API=false`。

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

健康檢查端點為 `/api/health`。

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

目前 worker 的寄信工作仍是開發用 stub，正式環境不會寄信。依目前決定暫不串接
SMTP／郵件服務，因此正式註冊流程在此功能完成前不可視為可用。

## 版本控制

以下內容不應提交：

- `.env*` 中的私密設定
- `node_modules/`
- `.go-build-cache/`
- `coverage/`、`playwright-report/`、`test-results/`
- `dist/` 與 `Type-WSP-deploy/frontend/dist/`
