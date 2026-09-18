/**
 * Unit tests for @videoconf/sdk. Browser APIs (fetch, WebSocket,
 * RTCPeerConnection, mediaDevices) are mocked; signalling-server HTTP and
 * WebSocket contracts are asserted, not the real network.
 */
import { VideoconfSDK, WebRTCMessage } from './index';

class MockWebSocket {
  static instances: MockWebSocket[] = [];
  static readonly OPEN = 1;

  url: string;
  readyState = 0;
  sent: string[] = [];
  onopen: (() => void) | null = null;
  onmessage: ((event: { data: string }) => void) | null = null;
  onerror: ((event: unknown) => void) | null = null;
  onclose: (() => void) | null = null;

  constructor(url: string) {
    this.url = url;
    MockWebSocket.instances.push(this);
  }

  send(data: string): void {
    this.sent.push(data);
  }

  close(): void {
    this.readyState = 3;
  }

  open(): void {
    this.readyState = MockWebSocket.OPEN;
    this.onopen?.();
  }

  receive(message: WebRTCMessage): void {
    this.onmessage?.({ data: JSON.stringify(message) });
  }
}

class MockRTCPeerConnection {
  static instances: MockRTCPeerConnection[] = [];

  ontrack: ((event: { streams: MediaStream[] }) => void) | null = null;
  onicecandidate: ((event: { candidate: unknown }) => void) | null = null;
  localDescription: unknown = null;
  remoteDescription: unknown = null;
  candidates: unknown[] = [];
  tracks: unknown[] = [];
  closed = false;
  config: unknown;

  constructor(config?: unknown) {
    this.config = config;
    MockRTCPeerConnection.instances.push(this);
  }

  addTrack(track: unknown): void {
    this.tracks.push(track);
  }

  getSenders(): { track: unknown }[] {
    return this.tracks.map((track) => ({ track }));
  }

  async createOffer(): Promise<unknown> {
    return { type: 'offer', sdp: 'mock-offer' };
  }

  async createAnswer(): Promise<unknown> {
    return { type: 'answer', sdp: 'mock-answer' };
  }

  async setLocalDescription(desc: unknown): Promise<void> {
    this.localDescription = desc;
  }

  async setRemoteDescription(desc: unknown): Promise<void> {
    this.remoteDescription = desc;
  }

  async addIceCandidate(candidate: unknown): Promise<void> {
    this.candidates.push(candidate);
  }

  close(): void {
    this.closed = true;
  }
}

class MockSessionDescription {
  constructor(public init: unknown) {}
}

class MockIceCandidate {
  constructor(public init: unknown) {}
}

function mockMediaStream(): MediaStream {
  const track = { enabled: true, stop: jest.fn() };
  return {
    getTracks: () => [track],
    getVideoTracks: () => [track],
    getAudioTracks: () => [track],
  } as unknown as MediaStream;
}

function mockFetchOnce(body: unknown, ok = true): void {
  (global.fetch as jest.Mock).mockResolvedValueOnce({
    ok,
    json: async () => body,
  });
}

beforeEach(() => {
  MockWebSocket.instances = [];
  MockRTCPeerConnection.instances = [];
  (global as unknown as { WebSocket: unknown }).WebSocket = MockWebSocket;
  (global as unknown as { RTCPeerConnection: unknown }).RTCPeerConnection =
    MockRTCPeerConnection;
  (global as unknown as { RTCSessionDescription: unknown }).RTCSessionDescription =
    MockSessionDescription;
  (global as unknown as { RTCIceCandidate: unknown }).RTCIceCandidate =
    MockIceCandidate;
  global.fetch = jest.fn();
  Object.defineProperty(global.navigator, 'mediaDevices', {
    value: { getUserMedia: jest.fn() },
    configurable: true,
  });
  jest.spyOn(console, 'error').mockImplementation(() => undefined);
});

afterEach(() => {
  jest.restoreAllMocks();
});

