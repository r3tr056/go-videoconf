import { ResponseData } from "../interfaces/response_data";

let BASE_URL: string;

export function setBaseUrl(url: string) {
    BASE_URL = url;
}

export async function createSesison(host: string, title: string, password: string): Promise<ResponseData> {
    const response = await fetch(`${BASE_URL}/session`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify({ title, host, password })
    });

    if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
    }

    return response.json();
}

export async function connectSession(host: string, password: string, socket: string): Promise<ResponseData> {
    const response = await fetch(`${BASE_URL}/connect/${socket}`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify({ host, password })
    });

    if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
    }
    return response.json();
}

export async function verifySocket(url: string): Promise<Response> {
    const response = await fetch(`${BASE_URL}/connect?url=${encodeURIComponent(url)}`);
    if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
    }
    return response;
}