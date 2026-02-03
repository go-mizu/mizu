import { useState, useEffect, useCallback } from 'react';
import { Gateway } from '../lib/gateway';
import { Icon } from '../components/Icon';

interface SystemStatus {
  ok: boolean;
  uptime: string;
  database: string;
  goVersion: string;
  sessions: number;
  messages: number;
}

interface HealthMemory {
  allocMB: number;
  sysMB: number;
  goroutines: number;
}

interface CronStatus {
  enabled: boolean;
  jobs: number;
  enabledJobs: number;
}

interface ChannelSummary {
  total: number;
  connected: number;
}

interface AgentSummary {
  count: number;
}

interface SkillsSummary {
  total: number;
  enabled: number;
  eligible: number;
}

interface MemorySummary {
  files: number;
  chunks: number;
}

interface AgentIdentity {
  name: string;
  emoji: string;
}

interface ChannelInfo {
  id?: string;
  name?: string;
  status?: string;
  connected?: boolean;
}

interface OverviewData {
  system: SystemStatus;
  health: HealthMemory;
  cron: CronStatus;
  channels: ChannelSummary;
  agents: AgentSummary;
  skills: SkillsSummary;
  memory: MemorySummary;
}

interface OverviewPageProps {
  gw: Gateway;
}

const EMPTY_DATA: OverviewData = {
  system: {
    ok: false,
    uptime: '--',
    database: '--',
    goVersion: '--',
    sessions: 0,
    messages: 0,
  },
  health: { allocMB: 0, sysMB: 0, goroutines: 0 },
  cron: { enabled: false, jobs: 0, enabledJobs: 0 },
  channels: { total: 0, connected: 0 },
  agents: { count: 0 },
  skills: { total: 0, enabled: 0, eligible: 0 },
  memory: { files: 0, chunks: 0 },
};

function navigateTo(tab: string) {
  window.location.hash = tab;
}

