# CHANGELOG — Type-WSP 合併部署與 API 建置

> 記錄 2026-05-24 ~ 2026-05-25

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
| POST | /api/auth/login | 否 | 登入（建立 Redis session + cookie） |
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

- [ ] 驗證碼寄信：目前 Worker 僅 log 輸出，需接入 SMTP / SendGrid / AWS SES
- [ ] 移除 `debug_code`：正式上線前移除 API 回傳的驗證碼明文
- [ ] `.env` 密碼替換：所有 `change_me_*` 需換成強密碼
- [ ] SSL 憑證：自簽憑證換成 Let's Encrypt 正式憑證
- [ ] 前端其他頁面：explore、social、personal、freq 尚未接 API
- [ ] 按讚/收藏：目前僅前端 toggle UI，未串接後端持久化
