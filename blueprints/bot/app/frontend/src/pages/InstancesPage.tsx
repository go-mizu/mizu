import { useState, useEffect, useCallback } from 'react';
import { Gateway } from '../lib/gateway';
import { formatAgo, truncate } from '../lib/utils';
import { Icon } from '../components/Icon';
import { useToast } from '../components/Toast';

interface Instance {
  id: string;
  remoteAddr: string;
  role: string;
  connectedAt: string;
  userAgent: string;
}

interface InstancesPageProps {
  gw: Gateway;
}

export function InstancesPage({ gw }: InstancesPageProps) {
  const [instances, setInstances] = useState<Instance[]>([]);
  const [loading, setLoading] = useState(true);
  const { toast } = useToast();

  const load = useCallback(() => {
    setLoading(true);
    gw.rpc('system.presence')
      .then((res) => {
        setInstances((res.instances as Instance[]) || []);
      })
      .catch((err) => {
        toast('Failed to load instances: ' + err.message, 'error');
      })
      .finally(() => setLoading(false));
  }, [gw, toast]);

  useEffect(() => {
    load();
  }, [load]);

  return (
    <div className="card">
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
        <div>
          <div className="card-title">Connected Instances</div>
          <div className="card-sub">
            Dashboard clients and nodes connected to this gateway.
          </div>
        </div>
        <button className="btn btn--sm" onClick={load} disabled={loading}>
          <Icon name="refresh" />
          <span>Refresh</span>
        </button>
      </div>

      {loading && instances.length === 0 ? (
        <div className="empty-state">
          <div className="spinner" />
        </div>
      ) : instances.length === 0 ? (
        <div className="empty-state">
          <div className="empty-state-icon">
            <Icon name="radio" size={36} />
          </div>
          <div>No instances connected.</div>
        </div>
      ) : (
        <div className="list" style={{ marginTop: 16 }}>
          {instances.map((inst) => (
            <div className="list-item" key={inst.id}>
              <div className="list-main">
                <div className="list-title">{inst.id}</div>
                <div className="list-sub">{inst.remoteAddr}</div>
              </div>
              <div className="list-meta">
                <div className="chip-row">
                  <span className="chip chip-ok">{inst.role}</span>
                </div>
                <div>{formatAgo(inst.connectedAt)}</div>
                <div className="mono" style={{ fontSize: 11 }}>
                  {truncate(inst.userAgent, 48)}
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
