/**
 * LiveClient - WebSocket client for real-time updates
 *
 * Features:
 * - Auto-reconnection with exponential backoff
 * - Topic-based subscriptions
 * - Message correlation via refs
 */
class LiveClient {
    constructor(options) {
        this.url = options.url;
        this.onMessage = options.onMessage || (() => {});
        this.onConnect = options.onConnect || (() => {});
        this.onDisconnect = options.onDisconnect || (() => {});

        this.ws = null;
        this.connected = false;
        this.subscriptions = new Set();
        this.pendingRefs = new Map();
        this.refCounter = 0;

        // Reconnection
        this.reconnectAttempts = 0;
        this.maxReconnectAttempts = 10;
        this.baseDelay = 1000;
        this.maxDelay = 30000;
    }

    /**
     * Connect to WebSocket server
     */
    connect() {
        if (this.ws) {
            return;
        }

        try {
            this.ws = new WebSocket(this.url);

            this.ws.onopen = () => {
                this.connected = true;
                this.reconnectAttempts = 0;
                this.onConnect();

                // Resubscribe to topics
                for (const topic of this.subscriptions) {
                    this.sendSubscribe(topic);
                }
            };

            this.ws.onmessage = (event) => {
                try {
                    const msg = JSON.parse(event.data);
                    this.handleMessage(msg);
                } catch (e) {
                    console.error('Failed to parse message:', e);
                }
            };

            this.ws.onclose = () => {
                this.connected = false;
                this.ws = null;
                this.onDisconnect();
                this.scheduleReconnect();
            };

            this.ws.onerror = (err) => {
                console.error('WebSocket error:', err);
            };
        } catch (e) {
            console.error('Failed to connect:', e);
            this.scheduleReconnect();
        }
    }

    /**
     * Disconnect from server
     */
    disconnect() {
        if (this.ws) {
            this.ws.close();
            this.ws = null;
        }
        this.connected = false;
    }

    /**
     * Subscribe to a topic
     */
    subscribe(topic) {
        this.subscriptions.add(topic);
        if (this.connected) {
            this.sendSubscribe(topic);
        }
    }

    /**
     * Unsubscribe from a topic
     */
    unsubscribe(topic) {
        this.subscriptions.delete(topic);
        if (this.connected) {
            this.send({ type: 'unsubscribe', topic: topic });
        }
    }

    /**
     * Send a message
     */
    send(msg) {
        if (!this.connected || !this.ws) {
            return false;
        }

        try {
            this.ws.send(JSON.stringify(msg));
            return true;
        } catch (e) {
            console.error('Failed to send message:', e);
            return false;
        }
    }

    // Private methods

    sendSubscribe(topic) {
        const ref = this.nextRef();
        this.send({ type: 'subscribe', topic: topic, ref: ref });
    }

    nextRef() {
        return `ref_${++this.refCounter}`;
    }

    handleMessage(msg) {
        // Handle acknowledgments
        if (msg.type === 'ack' && msg.ref) {
            const callback = this.pendingRefs.get(msg.ref);
            if (callback) {
                this.pendingRefs.delete(msg.ref);
                callback(msg);
            }
            return;
        }

        // Pass to handler
        this.onMessage(msg);
    }

    scheduleReconnect() {
        if (this.reconnectAttempts >= this.maxReconnectAttempts) {
            console.error('Max reconnect attempts reached');
            return;
        }

        const delay = Math.min(
            this.baseDelay * Math.pow(2, this.reconnectAttempts),
            this.maxDelay
        );

        this.reconnectAttempts++;

        setTimeout(() => {
            if (!this.connected) {
                this.connect();
            }
        }, delay);
    }
}
