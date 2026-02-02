import { useState, useEffect, useCallback } from 'react';
import { Gateway } from '../lib/gateway';
import { formatAgo } from '../lib/utils';
import { Icon } from '../components/Icon';
import { useToast } from '../components/Toast';

interface Channel {
  id: string;
  name: string;
  type: string;
  status: string;
  config?: Record<string, unknown>;
  createdAt: string;
}

interface ChannelsPageProps {
  gw: Gateway;
}

export function ChannelsPage({ gw }: ChannelsPageProps) {
  const [channels, setChannels] = useState<Channel[]>([]);
  const [loading, setLoading] = useState(true);
  const [expandedId, setExpandedId] = useState<string | null>(null);
  const [enabledMap, setEnabledMap] = useState<Record<string, boolean>>({});
  const [showCreate, setShowCreate] = useState(false);
  const [creating, setCreating] = useState(false);
  const [newName, setNewName] = useState('');
  const [newType, setNewType] = useState('');
  const [newConfig, setNewConfig] = useState('{}');
  const { toast } = useToast();

  const load = useCallback(() => {
    setLoading(true);
    gw.rpc('channels.status')
      .then((res) => {
        const list = (res.channels as Channel[]) || [];
        setChannels(list);
        setEnabledMap((prev) => {
          const next: Record<string, boolean> = { ...prev };
          for (const ch of list) {
            if (!(ch.id in next)) {
              next[ch.id] = ch.status === 'connected';
            }
          }
          return next;
        });
      })
      .catch((err) => {
        toast('Failed to load channels: ' + err.message, 'error');
      })
      .finally(() => setLoading(false));
  }, [gw, toast]);

  const handleToggle = useCallback((id: string) => {
    setEnabledMap((prev) => {
      const next = { ...prev, [id]: !prev[id] };
      const ch = channels.find((c) => c.id === id);
      toast(
        (ch?.name || id) + (next[id] ? ' enabled' : ' disabled'),
        'success',
      );
      return next;
    });
  }, [channels, toast]);

  const handleDelete = useCallback(async (id: string, name: string) => {
    if (!confirm(`Delete channel "${name || id}"? This cannot be undone.`)) return;
    try {
      await gw.rpc('channels.delete', { id });
      toast('Channel deleted', 'success');
      setExpandedId(null);
      load();
    } catch (err) {
      toast('Failed to delete: ' + (err instanceof Error ? err.message : 'unknown'), 'error');
    }
  }, [gw, toast, load]);

  const handleCreate = useCallback(async () => {
    if (!newName.trim() || !newType.trim()) {
      toast('Name and type are required', 'error');
      return;
    }
    let parsedConfig: Record<string, unknown> = {};
    try {
      parsedConfig = JSON.parse(newConfig);
    } catch {
      toast('Config must be valid JSON', 'error');
      return;
    }
    setCreating(true);
    try {
      await gw.rpc('channels.create', {
        name: newName.trim(),
        type: newType.trim(),
        config: parsedConfig,
        status: 'disconnected',
      });
      toast('Channel created', 'success');
      setShowCreate(false);
      setNewName('');
      setNewType('');
      setNewConfig('{}');
      load();
    } catch (err) {
      toast('Failed to create channel: ' + (err instanceof Error ? err.message : 'unknown'), 'error');
    } finally {
      setCreating(false);
    }
  }, [gw, newName, newType, newConfig, toast, load]);

  useEffect(() => {
    load();
  }, [load]);

  return (
    <div className="card">
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
        <div>
          <div className="card-title">Channels</div>
          <div className="card-sub">Connected messaging platform integrations.</div>
        </div>
        <div style={{ display: 'flex', gap: 8 }}>
          <button className="btn btn--sm" onClick={load} disabled={loading}>
            <Icon name="refresh" />
            <span>Refresh</span>
          </button>
          <button className="btn btn--sm primary" onClick={() => setShowCreate(true)}>
            <Icon name="plus" />
            <span>Create</span>
          </button>
        </div>
      </div>

      {/* Create channel form */}
      {showCreate && (
        <div
          style={{
            marginTop: 16,
            border: '1px solid var(--border)',
            borderRadius: 'var(--radius-md)',
            padding: 16,
            background: 'var(--bg-elevated)',
          }}
        >
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 12 }}>
            <span style={{ fontWeight: 600, fontSize: 14, color: 'var(--text-strong)' }}>
              New Channel
            </span>
            <button
              className="btn btn--sm"
              onClick={() => setShowCreate(false)}
              style={{ padding: '4px 8px' }}
            >
              <Icon name="x" size={14} />
            </button>
          </div>
          <div className="form-grid">
            <div className="field">
              <span>Name</span>
              <input
                type="text"
                value={newName}
                onChange={(e) => setNewName(e.target.value)}
                placeholder="e.g. my-slack-channel"
              />
            </div>
            <div className="field">
              <span>Type</span>
              <select value={newType} onChange={(e) => setNewType(e.target.value)}>
                <option value="">Select type...</option>
                <option value="telegram">Telegram</option>
                <option value="discord">Discord</option>
                <option value="mattermost">Mattermost</option>
                <option value="webhook">Webhook</option>
              </select>
            </div>
          </div>
          <div className="field" style={{ marginTop: 12 }}>
            <span>Config (JSON)</span>
            <textarea
              value={newConfig}
              onChange={(e) => setNewConfig(e.target.value)}
              placeholder='{"token": "...", "channel_id": "..."}'
              style={{ minHeight: 80 }}
            />
          </div>
          <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8, marginTop: 12 }}>
            <button className="btn btn--sm" onClick={() => setShowCreate(false)}>
              Cancel
            </button>
            <button
              className="btn btn--sm primary"
              onClick={handleCreate}
              disabled={creating || !newName.trim() || !newType.trim()}
            >
              {creating ? 'Creating...' : 'Create Channel'}
            </button>
          </div>
        </div>
      )}

      {loading && channels.length === 0 ? (
        <div className="empty-state">
          <div className="spinner" />
        </div>
      ) : channels.length === 0 ? (
        <div className="empty-state">
          <div className="empty-state-icon">
            <Icon name="link" size={36} />
          </div>
          <div>No channels connected.</div>
        </div>
      ) : (
        <div className="list" style={{ marginTop: 16 }}>
          {channels.map((ch) => {
            const isExpanded = expandedId === ch.id;
            const isEnabled = enabledMap[ch.id] ?? ch.status === 'connected';
            return (
              <div key={ch.id}>
                <div
                  className="list-item"
                  style={{ cursor: 'pointer' }}
                  onClick={() => setExpandedId(isExpanded ? null : ch.id)}
                >
                  <div className="list-main">
                    <div className="list-title" style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                      <span
                        style={{
                          display: 'inline-flex',
                          transition: 'transform 0.15s ease',
                          transform: isExpanded ? 'rotate(0deg)' : 'rotate(-90deg)',
                        }}
                      >
                        <Icon name="chevronDown" size={14} />
                      </span>
                      {ch.name}
                    </div>
                    <div className="list-sub mono">{ch.id}</div>
                  </div>
                  <div className="list-meta">
                    <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'flex-end', gap: 12 }}>
                      <div className="chip-row">
                        <span className="chip">{ch.type}</span>
                        <span
                          className={
                            'chip ' + (ch.status === 'connected' ? 'chip-ok' : 'chip-warn')
                          }
                        >
                          {ch.status}
                        </span>
                      </div>
                      <label
                        className="toggle"
                        onClick={(e) => e.stopPropagation()}
                      >
                        <input
                          type="checkbox"
                          checked={isEnabled}
                          onChange={() => handleToggle(ch.id)}
                        />
                        <span className="toggle-track">
                          <span className="toggle-thumb" />
                        </span>
                      </label>
                    </div>
                    <div>{formatAgo(ch.createdAt)}</div>
                  </div>
                </div>
                {isExpanded && (
                  <div
                    style={{
                      border: '1px solid var(--border)',
                      borderTop: 'none',
                      borderRadius: '0 0 var(--radius-md) var(--radius-md)',
                      padding: 16,
                      background: 'var(--bg-elevated)',
                      animation: 'rise 0.2s var(--ease-out)',
                    }}
                  >
                    <div style={{ display: 'grid', gap: 8 }}>
                      <div style={{ display: 'flex', gap: 12 }}>
                        <span className="label" style={{ minWidth: 80 }}>ID</span>
                        <span className="mono" style={{ fontSize: 13 }}>{ch.id}</span>
                      </div>
                      <div style={{ display: 'flex', gap: 12 }}>
                        <span className="label" style={{ minWidth: 80 }}>Name</span>
                        <span style={{ fontSize: 13 }}>{ch.name}</span>
                      </div>
                      <div style={{ display: 'flex', gap: 12 }}>
                        <span className="label" style={{ minWidth: 80 }}>Type</span>
                        <span style={{ fontSize: 13 }}>{ch.type}</span>
                      </div>
                      <div style={{ display: 'flex', gap: 12 }}>
                        <span className="label" style={{ minWidth: 80 }}>Status</span>
                        <span style={{ fontSize: 13 }}>{ch.status}</span>
                      </div>
                      <div style={{ display: 'flex', gap: 12 }}>
                        <span className="label" style={{ minWidth: 80 }}>Created</span>
                        <span style={{ fontSize: 13 }}>{ch.createdAt ? new Date(ch.createdAt).toLocaleString() : 'n/a'}</span>
                      </div>
                      <div style={{ display: 'flex', gap: 12 }}>
                        <span className="label" style={{ minWidth: 80 }}>Enabled</span>
                        <span style={{ fontSize: 13 }}>{isEnabled ? 'Yes' : 'No'}</span>
                      </div>
                    </div>
                    {ch.config && Object.keys(ch.config).length > 0 && (
                      <div style={{ marginTop: 12 }}>
                        <div className="label" style={{ marginBottom: 6 }}>Config</div>
                        <pre className="code-block">{JSON.stringify(ch.config, null, 2)}</pre>
                      </div>
                    )}
                    <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8, marginTop: 16, paddingTop: 12, borderTop: '1px solid var(--border)' }}>
                      <button
                        className="btn btn--sm danger"
                        onClick={(e) => {
                          e.stopPropagation();
                          handleDelete(ch.id, ch.name);
                        }}
                      >
                        <Icon name="trash" size={14} />
                        <span>Delete Channel</span>
                      </button>
                    </div>
                  </div>
                )}
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
