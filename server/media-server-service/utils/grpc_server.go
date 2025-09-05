package utils

import (
	"context"
	"io"
	"net"
	"strconv"

	"github.com/r3tr056/go-videoconf/media-service/config"
	"github.com/r3tr056/go-videoconf/media-service/internal/media"
	pb "github.com/r3tr056/go-videoconf/media-service/proto/mediaserver"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

type GRPCServer struct {
	config *config.Config
	pb.UnimplementedMediaStreamingServiceServer
	mediaPipeline media.MediaPipeline
}

func NewGRPCServer(cfg *config.Config, pipeline media.MediaPipeline) *GRPCServer {
	return &GRPCServer{
		config:        cfg,
		mediaPipeline: pipeline,
	}
}

func (s *GRPCServer) StartGRPCServer() error {
	port := s.config.ServicePort
	lis, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		logrus.Fatalf("failed to listen: %v", err)
		return err
	}

	grpcServer := grpc.NewServer()
	pb.RegisterMediaStreamingServiceServer(grpcServer, s)
	reflection.Register(grpcServer)

	logrus.Infof("gRPC server listening on port %d", port)
	if err := grpcServer.Serve(lis); err != nil {
		logrus.Fatalf("failed to serve gRPC: %v", err)
		return err
	}
	return nil
}

func (s *GRPCServer) StartSession(ctx context.Context, req *pb.StartSessionRequest) (*pb.StartSessionResponse, error) {
	sessionInfo := &media.SessionInfo{
		SessionID:     req.GetSessionInfo().GetSessionId(),
		RoomID:        req.GetSessionInfo().GetRoomId(),
		ParticipantID: req.GetSessionInfo().GetParticipantId(),
		MediaServerID: s.config.MediaServerID,
	}

	if err := s.mediaPipeline.StartSession(ctx, sessionInfo); err != nil {
		logrus.Error("StartSession gRPC error: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to start media session: %v", err)
	}

	return &pb.StartSessionResponse{Success: true, Message: "Media session started"}, nil
}

func (s *GRPCServer) StopSession(ctx context.Context, req *pb.StopSessionRequest) (*pb.StopSessionResponse, error) {
	sessionInfo := &media.SessionInfo{
		SessionID:     req.GetSessionInfo().GetSessionId(),
		RoomID:        req.GetSessionInfo().GetRoomId(),
		ParticipantID: req.GetSessionInfo().GetParticipantId(),
		MediaServerID: s.config.MediaServerID,
	}

	if err := s.mediaPipeline.StopSession(ctx, sessionInfo); err != nil {
		logrus.Errorf("StopSession gRPC error: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to stop media session: %v", err)
	}

	return &pb.StopSessionResponse{Success: true, Message: "Media session stopped"}, nil
}

func (s *GRPCServer) SendWebRTCSignal(ctx context.Context, req *pb.SendWebRTCSignalRequest) (*pb.SendWebRTCSignalResponse, error) {
	signal := &pb.WebRTCSignal{
		SessionId:     req.GetSignal().GetSessionId(),
		ParticipantId: req.GetSignal().GetParticipantId(),
		Type:          req.GetSignal().GetType(),
		Sdp:           req.GetSignal().GetSdp(),
		Candidate:     req.GetSignal().GetCandidate(),
	}

	if err := s.mediaPipeline.HandleWebRTCSignal(ctx, signal); err != nil {
		logrus.Errorf("SendWebRTCSignal gRPC error: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to handle WebRTC signal: %v", err)
	}

	return &pb.SendWebRTCSignalResponse{Success: true, Message: "WebRTC signal sent"}, nil
}

func (s *GRPCServer) ReceiveWebRTCSignal(req *pb.ReceiveWebRTCSignalRequest, stream pb.MediaStreamingService_ReceiveWebRTCSignalServer) error {
	sessionInfo := &media.SessionInfo{
		SessionID:     req.GetSessionId(),
		ParticipantID: req.GetParticipantId(),
	}

	signalChan := make(chan *pb.WebRTCSignal)

	go func() {
		err := s.mediaPipeline.ReceiveWebRTCSignalStream(stream.Context(), sessionInfo, signalChan)
		if err != nil {
			logrus.Errorf("ReceiveWebRTCSignalStream error: %v", err)
		}
		close(signalChan)
	}()

	for {
		_, err := stream.
		if err == io.EOF {
			logrus.Debugf("ReceiveWebRTCSignal - stream EOF for session %s, participant %s", sessionInfo.SessionID, sessionInfo.ParticipantID)
			return nil
		}
		if err != nil {
			logrus.Errorf("ReceiveWebRTCSignal - stream receive error for session %s, participant %s: %v", sessionInfo.SessionID, sessionInfo.ParticipantID, err)
			return status.Errorf(codes.Internal, "failed to receive WebRTC signal from stream: %v", err)
		}

		logrus.Warn("eceiveWebRTCSignal - Received unexpected signal from client stream (SFU -> Signaling Service direction). Ignoring.")
	}
}

func (s *GRPCServer) HealthCheck(ctx context.Context, req *pb.HealthCheckRequest) (*pb.HealthCheckResponse, error) {
	// Basic health check - can be expanded
	return &pb.HealthCheckResponse{Healthy: true}, nil
}
