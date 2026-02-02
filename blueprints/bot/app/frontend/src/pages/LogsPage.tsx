import { useState, useEffect, useCallback, useMemo, useRef } from 'react';
import { Gateway } from '../lib/gateway';
import { formatTime, downloadText } from '../lib/utils';
import { Icon } from '../components/Icon';
import { useToast } from '../components/Toast';

interface LogsPageProps {
  gw: Gateway;
}

interface LogEntry {
  timestamp: string;
  level: string;
  subsystem: string;
  message: string;
}

const LOG_LEVELS = ['trace', 'debug', 'info', 'warn', 'error', 'fatal'] as const;

export function LogsPage({ gw }: LogsPageProps) {
  const { toast } = useToast();
  const [entries, setEntries] = useState<LogEntry[]>([]);
  const [loading, setLoading] = useState(false);
  const [searchTerm, setSearchTerm] = useState('');
  const [autoFollow, setAutoFollow] = useState(true);
  const [activeLevels, setActiveLevels] = useState<Set<string>>(
    () => new Set(LOG_LEVELS),
  );
  const bottomRef = useRef<HTMLDivElement>(null);

  const loadLogs = useCallback(async () => {
    setLoading(true);
    try {
      const res = await gw.rpc('logs.tail', { limit: 200 });
      const list = (res.entries ?? []) as LogEntry[];
      setEntries(list);
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'unknown error';
      toast('Failed to load logs: ' + msg, 'error');
    } finally {
      setLoading(false);
    }
  }, [gw, toast]);

  useEffect(() => {
    loadLogs();
  }, [loadLogs]);

  useEffect(() => {
    if (autoFollow && bottomRef.current) {
      bottomRef.current.scrollIntoView({ behavior: 'smooth' });
    }
  }, [entries, autoFollow]);

  const toggleLevel = useCallback((level: string) => {
    setActiveLevels((prev) => {
      const next = new Set(prev);
      if (next.has(level)) {
        next.delete(level);
      } else {
        next.add(level);
      }
      return next;
    });
  }, []);

  const filteredEntries = useMemo((): LogEntry[] => {
    const lower = searchTerm.toLowerCase();
    return entries.filter((entry) => {
      if (!activeLevels.has(entry.level)) return false;
      if (lower) {
        const haystack =
          (entry.message + ' ' + entry.subsystem + ' ' + entry.level).toLowerCase();
        if (!haystack.includes(lower)) return false;
      }
      return true;
    });
  }, [entries, activeLevels, searchTerm]);

  const handleExport = useCallback(() => {
    if (filteredEntries.length === 0) {
      toast('No entries to export', 'info');
      return;
    }
    const lines = filteredEntries.map(
      (e) => `${e.timestamp} [${e.level}] ${e.subsystem}: ${e.message}`,
    );
    downloadText(lines.join('\n'), 'logs-export.txt');
    toast('Logs exported', 'success');
  }, [filteredEntries, toast]);

  return (
    <div className="logs-page">
      <div className="logs-toolbar">
        <div className="logs-toolbar__left">
          <div className="logs-search-wrapper">
            <Icon name="search" size={14} />
            <input
              type="text"
              className="logs-search"
              placeholder="Filter logs..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
            />
          </div>

          <div className="logs-level-chips">
            {LOG_LEVELS.map((level) => (
              <button
                key={level}
                className={`chip chip--level chip--${level}${activeLevels.has(level) ? ' active' : ''}`}
                onClick={() => toggleLevel(level)}
              >
                {level}
              </button>
            ))}
          </div>
        </div>

        <div className="logs-toolbar__right">
          <label className="logs-auto-follow">
            <input
              type="checkbox"
              checked={autoFollow}
              onChange={(e) => setAutoFollow(e.target.checked)}
            />
            Auto-follow
          </label>
          <span className="logs-count">
            {filteredEntries.length} / {entries.length} entries
          </span>
          <button className="btn btn-sm" onClick={handleExport}>
            <Icon name="download" size={14} />
            Export
          </button>
          <button className="btn btn-sm" onClick={loadLogs} disabled={loading}>
            <Icon name="refresh" size={14} />
            Refresh
          </button>
        </div>
      </div>

      <div className="log-stream">
        {loading && entries.length === 0 ? (
          <div className="logs-loading">
            <Icon name="loader" size={20} />
            Loading logs...
          </div>
        ) : filteredEntries.length === 0 ? (
          <div className="logs-empty">
            {entries.length === 0
              ? 'No log entries available.'
              : 'No entries match the current filters.'}
          </div>
        ) : (
          filteredEntries.map((entry, i) => (
            <div key={i} className="log-row">
              <span className="log-time">{formatTime(entry.timestamp)}</span>
              <span className={`log-level ${entry.level}`}>{entry.level}</span>
              <span className="log-subsystem">{entry.subsystem}</span>
              <span className="log-message">{entry.message}</span>
            </div>
          ))
        )}
        <div ref={bottomRef} />
      </div>
    </div>
  );
}
