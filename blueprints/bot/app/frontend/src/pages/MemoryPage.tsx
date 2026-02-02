import { useState, useEffect, useCallback } from 'react';
import { Gateway } from '../lib/gateway';
import { Icon } from '../components/Icon';
import { useToast } from '../components/Toast';

interface MemoryResult {
  path: string;
  source: string;
  startLine: number;
  endLine: number;
  score: number;
  snippet: string;
}

interface MemoryStats {
  files: number;
  chunks: number;
}

interface MemoryPageProps {
  gw: Gateway;
}

export function MemoryPage({ gw }: MemoryPageProps) {
  const [query, setQuery] = useState('');
  const [results, setResults] = useState<MemoryResult[]>([]);
  const [stats, setStats] = useState<MemoryStats | null>(null);
  const [loading, setLoading] = useState(false);
  const [searched, setSearched] = useState(false);
  const { toast } = useToast();

  const loadStats = useCallback(() => {
    gw.rpc('memory.stats')
      .then((res) => {
        setStats({ files: res.files as number, chunks: res.chunks as number });
      })
      .catch(() => {
        // stats not critical
      });
  }, [gw]);

  useEffect(() => {
    loadStats();
  }, [loadStats]);

  function search() {
    if (!query.trim()) return;
    setLoading(true);
    setSearched(true);
    gw.rpc('memory.search', { query: query.trim(), limit: 20 })
      .then((res) => {
        setResults((res.results as MemoryResult[]) || []);
      })
      .catch((err) => {
        toast('Search failed: ' + err.message, 'error');
      })
      .finally(() => setLoading(false));
  }

  return (
    <>
      <div className="card">
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
          <div>
            <div className="card-title">Memory</div>
            <div className="card-sub">
              Search indexed workspace files and session transcripts.
            </div>
          </div>
          <button className="btn btn--sm" onClick={loadStats}>
            <Icon name="refresh" />
            <span>Refresh Stats</span>
          </button>
        </div>

        {stats && (
          <div className="chip-row" style={{ marginTop: 12 }}>
            <span className="chip">{stats.files} files indexed</span>
            <span className="chip">{stats.chunks} chunks</span>
          </div>
        )}

        <div style={{ display: 'flex', gap: 8, marginTop: 16 }}>
          <div className="field" style={{ flex: 1 }}>
            <input
              type="text"
              placeholder="Search memory..."
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') search();
              }}
            />
          </div>
          <button className="btn primary" onClick={search} disabled={loading || !query.trim()}>
            {loading ? <div className="spinner" style={{ width: 16, height: 16 }} /> : <Icon name="search" />}
            <span>Search</span>
          </button>
        </div>
      </div>

      {searched && (
        <div className="card" style={{ marginTop: 16 }}>
          <div className="card-title">
            Results {results.length > 0 && `(${results.length})`}
          </div>

          {loading ? (
            <div className="empty-state">
              <div className="spinner" />
            </div>
          ) : results.length === 0 ? (
            <div className="empty-state">
              <div className="empty-state-icon">
                <Icon name="search" size={36} />
              </div>
              <div>No results found for &quot;{query}&quot;</div>
            </div>
          ) : (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 12, marginTop: 12 }}>
              {results.map((r, i) => (
                <div key={i} className="card" style={{ padding: '12px 16px' }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
                    <div className="mono" style={{ fontSize: 13, fontWeight: 600 }}>
                      {r.path}
                    </div>
                    <div className="chip-row">
                      <span className="chip">{r.source}</span>
                      <span className="chip">
                        L{r.startLine}-{r.endLine}
                      </span>
                      <span className="chip chip-ok">
                        {(r.score * 100).toFixed(0)}%
                      </span>
                    </div>
                  </div>
                  <pre className="code-block" style={{ margin: 0, fontSize: 12, maxHeight: 200, overflow: 'auto' }}>
                    {r.snippet}
                  </pre>
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </>
  );
}
