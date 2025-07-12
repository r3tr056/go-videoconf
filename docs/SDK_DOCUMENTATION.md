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

### Screen Sharing

```typescript
// Start screen sharing
const screenStream = await videoConf.startScreenShare({
  includeAudio: true,
  width: 1920,
  height: 1080,
  frameRate: 30
});

// Display screen share
const screenVideo = document.getElementById('screen-video');
screenVideo.srcObject = screenStream;

// Stop screen sharing
videoConf.stopScreenShare();

// Handle screen share events
videoConf.on('screen-share-started', (participantId, stream) => {
  console.log(`${participantId} started screen sharing`);
});

videoConf.on('screen-share-ended', (participantId) => {
  console.log(`${participantId} stopped screen sharing`);
});
```

### Recording

```typescript
// Start recording
await videoConf.startRecording({
  video: true,
  audio: true,
  format: 'webm',
  quality: 'high'
});

// Stop recording and get download URL
const recordingUrl = await videoConf.stopRecording();

// Download the recording
const link = document.createElement('a');
link.href = recordingUrl;
link.download = 'meeting-recording.webm';
link.click();
```

### Chat Functionality

```typescript
// Send a chat message
videoConf.sendChatMessage('Hello everyone!');

// Handle incoming chat messages
videoConf.on('chat-message', (message) => {
  console.log(`${message.from}: ${message.message}`);
  
  // Display in chat UI
  const chatContainer = document.getElementById('chat');
  const messageElement = document.createElement('div');
  messageElement.innerHTML = `
    <strong>${message.from}</strong>
    <span>${message.timestamp.toLocaleTimeString()}</span>
    <p>${message.message}</p>
  `;
  chatContainer.appendChild(messageElement);
});
```

## 🎛️ Advanced Features

### Adaptive Bitrate Control

```typescript
import { AdvancedFeatures } from '@videoconf/sdk/advanced';

const advanced = new AdvancedFeatures(videoConf);

// Enable adaptive bitrate based on network conditions
advanced.enableAdaptiveBitrate(true);

// Monitor network quality
videoConf.on('network-quality', (qualities) => {
  qualities.forEach(quality => {
    console.log(`${quality.participantId}: ${quality.quality} (${quality.latency}ms, ${quality.packetLoss}% loss)`);
  });
});
```

### Noise Suppression

```typescript
// Apply noise suppression to audio stream
const localStream = videoConf.getLocalStream();
if (localStream) {
  const enhancedStream = await advanced.enableNoiseSuppression(localStream);
}
```

### Virtual Backgrounds

```typescript
// Blur background
const stream = videoConf.getLocalStream();
const blurredStream = await advanced.enableVirtualBackground(stream, 'blur');

// Use custom background image
const customBgStream = await advanced.enableVirtualBackground(
  stream, 
  'image', 
  'https://example.com/background.jpg'
);

// Remove background effects
const normalStream = await advanced.enableVirtualBackground(stream, 'none');
```

## 🎨 React Components

### Basic Video Conference Component

