import { SocketEvent } from "../interfaces/response_data";

export class SocketService {
    private connection: WebSocket;

    constructor(url: string) {
        this.connection = new WebSocket(url);
    }

    disconnect() {
        this.connection.close();
    }

    onOpen(callback: () => void) {
        this.connection.onopen = callback;
    }

    send(type: string, userID: string) {
        this.connection.send(JSON.stringify({type, userID}));
    }

    sendDescription(type: string, description: RTCSessionDescriptionInit, userID: string, to: string) {
        this.connection.send(JSON.stringify({ type, userID, description: JSON.stringify(description), to }));
    }

    candidate(candidate: RTCIceCandidate, userID: string) {
        this.connection.send(JSON.stringify({
            candidate: JSON.stringify(candidate),
            userID,
            type: 'ice',
        }));
    }

    onMessage(callback: (event: Event & SocketEvent) => void) {
        this.connection.onmessage = callback;
    }

    isReady(): boolean {
        return this.connection.readyState === this.connection.OPEN;
    }
}