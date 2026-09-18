# Videoconf - Self-Hosted Open-Source Video Conferencing 🚀

Videoconf is a free, MIT-licensed video conferencing platform you can self-deploy
on your own infrastructure or cloud. It provides a Go signalling backend, a
browser WebRTC SDK, and a React demo client — with one-command Docker Compose
and Helm/Kubernetes deployment.

## 🌟 Features

- **Easy-to-use SDK and REST APIs** for video conferencing integration
- **Framework-agnostic browser SDK** (React demo included; works with any web stack)
- **Browser-native WebRTC** (no plugins) with mesh P2P for small rooms
- **Simple two-service Go backend** (signalling + users) with MongoDB
- **Password-protected sessions** and JWT-authenticated user management
- **Docker Compose and Kubernetes/Helm** deployment with pinned images
- **Bundled TURN server** for reliable NAT traversal
- **TLS/WSS** production path with Nginx/ingress termination
- **Health endpoints**, structured logs, and Prometheus metrics

## 🛠️ Tech Stack

- **Backend**: Golang with Gin framework
- **Database**: MongoDB
- **Frontend**: React with TypeScript
- **SDK**: TypeScript (`@videoconf/sdk`)
- **WebRTC**: Native browser WebRTC APIs (mesh P2P)
- **NAT traversal**: STUN + bundled coturn TURN server
- **Deployment**: Docker Compose, Kubernetes/Helm
- **Load Balancer**: Nginx
- **Authentication**: JWT tokens (users service), session passwords

## 🏗️ Architecture

Videoconf consists of five main components:

1. **Signalling Server** (Go): Handles WebRTC signalling and session management
2. **Users Service** (Go): Manages user authentication and user data
3. **Client SDK** (TypeScript): Provides easy WebRTC integration
4. **Frontend Client** (React): Demo web application
5. **MongoDB Database**: Stores user and session data

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
Mesh P2P suits small rooms (a few participants); a server-side SFU for larger
rooms is on the v2 roadmap.

## 🚦 Getting Started

### Prerequisites

- **Docker & Docker Compose** (recommended)
- **Go 1.23+** (for local development)
- **Node.js 18+** (for local development)
- **MongoDB** (for local development)

### 🐳 Quick Start with Docker

1. **Clone the repository:**
   ```bash
   git clone https://github.com/r3tr056/go-videoconf.git
   cd go-videoconf
   ```

2. **Generate secrets:**
   ```bash
   ./scripts/setup-env.sh
   ```

3. **Start the entire stack:**
   ```bash
   docker compose up --build -d
   ```

4. **Access the application:**
   - Web Client: http://localhost
   - Signalling API: http://localhost:8080
   - Users API: http://localhost:8081

### 🔧 Local Development

1. **Install dependencies:**
   ```bash
   make install-deps
   ```

2. **Build all services:**
   ```bash
   make build
   ```

3. **Run tests:**
   ```bash
   make test
   ```

4. **Start development environment:**
   ```bash
   make dev
   ```

### ☸️ Kubernetes Deployment

1. **Setup Kubernetes cluster**

2. **Deploy with Helm:**
   ```bash
   make deploy-k8s JWT_SECRET=$(openssl rand -base64 48) TURN_PASSWORD=$(openssl rand -base64 32) \
     INGRESS_HOST=meet.example.com TURN_PUBLIC_URL=turn:meet.example.com:3478
   ```

3. **Check deployment status:**
   ```bash
   kubectl get pods
   kubectl get services
   ```

See [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md) for the full deployment guide.

## 📚 SDK Usage

### Installation

The SDK publishes to GitHub Packages. Point the scope at the registry first:

```bash
# .npmrc
@videoconf:registry=https://npm.pkg.github.com
```

```bash
npm install @videoconf/sdk
```

### Basic Usage

