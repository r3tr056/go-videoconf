package media

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/pion/webrtc/v4"
	"github.com/r3tr056/go-videoconf/media-service/internal/monitoring"
	"github.com/r3tr056/go-videoconf/media-service/proto/mediaserver"
	"github.com/sirupsen/logrus"
)

type timeNowFunc func() time.Time

var defaultTimeNow = time.Now

type MediaPipeline interface {
	StartSession(ctx context.Context, sessionInfo *SessionInfo) error
	StopSession(ctx context.Context, sessionInfo *SessionInfo) error
	AddTrack(ctx context.Context, sessionInfo *SessionInfo, track *webrtc.TrackLocalStaticRTP) error
	RemoveTrack(ctx context.Context, sessionInfo *SessionInfo, track *webrtc.TrackLocalStaticRTP) error
	GetActiveSessionCount() int
	HandleWebRTCSignal(ctx context.Context, signal *mediaserver.WebRTCSignal) error
	ReceiveWebRTCSignalStream(ctx context.Context, sessionInfo *SessionInfo, signalChan chan *mediaserver.WebRTCSignal) error
}

type SessionInfo struct {
	SessionID     string
	RoomID        string
	ParticipantID string
	MediaServerID string
}

type MediaSession struct {
	SessionInfo      SessionInfo
	PeerConnection   *webrtc.PeerConnection
	RemoteTrackChan  chan *webrtc.TrackRemote
	localTracks      []*webrtc.TrackLocalStaticRTP
	mu               sync.Mutex
	createdAt        timeNowFunc
	signalStreamChan chan *mediaserver.WebRTCSignal
}

// SFUMediaPipeline - Selective Forwarding Unit implementation (basic placeholder for Module 1)
type SFUMediaPipeline struct {
	sessions      map[string]*MediaSession // SessionID -> MediaSession
	sessionsMutex sync.Mutex
	mediaServerID string
	mediaEngine   *webrtc.MediaEngine
	api           *webrtc.API
}

func NewSFUMediaPipeline(mediaServerID string) *SFUMediaPipeline {
	m := &webrtc.MediaEngine{}
	if err := m.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType:     webrtc.MimeTypeVP8,
			ClockRate:    90000,
			Channels:     0,
			SDPFmtpLine:  "",
			RTCPFeedback: nil,
		},
		PayloadType: 96,
	}, webrtc.RTPCodecTypeVideo); err != nil {
		logrus.Errorf("Failed to register VP8 codec: %v", err)
	}

	if err := m.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType:     webrtc.MimeTypeOpus,
			ClockRate:    48000,
			Channels:     2,
			SDPFmtpLine:  "",
			RTCPFeedback: nil,
		},
		PayloadType: 111,
	}, webrtc.RTPCodecTypeAudio); err != nil {
		logrus.Errorf("Failed to register Opus codec: %v", err)
	}

	api := webrtc.NewAPI(webrtc.WithMediaEngine(m))

	return &SFUMediaPipeline{
		sessions:      make(map[string]*MediaSession),
		mediaServerID: mediaServerID,
		mediaEngine:   m,
		api:           api,
	}
}

