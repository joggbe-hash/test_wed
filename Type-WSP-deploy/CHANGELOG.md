# CHANGELOG — Type-WSP 合併部署與 API 建置

> 記錄 2026-05-24 ~ 2026-05-25

## 2026-08-11 — 所有登入加入 Email 驗證碼

- `POST /api/auth/login` 在密碼正確後改為回傳短效 challenge 並寄出 Email 驗證碼，不再立即建立 Session。
- 新增 `POST /api/auth/login/verify`；驗證碼原子消耗成功後才設定 HttpOnly Session Cookie。
- 登入與註冊驗證碼使用獨立 Redis namespace、寄送冷卻與嘗試額度，避免跨流程重放或互相耗盡。
- 前端登入頁新增可存取的驗證碼步驟，密碼送出後立即從記憶體狀態清除。

---

## 2026-08-11 — 圖片上傳交易邊界修正

- 新增有期限的圖片上傳配額預留；短 transaction 內以 advisory lock 原子檢查並預留貼文、處理中圖片與儲存配額。
- MinIO 原始檔上傳只在 transaction 外執行，完成後再用第二個短 transaction 把預留轉成 `processing` 貼文。
- Worker 會清除逾時預留及其 `raw/` 物件；多個 Worker 以 `SKIP LOCKED` 分工，不會重複認領。
- MinIO bucket 保留既有 lifecycle 規則，並新增 `raw/` 物件一日後自動過期的最後防線。

---

## 2026-08-04 — 開發環境寄信、健康檢查與可觀測性

- Worker 新增可替換的 `MailSender` 與 SMTP 實作，寄送失敗沿用 Redis Stream 重試及 Dead Letter。
- development Compose 新增 Mailpit，Web UI 使用 `http://localhost:8025/`。
- API 新增 `/health/live`、`/health/ready`，並保留 `/api/health` 相容端點。
- API 新增 JSON HTTP request log 與 `X-Request-ID`。
- 登入頁流程移至 auth composable；任務清單與靈感日期篩選拆成 typed 子元件。
- 新增 SMTP、健康檢查、HTTP middleware、Vue 子元件及註冊 E2E 測試。

---

## 2026-07-25 — 維運結構修正與 nginx 部署

### 後端可靠性

- 圖片處理改用資料列層級的 `processing_token` 與租約進行原子 claim，避免重複任務同時處理同一篇貼文。
- 每次圖片工作使用獨立 processed object key；失敗回滾只會刪除該次工作建立的檔案。
- Redis Stream 的重試入列、ACK 與原訊息刪除改在同一個 Redis transaction 執行。
- API 與 Worker 共用 rollback manager；補償操作使用獨立、具 timeout 且不受原 request cancellation 影響的 context。

### 資料庫與部署

- 新增版本化 user/system database migration、`schema_migrations` 紀錄及 PostgreSQL advisory lock。
- PostgreSQL 初始化腳本改由環境變數建立 database，不再寫死 `user_db`、`system_db` 與 owner。
- API 啟動時先執行 migration，成功後才進入服務狀態。
- Go、Alpine、Node、PostgreSQL、Redis、MinIO 與 ModSecurity CRS 映像皆以 digest 鎖定。
- nginx 改為 multi-stage build，直接從 Vue 原始碼產生前端，不再依賴 Git 追蹤的 `frontend/dist`。
- 從 Git index 移除 2,478 個 `.go-build-cache` 檔案及 12 個舊 frontend dist 檔案。

### 前端維運

- 解開 `backendApi → router → useSession → backendApi` 循環依賴，401 導頁改由應用程式邊界注入。
- 將排程功能拆為 repository、seed、service、日期 editor、側欄日曆及互動 controller。
- `SidebarWidgets.vue` 從 535 行降至 234 行；`DateScheduleModal.vue` 從 599 行降至 360 行。
- 新增共用 dialog focus trap composable。

### 測試與 CI

- 新增 ESLint、Vitest、Vue Test Utils 與 Playwright 設定。
- 新增 API client、Pinia feed store、session、schedule repository、刪除對話框及登入 E2E 測試。
- 新增 Pull Request CI：前端 lint/test/build、Go test/vet 與 Playwright。
- GitHub Pages 部署前增加 lint 與單元測試。

### 驗證結果

- Go `shared`、`api`、`worker`：`go test ./...` 與 `go vet ./...` 通過。
- 前端：TypeScript typecheck、ESLint、5 個 Vitest 測試檔／7 項測試通過。
- Playwright 登入流程通過。
- Docker Compose 設定驗證通過，API、Worker、nginx images 建置成功。

### nginx 部署紀錄

