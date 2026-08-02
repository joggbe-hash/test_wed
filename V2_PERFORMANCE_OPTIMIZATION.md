# Type WSP V2 效能優化與改良建議

> 文件日期：2026-07-24  
> 專案技術：Vue 3、Vue Router、Pinia、Vite、Tailwind CSS  
> 文件性質：依目前程式碼與 production build 結果整理的效能改良清單；本文不代表項目已完成實作。

## 1. 目標

V2 效能優化的主要目標如下：

1. 縮短登入頁與主要頁面的首次載入時間。
2. 降低頁面切換、動態牆與探索頁互動時的繪製成本。
3. 避免重複 API 請求、重複事件監聽與不必要的響應式更新。
4. 讓貼文、任務、提醒與探索資料增加後，效能仍能維持穩定。
5. 建立可重複量測的驗收基準，避免只憑主觀感覺判斷優化成效。

## 2. 目前架構與責任範圍

| 區域 | 主要檔案 | 責任 |
|---|---|---|
| 應用程式入口 | `src/App.vue`、`src/main.ts` | 路由內容、全域彈窗、Pinia 與 Router 掛載 |
| 路由與驗證 | `src/router/index.ts`、`src/composables/useSession.ts` | 頁面載入、登入狀態還原、路由保護 |
| 共用版面 | `src/layouts/MainLayout.vue`、`src/components/AppNavbar.vue` | 導覽列、側欄、主內容與浮動按鈕 |
| 動態牆 | `src/stores/useFeedStore.ts`、`src/components/XPostCard.vue` | 貼文快取、貼文列表與時間更新 |
| 探索頁 | `src/views/ExplorePage.vue` | 搜尋、分類、橫向卡片與拖曳捲動 |
| 任務與提醒 | `src/composables/useScheduleMock.ts`、`src/components/SidebarWidgets.vue` | 本機資料、日期更新、排序與側欄互動 |
| 樣式與動畫 | `src/style.css` | 全域版面、元件樣式、轉場與動畫 |

## 3. Production build 基準

執行指令：

```bash
npm run build
```

建置成功，共轉換 116 個 modules。

| 產物 | 原始大小 | gzip 大小 | 備註 |
|---|---:|---:|---|
| `index.html` | 1.38 kB | 0.75 kB | 正常 |
| 單一 CSS chunk | 279.15 kB | 30.38 kB | 所有主要樣式集中載入 |
| 單一 JS chunk | 240.66 kB | 81.75 kB | 所有路由與多個低頻彈窗集中在首次載入 |
| 兩張登入背景 JPEG | 472.6 kB | 不適用 | 1440 × 1024，分別為 228.3 kB 與 244.3 kB |

目前 `dist/assets` 只有一個 JS 與一個 CSS 檔案，代表尚未形成有效的頁面級程式碼分割。

## 4. 優先級總覽

| 優先級 | 改良項目 | 主要影響 | 建議時程 |
|---|---|---|---|
| P0 | 路由與低頻彈窗延遲載入 | 首次載入、解析與執行時間 | 第一階段 |
| P0 | 移除大面積 `filter: blur()` 與持續陰影動畫 | 頁面切換、捲動、低階裝置流暢度 | 第一階段 |
| P0 | 重構探索頁捲動事件 | 搜尋輸入與橫向拖曳流暢度 | 第一階段 |
| P1 | 動態牆請求去重、分頁與取消機制 | API 負載、切頁穩定性、長列表 | 第二階段 |
| P1 | 合併時鐘與日期計時器 | 背景 CPU、響應式更新次數 | 第二階段 |
| P1 | 將頁面專用 CSS 移回元件或頁面 | CSS 首次載入與維護成本 | 第二階段 |
| P2 | 圖片格式、尺寸與載入策略 | 登入頁 LCP、流量 | 第三階段 |
| P2 | 本機儲存與大型陣列策略 | 任務資料成長後的主執行緒阻塞 | 第三階段 |

## 5. 詳細改良項目