describe('identity and events', () => {
  it('generates a distinct user id per instance', () => {
    const a = new VideoconfSDK('k', 'p');
    const b = new VideoconfSDK('k', 'p');
    expect(a.getUserId()).toBeTruthy();
    expect(a.getUserId()).not.toBe(b.getUserId());
  });

  it('delivers emitted events to subscribed listeners', () => {
    const sdk = new VideoconfSDK('k', 'p');
    const seen: unknown[][] = [];
    sdk.on('remote-stream', (...args: unknown[]) => seen.push(args));
    // Reach the private emitter through a synthetic remote track event.
    (sdk as unknown as { emit: (...a: unknown[]) => void }).emit(
      'remote-stream',
      'stream',
      'peer-1',
    );
    expect(seen).toEqual([['stream', 'peer-1']]);
  });

  it('isolates listener errors from other listeners', () => {
    const sdk = new VideoconfSDK('k', 'p');
    const calls: string[] = [];
    sdk.on('local-stream', () => {
      calls.push('bad');
      throw new Error('boom');
    });
    sdk.on('local-stream', () => calls.push('good'));
    (sdk as unknown as { emit: (...a: unknown[]) => void }).emit('local-stream');
    expect(calls).toEqual(['bad', 'good']);
    expect(console.error).toHaveBeenCalled();
  });
});

describe('session REST contract', () => {
  it('createSession posts host/title/password and returns the socket id', async () => {
    const sdk = new VideoconfSDK('k', 'p');
    mockFetchOnce({ socket: 'sock-123' });
    const socket = await sdk.createSession('Standup', 'secret');
    expect(socket).toBe('sock-123');
    expect(global.fetch).toHaveBeenCalledWith(
      '/session',
      expect.objectContaining({ method: 'POST' }),
    );
    const body = JSON.parse(
      (global.fetch as jest.Mock).mock.calls[0][1].body as string,
    );
    expect(body).toEqual({
      host: sdk.getUserId(),
      title: 'Standup',
      password: 'secret',
    });
  });

  it('createSession throws when the server rejects', async () => {
    const sdk = new VideoconfSDK('k', 'p');
    mockFetchOnce({}, false);
    await expect(sdk.createSession('t', 'p')).rejects.toThrow(
      'Failed to create session',
    );
  });

  it('joinSession posts the password then opens the socket websocket', async () => {
    const sdk = new VideoconfSDK('k', 'p', {
      signallingServerUrl: 'ws://signal:8080',
    });
    mockFetchOnce({ socket: 'sock-9' });
    const joined = sdk.joinSession('room-1', 'secret');
    // Flush the fetch promise so the WebSocket is constructed.
    await Promise.resolve();
    await Promise.resolve();
    expect(MockWebSocket.instances).toHaveLength(1);
    expect(MockWebSocket.instances[0].url).toBe('ws://signal:8080/ws/sock-9');
    MockWebSocket.instances[0].open();
    await joined;
    const connect = JSON.parse(MockWebSocket.instances[0].sent[0]);
    expect(connect.type).toBe('connect');
    expect(connect.userID).toBe(sdk.getUserId());
    expect(global.fetch).toHaveBeenCalledWith(
      '/connect/room-1',
      expect.objectContaining({ method: 'POST' }),
    );
  });

  it('joinSession throws when the server rejects', async () => {
    const sdk = new VideoconfSDK('k', 'p');
    mockFetchOnce({}, false);
    await expect(sdk.joinSession('room-1', 'bad')).rejects.toThrow(
      'Failed to join session',
    );
  });
});

