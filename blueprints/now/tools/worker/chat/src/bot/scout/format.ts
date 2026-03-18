import type { Standing, Fixture, TeamInfo, CompetitionCode } from "./data";
import { COMPETITION_NAMES, STANDINGS } from "./data";
import { SITE_NAME } from "../../constants";

// ── helpers ──────────────────────────────────────────────────────────────────

function fmtDate(iso: string): string {
  const [, m, d] = iso.split("-");
  const months = ["Jan","Feb","Mar","Apr","May","Jun","Jul","Aug","Sep","Oct","Nov","Dec"];
  return `${months[parseInt(m, 10) - 1]} ${parseInt(d, 10)}`;
}

const COMP_BADGE: Record<CompetitionCode, string> = {
  PL: "🏴󠁧󠁢󠁥󠁮󠁧󠁿 PL", PD: "🇪🇸 LaLiga", BL1: "🇩🇪 Bundesliga",
  SA: "🇮🇹 Serie A", FL1: "🇫🇷 Ligue 1", CL: "⭐ UCL", EL: "🟠 UEL",
};

/** UCL/UEL/relegation zone sizes per domestic competition. */
const ZONES: Record<string, { ucl: number; uel: number; rel: number }> = {
  PL:  { ucl: 4, uel: 2, rel: 3 },
  PD:  { ucl: 4, uel: 2, rel: 3 },
  BL1: { ucl: 4, uel: 2, rel: 3 },
  SA:  { ucl: 4, uel: 2, rel: 3 },
  FL1: { ucl: 4, uel: 2, rel: 3 },
};

function rankBadge(rank: number): string {
  if (rank === 1) return "🥇";
  if (rank === 2) return "🥈";
  if (rank === 3) return "🥉";
  return `${rank}. `;
}

// ── formatStandings ──────────────────────────────────────────────────────────

export function formatStandings(code: CompetitionCode, standings: Standing[]): string {
  const name = COMPETITION_NAMES[code];
  const zone = ZONES[code];

  const header = `## 🏆 ${name} Table`;
  const tableHead = `| | Team | P | W | D | L | GD | Pts |\n|---|------|---|---|---|---|----|-----|`;

  const rows = standings.slice(0, 10).map((s, i) => {
    const gd = s.gd >= 0 ? `+${s.gd}` : `${s.gd}`;
    const badge = rankBadge(s.rank);

    // bold top-3 team name & points
    const isTop3 = s.rank <= 3;
    const teamStr = isTop3 ? `**${s.team}**` : s.team;
    const ptsStr  = isTop3 ? `**${s.points}**` : `${s.points}`;

    // zone separator rows (blank separator line before zone boundary)
    const prevRank = standings[i - 1]?.rank;
    let sep = "";
    if (zone) {
      if (s.rank === zone.ucl + 1 && prevRank === zone.ucl) sep = `| | *— UCL zone above —* | | | | | | |\n`;
      if (s.rank === standings.length - zone.rel + 1) sep = `| | *— Relegation zone below —* | | | | | | |\n`;
    }

    return `${sep}| ${badge} | ${teamStr} | ${s.played} | ${s.won} | ${s.draw} | ${s.lost} | ${gd} | ${ptsStr} |`;
  }).join("\n");

  const legend = zone
    ? `\n> 🔵 Top ${zone.ucl} → Champions League &nbsp;·&nbsp; 🟠 Next ${zone.uel} → Europa League &nbsp;·&nbsp; 🔴 Bottom ${zone.rel} → Relegation`
    : "";

  return [header, "", tableHead, rows, legend].join("\n");
}

// ── formatFixtures ───────────────────────────────────────────────────────────

export function formatFixtures(
  code: CompetitionCode | undefined,
  fixtures: Fixture[]
): string {
  if (fixtures.length === 0) {
    const comp = code ? COMPETITION_NAMES[code] : "this competition";
    return `> 📅 No upcoming fixtures found for **${comp}**.`;
  }

  const title = code
    ? `## 📅 ${COMP_BADGE[code]} Upcoming Fixtures`
    : `## 📅 Upcoming Fixtures`;

  // Cross-competition fixture list (team view) → show competition column
  const showComp = code === undefined;
  const tableHead = showComp
    ? `| Date | Competition | Home | Away |\n|------|-------------|------|------|`
    : `| Date | Home | Away |\n|------|------|------|`;

  const rows = fixtures.slice(0, 5).map((f, i) => {
    const date = i === 0 ? `**${fmtDate(f.date)}**` : fmtDate(f.date);
    const next = i === 0 ? " 🔜" : "";
    return showComp
      ? `| ${date}${next} | ${COMP_BADGE[f.competition]} | ${f.home} | ${f.away} |`
      : `| ${date}${next} | ${f.home} | ${f.away} |`;
  }).join("\n");

  return [title, "", tableHead, rows].join("\n");
}

// ── formatTeamInfo ───────────────────────────────────────────────────────────

export function formatTeamInfo(team: TeamInfo): string {
  // Look up current league position
  const leagueTable = STANDINGS[team.competition];
  const entry = leagueTable?.find(s => s.team === team.name);
  const posLine = entry
    ? `| 📊 Position | **${entry.rank}${ordinal(entry.rank)}** of ${leagueTable.length} · ${entry.points} pts |`
    : "";

  const lines = [
    `## ⚽ ${team.name}`,
    ``,
    `| | |`,
    `|---|---|`,
    posLine,
    `| 🏟 Stadium  | ${team.stadium} |`,
    `| 👔 Manager  | ${team.manager} |`,
    `| 🏆 League   | ${COMPETITION_NAMES[team.competition]} |`,
  ].filter(Boolean);

  return lines.join("\n");
}

function ordinal(n: number): string {
  const s = ["th","st","nd","rd"];
  const v = n % 100;
  return s[(v - 20) % 10] || s[v] || s[0];
}

// ── formatHelp ───────────────────────────────────────────────────────────────

export function formatHelp(): string {
  return [
    `## 👋 I'm Scout`,
    `*Your football companion on ${SITE_NAME}*`,
    ``,
    `### What I know`,
    `| Competition | Coverage |`,
    `|-------------|----------|`,
    `| 🏴󠁧󠁢󠁥󠁮󠁧󠁿 Premier League | Table · Fixtures · Clubs |`,
    `| 🇪🇸 La Liga | Table · Fixtures · Clubs |`,
    `| 🇩🇪 Bundesliga | Table · Fixtures · Clubs |`,
    `| 🇮🇹 Serie A | Table · Fixtures · Clubs |`,
    `| 🇫🇷 Ligue 1 | Table · Fixtures · Clubs |`,
    `| ⭐ Champions League | Table · Fixtures |`,
    `| 🟠 Europa League | Table · Fixtures |`,
    ``,
    `### Try asking`,
    `- **"Premier League table"** — full standings with zones`,
    `- **"When is Arsenal's next match?"** — upcoming fixtures`,
    `- **"Tell me about Barcelona"** — club info + league position`,
    `- **"UCL fixtures"** — Champions League schedule`,
    `- **"Who is the Bundesliga leader?"** — quick standings check`,
  ].join("\n");
}
