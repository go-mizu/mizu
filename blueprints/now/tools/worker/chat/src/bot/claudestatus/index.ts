import { registerBot } from "../registry";
import { detectIntent } from "./intent";
import {
  formatStatus, formatComponents, formatIncidents,
  formatIncidentDetail, formatUptime, formatHelp,
} from "./format";
import { fetchSummary, fetchIncidents } from "./fetch";

async function claudestatusReply(msg: string, db: D1Database): Promise<string> {
  const { intent } = detectIntent(msg);

  switch (intent) {
    case "status":
    case "components":
    case "uptime": {
      const result = await fetchSummary(db);
      if (!result) return "⚠ Could not reach status.claude.com. Try again in a moment.";
      const prefix = result.stale ? "⚠ Data may be stale.\n\n" : "";
      if (intent === "status")     return prefix + formatStatus(result.data);
      if (intent === "components") return prefix + formatComponents(result.data);
      return prefix + formatUptime(result.data);
    }

    case "incidents":
    case "incident_detail": {
      const result = await fetchIncidents(db);
      if (!result) return "⚠ Could not reach status.claude.com. Try again in a moment.";
      const prefix = result.stale ? "⚠ Data may be stale.\n\n" : "";
      if (intent === "incident_detail") return prefix + formatIncidentDetail(result.data.incidents);
      return prefix + formatIncidents(result.data.incidents);
    }

    default:
      return formatHelp();
  }
}

registerBot({
  actor: "a/claudestatus",
  profile: {
    bio: "ClaudeStatus monitors Anthropic's services in real time. Ask about current status, recent incidents, component health, or uptime.",
    examples: [
      "Is Claude down?",
      "Any recent incidents?",
      "Latest incident details",
      "Is the API up?",
      "What's the uptime?",
    ],
  },
  reply: (msg, db) => claudestatusReply(msg, db),
});
