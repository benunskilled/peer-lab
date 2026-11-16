'use strict';

const net = require('net');
const crypto = require('crypto');
const config = require('./config');
const peerSync = require('./peer-sync');
const { isValidPort } = require('./validate');
// Address parsing/formatting lives in one module - see address.js for why.
const { resolveHostPort, formatAddress, hostFromAddress, unreachableNetwork, ipv4InCidr, DEFAULT_PORTS } = require('./address');
const rpc = require('./rpc');
const logger = require('./logger').make('manual-peer');
const { logAction } = require('./peer-rotation/log');
const spread = require('./peer-rotation/spread');

// The first four bytes of every P2P message, per chain. A node that is on
// another chain answers with other bytes, which is as useless as no answer.
const MAGIC = {
  mainnet: 'f9beb4d9',
  testnet: '0b110907',
  testnet4: '1c163f28',
  signet: '0a03cf40',
  regtest: 'fabfb5da',
};

const sha256d = (buf) => crypto.createHash('sha256').update(crypto.createHash('sha256').update(buf).digest()).digest();

// A bare version message: no services, height 0, no tx relay. Enough for a
// node to answer with its own version, and nothing more. The user agent is
// `mark`, random letters for this one probe: if our own Core answered, its
// getpeerinfo shows an inbound peer with that subver (answeredBySelf). Not
// the start height: Core 32 no longer shows it in getpeerinfo (tried against
// a regtest Core 32.1 before 1.0.0).
function versionMessage(magic, mark = '') {
  const ua = Buffer.from(mark, 'ascii');
  if (ua.length > 252) throw new Error('user agent too long for a one-byte length');
  const payload = Buffer.alloc(80 + 1 + ua.length + 5);
  payload.writeInt32LE(70016, 0);
  payload.writeBigInt64LE(BigInt(Math.floor(Date.now() / 1000)), 12);
  // addr_recv and addr_from stay zero: services 0, address ::, port 0.
  crypto.randomBytes(8).copy(payload, 72);
  payload.writeUInt8(ua.length, 80);
  ua.copy(payload, 81);
  // start height 0 and relay 0 after it.
  const header = Buffer.alloc(24);
  Buffer.from(magic, 'hex').copy(header, 0);
  header.write('version', 4, 'ascii');
  header.writeUInt32LE(payload.length, 16);
  sha256d(payload).copy(header, 20, 0, 4);
  return Buffer.concat([header, payload]);
}

/**
 * True only if a Bitcoin node on this chain answers here. An open port is not
 * enough: a web server or anything else listening on 8333 accepted the TCP
 * connection and used to count as a node. So this sends a version message and
 * waits for the first message header back: the chain's magic bytes and
 * "version". The handshake is never finished, the connection closes there.
 *
 * Only ever for an address a person typed in or picked: it goes out from this
 * container, not through Core's proxy. The slot fill never probes; it lets
 * Core connect (peer-rotation/fill.js).
 */
// Different on every probe, and nothing a real client sends.
const selfMark = () => `/${crypto.randomBytes(8).toString('hex')}/`;

// Whether the node that answered is our own Core: reached through the
// Umbrel's LAN or public address, umbrel.local, or a forwarded port, it looks
// like any other node from here. Core has just read our version, so the
// connection is in its getpeerinfo with our mark while the socket is open.
// Core not answering: no proof either way, the probe stands.
async function answeredBySelf(mark) {
  try {
    const peers = await rpc.getPeerInfo();
    return Array.isArray(peers) && peers.some((p) => p.inbound && p.subver === mark);
  } catch (err) {
    logger.debug('getpeerinfo failed during a probe', { error: err.message });
    return false;
  }
}

