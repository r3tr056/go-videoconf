# VideoConf API Documentation

## Overview

VideoConf provides a self-hostable video conferencing solution with the following components:

- **Signalling Server**: Handles WebRTC signalling and session management
- **Users Service**: Manages user authentication and user data
- **Client SDK**: TypeScript SDK for WebRTC integration (`@videoconf/sdk`)
- **Frontend Client**: React-based demo web application

## API Endpoints

### Signalling Server (Port 8080)

#### Create Session
```
POST /session
Content-Type: application/json

{
  "host": "user-id",
  "title": "Meeting Title",
  "password": "meeting-password"
}

Response:
{
  "socket": "generated-socket-url"
}
```

#### Join Session
```
POST /connect/{sessionUrl}
Content-Type: application/json

{
  "password": "meeting-password"
}

Response:
{
  "title": "Meeting Title",
  "socket": "socket-url-for-connection"
}
```

#### WebSocket Connection
```
GET /ws/{socketUrl}
```

WebSocket messages (JSON, relayed between peers on the same socket):
- `connect`: Join the session (sent on socket open)
- `offer`: WebRTC offer (`description` holds JSON SDP)
- `answer`: WebRTC answer (`description` holds JSON SDP)
- `ice-candidate`: ICE candidate (`candidate` holds JSON)
- `disconnect`: Leave the session

Message envelope: `{ "type", "userID", "description?", "candidate?", "to?" }`.
Messages with `to` route only to that member; `connect`/`disconnect` broadcast.
Rooms are capped at 8 members (HTTP 429 past the cap); ~2–6 is recommended.

The handler routes everything through the `MessageRouter` interface
(`MeshRelay` in v1): the seam a v2 SFU implements to take over media
switching without changing the WebSocket handler.

#### Health Check
```
GET /health

Response:
{
  "status": "healthy",
  "service": "signalling-server"
}
```

### Users Service (Port 8081)

#### Authentication
```
POST /auth
Content-Type: application/json

{
  "username": "user",
  "password": "password"
}

Response:
{
  "token": "jwt-token",
  "user": {
    "id": "user-id",
    "name": "username"
  }
}
```

#### User Management
```
GET /users          - Get all users
GET /users/{id}     - Get user by ID
POST /users         - Create new user
PUT /users/{id}     - Update user
DELETE /users/{id}  - Delete user
```

#### Health Check
```
GET /health

Response:
{
  "status": "healthy",
  "service": "users-service"
}
```

## JavaScript SDK Usage

The SDK publishes to GitHub Packages as `@videoconf/sdk` (see
[SDK documentation](docs/SDK_DOCUMENTATION.md) for the full reference).

### Basic Usage

```javascript
// .npmrc: @videoconf:registry=https://npm.pkg.github.com
import { VideoconfSDK } from '@videoconf/sdk';

// Initialize SDK
const videoconf = new VideoconfSDK('your-api-key', 'project-id', {
  signallingServerUrl: 'wss://meet.example.com',
});

// Create a session (reserves the room, returns the invite code)
const sessionUrl = await videoconf.createSession('Meeting Title', 'password');

// Join it: even the host must join, otherwise nobody can signal with them.
// Guests call this with the code the host shared.
await videoconf.joinSession(sessionUrl, 'password');

// Initialize local media
await videoconf.initializeCall();

// Get local stream
const localStream = videoconf.getLocalStream();

// Listen for events
videoconf.on('local-stream', (stream) => {
  console.log('Local stream ready');
});

videoconf.on('remote-stream', (stream, peerId) => {
  console.log('New participant stream:', peerId);
});

videoconf.on('stream-removed', (peerId) => {
  console.log('Participant left:', peerId);
});

// Control media
videoconf.toggleVideo(false);  // Disable video
videoconf.toggleAudio(false);  // Disable audio

// Leave call
await videoconf.leaveCall();
```

### React Integration