export function OverviewPage({ gw }: OverviewPageProps) {
  const [data, setData] = useState<OverviewData>(EMPTY_DATA);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [identity, setIdentity] = useState<AgentIdentity | null>(null);
  const [channelList, setChannelList] = useState<ChannelInfo[]>([]);

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const [systemRes, healthRes, cronRes, channelsRes, agentsRes, cronListRes, skillsRes, memoryRes] = await Promise.all([
        gw.rpc('system.status'),
        gw.rpc('health.check'),
        gw.rpc('cron.status'),
        gw.rpc('channels.status'),
        gw.rpc('agents.list'),
        gw.rpc('cron.list').catch(() => ({ jobs: [] })),
        gw.rpc('skills.status').catch(() => ({ skills: [] })),
        gw.rpc('memory.stats').catch(() => ({ files: 0, chunks: 0 })),
      ]);

      // Load agent identity separately (non-blocking)
      gw.rpc('agent.identity.get').then(res => {
        setIdentity({
          name: (res.name as string) || 'OpenBot',
          emoji: (res.emoji as string) || '\u{1F916}',
        });
      }).catch(() => {});

      // Fix system status parsing: check top-level fields first, fall back to nested status object
      const statusObj = (systemRes.status && typeof systemRes.status === 'object'
        ? systemRes.status as Record<string, unknown>
        : null);

      const system: SystemStatus = {
        ok: (systemRes.ok ?? statusObj?.ok ?? false) as boolean,
        uptime: (systemRes.uptime ?? statusObj?.uptime ?? '--') as string,
        database: (systemRes.database ?? statusObj?.database ?? '--') as string,
        goVersion: (systemRes.goVersion ?? statusObj?.goVersion ?? '--') as string,
        sessions: (systemRes.sessions ?? statusObj?.sessions ?? 0) as number,
        messages: (systemRes.messages ?? statusObj?.messages ?? 0) as number,
      };

      const mem = (healthRes.memory ?? healthRes) as Record<string, unknown>;
      const health: HealthMemory = {
        allocMB: (mem.allocMB ?? 0) as number,
        sysMB: (mem.sysMB ?? 0) as number,
        goroutines: (mem.goroutines ?? 0) as number,
      };

      // Parse cron status from cron.status response
      const cron: CronStatus = {
        enabled: (cronRes.enabled ?? false) as boolean,
        jobs: (cronRes.jobs ?? 0) as number,
        enabledJobs: (cronRes.enabledJobs ?? 0) as number,
      };

      // Enhance cron with cron.list data if cron.status didn't provide enabledJobs
      const cronJobs = Array.isArray(cronListRes.jobs) ? cronListRes.jobs : [];
      if (cronJobs.length > 0) {
        const enabledJobs = cronJobs.filter((j: Record<string, unknown>) => j.enabled);
        // Use cron.list data to fill in totals if cron.status returned zeros
        if (cron.jobs === 0) cron.jobs = cronJobs.length;
        if (cron.enabledJobs === 0) cron.enabledJobs = enabledJobs.length;
      }

      const chList = Array.isArray(channelsRes.channels) ? channelsRes.channels as ChannelInfo[] : [];
      setChannelList(chList);

      const channels: ChannelSummary = {
        total: (channelsRes.total ?? chList.length ?? 0) as number,
        connected: (channelsRes.connected ?? chList.filter(
          (ch: ChannelInfo) => ch.connected === true || ch.status === 'connected',
        ).length ?? 0) as number,
      };

      const agentList = Array.isArray(agentsRes.agents) ? agentsRes.agents : [];
      const agents: AgentSummary = {
        count: (agentsRes.total ?? agentList.length ?? 0) as number,
      };

      // Parse skills summary
      const skillList = Array.isArray(skillsRes.skills) ? skillsRes.skills as Record<string, unknown>[] : [];
      const skills: SkillsSummary = {
        total: skillList.length,
        enabled: skillList.filter((s) => s.enabled === true).length,
        eligible: skillList.filter((s) => s.eligible === true || s.eligible === undefined).length,
      };

      // Parse memory stats
      const memory: MemorySummary = {
        files: (memoryRes.files ?? 0) as number,
        chunks: (memoryRes.chunks ?? 0) as number,
      };

      setData({ system, health, cron, channels, agents, skills, memory });
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load status');
    } finally {
      setLoading(false);
    }
  }, [gw]);

  useEffect(() => {
    load();
  }, [load]);

  const { system, health, cron, channels, agents, skills, memory } = data;

  return (
    <div className="overview-page">
      <div className="overview-header">
        <h2>System Overview</h2>
        <button
          className="btn btn-outline"
          onClick={load}
          disabled={loading}
          title="Refresh"
        >
          <Icon name="refresh" size={16} />
          <span>{loading ? 'Loading...' : 'Refresh'}</span>
        </button>
      </div>

      {error && <div className="overview-error">{error}</div>}

      <div className="stat-grid">
        <div className="stat-card">
          <div className="stat-label">STATUS</div>
          <div className={`stat-value ${system.ok ? 'ok' : ''}`}>
            {system.ok ? 'Healthy' : 'Unhealthy'}
          </div>
        </div>
        <div className="stat-card">
          <div className="stat-label">UPTIME</div>
          <div className="stat-value">{system.uptime}</div>
        </div>
        <div className="stat-card">
          <div className="stat-label">DATABASE</div>
          <div className="stat-value">{system.database}</div>
        </div>
        <div className="stat-card">
          <div className="stat-label">GO VERSION</div>
          <div className="stat-value">{system.goVersion}</div>
        </div>
        <div
          className="stat-card stat-card--link"
          onClick={() => navigateTo('sessions')}
          title="View Sessions"
        >
          <div className="stat-label">SESSIONS</div>
          <div className="stat-value">{system.sessions}</div>
        </div>
        <div className="stat-card">
          <div className="stat-label">MESSAGES</div>
          <div className="stat-value">{system.messages}</div>
        </div>
        <div
          className="stat-card stat-card--link"
          onClick={() => navigateTo('channels')}
          title="View Channels"
        >
          <div className="stat-label">CHANNELS</div>
          <div className={`stat-value ${channels.connected === channels.total && channels.total > 0 ? 'ok' : ''}`}>
            {channels.connected}/{channels.total}
          </div>
          <div className="stat-sub">connected</div>
        </div>
        <div className="stat-card">
          <div className="stat-label">AGENTS</div>
          <div className="stat-value">{agents.count}</div>
        </div>
        <div
          className="stat-card stat-card--link"
          onClick={() => navigateTo('skills')}
          title="View Skills"
        >
          <div className="stat-label">SKILLS</div>
          <div className="stat-value">{skills.enabled}/{skills.total}</div>
          <div className="stat-sub">enabled</div>
        </div>
        <div
          className="stat-card stat-card--link"
          onClick={() => navigateTo('memory')}
          title="View Memory"
        >
          <div className="stat-label">MEMORY</div>
          <div className="stat-value">{memory.files}</div>
          <div className="stat-sub">{memory.chunks} chunks</div>
        </div>
        <div
          className="stat-card stat-card--link"
          onClick={() => navigateTo('cron')}
          title="View Cron Jobs"
        >
          <div className="stat-label">CRON JOBS</div>
          <div className="stat-value">{cron.enabledJobs}/{cron.jobs}</div>
          <div className="stat-sub">active</div>
        </div>
        <div
          className="stat-card stat-card--link"
          onClick={() => navigateTo('instances')}
          title="View Instances"
        >
          <div className="stat-label">INSTANCES</div>
          <div className="stat-value">
            <Icon name="radio" size={14} />
          </div>
          <div className="stat-sub">live</div>
        </div>
      </div>

      <div className="detail-grid">
        {identity && (
          <div className="detail-card">
            <h3>
              <span style={{ fontSize: 20 }}>{identity.emoji}</span>
              <span>Agent Identity</span>
            </h3>
            <table className="detail-table">
              <tbody>
                <tr>
                  <td className="detail-key">Name</td>
                  <td className="detail-val">{identity.name}</td>
                </tr>
                <tr>
                  <td className="detail-key">Emoji</td>
                  <td className="detail-val">{identity.emoji}</td>
                </tr>
              </tbody>
            </table>
          </div>
        )}

        <div className="detail-card">
          <h3>
            <Icon name="monitor" size={16} />
            <span>Health</span>
          </h3>
          <table className="detail-table">
            <tbody>
              <tr>
                <td className="detail-key">Alloc Memory</td>
                <td className="detail-val">{health.allocMB.toFixed(1)} MB</td>
              </tr>
              <tr>
                <td className="detail-key">System Memory</td>
                <td className="detail-val">{health.sysMB.toFixed(1)} MB</td>
              </tr>
              <tr>
                <td className="detail-key">Goroutines</td>
                <td className="detail-val">{health.goroutines}</td>
              </tr>
            </tbody>
          </table>
        </div>

        <div className="detail-card">
          <h3>
            <Icon name="loader" size={16} />
            <span>Scheduler</span>
          </h3>
          <table className="detail-table">
            <tbody>
              <tr>
                <td className="detail-key">Enabled</td>
                <td className="detail-val">{cron.enabled ? 'Yes' : 'No'}</td>
              </tr>
              <tr>
                <td className="detail-key">Total Jobs</td>
                <td className="detail-val">{cron.jobs}</td>
              </tr>
              <tr>
                <td className="detail-key">Enabled Jobs</td>
                <td className="detail-val">
                  {cron.enabledJobs}/{cron.jobs}
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        {skills.total > 0 && (
          <div className="detail-card">
            <h3>
              <Icon name="activity" size={16} />
              <span>Skills</span>
            </h3>
            <table className="detail-table">
              <tbody>
                <tr>
                  <td className="detail-key">Total</td>
                  <td className="detail-val">{skills.total}</td>
                </tr>
                <tr>
                  <td className="detail-key">Enabled</td>
                  <td className="detail-val">{skills.enabled}</td>
                </tr>
                <tr>
                  <td className="detail-key">Eligible</td>
                  <td className="detail-val">{skills.eligible}</td>
                </tr>
              </tbody>
            </table>
          </div>
        )}

        {(memory.files > 0 || memory.chunks > 0) && (
          <div className="detail-card">
            <h3>
              <Icon name="database" size={16} />
              <span>Memory Index</span>
            </h3>
            <table className="detail-table">
              <tbody>
                <tr>
                  <td className="detail-key">Indexed Files</td>
                  <td className="detail-val">{memory.files}</td>
                </tr>
                <tr>
                  <td className="detail-key">Chunks</td>
                  <td className="detail-val">{memory.chunks}</td>
                </tr>
              </tbody>
            </table>
          </div>
        )}

        {channelList.length > 0 && (
          <div className="detail-card">
            <h3>
              <Icon name="radio" size={16} />
              <span>Channels</span>
            </h3>
            <table className="detail-table">
              <tbody>
                {channelList.map((ch: ChannelInfo) => (
                  <tr key={ch.id || ch.name}>
                    <td className="detail-key">{ch.name || ch.id}</td>
                    <td className="detail-val">
                      <span className={'chip ' + (ch.status === 'connected' || ch.connected ? 'chip-ok' : 'chip-warn')}>
                        {ch.status || (ch.connected ? 'connected' : 'disconnected')}
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}
