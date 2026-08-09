# 登入密碼與驗證碼安全變更摘要

日期：2026-08-10  
狀態：已完成實作與驗證

## 變更背景

這次處理兩個認證流程問題：

1. 登入錯誤次數原本包含帳號層級的硬性封鎖。攻擊者只要持續對某個信箱送出錯誤密碼，就可能讓真正使用者即使想起正確密碼，也無法從乾淨來源登入。
2. 註冊驗證碼原本以信箱共用錯誤次數與驗證狀態。第三方可能故意輸入錯誤驗證碼，消耗或刪除真正使用者正在使用的驗證碼。

## 1. 登入密碼錯誤次數

### 決策

- 來源端的錯誤次數可以暫時阻擋該來源。
- 帳號總錯誤次數只作為風險訊號，不得在驗證密碼前阻擋其他乾淨來源。
- 使用者從乾淨來源輸入正確密碼時，必須可以登入。
- 正確登入後會清除帳號風險計數與目前來源的錯誤計數。
- 攻擊來源自己的來源端錯誤計數不會因其他人成功登入而被清除。

### 目前限制

| 範圍 | 限制 | 視窗 | 行為 |
| --- | ---: | ---: | --- |
| 信箱＋來源端 | 10 次錯誤 | 5 分鐘 | 暫時阻擋該來源 |
| 信箱帳號總計 | 25 次錯誤 | 15 分鐘 | 作為風險訊號；不提前阻擋正確密碼 |

### 安全理由

帳號層級的硬鎖容易被利用成阻斷服務攻擊。將硬性限制放在來源端，同時保留帳號總計作為風險訊號，可以減少暴力破解，又不讓攻擊者單方面鎖住受害者帳號。

## 2. 驗證碼與重新傳送

### 新 challenge 流程

1. `POST /api/auth/send-code` 每次成功寄送都會建立新的 UUID `challenge_id` 與新的 6 位數驗證碼。
2. 同一信箱先前的 challenge 與驗證碼會立即失效。
3. 前端只保存最近一次收到的 `challenge_id`。
4. 註冊時必須同時送出 email、code 與 `challenge_id`。
5. 後端會確認 `challenge_id` 是該信箱目前有效的 challenge，才會驗證 code。

### 錯誤嘗試規則

| 範圍 | 限制 | 行為 |
| --- | ---: | --- |
| challenge＋來源端 | 5 次錯誤 | 阻擋該來源繼續猜錯誤碼 |
| 整個 challenge | 20 次錯誤 | 阻擋所有來源繼續猜錯誤碼 |

正確驗證碼會在錯誤預算檢查之前比對。因此，即使攻擊者已耗盡來源端或 challenge 的錯誤次數，真正使用者只要持有正確驗證碼，仍可完成驗證。錯誤次數達上限不會刪除有效驗證碼。

### 寄送限制

| 範圍 | 每小時 | 每日 | 其他限制 |
| --- | ---: | ---: | --- |
| 收件信箱 | 5 次 | 10 次 | 每次寄送冷卻 60 秒 |
| 來源端 | 20 次 | 50 次 | 防止單一來源大量寄送至不同信箱 |

驗證碼與 challenge 有效期限均為 5 分鐘。

### 前端行為

- 重新傳送成功後，會以新的 `challenge_id` 取代舊值並清空驗證碼輸入欄位。
- 使用者修改電子信箱後，前端會清除原本的 challenge 與驗證碼。
- 沒有目前信箱對應的有效 challenge 時，註冊按鈕不會進入有效狀態。
- 驗證碼寄送成功時顯示：「驗證碼已送出，請查看最新一封信。」
- 登入被限制時顯示：「登入嘗試次數過多，請稍後再試（最長約 5 分鐘）。」
- 驗證碼寄送被限制時會說明每次需間隔 60 秒，並提醒可能已達寄送上限。
- 驗證碼不正確、過期或不是最新一封時，會提示使用者確認最新一封信。
- 驗證碼錯誤次數過多時，會提示使用最新一封的正確驗證碼或重新取得驗證碼。

## 3. API 合約變更

### `POST /api/auth/send-code`

Request：

```json
{
  "email": "amy@example.com"
}
```

Success `200`：

```json
{
  "message": "verification code sent",
  "challenge_id": "adf04b8e-9ae7-4dd5-a924-0b299a5aa865"
}
```

### `POST /api/auth/register`

Request：

```json
{
  "username": "Amy",
  "email": "amy@example.com",
  "password": "password1",
  "code": "123456",
  "challenge_id": "adf04b8e-9ae7-4dd5-a924-0b299a5aa865"
}
```

`challenge_id` 現在是必填欄位，而且必須是最近一次寄送驗證碼所回傳的 UUID。

## 4. 主要修改檔案

- `Type-WSP-deploy/api/auth.go`
- `Type-WSP-deploy/api/auth_security_test.go`
- `Type-WSP-deploy/api/auth_redis_integration_test.go`
- `Type-WSP-deploy/api/auth_session_test.go`
- `src/api/contracts.ts`
- `src/api/backendApi.ts`
- `src/api/backendApi.test.ts`
- `src/features/auth/useAuthPage.ts`
- `src/features/auth/useAuthPage.test.ts`
- `API_CONTRACT.md`

## 5. 驗證結果

| 檢查 | 結果 |
| --- | --- |
| API `go test ./...` | 通過 |
| Redis challenge／限流整合測試 | 通過 |
| API `go vet ./...` | 通過 |
| ESLint | 通過 |
| Vitest | 21 個測試檔、54 個測試全部通過 |
| Vue TypeScript 型別檢查 | 通過 |
| Vite production build | 通過 |
| `git diff --check` | 通過；僅有既有 CRLF 提示 |

Redis 整合測試使用一次性的本機 Redis 容器；測試完成後該容器已停止並自動刪除。

## 6. 尚未實作

忘記密碼尚未加入。後續應建立獨立的密碼重設 challenge、驗證碼用途與限流規則，不應重用註冊驗證碼或註冊 challenge。

## 7. Git 狀態

這次修改保留在目前工作區，尚未建立 Git commit。工作區原本已有的其他未提交修改沒有被覆蓋或重設。
