import fs from "node:fs/promises";
import path from "node:path";
import { FileBlob, SpreadsheetFile } from "@oai/artifact-tool";

const inputPath = "C:/Users/QAZ/Downloads/V2 api 列表.xlsx";
const outputDir = path.resolve("outputs", "api-list");
await fs.mkdir(outputDir, { recursive: true });

const input = await FileBlob.load(inputPath);
const workbook = await SpreadsheetFile.importXlsx(input);
const sheet = workbook.worksheets.getItem("工作表1");

const existingRange = sheet.getRange("A1:E29");
const values = existingRange.values;
const headers = values[0];
const rows = values.slice(1);

for (const row of rows) {
  if (row[0] === "GET" && row[1] === "/api/communities") {
    row[3] = "可選 ?sort=hot 或 ?sort=latest，預設 latest";
  }
}

const additions = [
  ["GET", "/api/posts/:id", "查看單篇貼文詳細頁", "post_id", "單篇貼文資料"],
  ["GET", "/api/tasks/history", "歷史任務紀錄", "Header Token", "歷史任務陣列"],
  ["POST", "/api/tasks/:id/reminders", "設定任務提醒", '{"remind_at": "..."}', "設定成功 / 失敗"],
  ["DELETE", "/api/media/:id", "刪除圖片 / 檔案", "media_id，Header Token", "刪除成功 / 失敗"],
  ["GET", "/api/communities/:id/rules", "取得社群規則", "community_id", "社群規則"],
  ["GET", "/api/notification-settings", "取得通知設定", "Header Token", "通知設定"],
  ["PATCH", "/api/notification-settings", "修改通知設定", '{"like", "comment", "task_reminder", "community"}', "更新成功"],
  ["POST", "/api/auth/refresh", "更新 access token", '{"refresh_token": "..."} 或 Cookie', "新的 access token"],
  ["PATCH", "/api/account/password", "修改密碼", '{"old_password", "new_password"}，Header Token', "修改成功 / 失敗"],
  ["DELETE", "/api/account", "刪除帳號", "Header Token", "刪除成功 / 失敗"],
  ["PATCH", "/api/communities/:id/avatar", "修改社群頭像", '{"avatar_url": "..."}，Header Token', "更新成功"],
  ["PATCH", "/api/communities/:id/cover", "修改社群封面", '{"cover_url": "..."}，Header Token', "更新成功"],
  ["PATCH", "/api/communities/:id/rules", "編輯社群規則", '{"rules": "..."}，Header Token', "更新成功"],
  ["GET", "/api/communities/:id/members", "社群成員列表", "community_id，Header Token", "成員陣列"],
  ["PATCH", "/api/communities/:id/members/:userId/role", "修改成員權限", '{"role": "admin|member"}，Header Token', "更新成功"],
  ["POST", "/api/communities/:id/ban", "封鎖社群成員", '{"user_id": "...", "reason": "..."}，Header Token', "封鎖成功"],
  ["DELETE", "/api/communities/:id/ban/:userId", "解除封鎖成員", "community_id, user_id，Header Token", "解除成功"],
];

const allRows = [headers, ...rows, ...additions];
const writeRange = sheet.getRange(`A1:E${allRows.length}`);
writeRange.values = allRows;
writeRange.format = {
  wrapText: true,
  verticalAlignment: "top",
  borders: { preset: "all", style: "thin", color: "#D9E2E1" },
};

sheet.getRange("A1:E1").format = {
  fill: "#0F766E",
  font: { bold: true, color: "#FFFFFF" },
  horizontalAlignment: "center",
  verticalAlignment: "center",
};

sheet.getRange("A:A").format.columnWidthPx = 90;
sheet.getRange("B:B").format.columnWidthPx = 300;
sheet.getRange("C:C").format.columnWidthPx = 230;
sheet.getRange("D:D").format.columnWidthPx = 320;
sheet.getRange("E:E").format.columnWidthPx = 230;
sheet.freezePanes.freezeRows(1);

const inspect = await workbook.inspect({
  kind: "table",
  range: `工作表1!A1:E${Math.min(allRows.length, 20)}`,
  include: "values",
  tableMaxRows: 20,
  tableMaxCols: 5,
});
console.log(inspect.ndjson);

const errors = await workbook.inspect({
  kind: "match",
  searchTerm: "#REF!|#DIV/0!|#VALUE!|#NAME\\?|#N/A",
  options: { useRegex: true, maxResults: 50 },
});
console.log(errors.ndjson);

const preview = await workbook.render({ sheetName: "工作表1", autoCrop: "all", scale: 1, format: "png" });
await fs.writeFile(path.join(outputDir, "V2-api-list-updated-preview.png"), new Uint8Array(await preview.arrayBuffer()));

const output = await SpreadsheetFile.exportXlsx(workbook);
const outputPath = path.join(outputDir, "V2 api 列表-修正版.xlsx");
await output.save(outputPath);
console.log(outputPath);
