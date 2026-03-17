// --- Shared types ---

interface StatusObj { indicator: string; description: string; }
interface ComponentObj {
  id: string; name: string; status: string;
  updated_at: string; group_id: string | null; group: boolean;
}
export interface SummaryResponse { status: StatusObj; components: ComponentObj[]; }

interface IncidentUpdate { status: string; body: string; created_at: string; }
export interface Incident {
  id: string; name: string; status: string; impact: string;
  created_at: string; resolved_at: string | null;
  incident_updates: IncidentUpdate[];
}
export interface IncidentsResponse { incidents: Incident[]; }

// --- Helpers ---

function indicatorEmoji(indicator: string): string {
  if (indicator === "none") return "✅";
  if (indicator === "minor") return "⚠";
  if (indicator === "major" || indicator === "critical") return "🔴";
  return "⚠";
}

function componentEmoji(status: string): string {
  return status === "operational" ? "✅" : "⚠";
}

function topLevel(components: ComponentObj[]): ComponentObj[] {
  return components.filter((c) => c.group_id === null && !c.group);
}

function fmtDate(iso: string): string {
  return new Date(iso).toLocaleString("en-US", {
    month: "short", day: "numeric", hour: "2-digit", minute: "2-digit",
    timeZone: "UTC", hour12: false,
  }) + " UTC";
}

// --- Formatters ---

export function formatStatus(summary: SummaryResponse): string {
  const emoji = indicatorEmoji(summary.status.indicator);
  const lines = [
    `## ${emoji} ${summary.status.description}`,
    ``,
    ...topLevel(summary.components).map(
      (c) => `- ${componentEmoji(c.status)} **${c.name}** — ${c.status.replace(/_/g, " ")}`
    ),
  ];
  return lines.join("\n");
}

export function formatComponents(summary: SummaryResponse): string {
  const rows = topLevel(summary.components).map(
    (c) => `| ${c.name} | ${componentEmoji(c.status)} ${c.status.replace(/_/g, " ")} | ${fmtDate(c.updated_at)} |`
  );
  return [
    `## 📋 Component Status`,
    ``,
    `| Component | Status | Updated |`,
    `|---|---|---|`,
    ...rows,
  ].join("\n");
}

export function formatIncidents(incidents: Incident[], limit = 5): string {
  if (incidents.length === 0) {
    return `> ✅ No recent incidents found.`;
  }
  const lines = [`## ⚠ Recent Incidents\n`];
  for (const inc of incidents.slice(0, limit)) {
    const date = fmtDate(inc.created_at);
    const resolved = inc.resolved_at ? `resolved ${fmtDate(inc.resolved_at)}` : "ongoing";
    lines.push(`- **${inc.name}** — ${inc.impact} · ${date} · ${resolved}`);
  }
  return lines.join("\n");
}

export function formatIncidentDetail(incidents: Incident[]): string {
  if (incidents.length === 0) {
    return `> ✅ No incidents on record.`;
  }
  const inc = incidents[0];
  const lines = [
    `## 🔍 ${inc.name}`,
    ``,
    `**Impact:** ${inc.impact} · **Status:** ${inc.status}`,
    `**Started:** ${fmtDate(inc.created_at)}`,
    inc.resolved_at ? `**Resolved:** ${fmtDate(inc.resolved_at)}` : `**Status:** ongoing`,
    ``,
    `### Timeline`,
  ];
  for (const update of [...inc.incident_updates].reverse()) {
    lines.push(`- **${update.status}** (${fmtDate(update.created_at)}): ${update.body}`);
  }
  return lines.join("\n");
}

export function formatUptime(summary: SummaryResponse): string {
  const componentList = topLevel(summary.components)
    .map((c) => `${c.name} ${componentEmoji(c.status)}`)
    .join(" · ");
  return [
    `## ℹ Uptime`,
    ``,
    `Uptime percentages are only shown on [status.claude.com](https://status.claude.com) and aren't available via the JSON API.`,
    ``,
    `**Current component status:** ${componentList}`,
  ].join("\n");
}

export function formatHelp(): string {
  return [
    `## 📡 ClaudeStatus — your Anthropic service monitor`,
    ``,
    `Ask me anything about Claude's service health:`,
    ``,
    `- **"Is Claude down?"** — overall status`,
    `- **"Is the API up?"** — per-component status`,
    `- **"Any recent incidents?"** — incident list`,
    `- **"Latest incident details"** — full incident timeline`,
    `- **"What's the uptime?"** — availability info`,
  ].join("\n");
}
