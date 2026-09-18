/**
 * VideoConf WebRTC SDK — mesh P2P audio/video over the go-videoconf
 * signalling server. Browser only. MIT licensed.
 */

export interface VideoconfConfig {
  apiKey: string;
  projectId: string;
  signallingServerUrl?: string;
  /**
   * ICE servers for NAT traversal. Defaults to public STUN; production
   * deployments should supply the bundled TURN servers.
   */
  iceServers?: RTCIceServer[];
  /**
   * ICE transport policy. Set to 'relay' to force all media through TURN
   * (maximum privacy; also how the e2e suite proves the relay path).
   * Defaults to the browser standard ('all').
   */
  iceTransportPolicy?: RTCIceTransportPolicy;
}

export interface WebRTCMessage {
  type: 'offer' | 'answer' | 'ice-candidate' | 'connect' | 'disconnect';
  userID: string;
  description?: string;
  candidate?: string;
  to?: string;
}

// Event payloads vary per event ('local-stream' passes a MediaStream,
// 'remote-stream' passes stream + peer id, 'stream-removed' passes peer id),
// so listeners intentionally receive untyped arguments.
// eslint-disable-next-line @typescript-eslint/no-explicit-any
export type VideoconfEventHandler = (...args: any[]) => void;

export class VideoconfSDK {
  private config: VideoconfConfig;
  private socket: WebSocket | null = null;
  private localStream: MediaStream | null = null;
  private peerConnections: Map<string, RTCPeerConnection> = new Map();
  private remoteStreams: Map<string, MediaStream> = new Map();
  private currentUserId: string;
  private eventListeners: Map<string, VideoconfEventHandler[]> = new Map();

  private socketUrl: string | null = null;
  private connectedOnce = false;
  private intentionalClose = false;
  private reconnectAttempts = 0;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  /** Backoff schedule for signalling reconnects; overridable in tests. */
  private reconnectDelays = [1000, 2000, 4000, 8000, 16000];

  private readonly defaultIceServers: RTCIceServer[] = [
    { urls: 'stun:stun.l.google.com:19302' },
  ];

  constructor(apiKey: string, projectId: string, options?: Partial<VideoconfConfig>) {
    this.config = {
      apiKey,
      projectId,
      signallingServerUrl: options?.signallingServerUrl || defaultSignallingUrl(),
      iceServers: options?.iceServers,
      iceTransportPolicy: options?.iceTransportPolicy,
    };
    this.currentUserId = generateUserId();
  }

  on(event: string, listener: VideoconfEventHandler): void {
    if (!this.eventListeners.has(event)) {
      this.eventListeners.set(event, []);
    }
    this.eventListeners.get(event)!.push(listener);
  }

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  private emit(event: string, ...args: any[]): void {
    const listeners = this.eventListeners.get(event);
    if (listeners) {
      listeners.forEach((listener) => {
        try {
          listener(...args);
        } catch (error) {
          console.error(`Error in event listener for ${event}:`, error);
        }
      });
    }
  }

  private get iceServers(): RTCIceServer[] {
    return this.config.iceServers ?? this.defaultIceServers;
  }

  getUserId(): string {
    return this.currentUserId;
  }

