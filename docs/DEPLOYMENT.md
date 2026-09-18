# VideoConf Deployment Guide

Self-hosting guide for the v1 stack: Docker Compose (primary), TURN, TLS,
configuration, and capacity. Kubernetes/Helm packaging lands with the v1
packaging work (see [implementation plan](./IMPLEMENTATION_PLAN.md));
raw legacy manifests in `.deployment/` are superseded by that work.

## Architecture

```
                    ┌─────────────────┐
                    │  Load Balancer  │
                    │  (Nginx, :80)   │
                    └────────┬────────┘
         ┌───────────────────┼───────────────────┐
         │                   │                   │
┌────────┴────────┐ ┌────────┴────────┐ ┌────────┴────────┐
│ Signalling :8080│ │  Users   :8081  │ │ Client    :80   │
│ sessions + WS   │ │ auth + users    │ │ (React demo)    │
└────────┬────────┘ └────────┬────────┘ └─────────────────┘
         │                   │
┌────────┴───────────────────┴────────┐  ┌─────────────────┐
│ MongoDB :27017 (sessions + users)   │  │ TURN :3478      │
└─────────────────────────────────────┘  │ (coturn relay)  │
                                         └─────────────────┘
```

Media flows peer-to-peer between browsers (STUN first, TURN relay when
needed); the servers handle signalling, auth, and session state only.

## Quickstart (Docker Compose)

```bash
git clone https://github.com/r3tr056/go-videoconf.git
cd go-videoconf
cp .env.example .env
# Edit .env: set JWT_SECRET, TURN_USER, TURN_PASSWORD (all required).
docker compose up --build -d
```

Then open http://localhost (client), signalling on :8080, users on :8081,
TURN on :3478. `docker compose logs -f` to follow logs.

## Configuration

All settings live in `.env` (see `.env.example`). Required values fail fast:
compose refuses to start (`:?` gates) and the users service exits when
`JWT_SECRET` is missing. Never commit `.env` with real secrets.

| Variable | Required | Default | Purpose |
|---|---|---|---|
| `JWT_SECRET` | yes | — | HS256 signing secret (long, random) |
| `JWT_ISSUER` | no | `VideoConf` | JWT issuer claim |
| `DB_HOST` / `DB_PORT` | no | `mongodb` / `27017` | MongoDB (compose service) |
| `DB_NAME` | no | `vidchat` | App database |
| `DB_USERNAME` / `DB_PASSWORD` | no | `root` / `rootpassword` | Dev credentials — change in production |
| `WS_ALLOWED_ORIGINS` | prod | empty (allow all) | CORS + WebSocket origin allowlist |
| `TURN_USER` / `TURN_PASSWORD` | yes | — | TURN relay credentials |
| `TURN_REALM` | no | `videoconf` | TURN realm |
| `TURN_PUBLIC_URL` | no | `turn:localhost:3478` | TURN URL baked into the client build |
| `TURN_EXTERNAL_IP` | WAN | — | coturn's public IP (uncomment in compose) |

The demo client loads TURN settings at runtime from `/turn-config.json`:
compose images bake it from `TURN_*` at build (overridable with a file mount),
Helm serves it from a ConfigMap, and `npm start` serves the dev defaults from
`client/public/turn-config.json`.

## TURN (NAT traversal)

The bundled coturn service (`coturn/coturn:4.6.3`) relays media when direct
P2P fails. It listens on 3478/tcp+udp with UDP relay ports 49160–49200
(~10 concurrent relayed calls; widen the range for more).

- **Localhost/LAN demo:** defaults work; browsers reach `turn:localhost:3478`
  (or set `TURN_PUBLIC_URL` to the LAN host and rebuild the client).
- **WAN deployment:** set `TURN_EXTERNAL_IP` to coturn's public IP and
  uncomment the `--external-ip` line in `docker-compose.yml`, otherwise relay
  candidates carry an unreachable internal address. Open 3478/tcp+udp and the
  relay range on the host firewall.
- **Limitation:** v1 uses static TURN credentials, which are visible to anyone
  served the client bundle (inherent to static-credential TURN in browsers).
  Time-limited REST credentials are future work.

Verify TURN with any WebRTC trickle-ICE page against your `TURN_PUBLIC_URL`
and credentials: a `relay` candidate must appear.

## TLS

