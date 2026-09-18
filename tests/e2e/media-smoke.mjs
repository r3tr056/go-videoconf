#!/usr/bin/env node
/**
 * media-smoke.mjs — N-peer WebRTC smoke test through the real UI.
 *
 * Drives N headless browser sessions (via chromedriver/geckodriver, raw
 * WebDriver protocol, zero npm dependencies): peer 0 creates a meeting in
 * the UI, peers 1..N-1 join with the invite code, and the harness asserts
 * live video flows in EVERY direction (every <video> on every peer advances
 * currentTime: 1 local + N-1 remote tiles each).
 *
 * Also captures screenshots and fails on severe browser console errors
 * (chromedriver log on Chrome, an injected error hook on Firefox, where the
 * log endpoint does not exist).
 *
 * Requires: node >= 18, a driver matching the browser binary — or a remote
 * WebDriver URL (selenium standalone images work: pass the /wd/hub URL and
 * omit the local binary/driver flags).
 *
 *   node tests/e2e/media-smoke.mjs --base-url http://localhost:4174 \
 *     --chrome-binary /path/to/chrome-headless-shell \
 *     --chromedriver /path/to/chromedriver [--peers 4] [--out-dir DIR]
 *
 *   node tests/e2e/media-smoke.mjs --base-url http://localhost:4174 \
 *     --browser firefox --firefox-binary /path/to/firefox \
 *     --geckodriver /path/to/geckodriver [--peers 2]
 *
 * Mesh cost grows quadratically (6 peers = 15 peer connections); raise
 * --media-timeout-ms for N=6 (180000 recommended).
 */
import { spawn } from 'node:child_process';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));

const EL_KEY = 'element-6066-11e4-a52e-4f735466cecf';

function parseArgs(argv) {
  const out = {
    'base-url': '',
    browser: 'chrome',
    'chrome-binary': '',
    chromedriver: 'chromedriver',
    'firefox-binary': '',
    geckodriver: 'geckodriver',
    'driver-url': '',
    'driver-port': '9515',
    'out-dir': path.join(__dirname, 'out'),
    peers: '2',
    'media-timeout-ms': '60000',
    'expect-relay': '',
  };
  for (let i = 0; i < argv.length; i++) {
    const m = argv[i].match(/^--([^=]+)(?:=(.*))?$/);
    if (!m) throw new Error(`bad arg: ${argv[i]}`);
    const key = m[1];
    if (!(key in out)) throw new Error(`unknown arg: --${key}`);
    out[key] = m[2] !== undefined ? m[2] : argv[++i];
  }
  if (!out['base-url']) throw new Error('--base-url is required');
  if (out.browser !== 'chrome' && out.browser !== 'firefox') {
    throw new Error(`--browser must be chrome or firefox (got ${out.browser})`);
  }
  const peers = Number(out.peers);
  if (!Number.isInteger(peers) || peers < 2 || peers > 8) {
    throw new Error('--peers must be an integer 2..8');
  }
  out.peers = peers;
  if (!out['driver-url']) {
    if (out.browser === 'chrome' && !out['chrome-binary']) {
      throw new Error('--chrome-binary is required (or pass --driver-url)');
    }
    if (out.browser === 'firefox' && !out['firefox-binary']) {
      throw new Error('--firefox-binary is required (or pass --driver-url)');
    }
  }
  return out;
}

const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

class Driver {
  constructor(base) {
    this.base = base;
  }

  async call(method, p, body) {
    const res = await fetch(this.base + p, {
      method,
      headers: { 'Content-Type': 'application/json' },
      body: body === undefined ? undefined : JSON.stringify(body),
    });
    const text = await res.text();
    let json;
    try {
      json = JSON.parse(text);
    } catch {
      throw new Error(`${method} ${p} -> HTTP ${res.status}: ${text.slice(0, 300)}`);
    }
    if (!res.ok || (json.status !== undefined && json.status !== 0)) {
      const msg = (json.value && json.value.message) || text;
      const err = new Error(`${method} ${p} -> ${String(msg).split('\n')[0].slice(0, 300)}`);
      err.status = res.status;
      throw err;
    }
    return json.value;
  }

