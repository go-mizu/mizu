import { useState, useEffect, useCallback, useMemo } from 'react';
import { Gateway } from '../lib/gateway';
import { truncate } from '../lib/utils';
import { Icon } from '../components/Icon';
import { useToast } from '../components/Toast';

/* ------------------------------------------------------------------ */
/*  Types                                                              */
/* ------------------------------------------------------------------ */

interface Skill {
  key: string;
  name: string;
  emoji: string;
  description: string;
  source: string;
  eligibility: string;
  userInvocable: boolean;
  enabled: boolean;
  missingBins: string[];
}

interface SkillsPageProps {
  gw: Gateway;
}

/* ------------------------------------------------------------------ */
/*  Component                                                          */
/* ------------------------------------------------------------------ */

export function SkillsPage({ gw }: SkillsPageProps) {
  const { toast } = useToast();
  const [skills, setSkills] = useState<Skill[]>([]);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState('');

  /* --- load ------------------------------------------------------- */
  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await gw.rpc('skills.status');
      setSkills(((res.skills ?? []) as Skill[]));
    } catch (err) {
      toast('Failed to load skills: ' + (err instanceof Error ? err.message : 'unknown'), 'error');
    } finally {
      setLoading(false);
    }
  }, [gw, toast]);

  useEffect(() => { load(); }, [load]);

  /* --- filter ----------------------------------------------------- */
  const filtered = useMemo(() => {
    if (!filter.trim()) return skills;
    const q = filter.toLowerCase();
    return skills.filter(
      (s) =>
        s.name.toLowerCase().includes(q) ||
        s.key.toLowerCase().includes(q) ||
        s.description.toLowerCase().includes(q),
    );
  }, [skills, filter]);

  /* --- toggle ----------------------------------------------------- */
  const handleToggle = useCallback(async (skill: Skill) => {
    try {
      await gw.rpc('skills.toggle', { key: skill.key, enabled: !skill.enabled });
      toast(skill.enabled ? `${skill.name} disabled` : `${skill.name} enabled`, 'success');
      await load();
    } catch (err) {
      toast('Toggle failed: ' + (err instanceof Error ? err.message : 'unknown'), 'error');
    }
  }, [gw, toast, load]);

  /* --- render ----------------------------------------------------- */
  return (
    <div className="card">
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
        <div>
          <div className="card-title">Skills</div>
          <div className="card-sub">
            {skills.length} skill{skills.length !== 1 ? 's' : ''} registered.
          </div>
        </div>
        <button className="btn btn--sm" onClick={load} disabled={loading}>
          <Icon name="refresh" />
          <span>Refresh</span>
        </button>
      </div>

      {/* Filter */}
      <div className="filters" style={{ marginTop: 14 }}>
        <div className="field" style={{ flex: 1, maxWidth: 320 }}>
          <input
            type="text"
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
            placeholder="Search skills..."
          />
        </div>
        <span className="label">{filtered.length} shown</span>
      </div>

      {/* List */}
      {loading && skills.length === 0 ? (
        <div className="empty-state">
          <div className="spinner" />
        </div>
      ) : filtered.length === 0 ? (
        <div className="empty-state">
          <div className="empty-state-icon">
            <Icon name="zap" size={36} />
          </div>
          <div>{skills.length === 0 ? 'No skills registered.' : 'No skills match your filter.'}</div>
        </div>
      ) : (
        <div className="list" style={{ marginTop: 16 }}>
          {filtered.map((skill) => (
            <div className="list-item" key={skill.key}>
              <div className="list-main">
                <div className="list-title">
                  {skill.emoji ? skill.emoji + ' ' : ''}
                  {skill.name}
                </div>
                <div className="list-sub">{truncate(skill.description, 140)}</div>
                <div className="chip-row" style={{ marginTop: 6 }}>
                  <span className="chip">{skill.source}</span>
                  <span
                    className={
                      'chip ' +
                      (skill.eligibility === 'ready' || skill.eligibility === 'ok'
                        ? 'chip-ok'
                        : 'chip-warn')
                    }
                  >
                    {skill.eligibility}
                  </span>
                  {skill.userInvocable && (
                    <span className="chip chip-ok">user-invocable</span>
                  )}
                </div>
                {skill.missingBins && skill.missingBins.length > 0 && (
                  <div className="label" style={{ marginTop: 6 }}>
                    Missing: {skill.missingBins.join(', ')}
                  </div>
                )}
              </div>
              <div className="list-meta">
                <label className="toggle">
                  <input
                    type="checkbox"
                    checked={skill.enabled}
                    onChange={() => handleToggle(skill)}
                  />
                  <span className="toggle-track">
                    <span className="toggle-thumb" />
                  </span>
                </label>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
