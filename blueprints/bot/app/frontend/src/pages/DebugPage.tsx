import { useState, useEffect, useCallback, useRef } from 'react';
import { Gateway } from '../lib/gateway';
import { formatTime } from '../lib/utils';
import { Icon } from '../components/Icon';
import { useToast } from '../components/Toast';

interface DebugPageProps {
  gw: Gateway;
}

interface EventEntry {
  id: number;
  timestamp: string;
  event: string;
  payload: string;
}

export function DebugPage({ gw }: DebugPageProps) {
  const { toast } = useToast();
  const [statusData, setStatusData] = useState('');
  const [healthData, setHealthData] = useState('');
  const [snapshotLoading, setSnapshotLoading] = useState(false);

  const [rpcMethod, setRpcMethod] = useState('');
  const [rpcParams, setRpcParams] = useState('{}');
  const [rpcResult, setRpcResult] = useState('');
  const [rpcError, setRpcError] = useState('');
  const [rpcCalling, setRpcCalling] = useState(false);
  const [rpcDuration, setRpcDuration] = useState<number | null>(null);

  const [events, setEvents] = useState<EventEntry[]>([]);
  const [lastEventTime, setLastEventTime] = useState<string | null>(null);
  const [connectedSince] = useState<string>(new Date().toISOString());
  const eventIdRef = useRef(0);

  const loadSnapshots = useCallback(async () => {
    setSnapshotLoading(true);
    try {
      const [statusRes, healthRes] = await Promise.all([
        gw.rpc('system.status'),
        gw.rpc('health.check'),
      ]);
      setStatusData(JSON.stringify(statusRes, null, 2));
      setHealthData(JSON.stringify(healthRes, null, 2));
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'unknown error';
      toast('Failed to load snapshots: ' + msg, 'error');
    } finally {
      setSnapshotLoading(false);
    }
  }, [gw, toast]);

  useEffect(() => {
    loadSnapshots();
  }, [loadSnapshots]);

  useEffect(() => {
    const unsubscribe = gw.on('event', (data?: unknown) => {
      const msg = data as Record<string, unknown> | undefined;
      const eventName = (msg?.event as string) ?? 'unknown';
      const payload = msg?.payload !== undefined ? JSON.stringify(msg.payload) : '';
      const now = new Date().toISOString();

      setLastEventTime(now);
      setEvents((prev) => {
        const entry: EventEntry = {
          id: ++eventIdRef.current,
          timestamp: now,
          event: eventName,
          payload: payload.length > 200 ? payload.slice(0, 200) + '...' : payload,
        };
        const next = [entry, ...prev];
        return next.length > 50 ? next.slice(0, 50) : next;
      });
    });
    return unsubscribe;
  }, [gw]);

  const handleRpcCall = useCallback(async () => {
    if (!rpcMethod.trim()) return;
    setRpcCalling(true);
    setRpcResult('');
    setRpcError('');
    setRpcDuration(null);
    const startTime = Date.now();
    try {
      let params: Record<string, unknown> = {};
      if (rpcParams.trim()) {
        params = JSON.parse(rpcParams) as Record<string, unknown>;
      }
      const res = await gw.rpc(rpcMethod.trim(), params);
      const duration = Date.now() - startTime;
      setRpcDuration(duration);
      setRpcResult(JSON.stringify(res, null, 2));
    } catch (err) {
      const duration = Date.now() - startTime;
      setRpcDuration(duration);
      const msg = err instanceof Error ? err.message : 'unknown error';
      setRpcError(msg);
    } finally {
      setRpcCalling(false);
    }
  }, [gw, rpcMethod, rpcParams]);

  const handleMethodClick = useCallback((method: string) => {
    setRpcMethod(method);
    setRpcResult('');
    setRpcError('');
    setRpcDuration(null);
  }, []);

  const handleClearEvents = useCallback(() => {
    setEvents([]);
  }, []);

  const availableMethods = gw.hello?.features?.methods ?? [];

  return (
    <div className="debug-page">
      <div className="debug-grid">
        <div className="card">
          <div className="card__header">
            <h3>Snapshots</h3>
            <button
              className="btn btn-sm"
              onClick={loadSnapshots}
              disabled={snapshotLoading}
            >
              <Icon name="refresh" size={14} />
              Refresh
            </button>
          </div>
          <div className="card__body">
            {snapshotLoading ? (
              <div className="debug-loading">
                <Icon name="loader" size={20} />
                Loading...
              </div>
            ) : (
              <>
                <div className="debug-snapshot">
                  <h4>system.status</h4>
                  <pre className="code-block">{statusData || 'No data'}</pre>
                </div>
                <div className="debug-snapshot">
                  <h4>health.check</h4>
                  <pre className="code-block">{healthData || 'No data'}</pre>
                </div>
              </>
            )}
          </div>
        </div>

        <div className="card">
          <div className="card__header">
            <h3>Manual RPC</h3>
          </div>
          <div className="card__body">
            <div className="debug-rpc-form">
              <label htmlFor="rpc-method">Method</label>
              <input
                id="rpc-method"
                type="text"
                className="input"
                placeholder="e.g. system.status"
                value={rpcMethod}
                onChange={(e) => setRpcMethod(e.target.value)}
              />

              <label htmlFor="rpc-params">Params (JSON)</label>
              <textarea
                id="rpc-params"
                className="input"
                rows={3}
                value={rpcParams}
                onChange={(e) => setRpcParams(e.target.value)}
                spellCheck={false}
              />

              <button
                className="btn primary"
                onClick={handleRpcCall}
                disabled={rpcCalling || !rpcMethod.trim()}
              >
                <Icon name="play" size={14} />
                Call
              </button>
            </div>

            {rpcResult && (
              <div className="debug-rpc-result">
                <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                  <h4>Result</h4>
                  {rpcDuration != null && (
                    <span style={{ color: 'var(--muted)', fontSize: 12 }}>
                      {rpcDuration}ms
                    </span>
                  )}
                </div>
                <pre className="code-block">{rpcResult}</pre>
              </div>
            )}

            {rpcError && (
              <div className="callout callout--error">
                <Icon name="x" size={14} />
                <span>{rpcError}</span>
                {rpcDuration != null && (
                  <span style={{ color: 'var(--muted)', fontSize: 12, marginLeft: 8 }}>
                    {rpcDuration}ms
                  </span>
                )}
              </div>
            )}

            {availableMethods.length > 0 && (
              <div className="debug-methods">
                <h4>Available Methods</h4>
                <div className="debug-methods__chips">
                  {availableMethods.map((method) => (
                    <button
                      key={method}
                      className="chip"
                      onClick={() => handleMethodClick(method)}
                    >
                      {method}
                    </button>
                  ))}
                </div>
              </div>
            )}
          </div>
        </div>
      </div>

      <div className="card debug-events-card">
        <div className="card__header">
          <h3>Event Log</h3>
          <div className="debug-events-meta">
            {lastEventTime && (
              <span className="debug-heartbeat">
                Last event: {formatTime(lastEventTime)}
              </span>
            )}
            <span className="debug-heartbeat">
              Connected since: {formatTime(connectedSince)}
            </span>
          </div>
          <button className="btn btn-sm" onClick={handleClearEvents}>
            <Icon name="trash" size={14} />
            Clear
          </button>
        </div>
        <div className="card__body">
          {events.length === 0 ? (
            <div className="debug-events-empty">No events received yet.</div>
          ) : (
            <div className="log-stream">
              {events.map((entry) => (
                <div key={entry.id} className="log-row debug-event-row">
                  <span className="log-time">{formatTime(entry.timestamp)}</span>
                  <span className="log-event-name">{entry.event}</span>
                  <span className="log-payload">{entry.payload}</span>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
