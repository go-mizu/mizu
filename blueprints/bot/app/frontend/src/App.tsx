import { useState, useEffect, useRef, useCallback, lazy, Suspense } from 'react';
import { Gateway } from './lib/gateway';
import { Topbar } from './components/Topbar';
import { Nav } from './components/Nav';
import { ContentHeader } from './components/ContentHeader';
import { ToastProvider } from './components/Toast';

const ChatPage = lazy(() => import('./pages/ChatPage').then((m) => ({ default: m.ChatPage })));
const OverviewPage = lazy(() => import('./pages/OverviewPage').then((m) => ({ default: m.OverviewPage })));
const ChannelsPage = lazy(() => import('./pages/ChannelsPage').then((m) => ({ default: m.ChannelsPage })));
const InstancesPage = lazy(() => import('./pages/InstancesPage').then((m) => ({ default: m.InstancesPage })));
const SessionsPage = lazy(() => import('./pages/SessionsPage').then((m) => ({ default: m.SessionsPage })));
const CronPage = lazy(() => import('./pages/CronPage').then((m) => ({ default: m.CronPage })));
const SkillsPage = lazy(() => import('./pages/SkillsPage').then((m) => ({ default: m.SkillsPage })));
const NodesPage = lazy(() => import('./pages/NodesPage').then((m) => ({ default: m.NodesPage })));
const ConfigPage = lazy(() => import('./pages/ConfigPage').then((m) => ({ default: m.ConfigPage })));
const DebugPage = lazy(() => import('./pages/DebugPage').then((m) => ({ default: m.DebugPage })));
const LogsPage = lazy(() => import('./pages/LogsPage').then((m) => ({ default: m.LogsPage })));

function Loader() {
  return (
    <div className="empty-state" style={{ paddingTop: 60 }}>
      <div className="spinner" />
    </div>
  );
}

export function App() {
  const [tab, setTab] = useState(window.location.hash.slice(1) || 'overview');
  const [connected, setConnected] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [collapsed, setCollapsed] = useState(() => localStorage.getItem('openbot-nav-collapsed') === '1');
  const gwRef = useRef<Gateway | null>(null);
  const [retry, setRetry] = useState(0);

  if (!gwRef.current) gwRef.current = new Gateway();
  const gw = gwRef.current;

  useEffect(() => {
    let cancelled = false;
    function connect() {
      const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
      const url = proto + '//' + window.location.host + '/ws';
      gw.connect(url)
        .then(() => {
          if (!cancelled) {
            setConnected(true);
            setError(null);
          }
        })
        .catch((err) => {
          if (!cancelled) {
            setConnected(false);
            setError(err.message);
            setTimeout(() => {
              if (!cancelled) setRetry((r) => r + 1);
            }, 3000);
          }
        });
    }
    connect();
    const unsub = gw.on('disconnected', (data?: unknown) => {
      if (!cancelled) {
        setConnected(false);
        const info = data as { code?: number; reason?: string } | undefined;
        if (info?.reason) setError(info.reason);
        setTimeout(() => {
          if (!cancelled) setRetry((r) => r + 1);
        }, 3000);
      }
    });
    return () => {
      cancelled = true;
      unsub();
    };
  }, [retry, gw]);

  const navigate = useCallback((t: string) => {
    setTab(t);
    window.location.hash = t;
  }, []);

  useEffect(() => {
    function onHash() {
      setTab(window.location.hash.slice(1) || 'overview');
    }
    window.addEventListener('hashchange', onHash);
    return () => window.removeEventListener('hashchange', onHash);
  }, []);

  const toggleCollapse = useCallback(() => {
    setCollapsed((c) => {
      const next = !c;
      localStorage.setItem('openbot-nav-collapsed', next ? '1' : '0');
      return next;
    });
  }, []);

  function renderView() {
    if (!connected) {
      return (
        <div className="empty-state" style={{ paddingTop: 80 }}>
          <div className="spinner" />
          <div>{error ? 'Connection failed: ' + error : 'Connecting to gateway...'}</div>
          <button className="btn" style={{ marginTop: 12 }} onClick={() => setRetry((r) => r + 1)}>
            Retry
          </button>
        </div>
      );
    }
    const props = { gw };
    switch (tab) {
      case 'chat': return <ChatPage {...props} />;
      case 'overview': return <OverviewPage {...props} />;
      case 'channels': return <ChannelsPage {...props} />;
      case 'instances': return <InstancesPage {...props} />;
      case 'sessions': return <SessionsPage {...props} />;
      case 'cron': return <CronPage {...props} />;
      case 'skills': return <SkillsPage {...props} />;
      case 'nodes': return <NodesPage {...props} />;
      case 'config': return <ConfigPage {...props} />;
      case 'debug': return <DebugPage {...props} />;
      case 'logs': return <LogsPage {...props} />;
      default: return <OverviewPage {...props} />;
    }
  }

  return (
    <ToastProvider>
      <div className={'shell' + (tab === 'chat' ? ' shell--chat' : '') + (collapsed ? ' shell--collapsed' : '')}>
        <Topbar connected={connected} collapsed={collapsed} onToggleCollapse={toggleCollapse} />
        <Nav tab={tab} onNavigate={navigate} />
        <div className={'content' + (tab === 'chat' ? ' content--chat' : '')}>
          {tab !== 'chat' && <ContentHeader tab={tab} />}
          <Suspense fallback={<Loader />}>
            {renderView()}
          </Suspense>
        </div>
      </div>
    </ToastProvider>
  );
}
