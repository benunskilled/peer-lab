'use strict';
/**
 * The regions the reader picked for Slot Fill (15.11.2026): tapped in the
 * card's list of regions, one or several. While any is picked, a free slot
 * goes to them, spread among them as usual and still never to an operator
 * already in use; once none of them has a candidate left, the usual pick
 * takes over, and the page says so. Only free slots: nobody connected is
 * dropped for a pick. Kept on the node, where Slot Fill runs, not in a browser.
 */
const db = require('../db');
const { REGION_NAMES } = require('./spread');

const KEY = 'slot_fill_regions';

function pickedRegions() {
  try {
    const row = db.instance.prepare('SELECT value FROM meta WHERE key = ?').get(KEY);
    const list = row ? JSON.parse(row.value) : [];
    return Array.isArray(list) ? list.filter((r) => Object.hasOwn(REGION_NAMES, r)) : [];
  } catch {
    return [];
  }
}

// Unknown names are dropped, not stored: the list is only ever regions.
function setPickedRegions(list) {
  const clean = [...new Set((Array.isArray(list) ? list : []).filter((r) => typeof r === 'string' && Object.hasOwn(REGION_NAMES, r)))];
  if (clean.length) {
    db.instance
      .prepare('INSERT INTO meta (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value')
      .run(KEY, JSON.stringify(clean));
  } else {
    db.instance.prepare('DELETE FROM meta WHERE key = ?').run(KEY);
  }
  return clean;
}

module.exports = { pickedRegions, setPickedRegions };
