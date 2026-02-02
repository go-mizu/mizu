import { TAB_GROUPS, TAB_META } from '../config/navigation';
import { Icon } from './Icon';

interface NavProps {
  tab: string;
  onNavigate: (tab: string) => void;
}

export function Nav({ tab, onNavigate }: NavProps) {
  return (
    <nav className="nav">
      {TAB_GROUPS.map((group) => (
        <div className="nav-group" key={group.label}>
          <div className="nav-label">
            <span>{group.label}</span>
          </div>
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
      ))}
    </nav>
  );
}