// Resolves true when a node on this chain answers, 'self' when that node is
// our own Core, false otherwise.
function probePort(host, port, timeoutMs = 3000) {
  const magic = MAGIC[config.bitcoin.network] || MAGIC.mainnet;
  const mark = selfMark();
  return new Promise((resolve) => {
    const socket = net.connect({ host, port });
    let settled = false;
    let got = Buffer.alloc(0);
    const done = async (ok) => {
      if (settled) return;
      settled = true;
      // Open until Core has been asked: closed, the connection leaves its list.
      const self = ok && await answeredBySelf(mark);
      socket.destroy();
      resolve(self ? 'self' : ok);
    };
    socket.setTimeout(timeoutMs);
    socket.once('connect', () => socket.write(versionMessage(magic, mark)));
    socket.on('data', (chunk) => {
      got = Buffer.concat([got, chunk]);
      if (got.length < 16) return;
      done(got.subarray(0, 4).toString('hex') === magic && got.subarray(4, 16).equals(Buffer.from('version\0\0\0\0\0', 'ascii')));
    });
    socket.once('end', () => done(false));
    socket.once('timeout', () => done(false));
    socket.once('error', () => done(false));
  });
}

// Bare-bones sanity check - real validation happens by probing the
// address itself; this just rejects obvious junk input. Brackets are
// allowed so bracketed IPv6 input ("[2001:db8::1]:8333") isn't rejected
// outright - see resolveHostPort below for why that notation matters.
function looksLikeHost(input) {
  return typeof input === 'string' && /^[a-zA-Z0-9.:\-[\]]{3,255}$/.test(input.trim());
}

/**
 * Manual "add peer by IP" flow - used both by the free-text IP field and by
 * "Add as Manual" on an existing peer row. The caller supplies a bare host
 * (no port); we probe the standard Bitcoin P2P port first, then the
 * configured fallback, and only persist + `addnode` whichever actually
 * answers as a Bitcoin node (see probePort). This is deliberate even for peers we already see
 * connected: an inbound peer's getpeerinfo address is its ephemeral
 * *outbound source* port, not the port its node listens on for incoming P2P
 * connections, so that port is useless for `addnode` - we always re-derive
 * the real listening port ourselves rather than trust whatever port we
 * happened to observe the peer on.
 *
 * Nothing is written and no existing connection is touched until that
 * handshake has succeeded and a manual slot is confirmed available - see
 * peerSync.addTrustedPeer, which owns that order. Trying an inbound peer to
 * see whether it can be promoted is therefore free: if it does not listen,
 * the attempt costs nothing at all.
 *
 * A successful add is persisted to trusted_peer (not just a one-off addnode
 * RPC call), so it survives container restarts/updates via
 * peer-sync.syncTrustedToAddnode() the same way peers trusted from the
 * dashboard do.
 */
async function manualAddPeer(rawInput, label, { kept = false } = {}) {
  const probed = await probePeer(rawInput);
  if (!probed.ok) return probed;

  // Asked for by hand, so it outranks whatever is currently weakest: at
  // capacity, free a slot rather than writing a ninth row Core will never be
  // told about. The response names what was dropped - a silent eviction of a
  // peer the user spent days earning would be far worse than a refusal.
  //
  // The padlock is NOT part of that, and is off unless the caller asks. Both
  // ways to add by hand ask for it - the typed-in box and a row's "Add as
  // Manual" (dashboard-server.js): a peer added by hand is a choice, and stays
  // until it is removed or unlocked. What Slot Fill adds comes without one.
  const result = await peerSync.addTrustedPeer(probed.address, label, { evictToFit: true, kept });
  if (!result.ok) {
    return { ok: false, address: probed.address, error: result.error };
  }
  logger.info('manually added peer', { address: probed.address, evicted: result.evicted?.address || null });
  // In the Slot Fill log too, every time: a hand-made choice belongs next to
  // Slot Fill's own, and a peer that lost its slot to one should not
  // vanish without a trace.
  // The peer is in by now: a busy database must not turn that into an error
  // the user would retry, displacing a second peer.
  try {
    logAction({
      action: 'manual',
      address: probed.address,
      replacedAddress: result.evicted?.address,
      // Where the slot came from: the spread decides which peer goes
      // (spread.leastMissed), so the region says more than "the weakest".
      note: result.evicted ? `replaced one in ${regionOf(result.evicted) || 'an unknown region'}` : null,
    });
  } catch (err) {
    logger.warn('Slot Fill log entry for a manual add failed', { address: probed.address, error: err.message });
  }
  return {
    ok: true,
    address: probed.address,
    ...(probed.via ? { note: ADDED_VIA(probed.via) } : {}),
    ...evictionNote(result),
  };
}

