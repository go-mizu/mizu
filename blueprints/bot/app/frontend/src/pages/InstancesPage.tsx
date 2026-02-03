import { useState, useEffect, useCallback, useMemo } from 'react';
import { Gateway } from '../lib/gateway';
import { formatAgo, truncate } from '../lib/utils';
import { Icon } from '../components/Icon';
import { useToast } from '../components/Toast';

interface Instance {
  id: string;
  remoteAddr: string;
  role: string;
  connectedAt: string | number;
  userAgent: string;
}

type SortField = 'id' | 'connectedAt' | 'remoteAddr';
type SortDir = 'asc' | 'desc';

interface InstancesPageProps {
  gw: Gateway;
}

export function InstancesPage({ gw }: InstancesPageProps) {
  const [instances, setInstances] = useState<Instance[]>([]);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState('');
  const [sortField, setSortField] = useState<SortField>('connectedAt');
  const [sortDir, setSortDir] = useState<SortDir>('desc');
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

  // Subscribe to presence events for real-time updates
  useEffect(() => {
    const unsub = gw.on('event:presence', () => {
      load();
    });
    return unsub;
  }, [gw, load]);

  const toggleSort = useCallback((field: SortField) => {
    setSortField((prev) => {
      if (prev === field) {
        setSortDir((d) => (d === 'asc' ? 'desc' : 'asc'));
        return prev;
      }
      setSortDir('desc');
      return field;
    });
  }, []);

  const filtered = useMemo(() => {
    const q = filter.toLowerCase().trim();
    let list = instances;
    if (q) {
      list = instances.filter(
        (inst) =>
          inst.id.toLowerCase().includes(q) ||
          inst.remoteAddr.toLowerCase().includes(q) ||
          inst.userAgent.toLowerCase().includes(q) ||
          inst.role.toLowerCase().includes(q),
      );
    }
    return [...list].sort((a, b) => {
      let cmp = 0;
      if (sortField === 'connectedAt') {
        const tA = typeof a.connectedAt === 'number' ? a.connectedAt : new Date(a.connectedAt).getTime();
        const tB = typeof b.connectedAt === 'number' ? b.connectedAt : new Date(b.connectedAt).getTime();
        cmp = tA - tB;
      } else if (sortField === 'id') {
        cmp = a.id.localeCompare(b.id);
      } else if (sortField === 'remoteAddr') {
        cmp = a.remoteAddr.localeCompare(b.remoteAddr);
      }
      return sortDir === 'asc' ? cmp : -cmp;
    });
  }, [instances, filter, sortField, sortDir]);

  const sortIcon = (field: SortField) =>
    sortField === field ? (sortDir === 'asc' ? ' \u2191' : ' \u2193') : '';

  return (
    <div className="card">
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
        <div>
          <div className="card-title">Connected Instances</div>
          <div className="card-sub">
            Dashboard clients and nodes connected to this gateway.
            {instances.length > 0 && (
              <span className="chip chip-ok" style={{ marginLeft: 8 }}>{instances.length} online</span>
            )}
          </div>
        </div>
        <button className="btn btn--sm" onClick={load} disabled={loading}>
          <Icon name="refresh" />
          <span>Refresh</span>
        </button>
      </div>

      {instances.length > 0 && (
        <div style={{ display: 'flex', gap: 8, marginTop: 12, alignItems: 'center', flexWrap: 'wrap' }}>
          <input
            type="text"
            placeholder="Filter by ID, address, or user agent..."
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
            style={{ flex: 1, minWidth: 200 }}
          />
          <div className="chip-row">
            <button
              className={'chip ' + (sortField === 'connectedAt' ? 'chip-ok' : '')}
              onClick={() => toggleSort('connectedAt')}
            >
              Time{sortIcon('connectedAt')}
            </button>
            <button
              className={'chip ' + (sortField === 'id' ? 'chip-ok' : '')}
              onClick={() => toggleSort('id')}
            >
              ID{sortIcon('id')}
            </button>
            <button
              className={'chip ' + (sortField === 'remoteAddr' ? 'chip-ok' : '')}
              onClick={() => toggleSort('remoteAddr')}
            >
              Address{sortIcon('remoteAddr')}
            </button>
          </div>
        </div>
      )}

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
      ) : filtered.length === 0 ? (
        <div className="empty-state">
          <div>No instances match "{filter}"</div>
        </div>
      ) : (
        <div className="list" style={{ marginTop: 16 }}>
          {filtered.map((inst) => (
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
