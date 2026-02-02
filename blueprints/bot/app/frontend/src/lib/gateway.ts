type RPCResult = Record<string, unknown>;
type EventHandler = (data?: unknown) => void;

interface PendingRPC {
  resolve: (value: RPCResult) => void;
  reject: (reason: Error) => void;
}

interface HelloOK {
  type: 'hello-ok';
  protocol: number;
  features: { methods: string[]; events: string[] };
}

export class Gateway {
  ws: WebSocket | null = null;
  connected = false;
  hello: HelloOK | null = null;
  private _rid = 0;
  private _pending: Record<string, PendingRPC> = {};
  private _handlers: Record<string, EventHandler[]> = {};

  connect(url: string, token?: string): Promise<HelloOK> {
    return new Promise((resolve, reject) => {
      try {
        this.ws = new WebSocket(url);
      } catch (e) {
        reject(e);
        return;
      }
      this.ws.onopen = () => {
        // Resolve token: explicit param > URL ?token= param > localStorage > empty
        const resolved =
          token ??
          new URLSearchParams(window.location.search).get('token') ??
          localStorage.getItem('openbot-gateway-token') ??
          '';
        if (resolved) localStorage.setItem('openbot-gateway-token', resolved);
        this.ws!.send(JSON.stringify({ type: 'hello', token: resolved }));
      };
      this.ws.onmessage = (evt) => {
        let msg: Record<string, unknown>;
        try {
          msg = JSON.parse(evt.data);
        } catch {
          return;
        }
        if (msg.type === 'hello-ok') {
          this.connected = true;
          this.hello = msg as unknown as HelloOK;
          this._emit('connected', msg);
          resolve(this.hello);
          return;
        }
        if (msg.type === 'hello-error') {
          reject(new Error((msg.error as string) || 'auth failed'));
          return;
        }
        if (msg.type === 'event') {
          this._emit('event', msg);
          this._emit('event:' + msg.event, msg.payload);
          return;
        }
        if (msg.id && this._pending[msg.id as string]) {
          const { resolve: res, reject: rej } = this._pending[msg.id as string];
          delete this._pending[msg.id as string];
          if (msg.error) rej(new Error(msg.error as string));
          else res(msg.result as RPCResult);
        }
      };
      this.ws.onclose = (evt) => {
        this.connected = false;
        const reason = evt.reason || (evt.code === 1008 ? 'unauthorized: check gateway token' : '');
        this._emit('disconnected', reason ? { code: evt.code, reason } : undefined);
      };
      this.ws.onerror = () => {
        this.connected = false;
        reject(new Error('connection failed'));
      };
    });
  }

  rpc(method: string, params?: Record<string, unknown>): Promise<RPCResult> {
    return new Promise((resolve, reject) => {
      if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
        reject(new Error('not connected'));
        return;
      }
      const id = 'r-' + ++this._rid;
      this._pending[id] = { resolve, reject };
      this.ws.send(JSON.stringify({ id, method, params: params || {} }));
      setTimeout(() => {
        if (this._pending[id]) {
          delete this._pending[id];
          reject(new Error('timeout'));
        }
      }, 15000);
    });
  }

  on(event: string, handler: EventHandler): () => void {
    if (!this._handlers[event]) this._handlers[event] = [];
    this._handlers[event].push(handler);
    return () => {
      this._handlers[event] = this._handlers[event].filter((h) => h !== handler);
    };
  }

  private _emit(event: string, data?: unknown) {
    (this._handlers[event] || []).forEach((h) => h(data));
  }

  close() {
    if (this.ws) this.ws.close();
  }
}
