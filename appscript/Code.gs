// ta-chip Google Sheets AppScript
// Deploy: Extensions → Apps Script → Deploy → New deployment
// Type: Web app, Execute as: Me, Who has access: Anyone
// Copy the /exec URL into ROOT/config.json → appscript_url

var SHEET_NAME = "Rounds";
var COLUMNS = [
  "PC Location", "Rounder", "Time", "Display", "Mouse & Keyboard",
  "Kensington Lock", "Conduiting", "Tidiness", "Boot to Windows",
  "Time & Date", "Wallpaper", "Domain", "Microsoft Office",
  "Microsoft Teams", "Internet", "Frozen", "Policy Name",
  "Disk Space", "Last Reboot", "Windows", "RAM",
  "Monitor", "Keyboard", "Mouse",
  "Defender", "Activation", "Audio", "Camera",
  "Remarks", "Timestamp"
];

var WEBHOOK_URL   = "https://arux.lrxrn.workers.dev/webhook?type=ta-chip";
var WEBHOOK_TOKEN = "vBqGZDlkHehytF/6sndeKIomdegambAtms+Sf37aTgg=";

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
      d.pc_location       || "",
      d.rounder           || "",
      d.shift_time        || "",
      d.display           || "",
      d.mouse_keyboard    || "",
      d.kensington_lock   || "",
      d.conduiting        || "",
      d.tidiness          || "",
      d.boot_to_windows   || "",
      d.time_date         || "",
      d.wallpaper         || "",
      d.domain            || "",
      d.microsoft_office  || "",
      d.microsoft_teams   || "",
      d.internet          || "",
      d.deepfreeze_frozen || "",
      d.deepfreeze_policy || "",
      d.disk_space        || "",
      d.last_reboot       || "",
      d.win_version       || "",
      d.ram               || "",
      d.monitor           || "",
      d.keyboard          || "",
      d.mouse             || "",
      d.defender          || "",
      d.activation        || "",
      d.audio             || "",
      d.camera            || "",
      d.remarks           || "",
      d.timestamp         || ""
    ]);

    // Fire-and-forget — don't let a webhook failure break the sheet write
    try { sendWebhook(d); } catch (webhookErr) {}

    return ContentService
      .createTextOutput(JSON.stringify({ success: true, row: sheet.getLastRow() }))
      .setMimeType(ContentService.MimeType.JSON);

  } catch (err) {
    return ContentService
      .createTextOutput(JSON.stringify({ success: false, error: err.toString() }))
      .setMimeType(ContentService.MimeType.JSON);
  }
}

// ── Webhook ───────────────────────────────────────────────────────────────────

function statusEmoji(s) {
  if (s === "V") return "✅";
  if (s === "Y") return "⚠️";
  if (s === "X") return "❌";
  if (!s || s === "N/A") return "➖";
  return s;
}

function embedColor(d) {
  var checks = [
    d.display, d.mouse_keyboard, d.kensington_lock, d.conduiting, d.tidiness,
    d.boot_to_windows, d.time_date, d.wallpaper, d.domain,
    d.microsoft_office, d.microsoft_teams, d.internet, d.deepfreeze_frozen,
    d.defender, d.activation
  ];
  if (checks.some(function(v) { return v === "X"; })) return 16711680; // red
  if (checks.some(function(v) { return v === "Y"; })) return 16776960; // yellow
  return 65280; // green
}

function sendWebhook(d) {
  var hw = [
    "Display " + statusEmoji(d.display),
    "Mouse & KB " + statusEmoji(d.mouse_keyboard),
    "Kensington " + statusEmoji(d.kensington_lock),
    "Conduiting " + statusEmoji(d.conduiting),
    "Tidiness " + statusEmoji(d.tidiness)
  ].join("  ·  ");

  var sw = [
    "Boot " + statusEmoji(d.boot_to_windows),
    "Time " + statusEmoji(d.time_date),
    "Wallpaper " + statusEmoji(d.wallpaper),
    "Domain " + statusEmoji(d.domain),
    "Office " + statusEmoji(d.microsoft_office),
    "Teams " + statusEmoji(d.microsoft_teams),
    "Internet " + statusEmoji(d.internet),
    "Defender " + statusEmoji(d.defender),
    "Activation " + statusEmoji(d.activation),
    "Audio " + statusEmoji(d.audio),
    "Camera " + statusEmoji(d.camera)
  ].join("  ·  ");

  var dfValue = "Frozen " + statusEmoji(d.deepfreeze_frozen);
  if (d.deepfreeze_policy && d.deepfreeze_policy !== "N/A") {
    dfValue += "  ·  Policy: " + d.deepfreeze_policy;
  }

  var sysInfo = [
    "Disk: " + (d.disk_space || "—"),
    "Reboot: " + (d.last_reboot || "—"),
    "OS: " + (d.win_version || "—"),
    "RAM: " + (d.ram || "—"),
    "Monitor: " + (d.monitor || "—"),
    "Keyboard: " + (d.keyboard || "—"),
    "Mouse: " + (d.mouse || "—")
  ].join("  ·  ");

  var fields = [
    { name: "Hardware",    value: hw,      inline: false },
    { name: "Software",    value: sw,      inline: false },
    { name: "DeepFreeze",  value: dfValue, inline: false },
    { name: "System Info", value: sysInfo, inline: false }
  ];

  if (d.remarks && d.remarks.trim() !== "") {
    fields.push({ name: "Remarks", value: d.remarks.trim(), inline: false });
  }

  var payload = {
    title:       "PC Inspection — " + (d.pc_location || "unknown"),
    description: "Checked by **" + (d.rounder || "unknown") + "** at " + (d.shift_time || d.timestamp || ""),
    color:       embedColor(d),
    fields:      fields,
    footer:      { text: d.timestamp || "" }
  };

  UrlFetchApp.fetch(WEBHOOK_URL, {
    method:  "post",
    contentType: "application/json",
    headers: { "Authorization": "Bearer " + WEBHOOK_TOKEN },
    payload: JSON.stringify(payload),
    muteHttpExceptions: true
  });
}

// Test via: run doGet in the editor to verify sheet access
function doGet(e) {
  return ContentService
    .createTextOutput(JSON.stringify({ status: "ta-chip AppScript running" }))
    .setMimeType(ContentService.MimeType.JSON);
}
