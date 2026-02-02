import { useEffect, useState } from 'react';
import { Icon } from './Icon';

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