func (sfu *SFUMediaPipeline) StartSession(ctx context.Context, sessionInfo *SessionInfo) error {
	sfu.sessionsMutex.Lock()
	defer sfu.sessionsMutex.Unlock()

	if _, exists := sfu.sessions[sessionInfo.SessionID]; exists {
		return fmt.Errorf("session already exists: %s", sessionInfo.SessionID)
	}

	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}

	// create new peer connection
	peerConnection, err := sfu.api.NewPeerConnection(config)
	if err != nil {
		return fmt.Errorf("failed to create new PeerConnection: %w", err)
	}

	// create data channel
	_, err = peerConnection.CreateDataChannel("data", nil)
	if err != nil {
		logrus.Warnf("Failed to create data channel: %v", err)
	}

	// channels for remote tracks and signals to gRPCStream
	remoteTrackChan := make(chan *webrtc.TrackRemote, 32)
	signalStreamChan := make(chan *mediaserver.WebRTCSignal, 32)

	// Handle ICE Candidates
	peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil {
			return
		}

		candidateString := candidate.String()
		signalStreamChan <- &mediaserver.WebRTCSignal{
			SessionId:     sessionInfo.SessionID,
			ParticipantId: sessionInfo.ParticipantID,
			Type:          "candidate",
			Candidate:     string(candidateString),
		}
	})

	peerConnection.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		logrus.Debugf("ICE Connection State fro Session %s, Participant %s changed %s", sessionInfo.SessionID, sessionInfo.ParticipantID, state.String())
		if state == webrtc.ICEConnectionStateFailed || state == webrtc.ICEConnectionStateDisconnected {
			logrus.Warnf("ICE Connection Failed or Disconnected for Session %s, Participant %s. Stopping session", sessionInfo.SessionID, sessionInfo.ParticipantID)
			go func() {
				if err := sfu.StopSession(context.Background(), sessionInfo); err != nil {
					logrus.Errorf("Error stopping session after ICE failure: %v", err)
				}
			}()
		}
	})

	// Handle new remote tracks
	peerConnection.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		logrus.Infof("Remote track received: SessionID=%s, Participant=%s, Kind=%s, StreamID=%s, TrackID=%s", sessionInfo.SessionID, sessionInfo.ParticipantID, track.Kind(), track.StreamID(), track.ID())

		// create a local track to forward to other participants in the session
		localTrack, newTrackErr := webrtc.NewTrackLocalStaticRTP(track.Codec().RTPCodecCapability, track.Kind().String(), track.StreamID())
		if newTrackErr != nil {
			logrus.Errorf("Failed to create local track: %v", newTrackErr)
			return
		}

		sfu.AddTrack(context.Background(), sessionInfo, localTrack)

		sfu.forwardTrackToRoomSessions(sessionInfo.RoomID, sessionInfo.SessionID, localTrack)

		go func() {
			for {
				rtpPacket, _, readErr := track.ReadRTP()
				if readErr != nil {
					logrus.Debugf("Error reading from remote track (Session %s, Participant %s, Kind %s): %v", sessionInfo.SessionID, sessionInfo.ParticipantID, track.Kind(), readErr)
					sfu.RemoveTrack(context.Background(), sessionInfo, localTrack)
					return
				}

				if writeErr := localTrack.WriteRTP(rtpPacket); writeErr != nil {
					logrus.Errorf("Error writing RTP packet to local track (Session %s, Participant %s, Kind %s): %v", sessionInfo.SessionID, sessionInfo.ParticipantID, track.Kind(), writeErr)
					return
				}
				monitoring.IncMediaBytesReceived(track.Kind().String(), float64(len(rtpPacket.Payload)))
			}
		}()

		remoteTrackChan <- track
	})

	session := &MediaSession{
		SessionInfo:      *sessionInfo,
		createdAt:        defaultTimeNow,
		PeerConnection:   peerConnection,
		RemoteTrackChan:  remoteTrackChan,
		localTracks:      []*webrtc.TrackLocalStaticRTP{},
		signalStreamChan: signalStreamChan,
	}
	sfu.sessions[sessionInfo.SessionID] = session

	// start go routing to send signals to gRPC stream
	go sfu.sendSignalsToGRPCStream(session, signalStreamChan)

	logrus.Infof("SFU Session started: SessionID=%s, RoomID=%s, ParticipantID=%s", sessionInfo.SessionID, sessionInfo.RoomID, sessionInfo.ParticipantID)
	monitoring.IncMediaSessionsStarted()
	monitoring.SetActiveMediaSessions(float64(sfu.GetActiveSessionCount()))
	return nil
}

func (sfu *SFUMediaPipeline) StopSession(ctx context.Context, sessionInfo *SessionInfo) error {
	sfu.sessionsMutex.Lock()
	defer sfu.sessionsMutex.Unlock()

	session, exists := sfu.sessions[sessionInfo.SessionID]
	if !exists {
		return fmt.Errorf("session not found: %s", sessionInfo.SessionID)
	}

	if err := session.PeerConnection.Close(); err != nil {
		logrus.Errorf("Error closing peer connection for session %s: %v", sessionInfo.SessionID, err)
	}
	close(session.RemoteTrackChan)
	close(session.signalStreamChan)

	delete(sfu.sessions, sessionInfo.SessionID)
	logrus.Infof("SFU Session stopped: SessionID=%s, RoomID=%s, ParticipantID=%s", sessionInfo.SessionID, sessionInfo.RoomID, sessionInfo.ParticipantID)
	monitoring.IncMediaSessionsStopped()
	monitoring.SetActiveMediaSessions(float64(sfu.GetActiveSessionCount())) // Update metrics
	return nil
}

