'use strict';

/**
 * The slot fill's audit trail: what it changed, and what a person changed by
 * hand, written down and read back for the dashboard. `rotation_log` is a
 * window - exactly what the dashboard shows and not one row more.
 */

const db = require('../db');
const { whereOf } = require('../queries/ranking-row');

// The dashboard shows the newest four and opens up to all of them. A slot
// fill changes something a few times a week, so this is weeks of history.
const LOG_ENTRIES = 30;

function logAction(entry) {
  db.instance
    .prepare(
      `INSERT INTO rotation_log (at, action, address, replaced_address, note)
       VALUES (@at, @action, @address, @replacedAddress, @note)`,
    )
    .run({
      at: Date.now(),
      action: entry.action,
      address: entry.address,
      replacedAddress: entry.replacedAddress || null,
      note: entry.note || null,
    });

  // Nothing is kept that is not shown. The dashboard offers the newest
  // LOG_ENTRIES behind its "Show all" button, so the table holds
  // exactly that many and the rest goes on the way in - no retention window to
  // reason about, no daily sweep, and no way for a misbehaving loop to leave
  // fifty thousand rows behind before anyone looks.
  //
  // By id, the order of writing (AUTOINCREMENT), not by `at`: two actions in
  // one tick can share a millisecond, and after the clock steps back a new
  // action carries an older `at` than all the rest and would be the one
  // deleted.
  db.instance
    .prepare(`DELETE FROM rotation_log WHERE id NOT IN (SELECT id FROM rotation_log ORDER BY id DESC LIMIT ?)`)
    .run(LOG_ENTRIES);
}

// Where each peer is goes with it, for the flag under its address: looked up
// now, from the address, as for the manual peers above the log.
function recentLog(limit = LOG_ENTRIES) {
  return db.instance
    .prepare(
      `SELECT id, at, action, address, replaced_address AS replacedAddress, note
       FROM rotation_log ORDER BY id DESC LIMIT ?`,
    )
    .all(limit)
    .map((e) => ({ ...e, where: whereOf(e.address) }));
}

// Since a slot last came free - removed by hand or retired, and not filled
// since: what freed it, and each candidate tried for it and how it ended,
// newest first (15.11.2026: five candidates in Africa failed in half an hour,
// and the page showed only the last one). Null with no slot freed.
const TRIES_SHOWN = 10;
// Counted from the last slot filled, not the first freed: with two slots
// free and one filled since, the other one's candidates are still shown
// (review, 15.11.2026). Only called while a slot is free (index.js).
function triesSinceFree(limit = TRIES_SHOWN) {
  const filled = db.instance
    .prepare(`SELECT MAX(at) AS at FROM rotation_log WHERE action IN ('promote', 'manual')`)
    .get().at || 0;
  const freed = db.instance
    .prepare(`SELECT at, action, address FROM rotation_log WHERE action IN ('unmanual', 'retire') AND at > ? ORDER BY at`)
    .all(filled);
  const since = freed.length ? freed[0].at : filled;
  const where = 'ok = 0 AND started_at IS NOT NULL AND started_at >= ?';
  const total = db.instance.prepare(`SELECT COUNT(*) AS n FROM slot_fill_tried WHERE ${where}`).get(since).n;
  if (!total) return null;
  const tries = db.instance
    .prepare(`SELECT address, started_at AS startedAt, tried_at AS endedAt, reason FROM slot_fill_tried
              WHERE ${where} ORDER BY started_at DESC LIMIT ?`)
    .all(since, limit);
  return {
    since,
    total,
    freed: freed.map((f) => ({ at: f.at, action: f.action, address: f.address, where: whereOf(f.address) })),
    tries: tries.map((t) => ({ ...t, where: whereOf(t.address) })),
  };
}

module.exports = { logAction, recentLog, triesSinceFree, LOG_ENTRIES };