  async createSession(title: string, password: string): Promise<string> {
    const response = await fetch('/session', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ host: this.currentUserId, title, password }),
    });

    if (!response.ok) throw new Error('Failed to create session');
    const data = await response.json();
    return data.socket;
  }

  async joinSession(sessionUrl: string, password: string): Promise<void> {
    const response = await fetch(`/connect/${sessionUrl}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ password }),
    });

    if (!response.ok) throw new Error('Failed to join session');
    const data = await response.json();

    this.socketUrl = data.socket;
    this.intentionalClose = false;
    this.reconnectAttempts = 0;
    await this.connectToSignallingServer(data.socket);
  }

  async initializeCall(): Promise<MediaStream> {
    this.localStream = await navigator.mediaDevices.getUserMedia({
      video: true,
      audio: true,
    });
    this.emit('local-stream', this.localStream);
    // Late media: peer connections may predate the local stream (peers that
    // joined first, or presence replayed while streamless). Attach tracks to
    // every known peer and renegotiate so media flows both ways.
    for (const peerId of this.peerConnections.keys()) {
      try {
        await this.renegotiate(peerId);
      } catch (error) {
        console.warn(`Renegotiation with ${peerId} failed:`, error);
      }
    }
    return this.localStream;
  }

  private attachSocketHandlers(resolve: () => void, reject: (reason?: unknown) => void): void {
    if (!this.socket) return;

    this.socket.onopen = () => {
      this.connectedOnce = true;
      this.sendMessage({
        type: 'connect',
        userID: this.currentUserId,
        description: '',
        candidate: '',
      });
      resolve();
    };

    this.socket.onmessage = (event) => {
      let message: WebRTCMessage;
      try {
        message = JSON.parse(event.data);
      } catch (error) {
        console.warn('Ignoring malformed signalling message:', error);
        return;
      }
      // Signalling races (e.g. simultaneous offers from both sides) must log,
      // never surface as unhandled rejections.
      this.handleSignallingMessage(message).catch((error) => {
        console.warn('Signalling handler failed:', error);
      });
    };

    this.socket.onerror = reject;
    this.socket.onclose = () => {
      this.socket = null;
      if (this.intentionalClose || !this.connectedOnce || !this.socketUrl) {
        return;
      }
      this.scheduleReconnect();
    };
  }

  private async connectToSignallingServer(socketUrl: string): Promise<void> {
    return new Promise((resolve, reject) => {
      const wsUrl = `${this.config.signallingServerUrl}/ws/${socketUrl}`;
      this.socket = new WebSocket(wsUrl);
      this.attachSocketHandlers(resolve, reject);
    });
  }

  private scheduleReconnect(): void {
    if (this.intentionalClose || this.reconnectTimer) return;
    if (this.reconnectAttempts >= this.reconnectDelays.length) {
      this.emit('disconnected');
      this.cleanup();
      return;
    }
    const delay = this.reconnectDelays[this.reconnectAttempts];
    this.reconnectAttempts += 1;
    this.emit('reconnecting', this.reconnectAttempts);
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null;
      if (this.intentionalClose || !this.socketUrl) return;
      this.connectToSignallingServer(this.socketUrl).then(
        () => {
          this.reconnectAttempts = 0;
          this.emit('reconnected');
        },
        () => this.scheduleReconnect(),
      );
    }, delay);
  }

  private async handleSignallingMessage(message: WebRTCMessage): Promise<void> {
    if (message.userID === this.currentUserId) return;

    switch (message.type) {
      case 'connect':
        // Always track the peer so late local media can reach them; only
        // offer immediately when there is media to send.
        await this.createPeerConnection(message.userID);
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
    const existing = this.peerConnections.get(peerId);
    if (existing) return existing;

    const peerConnection = new RTCPeerConnection({
      iceServers: this.iceServers,
      iceTransportPolicy: this.config.iceTransportPolicy,
    });

    if (this.localStream) {
      this.localStream.getTracks().forEach((track) => {
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
          description: '',
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
      candidate: '',
    });
  }

  private async renegotiate(peerId: string): Promise<void> {
    const peerConnection = this.peerConnections.get(peerId);
    if (!peerConnection || !this.localStream) return;
    const stream = this.localStream;
    stream.getTracks().forEach((track) => {
      const senders =
        typeof peerConnection.getSenders === 'function' ? peerConnection.getSenders() : [];
      if (!senders.some((sender) => sender.track === track)) {
        peerConnection.addTrack(track, stream);
      }
    });
    const offer = await peerConnection.createOffer();
    await peerConnection.setLocalDescription(offer);
    this.sendMessage({
      type: 'offer',
      userID: this.currentUserId,
      description: JSON.stringify(offer),
      to: peerId,
      candidate: '',
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
      candidate: '',
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
    this.intentionalClose = true;
    this.socketUrl = null;
    if (this.socket) {
      this.sendMessage({
        type: 'disconnect',
        userID: this.currentUserId,
        description: '',
        candidate: '',
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
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    this.peerConnections.forEach((pc) => pc.close());
    this.peerConnections.clear();
    this.remoteStreams.clear();

    if (this.localStream) {
      this.localStream.getTracks().forEach((track) => track.stop());
      this.localStream = null;
    }

    if (this.socket) {
      this.socket.close();
      this.socket = null;
    }
  }
}

export default VideoconfSDK;

function defaultSignallingUrl(): string {
  if (typeof window !== 'undefined' && window.location?.origin) {
    return window.location.origin.replace(/^http/, 'ws');
  }
  return 'ws://localhost:8080';
}

function generateUserId(): string {
  return Math.random().toString(36).substring(2, 15);
}

declare global {
  interface Window {
    VideoconfSDK: typeof VideoconfSDK;
  }
}

if (typeof window !== 'undefined') {
  window.VideoconfSDK = VideoconfSDK;
}