### 5.1 P0：路由與低頻彈窗延遲載入

#### 現況

`src/router/index.ts:2-11` 同步匯入全部 10 個頁面。`src/App.vue:5-8` 也同步匯入發文、每日任務、舊資料匯入與介紹頁等低頻內容。

因此，即使使用者只開啟登入頁，瀏覽器仍需下載、解析並執行其他頁面的程式碼。production build 也證實所有程式集中在單一 240.66 kB JS chunk。

#### 建議

- 保留預設登入頁為同步載入。
- 需要登入後才會使用的頁面改為動態 `import()`。
- 發文與大型彈窗使用 `defineAsyncComponent()`，只在第一次開啟時下載。
- 對最常使用的 `/home`，可在登入成功後或導覽連結 hover/focus 時預先載入。

```ts
import LoginPage from '../views/LoginPage.vue'

const FirstPage = () => import('../views/FirstPage.vue')
const ExplorePage = () => import('../views/ExplorePage.vue')
const PersonalPage = () => import('../views/PersonalPage.vue')
const FreqPage = () => import('../views/FreqPage.vue')
```

```ts
import { defineAsyncComponent } from 'vue'

const ComposeModal = defineAsyncComponent(
  () => import('./components/ComposeModal.vue'),
)
```

#### 驗收

- build 產物不再只有單一 JS chunk。
- 登入頁首次載入不包含已登入頁面的程式碼。
- 初始 JS gzip 由目前 81.75 kB 降至 60 kB 以下；實際數字以改完後 build 為準。
- 第一次開啟非同步頁面或彈窗時有 loading 狀態，且沒有版面跳動。

### 5.2 P0：降低大面積動畫的 layout／paint 成本

#### 現況

- `src/App.vue:40` 的頁面轉場使用 `mode="out-in"`，舊頁完成 350 ms 離場後才建立新頁，會直接增加使用者感受到的切頁等待。
- `src/style.css:2577-2594` 對整個路由內容同時動畫 `opacity`、`transform` 與 `filter: blur(4px)`。
- `src/style.css:1180` 對每張貼文執行 520 ms 的模糊進場動畫；貼文同時出現時會產生多個 paint 工作。
- `src/style.css:2252` 的 FAB 持續動畫 `box-shadow`，即使使用者沒有互動也會反覆重繪。
- 專案已有 `prefers-reduced-motion` 規則，無障礙降級處理是正確的，可保留。

#### 建議

1. 頁面與貼文動畫只使用 `transform` 與 `opacity`。
2. 移除全頁與列表項目的 `filter: blur()`。
3. 將頁面轉場縮短至約 150–200 ms；若產品允許，移除 `mode="out-in"` 的串行等待。
4. FAB 改為靜態陰影；若仍需提示效果，使用短時間、單次 `transform` 動畫。
5. 動態牆只為前幾張可見貼文播放一次進場動畫，不要讓整份長列表同時播放。

```css
.page-slide-enter-active,
.page-slide-leave-active {
  transition:
    opacity 180ms ease,
    transform 180ms ease;
}

.page-slide-enter-from {
  opacity: 0;
  transform: translateY(8px);
}

.page-slide-leave-to {
  opacity: 0;
  transform: translateY(-6px);
}
```

#### 驗收

- 全頁與貼文動畫不再包含 `filter`。
- 沒有無限循環的 `box-shadow` 動畫。
- 快速切換頁面時不會因 350 ms 串行轉場產生明顯等待。
- 使用瀏覽器 Performance 面板檢查動畫期間，沒有連續的大面積 Paint。

### 5.3 P0：重構探索頁橫向捲動事件

#### 現況

`src/views/ExplorePage.vue:50-93` 透過 `document.querySelectorAll()` 找出全部橫向列表，並為每一列註冊：

- `mousedown`
- 一組掛在 `window` 的 `mouseup`
- 一組掛在 `window` 的 `mousemove`
- 非 passive 的 `wheel`

