export type Intent =
  | "status"
  | "components"
  | "incidents"
  | "incident_detail"
  | "uptime"
  | "help";

export interface DetectedIntent {
  intent: Intent;
}

const INCIDENT_DETAIL_KEYWORDS = [
  "latest incident", "last incident", "most recent incident", "incident detail",
];
const INCIDENT_KEYWORDS = [
  "incident", "what happened", "recent issues", "history", "past issues",
];
const UPTIME_KEYWORDS = [
  "uptime", "reliable", "availability", "sla", "percentage",
];
const COMPONENT_KEYWORDS = [
  "api", "claude code", "platform", "claude.ai", "government", "component",
];
const STATUS_KEYWORDS = [
  "status", "down", "outage", "operational", "is claude", "is it up", "working",
];

function matches(msg: string, keywords: string[]): boolean {
  return keywords.some((k) => msg.includes(k));
}

export function detectIntent(message: string): DetectedIntent {
  const msg = ` ${message.toLowerCase()} `;

  if (matches(msg, INCIDENT_DETAIL_KEYWORDS)) return { intent: "incident_detail" };
  if (matches(msg, INCIDENT_KEYWORDS))        return { intent: "incidents" };
  if (matches(msg, UPTIME_KEYWORDS))          return { intent: "uptime" };
  if (matches(msg, COMPONENT_KEYWORDS))       return { intent: "components" };
  if (matches(msg, STATUS_KEYWORDS))          return { intent: "status" };
  return { intent: "help" };
}
