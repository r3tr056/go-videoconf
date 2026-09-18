# go-videoconf — Production-Ready Implementation Plan

**Status: Final** (accepted 2026-09-16; acceptance in §7)
**Target:** MIT-licensed, free, self-deployable video-conferencing tool (infra/cloud)
**Doc convention:** follows existing `docs/*.md` (all-caps) layout; no ADR dir in repo.
**Lineage check:** no `docs/adr/22440*` marker, no `scripts/check-decision-lineage.sh` → decision-lineage `inactive`, standard doc behavior.

## 1. Current state (researched, grounded)

- **Arch:** 2 Go 1.23 services, no root `go.mod`: `server/signalling-server` :8080 (gin, `POST /session`, `GET /connect`, `POST /connect/:url`, `GET /health`, `GET /ws/:socket`) + `server/users-service` :8081 (`POST /auth`, CRUD `/users`, `GET /health`); React18+TS client (`client/src/App.tsx`, `videoconf-sdk.ts`, proxy :8080); `mongo:latest`; nginx LB :80.
- **Signaling/media/auth:** custom gorilla-websocket relay hub, **no media server** — media is browser-native P2P (`RTCPeerConnection`/`getUserMedia`, Google STUN only). Sessions via bcrypt + SHA1-hashed socket URLs + password check. Config via `PORT`/`DB_*`/`JWT_*` env; JWT via deprecated `dgrijalva/jwt-go`, users DB via legacy `mgo.v2`. `sdk/` is an **empty submodule** (`r3tr056/videoconf-sdk`); real SDK code lives in `client/src/videoconf-sdk.ts`.
- **Deploy/test/docs:** `docker-compose.yml` (signalling/users/mongo/client/nginx); `nginx.conf` routes `/`, `/session`, `/connect`, `/ws/`, `/api/*` (note: stale `server:8080` upstream); `.deployment/` k8s manifests + 3 scripts; `Makefile` build/test/lint/bench; `ci-cd.yml` (Go tests, compose integration, trivy/codeql, ghcr push, tag release); signalling unit/bench/stress tests, users mocked tests, `tests/integration` (needs live backends); `README.md`, `API_DOCUMENTATION.md`, `docs/DEPLOYMENT.md`, `docs/SDK_DOCUMENTATION.md`. License already MIT.
- **Docs-vs-reality gaps:** docs claim "Google Meet level", adaptive bitrate, noise suppression, virtual backgrounds, analytics, Redis/CDN/Prometheus — none evidenced in code; `docs/DEPLOYMENT.md` shows TURN/STUN + Redis + Prometheus that compose does not provide.

## 2. OSS landscape (reference points)

- **Jitsi Meet** (Apache-2.0): full SFU (JVB + Prosody + Jicofo + nginx), Docker/Debian, E2EE/SIP/recording/mobile, Octo cascade scaling; cost = XMPP ops complexity, Jibri for recording.
- **LiveKit** (Apache-2.0): Go SFU, modern SDKs, E2EE, recording/egress, cloud-native; heavier than a relay but closest Go-stack comparator.
- **BigBlueButton** (LGPL): mediasoup/Kurento-based, classroom-focused; richer than needed here.
- **Janus / mediasoup / Pion / Galene:** toolkits/SFU cores, not full products. **Pion** (Go, MIT) fits this stack best for a future SFU; **Galene** (Go SFU) is the closest "small self-hostable Go" proof point.
- **This repo's position:** simplest to run (relay-only P2P) but lacks evidenced SFU multiparty scale, recording, SIP, E2EE, mobile, and TURN.

## 3. Trends / market requirements 2025–2026

- Multiparty expectation = SFU (not mesh P2P) beyond ~4–6 participants; mesh + STUN-only fails on quality/scale/NAT.
- Self-host demand: one-command compose, pinned images, Helm/K8s path, TURN bundled, sane defaults + env docs.
- Security/compliance baseline: maintained JWT lib, bcrypt/argon2, TLS/WSS, E2EE story (or explicit non-goal), SBOM/scans, no `latest` tags.
- Operability: health/readiness, Prometheus metrics, structured logs, backup/restore (Mongo), upgrade notes.
- UX/features: recording/transcription/AI notes, moderation, screen share, mobile web, SIP/HLS interop — must explicitly scope in/out for v1.

## 4. Production gaps blocking a real release

