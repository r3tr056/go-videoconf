# VideoConf API Documentation

## Overview

VideoConf provides a complete video conferencing solution with the following components:

- **Signalling Server**: Handles WebRTC signalling and session management
- **Users Service**: Manages user authentication and user data
- **Client SDK**: JavaScript SDK for WebRTC integration
- **Frontend Client**: React-based web application

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

WebSocket messages:
- `connect`: Join the session
- `offer`: WebRTC offer
- `answer`: WebRTC answer
- `ice-candidate`: ICE candidate
- `disconnect`: Leave the session

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

### Basic Usage

```javascript
// Import the SDK
import { VideoconfSDK, VideoContainer } from 'videoconf-sdk';

// Initialize SDK
const videoconf = new VideoconfSDK('your-api-key', 'project-id');

// Create a session
const sessionUrl = await videoconf.createSession('Meeting Title', 'password');

// Or join an existing session
await videoconf.joinSession('session-url', 'password');

// Initialize local media
await videoconf.initializeCall();

// Get local stream
const localStream = videoconf.getLocalStream();

// Listen for events
videoconf.on('stream-added', (stream, peerId) => {
  console.log('New stream added:', peerId);
});

videoconf.on('stream-removed', (peerId) => {
  console.log('Stream removed:', peerId);
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
import { VideoconfSDK, VideoContainer, VideoGrid } from 'videoconf-sdk';

function VideoCall() {
  const [sdk, setSdk] = useState(null);
  const [localStream, setLocalStream] = useState(null);
  const [remoteStreams, setRemoteStreams] = useState(new Map());

  useEffect(() => {
    const videoconf = new VideoconfSDK('api-key', 'project-id');
    
    videoconf.on('stream-added', (stream, peerId) => {
      if (peerId === 'local') {
        setLocalStream(stream);
      } else {
        setRemoteStreams(prev => new Map(prev).set(peerId, stream));
      }
    });

    setSdk(videoconf);
  }, []);

  return (
    <div>
      <VideoGrid 
        streams={remoteStreams}
        localStream={localStream}
      />
    </div>
  );
}
```

## Deployment

### Docker Compose

```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f

# Stop services
docker-compose down
```

### Kubernetes

```bash
# Deploy database
kubectl apply -f .deployment/mongo-deployment.yml

# Deploy services
kubectl apply -f .deployment/server-deployment.yml
kubectl apply -f .deployment/client-deployment.yml

# Deploy ingress
kubectl apply -f .deployment/ingress.yml
```

## Environment Variables

### Signalling Server
- `PORT`: Server port (default: 8080)
- `DB_URL`: MongoDB host (default: localhost)
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
- `JWT_SECRET`: JWT signing secret
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

## Security Considerations

1. **Authentication**: JWT tokens for user authentication
2. **Password Hashing**: Session passwords are hashed
3. **CORS**: Configured for cross-origin requests
4. **WebSocket Security**: Origin validation for WebSocket connections
5. **Environment Variables**: Sensitive data stored in environment variables

## Monitoring

- Health check endpoints available for all services
- Structured logging to files
- Docker container health checks
- Kubernetes readiness and liveness probes

## Scaling

- Horizontal scaling supported via Kubernetes
- Load balancing through Nginx
- Stateless service design
- Shared MongoDB for session persistence