func (sfu *SFUMediaPipeline) AddTrack(ctx context.Context, sessionInfo *SessionInfo, track *webrtc.TrackLocalStaticRTP) error {
	sfu.sessionsMutex.Lock()
	defer sfu.sessionsMutex.Unlock()

	session, exists := sfu.sessions[sessionInfo.SessionID]
	if !exists {
		return fmt.Errorf("session not found: %s", sessionInfo.SessionID)
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	session.localTracks = append(session.localTracks, track)
	logrus.Debugf("Local track added to session %s, Participant %s, Kind %s", sessionInfo.SessionID, sessionInfo.ParticipantID, track.Kind())
	return nil
}

func (sfu *SFUMediaPipeline) RemoveTrack(ctx context.Context, sessionInfo *SessionInfo, track *webrtc.TrackLocalStaticRTP) error {
	sfu.sessionsMutex.Lock()
	defer sfu.sessionsMutex.Unlock()

	session, exists := sfu.sessions[sessionInfo.SessionID]
	if !exists {
		return fmt.Errorf("session not found: %s", sessionInfo.SessionID)
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	logrus.Debugf("Local track removed from session %s, Participant %s, Kind %s", sessionInfo.SessionID, sessionInfo.ParticipantID, track.Kind())
	return nil
}

func (sfu *SFUMediaPipeline) GetActiveSessionCount() int {
	sfu.sessionsMutex.Lock()
	defer sfu.sessionsMutex.Unlock()
	return len(sfu.sessions)
}

func (sfu *SFUMediaPipeline) HandleWebRTCSignal(ctx context.Context, signal *mediaserver.WebRTCSignal) error {
	sfu.sessionsMutex.Lock()
	defer sfu.sessionsMutex.Unlock()

	session, exists := sfu.sessions[signal.SessionId]
	if !exists {
		return fmt.Errorf("session not found: %s", signal.SessionId)
	}

	switch signal.Type {
	case "offer":
		offer := webrtc.SessionDescription{
			Type: webrtc.SDPTypeOffer,
			SDP:  signal.Sdp,
		}
		if err := session.PeerConnection.SetRemoteDescription(offer); err != nil {
			return fmt.Errorf("failed to set remote description: %w", err)
		}

		answer, err := session.PeerConnection.CreateAnswer(nil)
		if err != nil {
			return fmt.Errorf("failed to create answer: %w", err)
		}

		gatherComplete := webrtc.GatheringCompletePromise(session.PeerConnection)
		if err := session.PeerConnection.SetLocalDescription(answer); err != nil {
			return fmt.Errorf("failed to set local description: %w", err)
		}

		<-gatherComplete // wait for ICE gathering to complete

		localSDP := session.PeerConnection.LocalDescription()
		signal.Sdp = localSDP.SDP
		signal.Type = "answer"

		select {
		case session.signalStreamChan <- signal:
		default:
			return fmt.Errorf("failed to send answer signal: channel full")
		}

	case "answer":
		answer := webrtc.SessionDescription{
			Type: webrtc.SDPTypeAnswer,
			SDP:  signal.Sdp,
		}
		if err := session.PeerConnection.SetRemoteDescription(answer); err != nil {
			return fmt.Errorf("failed to set remote description (answer): %w", err)
		}

	case "candidate":
		candidate := webrtc.ICECandidateInit{}
		if err := json.Unmarshal([]byte(signal.Sdp), &candidate); err != nil {
			return fmt.Errorf("failed to unmarshal ICE candidate: %w", err)
		}
		if err := session.PeerConnection.AddICECandidate(candidate); err != nil {
			return fmt.Errorf("failed to add ICE candidate: %w", err)
		}

	default:
		return fmt.Errorf("unknown signal type: %s", signal.Type)
	}

	return nil
}

func (sfu *SFUMediaPipeline) ReceiveWebRTCSignalStream(ctx context.Context, sessionInfo *SessionInfo, signalChan chan *mediaserver.WebRTCSignal) error {
	sfu.sessionsMutex.Lock()
	defer sfu.sessionsMutex.Unlock()

	session, exists := sfu.sessions[sessionInfo.SessionID]
	if !exists {
		return fmt.Errorf("session not found: %s", sessionInfo.SessionID)
	}

	for {
		select {
		case signal, ok := <-signalChan:
			if !ok {
				logrus.Debugf("ReceiveWebRTCSignalStream - signal channel closed for session %s, participant %s", sessionInfo.SessionID, sessionInfo.ParticipantID)
				return nil
			}
			if signal == nil {
				continue
			}

			if err := sfu.HandleWebRTCSignal(ctx, signal); err != nil {
				logrus.Errorf("Error handling received WebRTC signal for session %s, participant %s: %v", sessionInfo.SessionID, sessionInfo.ParticipantID, err)
			}

		case <-ctx.Done():
			logrus.Debugf("ReceiveWebRTCSignalStream - context cancelled for session %s, participant %s", sessionInfo.SessionID, sessionInfo.ParticipantID)
			return ctx.Err()
		}
	}
}

func (sfu *SFUMediaPipeline) forwardTrackToRoomSessions(roomID string, senderSessionID string, track *webrtc.TrackLocalStaticRTP) {
	sfu.sessionsMutex.Lock()
	defer sfu.sessionsMutex.Unlock()

	for _, session := range sfu.sessions {
		if session.SessionInfo.RoomID == roomID && session.SessionInfo.SessionID != senderSessionID {
			if _, err := session.PeerConnection.AddTrack(track); err != nil {
				logrus.Errorf("Failed to add track to PeerConnection for session %s, participant %s: %v", session.SessionInfo.SessionID, session.SessionInfo.ParticipantID, err)
			} else {
				logrus.Debugf("Forwarded track (Kind %s) to session %s, participant %s", track.Kind(), session.SessionInfo.SessionID, session.SessionInfo.ParticipantID)
			}
		}
	}
}

func (sfu *SFUMediaPipeline) sendSignalsToGRPCStream(session *MediaSession, signalChan chan *mediaserver.WebRTCSignal) {
	for signal := range signalChan {
		logrus.Debugf("Sending WebRTC signal (Type: %s) for Session %s, Participant %s via gRPC stream (placeholder): %+v", signal.Type, signal.SessionID, signal.ParticipantID, signal)
	}
	logrus.Debugf("Stopped sendSignalsToGRPCStream goroutine for session %s, participant %s", session.SessionInfo.SessionID, session.SessionInfo.ParticipantID)
}