`src/views/ExplorePage.vue:129-132` 又監聽 `filteredRows`。使用者每次輸入搜尋文字後，都會等待 DOM 更新、移除所有處理器，再重新查詢 DOM 與綁定事件。

當探索列數增加時，搜尋與滑鼠移動成本會隨列數一起增加。

#### 建議

- 優先使用原生 `overflow-x: auto`、觸控捲動與 `scroll-snap`。
- 仍需滑鼠拖曳時，改用 Pointer Events 與 `setPointerCapture()`，事件只綁在正在拖曳的元素。
- 以 template ref 或小型 `HorizontalScroller` 元件管理生命週期，不使用全域 `querySelectorAll()`。
- 不要因篩選結果變動而重綁整批事件。
- 若保留 wheel 轉橫向功能，只在確實需要攔截時 `preventDefault()`，避免所有 wheel 事件都成為阻塞式監聽。

#### 驗收

- 搜尋輸入不再觸發整批事件解除與重綁。
- 不再出現「每一列各有一組 `window.mousemove`」的情況。
- 新增或移除探索列後，不需手動呼叫 `bindHorizontalScrollers()`。
- 滑鼠、觸控板、鍵盤與觸控裝置皆可正常橫向瀏覽。

### 5.4 P1：動態牆請求去重、取消與分頁

#### 現況

- `src/stores/useFeedStore.ts:11-24` 有 `isLoaded` 快取，但沒有共用 pending Promise。
- 請求進行中切換到另一個會載入動態牆的頁面時，第二個頁面仍可能再次呼叫 `fetchFeed()`。
- `src/api/backendApi.ts:21` 已定義 `next_cursor`，但 store 尚未保存或使用。
- `src/api/backendApi.ts` 的共用 fetch 尚未接受 `AbortSignal` 或 timeout。

#### 建議

1. Store 保存 `pendingLoad`，相同資源同時間只允許一個請求。
2. API 層接受 `signal`，頁面卸載、搜尋條件變更或逾時時取消舊請求。
3. 保存 `nextCursor` 與 `hasMore`，採游標分頁。
4. 搜尋 API 上線後加入 200–300 ms debounce，並取消上一個搜尋。
5. 貼文約 50–100 筆以上且卡片內容複雜時，再評估虛擬列表；目前 API 若固定 20 筆，不需過早導入。

```ts
let pendingLoad: Promise<void> | null = null

async function loadPosts(forceRefresh = false) {
  if (isLoaded.value && !forceRefresh) return
  if (pendingLoad) return pendingLoad

  pendingLoad = fetchFeed()
    .then((response) => {
      posts.value = response.posts
      nextCursor.value = response.next_cursor
      isLoaded.value = true
    })
    .finally(() => {
      pendingLoad = null
    })

  return pendingLoad
}
```

#### 驗收

- 同一時間相同 feed 只出現一筆網路請求。
- 快速切頁不會留下不再需要的請求。
- 下一頁使用 `next_cursor`，不重抓已取得的貼文。
- 以 100 筆測試資料測試捲動與新增／刪除貼文，沒有明顯長任務。

### 5.5 P1：合併時鐘與日期計時器

#### 現況

目前可找到四個 interval：

| 位置 | 間隔 | 用途 |
|---|---:|---|
| `src/components/AppNavbar.vue:25` | 1 秒 | 更新只顯示到「分鐘」的時間 |
| `src/composables/useSharedMinuteNow.ts:14` | 1 分鐘 | 更新貼文相對時間 |
| `src/components/SidebarWidgets.vue:410` | 1 分鐘 | 更新日曆日期 |
| `src/composables/useScheduleMock.ts:217` | 1 分鐘 | 更新今日 key |

Navbar 每分鐘觸發約 60 次響應式更新，但畫面只顯示小時與分鐘。日曆與排程也分別維護功能相近的計時器。

#### 建議

