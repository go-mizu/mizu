import { describe, it, expect } from "vitest";
import { formatStandings, formatFixtures, formatTeamInfo, formatHelp } from "./format";
import type { Standing, Fixture, TeamInfo } from "./data";

const standings: Standing[] = [
  { rank: 1, team: "Liverpool", played: 29, won: 21, draw: 5, lost: 3,  gd: 42, points: 68 },
  { rank: 2, team: "Arsenal",   played: 29, won: 20, draw: 4, lost: 5,  gd: 28, points: 64 },
  { rank: 3, team: "Chelsea",   played: 29, won: 17, draw: 5, lost: 7,  gd: 22, points: 56 },
];

const fixtures: Fixture[] = [
  { date: "2026-03-22", home: "Arsenal",   away: "Chelsea",      competition: "PL" },
  { date: "2026-04-05", home: "Liverpool", away: "Nottm Forest", competition: "PL" },
];

const team: TeamInfo = {
  name: "Arsenal",
  aliases: ["arsenal"],
  competition: "PL",
  stadium: "Emirates Stadium",
  manager: "Mikel Arteta",
};

describe("formatStandings", () => {
  it("includes competition name in heading", () => {
    expect(formatStandings("PL", standings)).toContain("Premier League");
  });
  it("renders markdown table header", () => {
    const out = formatStandings("PL", standings);
    expect(out).toContain("| # |");
    expect(out).toContain("| Pts |");
  });
  it("includes team name and points", () => {
    const out = formatStandings("PL", standings);
    expect(out).toContain("Liverpool");
    expect(out).toContain("68");
  });
  it("bolds the top team", () => {
    const out = formatStandings("PL", standings);
    expect(out).toContain("**Liverpool**");
    expect(out).toContain("**68**");
  });
});

describe("formatFixtures", () => {
  it("includes both team names", () => {
    const out = formatFixtures("PL", fixtures);
    expect(out).toContain("Arsenal");
    expect(out).toContain("Chelsea");
  });
  it("includes formatted date", () => {
    expect(formatFixtures("PL", fixtures)).toContain("Mar 22");
  });
  it("works without competition code", () => {
    const out = formatFixtures(undefined, fixtures);
    expect(out).toContain("Arsenal");
    expect(out).toContain("Chelsea");
  });
  it("handles empty fixtures", () => {
    expect(formatFixtures("CL", [])).toContain("No upcoming fixtures");
  });
  it("renders markdown table header", () => {
    const out = formatFixtures("PL", fixtures);
    expect(out).toContain("| Date |");
    expect(out).toContain("| Home |");
    expect(out).toContain("| Away |");
  });
});

describe("formatTeamInfo", () => {
  it("includes team name as heading", () => {
    expect(formatTeamInfo(team)).toContain("## ⚽ Arsenal");
  });
  it("includes stadium", () => {
    expect(formatTeamInfo(team)).toContain("Emirates Stadium");
  });
  it("includes manager", () => {
    expect(formatTeamInfo(team)).toContain("Mikel Arteta");
  });
  it("includes competition", () => {
    expect(formatTeamInfo(team)).toContain("Premier League");
  });
});

describe("formatHelp", () => {
  it("mentions table", () => { expect(formatHelp()).toContain("table"); });
  it("mentions fixture", () => { expect(formatHelp()).toContain("fixture"); });
  it("has Scout heading", () => {
    expect(formatHelp()).toContain("Scout");
    expect(formatHelp()).toContain("##");
  });
  it("uses markdown bold", () => { expect(formatHelp()).toContain("**"); });
});