describe('call lifecycle', () => {
  it('initializeCall acquires media and emits local-stream', async () => {
    const stream = mockMediaStream();
    (navigator.mediaDevices.getUserMedia as jest.Mock).mockResolvedValue(stream);
    const sdk = new VideoconfSDK('k', 'p');
    const seen: unknown[] = [];
    sdk.on('local-stream', (s: unknown) => seen.push(s));
    const result = await sdk.initializeCall();
    expect(result).toBe(stream);
    expect(sdk.getLocalStream()).toBe(stream);
    expect(seen).toEqual([stream]);
    expect(navigator.mediaDevices.getUserMedia).toHaveBeenCalledWith({
      video: true,
      audio: true,
    });
  });

  it('leaveCall notifies peers and releases everything', async () => {
    const sdk = new VideoconfSDK('k', 'p', {
      signallingServerUrl: 'ws://signal:8080',
    });
    mockFetchOnce({ socket: 's' });
    const joined = sdk.joinSession('r', 'pw');
    await Promise.resolve();
    await Promise.resolve();
    MockWebSocket.instances[0].open();
    await joined;
    await sdk.leaveCall();
    const last = JSON.parse(
      MockWebSocket.instances[0].sent[
        MockWebSocket.instances[0].sent.length - 1
      ],
    );
    expect(last.type).toBe('disconnect');
    expect(sdk.getLocalStream()).toBeNull();
    expect(sdk.getRemoteStreams().size).toBe(0);
  });

  it('toggle helpers are safe without a stream and effective with one', async () => {
    const sdk = new VideoconfSDK('k', 'p');
    expect(() => sdk.toggleVideo(false)).not.toThrow();
    expect(() => sdk.toggleAudio(false)).not.toThrow();
    const stream = mockMediaStream();
    (navigator.mediaDevices.getUserMedia as jest.Mock).mockResolvedValue(stream);
    await sdk.initializeCall();
    sdk.toggleVideo(false);
    expect(stream.getVideoTracks()[0].enabled).toBe(false);
    sdk.toggleAudio(false);
    expect(stream.getAudioTracks()[0].enabled).toBe(false);
  });

  it('getRemoteStreams returns a copy, not the live map', () => {
    const sdk = new VideoconfSDK('k', 'p');
    const copy = sdk.getRemoteStreams();
    copy.set('x', mockMediaStream());
    expect(sdk.getRemoteStreams().size).toBe(0);
  });
});

