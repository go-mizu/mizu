import { useEffect, useState } from 'react';
import { Icon } from './Icon';

function BrandLogo({ size = 28 }: { size?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 32 32" fill="none">
      {/* Head */}
      <rect x="4" y="6" width="24" height="20" rx="4" stroke="currentColor" strokeWidth="1.8" fill="var(--accent-subtle)" />
      {/* Antenna */}
      <line x1="16" y1="6" x2="16" y2="2" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" />
      <circle cx="16" cy="2" r="1.5" fill="var(--accent)" />
      {/* Eyes */}
      <circle cx="11" cy="15" r="2.5" fill="var(--accent)" />
      <circle cx="21" cy="15" r="2.5" fill="var(--accent)" />
      {/* Mouth */}
      <path d="M11 21h10" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" />
      {/* Ear left */}
      <rect x="1" y="12" width="3" height="6" rx="1.5" stroke="currentColor" strokeWidth="1.5" fill="var(--bg-elevated)" />
      {/* Ear right */}
      <rect x="28" y="12" width="3" height="6" rx="1.5" stroke="currentColor" strokeWidth="1.5" fill="var(--bg-elevated)" />
    </svg>
  );
}

interface TopbarProps {
  connected: boolean;
  collapsed: boolean;
  onToggleCollapse: () => void;
}

function getTheme(): string {
  return document.documentElement.getAttribute('data-theme') || 'dark';
}

export function Topbar({ connected, collapsed, onToggleCollapse }: TopbarProps) {
  const [theme, setTheme] = useState(getTheme);

  useEffect(() => {
    document.documentElement.setAttribute('data-theme', theme);
    localStorage.setItem('openbot-theme', theme);
  }, [theme]);

  useEffect(() => {
    const saved = localStorage.getItem('openbot-theme');
    if (saved) {
      setTheme(saved);
      document.documentElement.setAttribute('data-theme', saved);
    }
  }, []);

  function toggleTheme() {
    setTheme((t) => (t === 'dark' ? 'light' : 'dark'));
  }

  return (
    <div className="topbar">
      <div className="topbar-left">
        <button className="nav-collapse-btn" onClick={onToggleCollapse} title={collapsed ? 'Expand' : 'Collapse'}>
          <Icon name="panelLeft" size={16} />
        </button>
        <div className="brand">
          <BrandLogo size={28} />
          <div className="brand-text">
            <div className="brand-title">OpenBot</div>
            <div className="brand-sub">Control Dashboard</div>
          </div>
        </div>
      </div>
      <div className="topbar-right">
        <button className="theme-btn" onClick={toggleTheme}>
          <Icon name={theme === 'dark' ? 'sun' : 'moon'} size={14} />
          <span>{theme === 'dark' ? 'Light' : 'Dark'}</span>
        </button>
        <div className="pill">
          <span className={'statusDot' + (connected ? ' ok' : '')} />
          <span className="mono">{connected ? 'Connected' : 'Disconnected'}</span>
        </div>
      </div>
    </div>
  );
}