/**
 * Reachability only: is there a Bitcoin node listening on this host, and on
 * which port? Writes nothing and disconnects nothing; the only RPC is
 * getnetworkinfo, cached, to ask whether a direct test is safe at all
 * (directRefusal).
 *
 * Split out of manualAddPeer (which now calls it) so the dashboard can offer
 * the same question as a standalone action. "Can I even add this peer?" and
 * "add this peer" are different questions, and only one of them should have
 * consequences.
 */
async function probePeer(rawInput) {
  const host = (typeof rawInput === 'string' ? rawInput : '').trim();
  if (!looksLikeHost(host)) {
    return { ok: false, error: 'invalid host/IP' };
  }
  const { addr, port: explicitPort } = resolveHostPort(host);
  const route = await routeOf(addr);
  if (route && route.refuse) return { ok: false, error: route.refuse };
  const own = ownAddress(addr);
  if (own) return { ok: false, error: own };
  // Reached through Tor or I2P: not tested from here, handed to Core as it is.
  // Without a port, the network's default - there is no port to search for.
  if (route && route.via) {
    // I2P peers are on port 0 and nothing else (SAM 3.1, Core's
    // I2P_SAM31_PORT), the way Core lists them; any other port Core would
    // never dial.
    const i2p = unreachableNetwork(addr) === 'I2P';
    if (!i2p && explicitPort != null && !isValidPort(explicitPort)) {
      return { ok: false, error: `${explicitPort} is not a valid port (1-65535)` };
    }
    const port = i2p ? 0 : explicitPort != null ? explicitPort : DEFAULT_PORTS[config.bitcoin.network] || 8333;
    return { ok: true, address: formatAddress(addr, port), via: route.via, untested: true, message: TEST_VIA(route.via) };
  }

  // Already has an explicit :port - respect it and skip the port search.
  if (explicitPort != null) {
    // resolveHostPort's pattern accepts up to five digits, so "1.2.3.4:99999"
    // parses as port 99999 - and net.connect() throws ERR_SOCKET_BAD_PORT
    // *synchronously* for anything outside 1-65535, from inside probePort's
    // Promise executor. The promise then rejects instead of resolving false,
    // and the user gets a bare 500 instead of a usable message. Exactly the
    // failure validate.js exists to prevent (see its header) - this path just
    // wasn't using it.
    if (!isValidPort(explicitPort)) {
      return { ok: false, error: `${explicitPort} is not a valid port (1-65535)` };
    }
    const reachable = await probePort(addr, explicitPort);
    if (reachable === 'self') return { ok: false, error: OWN };
    if (!reachable) return { ok: false, error: `no Bitcoin node answering at ${formatAddress(addr, explicitPort)}` };
    return { ok: true, address: formatAddress(addr, explicitPort), triedPorts: [explicitPort] };
  }

  const address = await findListeningAddress(addr);
  if (address === 'self') return { ok: false, error: OWN };
  if (address) return { ok: true, address, triedPorts: config.manualPeerPorts };

  return {
    ok: false,
    error: `no node answering on ${config.manualPeerPorts.join(' or ')}`,
  };
}

/**
 * How a typed-in address is reached: null for a direct test from here, `via`
 * for one Core reaches through Tor or I2P and is handed to it untested, or
 * `refuse` with the reason.
 *
 * The test goes out from this container, not through Core's proxy. A Tor or
 * I2P name cannot be reached that way at all, and asking for it would hand
 * the name to Docker's resolver. On a node that sends IPv4 or IPv6 through a
 * proxy (Tor), a direct test would connect from the node's real address - the
 * one thing such a node keeps hidden. So in both cases Core gets the address
 * and connects itself, the way it does for Slot Fill (15.11.2026; before, the
 * address was refused, and a node on Tor alone could add no peer at all).
 * getnetworkinfo says which, asked anew each time someone tests or adds a
 * peer. Only a fresh answer clears a direct test (review, 15.11.2026): a kept
 * one may be from before Core was restarted with a proxy, and with Core not
 * answering and nothing known, a direct test was made anyway - the one thing
 * that must not happen on a node that hides behind Tor. If Core does not
 * answer, the last answer may still send the address to Core or refuse it,
 * never test it from here: the reader is asked to try again.
 */
