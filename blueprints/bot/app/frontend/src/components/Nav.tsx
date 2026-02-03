import { useState, useCallback } from 'react';
import { TAB_GROUPS, TAB_META } from '../config/navigation';
import { Icon } from './Icon';

interface NavProps {
  tab: string;
  onNavigate: (tab: string) => void;
}

const STORAGE_KEY = 'openbot-nav-groups-collapsed';

function loadCollapsed(): Record<string, boolean> {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (raw) return JSON.parse(raw);
  } catch { /* ignore */ }
  return {};
}

function saveCollapsed(state: Record<string, boolean>) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(state));
}

export function Nav({ tab, onNavigate }: NavProps) {
  const [collapsedGroups, setCollapsedGroups] = useState<Record<string, boolean>>(loadCollapsed);

  const toggleGroup = useCallback((label: string) => {
    setCollapsedGroups((prev) => {
      const next = { ...prev, [label]: !prev[label] };
      saveCollapsed(next);
      return next;
    });
  }, []);

  return (
    <nav className="nav">
      {TAB_GROUPS.map((group) => {
        const isCollapsed = !!collapsedGroups[group.label];
        return (
          <div className={'nav-group' + (isCollapsed ? ' nav-group--collapsed' : '')} key={group.label}>
            <button
              className="nav-label nav-label--collapsible"
              onClick={() => toggleGroup(group.label)}
              title={isCollapsed ? 'Expand ' + group.label : 'Collapse ' + group.label}
            >
              <span className="nav-label__chevron">
                <Icon name={isCollapsed ? 'chevronRight' : 'chevronDown'} size={12} />
              </span>
              <span>{group.label}</span>
            </button>
            <div className="nav-group__items">
              {group.tabs.map((t) => {
                const meta = TAB_META[t];
                return (
                  <button
                    key={t}
                    className={'nav-item' + (tab === t ? ' active' : '')}
                    onClick={() => onNavigate(t)}
                  >
                    <span className="nav-icon"><Icon name={meta.icon} /></span>
                    <span>{meta.title}</span>
                  </button>
                );
              })}
            </div>
          </div>
        );
      })}
    </nav>
  );
}