  async waitReady(deadlineMs) {
    for (;;) {
      try {
        const v = await this.call('GET', '/status');
        if (v && (v.ready === true || v.ready === undefined)) return;
      } catch {
        /* not up yet */
      }
      if (Date.now() > deadlineMs) throw new Error('driver never became ready');
      await sleep(250);
    }
  }

  async newSession(args) {
    const caps =
      args.browser === 'chrome'
        ? {
            browserName: 'chrome',
            'goog:chromeOptions': {
              ...(args['chrome-binary'] ? { binary: args['chrome-binary'] } : {}),
              args: [
                '--no-sandbox',
                '--disable-gpu',
                '--window-size=1440,900',
                '--use-fake-device-for-media-stream',
                '--use-fake-ui-for-media-stream',
                '--autoplay-policy=no-user-gesture-required',
                '--mute-audio',
              ],
            },
            'goog:loggingPrefs': { browser: 'ALL' },
          }
        : {
            browserName: 'firefox',
            'moz:firefoxOptions': {
              ...(args['firefox-binary'] ? { binary: args['firefox-binary'] } : {}),
              // Headless for local runs; harmless under selenium's Xvfb.
              args: args['driver-url'] ? [] : ['-headless'],
              prefs: {
                'media.navigator.streams.fake': true,
                'media.navigator.permission.disabled': true,
                'media.autoplay.default': 0,
                'media.peerconnection.enabled': true,
              },
            },
          };
    const v = await this.call('POST', '/session', { capabilities: { alwaysMatch: caps } });
    const session = new Session(this, v.sessionId);
    if (args.browser === 'firefox') {
      await session.setRect(1440, 900);
    }
    return session;
  }
}

// Injected FIRST on every page load: geckodriver has no log endpoint, so
// Firefox console failures are collected in-page instead. Installed on both
// browsers for a uniform record; Chrome additionally checks /log.
const ERROR_HOOK = `
window.__smokeErrors = [];
window.addEventListener('error', (e) => {
  window.__smokeErrors.push(String((e && e.message) || e));
});
window.addEventListener('unhandledrejection', (e) => {
  window.__smokeErrors.push('unhandled: ' + String((e && e.reason) || e));
});
const __origConsoleError = console.error.bind(console);
console.error = (...a) => {
  try {
    window.__smokeErrors.push(a.map(String).join(' '));
  } catch {}
  __origConsoleError(...a);
};
return true;`;

class Session {
  constructor(driver, id) {
    this.driver = driver;
    this.id = id;
    this.p = `/session/${id}`;
  }

  call(method, p, body) {
    return this.driver.call(method, this.p + p, body);
  }

  navigate(url) {
    return this.call('POST', '/url', { url });
  }

  title() {
    return this.call('GET', '/title');
  }

  setRect(width, height) {
    return this.call('POST', '/window/rect', { width, height });
  }

  async elements(css) {
    const els = await this.call('POST', '/elements', { using: 'css selector', value: css });
    return els.map((e) => e[EL_KEY]);
  }

  async element(css) {
    const v = await this.call('POST', '/element', { using: 'css selector', value: css });
    return v[EL_KEY];
  }

  clickEl(id) {
    return this.call('POST', `/element/${id}/click`, {});
  }

  async click(css) {
    await this.clickEl(await this.element(css));
  }

  async type(css, text) {
    const id = await this.element(css);
    await this.call('POST', `/element/${id}/clear`, {});
    await this.call('POST', `/element/${id}/value`, { text });
  }

  execute(script, args = []) {
    return this.call('POST', '/execute/sync', { script, args });
  }

  async waitFor(css, timeoutMs, what) {
    const deadline = Date.now() + timeoutMs;
    for (;;) {
      if ((await this.elements(css)).length > 0) return;
      if (Date.now() > deadline) throw new Error(`timeout waiting for ${what || css}`);
      await sleep(300);
    }
  }

  async waitForGone(css, timeoutMs, what) {
    const deadline = Date.now() + timeoutMs;
    for (;;) {
      if ((await this.elements(css)).length === 0) return;
      if (Date.now() > deadline) throw new Error(`timeout waiting for ${what || css} to disappear`);
      await sleep(300);
    }
  }

  async screenshot(file) {
    const b64 = await this.call('GET', '/screenshot');
    fs.writeFileSync(file, Buffer.from(b64, 'base64'));
  }

