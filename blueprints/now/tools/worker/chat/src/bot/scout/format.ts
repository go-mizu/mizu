import type { Standing, Fixture, TeamInfo, CompetitionCode } from "./data";
import { COMPETITION_NAMES } from "./data";

function fmtDate(iso: string): string {
  const [, m, d] = iso.split("-");
  const month = ["Jan","Feb","Mar","Apr","May","Jun","Jul","Aug","Sep","Oct","Nov","Dec"][
    parseInt(m, 10) - 1
  ];
  return `${month} ${parseInt(d, 10)}`;
}

export function formatStandings(code: CompetitionCode, standings: Standing[]): string {
  const name = COMPETITION_NAMES[code];
  const header = `## 🏆 ${name} Table\n`;
  const tableHead = `| # | Team | P | W | D | L | GD | Pts |\n|---|------|---|---|---|---|----|-----|`;
  const rows = standings
    .slice(0, 10)
    .map(s => {
      const gd = s.gd >= 0 ? `+${s.gd}` : `${s.gd}`;
      const isLeader = s.rank === 1;
      const teamStr = isLeader ? `**${s.team}**`   : s.team;
      const ptsStr  = isLeader ? `**${s.points}**` : `${s.points}`;
      return `| ${s.rank} | ${teamStr} | ${s.played} | ${s.won} | ${s.draw} | ${s.lost} | ${gd} | ${ptsStr} |`;
    })
    .join("\n");
  return `${header}${tableHead}\n${rows}`;
}

export function formatFixtures(code: CompetitionCode | undefined, fixtures: Fixture[]): string {
  if (fixtures.length === 0) {
    const comp = code ? COMPETITION_NAMES[code] : "this competition";
    return `> 📅 No upcoming fixtures found for **${comp}**.`;
  }
  const title = code
    ? `## 📅 Upcoming ${COMPETITION_NAMES[code]} Fixtures\n`
    : `## 📅 Upcoming Fixtures\n`;
  const tableHead = `| Date | Home | Away |\n|------|------|------|`;
  const rows = fixtures
    .slice(0, 5)
    .map(f => `| ${fmtDate(f.date)} | ${f.home} | ${f.away} |`)
    .join("\n");
  return `${title}${tableHead}\n${rows}`;
}

export function formatTeamInfo(team: TeamInfo): string {
  return [
    `## ⚽ ${team.name}`,
    ``,
    `| | |`,
    `|---|---|`,
    `| 🏟 Stadium | ${team.stadium} |`,
    `| 👔 Manager | ${team.manager} |`,
    `| 🏆 League  | ${COMPETITION_NAMES[team.competition]} |`,
  ].join("\n");
}

export function formatHelp(): string {
  return [
    `## 👋 I'm Scout — your football companion!`,
    ``,
    `Ask me anything:`,
    ``,
    `- **"Premier League table"** — standings`,
    `- **"Champions League fixtures"** — upcoming matches`,
    `- **"Tell me about Barcelona"** — club info`,
    `- **"When is Arsenal's next match?"** — team fixtures`,
    `- **"Serie A standings"** — any league table`,
    ``,
    `Leagues covered: **PL** · **La Liga** · **Bundesliga** · **Serie A** · **Ligue 1** · **UCL** · **UEL**`,
  ].join("\n");
}
