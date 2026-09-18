# @videoconf/sdk

Browser WebRTC SDK for P2P video conferencing over the go-videoconf signalling
server. MIT licensed. This package is the single source of truth for the SDK;
the demo client consumes it as a dependency.

## Install

```bash
npm install @videoconf/sdk
```

## Usage

```typescript
import { VideoconfSDK } from '@videoconf/sdk';

const sdk = new VideoconfSDK('api-key', 'project-id', {
  signallingServerUrl: 'wss://meet.example.com',
  iceServers: [{ urls: 'turn:turn.example.com:3478', username: 'u', credential: 'p' }],
});

sdk.on('local-stream', (stream) => {
  (document.getElementById('local') as HTMLVideoElement).srcObject = stream;
});
sdk.on('remote-stream', (stream, peerId) => console.log('remote', peerId, stream));
sdk.on('stream-removed', (peerId) => console.log('left', peerId));

const socket = await sdk.createSession('Team standup', 'secret');
await sdk.joinSession(socket, 'secret');
await sdk.initializeCall();
// ... later
await sdk.leaveCall();
```

## API

- `new VideoconfSDK(apiKey, projectId, options?)` — `options.signallingServerUrl`
  defaults to the page origin with `ws(s)` scheme; `options.iceServers`
  defaults to public STUN (supply TURN for production NAT traversal).
- `createSession(title, password): Promise<string>` — returns the session socket id.
- `joinSession(sessionUrl, password): Promise<void>` — joins via signalling + WebSocket.
- `initializeCall(): Promise<MediaStream>` — acquires camera/mic, emits `local-stream`.
- `leaveCall(): Promise<void>` — notifies peers and releases all media/sockets.
- `toggleVideo(enabled)` / `toggleAudio(enabled)` — enable/disable local tracks.
- `getLocalStream()` / `getRemoteStreams()` / `getUserId()` — accessors.
- `on(event, listener)` — subscribe to `local-stream`, `remote-stream`,
  `stream-removed`, plus `reconnecting` (attempt), `reconnected`, and
  `disconnected` for signalling drops.

Dropped signalling sockets reconnect automatically (1s/2s/4s/8s/16s backoff,
presence re-announced, existing peer connections reused); `leaveCall()`
suppresses reconnecting. Offers, answers, and ICE candidates route only to
their `to` member; `connect`/`disconnect` broadcast to the room.

Mesh P2P supports small rooms (a few participants); a server-side SFU for
larger rooms is a planned v2 addition.
