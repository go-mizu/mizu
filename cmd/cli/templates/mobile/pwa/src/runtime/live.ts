import { MizuRuntime } from './MizuRuntime';

export interface ServerEvent {
  id?: string;
  event?: string;
  data: string;
  retry?: number;
}

export function decodeServerEvent<T>(event: ServerEvent): T {
  return JSON.parse(event.data) as T;
}

/**
 * Live connection manager for SSE
 */
export class LiveConnection {
  private runtime: MizuRuntime;
  private activeConnections: Map<string, EventSource> = new Map();

  constructor(runtime: MizuRuntime) {
    this.runtime = runtime;
  }

  /**
   * Connects to an SSE endpoint and returns subscription methods
   */
  connect(
    path: string,
    _headers?: Record<string, string>
  ): {
    subscribe: (onEvent: (event: ServerEvent) => void, onError?: (error: Error) => void) => () => void;
  } {
    return {
      subscribe: (onEvent, onError) => {
        // Cancel any existing connection to this path
        this.disconnect(path);

        const url = this._buildUrl(path);

        // EventSource doesn't support custom headers, so we use query params for auth
        const eventSource = new EventSource(url, { withCredentials: true });

        eventSource.onmessage = (event) => {
          onEvent({
            id: event.lastEventId,
            data: event.data,
          });
        };

        eventSource.onerror = () => {
          if (onError) {
            onError(new Error('SSE connection error'));
          }
        };

        this.activeConnections.set(path, eventSource);

        return () => {
          eventSource.close();
          this.activeConnections.delete(path);
        };
      },
    };
  }

  /** Disconnects from a specific path */
  disconnect(path: string): void {
    const source = this.activeConnections.get(path);
    if (source) {
      source.close();
      this.activeConnections.delete(path);
    }
  }

  /** Disconnects all active connections */
  disconnectAll(): void {
    this.activeConnections.forEach((source) => source.close());
    this.activeConnections.clear();
  }

  private _buildUrl(path: string): string {
    const base = this.runtime.baseURL.endsWith('/')
      ? this.runtime.baseURL.slice(0, -1)
      : this.runtime.baseURL;
    const cleanPath = path.startsWith('/') ? path : `/${path}`;
    return `${base}${cleanPath}`;
  }
}