```typescript
import React, { useEffect, useRef, useState } from 'react';
import { VideoConf, Participant } from '@videoconf/sdk';

interface VideoConferenceProps {
  sessionUrl: string;
  password?: string;
  onLeave: () => void;
}

export const VideoConference: React.FC<VideoConferenceProps> = ({
  sessionUrl,
  password,
  onLeave
}) => {
  const [videoConf] = useState(() => new VideoConf({
    signallingServer: process.env.REACT_APP_SIGNALLING_SERVER!
  }));
  
  const [participants, setParticipants] = useState<Participant[]>([]);
  const [isAudioEnabled, setIsAudioEnabled] = useState(true);
  const [isVideoEnabled, setIsVideoEnabled] = useState(true);
  const [isScreenSharing, setIsScreenSharing] = useState(false);
  
  const localVideoRef = useRef<HTMLVideoElement>(null);

  useEffect(() => {
    const initializeConference = async () => {
      try {
        // Set up event listeners
        videoConf.on('participant-joined', (participant) => {
          setParticipants(prev => [...prev, participant]);
        });
        
        videoConf.on('participant-left', (participantId) => {
          setParticipants(prev => prev.filter(p => p.id !== participantId));
        });
        
        videoConf.on('media-received', (participantId, stream) => {
          const videoElement = document.getElementById(`video-${participantId}`) as HTMLVideoElement;
          if (videoElement) {
            videoElement.srcObject = stream;
          }
        });

        // Join the session
        await videoConf.joinSession(sessionUrl, password);
        
        // Initialize local media
        const localStream = await videoConf.initializeMedia({
          audio: true,
          video: true
        });
        
        if (localVideoRef.current) {
          localVideoRef.current.srcObject = localStream;
        }
        
      } catch (error) {
        console.error('Failed to initialize conference:', error);
      }
    };

    initializeConference();

    return () => {
      videoConf.leaveSession();
    };
  }, [videoConf, sessionUrl, password]);

  const handleToggleAudio = () => {
    const enabled = videoConf.toggleAudio();
    setIsAudioEnabled(enabled);
  };

  const handleToggleVideo = () => {
    const enabled = videoConf.toggleVideo();
    setIsVideoEnabled(enabled);
  };

  const handleToggleScreenShare = async () => {
    if (isScreenSharing) {
      videoConf.stopScreenShare();
      setIsScreenSharing(false);
    } else {
      try {
        await videoConf.startScreenShare();
        setIsScreenSharing(true);
      } catch (error) {
        console.error('Screen share failed:', error);
      }
    }
  };

  const handleLeave = async () => {
    await videoConf.leaveSession();
    onLeave();
  };

  return (
    <div className="video-conference">
      {/* Local video */}
      <div className="local-video-container">
        <video
          ref={localVideoRef}
          autoPlay
          muted
          playsInline
          className="local-video"
        />
      </div>

      {/* Remote participants */}
      <div className="participants-grid">
        {participants.map(participant => (
          <div key={participant.id} className="participant">
            <video
              id={`video-${participant.id}`}
              autoPlay
              playsInline
              className="participant-video"
            />
            <div className="participant-info">
              <span>{participant.username}</span>
              {!participant.isAudioEnabled && <span>🔇</span>}
              {!participant.isVideoEnabled && <span>📹</span>}
              {participant.isScreenSharing && <span>🖥️</span>}
            </div>
          </div>
        ))}
      </div>

      {/* Controls */}
      <div className="controls">
        <button
          onClick={handleToggleAudio}
          className={`control-btn ${isAudioEnabled ? 'enabled' : 'disabled'}`}
        >
          {isAudioEnabled ? '🎤' : '🔇'}
        </button>
        
        <button
          onClick={handleToggleVideo}
          className={`control-btn ${isVideoEnabled ? 'enabled' : 'disabled'}`}
        >
          {isVideoEnabled ? '📹' : '📷'}
        </button>
        
        <button
          onClick={handleToggleScreenShare}
          className={`control-btn ${isScreenSharing ? 'enabled' : 'disabled'}`}
        >
          🖥️
        </button>
        
        <button onClick={handleLeave} className="control-btn leave-btn">
          📞
        </button>
      </div>
    </div>
  );
};
```

## 🔧 Configuration

### Environment Variables

```bash
# Signalling Server
REACT_APP_SIGNALLING_SERVER=wss://your-signalling-server.com

# STUN/TURN Servers
REACT_APP_STUN_SERVERS=stun:stun.l.google.com:19302
REACT_APP_TURN_SERVERS=turn:your-turn-server.com:3478

# API Endpoints
REACT_APP_USERS_API=https://your-users-api.com
```

### Advanced Configuration

```typescript
const config: VideoConfConfig = {
  signallingServer: 'wss://signalling.example.com',
  
  // ICE Servers for NAT traversal
  stunServers: [
    { urls: 'stun:stun.l.google.com:19302' },
    { urls: 'stun:stun1.l.google.com:19302' }
  ],
  
  turnServers: [
    {
      urls: 'turn:turn.example.com:3478',
      username: 'username',
      credential: 'password'
    }
  ],
  
  // Auto-join settings
  autoJoinAudio: true,
  autoJoinVideo: true,
  
  // Session limits
  maxParticipants: 100,
};

const videoConf = new VideoConf(config);
```

## 📊 Performance Optimization

### Best Practices

1. **Adaptive Bitrate**: Enable adaptive bitrate to automatically adjust video quality based on network conditions
2. **Audio Processing**: Use noise suppression and echo cancellation for better audio quality
3. **Video Constraints**: Set appropriate video constraints based on use case
4. **Connection Pooling**: Reuse WebSocket connections when possible
5. **Lazy Loading**: Load features only when needed

### Performance Monitoring

```typescript
// Monitor performance metrics
videoConf.on('network-quality', (qualities) => {
  const averageLatency = qualities.reduce((sum, q) => sum + q.latency, 0) / qualities.length;
  
  if (averageLatency > 200) {
    console.warn('High latency detected:', averageLatency);
    // Consider reducing video quality
  }
});

// Monitor bandwidth usage
const analytics = advanced.generateMeetingAnalytics();
console.log('Bandwidth usage:', analytics.technicalMetrics.averageBandwidth);
```

---

Built with ❤️ by the VideoConf Team