
export interface ResponseData {
    data: {
        socket?: string;
        title?: string;
    }
}

export interface SocketResponse {
    type: string;
    userID: string;
    description: string;
    candidate: string;
    label: string;
    to: string;
}

export interface SocketEvent {
    data: string
}