- 部署時間：2026-07-25（Asia/Taipei）。
- 部署方式：沿用運行中 nginx 的 `SERVER_NAME` 與 API port，以 `--no-deps` 僅重建 `nginx-waf`，未重建 API、PostgreSQL、Redis 或 MinIO。
- 部署 image：`type-wsp-deploy-nginx:latest`。
- 上線前端資產：`assets/index-BM73O5II.js`。
- 驗證結果：容器 `healthy`、`https://127.0.0.1/` 回傳 HTTP 200，且實際載入新版資產。
- 第一次部署因缺少 `Type-WSP-deploy/.env` 被 Compose 中止，未修改容器；後續使用現行 nginx 設定完成安全部署。

### 已知限制

- 正式驗證碼郵件仍是 Worker stub，尚未串接 SMTP／郵件服務。
- 專案尚未建立正式 `.env`；下次完整 Compose 部署前必須補齊正式環境設定與密碼。
- GitHub branch protection 仍需將 `CI` 設為 required check，才能真正阻止未通過測試的 PR 合併。

---

## 2026-07-15 — 登入狀態、個人排程與互動體驗更新

### 登入與 Session

- 新增 `GET /api/auth/session`，讓前端可透過既有 HttpOnly session cookie 恢復登入狀態。
- 為受保護頁面加入路由守衛；未登入時導向登入頁，登入完成後回到原本目標頁面。
- 登入與 session 探測不再由共用 `401` 處理流程提前改寫導向，避免遺失原始 redirect。
- 區分無效／過期 session 與 Redis 等 session service 故障，分別回傳 `401` 與 `503`。
- 登入錯誤、session 檢查與登出狀態改為明確的繁體中文提示。
- 驗證碼改用 `crypto/rand` 產生；`debug_code` 僅在 development 且明確開啟設定時回傳。

### 個人排程與提醒

- 排程 localStorage 改為 `type-wsp-schedule-mock:<userId>`，避免不同登入帳號共用同一份資料。
- 新增舊版排程匯入／放棄流程；使用者尚未決定前，禁止新增、修改、刪除及排序。
- 補強 localStorage 不可用、容量不足與資料格式錯誤時的失敗處理，避免顯示已成功但實際未儲存。
- 日期與今日種子資料改以本地時區建立，避免 UTC 日期造成跨日錯誤。
- 支援新增、編輯及刪除提醒，並為任務、提醒與貼文刪除加入共用確認對話框。

### 前端介面與無障礙

- 重整個人與設定頁，加入登入帳號摘要、設定入口卡片、載入骨架、錯誤重試與登出區塊。
- 排程、每日任務與刪除確認對話框加入焦點鎖定、關閉後焦點復原、`inert` 與 ARIA 狀態。
- 側欄今日任務與提醒支援完整清單、編輯／刪除選單及獨立捲動。
- 貼文時間改為依 `created_at` 顯示相對時間，並定期更新。
- 已部署前端 bundle 同步重建至 `Type-WSP-deploy/frontend/dist`。

### 部署與安全

- nginx 僅對登入、註冊與寄送驗證碼套用嚴格限流；session 與 logout 使用一般 API 限流。
- 補上安全 response headers，包含 HSTS、CSP、`X-Content-Type-Options`、`X-Frame-Options`、Referrer Policy 與 Permissions Policy。
- 圖片 worker 一律重新編碼上傳圖片，移除來源 EXIF 等中繼資料，並新增對應測試。
- 新增 auth/session 測試，涵蓋 session 恢復、無效 session 與 session service 故障情境。

---

## 1. 分支合併與 Docker 部署整合

### 背景
專案原有兩個分支各自獨立：
- **main**：Docker 基礎設施（docker-compose、nginx、PostgreSQL 初始化）
- **front_design_wed**：前端靜態頁面（HTML/CSS/JS/字型/圖片）

### 執行內容
- 建立 `deploy` 分支，以 `main` 為基底
- 將 `front_design_wed` 的前端檔案放入 `frontend/dist/`（nginx 掛載路徑）
- 產生開發用自簽 SSL 憑證（`nginx/certs/fullchain.pem`、`privkey.pem`）
- 建立 `.env` 和 `.gitignore`

### 新增檔案
```
frontend/dist/          ← 前端靜態檔（從 front_design_wed 分支遷入）
  index.html, first_page.html, explore_page.html, ...
  css/, js/, fonts/, picture/
nginx/certs/            ← 開發用 SSL 憑證（已加入 .gitignore）
.env                    ← 環境變數（已加入 .gitignore）
.gitignore              ← 新建
```

---

## 2. Nginx 設定修正

### 變更檔案
- `nginx/templates/conf.d/default.conf.template`（**新建**，取代原本的 `conf/server.conf.template`）
- `docker-compose.yaml`

