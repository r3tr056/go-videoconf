import { connectSession, verifySocket } from '../modules/api_service';
import { generateId } from '../modules/utils';

class VideoConfContainer {
    constructor(options) {
        this.state = 'INVALID';
        this.title = '';
        this.audio = false;
        this.video = false;
        this.videMode = false;
        this.connection = null;
        this.users = [];
        this.socket = '';
        this.url = options.url || window.location.pathname.split('/meeting/')[1];
        this.localStream = null;
        this.userId = generateId();
        this.localVideoElement = document.getElementById<HTMLVideoElement>('local-video');
        this.stunServer = options.stunServers || STUN_SERVERS;

        window.addEventListener('beforeunload', this.beforeUnloadHandler.bind(this));
    }

    beforeUnloadHandler(event) {
        if (this.connection) {
            this.connection.send('disconnect', this.userId);
        }
    }

    async validateURL() {
        try {
            await verifySocket(this.url);
            this.state = 'VALID_URL';
        } catch (error) {
            console.error('Invalid URL', error);
            window.location.href = '/';
        }
    }

    async connectSession(host, password) {
        try {
            const response = await connectSession(host, password, this.url);
            if (response.data.title) {
                this.title = response.data.title;
            }
            if (response.data.socket) {
                this.socket = response.data.socket;
            }
            this.state = 'LOGGED';
        } catch (error) {
            this.stats = "INVALID";
        }
    }

    toggleJoinMeeting() {
        if (this.state === 'LOGGED') {
            this.state = 'JOINED';
        } else if (this.state === 'JOINED') {
            this.state = 'LOGGED';
        }
    }

    async initMediaStream() {
        if (this.state >= 3 && this.localVideoElement && !this.localVideoElement?.srcObject) {
            try {
                const stream = await navigator.mediaDevices.getUserMedia({ audio: true, video: true });
                this.localVideoElement?.srcObject = stream;
                this.localStream = stream;
                this.audio = true;
                this.video = true;
            } catch (error) {
                console.log('Error accessing media devices: ', error);
            }
        }
    }

    toggleAudioTrack() {
        if (this.localStream) {
            this.localStream.getTracks().forEach(track => {
                if (track.kind === 'audio') {
                    track.enabled = !track.enabled;
                }
            });
            this.audio = !this.audio;
        }
    }

    toggleVideo() {
        if (this.localStream) {
            this.localStream.getTracks().forEach(track => {
                if (track.kind === 'video') {
                    track.enabled = !track.enabled;
                }
            });
            this.video = !this.video;
        }
    }

    async handleConnection() {
        
    }
}