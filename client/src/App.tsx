import React, { useState, useEffect, useRef } from 'react';
import './videoconf-sdk';

declare global {
  interface Window {
    VideoconfSDK: any;
  }
}

interface VideoStreamProps {
  stream: MediaStream | null;
  muted?: boolean;
  className?: string;
}

const VideoStream: React.FC<VideoStreamProps> = ({ stream, muted = false, className = '' }) => {
  const videoRef = useRef<HTMLVideoElement>(null);

  useEffect(() => {
    if (videoRef.current && stream) {
      videoRef.current.srcObject = stream;
    }
  }, [stream]);

  return (
    <video
      ref={videoRef}
      autoPlay
      playsInline
      muted={muted}
      className={`video-stream ${className}`}
      style={{
        width: '100%',
        height: '100%',
        objectFit: 'cover',
        backgroundColor: '#000',
        borderRadius: '8px'
      }}
    />
  );
};

const App: React.FC = () => {
  const [sdk, setSdk] = useState<any>(null);
  const [localStream, setLocalStream] = useState<MediaStream | null>(null);
  const [remoteStreams, setRemoteStreams] = useState<Map<string, MediaStream>>(new Map());
  const [sessionUrl, setSessionUrl] = useState('');
  const [sessionPassword, setSessionPassword] = useState('');
  const [sessionTitle, setSessionTitle] = useState('');
  const [isConnected, setIsConnected] = useState(false);
  const [isVideoEnabled, setIsVideoEnabled] = useState(true);
  const [isAudioEnabled, setIsAudioEnabled] = useState(true);
  const [currentView, setCurrentView] = useState<'home' | 'create' | 'join' | 'call'>('home');
  const [error, setError] = useState<string>('');

  useEffect(() => {
    const videoconfSDK = new window.VideoconfSDK('demo-api-key', 'demo-project');
    
    videoconfSDK.on('local-stream', (stream: MediaStream) => {
      setLocalStream(stream);
    });

    videoconfSDK.on('remote-stream', (stream: MediaStream, peerId: string) => {
      setRemoteStreams(prev => {
        const newStreams = new Map(prev);
        newStreams.set(peerId, stream);
        return newStreams;
      });
    });

    videoconfSDK.on('stream-removed', (peerId: string) => {
      setRemoteStreams(prev => {
        const newStreams = new Map(prev);
        newStreams.delete(peerId);
        return newStreams;
      });
    });

    setSdk(videoconfSDK);
  }, []);

  const createSession = async () => {
    try {
      setError('');
      const socketUrl = await sdk.createSession(sessionTitle, sessionPassword);
      setSessionUrl(socketUrl);
      await startCall();
      setIsConnected(true);
      setCurrentView('call');
    } catch (error) {
      setError(`Failed to create session: ${error}`);
    }
  };

  const joinSession = async () => {
    try {
      setError('');
      await sdk.joinSession(sessionUrl, sessionPassword);
      await startCall();
      setIsConnected(true);
      setCurrentView('call');
    } catch (error) {
      setError(`Failed to join session: ${error}`);
    }
  };

  const startCall = async () => {
    try {
      await sdk.initializeCall();
    } catch (error) {
      setError(`Failed to initialize call: ${error}`);
    }
  };

  const leaveCall = async () => {
    try {
      await sdk.leaveCall();
      setIsConnected(false);
      setLocalStream(null);
      setRemoteStreams(new Map());
      setCurrentView('home');
      setSessionUrl('');
      setSessionPassword('');
      setSessionTitle('');
    } catch (error) {
      setError(`Failed to leave call: ${error}`);
    }
  };

  const toggleVideo = () => {
    const newState = !isVideoEnabled;
    sdk.toggleVideo(newState);
    setIsVideoEnabled(newState);
  };

  const toggleAudio = () => {
    const newState = !isAudioEnabled;
    sdk.toggleAudio(newState);
    setIsAudioEnabled(newState);
  };

  const renderHome = () => (
    <div className="home-view">
      <div className="hero-section">
        <h1>VideoConf</h1>
        <p>Professional Video Conferencing Made Simple</p>
      </div>
      
      <div className="action-buttons">
        <button
          className="btn btn-primary"
          onClick={() => setCurrentView('create')}
        >
          Create Meeting
        </button>
        <button
          className="btn btn-secondary"
          onClick={() => setCurrentView('join')}
        >
          Join Meeting
        </button>
      </div>

      {error && <div className="error-message">{error}</div>}
    </div>
  );

  const renderCreateSession = () => (
    <div className="form-view">
      <h2>Create New Meeting</h2>
      
      <div className="form-group">
        <label>Meeting Title</label>
        <input
          type="text"
          value={sessionTitle}
          onChange={(e) => setSessionTitle(e.target.value)}
          placeholder="Enter meeting title"
          className="form-input"
        />
      </div>
      
      <div className="form-group">
        <label>Meeting Password</label>
        <input
          type="password"
          value={sessionPassword}
          onChange={(e) => setSessionPassword(e.target.value)}
          placeholder="Enter password"
          className="form-input"
        />
      </div>
      
      <div className="form-buttons">
        <button
          className="btn btn-primary"
          onClick={createSession}
          disabled={!sessionTitle || !sessionPassword}
        >
          Create Meeting
        </button>
        <button
          className="btn btn-secondary"
          onClick={() => setCurrentView('home')}
        >
          Cancel
        </button>
      </div>

      {sessionUrl && (
        <div className="session-info">
          <h3>Meeting Created!</h3>
          <p>Share this URL with participants:</p>
          <div className="session-url">{sessionUrl}</div>
        </div>
      )}

      {error && <div className="error-message">{error}</div>}
    </div>
  );

  const renderJoinSession = () => (
    <div className="form-view">
      <h2>Join Meeting</h2>
      
      <div className="form-group">
        <label>Meeting URL</label>
        <input
          type="text"
          value={sessionUrl}
          onChange={(e) => setSessionUrl(e.target.value)}
          placeholder="Enter meeting URL or ID"
          className="form-input"
        />
      </div>
      
      <div className="form-group">
        <label>Password</label>
        <input
          type="password"
          value={sessionPassword}
          onChange={(e) => setSessionPassword(e.target.value)}
          placeholder="Enter meeting password"
          className="form-input"
        />
      </div>
      
      <div className="form-buttons">
        <button
          className="btn btn-primary"
          onClick={joinSession}
          disabled={!sessionUrl || !sessionPassword}
        >
          Join Meeting
        </button>
        <button
          className="btn btn-secondary"
          onClick={() => setCurrentView('home')}
        >
          Cancel
        </button>
      </div>

      {error && <div className="error-message">{error}</div>}
    </div>
  );

  const renderCall = () => {
    const remoteStreamArray = Array.from(remoteStreams.entries());
    const totalStreams = remoteStreamArray.length + (localStream ? 1 : 0);
    
    const getGridCols = () => {
      if (totalStreams <= 1) return 1;
      if (totalStreams <= 4) return 2;
      return 3;
    };

    return (
      <div className="call-view">
        <div 
          className="video-grid"
          style={{
            display: 'grid',
            gridTemplateColumns: `repeat(${getGridCols()}, 1fr)`,
            gap: '10px',
            height: '70vh',
            padding: '20px'
          }}
        >
          {localStream && (
            <div className="video-container local">
              <VideoStream stream={localStream} muted={true} />
              <div className="video-label">You</div>
            </div>
          )}
          
          {remoteStreamArray.map(([peerId, stream]) => (
            <div key={peerId} className="video-container remote">
              <VideoStream stream={stream} />
              <div className="video-label">Participant {peerId.substring(0, 6)}</div>
            </div>
          ))}
        </div>
        
        <div className="call-controls">
          <button
            className={`control-btn ${isVideoEnabled ? 'active' : 'inactive'}`}
            onClick={toggleVideo}
          >
            {isVideoEnabled ? '📹' : '📹❌'}
          </button>
          
          <button
            className={`control-btn ${isAudioEnabled ? 'active' : 'inactive'}`}
            onClick={toggleAudio}
          >
            {isAudioEnabled ? '🎤' : '🎤❌'}
          </button>
          
          <button
            className="control-btn leave-btn"
            onClick={leaveCall}
          >
            📞❌
          </button>
        </div>

        {error && <div className="error-message">{error}</div>}
      </div>
    );
  };

  return (
    <div className="app">
      <style jsx>{`
        .app {
          font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
          min-height: 100vh;
          background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
          color: white;
        }

        .home-view, .form-view {
          display: flex;
          flex-direction: column;
          align-items: center;
          justify-content: center;
          min-height: 100vh;
          padding: 20px;
        }

        .hero-section {
          text-align: center;
          margin-bottom: 40px;
        }

        .hero-section h1 {
          font-size: 3rem;
          margin-bottom: 10px;
          font-weight: 700;
        }

        .hero-section p {
          font-size: 1.2rem;
          opacity: 0.9;
        }

        .action-buttons {
          display: flex;
          gap: 20px;
          margin-bottom: 20px;
        }

        .btn {
          padding: 12px 24px;
          font-size: 1rem;
          font-weight: 600;
          border: none;
          border-radius: 8px;
          cursor: pointer;
          transition: all 0.3s ease;
          min-width: 150px;
        }

        .btn-primary {
          background: #007bff;
          color: white;
        }

        .btn-primary:hover {
          background: #0056b3;
          transform: translateY(-2px);
        }

        .btn-secondary {
          background: rgba(255, 255, 255, 0.2);
          color: white;
          border: 1px solid rgba(255, 255, 255, 0.3);
        }

        .btn-secondary:hover {
          background: rgba(255, 255, 255, 0.3);
        }

        .btn:disabled {
          opacity: 0.5;
          cursor: not-allowed;
        }

        .form-view {
          max-width: 400px;
          width: 100%;
        }

        .form-view h2 {
          text-align: center;
          margin-bottom: 30px;
          font-size: 2rem;
        }

        .form-group {
          margin-bottom: 20px;
        }

        .form-group label {
          display: block;
          margin-bottom: 8px;
          font-weight: 600;
        }

        .form-input {
          width: 100%;
          padding: 12px;
          font-size: 1rem;
          border: none;
          border-radius: 8px;
          background: rgba(255, 255, 255, 0.9);
          color: #333;
        }

        .form-input::placeholder {
          color: #666;
        }

        .form-buttons {
          display: flex;
          gap: 10px;
          margin-top: 30px;
        }

        .session-info {
          margin-top: 20px;
          padding: 20px;
          background: rgba(255, 255, 255, 0.1);
          border-radius: 8px;
          text-align: center;
        }

        .session-url {
          font-family: monospace;
          background: rgba(0, 0, 0, 0.3);
          padding: 10px;
          border-radius: 4px;
          word-break: break-all;
          margin-top: 10px;
        }

        .call-view {
          height: 100vh;
          display: flex;
          flex-direction: column;
        }

        .video-container {
          position: relative;
          background: #000;
          border-radius: 8px;
          overflow: hidden;
        }

        .video-container.local {
          border: 2px solid #007bff;
        }

        .video-label {
          position: absolute;
          bottom: 10px;
          left: 10px;
          background: rgba(0, 0, 0, 0.7);
          color: white;
          padding: 4px 8px;
          border-radius: 4px;
          font-size: 0.8rem;
        }

        .call-controls {
          display: flex;
          justify-content: center;
          gap: 20px;
          padding: 20px;
          background: rgba(0, 0, 0, 0.3);
        }

        .control-btn {
          width: 60px;
          height: 60px;
          border: none;
          border-radius: 50%;
          font-size: 1.5rem;
          cursor: pointer;
          transition: all 0.3s ease;
        }

        .control-btn.active {
          background: #28a745;
          color: white;
        }

        .control-btn.inactive {
          background: #dc3545;
          color: white;
        }

        .control-btn.leave-btn {
          background: #dc3545;
          color: white;
        }

        .control-btn:hover {
          transform: scale(1.1);
        }

        .error-message {
          background: rgba(220, 53, 69, 0.9);
          color: white;
          padding: 12px;
          border-radius: 8px;
          margin-top: 20px;
          text-align: center;
        }

        @media (max-width: 768px) {
          .hero-section h1 {
            font-size: 2rem;
          }
          
          .action-buttons {
            flex-direction: column;
            align-items: center;
          }
          
          .form-buttons {
            flex-direction: column;
          }
          
          .video-grid {
            grid-template-columns: 1fr !important;
          }
        }
      `}</style>

      {currentView === 'home' && renderHome()}
      {currentView === 'create' && renderCreateSession()}
      {currentView === 'join' && renderJoinSession()}
      {currentView === 'call' && renderCall()}
    </div>
  );
};

export default App;