```jsx
import React, { useEffect, useState } from 'react';
import { VideoconfSDK } from '@videoconf/sdk';

function VideoCall() {
  const [sdk, setSdk] = useState(null);
  const [localStream, setLocalStream] = useState(null);
  const [remoteStreams, setRemoteStreams] = useState(new Map());

  useEffect(() => {
    const videoconf = new VideoconfSDK('api-key', 'project-id');

    videoconf.on('local-stream', setLocalStream);
    videoconf.on('remote-stream', (stream, peerId) => {
      setRemoteStreams(prev => new Map(prev).set(peerId, stream));
    });
    videoconf.on('stream-removed', (peerId) => {
      setRemoteStreams(prev => {
        const next = new Map(prev);
        next.delete(peerId);
        return next;
      });
    });

    setSdk(videoconf);
    return () => { videoconf.leaveCall(); };
  }, []);

  return (
    <div>
      {localStream && (
        <video ref={el => { if (el) el.srcObject = localStream; }}
          autoPlay muted playsInline />
      )}
      {[...remoteStreams].map(([peerId, stream]) => (
        <video key={peerId} ref={el => { if (el) el.srcObject = stream; }}
          autoPlay playsInline />
      ))}
    </div>
  );
}
```

## Deployment

### Docker Compose

```bash
# Start all services
docker compose up -d

# View logs
docker compose logs -f

# Stop services
docker compose down
```

### Kubernetes

```bash
# Deploy with the manifests (Helm chart lands with the v1 packaging work)
kubectl apply -f .deployment/mongo-deployment.yml
kubectl apply -f .deployment/server-deployment.yml
kubectl apply -f .deployment/client-deployment.yml
kubectl apply -f .deployment/ingress.yml
```

See [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md) for the full guide.

## Environment Variables

See `.env.example` for the full documented list.

### Signalling Server
- `PORT`: Server port (default: 8080)
- `DB_HOST`: MongoDB host (default: localhost)
- `DB_PORT`: MongoDB port (default: 27017)
- `DB_USERNAME`: MongoDB username (default: root)
- `DB_PASSWORD`: MongoDB password (default: rootpassword)

### Users Service
- `PORT`: Server port (default: 8081)
- `DB_HOST`: MongoDB host (default: 127.0.0.1)
- `DB_PORT`: MongoDB port (default: 27017)
- `DB_NAME`: Database name (default: vidchat)
- `DB_USERNAME`: MongoDB username (default: root)
- `DB_PASSWORD`: MongoDB password (default: rootpassword)
- `JWT_SECRET`: JWT signing secret (required, no default)
- `JWT_ISSUER`: JWT issuer (default: VideoConf)

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Load Balancer │    │   Users Service │    │Signalling Server│
│    (Nginx)      │    │   (Port 8081)   │    │   (Port 8080)   │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         │                       └───────────────────────┘
         │                              │
         │                    ┌─────────────────┐
         │                    │    MongoDB      │
         │                    │   (Port 27017)  │
         │                    └─────────────────┘
         │
┌─────────────────┐
│   Client App    │
│  (React/SDK)    │
└─────────────────┘
```

Media flows peer-to-peer between browsers; the server relays signalling only.

## Security Considerations

1. **Authentication**: JWT tokens for user authentication (maintained
   `golang-jwt` library)
2. **Password Hashing**: Session passwords are bcrypt-hashed
3. **CORS**: Currently permissive (`cors.DefaultConfig`); the v1 hardening work
   restricts allowed origins for production
4. **WebSocket Security**: Origin check currently accepts all origins; an
   origin allowlist lands with the signalling hardening (Phase 2)
5. **Secrets**: All secrets via environment (see `.env.example`); the server
   fails fast on missing secrets and ships no defaults

## Monitoring

- Health check endpoints available for all services (`GET /health`)
- Readiness endpoints, structured JSON logs, Prometheus metrics, container
  health checks, and Kubernetes probes ship with the v1 deployment work (see
  [implementation plan](docs/IMPLEMENTATION_PLAN.md))

## Scaling

- Services scale horizontally behind Nginx; session state lives in MongoDB
  (each signalling instance fans out its own in-memory WebSocket mesh)
- One signalling instance per deployment (rooms do not span instances)
- MongoDB replication for HA
