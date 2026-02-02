import { useState, useEffect, useCallback } from 'react';
import { Gateway } from '../lib/gateway';
import { formatAgo, truncate } from '../lib/utils';
import { Icon } from '../components/Icon';
import { useToast } from '../components/Toast';

interface Session {
  id: string;
  displayName: string;
  channelType: string;
  peerId: string;
  status: string;
  agentId: string;
  updatedAt: string;
  tokenCount?: number;
  messageCount?: number;
  inputTokens?: number;
  outputTokens?: number;
  metadata?: string;
}

interface Message {
  role: string;
  content: string;
  timestamp: string;
}

interface SessionsPageProps {
  gw: Gateway;
}

export function SessionsPage({ gw }: SessionsPageProps) {
  const [sessions, setSessions] = useState<Session[]>([]);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState('');
  const [preview, setPreview] = useState<Session | null>(null);
  const [messages, setMessages] = useState<Message[]>([]);
  const [messagesLoading, setMessagesLoading] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [editLabel, setEditLabel] = useState('');
  const { toast } = useToast();

  function startEditing(session: Session) {
    setEditingId(session.id);
    setEditLabel(session.displayName || '');
  }

  function cancelEditing() {
    setEditingId(null);
    setEditLabel('');
  }

  function saveLabel(sessionId: string) {
    gw.rpc('sessions.patch', { key: sessionId, label: editLabel })
      .then(() => {
        toast('Label updated.', 'success');
        setEditingId(null);
        setEditLabel('');
        load();
      })
      .catch((err) => {
        toast('Failed to update label: ' + err.message, 'error');
      });
  }

  const load = useCallback(() => {
    setLoading(true);
    gw.rpc('sessions.list', { limit: 50 })
      .then((res) => {
        setSessions((res.sessions as Session[]) || []);
      })
      .catch((err) => {
        toast('Failed to load sessions: ' + err.message, 'error');
      })
      .finally(() => setLoading(false));
  }, [gw, toast]);

  useEffect(() => {
    load();
  }, [load]);

  const filtered = sessions.filter((s) => {
    if (!filter) return true;
    const q = filter.toLowerCase();
    return (
      (s.displayName || '').toLowerCase().includes(q) ||
      (s.channelType || '').toLowerCase().includes(q) ||
      (s.peerId || '').toLowerCase().includes(q)
    );
  });

  function openPreview(session: Session) {
    setPreview(session);
    setMessages([]);
    setMessagesLoading(true);
    gw.rpc('sessions.preview', { key: session.id, limit: 20 })
      .then((res) => {
        setMessages((res.messages as Message[]) || []);
      })
      .catch((err) => {
        toast('Failed to load messages: ' + err.message, 'error');
      })
      .finally(() => setMessagesLoading(false));
  }

  function closePreview() {
    setPreview(null);
    setMessages([]);
  }

  function deleteSession(id: string) {
    if (!confirm('Delete this session? This cannot be undone.')) return;
    gw.rpc('sessions.delete', { key: id })
      .then(() => {
        setSessions((prev) => prev.filter((s) => s.id !== id));
        toast('Session deleted.', 'success');
        if (preview?.id === id) closePreview();
      })
      .catch((err) => {
        toast('Delete failed: ' + err.message, 'error');
      });
  }

  function getTokenUsage(s: Session): { input: number | null; output: number | null } {
    let input: number | null = s.inputTokens ?? null;
    let output: number | null = s.outputTokens ?? null;
    if (input == null && output == null && s.metadata) {
      try {
        const meta = JSON.parse(s.metadata);
        if (typeof meta.inputTokens === 'number') input = meta.inputTokens;
        if (typeof meta.outputTokens === 'number') output = meta.outputTokens;
      } catch {
        // ignore malformed metadata
      }
    }
    return { input, output };
  }

  return (
    <>
      <div className="card">
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
          <div>
            <div className="card-title">Sessions</div>
            <div className="card-sub">
              Inspect active sessions and adjust per-session defaults.
            </div>
          </div>
          <div style={{ display: 'flex', gap: 8 }}>
            <button
              className="btn btn--sm primary"
              onClick={() => {
                window.location.hash = 'chat';
              }}
            >
              <Icon name="plus" />
              <span>New Chat</span>
            </button>
            <button className="btn btn--sm" onClick={load} disabled={loading}>
              <Icon name="refresh" />
              <span>Refresh</span>
            </button>
          </div>
        </div>

        <div className="filters" style={{ marginTop: 16 }}>
          <div className="field" style={{ flex: 1, maxWidth: 320 }}>
            <input
              type="text"
              placeholder="Filter by name, channel, or peer..."
              value={filter}
              onChange={(e) => setFilter(e.target.value)}
            />
          </div>
        </div>

        {loading && sessions.length === 0 ? (
          <div className="empty-state">
            <div className="spinner" />
          </div>
        ) : filtered.length === 0 ? (
          <div className="empty-state">
            <div className="empty-state-icon">
              <Icon name="fileText" size={36} />
            </div>
            <div>{filter ? 'No sessions match your filter.' : 'No sessions found.'}</div>
          </div>
        ) : (
          <div className="table sessions-table" style={{ marginTop: 12 }}>
            <div className="table-head">
              <span>Session</span>
              <span>Channel</span>
              <span>Peer</span>
              <span>Status</span>
              <span>In Tokens</span>
              <span>Out Tokens</span>
              <span>Updated</span>
              <span>Actions</span>
            </div>
            {filtered.map((s) => {
              const tokens = getTokenUsage(s);
              return (
              <div className="table-row" key={s.id}>
                <div>
                  {editingId === s.id ? (
                    <div style={{ display: 'flex', gap: 4, alignItems: 'center' }}>
                      <input
                        type="text"
                        value={editLabel}
                        onChange={(e) => setEditLabel(e.target.value)}
                        onKeyDown={(e) => {
                          if (e.key === 'Enter') saveLabel(s.id);
                          if (e.key === 'Escape') cancelEditing();
                        }}
                        autoFocus
                        style={{ flex: 1, minWidth: 0, padding: '2px 6px', fontSize: 13 }}
                      />
                      <button
                        className="btn btn--sm"
                        onClick={() => saveLabel(s.id)}
                        title="Save"
                      >
                        <Icon name="check" />
                      </button>
                      <button
                        className="btn btn--sm"
                        onClick={cancelEditing}
                        title="Cancel"
                      >
                        <Icon name="x" />
                      </button>
                    </div>
                  ) : (
                    <>
                      <button
                        className="session-link"
                        onClick={() => openPreview(s)}
                      >
                        {s.displayName || truncate(s.id, 24)}
                      </button>
                      {(s.tokenCount != null || s.messageCount != null) && (
                        <div className="label" style={{ marginTop: 4 }}>
                          {s.messageCount != null && (
                            <span>{s.messageCount} msgs</span>
                          )}
                          {s.messageCount != null && s.tokenCount != null && (
                            <span> / </span>
                          )}
                          {s.tokenCount != null && (
                            <span>{s.tokenCount.toLocaleString()} tokens</span>
                          )}
                        </div>
                      )}
                    </>
                  )}
                </div>
                <div>
                  <span className="chip">{s.channelType}</span>
                </div>
                <div className="mono" style={{ fontSize: 12 }}>
                  {truncate(s.peerId, 20)}
                </div>
                <div>
                  <span
                    className={
                      'chip ' +
                      (s.status === 'active'
                        ? 'chip-ok'
                        : s.status === 'error'
                          ? 'chip-danger'
                          : 'chip-warn')
                    }
                  >
                    {s.status}
                  </span>
                </div>
                <div className="mono" style={{ fontSize: 12 }}>
                  {tokens.input != null ? tokens.input.toLocaleString() : '--'}
                </div>
                <div className="mono" style={{ fontSize: 12 }}>
                  {tokens.output != null ? tokens.output.toLocaleString() : '--'}
                </div>
                <div className="label">{formatAgo(s.updatedAt)}</div>
                <div style={{ display: 'flex', gap: 4 }}>
                  <button
                    className="btn btn--sm"
                    onClick={() => startEditing(s)}
                    title="Edit label"
                  >
                    <Icon name="pencil" />
                  </button>
                  <button
                    className="btn btn--sm danger"
                    onClick={() => deleteSession(s.id)}
                    title="Delete session"
                  >
                    <Icon name="trash" />
                  </button>
                </div>
              </div>
              );
            })}
          </div>
        )}
      </div>

      {preview && (
        <div className="modal-overlay" onClick={closePreview}>
          <div className="modal-card" onClick={(e) => e.stopPropagation()}>
            <div className="modal-header">
              <div className="modal-title">
                {preview.displayName || truncate(preview.id, 32)}
              </div>
              <button className="btn btn--sm" onClick={closePreview}>
                <Icon name="x" />
              </button>
            </div>

            <div className="chip-row" style={{ marginBottom: 16 }}>
              <span className="chip">{preview.channelType}</span>
              <span
                className={
                  'chip ' +
                  (preview.status === 'active'
                    ? 'chip-ok'
                    : preview.status === 'error'
                      ? 'chip-danger'
                      : 'chip-warn')
                }
              >
                {preview.status}
              </span>
              {preview.agentId && (
                <span className="chip">{preview.agentId}</span>
              )}
            </div>

            {(() => {
              const pt = getTokenUsage(preview);
              const hasTokenInfo = preview.tokenCount != null || preview.messageCount != null || pt.input != null || pt.output != null;
              if (!hasTokenInfo) return null;
              return (
                <div className="label" style={{ marginBottom: 12 }}>
                  {preview.messageCount != null && (
                    <span>{preview.messageCount} messages</span>
                  )}
                  {preview.messageCount != null && preview.tokenCount != null && (
                    <span> / </span>
                  )}
                  {preview.tokenCount != null && (
                    <span>{preview.tokenCount.toLocaleString()} tokens</span>
                  )}
                  {(pt.input != null || pt.output != null) && (
                    <span style={{ marginLeft: preview.tokenCount != null || preview.messageCount != null ? 8 : 0 }}>
                      {pt.input != null && <span>In: {pt.input.toLocaleString()}</span>}
                      {pt.input != null && pt.output != null && <span> / </span>}
                      {pt.output != null && <span>Out: {pt.output.toLocaleString()}</span>}
                    </span>
                  )}
                </div>
              );
            })()}

            {messagesLoading ? (
              <div className="empty-state">
                <div className="spinner" />
              </div>
            ) : messages.length === 0 ? (
              <div className="empty-state">
                <div>No messages in this session.</div>
              </div>
            ) : (
              <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
                {messages.map((msg, i) => (
                  <div
                    className={'chat-line ' + (msg.role === 'user' ? 'user' : 'assistant')}
                    key={i}
                  >
                    <div className="chat-msg">
                      <div className="chat-bubble">
                        <div className="chat-text">{msg.content}</div>
                      </div>
                      <div className="chat-stamp">{formatAgo(msg.timestamp)}</div>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      )}
    </>
  );
}
