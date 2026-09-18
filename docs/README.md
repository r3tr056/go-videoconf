# VideoConf Documentation

## SDK

The [`@videoconf/sdk`](../sdk/) package is the browser TypeScript SDK for mesh
P2P video conferencing over the go-videoconf signalling server.

```typescript
import { VideoconfSDK } from '@videoconf/sdk';

const sdk = new VideoconfSDK('api-key', 'project-id', {
  signallingServerUrl: 'wss://meet.example.com',
});

sdk.on('remote-stream', (stream, peerId) => {
  (document.getElementById(`video-${peerId}`) as HTMLVideoElement).srcObject = stream;
});

const socket = await sdk.createSession('Team standup', 'secret');
await sdk.joinSession(socket, 'secret');
await sdk.initializeCall();
```

Full reference: [SDK documentation](./SDK_DOCUMENTATION.md).

## Server APIs

REST + WebSocket reference: [API documentation](../API_DOCUMENTATION.md).

## Deployment

Self-hosting guide (Compose, Kubernetes/Helm, TURN, TLS, Mongo ops):
[deployment guide](./DEPLOYMENT.md).

## Release Plan

Production-readiness plan with decisions, phases, and verification gates:
[implementation plan](./IMPLEMENTATION_PLAN.md).
