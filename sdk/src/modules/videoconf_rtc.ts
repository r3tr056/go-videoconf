
import { StunServers } from '../interfaces/stun_server';

export class VideoConfRTC {
    servers: StunServers;
    stream: MediaStream;
    peerConnection: RTCPeerConnection;

    constructor(servers: StunServers, stream: MediaStream) {
        this.servers = servers;
        this.stream = stream;

        this.peerConnection = new RTCPeerConnection({iceServers: servers.iceServer});

        this.stream.getTracks().forEach((track: MediaStreamTrack) => {
            this.peerConnection.addTrack(track);
        });
    }

    async createOffer() {
        let sdp: RTCSessionDescriptionInit;
        sdp = await this.peerConnection.createOffer();
        this.peerConnection.setLocalDescription(sdp);

        return sdp;
    }

    async createAnswer() {
        let sdp: RTCSessionDescriptionInit;
        sdp = await this.peerConnection.createAnswer();
        this.peerConnection.setLocalDescription(sdp);
        return sdp;
    }

    async receiveAnswer(event: RTCSessionDescriptionInit | undefined) {
        if (event) {
            await this.peerConnection.setRemoteDescription(new RTCSessionDescription(event));
        }
    }

    async addCandidate(candidate: RTCIceCandidate) {
        await this.peerConnection.addIceCandidate(candidate);
    }

    onTrack(callback: (event: RTCTrackEvent) => void) {
        this.peerConnection.ontrack = callback;
    }

    onIceCandidate(callback: (event: RTCPeerConnectionIceEvent) => void) {
        this.peerConnection.onicecandidate = callback;
    }

    async setRemoteDescription(description: RTCSessionDescriptionInit) {
        await this.peerConnection.setRemoteDescription(description);
    }

    addIceCandidate(candidate: RTCIceCandidate) {
        this.peerConnection.addIceCandidate(candidate);
    }

    diconnect() {
        this.peerConnection.close();
    }
}