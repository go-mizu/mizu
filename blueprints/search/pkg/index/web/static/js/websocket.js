// ===================================================================
// WebSocket Client
// ===================================================================
class WSClient {
  constructor() {
    this.ws = null;
    this.listeners = new Map();
    this.connected = false;
    this.reconnectTimer = null;
  }
  connect() {
    if (this.ws && (this.ws.readyState === WebSocket.OPEN || this.ws.readyState === WebSocket.CONNECTING)) return;
    const proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
    try {
      this.ws = new WebSocket(`${proto}//${location.host}/ws`);
    } catch (e) {
      this.scheduleReconnect();
      return;
    }
    this.ws.onopen = () => {
      this.connected = true;
      // Re-subscribe existing listeners
      for (const [jobId] of this.listeners) {
        if (jobId !== '*') {
          this.ws.send(JSON.stringify({ type: 'subscribe', job_ids: [jobId] }));
        }
      }
    };
    this.ws.onmessage = (e) => {
      let msg;
      try { msg = JSON.parse(e.data); } catch { return; }
      const jobId = msg.job_id;
      if (jobId) {
        const cbs = this.listeners.get(jobId);
        if (cbs) cbs.forEach(cb => cb(msg));
        const wildcard = this.listeners.get('*');
        if (wildcard) wildcard.forEach(cb => cb(msg));
      }
    };
    this.ws.onclose = () => {
      this.connected = false;
      this.scheduleReconnect();
    };
    this.ws.onerror = () => {
      this.connected = false;
    };
  }
  scheduleReconnect() {
    if (this.reconnectTimer) return;
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null;
      this.connect();
    }, 2000);
  }
  subscribe(jobId, cb) {
    if (!this.listeners.has(jobId)) this.listeners.set(jobId, []);
    this.listeners.get(jobId).push(cb);
    if (this.ws && this.ws.readyState === WebSocket.OPEN && jobId !== '*') {
      this.ws.send(JSON.stringify({ type: 'subscribe', job_ids: [jobId] }));
    }
  }
  unsubscribe(jobId) {
    this.listeners.delete(jobId);
    if (this.ws && this.ws.readyState === WebSocket.OPEN && jobId !== '*') {
      this.ws.send(JSON.stringify({ type: 'unsubscribe', job_ids: [jobId] }));
    }
  }
  unsubscribeAll() {
    const ids = [...this.listeners.keys()].filter(k => k !== '*');
    this.listeners.clear();
    if (this.ws && this.ws.readyState === WebSocket.OPEN && ids.length > 0) {
      this.ws.send(JSON.stringify({ type: 'unsubscribe', job_ids: ids }));
    }
  }
}

const wsClient = new WSClient();