/**
 * The node itself is no peer. Two ways it could be typed in:
 *  - an address inside Umbrel's app network: the Docker gateway (every relayed
 *    inbound peer shows as 10.21.0.1, so adding it would disconnect them all),
 *    Core's own container, or another app;
 *  - one of the addresses Core says it is reachable at (getnetworkinfo
 *    localaddresses, its own onion among them).
 * Either would hand Core an addnode to itself, which it refuses forever while
 * the entry holds a manual slot (review before 1.0.0). Loopback is left alone:
 * in this container it is the container, never Core, and nothing listens.
 */
const OWN = "That is your own node, or an app on your Umbrel, not a peer.";
// One spelling per IPv6 literal (2001:0db8:0::7 is 2001:db8::7), and an
// IPv4-mapped one in either form back to dotted IPv4.
function plainHost(raw) {
  let host = String(raw || '').replace(/^\[|\]$/g, '').toLowerCase();
  if (host.includes(':')) {
    try {
      host = new URL(`http://[${host}]/`).hostname.slice(1, -1);
    } catch {
      return host;
    }
    const mapped = host.match(/^::ffff:([0-9a-f]{1,4}):([0-9a-f]{1,4})$/);
    if (mapped) {
      const [hi, lo] = mapped.slice(1).map((h) => parseInt(h, 16));
      host = [hi >> 8, hi & 255, lo >> 8, lo & 255].join('.');
    }
  }
  return host;
}
function ownAddress(addr) {
  const host = plainHost(addr);
  if (/^\d{1,3}(\.\d{1,3}){3}$/.test(host) && config.umbrelInternalNetworkCidr
    && ipv4InCidr(host, config.umbrelInternalNetworkCidr)) return OWN;
  const local = ((netInfo && netInfo.localaddresses) || []).map((a) => plainHost(a.address));
  return local.includes(host) ? OWN : null;
}

const NET_NAME = { Tor: 'onion', I2P: 'i2p' };
const TEST_VIA = (via) => `Your node reaches this peer through ${via}. To keep your node's address private, ` +
  "Peer Lab doesn't test it from here. Add & Keep hands it straight to Core.";
const ADDED_VIA = (via) => `Added. To keep your node's address private, Core connects through ${via} itself. ` +
  'That can take a few minutes. It shows up under Manual as soon as Core has connected, no need to add it again.';
const NOT_USED = (label) => `Your node doesn't use ${label}, so it can't reach this peer.`;
const NO_ANSWER = (what) => `Bitcoin Core didn't answer, so Peer Lab can't tell whether your node uses ${what}. Try again in a moment.`;
let netInfo = null;
async function routeOf(addr) {
  const overlay = unreachableNetwork(addr);
  // CJDNS: Core has no proxy for it, and this container no way to it.
  if (overlay === 'CJDNS') return { refuse: 'CJDNS peers cannot be tested or added here: Peer Lab reaches peers only directly' };
  try {
    const answer = await rpc.getNetworkInfo();
    // An answer without its list of networks says nothing (review G1).
    if (!answer || !Array.isArray(answer.networks)) throw new Error('getnetworkinfo without networks');
    netInfo = answer;
  } catch (err) {
    logger.debug('getnetworkinfo failed before a probe', { error: err.message });
    if (!netInfo) return { refuse: NO_ANSWER(overlay || 'a proxy') };
    // An older answer may still say "through Tor" or "not used"; never "direct".
    const kept = routeFrom(netInfo, addr, overlay);
    return kept ? kept : { refuse: NO_ANSWER('a proxy') };
  }
  return routeFrom(netInfo, addr, overlay);
}

