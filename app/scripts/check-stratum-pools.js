'use strict';

// Checks every pool in src/data/stratum-pools.json the way Stratum Race
// connects: mining.subscribe, mining.authorize with the burn address, then
// listening. A pool passes when it accepts the login and sends a job for the
// block most pools are on. Run before a release; nothing is mined.
//   node scripts/check-stratum-pools.js [seconds, default 45]
// Prints no addresses, only names, hosts and the verdict.

const net = require('net');
const { pools } = require('../src/data/stratum-pools.json');
const { canonical } = require('../src/lib/prevhash');

const WAIT_MS = (Number(process.argv[2]) || 45) * 1000;
const USER = '1BitcoinEaterAddressDontSendf59kuE.worker1';

function check(p) {
  return new Promise((resolve) => {
    const r = { name: p.name, ep: `${p.host}:${p.port}`, loginRefused: Boolean(p.loginRefused), auth: null, prev: null, jobs: 0, err: '' };
    let buf = '';
    const s = net.connect({ host: p.host, port: p.port });
    const end = () => { s.destroy(); resolve(r); };
    const timer = setTimeout(end, WAIT_MS);
    s.on('connect', () => {
      s.write(`${JSON.stringify({ id: 1, method: 'mining.subscribe', params: ['peer-lab/1.0'] })}\n`);
      s.write(`${JSON.stringify({ id: 2, method: 'mining.authorize', params: [USER, 'x'] })}\n`);
    });
    s.on('data', (chunk) => {
      buf += chunk.toString();
      if (buf.length > 512 * 1024) { r.err = 'oversized'; return end(); }
      let i;
      while ((i = buf.indexOf('\n')) >= 0) {
        const line = buf.slice(0, i).trim();
        buf = buf.slice(i + 1);
        let m;
        try { m = JSON.parse(line); } catch { continue; }
        if (!m || typeof m !== 'object') continue; // "null" parses too, and m.id would throw
        if (m.id === 2) r.auth = m.result === true;
        if (m.method === 'mining.notify' && Array.isArray(m.params)) { r.jobs += 1; if (!r.prev) r.prev = canonical(String(m.params[1])); }
      }
    });
    s.on('error', (e) => { r.err = e.code || e.message; clearTimeout(timer); end(); });
    s.on('close', () => { clearTimeout(timer); resolve(r); });
  });
}

(async () => {
  const results = await Promise.all(pools.map(check));
  const tally = new Map();
  for (const r of results) if (r.prev) tally.set(r.prev, (tally.get(r.prev) || 0) + 1);
  const tip = [...tally].sort((a, b) => b[1] - a[1])[0]?.[0];
  let bad = 0;
  for (const r of results) {
    // loginRefused: a pool known to say no to the login and send jobs anyway.
    const ok = (r.auth !== false || r.loginRefused) && r.prev === tip && r.jobs > 0;
    if (!ok) bad += 1;
    const why = ok ? 'ok' : r.err || (r.auth === false && !r.loginRefused ? 'login refused' : !r.jobs ? 'no job' : 'job on another block');
    console.log(`${ok ? '  ' : '✗ '}${r.name.padEnd(28)} ${r.ep.padEnd(36)} ${why} (${r.jobs} jobs)`);
  }
  console.log(`\n${results.length - bad} of ${results.length} pass.`);
  process.exitCode = bad ? 1 : 0;
})();
