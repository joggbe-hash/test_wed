# 前後端交接改善修改紀錄

日期：2026-08-04

本次修改處理不需要新增大型資料庫功能、可直接在現有架構上完成的交接問題。任務、提醒、靈感等正式後端化需要另外設計資料表與同步策略，因此沒有混入本次變更。

## 1. API 合約集中管理

新增：

- `API_CONTRACT.md`：記錄 endpoint、request、response、HTTP status、Cookie、enum、分頁與上傳限制。
- `src/api/contracts.ts`：集中前端共用 API TypeScript 型別。

調整：

- `src/api/backendApi.ts` 不再重複宣告 User、Post、Feed 等資料型別。
- `ImageStatus` 收緊為 `none | processing | ready | failed`。
- 共用 response 型別包含 `ApiMessageResponse`、`CreatePostResponse`。

後續若修改 API，必須同步更新：

1. `API_CONTRACT.md`
2. Go handler／response struct
3. `src/api/contracts.ts`
4. 對應測試

## 2. 圖片選擇與上傳驗證

新增：

- `src/utils/postImages.ts`
- `src/utils/postImages.test.ts`

前端現在會在送出前檢查：

- 最多 4 張圖片。
- 僅支援 JPEG、PNG。
- 每張圖片最多 8 MiB。
- 錯誤時清空 file input 並顯示可理解的訊息。

`ComposeModal.vue` 的 file input 已從 `accept="image/*"` 改為只接受 `image/jpeg,image/png`。

後端仍保留原有驗證，前端驗證只用於提早回饋，不能取代後端安全檢查。

## 3. 圖片處理狀態

新增：

- `src/api/postProcessing.ts`
- `src/api/postProcessing.test.ts`

修改流程：

- 舊流程會讓發文 Modal 每秒抓取完整 Feed，最多等待 20 秒後才關閉。
- 新流程建立貼文後立即重新整理 Feed、關閉 Modal，再由 Pinia Store 在背景追蹤狀態。
- 輪詢間隔調整為 1、2、3、5、8 秒，降低 API request 數量。
- 登出或 Store reset 時會取消仍在執行的輪詢。
- 同一篇貼文重複啟動追蹤時會取消舊任務。

`XPostCard.vue` 新增 `failed` 狀態提示：圖片處理失敗時不會再停留在不明狀態。

目前後端 WebSocket 只有 `post_ready` 通知，沒有 `post_failed`；因此這次保留輪詢作為完整狀態確認方式。若未來補齊 WebSocket event，可再改為 WebSocket 優先、輪詢 fallback。

## 4. Feed Cursor 分頁

修改：

- `src/stores/useFeedStore.ts`
- `src/stores/useFeedStore.test.ts`
- `src/components/FeedLoadMoreButton.vue`
- `src/views/FirstPage.vue`
- `src/views/SocialPage.vue`
- `src/views/PersonalPage.vue`
- `src/style.css`

Store 新增：

- `nextCursor`
- `hasMore`
- `isLoadingMore`
- `loadMore()`

行為：

- 第一頁載入後保存後端 `next_cursor`。
- 點擊「載入更多」時將 cursor 原樣傳回後端。
- 新頁面資料 append 到現有貼文。
- 依貼文 ID 去除重複資料。
- `next_cursor` 為空時不再顯示載入按鈕。
- refresh／logout 時重設 cursor。

## 5. 統一 API 錯誤訊息

新增：

- `src/api/errors.ts`
- `src/api/errors.test.ts`

目前統一處理：

- `400`：輸入資料不正確。
- `401`：登入狀態失效。
- `403`：沒有操作權限。
- `404`：找不到資料。
- `409`：資料衝突。
- `413`：上傳超過限制。
- `429`：操作太頻繁。
- `500`：伺服器錯誤。
- `503`：服務忙碌。
- Network `TypeError`：無法連線到伺服器。

登入／註冊、發文與主要 Feed 頁面已改用共用轉換函式，避免直接將不穩定的後端英文訊息顯示給使用者。

## 6. 本機前後端串接

修改：

- `vite.config.ts`
- `src/env.d.ts`
- `.env.example`
- `README.md`

新增環境變數：

```dotenv
VITE_DEV_API_TARGET=http://127.0.0.1:5000
```

設定後 Vite 會代理：

- `/api/*` HTTP request
- `/api/ws/` WebSocket upgrade

瀏覽器端仍呼叫同源 `/api/*`，不需要在前端處理 CORS。

## 7. Session Cookie 開發／正式環境區分

修改：

- `Type-WSP-deploy/api/auth.go`
- `Type-WSP-deploy/api/auth_session_test.go`

Cookie 規則：

- production：`Secure=true`
- development／test：`Secure=false`，允許本機 HTTP + Vite proxy 開發
- 所有環境仍保留 `HttpOnly=true`、`SameSite=Lax`、`Path=/`
- 登出清除 Cookie 時使用與登入相同的 Cookie 屬性

正式環境安全設定沒有放寬。

## 8. Vue 結構調整

- `ComposeModal.vue` 保留表單與使用者互動責任。
- 圖片驗證拆成純工具函式。
- 圖片非同步處理拆到 API service 與 Pinia Store。
- Feed 載入更多按鈕拆成獨立、typed props／emits 元件。
- primitive state 改用 `shallowRef`，陣列保留 `ref`。
- DOM input ref 改用 Vue 3.5 `useTemplateRef`。

## 9. 新增與更新測試

新增測試涵蓋：

- JPEG／PNG 圖片驗證。
- 圖片數量與單檔大小限制。
- API status 對使用者訊息的轉換。
- 圖片背景處理狀態追蹤。
- Feed cursor 分頁與貼文去重。
- development／production Cookie Secure 規則。

## 10. 驗證結果

本次完成後執行：

```text
npm run verify
  ESLint                 通過
  Vitest                 11 files / 23 tests 通過
  vue-tsc + Vite build   通過

npm run test:e2e
  Playwright Chromium    2 tests 通過

Type-WSP-deploy/api
  go test ./...          通過
  go vet ./...           通過

Type-WSP-deploy/worker
  go test ./...          通過
  go vet ./...           通過

Type-WSP-deploy/shared
  go test ./...          通過
  go vet ./...           通過
```

`git diff --check` 亦通過，沒有空白錯誤。

## 11. 本次未修改的後續工作

以下項目需要新的產品／資料庫決策，應另開功能工作：

- 忘記密碼與重設密碼 API。
- 任務、提醒由 localStorage 遷移到後端。
- 靈感資料由 localStorage 遷移到後端。
- Explore、Freq、個人資料正式 API。
- WebSocket `post_failed` event。
- 使用 OpenAPI 自動產生 TypeScript client，完全取代人工型別同步。
- 擴充 Feed 與圖片處理的 Playwright E2E 覆蓋。
