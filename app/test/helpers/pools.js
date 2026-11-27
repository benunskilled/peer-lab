'use strict';

// Eight public pools, as an install from before 25.11.2026 has them: a fresh
// database now starts with none (lib/db.js), and the race and ranking tests
// need pools with ids to attach their observations to.
function seedPools(db) {
  const insert = db.instance.prepare(
    'INSERT OR IGNORE INTO stratum_pool (label, host, port, enabled, is_default, created_at) VALUES (?, ?, ?, 1, 1, ?)',
  );
  for (let i = 1; i <= 8; i++) insert.run(`Pool ${i}`, `pool${i}.example`, 3333, Date.now());
}

module.exports = { seedPools };
