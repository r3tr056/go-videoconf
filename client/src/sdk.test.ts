import { VideoconfSDK } from '@videoconf/sdk';

describe('vendored SDK wiring', () => {
  it('exposes a constructible VideoconfSDK class', () => {
    expect(typeof VideoconfSDK).toBe('function');
    const sdk = new VideoconfSDK('demo', 'demo', {
      signallingServerUrl: 'ws://localhost:8080',
    });
    expect(typeof sdk.getUserId()).toBe('string');
    expect(sdk.getLocalStream()).toBeNull();
    expect(sdk.getRemoteStreams().size).toBe(0);
  });
});