Production traffic must be HTTPS/WSS (browsers also require a secure context
for camera/mic, except localhost). Termination happens at Nginx:

1. Obtain a certificate (e.g. `certbot certonly --standalone -d meet.example.com`).
2. Copy `fullchain.pem` + `privkey.pem` into `./certs/` (chmod 600 the key).
3. Publish `443:443` and mount `./certs:/etc/nginx/certs:ro` on the
   `load-balancer` service.
4. Uncomment the `:443` server block in `nginx.conf` and set its `server_name`.
5. Set `WS_ALLOWED_ORIGINS=https://meet.example.com` in `.env` and restart.

The SDK derives `wss://` automatically from an `https://` page — no client
change needed. Keep HSTS enabled (in the template) once TLS is verified.

## Capacity

- **Room size:** mesh P2P is recommended for ~2–6 participants per room.
- **Server cap:** the signalling server refuses WebSocket upgrades past
  **8 members per socket** (HTTP 429; the SDK surfaces it as a join error).
- **Scaling out:** run one signalling instance per deployment (rooms are
  in-memory fan-out; cross-instance rooms need the v2 SFU). Session/user
  state lives in MongoDB, so the REST paths scale horizontally behind Nginx.
- A server-side SFU for larger rooms is the headline v2 item.

## Kubernetes (Helm)

```bash
helm install videoconf ./helm/videoconf \
  --set secrets.jwtSecret=$(openssl rand -base64 48) \
  --set secrets.turnPassword=$(openssl rand -base64 32) \
  --set ingress.host=meet.example.com \
  --set signalling.wsAllowedOrigins=https://meet.example.com \
  --set turn.publicUrl=turn:meet.example.com:3478 \
  --set turn.externalIP=203.0.113.10
# or: make deploy-k8s JWT_SECRET=... TURN_PASSWORD=... INGRESS_HOST=... TURN_PUBLIC_URL=...
```

- TLS: create a TLS secret, then
  `--set ingress.tls.enabled=true --set ingress.tls.secretName=<secret>`.
- Images come from ghcr.io (`global.imageRepository` forks cleanly).
- Signalling stays at 1 replica (in-memory rooms); users/client autoscale.
- TURN runs with host networking for direct UDP; `turn.externalIP` must be the
  node/public IP on WAN. The client gets TURN via a mounted ConfigMap, so no
  image rebuild is needed per environment.
- Upgrades re-run the same command as `helm upgrade`; removal is
  `helm uninstall videoconf` (`make undeploy-k8s`).

## MongoDB Operations

Replica-set topology plus backup/restore runbooks land with the persistence
phase. Until then: the compose volume `mongodb_data_container` holds all data
— back it up before upgrades:

### Indexes (created automatically at startup, idempotent)

Both services share one logical database (`vidchat`): `sessions` + `sockets`
(signalling) and `users` (users service). Both use the maintained
`mongo-driver` (unified in v1.0.0; the legacy `mgo.v2` driver is gone).

| Collection | Index | Purpose |
|---|---|---|
| `sockets` | `hashedUrl` unique | public session URL lookup per join; uniqueness guarantees one session per URL |
| `users` | `name` | authentication lookup (no collection scan) |
| `sessions`, `users` | `_id` (default) | primary key |

### Backup and Restore (drill-tested)

Back up with `mongodump` (indexes included in the dump):

```bash
# Compose
docker exec <mongo-container> sh -c \
  'mongodump --username root --password "$MONGO_INITDB_ROOT_PASSWORD" \
   --authenticationDatabase admin --db vidchat --out /tmp/vc-backup' \
&& docker cp <mongo-container>:/tmp/vc-backup ./backup-$(date +%F)
```

Restore (drops and replaces the database — stop writes first):

```bash
mongorestore --host <mongo-host> --username root --password <pw> \
  --authenticationDatabase admin --db vidchat --drop ./backup-YYYY-MM-DD/vidchat
```

Verify after restore: `POST /auth` with a known user returns a token, and
`POST /connect/<socket>` with a known session password returns its title.
The v1.0.0 drill ran exactly this: dump → drop (APIs 400/401) → restore →
identical socket URLs and working auth.

The compose volume `mongodb_data_container` holds all data; a filesystem
snapshot of the volume is a valid cold backup too (stop mongo first).

### Replica Set (HA)

For production HA, run 3 mongod instances as a replica set instead of the
single compose/Helm instance:

