'use strict';

/**
 * The public pools a user can pick for Stratum Race (data/stratum-pools.json).
 *
 * Since 25.11.2026 a fresh install starts with none: the Stratum Race card
 * lists these by country and the user ticks the ones to race. A tick only
 * ever adds a pool from this list - its host, port and name come from here,
 * never from the request - so the list is also what bounds what the tick can
 * open a socket to.
 */

const path = require('path');

const FILE = path.join(__dirname, '..', 'data', 'stratum-pools.json');
const data = require(FILE);

const key = (host, port) => `${String(host).toLowerCase()}:${Number(port)}`;
const byKey = new Map(data.pools.map((p) => [key(p.host, p.port), p]));

function find(host, port) {
  return byKey.get(key(host, port)) || null;
}

/**
 * The list as the page draws it: each pool with the row it has in the
 * database, if any - `id` and `enabled` - so a pool added years ago from the
 * old defaults shows ticked like one ticked today.
 */
function withRows(rows) {
  const rowOf = new Map(rows.map((r) => [key(r.host, r.port), r]));
  return data.pools.map((p) => {
    const row = rowOf.get(key(p.host, p.port));
    return {
      name: p.name,
      host: p.host,
      port: p.port,
      country: p.country,
      region: p.region,
      kind: p.kind,
      routed: Boolean(p.routed),
      id: row ? row.id : null,
      // Ticked: in the race table, running or paused by its switch there.
      enabled: Boolean(row),
      paused: Boolean(row && !row.enabled),
    };
  });
}

module.exports = { find, withRows, checked: data.checked, size: data.pools.length };