1. No SFU/MCU; no TURN (Google STUN only) → NAT failures + no group-call story.
2. Supply chain: empty `sdk/` submodule, stale `server/Dockerfile` (go1.19), `mongo:latest` + `nginx:latest` unpinned, deprecated jwt-go + legacy mgo.v2, SHA1 URL hashing, committed binaries (`signalling-server`, `users-service`).
3. Config/secret hygiene: default `JWT_SECRET your-secret-key`, root passwords in compose, inconsistent env names per service.
4. Contract risk: signalling/users split with integration tests requiring live backends; nginx upstream mismatch (`server:8080`).
5. Verification: no release gate defined (coverage, load/N-party targets, browser matrix, soak); docs overclaim vs code.

## 5. Decisions (settled during grill — Final)

| ID | Decision | Status | Rationale |
|----|----------|--------|-----------|
| D1 | v1.0 media scope: **harden P2P + signalling; SFU seam defined, full SFU deferred to v2** | **SETTLED (Final)** | Fastest verified self-deployable release; SFU gets its own design cycle in v2 |
| D2 | Packaging targets: **Compose one-command (primary) + Helm chart; pinned versioned images on ghcr.io** | **SETTLED (Final)** | Covers infra + cloud with standard tooling; graduates existing compose + .deployment/ assets |
| D3 | Auth/identity model: **harden existing users-service JWT (maintained lib, secret hygiene); OIDC a v2 seam** | **SETTLED (Final)** | Keeps v1 self-contained/offline-friendly; fixes deprecated jwt-go + default secrets without IdP scope |
| D4 | Persistence: **keep MongoDB, unify on mongo-driver, pin image, document replica-set + backup/restore; no Redis in v1** | **SETTLED (Final)** | Least migration risk; fixes legacy driver + latest tag + missing ops docs |
| D5 | Media adjuncts in v1: **bundle TURN (coturn) + require TLS/WSS; recording/E2EE-additional explicit non-goals with v2 story** | **SETTLED (Final)** | TURN required for P2P across NATs; recording/E2EE deferred to bound v1 scope |
| D6 | SDK shape: **vendor SDK in-repo as real npm package (`sdk/` source of truth, client consumes it, publish `@videoconf/sdk` on release)** | **SETTLED (Final)** | Empty submodule breaks fresh clones; vendoring makes repo self-contained and releases reproducible |
| D7 | Release/verification bar: **full gate — unit + live integration + headless-browser N-party media smoke (2/4/6) + Chrome/Firefox matrix + load/soak + trivy/codeql + pinned SBOM; tags release only on green** | **SETTLED (Final)** | Matches "rigorously tested, release ready"; covers media failure seams (NAT, N-party, browsers) |

## 6. v1.0 implementation plan (task-level)

### Phase 0 — Hygiene + supply chain ✅ DONE (verified 2026-09-16)
- [x] P0.1 Pin images. Recorded pins: `mongo:7.0` (compose, k8s manifest, CI services), `nginx:1.27-alpine` (compose LB, client image), `golang:1.23-alpine` + `alpine:3.21` (Go images), `node:18` kept (client build). Zero `:latest`/bare tags.
- [x] P0.2 Deleted committed binaries + stale `server/Dockerfile` + dead code (`signal.go`, `consul_util.go`, `errors.go`; consul dep dropped); `.gitignore` covers binaries, `client/build/`, `sdk/docs/`.
- [x] P0.3 `server/Dockerfile` deleted (unreferenced; per-service Dockerfiles are canonical).
- [x] P0.4 Env unified (`DB_URL`→`DB_HOST`, explicit DB creds, dropped unused `HOST_URL`); `.env.example` added; `JWT_SECRET` fail-fast in code (proven) + compose `:?` gate; CI compose step gets a test-only secret.
- [x] P0.5 Dead `videoconf_backend` upstream (`server:8080`) removed; all nginx hosts/upstreams resolve against compose (validated).
- [x] P0.6 `dgrijalva/jwt-go` → `golang-jwt/jwt/v5` (code + tests); issuer default `VideoConf`.
- [x] P0.7 `mgo.v2` → `mongo-driver` v1.17.10 both services; hex-string `id` JSON shape preserved; 404 semantics preserved via matched/deleted counts; mongo auth via `admin` AuthSource.
- [x] P0.8 SHA-256 replaces SHA1 for session/socket URL hashing.
- **Acceptance:** ✅ `go build/vet/test` (+`-race`) green in both services; compose YAML valid with all nginx refs resolving (`docker compose config` binary unavailable in sandbox — structural check instead); zero `:latest`; zero tracked secrets; fresh-clone builds proven for sdk+client, Go via `go mod tidy`+build.

