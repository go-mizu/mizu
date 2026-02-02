import { useState, useEffect, useCallback } from 'react';
import { Gateway } from '../lib/gateway';
import { truncate } from '../lib/utils';
import { Icon } from '../components/Icon';
import { useToast } from '../components/Toast';

/* ------------------------------------------------------------------ */
/*  Types                                                              */
/* ------------------------------------------------------------------ */

interface Node {
  id: string;
  remoteAddr: string;
  role: string;
  userAgent: string;
}

interface NodesPageProps {
  gw: Gateway;
}

/* ------------------------------------------------------------------ */
/*  Component                                                          */
/* ------------------------------------------------------------------ */

export function NodesPage({ gw }: NodesPageProps) {
  const { toast } = useToast();
  const [nodes, setNodes] = useState<Node[]>([]);
  const [loading, setLoading] = useState(true);
  const [systemInfo, setSystemInfo] = useState<Record<string, string>>({});

  /* --- load ------------------------------------------------------- */
  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await gw.rpc('system.presence');
      setNodes(((res.instances ?? []) as Node[]));
    } catch (err) {
      toast('Failed to load nodes: ' + (err instanceof Error ? err.message : 'unknown'), 'error');
    } finally {
      setLoading(false);
    }
    gw.rpc('system.status').then(res => {
      setSystemInfo({
        go: (res.goVersion || res.go || '') as string,
        os: (res.os || '') as string,
        arch: (res.arch || '') as string,
      });
    }).catch(() => {});
  }, [gw, toast]);

  useEffect(() => { load(); }, [load]);

  /* --- render ----------------------------------------------------- */
  return (
    <div className="card">
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
        <div>
          <div className="card-title">Nodes</div>
          <div className="card-sub">Paired devices and command exposure.</div>
        </div>
        <button className="btn btn--sm" onClick={load} disabled={loading}>
          <Icon name="refresh" />
          <span>Refresh</span>
        </button>
      </div>

      {systemInfo.go && (
        <div className="card" style={{ marginBottom: 16 }}>
          <div className="card-title">System Info</div>
          <div style={{ display: 'flex', gap: 12, marginTop: 8, flexWrap: 'wrap' }}>
            {systemInfo.go && <span className="chip">{systemInfo.go}</span>}
            {systemInfo.os && <span className="chip">{systemInfo.os}/{systemInfo.arch}</span>}
          </div>
        </div>
      )}

      {loading && nodes.length === 0 ? (
        <div className="empty-state">
          <div className="spinner" />
        </div>
      ) : nodes.length === 0 ? (
        <div className="empty-state">
          <div className="empty-state-icon">
            <Icon name="monitor" size={36} />
          </div>
          <div>No nodes connected.</div>
        </div>
      ) : (
        <div className="list" style={{ marginTop: 16 }}>
          {nodes.map((node) => (
            <div className="list-item" key={node.id}>
              <div className="list-main">
                <div className="list-title">{node.id}</div>
                <div className="list-sub">{node.remoteAddr}</div>
                <div className="list-sub mono" style={{ marginTop: 4 }}>
                  {truncate(node.userAgent, 80)}
                </div>
              </div>
              <div className="list-meta">
                <div className="chip-row">
                  <span className="chip chip-ok">{node.role}</span>
                  <span className="chip chip-ok">Connected</span>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
