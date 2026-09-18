import React, { useState, useEffect, useRef } from 'react';
import * as Dialog from '@radix-ui/react-dialog';
import * as DropdownMenu from '@radix-ui/react-dropdown-menu';
import * as Separator from '@radix-ui/react-separator';
import * as Switch from '@radix-ui/react-switch';
import * as Tooltip from '@radix-ui/react-tooltip';
import * as VisuallyHidden from '@radix-ui/react-visually-hidden';
import {
  CheckIcon,
  Cross2Icon,
  DotsHorizontalIcon,
  ExitIcon,
  EyeClosedIcon,
  EyeOpenIcon,
  GitHubLogoIcon,
  InfoCircledIcon,
  Link2Icon,
  PersonIcon,
} from '@radix-ui/react-icons';
import { VideoconfSDK } from '@videoconf/sdk';
import { Badge, Button, Callout, CopyButton, Field, IconButton, Tip, copyText } from './components/ui';
import {
  LogoMark,
  MicOffIcon,
  MicOnIcon,
  PhoneDownIcon,
  VideoOffIcon,
  VideoOnIcon,
} from './components/icons';

interface VideoStreamProps {
  stream: MediaStream | null;
  muted?: boolean;
}

const VideoStream: React.FC<VideoStreamProps> = ({ stream, muted = false }) => {
  const videoRef = useRef<HTMLVideoElement>(null);

  useEffect(() => {
    if (videoRef.current && stream) {
      videoRef.current.srcObject = stream;
    }
  }, [stream]);

  return <video ref={videoRef} autoPlay playsInline muted={muted} />;
};

type BusyState = 'idle' | 'creating' | 'joining';

