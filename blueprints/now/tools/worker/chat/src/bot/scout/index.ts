import { detectIntent } from "./intent";
import { STANDINGS, findTeam, fixturesByCompetition, fixturesByTeam } from "./data";
import { formatStandings, formatFixtures, formatTeamInfo, formatHelp } from "./format";
import { registerBot } from "../registry";

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

registerBot({
  actor: "a/scout",
  profile: {
    bio: "Scout is your football companion. Ask about standings, fixtures, and club info across 7 major leagues.",
    examples: [
      "Premier League table",
      "When is Arsenal's next match?",
      "Tell me about Barcelona",
      "Champions League fixtures",
    ],
  },
  reply: (msg) => scoutReply(msg),
  // db is part of BotDef interface but scout is synchronous and needs no DB access
});