- Navbar 改用既有的 `useSharedMinuteNow()`。
- 共用一個 minute tick，從它衍生目前時間、今日日期與貼文相對時間。
- `Intl.DateTimeFormat` 實例建立在模組層或元件初始化時，不要在每次 computed 重算時重新建立。
- 使用 `visibilitychange`：頁面隱藏時暫停，回到前景時立即校正。
- 計時器對齊下一個整分鐘，避免進站秒數造成顯示延遲。

#### 驗收

- Navbar 不再每秒更新。
- 正常頁面只保留一個共用分鐘級 tick。
- 分頁切到背景後不持續做不必要的時間更新。
- 回到前景後，時間與今日日期立即正確。

### 5.6 P1：拆分全域 CSS，讓頁面樣式可按需載入

#### 現況

`src/style.css` 約 2,700 行、93.9 kB 原始碼，包含約 556 個 `@apply`。production 展開後成為 279.15 kB 的單一 CSS 檔。

目前大部分頁面與元件樣式集中在全域檔案；即使路由改為動態載入，頁面專用 CSS 仍會在首次進站全部下載。

#### 建議

- `src/style.css` 只保留 reset、design tokens、排版與真正共用的 layout/utilities。
- 頁面或元件專用樣式移入對應 `.vue` 的 `<style scoped>`。
- 將登入、探索、設定、動態牆、任務與彈窗樣式分開。
- scoped CSS 優先使用 class selector，避免成本較高且容易外溢的元素 selector。
- 清查重複 modal backdrop、modal panel、button 與 card 規則，保留共用基底與少量變體。

#### 驗收

- 路由 lazy loading 後，可在 build 產物看到頁面級 CSS chunk。
- 登入頁不載入動態牆、探索與任務頁專用樣式。
- 初始 CSS gzip 由目前 30.38 kB 降至 25 kB 以下；實際數字以改完後 build 為準。
- 視覺回歸測試確認主要頁面、彈窗與響應式版面沒有變化。

### 5.7 P2：圖片與字型載入策略

#### 現況

- `meme_background.jpg`：1440 × 1024，228.3 kB。
- `register_round.jpg`：1440 × 1024，244.3 kB。
- 貼文圖片已使用 `loading="lazy"` 並提供寬高，這部分可保留。
- 首頁載入兩套 Google Fonts CSS；字型顯示與外部連線速度仍會影響首屏。

#### 建議

- 登入背景輸出 WebP／AVIF，JPEG 作 fallback。
- 依桌機與手機提供不同尺寸，避免手機下載 1440 px 圖片。
- 首屏必要背景可預載；註冊頁專用圖片應在切換到註冊模式後才載入。
- 確認字型使用 `font-display: swap`；若上線環境需穩定效能，可評估自架並只保留實際使用的字重與字形。
- 使用者上傳圖片由後端產生縮圖與現代格式，列表先載縮圖，圖片檢視器再載原圖。

#### 驗收

- 登入頁不會在使用者未開啟註冊模式時先下載註冊背景。
- 手機下載的背景尺寸不超過實際顯示需求。
- 圖片最佳化後，以相同視覺品質比較傳輸大小。
- 貼文列表不直接下載每張原始大圖。

### 5.8 P2：本機儲存與大型資料的響應式策略

#### 現況

`src/composables/useScheduleMock.ts:164` 每次新增、更新、勾選、刪除與重新排序，都會同步執行：

```ts
localStorage.setItem(key, JSON.stringify(schedule))
```

`localStorage` 與 `JSON.stringify()` 都在主執行緒同步執行。資料量小時影響有限，但任務與提醒持續累積後，拖曳排序或快速勾選可能產生卡頓。

此外，feed、探索 rows、tasks 與 reminders 使用深層響應式陣列。若資料量增大且採整體替換，可評估 `shallowRef()` 搭配不可變更新，減少深層 Proxy 成本。

#### 建議

- 小型更新先留在記憶體，使用短 debounce 合併多次寫入。
- 在 `pagehide` 或明確儲存時 flush 尚未寫入的資料。
- 資料量明顯增長或需要索引查詢時，改用 IndexedDB。
- 只有在量測證實深層 Proxy 是瓶頸時，再將大型資料改為 `shallowRef()`；不要為小陣列增加不必要複雜度。

