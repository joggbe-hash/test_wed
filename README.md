# Joggbe 專案啟動說明

這份文件給團隊成員在另一台電腦第一次抓專案、啟動網站、修改後上傳使用。

## 1. 第一次抓專案

```powershell
git clone https://github.com/joggbe-hash/test_wed.git
cd test_wed
git switch V1-Version
npm install
```

## 2. 只開前端開發版

```powershell
npm run dev
```

Vite 會顯示本機網址，通常是：

```text
http://localhost:5173/
```

## 3. 開 Docker 全端版

這會在目前這台電腦用 Docker 開一整套本機服務：

- `nginx`：對外網站入口
- `api`：Go 後端
- `worker`：圖片處理與背景工作
- `postgres`：本機資料庫
- `redis`：session、驗證碼、工作佇列
- `minio`：本機圖片儲存

資料會存在這台電腦自己的 Docker volume，不會自動連到其他人的資料庫。

第一次啟動前先建立本機環境變數檔：

```powershell
cd Type-WSP-deploy
copy .env.example .env
cd ..
```

先把前端 build 到 Docker nginx 使用的資料夾：

```powershell
npm run build:backend
```

再啟動後端、資料庫、Redis、MinIO、nginx：

```powershell
cd Type-WSP-deploy
docker compose up --build
```

啟動後瀏覽：

```text
https://127.0.0.1/social
```

如果瀏覽器跳出憑證警告，這是本機自簽憑證造成的，開發環境可以先允許繼續。

## 4. 每次開始改之前

先確認在展示分支，並拉 GitHub 最新版：

```powershell
git switch V1-Version
git pull origin V1-Version
```

## 5. 改完後上傳

```powershell
git status
git add .
git commit -m "寫清楚這次改了什麼"
git push origin V1-Version
```

## 6. 注意事項

- 不要提交 `node_modules/`。
- 不要提交 `.env` 或密碼。
- 如果只改前端原始碼，通常需要重新跑 `npm run build:backend`，Docker 版才會更新。
- 如果多人同時改同一個檔案，`git pull` 時可能會出現衝突，需要先解完衝突再 commit。