### Phase 1 — SDK + docs truth (D6) ✅ DONE (verified 2026-09-16)
- [x] P1.1 `sdk/` is a real `@videoconf/sdk` package (build/type-check/lint/test:coverage/docs); submodule pointer removed; 18/18 jest tests, ~94% coverage.
- [x] P1.2 Client consumes `file:../sdk` (typed import in `App.tsx`, old inline file deleted, wiring smoke test); client tests + production build green; client Dockerfile builds SDK first (root context).
- [x] P1.3 CI builds/tests SDK and publishes on tags (publish dry-run proven; `test-client` builds the SDK prerequisite first).
- [x] P1.4 Docs rewritten to the real API (`SDK_DOCUMENTATION.md`, `docs/README.md`, root `README.md`, `API_DOCUMENTATION.md`); N-party ceiling + v2 roadmap documented.
- **Acceptance:** ✅ `make build`/`make test` dependency chain proven per-component from a clean tree (sdk `npm ci`+build+test, client `npm ci`+test+build, Go build+test); full `make` run lands in Phase 6 with live backends.

### Phase 2 — Media hardening (D1+D5) ✅ DONE (verified 2026-09-16)
- [x] P2.1 `coturn/coturn:4.6.3` in compose (3478 + UDP relay range, realm/user from env, WAN external-ip documented); client bakes `TURN_PUBLIC_URL/USER/PASS` at build; k8s/Helm TURN values ride with the Phase 3 chart.
- [x] P2.2 TLS template in `nginx.conf` (443 + HSTS, cert mounts) + TLS runbook in `docs/DEPLOYMENT.md`; SDK derives `wss://` from https pages; `WS_ALLOWED_ORIGINS` gates production origins.
- [x] P2.3 Signalling hardened: `MeshRelay` (mutex, targeted `to` routing, dead-peer prune), two-way presence (newcomer replay fixes mesh bootstrap), ping/pong heartbeat, read limits, per-conn rate limit, 8-member cap (429), origin allowlist + restrictive CORS, message validation, spoof-drop, unclean-drop farewell, controller error paths fixed. SDK: get-or-create PCs, late-media renegotiation, reconnect backoff (reconnecting/reconnected/disconnected), malformed/glare guards.
- [x] P2.4 `MessageRouter` seam (interface + `MeshRelay` impl, DI into handler) + doc in `API_DOCUMENTATION.md`.
- [x] P2.5 Ceiling documented (recommend ≤6, cap 8) in `DEPLOYMENT.md`/API/SDK docs + v2 SFU pointer; `DEPLOYMENT.md` rewritten to truth (compose/TURN/TLS/env/capacity).
- **Acceptance:** ✅ signalling unit/stress + new WS regression suite race-green; SDK 26/26; client test+build green; compose valid with all env cross-refs documented. Live 1-1/4-party + TURN-relay proof runs at the Phase 6 gate (P6.1), which owns live browsers/backends — no docker or browsers exist in this sandbox.

### Phase 3 — Packaging (D2) ✅ DONE (verified 2026-09-16)
- [x] P3.1 `scripts/setup-env.sh` (randomized `.env`, no-clobber + `--force`, `make setup-env`) + README quickstart; `docker-compose` v1 → `docker compose` everywhere.
- [x] P3.2 `helm/videoconf` chart (5 deployments, 4 services, ingress+TLS, 2 HPAs, PVC, secret with `required()` fail-fast, turn-config ConfigMap, NOTES); legacy `.deployment/` deleted; `make deploy-k8s/undeploy-k8s` reimplemented via helm. Verified: `helm lint` clean, full render + TLS variant, secrets refusal.
- [x] P3.3 CI builds all 3 images with correct contexts (client root context + TURN build args), semver+sha tags to ghcr.io; TURN config moved to runtime `turn-config.json` (baked for compose, ConfigMap-mounted for Helm) so prebuilt images work per-environment; install docs in README + DEPLOYMENT (incl. TURN/TLS runbooks).
- **Acceptance:** ✅ structural (compose validates, chart lints+renders, images build in CI, client test+build green). Live install proof at the Phase 6 gate: compose e2e + kind-based helm smoke (P6.3).

