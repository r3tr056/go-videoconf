// Simple inline VideoConf SDK for the client (avoiding complex build dependencies)

interface VideoconfConfig {
  apiKey: string;
  projectId: string;
  signallingServerUrl?: string;
}

interface WebRTCMessage {
  type: 'offer' | 'answer' | 'ice-candidate' | 'connect' | 'disconnect';
  userID: string;
  description?: string;
  candidate?: string;
  to?: string;
}

declare global {
  interface Window {
    VideoconfSDK: typeof VideoconfSDK;
  }
}

class VideoconfSDK {
  private config: VideoconfConfig;
  private socket: WebSocket | null = null;
  private localStream: MediaStream | null = null;
  private peerConnections: Map<string, RTCPeerConnection> = new Map();
  private remoteStreams: Map<string, MediaStream> = new Map();
  private currentUserId: string;
  private eventListeners: Map<string, Function[]> = new Map();

  private readonly defaultSTUNServers: RTCIceServer[] = [
    { urls: 'stun:stun.l.google.com:19302' }
  ];

  constructor(apiKey: string, projectId: string, options?: Partial<VideoconfConfig>) {
    this.config = {
      apiKey,
      projectId,
      signallingServerUrl: options?.signallingServerUrl || window.location.origin.replace(/^http/, 'ws')
    };
    this.currentUserId = this.generateUserId();
  }

  on(event: string, listener: Function): void {
    if (!this.eventListeners.has(event)) {
      this.eventListeners.set(event, []);
    }
    this.eventListeners.get(event)!.push(listener);
  }

  private emit(event: string, ...args: any[]): void {
    const listeners = this.eventListeners.get(event);
    if (listeners) {
      listeners.forEach(listener => {
        try {
          listener(...args);
        } catch (error) {
          console.error(`Error in event listener for ${event}:`, error);
        }
      });
    }
  }

