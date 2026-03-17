import { describe, it, expect } from "vitest";
import { scoutReply } from "./scout";

describe("scoutReply", () => {
  it("returns standings table for PL query", () => {
    const out = scoutReply("show me the Premier League table");
    expect(out).toContain("Premier League Table");
    expect(out).toContain("Pts");
  });
  it("returns UCL fixtures", () => {
    const out = scoutReply("Champions League fixtures");
    expect(out).toContain("Champions League");
    expect(out).toContain("Home");
  });
  it("returns team info for Barcelona", () => {
    const out = scoutReply("tell me about Barcelona");
    expect(out).toContain("Barcelona");
    expect(out).toContain("Manager");
  });
  it("returns team fixtures for Arsenal next match", () => {
    const out = scoutReply("when is Arsenal's next match?");
    expect(out).toContain("Arsenal");
    expect(out).toContain("Home");
  });
  it("returns help for unknown input", () => {
    const out = scoutReply("hello");
    expect(out).toContain("Scout");
    expect(out).toContain("table");
  });
  it("handles case-insensitive team", () => {
    const out = scoutReply("LIVERPOOL squad");
    expect(out).toContain("Liverpool");
  });
  it("defaults to PL standings when no competition specified", () => {
    const out = scoutReply("show me the table");
    expect(out).toContain("Premier League");
  });
});
