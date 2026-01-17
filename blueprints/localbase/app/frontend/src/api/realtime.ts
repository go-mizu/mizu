import { api } from './client';
import type { Channel, RealtimeStats } from '../types';

export const realtimeApi = {
  // Get realtime stats
  getStats: (): Promise<RealtimeStats> => {
    return api.get<RealtimeStats>('/api/realtime/stats');
  },

  // List active channels
  listChannels: (): Promise<Channel[]> => {
    return api.get<Channel[]>('/api/realtime/channels');
  },

  // Create WebSocket connection URL
  getWebSocketUrl: (): string => {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const serviceKey = localStorage.getItem('serviceKey') || '';
    return `${protocol}//${window.location.host}/realtime/v1/websocket?apikey=${serviceKey}`;
  },
};

// WebSocket client for realtime subscriptions
export class RealtimeClient {
  private ws: WebSocket | null = null;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectDelay = 1000;
  private messageHandlers: ((message: any) => void)[] = [];
  private connectionHandlers: ((connected: boolean) => void)[] = [];

  connect(): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      return;
    }

    const url = realtimeApi.getWebSocketUrl();
    this.ws = new WebSocket(url);

    this.ws.onopen = () => {
      this.reconnectAttempts = 0;
      this.notifyConnectionChange(true);
    };

    this.ws.onclose = () => {
      this.notifyConnectionChange(false);
      this.attemptReconnect();
    };

    this.ws.onerror = () => {
      this.notifyConnectionChange(false);
    };

    this.ws.onmessage = (event) => {
      try {
        const message = JSON.parse(event.data);
        this.notifyMessage(message);
      } catch {
        // Ignore non-JSON messages
      }
    };
  }

  disconnect(): void {
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
  }

  send(message: any): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(message));
    }
  }

  subscribe(channel: string): void {
    this.send({
      type: 'subscribe',
      channel,
    });
  }

  unsubscribe(channel: string): void {
    this.send({
      type: 'unsubscribe',
      channel,
    });
  }

  onMessage(handler: (message: any) => void): () => void {
    this.messageHandlers.push(handler);
    return () => {
      this.messageHandlers = this.messageHandlers.filter((h) => h !== handler);
    };
  }

  onConnectionChange(handler: (connected: boolean) => void): () => void {
    this.connectionHandlers.push(handler);
    return () => {
      this.connectionHandlers = this.connectionHandlers.filter((h) => h !== handler);
    };
  }

  private notifyMessage(message: any): void {
    this.messageHandlers.forEach((handler) => handler(message));
  }

  private notifyConnectionChange(connected: boolean): void {
    this.connectionHandlers.forEach((handler) => handler(connected));
  }

  private attemptReconnect(): void {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      return;
    }

    this.reconnectAttempts++;
    const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1);

    setTimeout(() => {
      this.connect();
    }, delay);
  }
}

export const realtimeClient = new RealtimeClient();
