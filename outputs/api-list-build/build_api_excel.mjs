import fs from "node:fs/promises";
import path from "node:path";
import { SpreadsheetFile, Workbook } from "@oai/artifact-tool";

const outputDir = path.resolve("outputs", "api-list");
await fs.mkdir(outputDir, { recursive: true });

const workbook = Workbook.create();
const sheet = workbook.worksheets.add("API列表");
sheet.showGridLines = false;

const headers = ["版本", "優先順序", "Method", "API", "功能", "Request", "Response", "備註"];

const rows = [
  ["V1 (MVP 核心)", 1, "POST", "/api/auth/send-code", "發送驗證碼到信箱", '{"email": "..."}', "發送成功 / 失敗狀態", "已規劃"],
  ["V1 (MVP 核心)", 2, "POST", "/api/auth/register", "會員註冊 (含驗證碼核對)", '{"username", "email", "password", "code"}', "註冊成功 / 失敗狀態", "已規劃"],
  ["V1 (MVP 核心)", 3, "POST", "/api/auth/login", "會員登入", '{"email", "password"}', "token, user資料", "已規劃"],
  ["V1 (MVP 核心)", 4, "POST", "/api/posts", "發布貼文 (純文字)", '{"content": "..."}，Header Token', "post_id, 發布成功狀態", "已規劃"],
  ["V1 (MVP 核心)", 5, "GET", "/api/feed", "讀取動態牆 (最新20筆)", "Header Token", "貼文陣列，包含 author 資訊", "已規劃"],

  ["V2 新增", 1, "GET", "/api/users/me", "取得自己的個人資料", "Header Token", "user 資料", "第一階段：個人頁"],
  ["V2 新增", 2, "PATCH", "/api/users/me", "編輯個人資料", '{"username", "bio", "avatar_url"}', "更新成功 / user 資料", "第一階段：個人頁"],
  ["V2 新增", 3, "GET", "/api/users/:id", "查看他人個人頁", "user_id", "user 公開資料", "第一階段：個人頁"],
  ["V2 新增", 4, "GET", "/api/users/:id/posts", "查看某使用者貼文", "user_id", "貼文陣列", "第一階段：個人頁"],
  ["V2 新增", 5, "POST", "/api/media/upload", "上傳頭像 / 封面 / 貼文圖片", "form-data file，Header Token", "image_url", "第一階段：圖片"],

  ["V2 新增", 6, "PATCH", "/api/posts/:id", "編輯貼文", '{"content": "..."}，Header Token', "更新成功 / 失敗", "第二階段：貼文互動"],
  ["V2 新增", 7, "DELETE", "/api/posts/:id", "刪除貼文", "post_id，Header Token", "刪除成功 / 失敗", "第二階段：貼文互動"],
  ["V2 新增", 8, "POST", "/api/posts/:id/like", "貼文按讚", "post_id，Header Token", "按讚成功", "第二階段：貼文互動"],
  ["V2 新增", 9, "DELETE", "/api/posts/:id/like", "取消按讚", "post_id，Header Token", "取消成功", "第二階段：貼文互動"],
  ["V2 新增", 10, "GET", "/api/posts/:id/comments", "取得貼文留言", "post_id，Header Token", "留言陣列", "第二階段：貼文互動"],
  ["V2 新增", 11, "POST", "/api/posts/:id/comments", "新增留言", '{"content": "..."}，Header Token', "comment_id", "第二階段：貼文互動"],

  ["V2 新增", 12, "POST", "/api/tasks", "新增任務", '{"title", "date", "remind_at"}，Header Token', "task_id", "第三階段：任務"],
  ["V2 新增", 13, "GET", "/api/tasks/today", "今日任務清單", "Header Token", "任務陣列", "第三階段：任務"],
  ["V2 新增", 14, "PATCH", "/api/tasks/:id", "編輯任務 / 打勾完成", '{"title", "is_done", "remind_at"}，Header Token', "更新成功", "第三階段：任務"],
  ["V2 新增", 15, "DELETE", "/api/tasks/:id", "刪除任務", "task_id，Header Token", "刪除成功", "第三階段：任務"],

  ["V2 新增", 16, "GET", "/api/notifications", "取得通知列表", "Header Token", "通知陣列", "第四階段：通知"],
  ["V2 新增", 17, "PATCH", "/api/notifications/:id/read", "單筆通知已讀", "notification_id，Header Token", "更新成功", "第四階段：通知"],
  ["V2 新增", 18, "PATCH", "/api/notifications/read-all", "全部通知已讀", "Header Token", "更新成功", "第四階段：通知"],

  ["V2 新增", 19, "GET", "/api/search", "搜尋貼文 / 使用者 / 社群", "?q=keyword，Header Token", "搜尋結果", "第五階段：探索"],
  ["V2 新增", 20, "GET", "/api/communities", "社群列表", "可選 ?sort=hot 或 ?sort=latest", "社群陣列", "第五階段：社群"],
  ["V2 新增", 21, "POST", "/api/communities", "建立社群", '{"name", "description"}，Header Token', "community_id", "第五階段：社群"],
  ["V2 新增", 22, "GET", "/api/communities/:id", "社群頁資料", "community_id", "社群資料", "第五階段：社群"],
  ["V2 新增", 23, "PATCH", "/api/communities/:id", "編輯社群資料", '{"name", "description", "rules"}，Header Token', "更新成功", "第五階段：社群"],
  ["V2 新增", 24, "POST", "/api/communities/:id/join", "加入社群", "community_id，Header Token", "加入成功", "第五階段：社群"],
  ["V2 新增", 25, "DELETE", "/api/communities/:id/join", "離開社群", "community_id，Header Token", "離開成功", "第五階段：社群"],
  ["V2 新增", 26, "GET", "/api/communities/:id/posts", "社群貼文列表", "community_id", "貼文陣列", "第五階段：社群"],

  ["V2 新增", 27, "GET", "/api/stats/me", "個人使用統計", "Header Token", "使用時間、任務完成數、回訪次數", "第六階段：統計"],
  ["完成品建議", 28, "POST", "/api/auth/logout", "登出並讓 token / refresh token 失效", "Header Token", "登出成功 / 失敗", "MVP 可先不做，完成品建議加入"],
  ["完成品建議", 29, "POST", "/api/auth/refresh", "更新 access token", '{"refresh_token": "..."} 或 Cookie', "新的 access token", "若有 refresh token 建議加入"],
  ["完成品建議", 30, "POST", "/api/auth/forgot-password", "忘記密碼，發送重設信", '{"email": "..."}', "發送成功 / 失敗", "帳號完整性"],
  ["完成品建議", 31, "POST", "/api/auth/reset-password", "重設密碼", '{"token", "new_password"}', "重設成功 / 失敗", "帳號完整性"],
  ["完成品建議", 32, "PATCH", "/api/auth/password", "登入後修改密碼", '{"old_password", "new_password"}，Header Token', "修改成功 / 失敗", "帳號完整性"],
];