function routeFrom(answer, addr, overlay) {
  const networks = (answer && answer.networks) || [];
  if (overlay) {
    // -onlynet limits only Core's own picks: a manual peer still goes through
    // the proxy (init.cpp, -onlynet).
    const n = networks.find((x) => x.name === NET_NAME[overlay]);
    return n && ((n.reachable && !n.limited) || n.proxy) ? { via: overlay } : { refuse: NOT_USED(overlay) };
  }
  // A name could resolve to either; an IPv6 literal has a colon. A proxy on
  // any network the address may use sends it to Core (a test from here would
  // leak) - also under -onlynet=onion, which leaves manual peers alone and
  // still sends them through the proxy. A network Core has neither a proxy
  // for nor uses refuses an address that is surely on it, and a name only
  // when neither IPv4 nor IPv6 is usable.
  // "[::ffff:1.2.3.4]" (or ::ffff:102:304) is IPv4 to Core and to this container's socket alike,
  // so it goes by IPv4's proxy (review G1).
  const v4 = /^[0-9.]+$/.test(addr) || /^\[?::ffff:([0-9.]+|[0-9a-f]{1,4}:[0-9a-f]{1,4})\]?$/i.test(addr);
  const nets = v4 ? ['ipv4'] : addr.includes(':') ? ['ipv6'] : ['ipv4', 'ipv6'];
  const info = nets.map((name) => networks.find((x) => x.name === name));
  const label = (n) => (n.name === 'ipv4' ? 'IPv4' : 'IPv6');
  // Umbrel's proxy is Tor (port 9050); any other is called what it is.
  const proxied = info.find((n) => n && n.proxy);
  if (proxied) return { via: /:9050$/.test(proxied.proxy) || /tor/i.test(proxied.proxy) ? 'Tor' : 'its proxy' };
  // Direct only on Core's plain word: each network listed, with no proxy.
  if (info.some((n) => !n || typeof n.proxy !== 'string')) return { refuse: NO_ANSWER('a proxy') };
  if (info.every((n) => n.limited || !n.reachable)) return { refuse: NOT_USED(info.map(label).join(' or ')) };
  return null;
}

/**
 * Given a bare host, returns the first configured P2P port where a Bitcoin
 * node answers - already in Core's own address format - 'self' if that node
 * is our own Core, or null if none does.
 *
 * Shared on purpose: Slot Fill has to answer exactly this
 * question for an inbound candidate (whose observed port is its ephemeral
 * source port, not what it listens on). A second private copy of this loop
 * over there was one more place for the port list, the bracket format, or the
 * "not every peer listens" case to drift out of sync.
 */
async function findListeningAddress(host) {
  for (const port of config.manualPeerPorts) {
    const reachable = await probePort(host, port);
    if (reachable === 'self') return 'self';
    if (reachable) return formatAddress(host, port);
  }
  return null;
}

// The region a peer is in, by name, or null: what the spread goes by.
function regionOf(peer) {
  return spread.REGION_NAMES[spread.placeOf(peer).region] || null;
}

// Bitcoin Core holds MAX_ADDNODE_CONNECTIONS (8) manual connections at once,
// so the 9th add is not a queue, it is a swap - and the user has to be told
// which peer paid for it, by name and by region (what the spread chose it
// by, as the log says), or a good peer can vanish from the list without
// anyone ever seeing why.
function evictionNote(result) {
  if (!result.evicted) return {};
  const region = regionOf(result.evicted);
  return {
    warning:
      `all ${result.max} manual slots were taken, so ${result.evicted.address} (${region ? `in ${region}` : 'region unknown'}) ` +
      (result.evicted.live
        ? 'was disconnected to make room. It keeps its record.'
        : 'was removed to make room.'),
  };
}

module.exports = {
  // For the tests: forget the cached getnetworkinfo.
  _resetNetInfo: () => { netInfo = null; },
  manualAddPeer,
  probePeer,
  hostFromAddress,
  resolveHostPort,
  formatAddress,
  probePort,
  versionMessage,
  MAGIC,
};
