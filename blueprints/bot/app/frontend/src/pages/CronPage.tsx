import { useState, useEffect, useCallback } from 'react';
import { Gateway } from '../lib/gateway';
import { formatAgo } from '../lib/utils';
import { Icon } from '../components/Icon';
import { useToast } from '../components/Toast';

/* ------------------------------------------------------------------ */
/*  Types                                                              */
/* ------------------------------------------------------------------ */

interface CronJob {
  id: string;
  name: string;
  description: string;
  schedule: string;
  payload: string;
  enabled: boolean;
  lastStatus: string;
  agent: string;
}

interface CronRun {
  id: string;
  status: string;
  started_at: string;
  duration: string;
  summary: string;
}

interface CronStatus {
  enabled: boolean;
  jobs: number;
}

type ScheduleType = 'every' | 'at' | 'cron';
type PayloadType = 'systemEvent' | 'agentTurn';

interface CronPageProps {
  gw: Gateway;
}

/* ------------------------------------------------------------------ */
/*  Component                                                          */
/* ------------------------------------------------------------------ */

export function CronPage({ gw }: CronPageProps) {
  const { toast } = useToast();

  /* --- data state ------------------------------------------------- */
  const [status, setStatus] = useState<CronStatus>({ enabled: false, jobs: 0 });
  const [jobs, setJobs] = useState<CronJob[]>([]);
  const [loading, setLoading] = useState(true);

  /* --- new-job form state ----------------------------------------- */
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [scheduleType, setScheduleType] = useState<ScheduleType>('every');
  const [interval, setInterval] = useState(5);
  const [unit, setUnit] = useState<'minutes' | 'hours' | 'days'>('minutes');
  const [atTime, setAtTime] = useState('09:00');
  const [cronExpr, setCronExpr] = useState('');
  const [payloadType, setPayloadType] = useState<PayloadType>('systemEvent');
  const [payloadText, setPayloadText] = useState('');
  const [adding, setAdding] = useState(false);

  /* --- expandable run-history ------------------------------------- */
  const [expandedJob, setExpandedJob] = useState<string | null>(null);
  const [runs, setRuns] = useState<CronRun[]>([]);
  const [loadingRuns, setLoadingRuns] = useState(false);

  /* --- load ------------------------------------------------------- */
  const load = useCallback(async () => {
    setLoading(true);
    try {
      const [statusRes, listRes] = await Promise.all([
        gw.rpc('cron.status'),
        gw.rpc('cron.list'),
      ]);
      setStatus({
        enabled: (statusRes.enabled ?? false) as boolean,
        jobs: (statusRes.jobs ?? 0) as number,
      });
      setJobs(((listRes.jobs ?? []) as CronJob[]));
    } catch (err) {
      toast('Failed to load cron data: ' + (err instanceof Error ? err.message : 'unknown'), 'error');
    } finally {
      setLoading(false);
    }
  }, [gw, toast]);

  useEffect(() => { load(); }, [load]);

  /* --- add job ---------------------------------------------------- */
  const handleAdd = useCallback(async () => {
    if (!name.trim()) { toast('Job name is required', 'error'); return; }
    setAdding(true);

    let scheduleObj: Record<string, unknown>;
    if (scheduleType === 'every') {
      scheduleObj = { type: 'every', interval, unit };
    } else if (scheduleType === 'at') {
      scheduleObj = { type: 'at', time: atTime };
    } else {
      scheduleObj = { type: 'cron', expression: cronExpr };
    }

    const payloadObj = { type: payloadType, text: payloadText };

    try {
      await gw.rpc('cron.add', {
        name: name.trim(),
        description: description.trim(),
        schedule: JSON.stringify(scheduleObj),
        payload: JSON.stringify(payloadObj),
      });
      toast('Job added', 'success');
      setName('');
      setDescription('');
      setCronExpr('');
      setPayloadText('');
      await load();
    } catch (err) {
      toast('Failed to add job: ' + (err instanceof Error ? err.message : 'unknown'), 'error');
    } finally {
      setAdding(false);
    }
  }, [gw, toast, load, name, description, scheduleType, interval, unit, atTime, cronExpr, payloadType, payloadText]);

  /* --- toggle / run / remove -------------------------------------- */
  const handleToggle = useCallback(async (job: CronJob) => {
    try {
      await gw.rpc('cron.update', { id: job.id, enabled: !job.enabled });
      toast(job.enabled ? 'Job disabled' : 'Job enabled', 'success');
      await load();
    } catch (err) {
      toast('Toggle failed: ' + (err instanceof Error ? err.message : 'unknown'), 'error');
    }
  }, [gw, toast, load]);

  const handleRun = useCallback(async (id: string) => {
    try {
      await gw.rpc('cron.run', { id });
      toast('Job triggered', 'success');
    } catch (err) {
      toast('Run failed: ' + (err instanceof Error ? err.message : 'unknown'), 'error');
    }
  }, [gw, toast]);

  const handleRemove = useCallback(async (id: string) => {
    try {
      await gw.rpc('cron.remove', { id });
      toast('Job removed', 'success');
      await load();
    } catch (err) {
      toast('Remove failed: ' + (err instanceof Error ? err.message : 'unknown'), 'error');
    }
  }, [gw, toast, load]);

  /* --- load runs for a job ---------------------------------------- */
  const handleToggleRuns = useCallback(async (id: string) => {
    if (expandedJob === id) {
      setExpandedJob(null);
      setRuns([]);
      return;
    }
    setExpandedJob(id);
    setLoadingRuns(true);
    try {
      const res = await gw.rpc('cron.runs', { id, limit: 10 });
      setRuns((res.runs ?? []) as CronRun[]);
    } catch (err) {
      toast('Failed to load runs: ' + (err instanceof Error ? err.message : 'unknown'), 'error');
      setRuns([]);
    } finally {
      setLoadingRuns(false);
    }
  }, [gw, toast, expandedJob]);

  /* --- render ----------------------------------------------------- */
  return (
    <>
      {/* Top two-column grid: Scheduler + New Job */}
      <div className="grid grid-cols-2">
        {/* Scheduler card */}
        <div className="card">
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
            <div className="card-title">Scheduler</div>
            <button className="btn btn--sm" onClick={load} disabled={loading}>
              <Icon name="refresh" />
              <span>Refresh</span>
            </button>
          </div>
          <div className="stat-grid" style={{ marginTop: 16 }}>
            <div className="stat">
              <div className="stat-label">ENABLED</div>
              <div className={'stat-value' + (status.enabled ? ' ok' : ' warn')}>
                {status.enabled ? 'Yes' : 'No'}
              </div>
            </div>
            <div className="stat">
              <div className="stat-label">JOBS</div>
              <div className="stat-value">{status.jobs}</div>
            </div>
          </div>
        </div>

        {/* New Job form */}
        <div className="card">
          <div className="card-title">New Job</div>
          <div className="card-sub">Create a new scheduled job.</div>
          <div className="form-grid" style={{ marginTop: 12 }}>
            <div className="field">
              <span>Name</span>
              <input
                type="text"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="my-task"
              />
            </div>
            <div className="field">
              <span>Description</span>
              <input
                type="text"
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                placeholder="Brief description"
              />
            </div>
          </div>

          {/* Schedule type */}
          <div className="form-grid" style={{ marginTop: 12 }}>
            <div className="field">
              <span>Schedule Type</span>
              <select
                value={scheduleType}
                onChange={(e) => setScheduleType(e.target.value as ScheduleType)}
              >
                <option value="every">Every (interval)</option>
                <option value="at">At (fixed time)</option>
                <option value="cron">Cron expression</option>
              </select>
            </div>

            {scheduleType === 'every' && (
              <>
                <div className="field">
                  <span>Interval</span>
                  <input
                    type="number"
                    min={1}
                    value={interval}
                    onChange={(e) => setInterval(Number(e.target.value))}
                  />
                </div>
                <div className="field">
                  <span>Unit</span>
                  <select value={unit} onChange={(e) => setUnit(e.target.value as 'minutes' | 'hours' | 'days')}>
                    <option value="minutes">minutes</option>
                    <option value="hours">hours</option>
                    <option value="days">days</option>
                  </select>
                </div>
              </>
            )}

            {scheduleType === 'at' && (
              <div className="field">
                <span>Time (HH:MM)</span>
                <input
                  type="text"
                  value={atTime}
                  onChange={(e) => setAtTime(e.target.value)}
                  placeholder="09:00"
                />
              </div>
            )}

            {scheduleType === 'cron' && (
              <div className="field">
                <span>Cron Expression</span>
                <input
                  type="text"
                  value={cronExpr}
                  onChange={(e) => setCronExpr(e.target.value)}
                  placeholder="*/5 * * * *"
                />
              </div>
            )}
          </div>

          {/* Payload */}
          <div className="form-grid" style={{ marginTop: 12 }}>
            <div className="field">
              <span>Payload Type</span>
              <select
                value={payloadType}
                onChange={(e) => setPayloadType(e.target.value as PayloadType)}
              >
                <option value="systemEvent">System Event</option>
                <option value="agentTurn">Agent Turn</option>
              </select>
            </div>
            <div className="field">
              <span>Payload Text</span>
              <input
                type="text"
                value={payloadText}
                onChange={(e) => setPayloadText(e.target.value)}
                placeholder="Payload content"
              />
            </div>
          </div>

          <div style={{ marginTop: 16 }}>
            <button className="btn primary" onClick={handleAdd} disabled={adding}>
              <Icon name="plus" />
              <span>{adding ? 'Adding...' : 'Add Job'}</span>
            </button>
          </div>
        </div>
      </div>

      {/* Jobs list */}
      <div className="card">
        <div className="card-title">Jobs</div>
        <div className="card-sub">All scheduled jobs.</div>

        {loading && jobs.length === 0 ? (
          <div className="empty-state">
            <div className="spinner" />
          </div>
        ) : jobs.length === 0 ? (
          <div className="empty-state">
            <div className="empty-state-icon">
              <Icon name="loader" size={36} />
            </div>
            <div>No cron jobs configured.</div>
          </div>
        ) : (
          <div className="list" style={{ marginTop: 16 }}>
            {jobs.map((job) => (
              <div key={job.id}>
                <div className="list-item">
                  <div className="list-main">
                    <div className="list-title">{job.name}</div>
                    <div className="list-sub">{job.description}</div>
                    <div className="chip-row" style={{ marginTop: 6 }}>
                      <span className={'chip ' + (job.enabled ? 'chip-ok' : 'chip-warn')}>
                        {job.enabled ? 'enabled' : 'disabled'}
                      </span>
                      {job.lastStatus && (
                        <span className="chip">{job.lastStatus}</span>
                      )}
                      {job.agent && (
                        <span className="chip">{job.agent}</span>
                      )}
                    </div>
                  </div>
                  <div className="list-meta">
                    <div className="row" style={{ justifyContent: 'flex-end', flexWrap: 'wrap' }}>
                      <label className="toggle">
                        <input
                          type="checkbox"
                          checked={job.enabled}
                          onChange={() => handleToggle(job)}
                        />
                        <span className="toggle-track">
                          <span className="toggle-thumb" />
                        </span>
                      </label>
                      <button className="btn btn--sm" onClick={() => handleRun(job.id)}>
                        <Icon name="play" />
                        <span>Run</span>
                      </button>
                      <button className="btn btn--sm danger" onClick={() => handleRemove(job.id)}>
                        <Icon name="trash" />
                        <span>Remove</span>
                      </button>
                      <button className="btn btn--sm" onClick={() => handleToggleRuns(job.id)}>
                        <Icon name="chevronDown" />
                        <span>Runs</span>
                      </button>
                    </div>
                  </div>
                </div>

                {/* Expandable run history */}
                {expandedJob === job.id && (
                  <div style={{ padding: '8px 12px', borderLeft: '2px solid var(--border)', marginLeft: 12, marginBottom: 8 }}>
                    {loadingRuns ? (
                      <div className="empty-state" style={{ padding: 16 }}>
                        <div className="spinner" />
                      </div>
                    ) : runs.length === 0 ? (
                      <div className="label" style={{ padding: '8px 0' }}>No runs recorded.</div>
                    ) : (
                      <div className="list">
                        {runs.map((run) => (
                          <div className="list-item" key={run.id} style={{ padding: '8px 10px' }}>
                            <div className="list-main">
                              <div className="list-sub">
                                <span className={'chip ' + (run.status === 'ok' ? 'chip-ok' : run.status === 'error' ? 'chip-danger' : '')} style={{ marginRight: 8 }}>
                                  {run.status}
                                </span>
                                {run.summary}
                              </div>
                            </div>
                            <div className="list-meta">
                              <div className="label">{formatAgo(run.started_at)}</div>
                              <div className="label">{run.duration}</div>
                            </div>
                          </div>
                        ))}
                      </div>
                    )}
                  </div>
                )}
              </div>
            ))}
          </div>
        )}
      </div>
    </>
  );
}
