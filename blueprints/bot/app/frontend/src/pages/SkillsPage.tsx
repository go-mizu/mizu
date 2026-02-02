import { useState, useEffect, useCallback, useMemo } from 'react';
import { Gateway } from '../lib/gateway';
import { truncate } from '../lib/utils';
import { Icon } from '../components/Icon';
import { useToast } from '../components/Toast';

/* ------------------------------------------------------------------ */
/*  Types                                                              */
/* ------------------------------------------------------------------ */

interface SkillRequirements {
  bins: string[];
  anyBins: string[];
  env: string[];
  config: string[];
  os: string[];
}

interface SkillMissing {
  bins: string[];
  anyBins: string[];
  env: string[];
  config: string[];
  os: string[];
}

interface SkillInstallOpt {
  id: string;
  kind: string;
  label: string;
  bins: string[];
}

interface Skill {
  key: string;
  name: string;
  emoji: string;
  description: string;
  source: string;
  filePath?: string;
  baseDir?: string;
  skillKey: string;
  primaryEnv?: string;
  homepage?: string;
  always: boolean;
  disabled: boolean;
  blockedByAllowlist: boolean;
  eligible: boolean;
  enabled: boolean;
  userInvocable: boolean;
  requirements: SkillRequirements;
  missing: SkillMissing;
  install: SkillInstallOpt[];
}

interface SkillsPageProps {
  gw: Gateway;
}

/* ------------------------------------------------------------------ */
/*  Helpers                                                            */
/* ------------------------------------------------------------------ */

function statusBadge(s: Skill): { label: string; cls: string } {
  if (s.disabled) return { label: 'disabled', cls: 'chip-err' };
  if (s.blockedByAllowlist) return { label: 'blocked', cls: 'chip-err' };
  if (!s.eligible) return { label: 'missing deps', cls: 'chip-warn' };
  if (s.always) return { label: 'always', cls: 'chip-ok' };
  return { label: 'ready', cls: 'chip-ok' };
}

function hasMissing(m: SkillMissing): boolean {
  return (
    m.bins.length > 0 ||
    m.anyBins.length > 0 ||
    m.env.length > 0 ||
    m.config.length > 0 ||
    m.os.length > 0
  );
}

/* ------------------------------------------------------------------ */
/*  Component                                                          */
/* ------------------------------------------------------------------ */