sheet.getRange("A1:H1").values = [headers];
sheet.getRangeByIndexes(1, 0, rows.length, headers.length).values = rows;

sheet.getRange("A1:H1").format = {
  fill: "#0F766E",
  font: { bold: true, color: "#FFFFFF" },
  horizontalAlignment: "center",
  verticalAlignment: "center",
};

sheet.getRangeByIndexes(0, 0, rows.length + 1, headers.length).format = {
  wrapText: true,
  verticalAlignment: "top",
  borders: { preset: "all", style: "thin", color: "#D9E2E1" },
};

sheet.getRange("A1:H1").format.rowHeightPx = 34;
sheet.getRange("A:A").format.columnWidthPx = 120;
sheet.getRange("B:B").format.columnWidthPx = 76;
sheet.getRange("C:C").format.columnWidthPx = 76;
sheet.getRange("D:D").format.columnWidthPx = 230;
sheet.getRange("E:E").format.columnWidthPx = 190;
sheet.getRange("F:F").format.columnWidthPx = 270;
sheet.getRange("G:G").format.columnWidthPx = 230;
sheet.getRange("H:H").format.columnWidthPx = 190;

sheet.freezePanes.freezeRows(1);
sheet.tables.add(`A1:H${rows.length + 1}`, true, "ApiListTable");

const prioritySheet = workbook.worksheets.add("階段摘要");
prioritySheet.showGridLines = false;
prioritySheet.getRange("A1:C1").values = [["階段", "優先項目", "說明"]];
prioritySheet.getRange("A2:C8").values = [
  ["第一階段", "個人頁 + 圖片", "先完成使用者資料、個人頁、頭像與封面，後續貼文與社群都會用到。"],
  ["第二階段", "貼文互動", "補上編輯、刪除、按讚、留言，讓動態牆更接近完整社群功能。"],
  ["第三階段", "任務", "對應每日任務、提醒、完成狀態。"],
  ["第四階段", "通知", "支援按讚、留言、任務提醒與社群互動通知。"],
  ["第五階段", "搜尋 + 社群", "完成探索頁、社群頁、加入社群與社群貼文列表。"],
  ["第六階段", "統計", "補個人使用時間、任務完成數與回訪次數。"],
  ["完成品建議", "Auth 完整性", "登出、refresh token、忘記密碼、重設密碼、修改密碼。"],
];
prioritySheet.getRange("A1:C1").format = {
  fill: "#0F766E",
  font: { bold: true, color: "#FFFFFF" },
  horizontalAlignment: "center",
};
prioritySheet.getRange("A1:C8").format = {
  wrapText: true,
  verticalAlignment: "top",
  borders: { preset: "all", style: "thin", color: "#D9E2E1" },
};
prioritySheet.getRange("A:A").format.columnWidthPx = 120;
prioritySheet.getRange("B:B").format.columnWidthPx = 180;
prioritySheet.getRange("C:C").format.columnWidthPx = 520;
prioritySheet.freezePanes.freezeRows(1);
prioritySheet.tables.add("A1:C8", true, "PrioritySummaryTable");

const inspect = await workbook.inspect({
  kind: "table",
  range: "API列表!A1:H12",
  include: "values",
  tableMaxRows: 12,
  tableMaxCols: 8,
});
console.log(inspect.ndjson);

const errors = await workbook.inspect({
  kind: "match",
  searchTerm: "#REF!|#DIV/0!|#VALUE!|#NAME\\?|#N/A",
  options: { useRegex: true, maxResults: 50 },
});
console.log(errors.ndjson);

const preview = await workbook.render({ sheetName: "API列表", autoCrop: "all", scale: 1, format: "png" });
await fs.writeFile(path.join(outputDir, "api-list-preview.png"), new Uint8Array(await preview.arrayBuffer()));

const xlsx = await SpreadsheetFile.exportXlsx(workbook);
const outputPath = path.join(outputDir, "api-list-v1-v2.xlsx");
await xlsx.save(outputPath);
console.log(outputPath);
