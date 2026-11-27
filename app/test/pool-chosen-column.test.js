'use strict';

// stratum_pool.chosen came with 1.1.0: a database from 1.0 must get the
// column on open, with every pool it already had still chosen, and four
// processes opening it at once must not trip over each other.
const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('fs');
const os = require('os');
const path = require('path');
const { spawn } = require('child_process');
const Database = require('better-sqlite3');

const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'peerlab-chosen-'));
const file = path.join(tmpDir, 'old.db');

test('a 1.0 database gets the chosen column, every pool still chosen', async () => {
  const old = new Database(file);
  old.exec(`CREATE TABLE stratum_pool (
    id INTEGER PRIMARY KEY AUTOINCREMENT, label TEXT NOT NULL, host TEXT NOT NULL, port INTEGER NOT NULL,
    enabled INTEGER NOT NULL DEFAULT 1, is_default INTEGER NOT NULL DEFAULT 0, created_at INTEGER NOT NULL, UNIQUE(host, port));
    INSERT INTO stratum_pool (label, host, port, enabled, is_default, created_at) VALUES ('Old', 'old.example', 3333, 0, 1, 1);`);
  old.close();
  const opener = `process.env.SQLITE_PATH=${JSON.stringify(file)};process.env.DATA_DIR=${JSON.stringify(tmpDir)};process.env.LOG_LEVEL='error';require(${JSON.stringify(path.join(__dirname, '../src/lib/db'))}).open();`;
  const codes = await Promise.all([0, 1, 2, 3].map(() => new Promise((resolve) => {
    const child = spawn(process.execPath, ['-e', opener], { stdio: ['ignore', 'ignore', 'pipe'] });
    let err = '';
    child.stderr.on('data', (d) => { err += d; });
    child.on('exit', (code) => resolve([code, err]));
  })));
  for (const [code, err] of codes) assert.equal(code, 0, err);
  const db = new Database(file);
  const row = db.prepare('SELECT enabled, chosen FROM stratum_pool WHERE host = ?').get('old.example');
  db.close();
  assert.deepEqual(row, { enabled: 0, chosen: 1 }, 'paused before, still chosen and paused');
});

test.after(() => fs.rmSync(tmpDir, { recursive: true, force: true }));