describe('signalling message handling', () => {
  async function connectedSdk(): Promise<{
    sdk: VideoconfSDK;
    socket: MockWebSocket;
  }> {
    const sdk = new VideoconfSDK('k', 'p', {
      signallingServerUrl: 'ws://signal:8080',
    });
    mockFetchOnce({ socket: 's' });
    const joined = sdk.joinSession('r', 'pw');
    await Promise.resolve();
    await Promise.resolve();
    const socket = MockWebSocket.instances[0];
    socket.open();
    await joined;
    return { sdk, socket };
  }

  async function flush(): Promise<void> {
    for (let i = 0; i < 10; i++) {
      await Promise.resolve();
    }
  }

  it('ignores its own messages', async () => {
    const { sdk, socket } = await connectedSdk();
    socket.receive({
      type: 'connect',
      userID: sdk.getUserId(),
      description: '',
      candidate: '',
    });
    await flush();
    expect(MockRTCPeerConnection.instances).toHaveLength(0);
  });

  it('answers a peer offer and applies the remote description', async () => {
    const { socket } = await connectedSdk();
    socket.receive({
      type: 'offer',
      userID: 'peer-1',
      description: JSON.stringify({ type: 'offer', sdp: 'o' }),
      candidate: '',
    });
    await flush();
    expect(MockRTCPeerConnection.instances).toHaveLength(1);
    const pc = MockRTCPeerConnection.instances[0];
    expect(pc.remoteDescription).toBeInstanceOf(MockSessionDescription);
    expect(pc.localDescription).toEqual({ type: 'answer', sdp: 'mock-answer' });
    const answer = JSON.parse(
      socket.sent[socket.sent.length - 1],
    ) as WebRTCMessage;
    expect(answer.type).toBe('answer');
    expect(answer.to).toBe('peer-1');
  });

  it('creates an offer when a peer connects after local media exists', async () => {
    const stream = mockMediaStream();
    (navigator.mediaDevices.getUserMedia as jest.Mock).mockResolvedValue(stream);
    const { sdk, socket } = await connectedSdk();
    await sdk.initializeCall();
    socket.receive({
      type: 'connect',
      userID: 'peer-2',
      description: '',
      candidate: '',
    });
    await flush();
    expect(MockRTCPeerConnection.instances).toHaveLength(1);
    const offer = JSON.parse(
      socket.sent[socket.sent.length - 1],
    ) as WebRTCMessage;
    expect(offer.type).toBe('offer');
    expect(offer.to).toBe('peer-2');
  });

  it('applies answers and ICE candidates to the matching peer', async () => {
    const { socket } = await connectedSdk();
    socket.receive({
      type: 'offer',
      userID: 'peer-3',
      description: JSON.stringify({ type: 'offer', sdp: 'o' }),
      candidate: '',
    });
    await flush();
    socket.receive({
      type: 'answer',
      userID: 'peer-3',
      description: JSON.stringify({ type: 'answer', sdp: 'a' }),
      candidate: '',
    });
    socket.receive({
      type: 'ice-candidate',
      userID: 'peer-3',
      description: '',
      candidate: JSON.stringify({ candidate: 'c' }),
    });
    await flush();
    const pc = MockRTCPeerConnection.instances[0];
    expect(pc.remoteDescription).toBeInstanceOf(MockSessionDescription);
    expect(pc.candidates).toHaveLength(1);
  });

  it('passes configured ICE servers (TURN) to peer connections', async () => {
    const turn = {
      urls: 'turn:turn.example.com:3478',
      username: 'u',
      credential: 'p',
    };
    const sdk = new VideoconfSDK('k', 'p', {
      signallingServerUrl: 'ws://signal:8080',
      iceServers: [turn],
    });
    mockFetchOnce({ socket: 's' });
    const joined = sdk.joinSession('r', 'pw');
    await Promise.resolve();
    await Promise.resolve();
    const socket = MockWebSocket.instances[0];
    socket.open();
    await joined;
    socket.receive({
      type: 'offer',
      userID: 'peer-4',
      description: JSON.stringify({ type: 'offer', sdp: 'o' }),
      candidate: '',
    });
    await flush();
    expect(MockRTCPeerConnection.instances).toHaveLength(1);
    expect(MockRTCPeerConnection.instances[0].config).toEqual({
      iceServers: [turn],
    });
  });

  it('disconnect removes the peer connection and emits stream-removed', async () => {
    const { sdk, socket } = await connectedSdk();
    socket.receive({
      type: 'offer',
      userID: 'peer-5',
      description: JSON.stringify({ type: 'offer', sdp: 'o' }),
      candidate: '',
    });
    await flush();
    const removed: unknown[] = [];
    sdk.on('stream-removed', (id: unknown) => removed.push(id));
    // Simulate a received remote stream so removal emits the event.
    (sdk as unknown as { remoteStreams: Map<string, unknown> }).remoteStreams.set(
      'peer-5',
      mockMediaStream(),
    );
    expect(sdk.getRemoteStreams().size).toBe(1);
    socket.receive({
      type: 'disconnect',
      userID: 'peer-5',
      description: '',
      candidate: '',
    });
    await flush();
    expect(MockRTCPeerConnection.instances[0].closed).toBe(true);
    expect(sdk.getRemoteStreams().size).toBe(0);
    expect(removed).toEqual(['peer-5']);
  });

  it('forwards remote tracks as remote-stream events', async () => {
    const { sdk, socket } = await connectedSdk();
    const seen: unknown[][] = [];
    sdk.on('remote-stream', (...args: unknown[]) => seen.push(args));
    socket.receive({
      type: 'offer',
      userID: 'peer-6',
      description: JSON.stringify({ type: 'offer', sdp: 'o' }),
      candidate: '',
    });
    await flush();
    const stream = mockMediaStream();
    MockRTCPeerConnection.instances[0].ontrack?.({ streams: [stream] });
    expect(seen).toEqual([[stream, 'peer-6']]);
    expect(sdk.getRemoteStreams().get('peer-6')).toBe(stream);
  });
});