```javascript
import { VideoconfSDK } from '@videoconf/sdk';

// Initialize SDK
const videoconf = new VideoconfSDK('your-api-key', 'project-id', {
  signallingServerUrl: 'wss://meet.example.com',
});

// Create a meeting
const sessionUrl = await videoconf.createSession('My Meeting', 'password123');

// Or join existing meeting
await videoconf.joinSession(sessionUrl, 'password123');

// Initialize camera and microphone
await videoconf.initializeCall();

// Handle events
videoconf.on('local-stream', (stream) => {
  document.getElementById('local-video').srcObject = stream;
});

videoconf.on('remote-stream', (stream, peerId) => {
  console.log('New participant stream:', peerId);
});

videoconf.on('stream-removed', (peerId) => {
  console.log('Participant left:', peerId);
});

// Control media
videoconf.toggleVideo(false);  // Turn off camera
videoconf.toggleAudio(false);  // Mute microphone

// Leave meeting
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

See [docs/SDK_DOCUMENTATION.md](docs/SDK_DOCUMENTATION.md) for the full SDK reference.

## 📖 API Documentation

### Session Management

**Create Session:**
```http
POST /session
Content-Type: application/json

{
  "host": "user-id",
  "title": "Meeting Title",
  "password": "meeting-password"
}
```

**Join Session:**
```http
POST /connect/{sessionUrl}
Content-Type: application/json

{
  "password": "meeting-password"
}
```

**WebSocket Connection:**
```
ws://localhost:8080/ws/{socketUrl}
```

See [API_DOCUMENTATION.md](API_DOCUMENTATION.md) for complete API reference.

## 🧪 Testing

```bash
# Run all tests
make test

# Run specific service tests
cd server/signalling-server && go test -v
cd server/users-service && go test -v
```

## 🔍 Health Monitoring

All services provide health check endpoints:

- Signalling Server: `GET /health`
- Users Service: `GET /health`
- Load Balancer: `GET /` (proxies to services)

```bash
# Check all services
make check-health
```

## 🐛 Debugging

**View logs:**
```bash
docker compose logs -f
```

**Access individual services:**
```bash
# Signalling server logs
docker compose logs signalling-server

# Users service logs
docker compose logs users-service

# Client logs
docker compose logs videoconf-client
```

## 🚀 Production Deployment

### Environment Variables

See `.env.example` for the full documented list. Key variables:

**Signalling Server:**
- `PORT`: Server port (default: 8080)
- `DB_HOST`: MongoDB host
- `DB_PORT`: MongoDB port
- `DB_USERNAME`: MongoDB username
- `DB_PASSWORD`: MongoDB password

**Users Service:**
- `PORT`: Server port (default: 8081)
- `DB_HOST`: MongoDB host
- `DB_NAME`: Database name
- `JWT_SECRET`: JWT signing secret (required, no default)

### Security Considerations

- JWT authentication for users
- Session password hashing
- CORS configuration
- WebSocket origin validation
- Environment-based secrets (never commit real secrets)

### Scaling

- Horizontally scalable services behind Nginx (one signalling instance per
  deployment fans out its own WebSocket mesh; session state lives in MongoDB)
- MongoDB replication for HA

## 🔧 Development Tools

**Available Make commands:**
```bash
make help          # Show all available commands
make build         # Build all services
make test          # Run tests
make clean         # Clean build artifacts
make dev           # Start development environment
make docker-up     # Start with Docker
make deploy-k8s    # Deploy to Kubernetes
make lint          # Run linters
make format        # Format code
```

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 📞 Support

- 📧 Email: [support@ankurdebnath.me](mailto:support@ankurdebnath.me)
- 🐛 Issues: [GitHub Issues](https://github.com/r3tr056/go-videoconf/issues)
- 📖 Documentation: [API Docs](API_DOCUMENTATION.md)

## 🙏 Acknowledgments

- [Gin Web Framework](https://github.com/gin-gonic/gin)
- [MongoDB Go Driver](https://github.com/mongodb/mongo-go-driver)
- [WebRTC](https://webrtc.org/)
- [React](https://reactjs.org/)
- [TypeScript](https://www.typescriptlang.org/)

---

**Made with ❤️ by [Ankur Debnath](https://github.com/r3tr056)**
