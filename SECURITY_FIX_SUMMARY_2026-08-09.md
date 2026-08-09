# Codex Security 修補內容

日期：2026-08-09  
掃描 ID：`9eabea6c-92a5-46b2-a759-40b93eafc140`  
結果：五項已確認的安全問題皆已修補。

## 1. 寫入 API 缺少 CSRF 防護

### 原始問題

登入、登出、貼文及靈感等狀態變更端點沒有套用既有的 browser-request guard，瀏覽器可能送出跨站簡單請求來改變使用者資料或 Session。

### 修改內容

- 所有 `POST`、`PUT`、`PATCH`、`DELETE` API 路由統一套用 `requireBrowserRequest`。
- 前端共用 API client 固定加入 `X-Type-WSP-Request: 1`，呼叫端不能覆寫此安全標頭。
- JSON request 必須使用 `Content-Type: application/json`。
- JSON body 只能包含單一 JSON value，拒絕尾隨資料。
- multipart 圖片上傳仍維持原有行為。

### 主要檔案

- `Type-WSP-deploy/api/main.go`
- `Type-WSP-deploy/api/request_security.go`
- `Type-WSP-deploy/api/request_security_test.go`
- `src/api/backendApi.ts`
- `src/api/backendApi.test.ts`
- `API_CONTRACT.md`

## 2. 登入節流可被用來鎖住其他帳號

### 原始問題

登入嘗試在驗證密碼前就增加以 email 為單位的 Redis counter。攻擊者可持續對指定 email 輸入錯誤密碼，使真正使用者即使提供正確密碼仍被阻擋。

### 修改內容

- 先查詢帳號並執行固定成本的密碼比較，再處理失敗計數。
- 只有錯誤密碼會增加 failure budget。
- Redis key 改為雜湊後的「正規化 email＋來源 client identity」。
- 正確密碼不受已耗盡的失敗額度影響，登入成功後會清除該來源的 counter。
- Client identity 優先採用 Nginx 覆寫的有效 `X-Real-IP`，否則安全地退回連線位址。

### 主要檔案

- `Type-WSP-deploy/api/auth.go`
- `Type-WSP-deploy/api/auth_security_test.go`
- `Type-WSP-deploy/api/request_security.go`

## 3. 未登入攻擊者可讓他人的驗證碼失效

### 原始問題

同一 email 累積五次錯誤驗證碼後，後端會刪除仍然有效的 challenge。攻擊者因此可以反覆阻止其他使用者完成註冊。

### 修改內容

- 錯誤嘗試達上限時不再刪除有效驗證碼。
- Challenge 只會在成功使用或 TTL 到期後失效。
- 嘗試次數以「正規化 email＋client identity＋驗證碼版本」隔離。
- 新寄出的驗證碼不會繼承舊 challenge 的失敗次數。

### 主要檔案

- `Type-WSP-deploy/api/auth.go`
- `Type-WSP-deploy/api/auth_security_test.go`

## 4. 處理完成的圖片沒有累積儲存上限

### 原始問題

原本只限制同時處理中的圖片貼文數量。圖片完成後不再計入任何 owner quota，已登入使用者可以持續建立 MinIO 物件與資料庫紀錄。

### 修改內容

- 每位使用者的處理後圖片儲存上限設為 `1 GiB`。
- 建立圖片貼文時，以每張圖片最大 `16 MiB` 預留容量。
- 在 PostgreSQL per-user advisory transaction lock 內重新檢查容量並寫入預留值，避免並行請求超額。
- Worker 完成處理時，以實際輸出 byte 數取代預留值。
- 處理失敗時清除預留與實際用量。
- 新增 `image_reserved_bytes` 與 `image_storage_bytes` 欄位、非負限制及查詢索引。
- 遷移時對既有 ready 圖片採保守最大值回填，避免低估既有使用量。
- 超過配額時 API 回傳 `429 image storage quota exceeded`。

### 主要檔案

- `Type-WSP-deploy/api/posts.go`
- `Type-WSP-deploy/api/posts_test.go`
- `Type-WSP-deploy/api/migrations/system/004_image_storage_quota.sql`
- `Type-WSP-deploy/api/migrations_test.go`
- `Type-WSP-deploy/shared/contracts/contracts.go`
- `Type-WSP-deploy/worker/image.go`
- `Type-WSP-deploy/worker/image_test.go`
- `API_CONTRACT.md`

## 5. Production 可使用明文 SMTP

### 原始問題

Production worker 接受 `SMTP_SECURE=false`，註冊驗證碼可能在 worker 與 SMTP relay 之間以明文傳輸。

### 修改內容

- `APP_ENV=production` 時強制 `SMTP_SECURE=true`。
- Production 明文 SMTP 設定會在 worker 啟動載入設定時直接失敗。
- Development 與 test 環境仍可使用 Mailpit 等本機非 TLS relay。
- README 已標示 production 限制。

### 主要檔案

- `Type-WSP-deploy/worker/config.go`
- `Type-WSP-deploy/worker/config_test.go`
- `README.md`

## Nginx 檢查

重新檢查目前 Nginx templates，以下項目均為有效指令而非被註解吞掉的文字：

- 一般 API rate-limit zone。
- 登入與驗證碼端點的嚴格 rate-limit zone。
- Per-IP connection zone。
- `/api/auth/(login|register|send-code)` location block。

本次不需要額外產生 Nginx diff。

## 新增或擴充的安全測試

- 寫入路由缺少 browser-request header 時必須回傳 `403`。
- 前端所有 API 呼叫必須帶安全標頭。
- JSON media type 與單一 value 限制。
- 登入 attempt key 的帳號／來源隔離。
- 正確密碼可略過失敗額度。
- 驗證碼 attempt key 的來源／challenge 版本隔離。
- 錯誤驗證不會讓 challenge 失效。
- 圖片配額邊界、超額與超大預留測試。
- Migration 必須包含持久化容量欄位。
- Worker 實際圖片 byte 加總測試。
- Production 明文 SMTP 拒絕測試。

## 驗證結果

| 驗證 | 結果 |
| --- | --- |
| API `go test ./...` | 通過 |
| Worker `go test ./...` | 通過 |
| Shared `go test ./...` | 通過 |
| API／Worker／Shared `go vet ./...` | 通過 |
| ESLint | 通過 |
| Vitest | 18 個測試檔、44 個測試全部通過 |
| Vue TypeScript check | 通過 |
| Vite production build | 通過 |
| Docker Compose config validation | 通過 |
| `git diff --check` | 通過；僅顯示既有 `src/App.vue` 換行格式警告 |

## 部署注意事項

1. 部署 API 新版本前必須執行 system database migration `004_image_storage_quota.sql`。API 啟動流程會自動執行 migration。
2. Production worker 的 `SMTP_SECURE` 必須設為 `true`，並確認 SMTP relay 支援 TLS／STARTTLS。
3. 遷移會保守計算既有圖片容量，部分既有使用者的計量可能高於實際 byte 數，但不會低估儲存使用量。
4. 正式部署後仍建議執行 PostgreSQL、Redis、MinIO、SMTP 與 Nginx 的整合 smoke test。

## 工作區狀態

- 未建立 Git commit。
- 未移除或覆寫工作區內原本存在的其他修改。
- 完整 Codex Security 修補證據另存於原掃描目錄的 `artifacts/fix_report.md`。
