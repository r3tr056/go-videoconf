/**
 * signalling-load.js — k6 load profile for the signalling server.
 *
 * Each VU plays a lone guest: create a session, join it, hold a WebSocket
 * for a while exchanging presence, then leave. Lone guests never get a
 * peer, which is exactly what we want here: the server only signals (mesh
 * media is peer-to-peer), so this measures session churn + socket fan-out
 * capacity without any browser.
 *
 *   k6 run -e BASE_URL=http://localhost:8080 tests/load/signalling-load.js
 *
 * Soak mode (nightly / pre-release): SOAK=true stretches the sustain stage
 * to 30 minutes at 25 VUs.
 */
import http from 'k6/http';
import ws from 'k6/ws';
import { check, sleep } from 'k6';
import { Counter, Rate } from 'k6/metrics';

const SOAK = __ENV.SOAK === 'true';

export const options = SOAK
  ? {
      stages: [
        { duration: '2m', target: 25 },
        { duration: '30m', target: 25 },
        { duration: '2m', target: 0 },
      ],
      thresholds: {
        http_req_failed: ['rate<0.01'],
        checks: ['rate>0.99'],
      },
    }
  : {
      stages: [
        { duration: '20s', target: 10 },
        { duration: '60s', target: 10 },
        { duration: '10s', target: 0 },
      ],
      thresholds: {
        http_req_failed: ['rate<0.01'],
        http_req_duration: ['p(95)<1500'],
        ws_connecting: ['p(95)<1500'],
        ws_session_duration: ['p(95)<12000'],
        checks: ['rate>0.99'],
      },
    };

const wsMsgsReceived = new Counter('ws_msgs_received');
const wsErrors = new Counter('ws_errors');
const joinRate = new Rate('join_ok');

const BASE = __ENV.BASE_URL || 'http://localhost:8080';
const WS_BASE = BASE.replace(/^http/, 'ws');
const PASSWORD = 'loadtest-password';

export default function () {
  const id = `load-${__VU}-${__ITER}-${Date.now()}`;
  const headers = { 'Content-Type': 'application/json' };

  const created = http.post(
    `${BASE}/session`,
    JSON.stringify({ host: id, title: id, password: PASSWORD }),
    { headers, tags: { name: 'create-session' } },
  );
  const createOk = check(created, {
    'session created': (r) => r.status === 200 && r.json('socket') !== undefined,
  });
  if (!createOk) return;
  const invite = created.json('socket');

  const joined = http.post(`${BASE}/connect/${invite}`, JSON.stringify({ password: PASSWORD }), {
    headers,
    tags: { name: 'join-session' },
  });
  const room = joined.status === 200 ? joined.json('socket') : undefined;
  const joinOk = check(joined, { 'session joined': () => room !== undefined });
  joinRate.add(joinOk);
  if (!joinOk) return;

  const holdMs = 3000 + Math.floor(Math.random() * 5000);
  const res = ws.connect(
    `${WS_BASE}/ws/${room}`,
    { tags: { name: 'signal-socket' } },
    (socket) => {
      socket.on('open', () => {
        socket.send(JSON.stringify({ type: 'connect', userID: id }));
        socket.setTimeout(() => {
          socket.send(JSON.stringify({ type: 'disconnect', userID: id }));
          socket.close();
        }, holdMs);
      });
      socket.on('message', () => {
        wsMsgsReceived.add(1);
      });
      socket.on('error', () => {
        wsErrors.add(1);
      });
    },
  );
  check(res, { 'ws status 101': (r) => r && r.status === 101 });

  sleep(Math.random() * 2);
}
