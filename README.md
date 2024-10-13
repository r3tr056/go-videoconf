# Videoconf - AaaS :-)

Videoconf is a robust Video Conference API as a Service platform (AaaS :-)) that provides seamless integration of video conferencing capabilities into JavaScript (React, Angular, etc.) and Node.js applications.

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

The entire stack is deployed on Kubernetes for optimal performance and scalability.

## 🚦 Getting Started

### Prerequisites

- Node.js (v14+)
- Go (v1.16+)
- MongoDB
- Kubernetes cluster

### Installation

1. Clone the repository:
   ```
   git clone https://github.com/yourusername/videoconf.git
   ```

2. Install dependencies:
   ```
   cd videoconf
   npm install
   ```

3. Set up environment variables (see `.env.example`)

4. Start the development server:
   ```
   npm run dev
   ```

## 📚 Documentation

For detailed documentation on how to use the Videoconf API and SDK, please visit our [documentation site](https://docs.videoconf.example.com).

## 🔧 Usage

Here's a quick example of how to use the Videoconf SDK in a React application:

```javascript
import { VideoconfSDK } from 'videoconf-sdk';

const videoconf = new VideoconfSDK('YOUR_API_KEY');

function VideoCall() {
  useEffect(() => {
    videoconf.initializeCall('room-id');
  }, []);

  return <div id="video-container"></div>;
}
```

## 🤝 Contributing

We welcome contributions to Videoconf! Please see our [Contributing Guide](CONTRIBUTING.md) for more details.

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 📞 Support

If you encounter any issues or have questions, please file an issue on GitHub or contact our support team at [Support](support@ankurdebnath.me).

## 🙏 Tech Used

- [Gin Web Framework](https://github.com/gin-gonic/gin)
- [MongoDB Go Driver](https://github.com/mongodb/mongo-go-driver)
- [WebRTC](https://webrtc.org/)
