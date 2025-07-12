# VideoConf SDK Documentation

## 🎯 Overview

The VideoConf SDK is a comprehensive TypeScript library for building Google Meet level video conferencing applications. It provides enterprise-grade WebRTC functionality with advanced features like adaptive bitrate, noise suppression, virtual backgrounds, and real-time analytics.

## 🚀 Quick Start

### Installation

```bash
npm install @videoconf/sdk
```

### Basic Usage

```typescript
import { VideoConf } from '@videoconf/sdk';

// Initialize the SDK
const videoConf = new VideoConf({
  signallingServer: 'wss://your-signalling-server.com',
  stunServers: [{ urls: 'stun:stun.l.google.com:19302' }],
  autoJoinAudio: true,
  autoJoinVideo: true
});

// Create a meeting
const sessionUrl = await videoConf.createSession({
  title: 'Team Standup',
  password: 'secure123',
  maxParticipants: 50
});

// Join a meeting
await videoConf.joinSession(sessionUrl, 'secure123');

// Handle events
videoConf.on('participant-joined', (participant) => {
  console.log(`${participant.username} joined the meeting`);
});

videoConf.on('media-received', (participantId, stream) => {
  // Display remote video/audio
  const videoElement = document.getElementById('remote-video');
  videoElement.srcObject = stream;
});
```

## 📚 Core Features

### Session Management

#### Creating Sessions
```typescript
const sessionConfig = {
  title: 'Product Demo',
  password: 'demo123',
  isPrivate: true,
  maxParticipants: 100,
  recording: true
};

const sessionUrl = await videoConf.createSession(sessionConfig);
```

#### Joining Sessions
```typescript
// Join with password
await videoConf.joinSession(sessionUrl, 'password123');

// Join as guest (if allowed)
await videoConf.joinSession(sessionUrl);
```

#### Leaving Sessions
```typescript
await videoConf.leaveSession();
```

### Media Management

#### Initialize Camera and Microphone
```typescript
const stream = await videoConf.initializeMedia({
  video: {
    width: { ideal: 1920 },
    height: { ideal: 1080 },
    frameRate: { ideal: 30 }
  },
  audio: {
    echoCancellation: true,
    noiseSuppression: true,
    autoGainControl: true
  }
});

// Display local video
const localVideo = document.getElementById('local-video');
localVideo.srcObject = stream;
```

#### Media Controls
```typescript
// Toggle audio
const isAudioEnabled = videoConf.toggleAudio(); // Toggle current state
videoConf.toggleAudio(false); // Mute
videoConf.toggleAudio(true);  // Unmute

// Toggle video
const isVideoEnabled = videoConf.toggleVideo(); // Toggle current state
videoConf.toggleVideo(false); // Turn off camera
videoConf.toggleVideo(true);  // Turn on camera
```

For complete documentation, see the [full SDK documentation](./SDK_DOCUMENTATION.md).