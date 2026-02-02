export interface TabMeta {
  title: string;
  sub: string;
  icon: string;
}

export interface TabGroup {
  label: string;
  tabs: string[];
}

export const TAB_GROUPS: TabGroup[] = [
  { label: 'Chat', tabs: ['chat'] },
  { label: 'Control', tabs: ['overview', 'channels', 'instances', 'sessions', 'cron'] },
  { label: 'Agent', tabs: ['skills', 'nodes'] },
  { label: 'Settings', tabs: ['config', 'debug', 'logs'] },
];

export const TAB_META: Record<string, TabMeta> = {
  chat:      { title: 'Chat',       sub: 'Direct gateway chat session for quick interventions.', icon: 'messageSquare' },
  overview:  { title: 'Overview',   sub: 'Gateway status, entry points, and a fast health read.',  icon: 'barChart' },
  channels:  { title: 'Channels',   sub: 'Manage channels and settings.',                         icon: 'link' },
  instances: { title: 'Instances',  sub: 'Presence beacons from connected clients and nodes.',     icon: 'radio' },
  sessions:  { title: 'Sessions',   sub: 'Inspect active sessions and adjust per-session defaults.', icon: 'fileText' },
  cron:      { title: 'Cron Jobs',  sub: 'Schedule wakeups and recurring agent runs.',             icon: 'loader' },
  skills:    { title: 'Skills',     sub: 'Manage skill availability and API key injection.',       icon: 'zap' },
  nodes:     { title: 'Nodes',      sub: 'Paired devices, capabilities, and command exposure.',    icon: 'monitor' },
  config:    { title: 'Config',     sub: 'Edit configuration safely.',                             icon: 'settings' },
  debug:     { title: 'Debug',      sub: 'Gateway snapshots, events, and manual RPC calls.',       icon: 'bug' },
  logs:      { title: 'Logs',       sub: 'Live tail of the gateway logs.',                         icon: 'scrollText' },
};
