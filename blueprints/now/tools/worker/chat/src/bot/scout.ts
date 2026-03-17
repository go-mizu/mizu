import { detectIntent } from "./intent";
import { STANDINGS, findTeam, fixturesByCompetition, fixturesByTeam } from "./data";
import { formatStandings, formatFixtures, formatTeamInfo, formatHelp } from "./format";

export function scoutReply(message: string): string {
  const { intent, competition, teamName } = detectIntent(message);

  switch (intent) {
    case "standings": {
      const code = competition ?? "PL";
      return formatStandings(code, STANDINGS[code]);
    }
    case "fixtures": {
      if (teamName) {
        const team = findTeam(teamName);
        return formatFixtures(undefined, fixturesByTeam(team?.name ?? teamName));
      }
      const code = competition ?? "PL";
      return formatFixtures(code, fixturesByCompetition(code));
    }
    case "team": {
      if (teamName) {
        const team = findTeam(teamName);
        if (team) return formatTeamInfo(team);
        return `❓ I don't have info on "${teamName}". Try another team name.`;
      }
      return formatHelp();
    }
    default:
      return formatHelp();
  }
}
