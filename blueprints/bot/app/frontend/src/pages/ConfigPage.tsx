import { useState, useEffect, useCallback, useMemo } from 'react';
import { Gateway } from '../lib/gateway';
import { Icon } from '../components/Icon';
import { useToast } from '../components/Toast';

interface ConfigPageProps {
  gw: Gateway;
}

type ConfigData = Record<string, unknown>;

const sectionDescriptions: Record<string, string> = {
  meta: 'Bot identity, name, version, and metadata.',
  wizard: 'Setup wizard configuration and first-run settings.',
  auth: 'Authentication providers, tokens, and access control.',
  agents: 'Agent definitions, models, and behavior tuning.',
  messages: 'Message formatting, templates, and limits.',
  commands: 'Slash commands, aliases, and permission overrides.',
  channels: 'Channel bindings, routing rules, and filters.',
  gateway: 'WebSocket gateway, protocol, and connection settings.',
  plugins: 'Plugin registry, load order, and per-plugin options.',
};

/* ---------- Recursive form field renderer ---------- */

interface FormFieldProps {
  path: string[];
  label: string;
  value: unknown;
  onChange: (path: string[], value: unknown) => void;
  depth?: number;
}

function FormField({ path, label, value, onChange, depth = 0 }: FormFieldProps) {
  if (value === null || value === undefined) {
    return (
      <div className="field" style={{ paddingLeft: depth > 0 ? 16 : 0 }}>
        <span>{label}</span>
        <span style={{ color: 'var(--muted)', fontStyle: 'italic', fontSize: 12 }}>null</span>
      </div>
    );
  }

  if (typeof value === 'boolean') {
    return (
      <div className="field" style={{ paddingLeft: depth > 0 ? 16 : 0 }}>
        <label className="toggle">
          <input
            type="checkbox"
            checked={value}
            onChange={(e) => onChange(path, e.target.checked)}
          />
          <div className="toggle-track">
            <div className="toggle-thumb" />
          </div>
          <span style={{ fontSize: 13, fontWeight: 500, color: 'var(--text)' }}>{label}</span>
        </label>
      </div>
    );
  }

  if (typeof value === 'number') {
    return (
      <div className="field" style={{ paddingLeft: depth > 0 ? 16 : 0 }}>
        <span>{label}</span>
        <input
          type="number"
          value={value}
          onChange={(e) => {
            const v = e.target.value;
            onChange(path, v === '' ? 0 : Number(v));
          }}
        />
      </div>
    );
  }

  if (typeof value === 'string') {
    return (
      <div className="field" style={{ paddingLeft: depth > 0 ? 16 : 0 }}>
        <span>{label}</span>
        <input
          type="text"
          value={value}
          onChange={(e) => onChange(path, e.target.value)}
        />
      </div>
    );
  }

  if (Array.isArray(value)) {
    return (
      <div style={{ paddingLeft: depth > 0 ? 16 : 0, display: 'grid', gap: 8 }}>
        <span style={{ color: 'var(--muted)', fontSize: 13, fontWeight: 500 }}>{label}</span>
        <div
          style={{
            border: '1px solid var(--border)',
            borderRadius: 'var(--radius-md)',
            padding: 12,
            background: 'var(--secondary)',
            display: 'grid',
            gap: 8,
          }}
        >
          {value.length === 0 && (
            <span style={{ color: 'var(--muted)', fontStyle: 'italic', fontSize: 12 }}>
              Empty list
            </span>
          )}
          {value.map((item, idx) => (
            <FormField
              key={idx}
              path={[...path, String(idx)]}
              label={`[${idx}]`}
              value={item}
              onChange={onChange}
              depth={depth + 1}
            />
          ))}
        </div>
      </div>
    );
  }

  if (typeof value === 'object') {
    const entries = Object.entries(value as Record<string, unknown>);
    return (
      <div style={{ paddingLeft: depth > 0 ? 16 : 0, display: 'grid', gap: 10 }}>
        {depth > 0 && (
          <span style={{ color: 'var(--text-strong)', fontSize: 13, fontWeight: 600 }}>
            {label}
          </span>
        )}
        <div className="form-grid">
          {entries.map(([key, val]) => (
            <FormField
              key={key}
              path={[...path, key]}
              label={key}
              value={val}
              onChange={onChange}
              depth={depth + 1}
            />
          ))}
        </div>
      </div>
    );
  }

  // Fallback: render as string
  return (
    <div className="field" style={{ paddingLeft: depth > 0 ? 16 : 0 }}>
      <span>{label}</span>
      <input type="text" value={String(value)} readOnly />
    </div>
  );
}

