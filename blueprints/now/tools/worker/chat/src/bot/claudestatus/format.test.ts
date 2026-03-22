import { describe, it, expect } from "vitest";
import {
  formatStatus, formatComponents, formatIncidents,
  formatIncidentDetail, formatUptime, formatHelp,
} from "./format";

const okSummary = {
  status: { indicator: "none", description: "All Systems Operational" },
  components: [
    { id: "1", name: "claude.ai",            status: "operational", updated_at: "2026-03-17T15:00:00Z", group_id: null, group: false },
    { id: "2", name: "Claude API",            status: "operational", updated_at: "2026-03-17T15:00:00Z", group_id: null, group: false },
    { id: "3", name: "Claude Code",           status: "operational", updated_at: "2026-03-17T15:00:00Z", group_id: null, group: false },
    { id: "4", name: "Claude API sub",        status: "operational", updated_at: "2026-03-17T15:00:00Z", group_id: "2", group: false },
  ],
};

const degradedSummary = {
  status: { indicator: "minor", description: "Partial System Outage" },
  components: [
    { id: "1", name: "claude.ai",  status: "degraded_performance", updated_at: "2026-03-17T15:00:00Z", group_id: null, group: false },
    { id: "2", name: "Claude API", status: "operational",          updated_at: "2026-03-17T15:00:00Z", group_id: null, group: false },
  ],
};

const incidents = [
  {
    id: "abc1", name: "Elevated errors on Claude Sonnet 4.6",
    status: "resolved", impact: "minor",
    created_at: "2026-03-17T14:07:53Z", resolved_at: "2026-03-17T15:45:20Z",
    incident_updates: [
      { status: "investigating", body: "We are looking into this.", created_at: "2026-03-17T14:07:53Z" },
      { status: "resolved",      body: "Issue has been resolved.",  created_at: "2026-03-17T15:45:20Z" },
    ],
  },
];

describe("formatStatus", () => {
  it("shows ✅ for indicator=none", () => {
    expect(formatStatus(okSummary)).toContain("✅");
  });
  it("shows ⚠ for indicator=minor", () => {
    expect(formatStatus(degradedSummary)).toContain("⚠");
  });
  it("includes description", () => {
    expect(formatStatus(okSummary)).toContain("All Systems Operational");
  });
  it("lists top-level component names", () => {
    const out = formatStatus(okSummary);
    expect(out).toContain("claude.ai");
    expect(out).toContain("Claude API");
  });
  it("does NOT include sub-components (group_id != null)", () => {
    expect(formatStatus(okSummary)).not.toContain("Claude API sub");
  });
});

describe("formatComponents", () => {
  it("renders a markdown table", () => {
    const out = formatComponents(okSummary);
    expect(out).toContain("| Component |");
    expect(out).toContain("| Status |");
  });
  it("excludes sub-components", () => {
    expect(formatComponents(okSummary)).not.toContain("Claude API sub");
  });
  it("includes top-level components", () => {
    expect(formatComponents(okSummary)).toContain("claude.ai");
  });
});

describe("formatIncidents", () => {
  it("includes incident name", () => {
    expect(formatIncidents(incidents)).toContain("Elevated errors on Claude Sonnet 4.6");
  });
  it("shows impact", () => {
    expect(formatIncidents(incidents)).toContain("minor");
  });
  it("shows resolved status", () => {
    expect(formatIncidents(incidents)).toContain("resolved");
  });
  it("shows 'No recent incidents' when list is empty", () => {
    expect(formatIncidents([])).toContain("No recent incidents");
  });
});

describe("formatIncidentDetail", () => {
  it("shows incident name as heading", () => {
    expect(formatIncidentDetail(incidents)).toContain("Elevated errors on Claude Sonnet 4.6");
  });
  it("shows each update body", () => {
    const out = formatIncidentDetail(incidents);
    expect(out).toContain("We are looking into this.");
    expect(out).toContain("Issue has been resolved.");
  });
  it("shows update statuses", () => {
    const out = formatIncidentDetail(incidents);
    expect(out).toContain("investigating");
    expect(out).toContain("resolved");
  });
  it("handles empty incident list gracefully", () => {
    expect(formatIncidentDetail([])).toContain("No incidents");
  });
});

describe("formatUptime", () => {
  it("explains API limitation", () => {
    expect(formatUptime(okSummary)).toContain("status.claude.com");
  });
  it("shows component statuses", () => {
    expect(formatUptime(okSummary)).toContain("claude.ai");
  });
  it("uses ✅ for operational", () => {
    expect(formatUptime(okSummary)).toContain("✅");
  });
});

describe("formatHelp", () => {
  it("mentions claudestatus", () => {
    expect(formatHelp()).toContain("ClaudeStatus");
  });
  it("shows example questions", () => {
    const out = formatHelp();
    expect(out).toContain("Is Claude down");
    expect(out).toContain("incident");
  });
});