const App: React.FC = () => {
  const [sdk, setSdk] = useState<VideoconfSDK | null>(null);
  const [localStream, setLocalStream] = useState<MediaStream | null>(null);
  const [remoteStreams, setRemoteStreams] = useState<Map<string, MediaStream>>(new Map());
  const [createTitle, setCreateTitle] = useState('');
  const [createPassword, setCreatePassword] = useState('');
  const [joinCode, setJoinCode] = useState('');
  const [joinPassword, setJoinPassword] = useState('');
  const [inviteCode, setInviteCode] = useState('');
  const [meetingTitle, setMeetingTitle] = useState('');
  const [inCall, setInCall] = useState(false);
  const [isVideoEnabled, setIsVideoEnabled] = useState(true);
  const [isAudioEnabled, setIsAudioEnabled] = useState(true);
  const [cameraOn, setCameraOn] = useState(true);
  const [micOn, setMicOn] = useState(true);
  const [createOpen, setCreateOpen] = useState(false);
  const [showCreatePassword, setShowCreatePassword] = useState(false);
  const [showJoinPassword, setShowJoinPassword] = useState(false);
  const [busy, setBusy] = useState<BusyState>('idle');
  const [menuCopied, setMenuCopied] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    let cancelled = false;

    const init = async () => {
      const iceServers: RTCIceServer[] = [{ urls: 'stun:stun.l.google.com:19302' }];
      let forceRelay = false;
      try {
        // Runtime configuration (baked at image build or mounted per
        // deployment) so prebuilt images work without rebuilding.
        const res = await fetch('/turn-config.json', { cache: 'no-store' });
        if (res.ok) {
          const turn = await res.json();
          if (turn?.url) {
            iceServers.push({ urls: turn.url, username: turn.username, credential: turn.credential });
          }
          forceRelay = turn?.forceRelay === true;
        }
      } catch {
        // STUN-only fallback when no TURN config is served.
      }
      if (cancelled) return;
      const videoconfSDK = new VideoconfSDK('demo-api-key', 'demo-project', {
        iceServers,
        ...(forceRelay ? { iceTransportPolicy: 'relay' as RTCIceTransportPolicy } : {}),
      });

      videoconfSDK.on('local-stream', (stream: MediaStream) => {
        setLocalStream(stream);
      });

      videoconfSDK.on('remote-stream', (stream: MediaStream, peerId: string) => {
        setRemoteStreams((prev) => {
          const next = new Map(prev);
          next.set(peerId, stream);
          return next;
        });
      });

      videoconfSDK.on('stream-removed', (peerId: string) => {
        setRemoteStreams((prev) => {
          const next = new Map(prev);
          next.delete(peerId);
          return next;
        });
      });

      setSdk(videoconfSDK);
    };

    init();
    return () => {
      cancelled = true;
    };
  }, []);

  const participantCount = remoteStreams.size + (localStream ? 1 : 0);

  const startCall = async () => {
    if (!sdk) return;
    await sdk.initializeCall();
    // Apply the join options picked on the home screen.
    if (!cameraOn) sdk.toggleVideo(false);
    if (!micOn) sdk.toggleAudio(false);
    setIsVideoEnabled(cameraOn);
    setIsAudioEnabled(micOn);
  };

  const createSession = async (event: React.FormEvent) => {
    event.preventDefault();
    if (!sdk || busy !== 'idle') return;
    try {
      setError('');
      setBusy('creating');
      const socketUrl = await sdk.createSession(createTitle, createPassword);
      setInviteCode(socketUrl);
      setMeetingTitle(createTitle);
      // Creating only reserves the room; the host must join it like any
      // other peer, otherwise nobody can signal with them.
      await sdk.joinSession(socketUrl, createPassword);
      await startCall();
      setCreateOpen(false);
      setInCall(true);
    } catch (err) {
      setError(`Could not create the meeting: ${err}`);
    } finally {
      setBusy('idle');
    }
  };

  const joinSession = async (event: React.FormEvent) => {
    event.preventDefault();
    if (!sdk || busy !== 'idle') return;
    try {
      setError('');
      setBusy('joining');
      await sdk.joinSession(joinCode.trim(), joinPassword);
      setInviteCode(joinCode.trim());
      setMeetingTitle('');
      await startCall();
      setInCall(true);
    } catch (err) {
      setError(`Could not join the meeting: ${err}`);
    } finally {
      setBusy('idle');
    }
  };

  const leaveCall = async () => {
    if (!sdk) return;
    try {
      await sdk.leaveCall();
    } catch (err) {
      setError(`Could not leave cleanly: ${err}`);
    } finally {
      setInCall(false);
      setLocalStream(null);
      setRemoteStreams(new Map());
      setInviteCode('');
      setMeetingTitle('');
      setJoinCode('');
      setJoinPassword('');
      setCreateTitle('');
      setCreatePassword('');
      setIsVideoEnabled(true);
      setIsAudioEnabled(true);
    }
  };

  const toggleVideo = () => {
    if (!sdk) return;
    const next = !isVideoEnabled;
    sdk.toggleVideo(next);
    setIsVideoEnabled(next);
  };

  const toggleAudio = () => {
    if (!sdk) return;
    const next = !isAudioEnabled;
    sdk.toggleAudio(next);
    setIsAudioEnabled(next);
  };

  const copyInviteFromMenu = async () => {
    if (!inviteCode) return;
    if (await copyText(inviteCode)) {
      setMenuCopied(true);
      setTimeout(() => setMenuCopied(false), 1600);
    }
  };

  const statusText = !sdk
    ? 'Loading'
    : inCall
      ? `In call, ${participantCount} participant${participantCount === 1 ? '' : 's'}`
      : 'Ready';

  const renderHeader = () => (
    <header className="vc-header">
      <div className="vc-brand">
        <span className="vc-brand-mark">
          <LogoMark />
        </span>
        VideoConf
      </div>
      <div className="vc-header-actions">
        <Badge live={inCall}>{statusText}</Badge>
        <DropdownMenu.Root>
          <DropdownMenu.Trigger asChild>
            <IconButton label="Meeting options" bordered>
              <DotsHorizontalIcon width={16} height={16} />
            </IconButton>
          </DropdownMenu.Trigger>
          <DropdownMenu.Portal>
            <DropdownMenu.Content className="vc-menu" sideOffset={6} align="end">
              <DropdownMenu.Label className="vc-menu-label">Meeting</DropdownMenu.Label>
              <DropdownMenu.Item
                className="vc-menu-item"
                disabled={!inviteCode}
                onSelect={() => void copyInviteFromMenu()}
              >
                {menuCopied ? <CheckIcon /> : <Link2Icon />}
                {menuCopied ? 'Code copied' : 'Copy meeting code'}
              </DropdownMenu.Item>
              {inCall && (
                <DropdownMenu.Item className="vc-menu-item" onSelect={() => void leaveCall()}>
                  <ExitIcon />
                  Leave call
                </DropdownMenu.Item>
              )}
              <Separator.Root className="vc-menu-separator" />
              <DropdownMenu.Item className="vc-menu-item" asChild>
                <a
                  href="https://github.com/r3tr056/go-videoconf"
                  target="_blank"
                  rel="noreferrer"
                  style={{ textDecoration: 'none', color: 'inherit' }}
                >
                  <GitHubLogoIcon />
                  Star on GitHub
                </a>
              </DropdownMenu.Item>
            </DropdownMenu.Content>
          </DropdownMenu.Portal>
        </DropdownMenu.Root>
      </div>
    </header>
  );

  const renderJoinOptions = () => (
    <div style={{ margin: '6px 0 16px' }}>
      <div className="vc-switch-row">
        <label className="vc-switch-label" htmlFor="opt-camera">
          <VideoOnIcon size={16} />
          Camera on when joining
        </label>
        <Switch.Root
          id="opt-camera"
          className="vc-switch"
          checked={cameraOn}
          onCheckedChange={setCameraOn}
        >
          <Switch.Thumb className="vc-switch-thumb" />
        </Switch.Root>
      </div>
      <div className="vc-switch-row">
        <label className="vc-switch-label" htmlFor="opt-mic">
          <MicOnIcon size={16} />
          Microphone on when joining
        </label>
        <Switch.Root id="opt-mic" className="vc-switch" checked={micOn} onCheckedChange={setMicOn}>
          <Switch.Thumb className="vc-switch-thumb" />
        </Switch.Root>
      </div>
    </div>
  );

  const renderHome = () => (
    <main className="vc-main">
      <div className="vc-home">
        <div className="vc-home-copy">
          <h1>
            Video calls,
            <br />
            <span className="vc-accent-word">self-hosted.</span>
          </h1>
          <p className="vc-lede">
            Peer-to-peer meetings on your own infrastructure. No accounts, no tracking, no per-seat
            pricing — just share a code and talk.
          </p>
          <ul className="vc-points">
            <li>
              <CheckIcon width={16} height={16} aria-hidden="true" />
              End-to-end encrypted media over WebRTC
            </li>
            <li>
              <CheckIcon width={16} height={16} aria-hidden="true" />
              One-command deploy with Docker or Helm
            </li>
            <li>
              <CheckIcon width={16} height={16} aria-hidden="true" />
              Free forever under the MIT license
            </li>
          </ul>
          <p className="vc-home-foot">
            <InfoCircledIcon
              width={13}
              height={13}
              aria-hidden="true"
              style={{ verticalAlign: '-2px', marginRight: 6 }}
            />
            Runs fully in your browser; the server only introduces peers.
          </p>
        </div>

        <div className="vc-panel">
          <h2 className="vc-panel-title">Join a meeting</h2>
          <p className="vc-panel-sub">Enter the code shared by the host.</p>
          <form onSubmit={(e) => void joinSession(e)}>
            <Field
              label="Meeting code"
              id="join-code"
              mono
              value={joinCode}
              onChange={(e) => setJoinCode(e.target.value)}
              placeholder="e.g. a3f9-…"
              autoComplete="off"
              spellCheck={false}
            />
            <Field
              label="Password"
              id="join-password"
              type={showJoinPassword ? 'text' : 'password'}
              value={joinPassword}
              onChange={(e) => setJoinPassword(e.target.value)}
              placeholder="Meeting password"
              autoComplete="current-password"
              trailing={
                <IconButton
                  label={showJoinPassword ? 'Hide password' : 'Show password'}
                  onClick={() => setShowJoinPassword((v) => !v)}
                >
                  {showJoinPassword ? (
                    <EyeClosedIcon width={16} height={16} />
                  ) : (
                    <EyeOpenIcon width={16} height={16} />
                  )}
                </IconButton>
              }
            />
            {renderJoinOptions()}
            <Button
              type="submit"
              block
              variant="teal"
              loading={busy === 'joining'}
              disabled={!sdk || !joinCode.trim() || !joinPassword || busy !== 'idle'}
            >
              Join meeting
            </Button>
          </form>

          <div className="vc-divider-row">or</div>

          <Dialog.Root open={createOpen} onOpenChange={setCreateOpen}>
            <Dialog.Trigger asChild>
              <Button variant="ghost" block data-testid="new-meeting-btn">
                New meeting
              </Button>
            </Dialog.Trigger>
            <Dialog.Portal>
              <Dialog.Overlay className="vc-overlay" />
              <Dialog.Content
                className="vc-dialog"
                aria-describedby="create-desc"
                onOpenAutoFocus={(e) => e.preventDefault()}
              >
                <Dialog.Title className="vc-dialog-title">New meeting</Dialog.Title>
                <Dialog.Description className="vc-dialog-desc" id="create-desc">
                  Pick a title and password, then share the code with your guests.
                </Dialog.Description>
                <form onSubmit={(e) => void createSession(e)}>
                  <Field
                    label="Meeting title"
                    id="create-title"
                    value={createTitle}
                    onChange={(e) => setCreateTitle(e.target.value)}
                    placeholder="Weekly sync"
                    autoComplete="off"
                  />
                  <Field
                    label="Meeting password"
                    id="create-password"
                    type={showCreatePassword ? 'text' : 'password'}
                    value={createPassword}
                    onChange={(e) => setCreatePassword(e.target.value)}
                    placeholder="Required to join"
                    autoComplete="new-password"
                    trailing={
                      <IconButton
                        label={showCreatePassword ? 'Hide password' : 'Show password'}
                        onClick={() => setShowCreatePassword((v) => !v)}
                      >
                        {showCreatePassword ? (
                          <EyeClosedIcon width={16} height={16} />
                        ) : (
                          <EyeOpenIcon width={16} height={16} />
                        )}
                      </IconButton>
                    }
                  />
                  <div className="vc-dialog-actions">
                    <Dialog.Close asChild>
                      <Button type="button" variant="ghost">
                        Cancel
                      </Button>
                    </Dialog.Close>
                    <Button
                      type="submit"
                      loading={busy === 'creating'}
                      disabled={!sdk || !createTitle.trim() || !createPassword || busy !== 'idle'}
                    >
                      Create meeting
                    </Button>
                  </div>
                </form>
                <Dialog.Close asChild>
                  <IconButton label="Close" className="vc-dialog-close">
                    <Cross2Icon width={16} height={16} />
                  </IconButton>
                </Dialog.Close>
              </Dialog.Content>
            </Dialog.Portal>
          </Dialog.Root>

          {error && !inCall && <Callout>{error}</Callout>}
        </div>
      </div>
    </main>
  );

  const renderCall = () => {
    const remoteEntries = Array.from(remoteStreams.entries());
    const totalTiles = remoteEntries.length + (localStream ? 1 : 0);
    const cols = totalTiles <= 1 ? 1 : totalTiles <= 4 ? 2 : 3;

    return (
      <div className="vc-call">
        <div className="vc-call-top">
          <Badge live>Live</Badge>
          {meetingTitle && <span className="vc-call-title">{meetingTitle}</span>}
          {inviteCode && (
            <span className="vc-code-chip" title={inviteCode}>
              {inviteCode.length > 18 ? `${inviteCode.slice(0, 18)}…` : inviteCode}
              <CopyButton value={inviteCode} label="Copy meeting code" size={14} />
            </span>
          )}
          <span className="vc-call-top-spacer" />
          <Badge>
            <PersonIcon width={13} height={13} aria-hidden="true" />
            {participantCount} in call
          </Badge>
        </div>

        <div className="vc-grid" style={{ gridTemplateColumns: `repeat(${cols}, 1fr)` }}>
          {localStream && (
            <div className="vc-tile local">
              <VideoStream stream={localStream} muted />
              <span className="vc-tile-label">
                You
                {!isAudioEnabled && (
                  <span className="muted">
                    <MicOffIcon size={13} />
                  </span>
                )}
              </span>
            </div>
          )}
          {remoteEntries.map(([peerId, stream]) => (
            <div key={peerId} className="vc-tile">
              <VideoStream stream={stream} />
              <span className="vc-tile-label">Guest {peerId.substring(0, 6)}</span>
            </div>
          ))}
          {remoteEntries.length === 0 && (
            <div className="vc-tile-empty">
              <div>You are the only one here.</div>
              <div>Share the meeting code so others can join.</div>
              {inviteCode && (
                <div className="vc-full-code-row">
                  <span className="vc-full-code" data-testid="full-code">
                    {inviteCode}
                  </span>
                  <CopyButton value={inviteCode} label="Copy meeting code" />
                </div>
              )}
            </div>
          )}
        </div>

        {error && (
          <div className="vc-call-error">
            <Callout>{error}</Callout>
          </div>
        )}

        <div className="vc-dock-wrap">
          <div className="vc-dock" role="toolbar" aria-label="Call controls">
            <Tip content={isAudioEnabled ? 'Mute microphone' : 'Unmute microphone'}>
              <button
                type="button"
                className={`vc-control${isAudioEnabled ? '' : ' off'}`}
                onClick={toggleAudio}
                aria-label={isAudioEnabled ? 'Mute microphone' : 'Unmute microphone'}
                aria-pressed={!isAudioEnabled}
              >
                {isAudioEnabled ? <MicOnIcon size={20} /> : <MicOffIcon size={20} />}
              </button>
            </Tip>
            <Tip content={isVideoEnabled ? 'Turn camera off' : 'Turn camera on'}>
              <button
                type="button"
                className={`vc-control${isVideoEnabled ? '' : ' off'}`}
                onClick={toggleVideo}
                aria-label={isVideoEnabled ? 'Turn camera off' : 'Turn camera on'}
                aria-pressed={!isVideoEnabled}
              >
                {isVideoEnabled ? <VideoOnIcon size={20} /> : <VideoOffIcon size={20} />}
              </button>
            </Tip>
            <Tip content="Copy meeting code">
              <button
                type="button"
                className="vc-control"
                onClick={() => void copyInviteFromMenu()}
                aria-label="Copy meeting code"
                disabled={!inviteCode}
              >
                {menuCopied ? <CheckIcon width={19} height={19} /> : <Link2Icon width={19} height={19} />}
              </button>
            </Tip>
            <Separator.Root orientation="vertical" decorative className="vc-dock-separator" />
            <Tip content="Leave call">
              <button
                type="button"
                className="vc-control leave"
                onClick={() => void leaveCall()}
                aria-label="Leave call"
              >
                <PhoneDownIcon size={20} />
              </button>
            </Tip>
          </div>
        </div>
      </div>
    );
  };

  return (
    <Tooltip.Provider delayDuration={250}>
      <div className={`vc-app${inCall ? ' in-call' : ''}`}>
        <VisuallyHidden.Root>
          <div role="status">{statusText}</div>
        </VisuallyHidden.Root>
        {renderHeader()}
        {inCall ? renderCall() : renderHome()}
      </div>
    </Tooltip.Provider>
  );
};

export default App;