describe('late media and renegotiation', () => {
  async function flush(): Promise<void> {
    for (let i = 0; i < 10; i++) {
      await Promise.resolve();
    }
  }

  async function joinedSdk(): Promise<{ sdk: VideoconfSDK; socket: MockWebSocket }> {
    const sdk = new VideoconfSDK('k', 'p', {
      signallingServerUrl: 'ws://signal:8080',
    });
    mockFetchOnce({ socket: 's' });
    const joined = sdk.joinSession('r', 'pw');
    await flush();
    const socket = MockWebSocket.instances[0];
    socket.open();
    await joined;
    return { sdk, socket };
  }

  it('tracks streamless peers without offering, then renegotiates on media', async () => {
    const { sdk, socket } = await joinedSdk();
    const sentBefore = socket.sent.length;
    socket.receive({
      type: 'connect',
      userID: 'peer-7',
      description: '',
      candidate: '',
    });
    await flush();
    // Peer tracked (PC created) but no offer while streamless.
    expect(MockRTCPeerConnection.instances).toHaveLength(1);
    expect(socket.sent.length).toBe(sentBefore);

    const stream = mockMediaStream();
    (navigator.mediaDevices.getUserMedia as jest.Mock).mockResolvedValue(stream);
    await sdk.initializeCall();
    await flush();

    // Renegotiation attached tracks and offered to the known peer.
    const pc = MockRTCPeerConnection.instances[0];
    expect(pc.tracks.length).toBeGreaterThan(0);
    const offer = JSON.parse(socket.sent[socket.sent.length - 1]) as WebRTCMessage;
    expect(offer.type).toBe('offer');
    expect(offer.to).toBe('peer-7');
  });

  it('reuses one peer connection per peer across repeated offers', async () => {
    const { socket } = await joinedSdk();
    const offer = {
      type: 'offer',
      userID: 'peer-8',
      description: JSON.stringify({ type: 'offer', sdp: 'o' }),
      candidate: '',
    } as WebRTCMessage;
    socket.receive(offer);
    await flush();
    socket.receive(offer);
    await flush();
    expect(MockRTCPeerConnection.instances).toHaveLength(1);
  });

  it('does not duplicate tracks when media already attached', async () => {
    const stream = mockMediaStream();
    (navigator.mediaDevices.getUserMedia as jest.Mock).mockResolvedValue(stream);
    const { sdk, socket } = await joinedSdk();
    await sdk.initializeCall();
    socket.receive({
      type: 'connect',
      userID: 'peer-9',
      description: '',
      candidate: '',
    });
    await flush();
    const pc = MockRTCPeerConnection.instances[0];
    const trackCount = pc.tracks.length;
    expect(trackCount).toBeGreaterThan(0);
    // A second initializeCall reuses senders instead of duplicating tracks.
    await sdk.initializeCall();
    await flush();
    expect(pc.tracks.length).toBe(trackCount);
  });
});

