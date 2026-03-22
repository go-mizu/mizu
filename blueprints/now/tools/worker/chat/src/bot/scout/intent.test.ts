import { describe, it, expect } from "vitest";
import { detectIntent } from "./intent";

describe("detectIntent", () => {
  it("detects standings intent from 'table'", () => {
    const r = detectIntent("show me the Premier League table");
    expect(r.intent).toBe("standings");
    expect(r.competition).toBe("PL");
  });
  it("detects standings from 'standings'", () => {
    const r = detectIntent("Champions League standings");
    expect(r.intent).toBe("standings");
    expect(r.competition).toBe("CL");
  });
  it("detects fixtures from 'next match' with team", () => {
    const r = detectIntent("when is Arsenal's next match?");
    expect(r.intent).toBe("fixtures");
    expect(r.teamName).toBe("arsenal");
  });
  it("detects fixtures from 'fixtures' keyword", () => {
    const r = detectIntent("Bundesliga fixtures");
    expect(r.intent).toBe("fixtures");
    expect(r.competition).toBe("BL1");
  });
  it("detects team intent from 'squad'", () => {
    const r = detectIntent("tell me about Barcelona squad");
    expect(r.intent).toBe("team");
    expect(r.teamName).toBe("barcelona");
  });
  it("detects team intent from 'manager'", () => {
    const r = detectIntent("who is the manager of Liverpool?");
    expect(r.intent).toBe("team");
    expect(r.teamName).toBe("liverpool");
  });
  it("falls back to help", () => {
    const r = detectIntent("hello, who are you?");
    expect(r.intent).toBe("help");
  });
  it("detects La Liga", () => {
    const r = detectIntent("la liga table");
    expect(r.competition).toBe("PD");
  });
  it("detects Serie A", () => {
    const r = detectIntent("Serie A standings");
    expect(r.competition).toBe("SA");
  });
  it("detects Ligue 1", () => {
    const r = detectIntent("ligue 1 table");
    expect(r.competition).toBe("FL1");
  });
  it("detects Europa League fixtures", () => {
    const r = detectIntent("europa league fixtures");
    expect(r.competition).toBe("EL");
    expect(r.intent).toBe("fixtures");
  });
  it("defaults to team intent when only team mentioned", () => {
    const r = detectIntent("Barcelona");
    expect(r.intent).toBe("team");
    expect(r.teamName).toBe("barcelona");
  });
  it("defaults to standings when only competition mentioned", () => {
    const r = detectIntent("bundesliga");
    expect(r.intent).toBe("standings");
    expect(r.competition).toBe("BL1");
  });
});