1. Start each member with `--replSet videoconf-rs` and a shared keyfile
   (`openssl rand -base64 756 > mongo.key`, chmod 400, same file on all
   members; Helm: mount it from a Secret).
2. Initiate once: `mongosh --eval 'rs.initiate({_id:"videoconf-rs",
   members:[{_id:0,host:"mongo-0:27017"},{_id:1,host:"mongo-1:27017"},
   {_id:2,host:"mongo-2:27017"}]})'`.
3. Point the services at the set (`DB_HOST=mongo-0,mongo-1,mongo-2` — the
   driver discovers the primary). No application change is needed: the code
   uses no transactions and tolerates primary stepdown via driver retry.
4. Back up from a secondary with the `mongodump` command above.

### Upgrades

- **mongo:7.0** is the pinned server. Fresh installs start there directly.
- **Driver:** both services use `mongo-driver` v1.17 (BSON-compatible with the
  old `mgo` documents: `ObjectId` encodings are identical, so no data
  migration was needed for the v1.0.0 driver switch).
- **Index builds:** index setup runs at every startup and is a no-op once
  indexes exist; unique `hashedUrl` builds fail only if true duplicates exist
  (fix by deleting the ambiguous dup sessions first).

## Observability

- **Liveness vs readiness:** `GET /health` is shallow (process alive);
  `GET /ready` is deep (MongoDB reachable, 503 otherwise). Kubernetes probes
  and the Helm chart use both; point load-balancer health checks at `/ready`.
- **Metrics:** `GET /metrics` on :8080/:8081 exposes Prometheus metrics:
  `videoconf_http_requests_total` and `videoconf_http_request_duration_seconds`
  (by service/method/route/status), plus `videoconf_ws_connections_current`
  and `videoconf_ws_messages_total` on signalling. Scrape them directly or
  with the ServiceMonitor of your choice; restrict `/metrics` to cluster
  networks in production.
- **Logs:** both Go services emit structured JSON to stdout (no log files);
  compose rotates container logs (10 MB × 3). Coturn and nginx log to stdout
  by default.
- **SBOMs:** every release attaches `sbom-*.spdx.json` per image, and pushed
  images carry SLSA provenance + SBOM attestations in ghcr.io.

## Security Checklist

- [ ] Unique `JWT_SECRET`, `TURN_PASSWORD`, and MongoDB credentials in `.env`
- [ ] `WS_ALLOWED_ORIGINS` set to the exact https origin
- [ ] TLS enabled with HSTS (except localhost-only demos)
- [ ] Host firewall: 80/443 public; 3478 + relay range public; 27017/8080/8081
      loopback-only (or behind the LB)
- [ ] Images pinned (no `:latest`); `trivy`/`codeql` gates green in CI

### Secret Management

- **Compose:** secrets live in `.env` (`scripts/setup-env.sh` creates it with
  mode 600 and random `JWT_SECRET`/`TURN_PASSWORD`). Never commit `.env`;
  back it up separately — losing `JWT_SECRET` invalidates all sessions.
- **Helm:** pass secrets via `--set secrets.*` (dev) or, for production, set
  `secrets.existingSecret` to a pre-created Secret (via
  [Sealed Secrets](https://github.com/bitnami-labs/sealed-secrets) /
  [external-secrets](https://external-secrets.io/)) holding `jwt-secret`,
  `mongo-username`, `mongo-password`, `turn-username`, `turn-password`. Note
  `secrets.turnPassword` is ALSO required in values: it is served to browsers
  in `turn-config.json`, so it is client-visible by design — keep both copies
  in sync.
- **Rotation:** `JWT_SECRET` rotation logs every user out (1h token TTL);
  `TURN_PASSWORD` rotation needs coturn restart plus client `turn-config.json`
  refresh. MongoDB passwords rotate via standard `changeUserPassword` + restart.

### Vulnerability Policy

- CI fails on any CRITICAL/HIGH finding (`trivy fs`, vuln+secret scanners)
  and runs CodeQL (Go + JavaScript) on every push.
- Runtime/shipped dependencies must be clean — no exceptions.
- `.trivyignore` holds per-CVE exceptions **only** for build-toolchain
  transitives that provably never ship (react-scripts dev chain), each with a
  justification comment. Entries are CVE IDs, so future advisories in the same
  packages still alert. Revisit on every toolchain upgrade.
