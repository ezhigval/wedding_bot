/**********************************************************
 * CONFIG
 **********************************************************/
const GUEST_SHEET = "Список гостей";
const SEATING_SHEET = "Рассадка";

const COL_NAME = 1;       // A: имя гостя
const COL_TABLE = 7;      // G: столик  (БЫЛО 6 / F)

/**********************************************************
 * MAIN ENTRY — onEdit
 **********************************************************/
function onEdit(e) {
  const sh = e.source.getActiveSheet();
  if (!sh) return;

  if (sh.getName() === GUEST_SHEET && e.range.columnStart === COL_TABLE) {
    syncFromGuests();
  }

  if (sh.getName() === SEATING_SHEET) {
    syncFromSeating();
  }
}

/**********************************************************
 * BUILD SEATING HEADER FROM DROPDOWN
 **********************************************************/
function rebuildSeatingHeader() {
  const ss = SpreadsheetApp.getActive();
  const guestSheet = ss.getSheetByName(GUEST_SHEET);
  const seatSheet = ss.getSheetByName(SEATING_SHEET);

  const rule = guestSheet.getRange(2, COL_TABLE).getDataValidation();
  if (!rule) return;

  const criteria = rule.getCriteriaType();
  const args = rule.getCriteriaValues();

  if (criteria !== SpreadsheetApp.DataValidationCriteria.VALUE_IN_LIST) return;

  const tables = args[0].filter(x => x && x.toString().trim() !== "");

  seatSheet.getRange(1, 2, 1, seatSheet.getLastColumn()).clearContent();

  for (let i = 0; i < tables.length; i++) {
    seatSheet.getRange(1, i + 2).setValue(tables[i]);
  }
}

/**********************************************************
 * SYNC: Список гостей → Рассадка
 **********************************************************/
function syncFromGuests() {
  const ss = SpreadsheetApp.getActive();
  const guest = ss.getSheetByName(GUEST_SHEET);
  const seating = ss.getSheetByName(SEATING_SHEET);

  const gLast = guest.getLastRow();
  if (gLast < 2) return;

  const names = guest.getRange(2, COL_NAME, gLast - 1, 1).getValues().flat();
  const tables = guest.getRange(2, COL_TABLE, gLast - 1, 1).getValues().flat();

  const sLast = seating.getLastRow();
  const sCols = seating.getLastColumn();

  seating.getRange(2, 2, sLast, sCols).clearContent();

  const header = seating.getRange(1, 2, 1, sCols - 1).getValues()[0];

  const tableMap = {};
  for (let i = 0; i < header.length; i++) {
    const t = header[i];
    if (t) tableMap[t] = i + 2;
  }

  const tableBuckets = {};
  for (const t of header) tableBuckets[t] = [];

  for (let i = 0; i < names.length; i++) {
    const n = names[i];
    const t = tables[i];
    if (!n || !t || !(t in tableMap)) continue;
    tableBuckets[t].push(n);
  }

  for (const t in tableBuckets) {
    tableBuckets[t].sort((a, b) => a.localeCompare(b));
    const col = tableMap[t];
    for (let i = 0; i < tableBuckets[t].length; i++) {
      seating.getRange(i + 2, col).setValue(tableBuckets[t][i]);
    }
  }
}

/**********************************************************
 * SYNC: Рассадка → Список гостей
 **********************************************************/
function syncFromSeating() {
  const ss = SpreadsheetApp.getActive();
  const guest = ss.getSheetByName(GUEST_SHEET);
  const seating = ss.getSheetByName(SEATING_SHEET);

  const header = seating.getRange(1, 2, 1, seating.getLastColumn() - 1).getValues()[0];
  const rows = seating.getLastRow();

  const tableMap = {};
  for (let i = 0; i < header.length; i++) {
    const t = header[i];
    if (t) tableMap[t] = i + 2;
  }

  const gLast = guest.getLastRow();
  const gNames = guest.getRange(2, COL_NAME, gLast - 1, 1).getValues().flat();

  const newTables = {};

  for (let t in tableMap) {
    const col = tableMap[t];
    const values = seating.getRange(2, col, rows - 1, 1).getValues().flat();

    for (const name of values) {
      if (!name) continue;
      newTables[name] = t;
    }
  }

  for (let i = 0; i < gNames.length; i++) {
    const name = gNames[i];
    if (newTables[name]) {
      guest.getRange(i + 2, COL_TABLE).setValue(newTables[name]);
    }
  }
}

/**********************************************************
 * FULL CHECK — полная пересборка и выравнивание
 **********************************************************/
function fullReconcile() {
  rebuildSeatingHeader();
  syncFromGuests();
  syncFromSeating();
}