### 修正內容
| 問題 | 修正 |
|------|------|
| OWASP CRS 映像要求容器內 port > 1024 | `PORT: 80` → `8080`、`SSL_PORT: 443` → `8443`，對外映射不變 |
| 原本的 `conf/server.conf.template` 不被 nginx 載入 | 改掛載到 `conf.d/default.conf.template`，取代映像預設設定 |
| `log_format "main"` 未定義導致 nginx 啟動失敗 | 改用 `combined` 格式 |
| CSP 阻擋 inline `onclick` 事件 | `script-src` 加入 `'unsafe-inline'` |
| `/api/` 的 `proxy_pass` 帶尾部 `/` 會剝除路徑前綴，與 auth regex location 不一致 | 移除尾部 `/`，統一保留完整路徑 |
| CSP 未允許 WebSocket 連線 | `connect-src` 加入 `wss:` |

---

## 3. Go API Server（全部新建）

### 架構設計
- 使用 Go 標準庫 `net/http`（Go 1.22+ ServeMux 支援 method routing）
- Session 機制：HMAC-SHA256 簽名 + Redis 儲存 + HttpOnly Secure Cookie
- 原子性回滾：`AtomicRollback` 模式處理跨服務操作的補償
- 圖片代理：`/api/images/{key}` 從 MinIO 串流圖片，MinIO 不對外暴露
- WebSocket：`/api/ws/` 訂閱 Redis Pub/Sub，即時推送 Worker 完成通知

### 新增檔案
```
api/
  main.go           ← 進入點、路由註冊
  config.go         ← 環境變數載入
  db.go             ← PostgreSQL 連線池（user_db + system_db）、交易包裝
  redis.go          ← Redis 客戶端
  minio_init.go     ← MinIO 客戶端、bucket 初始化
  session.go        ← Session CRUD、requireAuth 中介層
  rollback.go       ← AtomicRollback 跨服務補償回滾
  auth.go           ← 認證端點（send-code、register、login、logout）
  posts.go          ← 貼文端點（feed、create post、image proxy）
  ws.go             ← WebSocket 即時通知（Redis Pub/Sub → 前端）
  go.mod            ← Go module 定義
  Dockerfile        ← 多階段建置（golang:1-alpine → alpine:3.21）
```

### API 端點
| Method | Path | 需登入 | 說明 |
|--------|------|--------|------|
| GET | /api/health | 否 | 健康檢查 |
| POST | /api/auth/send-code | 否 | 發送驗證碼（存 Redis + Worker 寄信） |
| POST | /api/auth/register | 否 | 註冊（驗碼比對 + bcrypt + DB） |
| POST | /api/auth/login | 否 | 驗證密碼並寄送登入驗證碼（不建立 Session） |
| POST | /api/auth/login/verify | 否 | 驗證登入碼後建立 Redis Session + Cookie |
| POST | /api/auth/logout | 否 | 登出（刪除 session） |
| GET | /api/feed | 是 | 動態牆（最新 20 筆，Redis 快取 30s，cursor 分頁） |
| POST | /api/posts | 是 | 發文（純文字直寫 DB / 帶圖走 Worker） |
| GET | /api/ws/ | 是 | WebSocket 即時通知 |
| GET | /api/images/{key} | 否 | MinIO 圖片代理 |

### 原子性回滾涵蓋的場景
| 場景 | 步驟 | 失敗時回滾 |
|------|------|------------|
| 發送驗證碼 | Redis SET → Redis RPUSH 佇列 | 刪除已存的驗證碼 |
| 圖片發文 (API) | MinIO 上傳 → DB INSERT → Redis RPUSH | 刪除 MinIO 檔案 + 刪除 DB 記錄 |
| 圖片處理 (Worker) | MinIO 上傳處理後圖片 → DB UPDATE | 刪除處理後圖片 |

---

## 4. Go Worker（全部新建）

### 架構設計
- 從 Redis `task_queue` 以 BLPOP 取出任務（阻塞等待，不浪費 CPU）
- 根據 `type` 欄位分派到對應處理函式
- 圖片處理：下載原圖 → 等比縮放（最大 1920px）→ JPEG 壓縮（品質 85）→ 更新 DB
- 處理完成後透過 Redis PUBLISH 通知前端

### 新增檔案
```
worker/
  main.go           ← 進入點、任務分派迴圈、Email 佔位處理
  config.go         ← 環境變數載入
  rollback.go       ← AtomicRollback（與 API 相同模式）
  image.go          ← 圖片處理（縮圖 + 壓縮 + DB 更新 + Pub/Sub 通知）
  go.mod            ← Go module 定義
  Dockerfile        ← 多階段建置
```

