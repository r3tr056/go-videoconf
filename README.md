# Videoconf - Video Conferencing as a Service (AaaS) 🚀

Videoconf is a robust, production-ready Video Conference API as a Service platform that provides seamless integration of video conferencing capabilities into JavaScript (React, Angular, etc.) and Node.js applications.

## 🌟 Features

- **Easy-to-use APIs and SDKs** for video conferencing integration
- **Support for multiple JavaScript frameworks** (React, Angular, etc.)
- **Node.js compatibility** with TypeScript SDK
- **Scalable microservice architecture** with Go backend
- **Real-time video and audio streaming** via WebRTC
- **Secure session management** with JWT authentication
- **Docker and Kubernetes ready** for production deployment
- **Load balancing** with Nginx
- **Health monitoring** and logging
- **RESTful APIs** for session and user management

## 🛠️ Tech Stack

- **Backend**: Golang with Gin framework
- **Database**: MongoDB
- **Frontend**: React with TypeScript
- **SDK**: TypeScript/JavaScript
- **WebRTC**: Native browser WebRTC APIs
- **Deployment**: Docker, Kubernetes
- **Load Balancer**: Nginx
- **Authentication**: JWT tokens

## 🏗️ Architecture

Videoconf consists of four main components:

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

## 🚦 Getting Started

### Prerequisites

- **Docker & Docker Compose** (recommended)
- **Go 1.22+** (for local development)
- **Node.js 18+** (for local development)
- **MongoDB** (for local development)

### 🐳 Quick Start with Docker

1. **Clone the repository:**
   ```bash
   git clone https://github.com/r3tr056/go-videoconf.git
   cd go-videoconf
   ```

2. **Start the entire stack:**
   ```bash
   docker-compose up --build
   ```

3. **Access the application:**
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

2. **Deploy to Kubernetes:**
   ```bash
   make deploy-k8s
   ```

3. **Check deployment status:**
   ```bash
   kubectl get pods
   kubectl get services
   ```

## 📚 SDK Usage

### Installation

```bash
npm install videoconf-sdk
```

### Basic Usage

```javascript
import { VideoconfSDK, VideoContainer } from 'videoconf-sdk';

// Initialize SDK
const videoconf = new VideoconfSDK('your-api-key', 'project-id');

// Create a meeting
const sessionUrl = await videoconf.createSession('My Meeting', 'password123');

// Or join existing meeting
await videoconf.joinSession(sessionUrl, 'password123');

// Initialize camera and microphone
await videoconf.initializeCall();

// Handle events
videoconf.on('stream-added', (stream, peerId) => {
  console.log('New participant joined:', peerId);
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
import { VideoconfSDK, VideoGrid } from 'videoconf-sdk';

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

    setSdk(videoconf);
  }, []);

  return (
    <VideoGrid 
      streams={remoteStreams}
      localStream={localStream}
    />
  );
}
```

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
docker-compose logs -f
```

**Access individual services:**
```bash
# Signalling server logs
docker-compose logs signalling-server

# Users service logs  
docker-compose logs users-service

# Client logs
docker-compose logs videoconf-client
```

## 🚀 Production Deployment

### Environment Variables

**Signalling Server:**
- `PORT`: Server port (default: 8080)
- `DB_URL`: MongoDB host
- `DB_PORT`: MongoDB port  
- `DB_USERNAME`: MongoDB username
- `DB_PASSWORD`: MongoDB password

**Users Service:**
- `PORT`: Server port (default: 8081)
- `DB_HOST`: MongoDB host
- `DB_NAME`: Database name
- `JWT_SECRET`: JWT signing secret

### Security Considerations

- JWT authentication for users
- Session password hashing
- CORS configuration
- WebSocket origin validation
- Environment-based secrets

### Scaling

- Horizontal scaling via Kubernetes
- Load balancing with Nginx
- Stateless service design
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