  async severeLogs() {
    const entries = await this.call('POST', '/log', { type: 'browser' });
    return entries.filter((e) => e.level === 'SEVERE').map((e) => e.message);
  }

  async hookErrors() {
    try {
      return (await this.execute('return window.__smokeErrors || null;')) || [];
    } catch {
      return ['<could not read error hook>'];
    }
  }

  delete() {
    return this.driver.call('DELETE', this.p).catch(() => {});
  }
}

const VIDEO_PROBE = `return Array.from(document.querySelectorAll('.vc-tile video')).map(v => ({
  local: !!v.closest('.vc-tile.local'),
  readyState: v.readyState,
  width: v.videoWidth,
  height: v.videoHeight,
  time: v.currentTime,
}));`;

function liveVideos(probe) {
  return probe.filter((v) => v.readyState >= 2 && v.width > 0 && v.time > 0);
}

async function main() {
  const args = parseArgs(process.argv.slice(2));
  const outDir = args['out-dir'];
  fs.mkdirSync(outDir, { recursive: true });
  const result = { ok: false, browser: args.browser, peers: args.peers, steps: [], severe: {}, hook: {}, videos: {} };
  const step = (name) => {
    result.steps.push(name);
    console.log(`[smoke] ${name}`);
  };

  let driverProc = null;
  let driverLog = '';
  let driverBase;
  if (args['driver-url']) {
    driverBase = args['driver-url'].replace(/\/$/, '');
    step(`using remote driver at ${driverBase}`);
  } else {
    const driverBin = args.browser === 'chrome' ? args.chromedriver : args.geckodriver;
    driverProc = spawn(driverBin, [`--port=${args['driver-port']}`], {
      stdio: ['ignore', 'pipe', 'pipe'],
    });
    driverProc.stderr.on('data', (d) => {
      driverLog += d.toString();
    });
    driverBase = `http://127.0.0.1:${args['driver-port']}`;
  }
  const killDriver = () => {
    try {
      driverProc && driverProc.kill('SIGKILL');
    } catch {
      /* already gone */
    }
  };
  process.on('exit', killDriver);

  const sessions = [];
  try {
    const driver = new Driver(driverBase);
    await driver.waitReady(Date.now() + 30000);
    step('driver ready');

    if (args['expect-relay'] === 'true') {
      // Fail fast unless the deployment actually serves relay-only mode:
      // with iceTransportPolicy 'relay', advancing video proves TURN relay.
      const res = await fetch(`${args['base-url']}/turn-config.json`, { cache: 'no-store' });
      const cfg = await res.json();
      if (!cfg || cfg.forceRelay !== true || !cfg.url) {
        throw new Error(`--expect-relay but turn-config lacks relay mode: ${JSON.stringify(cfg)}`);
      }
      step(`relay mode confirmed (${cfg.url})`);
    }

    // ---- Peer 0: create a meeting through the UI ----
    const first = await driver.newSession(args);
    sessions.push(first);
    await first.navigate(args['base-url']);
    await first.execute(ERROR_HOOK);
    const title = await first.title();
    if (!/videoconf/i.test(title)) throw new Error(`unexpected title: ${title}`);
    step('peer 0: home loaded');
    await first.waitFor('[data-testid="new-meeting-btn"]', 15000, 'new-meeting button');
    await first.click('[data-testid="new-meeting-btn"]');
    await first.waitFor('#create-title', 10000, 'create dialog');
    await first.screenshot(path.join(outDir, 'dialog.png'));
    step('peer 0: create dialog open');
    const password = `smoke-${Date.now()}`;
    await first.type('#create-title', 'Smoke test');
    await first.type('#create-password', password);
    await first.click('.vc-dialog button[type="submit"]');
    await first.waitFor('.vc-grid', 20000, 'peer 0 call view');
    await first.waitForGone('.vc-dialog', 10000, 'create dialog');
    const code = await first.execute(
      `return (document.querySelector('[data-testid="full-code"]') ||
        document.querySelector('.vc-code-chip')).textContent.trim();`,
    );
    if (!code) throw new Error('peer 0: no invite code rendered');
    result.codeLength = code.length;
    step(`peer 0: in call, invite code (${code.length} chars)`);
    await sleep(1500);
    await first.screenshot(path.join(outDir, 'waiting.png'));

    // ---- Peers 1..N-1: join with the code ----
    for (let i = 1; i < args.peers; i++) {
      const peer = await driver.newSession(args);
      sessions.push(peer);
      await peer.navigate(args['base-url']);
      await peer.execute(ERROR_HOOK);
      await peer.waitFor('#join-code', 15000, `peer ${i} join form`);
      await peer.type('#join-code', code);
      await peer.type('#join-password', password);
      await peer.click('.vc-panel button[type="submit"]');
      await peer.waitFor('.vc-grid', 25000, `peer ${i} call view`);
      step(`peer ${i}: joined call`);
    }

    // ---- Assert live media in every direction ----
    const deadline = Date.now() + Number(args['media-timeout-ms']);
    let probes = [];
    for (;;) {
      probes = await Promise.all(sessions.map((s) => s.execute(VIDEO_PROBE)));
      const allLive = probes.every((p) => {
        const live = liveVideos(p);
        return live.length >= args.peers && live.filter((v) => !v.local).length >= args.peers - 1;
      });
      if (allLive) break;
      if (Date.now() > deadline) {
        throw new Error(`media timeout: ${JSON.stringify(probes).slice(0, 2000)}`);
      }
      await sleep(1000);
    }
    step(`all ${args.peers} peers render local + ${args.peers - 1} remote video(s)`);
    // Frames must ADVANCE (a frozen first frame is not a working call).
    const before = probes.map((p) => p.map((v) => v.time));
    await sleep(2500);
    probes = await Promise.all(sessions.map((s) => s.execute(VIDEO_PROBE)));
    probes.forEach((probe, i) => {
      result.videos[`peer${i}`] = probe;
      const after = probe.map((v) => v.time);
      if (after.length !== before[i].length) throw new Error(`video set changed on peer ${i}`);
      after.forEach((t, j) => {
        if (!(t > before[i][j])) throw new Error(`video ${j} on peer ${i} is frozen (t=${t})`);
      });
    });
    step('video currentTime advances on every tile of every peer');

    await first.screenshot(path.join(outDir, 'call.png'));
    const last = sessions[sessions.length - 1];
    await last.click('[aria-label="Meeting options"]');
    await last.waitFor('.vc-menu', 5000, 'settings menu');
    await last.screenshot(path.join(outDir, 'menu.png'));
    step('screenshots captured');

    // ---- Console hygiene ----
    if (args.browser === 'chrome') {
      const sev = await Promise.all(sessions.map((s) => s.severeLogs()));
      sev.forEach((entries, i) => {
        result.severe[`peer${i}`] = entries;
      });
      const flat = sev.flat();
      if (flat.length > 0) {
        throw new Error(`SEVERE console entries: ${JSON.stringify(flat).slice(0, 2000)}`);
      }
      step('no SEVERE console entries');
    }
    const hooks = await Promise.all(sessions.map((s) => s.hookErrors()));
    hooks.forEach((entries, i) => {
      result.hook[`peer${i}`] = entries;
    });
    if (hooks.flat().length > 0) {
      throw new Error(`page errors captured: ${JSON.stringify(hooks).slice(0, 2000)}`);
    }
    step('no page errors via hook');

    result.ok = true;
    console.log(`[smoke] PASS: ${args.peers}-peer call with live video in every direction`);
  } catch (err) {
    result.error = String((err && err.stack) || err);
    console.error(`[smoke] FAIL: ${result.error.split('\n').slice(0, 5).join('\n')}`);
    // Best-effort failure artifacts.
    for (let i = 0; i < sessions.length; i++) {
      try {
        await sessions[i].screenshot(path.join(outDir, `fail-${i}.png`));
      } catch {
        /* ignore */
      }
    }
    try {
      fs.writeFileSync(path.join(outDir, 'driver.log'), driverLog);
    } catch {
      /* ignore */
    }
  } finally {
    for (const s of sessions) await s.delete();
    killDriver();
    fs.writeFileSync(path.join(outDir, 'smoke-result.json'), JSON.stringify(result, null, 2));
  }
  process.exit(result.ok ? 0 : 1);
}

main().catch((err) => {
  console.error(`[smoke] harness error: ${err && err.stack ? err.stack : err}`);
  process.exit(2);
});