### 任務類型
| type | 來源 | 處理內容 |
|------|------|----------|
| `process_image_post` | POST /api/posts（帶圖） | 縮圖壓縮 → 更新 DB → 通知前端 |
| `send_verification_email` | POST /api/auth/send-code | 寄驗證碼（目前僅 log） |

---

## 5. PostgreSQL Schema 變更

### 變更檔案
- `postgres/init/01-create-databases.sql`

### 新增的表
**user_db.users**
| 欄位 | 型別 | 說明 |
|------|------|------|
| id | SERIAL PK | 自動遞增主鍵 |
| username | VARCHAR(50) | 顯示名稱 |
| email | VARCHAR(255) UNIQUE | 登入信箱 |
| password_hash | VARCHAR(255) | bcrypt 雜湊密碼 |
| created_at | TIMESTAMP | 建立時間 |

**system_db.posts**
| 欄位 | 型別 | 說明 |
|------|------|------|
| id | SERIAL PK | 自動遞增主鍵 |
| user_id | INTEGER | 發文者 ID |
| username | VARCHAR(50) | 發文者名稱（反正規化） |
| content | TEXT | 文字內容 |
| image_url | TEXT | 圖片路徑 JSON 陣列 |
| image_status | VARCHAR(20) | none / processing / ready / failed |
| created_at | TIMESTAMP | 發文時間（含倒序索引） |

---

## 6. 前端改造

### 變更檔案
- `frontend/dist/index.html`
- `frontend/dist/js/index.js`
- `frontend/dist/first_page.html`
- `frontend/dist/js/first_page.js`

### index.html + index.js（登入/註冊頁）
| 原本 | 改為 |
|------|------|
| 帳號輸入框（純 UI） | 電子郵件輸入框 → `POST /api/auth/login` |
| 註冊按鈕 alert | 發送驗證碼 → `POST /api/auth/send-code`（60s 冷卻） |
| 靜態跳轉 | 驗碼比對 → `POST /api/auth/register` → 成功後跳回登入 |
| 無錯誤提示 | 每個表單加 error 訊息區 |
| — | DEBUG 模式：驗證碼直接顯示在頁面上 |
| — | Enter 鍵觸發登入/註冊 |

### first_page.html + first_page.js（動態牆）
| 原本 | 改為 |
|------|------|
| 假的打字框 UI | 真的 `<textarea>` + 發文功能 → `POST /api/posts` |
| 靜態貼文卡片 | 從 `GET /api/feed` 動態載入 |
| 單圖上傳 | 支援多圖選取（`multiple`），可多次累加，個別刪除 |
| 無 cursor 分頁 | 支援 cursor 分頁 + 「載入更多」按鈕 |
| 圖片處理後需手動重整 | WebSocket 即時接收 Worker 完成信號 → 自動 refreshFeed |
| 未登入可直接進入 | `checkAuth()` 檢查 session，未登入跳回 index.html |

---

## 7. 即時通知機制（WebSocket + Redis Pub/Sub）

### 信號流程
```
使用者發帶圖貼文
  → API：上傳 MinIO + 寫 DB (processing) + 丟 Redis queue
  → 前端：馬上看到「圖片處理中...」

Worker 從 queue 取出任務
  → 壓縮/縮圖 → 更新 DB (ready)
  → Redis PUBLISH "notify:user:{id}" → '{"type":"post_ready"}'

API WebSocket handler（持續訂閱 Redis channel）
  → 收到 Pub/Sub 訊息 → 轉發給該使用者的 WebSocket 連線

前端 WebSocket onmessage
  → 收到 post_ready → 自動 refreshFeed()
  → 圖片出現，不需手動重整
```

### 涉及的檔案
- `api/ws.go` — WebSocket handler + Redis Subscribe
- `worker/image.go` — 處理完成後 Redis Publish
- `frontend/dist/js/first_page.js` — WebSocket 連線 + 自動重連
- `nginx/templates/conf.d/default.conf.template` — `/api/ws/` 代理設定

---

## 8. 尚未實作（TODO）

- [x] 驗證碼寄信：Worker 已接入可設定 SMTP，開發環境使用 Mailpit
- [x] 限制 `debug_code`：正式部署預設不回傳；僅 `APP_ENV=development` 且 `EXPOSE_VERIFICATION_CODE=true` 時允許
- [ ] `.env` 密碼替換：所有 `change_me_*` 需換成強密碼
- [ ] SSL 憑證：自簽憑證換成 Let's Encrypt 正式憑證
- [ ] 前端其他頁面：explore、social、personal、freq 尚未接 API
- [ ] 按讚/收藏：目前僅前端 toggle UI，未串接後端持久化