### Phase 4 — Persistence ops (D4) ✅ DONE (verified 2026-09-16)
- [x] P4.1 Unified driver verified under load: env-gated live suites (`VIDEOCONF_LIVE_MONGO`) run in CI against mongo:7.0 and locally — full session-join flow, user HTTP flow incl. JWT, 500-op concurrent storms per service, all green. Indexes: `sockets.hashedUrl` unique + `users.name`, created idempotently at startup and asserted in tests. Auth now queries by indexed name (no collection scan). Live testing caught and fixed a real join-breaking bug: missing BSON tags lowercased `hashedUrl` so joins never matched.
- [x] P4.2 Runbooks in `docs/DEPLOYMENT.md` (indexes, replica-set HA procedure, upgrades). Restore drill executed for real: seeded user+session via live APIs → `mongodump` → drop (APIs 400/401) → `mongorestore` → identical socket URLs + working auth.
- [x] P4.3 Upgrade notes in runbook (7.0 pinned, mgo→driver BSON compat, index-build guidance, CI Go 1.22→1.23 fix required by go.mod).
- **Acceptance:** ✅ drill passes against real mongod 7.0.14 + both live services (compose/k8s variants are the same commands against the packaged mongo; Helm chart ships single-instance mongo with the replica-set runbook for HA).

### Phase 5 — Operability + security ✅ DONE (verified 2026-09-16)
- [x] P5.1 `/health` (shallow) + `/ready` (mongo ping, 503 when down) on both services; Prometheus `/metrics` (RED + WS gauges/counters); `log/slog` JSON to stdout (file logging removed); `GIN_MODE=release` in compose+chart; chart probes use `/ready`; compose log rotation.
- [x] P5.2 Trivy pinned (0.36.0) + enforced (CRITICAL/HIGH, vuln+secret, exit 1); CodeQL fixed (init/autobuild/analyze v3, was broken analyze-only); image provenance+SBOM attestations; per-image SPDX SBOMs attached to releases; Go toolchain pinned 1.23. Triage: x/crypto bumped (both), npm `audit fix` + underscore override, 6 build-only CVEs in `.trivyignore` with per-ID justification (never shipped; future advisories still alert).
- [x] P5.3 Secret docs (`.env` 600 via setup script, rotation, Helm `--set` vs `existingSecret` — implemented + rendered both ways, sealed/ESO pointers) + vulnerability policy in DEPLOYMENT.
- **Acceptance:** ✅ `/ready`+`/metrics` scraped live with real traffic; trivy gate exit 0 locally (= CI gate); SBOM/release steps wired (release execution is the owner's tag push; CodeQL runs in CI).

### Phase 6 — Verification gate + v1.0 release (D7)
- [ ] P6.1 Live-backend integration in CI; headless-browser N-party smoke (2/4/6) incl. TURN-relay verification (relay candidate present, media flows via coturn); Chrome+Firefox matrix. NOTE: current `tests/integration` is non-functional and must be rewritten — no `go.mod`, dials `/ws?session=` instead of `/ws/:socket`, uses `{type,data,from,to}` instead of `{type,userID,description,candidate,to}`, mocks session creation, and posts `/register` (no such route).
- [ ] P6.2 Signalling load/soak targets defined + met; release-on-green-tags enforced.
- [ ] P6.3 Full release drill → GitHub release v1.0.0 (images + SDK + SBOM + notes); kind-based helm install smoke (chart installs, pods ready, /health green through ingress).
- **Acceptance:** tag pipeline green; artifacts verified installable per Phase 3 acceptance.

### v2 deferred proposals (NOT approved — each needs its own interview)
Full SFU build · recording · E2EE-additional · OIDC · Redis session store · single-binary/offline bundle.

## 7. Scope contract (ACCEPTED)

- **Artifact boundary (grill):** deliverable was this plan doc only. Out of scope then: runtime code, tests, CI/infra edits, publishes, follow-on specs.
- **Implementation authorization:** full v1 implementation (Phases 0–6) explicitly authorized by the user (see Acceptance). v2 stages remain deferred proposals.
- **Done means:** D1–D7 settled + recorded ✓; scope text explicitly accepted ✓; doc Final ✓; limitations documented as-is ✓.
- **Staged approval:** accepting this plan never approves v2 stages. "Go" authorizes only the accepted boundary.
- **Acceptance:** user replied "accepted" (session chat, 2026-09-16), adding: "and now make the full implementation plan and setup the implementation plan in the task system and the goal system. The goal is a full end to end implementation of the full implementation plan scope." — scope contract accepted; doc flipped Draft → Final; v1 Phases 0–6 implementation authorized.

## 8. Unresolved / limitations

- Q1–Q8 all settled (D1–D7 + scope acceptance).
- BBB/mediasoup comparison detail was truncated in the research sweep — documented as-is; non-blocking for v1 decisions.
- Exact N-party ceiling (~4–6) to be verified empirically in Phase 6.
