#!/usr/bin/env node
/**
 * serve-client.mjs — local stand-in for the nginx load balancer.
 *
 * Serves a client build directory statically (SPA fallback) and proxies API
 * routes exactly like nginx.conf so the media smoke can run without docker:
 *
 *   /session*, /connect*, /ws/* (incl. upgrade), /health, /ready, /metrics
 *       -> signalling server
 *   /api/* -> users service, with the /api prefix stripped
 *   everything else -> static files from --dir, SPA fallback to index.html
 *
 * Zero dependencies, node >= 18.
 *
 *   node tests/e2e/serve-client.mjs --dir client/build --port 4174 \
 *     --signalling http://127.0.0.1:8080 --users http://127.0.0.1:8081
 */
import http from 'node:http';
import net from 'node:net';
import fs from 'node:fs';
import path from 'node:path';

function parseArgs(argv) {
  const out = {
    dir: 'client/build',
    port: '4174',
    signalling: 'http://127.0.0.1:8080',
    users: 'http://127.0.0.1:8081',
  };
  for (let i = 0; i < argv.length; i++) {
    const m = argv[i].match(/^--([^=]+)(?:=(.*))?$/);
    if (!m) throw new Error(`bad arg: ${argv[i]}`);
    if (!(m[1] in out)) throw new Error(`unknown arg: --${m[1]}`);
    out[m[1]] = m[2] !== undefined ? m[2] : argv[++i];
  }
  return out;
}

const MIME = {
  '.html': 'text/html; charset=utf-8',
  '.js': 'text/javascript; charset=utf-8',
  '.css': 'text/css; charset=utf-8',
  '.json': 'application/json; charset=utf-8',
  '.png': 'image/png',
  '.svg': 'image/svg+xml',
  '.ico': 'image/x-icon',
  '.map': 'application/json; charset=utf-8',
  '.txt': 'text/plain; charset=utf-8',
};

/** Mirror of nginx.conf locations. Returns {host, port, path} or null. */
function route(url, signalling, users) {
  const pathname = url.split('?')[0];
  if (
    pathname === '/session' ||
    pathname.startsWith('/session/') ||
    pathname.startsWith('/session?') ||
    pathname === '/connect' ||
    pathname.startsWith('/connect/') ||
    pathname.startsWith('/connect?') ||
    pathname.startsWith('/ws/') ||
    pathname === '/health' ||
    pathname === '/ready' ||
    pathname === '/metrics'
  ) {
    return { host: signalling.hostname, port: signalling.port, path: url };
  }
  if (pathname === '/api' || pathname.startsWith('/api/')) {
    return { host: users.hostname, port: users.port, path: url.replace(/^\/api/, '') || '/' };
  }
  return null;
}

function proxyHttp(req, res, target) {
  const proxyReq = http.request(
    {
      host: target.host,
      port: target.port,
      method: req.method,
      path: target.path,
      headers: { ...req.headers, host: `${target.host}:${target.port}` },
    },
    (proxyRes) => {
      res.writeHead(proxyRes.statusCode, proxyRes.headers);
      proxyRes.pipe(res);
    },
  );
  proxyReq.on('error', () => {
    if (!res.headersSent) res.writeHead(502, { 'content-type': 'text/plain' });
    res.end('bad gateway');
  });
  req.pipe(proxyReq);
}

function proxyUpgrade(req, sock, head, target) {
  const proxy = net.connect(target.port, target.host, () => {
    let raw = `${req.method} ${target.path} HTTP/${req.httpVersion}\r\n`;
    for (const [k, v] of Object.entries(req.headers)) raw += `${k}: ${v}\r\n`;
    raw += '\r\n';
    proxy.write(raw);
    if (head && head.length) proxy.write(head);
    sock.pipe(proxy);
    proxy.pipe(sock);
  });
  proxy.on('error', () => sock.destroy());
  sock.on('error', () => proxy.destroy());
}

function main() {
  const args = parseArgs(process.argv.slice(2));
  const dir = path.resolve(args.dir);
  if (!fs.statSync(dir, { throwIfNoEntry: false })?.isDirectory()) {
    throw new Error(`not a directory: ${dir} (run the client build first)`);
  }
  const signalling = new URL(args.signalling);
  const users = new URL(args.users);

  const server = http.createServer((req, res) => {
    const target = route(req.url, signalling, users);
    if (target) {
      proxyHttp(req, res, target);
      return;
    }
    // Static file with SPA fallback.
    let pathname = decodeURIComponent(req.url.split('?')[0]);
    let file = path.normalize(path.join(dir, pathname));
    if (!file.startsWith(dir)) {
      res.writeHead(403).end();
      return;
    }
    if (!fs.statSync(file, { throwIfNoEntry: false })?.isFile()) {
      if (path.extname(pathname) === '') file = path.join(dir, 'index.html');
      else {
        res.writeHead(404).end();
        return;
      }
    }
    res.writeHead(200, { 'content-type': MIME[path.extname(file)] || 'application/octet-stream' });
    fs.createReadStream(file).pipe(res);
  });

  server.on('upgrade', (req, sock, head) => {
    const target = route(req.url, signalling, users);
    if (!target) {
      sock.destroy();
      return;
    }
    proxyUpgrade(req, sock, head, target);
  });

  server.listen(Number(args.port), '127.0.0.1', () => {
    console.log(`[serve] ${dir} on http://127.0.0.1:${args.port}`);
  });
}

main();
