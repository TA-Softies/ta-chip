// ta-chip Google Sheets AppScript
// Deploy: Extensions → Apps Script → Deploy → New deployment
// Type: Web app, Execute as: Me, Who has access: Anyone
// Copy the /exec URL into ROOT/config.json → appscript_url

var SHEET_NAME = "Rounds";
var COLUMNS = [
  "PC Location", "Rounder", "Time", "Display", "Mouse & Keyboard",
  "Kensington Lock", "Conduiting", "Tidiness", "Boot to Windows",
  "Time & Date", "Wallpaper", "Domain (TECHLAB)", "Microsoft Office",
  "Microsoft Teams", "Browser", "Frozen", "Policy Name", "Remarks", "Timestamp"
];

function doPost(e) {
  try {
    var d = JSON.parse(e.postData.contents);
    var ss = SpreadsheetApp.getActiveSpreadsheet();
    var sheet = ss.getSheetByName(SHEET_NAME);
    if (!sheet) {
      sheet = ss.insertSheet(SHEET_NAME);
    }

    // Auto-create header row on first use
    if (sheet.getLastRow() === 0) {
      sheet.appendRow(COLUMNS);
      sheet.getRange(1, 1, 1, COLUMNS.length).setFontWeight("bold");
      sheet.setFrozenRows(1);
    }

    sheet.appendRow([
      d.pc_location      || "",
      d.rounder          || "",
      d.shift_time       || "",
      d.display          || "",
      d.mouse_keyboard   || "",
      d.kensington_lock  || "",
      d.conduiting       || "",
      d.tidiness         || "",
      d.boot_to_windows  || "",
      d.time_date        || "",
      d.wallpaper        || "",
      d.domain           || "",
      d.microsoft_office || "",
      d.microsoft_teams  || "",
      d.browser          || "",
      d.deepfreeze_frozen || "",
      d.deepfreeze_policy || "",
      d.remarks          || "",
      d.timestamp        || ""
    ]);

    return ContentService
      .createTextOutput(JSON.stringify({ success: true, row: sheet.getLastRow() }))
      .setMimeType(ContentService.MimeType.JSON);

  } catch (err) {
    return ContentService
      .createTextOutput(JSON.stringify({ success: false, error: err.toString() }))
      .setMimeType(ContentService.MimeType.JSON);
  }
}

// Test via: run doGet in the editor to verify sheet access
function doGet(e) {
  return ContentService
    .createTextOutput(JSON.stringify({ status: "ta-chip AppScript running" }))
    .setMimeType(ContentService.MimeType.JSON);
}
