# Videoconf - AaaS :-)

Videoconf is a robust Video Conference API as a Service platform (AaaS :-)) that
provides seamless integration of video conferencing capabilities into JavaScript
(React, Angular, etc.) and Node.js applications.

## 🚀 Features

- Easy-to-use APIs and SDKs for video conferencing integration
- Support for multiple JavaScript frameworks (React, Angular, etc.)
- Node.js compatibility
- Scalable microservice architecture
- Real-time video and audio streaming
- Secure and efficient call routing
- Kubernetes-based deployment for high availability and scalability

## 🛠️ Tech Stack

- **Database**: MongoDB
- **Backend**: Golang with Gin framework
- **Client**: JavaScript SDK
- **Deployment**: Kubernetes (K8s)

## 🏗️ Architecture

Videoconf consists of three main components:

1. **Golang Microservice**: Handles video call sessions and routes requests
2. **Client SDK**: Provides easy integration for web applications
3. **MongoDB Database**: Stores user and session data

The entire stack is deployed on Kubernetes for optimal performance and
scalability.

## 🚦 Getting Started

### Prerequisites

- Node.js (v14+)
- Go (v1.16+)
- MongoDB
- Kubernetes cluster

### ☸️ Getting it Up - k8s

A working kubernetes cluster is needed for this project

1. Clone the repository:
   ```
   git clone https://github.com/r3tro56/go-videoconf.git
   ```

2. Setup the Kubernetes Cluster:
   ```
   cd videoconf/.deployment
   bash ./01-deploy-db.sh
   bash ./02-configure-mongodb-repset.sh
   bash ./03-deploy-rest.sh
   ```

## 📚 Documentation

For detailed documentation on how to use the Videoconf API and SDK, please visit
our [documentation site](https://docs.videoconf.example.com).

## 🔧 SDK Usage

Here's a quick example of how to use the Videoconf SDK (ES6) in a React
application:

```javascript
import { VideoconfSDK, VideoContainer } from "videoconf-sdk";

const videoconf = new VideoconfSDK("YOUR_API_KEY", "PROJECT_ID");

function VideoCall() {
   useEffect(() => {
      videoconf.initializeCall("room-id");
   }, []);

   return <VideoContainer videoconf={videoConf} />;
}
```

## 🤝 Contributing

We welcome contributions to Videoconf! Please see our
[Contributing Guide](CONTRIBUTING.md) for more details.

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file
for details.

## 📞 Support

If you encounter any issues or have questions, please file an issue on GitHub or
contact our support team at [Support](support@ankurdebnath.me).

## 🙏 Tech Used

- [Gin Web Framework](https://github.com/gin-gonic/gin)
- [MongoDB Go Driver](https://github.com/mongodb/mongo-go-driver)
- [WebRTC](https://webrtc.org/)
- [NodeJS](https://nodejs.org)
- [ReactJS](https://reactjs.dev)
