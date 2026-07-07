import { FileBlob, SpreadsheetFile } from "@oai/artifact-tool";

const inputPath = "C:/Users/QAZ/Downloads/V2 api 列表.xlsx";
const input = await FileBlob.load(inputPath);
const workbook = await SpreadsheetFile.importXlsx(input);

const overview = await workbook.inspect({
  kind: "workbook,sheet,table",
  maxChars: 12000,
  tableMaxRows: 80,
  tableMaxCols: 10,
  tableMaxCellChars: 120,
});

console.log(overview.ndjson);