describe('signalling reconnect', () => {
  async function flush(): Promise<void> {
    for (let i = 0; i < 10; i++) {
      await Promise.resolve();
    }
  }

  const sleep = (ms: number): Promise<void> =>
    new Promise((resolve) => setTimeout(resolve, ms));

  it('reconnects with backoff and resends presence', async () => {
    const sdk = new VideoconfSDK('k', 'p', {
      signallingServerUrl: 'ws://signal:8080',
    });
    (sdk as unknown as { reconnectDelays: number[] }).reconnectDelays = [5, 5];
    mockFetchOnce({ socket: 's' });
    const joined = sdk.joinSession('r', 'pw');
    await flush();
    MockWebSocket.instances[0].open();
    await joined;

    const events: unknown[][] = [];
    sdk.on('reconnecting', (...args: unknown[]) => events.push(['reconnecting', ...args]));
    sdk.on('reconnected', () => events.push(['reconnected']));

    // Unexpected drop triggers a reconnected socket with fresh presence.
    MockWebSocket.instances[0].onclose?.();
    await sleep(30);
    expect(MockWebSocket.instances).toHaveLength(2);
    MockWebSocket.instances[1].open();
    await flush();
    const resent = JSON.parse(MockWebSocket.instances[1].sent[0]) as WebRTCMessage;
    expect(resent.type).toBe('connect');
    expect(resent.userID).toBe(sdk.getUserId());
    expect(events).toEqual([['reconnecting', 1], ['reconnected']]);
  });

  it('gives up after exhausting backoff and emits disconnected', async () => {
    const sdk = new VideoconfSDK('k', 'p', {
      signallingServerUrl: 'ws://signal:8080',
    });
    (sdk as unknown as { reconnectDelays: number[] }).reconnectDelays = [5, 5];
    mockFetchOnce({ socket: 's' });
    const joined = sdk.joinSession('r', 'pw');
    await flush();
    MockWebSocket.instances[0].open();
    await joined;

    const events: string[] = [];
    sdk.on('disconnected', () => events.push('disconnected'));

    MockWebSocket.instances[0].onclose?.();
    await sleep(20);
    expect(MockWebSocket.instances).toHaveLength(2);
    // Second socket also drops before opening.
    MockWebSocket.instances[1].onclose?.();
    await sleep(20);
    expect(MockWebSocket.instances).toHaveLength(3);
    MockWebSocket.instances[2].onclose?.();
    await sleep(30);
    expect(events).toEqual(['disconnected']);
    expect(sdk.getLocalStream()).toBeNull();
  });

  it('leaveCall suppresses any pending reconnect', async () => {
    const sdk = new VideoconfSDK('k', 'p', {
      signallingServerUrl: 'ws://signal:8080',
    });
    (sdk as unknown as { reconnectDelays: number[] }).reconnectDelays = [5];
    mockFetchOnce({ socket: 's' });
    const joined = sdk.joinSession('r', 'pw');
    await flush();
    MockWebSocket.instances[0].open();
    await joined;

    MockWebSocket.instances[0].onclose?.();
    await sdk.leaveCall();
    await sleep(30);
    expect(MockWebSocket.instances).toHaveLength(1);
  });
});

describe('signalling error guards', () => {
  async function flush(): Promise<void> {
    for (let i = 0; i < 10; i++) {
      await Promise.resolve();
    }
  }

  it('ignores malformed signalling payloads', async () => {
    const sdk = new VideoconfSDK('k', 'p', {
      signallingServerUrl: 'ws://signal:8080',
    });
    const warned: unknown[][] = [];
    jest.spyOn(console, 'warn').mockImplementation((...args: unknown[]) => warned.push(args));
    mockFetchOnce({ socket: 's' });
    const joined = sdk.joinSession('r', 'pw');
    await flush();
    const socket = MockWebSocket.instances[0];
    socket.open();
    await joined;
    socket.onmessage?.({ data: '{not-json' });
    await flush();
    expect(warned.length).toBeGreaterThan(0);
    expect(MockRTCPeerConnection.instances).toHaveLength(0);
  });

  it('contains handler failures instead of rejecting', async () => {
    const sdk = new VideoconfSDK('k', 'p', {
      signallingServerUrl: 'ws://signal:8080',
    });
    const warned: unknown[][] = [];
    jest.spyOn(console, 'warn').mockImplementation((...args: unknown[]) => warned.push(args));
    mockFetchOnce({ socket: 's' });
    const joined = sdk.joinSession('r', 'pw');
    await flush();
    const socket = MockWebSocket.instances[0];
    socket.open();
    await joined;
    // Offer with corrupt SDP JSON: JSON.parse throws inside the handler.
    socket.onmessage?.({
      data: JSON.stringify({
        type: 'offer',
        userID: 'peer-x',
        description: '{bad',
        candidate: '',
      }),
    });
    await flush();
    await flush();
    expect(warned.length).toBeGreaterThan(0);
  });
});