#### 驗收

- 連續拖曳或快速勾選不會對每個中間狀態各寫入一次完整資料。
- 重新整理或關閉分頁前，最後狀態仍能正確保存。
- 以大量測試資料驗證序列化時間與 UI 回應性。

## 6. 建議實作順序

### 第一階段：立即改善首屏與互動

1. 將登入以外路由改為動態載入。
2. 將低頻彈窗改為非同步元件。
3. 移除頁面與貼文的 blur 動畫。
4. 停止 FAB 無限 box-shadow 動畫。
5. 重構探索頁捲動事件。
6. 重新執行 production build 並保存比較數據。

### 第二階段：改善資料流與背景工作

1. Feed 請求去重與 AbortSignal。
2. 完成 `next_cursor` 分頁。
3. 合併分鐘計時器。
4. 拆分全域 CSS。
5. 以 100 筆貼文與大量探索卡片執行壓力測試。

### 第三階段：資產與資料成長

1. 背景圖輸出 WebP／AVIF 與響應式尺寸。
2. 上傳圖片導入縮圖。
3. 合併 localStorage 寫入；必要時導入 IndexedDB。
4. 依量測結果決定是否使用虛擬列表或 shallow reactivity。

## 7. 驗收與量測清單

每次效能變更都應保留「修改前／修改後」資料：

| 類別 | 量測方式 | 本次基準 | 建議目標 |
|---|---|---:|---:|
| 初始 JS | Vite build gzip | 81.75 kB | ≤ 60 kB |
| 初始 CSS | Vite build gzip | 30.38 kB | ≤ 25 kB |
| 初始 chunk | `dist/assets` | 1 JS + 1 CSS | 形成核心與頁面 chunk |
| 頁面轉場 | CSS duration | 350 ms 且 `out-in` | 150–200 ms，避免 blur |
| Navbar 更新 | interval | 每秒一次 | 每分鐘一次 |
| Feed 重複請求 | Network 面板 | 尚無 pending 去重 | 同資源同時間一筆 |
| 探索事件 | Event Listeners／Performance | 每列各自註冊全域 move/up | 無每列全域 listener |
| 長列表 | 100 筆測試資料 | 尚未建立固定測試 | 捲動與更新無明顯長任務 |

建議至少測試以下情境：

1. 清除快取後首次進入登入頁。
2. 登入後第一次進入首頁、探索頁與設定頁。
3. 首頁、社群頁、個人頁之間快速切換。
4. 探索頁持續輸入與刪除搜尋文字，同時拖曳橫向列表。
5. 100 筆貼文含圖片時持續捲動。
6. 分頁切到背景 5 分鐘後返回。
7. 模擬慢速網路時打開非同步頁面與彈窗。

## 8. 不建議過早進行的優化

- 目前列表只有約 20 筆時，不需要立即加入虛擬列表套件。
- 不要為了幾個 primitive state 大量重寫現有 Composition API；優先處理 chunk、動畫、事件與網路請求。
- 不要在沒有量測前任意加入 `will-change`，大量 layer promotion 可能增加記憶體。
- 不要同時混用多套動畫函式庫；現有 CSS 動畫即可用較低成本的 properties 改良。
- 不要先拆出大量只有一層 DOM 的包裝元件；列表熱路徑應維持扁平。

## 9. 完成定義

V2 效能優化可視為完成，需同時符合：

- production build、TypeScript 檢查通過。
- 首次載入已形成路由／功能 chunk。
- 全頁與貼文進場沒有 blur 或持續 paint-heavy 動畫。
- 探索頁不再因搜尋結果變動重綁全部事件。
- Feed 具備請求去重、取消與 cursor 分頁。
- 分鐘級資訊共用單一時間來源。
- 已保存修改前後的 build 與瀏覽器 Performance 比較結果。
- 主要操作、鍵盤導覽、觸控捲動及 reduced-motion 行為皆未退化。
