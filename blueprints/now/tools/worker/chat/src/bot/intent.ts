import type { CompetitionCode } from "./data";

export type Intent = "standings" | "fixtures" | "team" | "help";

export interface DetectedIntent {
  intent: Intent;
  competition?: CompetitionCode;
  teamName?: string;
}

const COMPETITION_KEYWORDS: Array<[string[], CompetitionCode]> = [
  [["premier league", "epl", " pl ", "prem"],          "PL"],
  [["la liga", "laliga", "primera"],                   "PD"],
  [["bundesliga", "buli"],                             "BL1"],
  [["serie a", "calcio"],                              "SA"],
  [["ligue 1", "ligue1", "ligue un"],                  "FL1"],
  [["champions league", "ucl", " cl "],               "CL"],
  [["europa league", "uel", " el "],                  "EL"],
];

const STANDINGS_KEYWORDS = ["table", "standings", "leaderboard", "ranking", "points", "top of"];
const FIXTURE_KEYWORDS   = ["next match", "next game", "fixture", "upcoming", "when do", "when is", "when will", "schedule", "kickoff", "kick off"];
const TEAM_KEYWORDS      = ["squad", "team info", "manager", "coach", "stadium", "ground", "tell me about", "who plays for", "players", "info about"];

const KNOWN_TEAM_ALIASES = [
  "paris saint-germain", "manchester united", "manchester city",
  "atletico madrid", "real madrid", "borussia dortmund", "bayer leverkusen",
  "eintracht frankfurt", "rb leipzig", "nottingham forest", "nottm forest",
  "real sociedad", "real betis", "athletic club", "athletic bilbao",
  "ac milan", "inter milan", "aston villa",
  "man united", "man utd", "man city",
  "liverpool", "arsenal", "chelsea", "tottenham", "spurs",
  "newcastle", "fulham", "forest", "barcelona", "barca", "fcb",
  "sevilla", "betis", "villarreal", "bilbao", "la real",
  "madrid", "atletico", "atleti",
  "bayern munich", "fcbayern", "dortmund", "bvb", "frankfurt",
  "leipzig", "leverkusen", "freiburg", "wolfsburg", "napoli",
  "inter", "atalanta", "juventus", "juve", "lazio", "milan", "roma",
  "fiorentina", "viola", "psg", "monaco", "lille", "lyon",
  "marseille", "nice", "rennes", "lens", "ajax", "porto", "fenerbahce",
];

export function detectIntent(message: string): DetectedIntent {
  const msg = ` ${message.toLowerCase()} `;

  let competition: CompetitionCode | undefined;
  for (const [keywords, code] of COMPETITION_KEYWORDS) {
    if (keywords.some(k => msg.includes(k))) {
      competition = code;
      break;
    }
  }

  let teamName: string | undefined;
  for (const alias of KNOWN_TEAM_ALIASES) {
    if (msg.includes(alias)) {
      teamName = alias;
      break;
    }
  }

  if (STANDINGS_KEYWORDS.some(k => msg.includes(k))) {
    return { intent: "standings", competition, teamName };
  }
  if (FIXTURE_KEYWORDS.some(k => msg.includes(k))) {
    return { intent: "fixtures", competition, teamName };
  }
  if (TEAM_KEYWORDS.some(k => msg.includes(k))) {
    return { intent: "team", competition, teamName };
  }
  if (teamName) {
    return { intent: "team", competition, teamName };
  }
  if (competition) {
    return { intent: "standings", competition };
  }
  return { intent: "help" };
}