export function ConfigPage({ gw }: ConfigPageProps) {
  const { toast } = useToast();
  const [raw, setRaw] = useState('');
  const [originalRaw, setOriginalRaw] = useState('');
  const [valid, setValid] = useState(true);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [mode, setMode] = useState<'form' | 'raw'>('form');
  const [activeSection, setActiveSection] = useState('');
  const [searchTerm, setSearchTerm] = useState('');
  const [statusMessage, setStatusMessage] = useState('');

  const loadConfig = useCallback(async () => {
    setLoading(true);
    setStatusMessage('');
    try {
      const res = await gw.rpc('config.read');
      const rawStr = (res.raw ?? '{}') as string;
      const isValid = (res.valid ?? true) as boolean;
      setRaw(rawStr);
      setOriginalRaw(rawStr);
      setValid(isValid);
    } catch (err) {
      toast(
        'Failed to load config: ' + (err instanceof Error ? err.message : 'unknown error'),
        'error',
      );
    } finally {
      setLoading(false);
    }
  }, [gw, toast]);

  useEffect(() => {
    loadConfig();
  }, [loadConfig]);

  const parsedConfig = useMemo((): ConfigData | null => {
    try {
      return JSON.parse(raw) as ConfigData;
    } catch {
      return null;
    }
  }, [raw]);

  const sections = useMemo((): string[] => {
    if (!parsedConfig) return [];
    return Object.keys(parsedConfig);
  }, [parsedConfig]);

  const filteredSections = useMemo((): string[] => {
    if (!searchTerm) return sections;
    const lower = searchTerm.toLowerCase();
    return sections.filter((s) => s.toLowerCase().includes(lower));
  }, [sections, searchTerm]);

  useEffect(() => {
    if (filteredSections.length > 0 && !filteredSections.includes(activeSection)) {
      setActiveSection(filteredSections[0]);
    } else if (filteredSections.length === 0) {
      setActiveSection('');
    }
  }, [filteredSections, activeSection]);

  const hasChanges = raw !== originalRaw;

  const handleRawChange = useCallback((value: string) => {
    setRaw(value);
    try {
      JSON.parse(value);
      setValid(true);
    } catch {
      setValid(false);
    }
  }, []);

  const handleSave = useCallback(async () => {
    if (!valid) {
      toast('Cannot save invalid JSON', 'error');
      return;
    }
    setSaving(true);
    setStatusMessage('');
    try {
      await gw.rpc('config.write', { raw });
      setOriginalRaw(raw);
      setStatusMessage('Saved');
      toast('Configuration saved successfully', 'success');
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'unknown error';
      setStatusMessage('Save failed');
      toast('Failed to save config: ' + msg, 'error');
    } finally {
      setSaving(false);
    }
  }, [gw, raw, valid, toast]);

  const handleApply = useCallback(async () => {
    try {
      await gw.rpc('config.apply');
      toast('Configuration applied (runtime reload)', 'success');
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'unknown error';
      toast('Failed to apply config: ' + msg, 'error');
    }
  }, [gw, toast]);

  const updateSectionValue = useCallback(
    (path: string[], newValue: unknown) => {
      if (!parsedConfig) return;
      // path[0] is section key, rest is nested path within the section
      const updated = JSON.parse(JSON.stringify(parsedConfig)) as ConfigData;
      let target: unknown = updated;
      for (let i = 0; i < path.length - 1; i++) {
        if (target && typeof target === 'object') {
          target = (target as Record<string, unknown>)[path[i]];
        }
      }
      if (target && typeof target === 'object') {
        (target as Record<string, unknown>)[path[path.length - 1]] = newValue;
      }
      const newRaw = JSON.stringify(updated, null, 2);
      setRaw(newRaw);
      setValid(true);
    },
    [parsedConfig],
  );

  const sectionData = useMemo((): string => {
    if (!parsedConfig || !activeSection || !(activeSection in parsedConfig)) {
      return '';
    }
    return JSON.stringify(parsedConfig[activeSection], null, 2);
  }, [parsedConfig, activeSection]);

  const activeSectionValue = useMemo((): unknown => {
    if (!parsedConfig || !activeSection || !(activeSection in parsedConfig)) {
      return undefined;
    }
    return parsedConfig[activeSection];
  }, [parsedConfig, activeSection]);

  return (
    <div className="config-layout">
      <aside className="config-sidebar">
        <div className="config-sidebar__header">
          <h2>Configuration</h2>
          <div className="config-sidebar__modes">
            <button
              className={`btn btn-sm${mode === 'form' ? ' active' : ''}`}
              onClick={() => setMode('form')}
            >
              Form
            </button>
            <button
              className={`btn btn-sm${mode === 'raw' ? ' active' : ''}`}
              onClick={() => setMode('raw')}
            >
              Raw
            </button>
          </div>
        </div>

        {mode === 'form' && (
          <>
            <input
              className="config-search"
              type="text"
              placeholder="Search sections..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
            />
            <nav className="config-nav">
              {filteredSections.map((section) => (
                <button
                  key={section}
                  className={`config-nav__item${activeSection === section ? ' active' : ''}`}
                  onClick={() => setActiveSection(section)}
                >
                  {section}
                </button>
              ))}
              {filteredSections.length === 0 && (
                <div className="config-nav__empty">No sections found</div>
              )}
            </nav>
          </>
        )}
      </aside>

      <main className="config-main">
        <div className="config-actions">
          <button className="btn btn-sm" onClick={loadConfig} disabled={loading}>
            <Icon name="refresh" size={14} />
            Reload
          </button>
          <button
            className="btn btn-sm primary"
            onClick={handleSave}
            disabled={saving || !valid || !hasChanges}
          >
            <Icon name="save" size={14} />
            Save
          </button>
          <button className="btn btn-sm" onClick={handleApply}>
            <Icon name="zap" size={14} />
            Apply
          </button>
          {hasChanges && <span className="config-changes-badge">Unsaved changes</span>}
          {statusMessage && <span className="config-status">{statusMessage}</span>}
          {!valid && <span className="config-status error">Invalid JSON</span>}
        </div>

        {loading ? (
          <div className="config-loading">
            <Icon name="loader" size={24} />
            Loading configuration...
          </div>
        ) : mode === 'raw' ? (
          <textarea
            className="config-raw-editor"
            value={raw}
            onChange={(e) => handleRawChange(e.target.value)}
            spellCheck={false}
          />
        ) : (
          <div className="config-form-view">
            {activeSection ? (
              <>
                <h3 className="config-section-title">{activeSection}</h3>
                {sectionDescriptions[activeSection] && (
                  <p style={{ color: 'var(--muted)', fontSize: 13, margin: '0 0 16px' }}>
                    {sectionDescriptions[activeSection]}
                  </p>
                )}
                {activeSectionValue !== undefined &&
                typeof activeSectionValue === 'object' &&
                activeSectionValue !== null &&
                !Array.isArray(activeSectionValue) ? (
                  <div className="form-grid">
                    {Object.entries(activeSectionValue as Record<string, unknown>).map(
                      ([key, val]) => (
                        <FormField
                          key={key}
                          path={[activeSection, key]}
                          label={key}
                          value={val}
                          onChange={updateSectionValue}
                          depth={0}
                        />
                      ),
                    )}
                  </div>
                ) : (
                  <pre className="code-block">{sectionData}</pre>
                )}
              </>
            ) : (
              <div className="config-empty">
                Select a section from the sidebar to view its configuration.
              </div>
            )}
          </div>
        )}
      </main>
    </div>
  );
}