  async createSession(title: string, password: string): Promise<string> {
    const response = await fetch('/session', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ host: this.currentUserId, title, password })
    });

    if (!response.ok) throw new Error('Failed to create session');
    const data = await response.json();
    return data.socket;
  }

  async joinSession(sessionUrl: string, password: string): Promise<void> {
    const response = await fetch(`/connect/${sessionUrl}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ password })
    });

    if (!response.ok) throw new Error('Failed to join session');
    const data = await response.json();
    
    await this.connectToSignallingServer(data.socket);
  }

  async initializeCall(): Promise<MediaStream> {
    this.localStream = await navigator.mediaDevices.getUserMedia({
      video: true,
      audio: true
    });
    this.emit('local-stream', this.localStream);
    return this.localStream;
  }

  private async connectToSignallingServer(socketUrl: string): Promise<void> {
    return new Promise((resolve, reject) => {
      const wsUrl = `${this.config.signallingServerUrl}/ws/${socketUrl}`;
      this.socket = new WebSocket(wsUrl);

      this.socket.onopen = () => {
        this.sendMessage({
          type: 'connect',
          userID: this.currentUserId,
          description: '',
          candidate: ''
        });
        resolve();
      };

      this.socket.onmessage = (event) => {
        const message: WebRTCMessage = JSON.parse(event.data);
        this.handleSignallingMessage(message);
      };

      this.socket.onerror = reject;
      this.socket.onclose = () => this.cleanup();
    });
  }

  private async handleSignallingMessage(message: WebRTCMessage): Promise<void> {
    if (message.userID === this.currentUserId) return;

    switch (message.type) {
      case 'connect':
        if (this.localStream) {
          await this.createOffer(message.userID);
        }
        break;
      case 'offer':
        await this.handleOffer(message.userID, message.description!);
        break;
      case 'answer':
        await this.handleAnswer(message.userID, message.description!);
        break;
      case 'ice-candidate':
        await this.handleIceCandidate(message.userID, message.candidate!);
        break;
      case 'disconnect':
        this.removePeerConnection(message.userID);
        break;
    }
  }

  private async createPeerConnection(peerId: string): Promise<RTCPeerConnection> {
    const peerConnection = new RTCPeerConnection({
      iceServers: this.defaultSTUNServers
    });

    if (this.localStream) {
      this.localStream.getTracks().forEach(track => {
        peerConnection.addTrack(track, this.localStream!);
      });
    }

    peerConnection.ontrack = (event) => {
      const [remoteStream] = event.streams;
      this.remoteStreams.set(peerId, remoteStream);
      this.emit('remote-stream', remoteStream, peerId);
    };

    peerConnection.onicecandidate = (event) => {
      if (event.candidate) {
        this.sendMessage({
          type: 'ice-candidate',
          userID: this.currentUserId,
          candidate: JSON.stringify(event.candidate),
          to: peerId,
          description: ''
        });
      }
    };

    this.peerConnections.set(peerId, peerConnection);
    return peerConnection;
  }

  private async createOffer(peerId: string): Promise<void> {
    const peerConnection = await this.createPeerConnection(peerId);
    const offer = await peerConnection.createOffer();
    await peerConnection.setLocalDescription(offer);

    this.sendMessage({
      type: 'offer',
      userID: this.currentUserId,
      description: JSON.stringify(offer),
      to: peerId,
      candidate: ''
    });
  }

  private async handleOffer(peerId: string, offerDescription: string): Promise<void> {
    const peerConnection = await this.createPeerConnection(peerId);
    const offer = JSON.parse(offerDescription);
    await peerConnection.setRemoteDescription(new RTCSessionDescription(offer));

    const answer = await peerConnection.createAnswer();
    await peerConnection.setLocalDescription(answer);

    this.sendMessage({
      type: 'answer',
      userID: this.currentUserId,
      description: JSON.stringify(answer),
      to: peerId,
      candidate: ''
    });
  }

  private async handleAnswer(peerId: string, answerDescription: string): Promise<void> {
    const peerConnection = this.peerConnections.get(peerId);
    if (peerConnection) {
      const answer = JSON.parse(answerDescription);
      await peerConnection.setRemoteDescription(new RTCSessionDescription(answer));
    }
  }

  private async handleIceCandidate(peerId: string, candidateJson: string): Promise<void> {
    const peerConnection = this.peerConnections.get(peerId);
    if (peerConnection) {
      const candidate = JSON.parse(candidateJson);
      await peerConnection.addIceCandidate(new RTCIceCandidate(candidate));
    }
  }

  private sendMessage(message: WebRTCMessage): void {
    if (this.socket && this.socket.readyState === WebSocket.OPEN) {
      this.socket.send(JSON.stringify(message));
    }
  }

  private removePeerConnection(peerId: string): void {
    const peerConnection = this.peerConnections.get(peerId);
    if (peerConnection) {
      peerConnection.close();
      this.peerConnections.delete(peerId);
    }
    
    const remoteStream = this.remoteStreams.get(peerId);
    if (remoteStream) {
      this.remoteStreams.delete(peerId);
      this.emit('stream-removed', peerId);
    }
  }

  async leaveCall(): Promise<void> {
    if (this.socket) {
      this.sendMessage({
        type: 'disconnect',
        userID: this.currentUserId,
        description: '',
        candidate: ''
      });
    }
    this.cleanup();
  }

  toggleVideo(enabled: boolean): void {
    if (this.localStream) {
      const videoTrack = this.localStream.getVideoTracks()[0];
      if (videoTrack) videoTrack.enabled = enabled;
    }
  }

  toggleAudio(enabled: boolean): void {
    if (this.localStream) {
      const audioTrack = this.localStream.getAudioTracks()[0];
      if (audioTrack) audioTrack.enabled = enabled;
    }
  }

  getLocalStream(): MediaStream | null {
    return this.localStream;
  }

  getRemoteStreams(): Map<string, MediaStream> {
    return new Map(this.remoteStreams);
  }

  private cleanup(): void {
    this.peerConnections.forEach(pc => pc.close());
    this.peerConnections.clear();
    this.remoteStreams.clear();

    if (this.localStream) {
      this.localStream.getTracks().forEach(track => track.stop());
      this.localStream = null;
    }

    if (this.socket) {
      this.socket.close();
      this.socket = null;
    }
  }

  private generateUserId(): string {
    return Math.random().toString(36).substring(2, 15);
  }
}

// Make SDK available globally
window.VideoconfSDK = VideoconfSDK;