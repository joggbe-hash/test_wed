# Code Security 修復報告

- 日期：2026-08-12
- 原始掃描 ID：`e44a4cf0-87df-4bbe-9698-ef906664735d`
- 結果：三項已驗證 finding 均已修復並完成回歸測試
- 範圍：登入防暴力破解、圖片 worker CPU 公平性、multipart 暫存空間

## 修復摘要

| Finding | 原始風險 | 修復後控制 |
|---|---|---|
| Account-wide 登入限制 | 輪替 client identity 可繼續對同一帳號執行 bcrypt | Redis Lua 在 bcrypt 前原子保留 client 與 account 嘗試額度；account 15 分鐘最多 25 次 |
| 圖片 worker CPU | 單一使用者可持續補入高運算量圖片，占用兩個 worker | 每位使用者最多 1 個待處理圖片貼文；單檔上限 1,200 萬像素；每 15 分鐘最多 4,800 萬像素，Redis 原子計數 |
| Multipart `/tmp` | 合法請求約 18 MiB spill，超過 16 MiB tmpfs | 請求降至 13 MiB、單檔 3 MiB、最多 4 張、同時解析最多 4 個，API tmpfs 提高至 64 MiB |

## 1. Account-wide 登入防猜測

### 漏洞路徑

未驗證的登入 JSON 經 client-only 前置判斷後進入資料庫查詢與 bcrypt。account counter 雖有累計，但未阻止不同 client identity 繼續驗證密碼。

### 安全不變量

任何密碼驗證開始前，都必須原子取得 client 與 account 兩個預算；任一預算耗盡即回傳 HTTP 429，不得進入 bcrypt。

### 實作

- 以 Redis Lua 同時檢查並增加兩個 counter，消除「先讀取、後增加」的並行競爭窗口。
- client 視窗維持 5 分鐘 10 次；account 視窗維持 15 分鐘 25 次。
- 成功完成 email verification 後沿用既有流程清除 counter。
- Redis 錯誤時 fail closed，回傳 authentication service unavailable。

主要檔案：

- `Type-WSP-deploy/api/auth.go`
- `Type-WSP-deploy/api/auth_security_test.go`
- `Type-WSP-deploy/api/auth_redis_integration_test.go`

## 2. 圖片 worker CPU 公平配額

### 漏洞路徑

原本只計算處理後儲存空間與待處理貼文數，未計算 decoded pixels 或時間窗內累積工作量。使用者可在工作完成後持續補入圖片。

### 安全不變量

單一使用者不能同時占用兩個 worker，也不能在固定時間窗內無上限地提交 decoded-pixel 工作量；API 與 worker 必須使用同一像素上限。

### 實作

- `contracts.MaxImagePixels` 設為 12,000,000，API 入列前與 worker 解碼前均驗證。
- 每位使用者最多 1 個 processing／reserved 圖片貼文。
- 每 15 分鐘每位使用者最多 48,000,000 pixels，足以保留一個最大合法的四圖貼文。
- 像素預算使用 Redis Lua 原子保留；MinIO、DB finalization 或 queue enqueue 失敗時回滾。
- 前端同步限制為最多 4 張、每張 3 MiB，避免送出後才由 API 拒絕。

主要檔案：

- `Type-WSP-deploy/api/image_processing_budget.go`
- `Type-WSP-deploy/api/posts.go`
- `Type-WSP-deploy/shared/contracts/contracts.go`
- `Type-WSP-deploy/worker/image.go`
- `src/utils/postImages.ts`
- `src/components/ComposeModal.vue`

## 3. Multipart 暫存空間

### 漏洞路徑

`ParseMultipartForm` 的記憶體門檻為 8 MiB，原本合法的 25 MiB 請求可向 `/tmp` spill 約 18 MiB，但 API tmpfs 只有 16 MiB，而且解析前沒有 process-wide 並行限制。

### 安全不變量

`最壞單次 spill × 同時 multipart 解析數` 必須小於 API tmpfs，且 edge 與 API body limit 必須一致。

### 實作與容量模型

- API 與 nginx request body：13 MiB。
- Multipart 記憶體：4 MiB。
- 單檔：3 MiB；每篇最多 4 張。
- 每個 API process 同時最多 4 個 multipart 圖片上傳。
- 保守最壞 spill：`(13 - 4) MiB × 4 = 36 MiB`。
- API `/tmp`：64 MiB，因此仍保留約 28 MiB 緩衝。

主要檔案：

- `Type-WSP-deploy/api/image_upload_limiter.go`
- `Type-WSP-deploy/api/posts.go`
- `Type-WSP-deploy/docker-compose.yaml`
- `Type-WSP-deploy/nginx/templates/conf.d/default.conf.template`
- `Type-WSP-deploy/nginx/templates/conf.d/default.dev.conf.template`
- `Type-WSP-deploy/nginx/templates/conf/server.conf.template`

## 驗證結果

### RED → GREEN 回歸測試

- Account budget：舊程式允許 account counter 已達上限的乾淨 client；測試先失敗，修復後通過。
- Worker CPU：舊程式缺少共用像素上限、時間窗預算與單一 pending 限制；測試先編譯失敗，實作後通過。
- Multipart：舊設定缺少 concurrency limiter，且仍為 25／8／4；測試先失敗，修復後通過。
- Frontend：舊程式仍允許 4 張與 8 MiB；測試先失敗，改為 4 張與 3 MiB 後通過。

### 通過的命令

```text
cd Type-WSP-deploy/api && go test ./... -count=1
cd Type-WSP-deploy/api && go vet ./...
cd Type-WSP-deploy/worker && go test ./... -count=1
cd Type-WSP-deploy/worker && go vet ./...
cd Type-WSP-deploy/shared && go test ./... -count=1
cd Type-WSP-deploy/shared && go vet ./...
npm.cmd run verify
docker compose -f docker-compose.yaml config --quiet
```

`npm.cmd run verify` 結果：ESLint 通過、Vitest 21 files／56 tests 通過、Vue TypeScript 與 production build 通過。

### Redis 實際整合測試

使用無持久資料的本機 `redis:7-alpine` 容器執行後，容器已停止並自動刪除：

```text
TestRedisLoginAccountBudgetCannotBeResetByChangingClient           PASS
TestRedisLoginAccountBudgetIsAtomicAcrossConcurrentClients        PASS
TestRedisImageProcessingBudgetBoundsAndRollback                   PASS
```

並行測試同時送出 64 個不同 client identity，只有 account 上限允許的 25 個取得預算。

## 保留行為與相容性

- JSON 純文字貼文不使用圖片上傳 limiter。
- JPEG 與 PNG 仍受支援。
- 使用者仍可在一篇貼文上傳最多 4 張、每張最多 3 MiB 的合法圖片。
- 一個最大合法四圖貼文仍完整落在每使用者像素預算內。
- 登入仍保留 mandatory email verification；成功完成驗證後會清除登入嘗試 counter。

## 剩餘限制

- Windows/386 Go toolchain 不支援 `go test -race`；Redis Lua 的原子性改以真實 Redis 與 64 個並行請求驗證。
- 這次沒有執行完整瀏覽器 E2E；前端變更僅涉及既有純函式驗證常數與 aria-label，已通過完整 Vitest、lint、typecheck 與 build。
- 變更尚未提交或部署；部署前仍需由人員審閱差異。
