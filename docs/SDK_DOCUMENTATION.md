# VideoConf SDK Documentation

Browser TypeScript SDK for mesh P2P video conferencing over the go-videoconf
signalling server. Framework-agnostic (works in React, Angular, Vue, or plain
TypeScript); published as `@videoconf/sdk`. MIT licensed.

Source of truth: [`sdk/`](../sdk/) — tested (jest, ~94% coverage), typed,
and consumed by the demo client in [`client/`](../client/).

## Installation

The package publishes to GitHub Packages on release. Point the `@videoconf`
scope at the registry first:

```bash
# .npmrc
@videoconf:registry=https://npm.pkg.github.com
```

```bash
npm install @videoconf/sdk
```

## Quick Start

```typescript
import { VideoconfSDK } from '@videoconf/sdk';

const sdk = new VideoconfSDK('api-key', 'project-id', {
  signallingServerUrl: 'wss://meet.example.com',
});

sdk.on('local-stream', (stream) => {
  (document.getElementById('local') as HTMLVideoElement).srcObject = stream;
});
sdk.on('remote-stream', (stream, peerId) => {
  console.log('remote stream from', peerId, stream);
});
sdk.on('stream-removed', (peerId) => {
  console.log('peer left', peerId);
});

const socket = await sdk.createSession('Team standup', 'secret');
await sdk.joinSession(socket, 'secret');
await sdk.initializeCall();

// ... later
await sdk.leaveCall();
```

## Configuration

```typescript
new VideoconfSDK(apiKey, projectId, options?)
```

| Option | Type | Default | Description |
|---|---|---|---|
| `signallingServerUrl` | `string` | page origin with `ws(s)` scheme | Signalling server WebSocket base URL |
| `iceServers` | `RTCIceServer[]` | public STUN (`stun.l.google.com:19302`) | ICE servers for NAT traversal |

Production deployments behind NAT should supply the bundled TURN servers:

```typescript
const sdk = new VideoconfSDK('api-key', 'project-id', {
  signallingServerUrl: 'wss://meet.example.com',
  iceServers: [
    { urls: 'turn:turn.example.com:3478', username: 'user', credential: 'pass' },
  ],
});
```

## API Reference

### Session Management

- `createSession(title: string, password: string): Promise<string>` —
  creates a password-protected session; resolves with the session socket id.
- `joinSession(sessionUrl: string, password: string): Promise<void>` —
  joins a session (password required) and opens the signalling WebSocket.
- `leaveCall(): Promise<void>` — notifies peers with `disconnect` and releases
  all media tracks, peer connections, and the socket.

### Media

- `initializeCall(): Promise<MediaStream>` — acquires camera + microphone via
  `getUserMedia({ video: true, audio: true })`; emits `local-stream`.
- `toggleVideo(enabled: boolean): void` / `toggleAudio(enabled: boolean): void` —
  enable/disable the local tracks. Safe to call before `initializeCall` (no-op).
- `getLocalStream(): MediaStream | null`
- `getRemoteStreams(): Map<string, MediaStream>` — returns a copy keyed by peer id.
- `getUserId(): string` — this client's generated id.

### Events (`on(event, listener)`)

| Event | Payload | Fired when |
|---|---|---|
| `local-stream` | `(stream)` | local media acquired |
| `remote-stream` | `(stream, peerId)` | a peer's track arrives |
| `stream-removed` | `(peerId)` | a peer disconnects |
| `reconnecting` | `(attempt)` | signalling dropped; retry scheduled |
| `reconnected` | — | signalling re-established, presence re-sent |
| `disconnected` | — | reconnect backoff exhausted; call cleaned up |

Listener exceptions are caught and logged; they never break other listeners.

## Signalling Protocol

The SDK speaks to `GET /ws/{socket}` with JSON messages:

| `type` | Direction | Purpose |
|---|---|---|
| `connect` | → server | announce presence on socket open |
| `offer` / `answer` | ↔ peers (relayed) | SDP exchange, `description` is JSON |
| `ice-candidate` | ↔ peers (relayed) | trickle ICE, `candidate` is JSON |
| `disconnect` | → server | graceful leave |

The server relays messages; media flows peer-to-peer (mesh). Messages with a
`to` member route only to that member; `connect`/`disconnect` broadcast to the
room. Mesh suits small rooms (a few participants, hard server cap of 8 per
socket); a server-side SFU for larger rooms is a planned v2 addition.

Dropped sockets reconnect automatically (1s/2s/4s/8s/16s backoff): presence is
re-announced and existing peer connections are reused and renegotiated, so a
brief network blip does not end the call. `leaveCall()` suppresses
reconnecting. Known limitation: two peers joining in the same instant can
cross offers (glare); the SDK logs and continues, and rejoining resolves it.

## React Integration

```jsx
import React, { useEffect, useState } from 'react';
import { VideoconfSDK } from '@videoconf/sdk';

function VideoCall({ sessionUrl, password }) {
  const [sdk, setSdk] = useState(null);
  const [localStream, setLocalStream] = useState(null);
  const [remoteStreams, setRemoteStreams] = useState(new Map());

  useEffect(() => {
    const client = new VideoconfSDK('api-key', 'project-id');
    client.on('local-stream', setLocalStream);
    client.on('remote-stream', (stream, peerId) => {
      setRemoteStreams((prev) => new Map(prev).set(peerId, stream));
    });
    client.on('stream-removed', (peerId) => {
      setRemoteStreams((prev) => {
        const next = new Map(prev);
        next.delete(peerId);
        return next;
      });
    });
    (async () => {
      await client.joinSession(sessionUrl, password);
      await client.initializeCall();
    })();
    setSdk(client);
    return () => { client.leaveCall(); };
  }, [sessionUrl, password]);

  return (
    <div>
      {localStream && (
        <video
          ref={(el) => { if (el) el.srcObject = localStream; }}
          autoPlay muted playsInline
        />
      )}
      {[...remoteStreams].map(([peerId, stream]) => (
        <video
          key={peerId}
          ref={(el) => { if (el) el.srcObject = stream; }}
          autoPlay playsInline
        />
      ))}
    </div>
  );
}
```

See [`client/src/App.tsx`](../client/src/App.tsx) for the full working demo.

## Browser Support

Any modern browser with WebRTC + WebSocket (Chrome and Firefox are covered by
the release gate). Camera/mic access requires a secure context (HTTPS or
localhost). The SDK is browser-only; it has no Node.js runtime support.

## Roadmap (v2, not in this release)

Server-side SFU for large rooms · recording · additional end-to-end encryption ·
screen-sharing helpers · chat signalling.