export function SkillsPage({ gw }: SkillsPageProps) {
  const { toast } = useToast();
  const [skills, setSkills] = useState<Skill[]>([]);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState('');
  const [expanded, setExpanded] = useState<string | null>(null);
  const [apiKeyInputs, setApiKeyInputs] = useState<Record<string, string>>({});
  const [envInputs, setEnvInputs] = useState<Record<string, { key: string; value: string }>>({});
  const [installing, setInstalling] = useState<string | null>(null);

  /* --- load ------------------------------------------------------- */
  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await gw.rpc('skills.status');
      setSkills((res.skills ?? []) as Skill[]);
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

  /* --- toggle enabled --------------------------------------------- */
  const handleToggle = useCallback(async (skill: Skill) => {
    try {
      const newEnabled = skill.disabled ? true : !skill.enabled;
      await gw.rpc('skills.update', { skillKey: skill.skillKey, enabled: newEnabled });
      toast(newEnabled ? `${skill.name} enabled` : `${skill.name} disabled`, 'success');
      await load();
    } catch (err) {
      toast('Toggle failed: ' + (err instanceof Error ? err.message : 'unknown'), 'error');
    }
  }, [gw, toast, load]);

  /* --- save API key ----------------------------------------------- */
  const handleSaveApiKey = useCallback(async (skill: Skill) => {
    const val = apiKeyInputs[skill.skillKey] ?? '';
    try {
      await gw.rpc('skills.update', { skillKey: skill.skillKey, apiKey: val });
      toast(`API key ${val ? 'saved' : 'cleared'} for ${skill.name}`, 'success');
      setApiKeyInputs((prev) => {
        const next = { ...prev };
        delete next[skill.skillKey];
        return next;
      });
      await load();
    } catch (err) {
      toast('Save failed: ' + (err instanceof Error ? err.message : 'unknown'), 'error');
    }
  }, [gw, toast, load, apiKeyInputs]);

  /* --- save env var ----------------------------------------------- */
  const handleSaveEnv = useCallback(async (skill: Skill) => {
    const input = envInputs[skill.skillKey];
    if (!input || !input.key.trim()) return;
    try {
      await gw.rpc('skills.update', {
        skillKey: skill.skillKey,
        env: { [input.key.trim()]: input.value },
      });
      toast(`Env ${input.key} ${input.value ? 'set' : 'removed'} for ${skill.name}`, 'success');
      setEnvInputs((prev) => {
        const next = { ...prev };
        delete next[skill.skillKey];
        return next;
      });
      await load();
    } catch (err) {
      toast('Save failed: ' + (err instanceof Error ? err.message : 'unknown'), 'error');
    }
  }, [gw, toast, load, envInputs]);

  /* --- install dep ------------------------------------------------ */
  const handleInstall = useCallback(async (skill: Skill, opt: SkillInstallOpt) => {
    setInstalling(opt.id);
    try {
      const res = await gw.rpc('skills.install', { name: skill.name, installId: opt.id, timeoutMs: 120000 });
      if (res.ok) {
        toast(`Installed ${opt.label} for ${skill.name}`, 'success');
      } else {
        toast(`Install failed: ${res.message || 'unknown error'}`, 'error');
      }
      await load();
    } catch (err) {
      toast('Install error: ' + (err instanceof Error ? err.message : 'unknown'), 'error');
    } finally {
      setInstalling(null);
    }
  }, [gw, toast, load]);

  /* --- render ----------------------------------------------------- */
  return (
    <div className="card">
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
        <div>
          <div className="card-title">Skills</div>
          <div className="card-sub">
            {skills.length} skill{skills.length !== 1 ? 's' : ''} registered
            {' \u00b7 '}
            {skills.filter((s) => s.eligible).length} eligible
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
          {filtered.map((sk) => {
            const badge = statusBadge(sk);
            const isExpanded = expanded === sk.key;
            return (
              <div className="list-item" key={sk.key} style={{ flexDirection: 'column', alignItems: 'stretch' }}>
                <div style={{ display: 'flex', alignItems: 'flex-start' }}>
                  <div
                    className="list-main"
                    style={{ cursor: 'pointer', flex: 1 }}
                    onClick={() => setExpanded(isExpanded ? null : sk.key)}
                  >
                    <div className="list-title">
                      {sk.emoji ? sk.emoji + ' ' : ''}
                      {sk.name}
                      {sk.homepage && (
                        <a
                          href={sk.homepage}
                          target="_blank"
                          rel="noopener noreferrer"
                          style={{ marginLeft: 8, fontSize: '0.8em', opacity: 0.6 }}
                          onClick={(e) => e.stopPropagation()}
                        >
                          homepage
                        </a>
                      )}
                    </div>
                    <div className="list-sub">{truncate(sk.description, 140)}</div>
                    <div className="chip-row" style={{ marginTop: 6 }}>
                      <span className="chip">{sk.source}</span>
                      <span className={'chip ' + badge.cls}>{badge.label}</span>
                      {sk.userInvocable && (
                        <span className="chip chip-ok">user-invocable</span>
                      )}
                      {sk.always && (
                        <span className="chip">always</span>
                      )}
                    </div>

                    {/* Missing summary */}
                    {hasMissing(sk.missing) && (
                      <div className="label" style={{ marginTop: 6, color: 'var(--c-warn, #e8a735)' }}>
                        {sk.missing.bins.length > 0 && (
                          <span>Missing bins: {sk.missing.bins.join(', ')}. </span>
                        )}
                        {sk.missing.anyBins.length > 0 && (
                          <span>Need one of: {sk.missing.anyBins.join(', ')}. </span>
                        )}
                        {sk.missing.env.length > 0 && (
                          <span>Missing env: {sk.missing.env.join(', ')}. </span>
                        )}
                        {sk.missing.config.length > 0 && (
                          <span>Missing config: {sk.missing.config.join(', ')}. </span>
                        )}
                        {sk.missing.os.length > 0 && (
                          <span>Wrong OS (needs: {sk.missing.os.join(', ')}). </span>
                        )}
                      </div>
                    )}
                  </div>

                  <div className="list-meta">
                    <label className="toggle">
                      <input
                        type="checkbox"
                        checked={sk.enabled && !sk.disabled}
                        onChange={() => handleToggle(sk)}
                      />
                      <span className="toggle-track">
                        <span className="toggle-thumb" />
                      </span>
                    </label>
                  </div>
                </div>

                {/* --- Expanded detail view --- */}
                {isExpanded && (
                  <div style={{ padding: '12px 0 4px', borderTop: '1px solid var(--c-border, #333)' }}>
                    {/* Requirements */}
                    {(sk.requirements.bins.length > 0 || sk.requirements.anyBins.length > 0 ||
                      sk.requirements.env.length > 0 || sk.requirements.config.length > 0 ||
                      sk.requirements.os.length > 0) && (
                      <div style={{ marginBottom: 12 }}>
                        <div className="label" style={{ marginBottom: 4, fontWeight: 600 }}>Requirements</div>
                        {sk.requirements.bins.length > 0 && (
                          <div className="label">Binaries (all): {sk.requirements.bins.join(', ')}</div>
                        )}
                        {sk.requirements.anyBins.length > 0 && (
                          <div className="label">Binaries (any): {sk.requirements.anyBins.join(', ')}</div>
                        )}
                        {sk.requirements.env.length > 0 && (
                          <div className="label">Env vars: {sk.requirements.env.join(', ')}</div>
                        )}
                        {sk.requirements.config.length > 0 && (
                          <div className="label">Config paths: {sk.requirements.config.join(', ')}</div>
                        )}
                        {sk.requirements.os.length > 0 && (
                          <div className="label">OS: {sk.requirements.os.join(', ')}</div>
                        )}
                      </div>
                    )}

                    {/* Install options */}
                    {sk.install && sk.install.length > 0 && (
                      <div style={{ marginBottom: 12 }}>
                        <div className="label" style={{ marginBottom: 4, fontWeight: 600 }}>Install</div>
                        <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
                          {sk.install.map((opt) => (
                            <button
                              key={opt.id}
                              className="btn btn--sm"
                              disabled={installing === opt.id}
                              onClick={() => handleInstall(sk, opt)}
                            >
                              {installing === opt.id ? (
                                <span className="spinner" style={{ width: 14, height: 14 }} />
                              ) : (
                                <Icon name="download" />
                              )}
                              <span>{opt.label || `${opt.kind}: ${opt.bins.join(', ')}`}</span>
                            </button>
                          ))}
                        </div>
                      </div>
                    )}

                    {/* API Key input */}
                    {sk.primaryEnv && (
                      <div style={{ marginBottom: 12 }}>
                        <div className="label" style={{ marginBottom: 4, fontWeight: 600 }}>
                          API Key ({sk.primaryEnv})
                        </div>
                        <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
                          <div className="field" style={{ flex: 1, maxWidth: 400 }}>
                            <input
                              type="password"
                              placeholder={`Enter ${sk.primaryEnv}...`}
                              value={apiKeyInputs[sk.skillKey] ?? ''}
                              onChange={(e) =>
                                setApiKeyInputs((prev) => ({ ...prev, [sk.skillKey]: e.target.value }))
                              }
                            />
                          </div>
                          <button className="btn btn--sm" onClick={() => handleSaveApiKey(sk)}>
                            Save
                          </button>
                        </div>
                      </div>
                    )}

                    {/* Env vars editor */}
                    {sk.requirements.env.length > 0 && (
                      <div style={{ marginBottom: 12 }}>
                        <div className="label" style={{ marginBottom: 4, fontWeight: 600 }}>Environment Variables</div>
                        <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
                          <div className="field" style={{ maxWidth: 160 }}>
                            <input
                              type="text"
                              placeholder="VAR_NAME"
                              value={envInputs[sk.skillKey]?.key ?? ''}
                              onChange={(e) =>
                                setEnvInputs((prev) => ({
                                  ...prev,
                                  [sk.skillKey]: { ...prev[sk.skillKey], key: e.target.value, value: prev[sk.skillKey]?.value ?? '' },
                                }))
                              }
                            />
                          </div>
                          <div className="field" style={{ flex: 1, maxWidth: 300 }}>
                            <input
                              type="text"
                              placeholder="value"
                              value={envInputs[sk.skillKey]?.value ?? ''}
                              onChange={(e) =>
                                setEnvInputs((prev) => ({
                                  ...prev,
                                  [sk.skillKey]: { ...prev[sk.skillKey], key: prev[sk.skillKey]?.key ?? '', value: e.target.value },
                                }))
                              }
                            />
                          </div>
                          <button className="btn btn--sm" onClick={() => handleSaveEnv(sk)}>
                            Set
                          </button>
                        </div>
                      </div>
                    )}

                    {/* Metadata */}
                    <div className="label" style={{ opacity: 0.5 }}>
                      Key: {sk.skillKey}
                      {sk.filePath && <> &middot; {sk.filePath}</>}
                      {sk.baseDir && <> &middot; Dir: {sk.baseDir}</>}
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
