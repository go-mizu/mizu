// src/bot/data.ts

export type CompetitionCode =
  | "PL" | "PD" | "BL1" | "SA" | "FL1" | "CL" | "EL";

export interface Standing {
  rank: number;
  team: string;
  played: number;
  won: number;
  draw: number;
  lost: number;
  gd: number;
  points: number;
}

export interface Fixture {
  date: string;
  home: string;
  away: string;
  competition: CompetitionCode;
}

export interface TeamInfo {
  name: string;
  aliases: string[];
  competition: CompetitionCode;
  stadium: string;
  manager: string;
}

export const COMPETITION_NAMES: Record<CompetitionCode, string> = {
  PL:  "Premier League",
  PD:  "La Liga",
  BL1: "Bundesliga",
  SA:  "Serie A",
  FL1: "Ligue 1",
  CL:  "Champions League",
  EL:  "Europa League",
};

export const STANDINGS: Record<CompetitionCode, Standing[]> = {
  PL: [
    { rank: 1,  team: "Liverpool",            played: 29, won: 21, draw: 5, lost: 3,  gd: 42,  points: 68 },
    { rank: 2,  team: "Arsenal",              played: 29, won: 20, draw: 4, lost: 5,  gd: 28,  points: 64 },
    { rank: 3,  team: "Chelsea",              played: 29, won: 17, draw: 5, lost: 7,  gd: 22,  points: 56 },
    { rank: 4,  team: "Nottm Forest",         played: 29, won: 16, draw: 6, lost: 7,  gd: 11,  points: 54 },
    { rank: 5,  team: "Manchester City",      played: 29, won: 15, draw: 6, lost: 8,  gd: 14,  points: 51 },
    { rank: 6,  team: "Newcastle",            played: 29, won: 14, draw: 8, lost: 7,  gd:  9,  points: 50 },
    { rank: 7,  team: "Aston Villa",          played: 29, won: 13, draw: 7, lost: 9,  gd:  5,  points: 46 },
    { rank: 8,  team: "Manchester United",    played: 29, won: 11, draw: 6, lost: 12, gd: -8,  points: 39 },
    { rank: 9,  team: "Tottenham",            played: 29, won: 10, draw: 6, lost: 13, gd: -5,  points: 36 },
    { rank: 10, team: "Fulham",               played: 29, won: 10, draw: 5, lost: 14, gd: -3,  points: 35 },
  ],
  PD: [
    { rank: 1,  team: "Barcelona",            played: 28, won: 20, draw: 4, lost: 4, gd: 38,  points: 64 },
    { rank: 2,  team: "Real Madrid",          played: 28, won: 19, draw: 4, lost: 5, gd: 31,  points: 61 },
    { rank: 3,  team: "Atletico Madrid",      played: 28, won: 18, draw: 5, lost: 5, gd: 22,  points: 59 },
    { rank: 4,  team: "Athletic Club",        played: 28, won: 14, draw: 7, lost: 7, gd: 14,  points: 49 },
    { rank: 5,  team: "Villarreal",           played: 28, won: 13, draw: 6, lost: 9, gd:  8,  points: 45 },
    { rank: 6,  team: "Real Sociedad",        played: 28, won: 12, draw: 7, lost: 9, gd:  5,  points: 43 },
    { rank: 7,  team: "Betis",                played: 28, won: 11, draw: 8, lost: 9, gd:  2,  points: 41 },
    { rank: 8,  team: "Sevilla",              played: 28, won: 10, draw: 6, lost: 12, gd: -4, points: 36 },
  ],
  BL1: [
    { rank: 1,  team: "Bayern Munich",        played: 26, won: 18, draw: 4, lost: 4, gd: 44,  points: 58 },
    { rank: 2,  team: "Bayer Leverkusen",     played: 26, won: 17, draw: 5, lost: 4, gd: 31,  points: 56 },
    { rank: 3,  team: "Borussia Dortmund",    played: 26, won: 14, draw: 6, lost: 6, gd: 18,  points: 48 },
    { rank: 4,  team: "Eintracht Frankfurt",  played: 26, won: 13, draw: 5, lost: 8, gd:  9,  points: 44 },
    { rank: 5,  team: "RB Leipzig",           played: 26, won: 12, draw: 6, lost: 8, gd:  6,  points: 42 },
    { rank: 6,  team: "Freiburg",             played: 26, won: 10, draw: 8, lost: 8, gd:  1,  points: 38 },
    { rank: 7,  team: "Wolfsburg",            played: 26, won: 9,  draw: 6, lost: 11, gd: -5, points: 33 },
    { rank: 8,  team: "Borussia M'gladbach",  played: 26, won: 8,  draw: 8, lost: 10, gd: -3, points: 32 },
  ],
  SA: [
    { rank: 1,  team: "Napoli",               played: 28, won: 19, draw: 5, lost: 4, gd: 30,  points: 62 },
    { rank: 2,  team: "Inter Milan",          played: 28, won: 18, draw: 6, lost: 4, gd: 29,  points: 60 },
    { rank: 3,  team: "Atalanta",             played: 28, won: 17, draw: 5, lost: 6, gd: 25,  points: 56 },
    { rank: 4,  team: "Juventus",             played: 28, won: 15, draw: 7, lost: 6, gd: 16,  points: 52 },
    { rank: 5,  team: "Lazio",                played: 28, won: 14, draw: 5, lost: 9, gd:  9,  points: 47 },
    { rank: 6,  team: "AC Milan",             played: 28, won: 13, draw: 6, lost: 9, gd:  7,  points: 45 },
    { rank: 7,  team: "Roma",                 played: 28, won: 11, draw: 7, lost: 10, gd: 0,  points: 40 },
    { rank: 8,  team: "Fiorentina",           played: 28, won: 11, draw: 5, lost: 12, gd: -2, points: 38 },
  ],
  FL1: [
    { rank: 1,  team: "Paris Saint-Germain",  played: 27, won: 20, draw: 4, lost: 3, gd: 46,  points: 64 },
    { rank: 2,  team: "Monaco",               played: 27, won: 17, draw: 4, lost: 6, gd: 22,  points: 55 },
    { rank: 3,  team: "Lille",                played: 27, won: 15, draw: 6, lost: 6, gd: 15,  points: 51 },
    { rank: 4,  team: "Lyon",                 played: 27, won: 13, draw: 6, lost: 8, gd:  9,  points: 45 },
    { rank: 5,  team: "Marseille",            played: 27, won: 12, draw: 7, lost: 8, gd:  6,  points: 43 },
    { rank: 6,  team: "Nice",                 played: 27, won: 12, draw: 5, lost: 10, gd: 4,  points: 41 },
    { rank: 7,  team: "Rennes",               played: 27, won: 10, draw: 7, lost: 10, gd: -2, points: 37 },
    { rank: 8,  team: "Lens",                 played: 27, won: 10, draw: 4, lost: 13, gd: -5, points: 34 },
  ],
  CL: [
    { rank: 1,  team: "Liverpool",            played: 8, won: 7, draw: 0, lost: 1, gd: 18,  points: 21 },
    { rank: 2,  team: "Barcelona",            played: 8, won: 6, draw: 1, lost: 1, gd: 14,  points: 19 },
    { rank: 3,  team: "Bayern Munich",        played: 8, won: 6, draw: 1, lost: 1, gd: 12,  points: 19 },
    { rank: 4,  team: "Arsenal",              played: 8, won: 5, draw: 2, lost: 1, gd:  8,  points: 17 },
    { rank: 5,  team: "Inter Milan",          played: 8, won: 5, draw: 1, lost: 2, gd:  6,  points: 16 },
    { rank: 6,  team: "Atletico Madrid",      played: 8, won: 5, draw: 1, lost: 2, gd:  4,  points: 16 },
    { rank: 7,  team: "Bayer Leverkusen",     played: 8, won: 4, draw: 2, lost: 2, gd:  5,  points: 14 },
    { rank: 8,  team: "Borussia Dortmund",    played: 8, won: 4, draw: 2, lost: 2, gd:  2,  points: 14 },
  ],
  EL: [
    { rank: 1,  team: "Manchester United",    played: 8, won: 6, draw: 1, lost: 1, gd: 12,  points: 19 },
    { rank: 2,  team: "Roma",                 played: 8, won: 5, draw: 2, lost: 1, gd:  8,  points: 17 },
    { rank: 3,  team: "Lazio",                played: 8, won: 5, draw: 1, lost: 2, gd:  6,  points: 16 },
    { rank: 4,  team: "Lyon",                 played: 8, won: 4, draw: 3, lost: 1, gd:  5,  points: 15 },
    { rank: 5,  team: "Ajax",                 played: 8, won: 4, draw: 2, lost: 2, gd:  4,  points: 14 },
    { rank: 6,  team: "Porto",                played: 8, won: 4, draw: 1, lost: 3, gd:  2,  points: 13 },
    { rank: 7,  team: "Tottenham",            played: 8, won: 3, draw: 3, lost: 2, gd:  1,  points: 12 },
    { rank: 8,  team: "Fenerbahce",           played: 8, won: 3, draw: 2, lost: 3, gd: -1,  points: 11 },
  ],
};

export const FIXTURES: Fixture[] = [
  { date: "2026-03-22", home: "Arsenal",              away: "Chelsea",             competition: "PL" },
  { date: "2026-03-22", home: "Manchester City",      away: "Liverpool",           competition: "PL" },
  { date: "2026-03-22", home: "Tottenham",            away: "Newcastle",           competition: "PL" },
  { date: "2026-04-05", home: "Liverpool",            away: "Nottm Forest",        competition: "PL" },
  { date: "2026-04-05", home: "Chelsea",              away: "Aston Villa",         competition: "PL" },
  { date: "2026-03-22", home: "Real Madrid",          away: "Barcelona",           competition: "PD" },
  { date: "2026-03-22", home: "Atletico Madrid",      away: "Sevilla",             competition: "PD" },
  { date: "2026-04-05", home: "Barcelona",            away: "Athletic Club",       competition: "PD" },
  { date: "2026-04-05", home: "Villarreal",           away: "Real Sociedad",       competition: "PD" },
  { date: "2026-03-22", home: "Bayern Munich",        away: "Bayer Leverkusen",   competition: "BL1" },
  { date: "2026-03-22", home: "Borussia Dortmund",    away: "RB Leipzig",         competition: "BL1" },
  { date: "2026-04-05", home: "Eintracht Frankfurt",  away: "Bayern Munich",      competition: "BL1" },
  { date: "2026-03-23", home: "Inter Milan",          away: "Napoli",              competition: "SA" },
  { date: "2026-03-23", home: "Juventus",             away: "Atalanta",            competition: "SA" },
  { date: "2026-04-06", home: "Napoli",               away: "AC Milan",            competition: "SA" },
  { date: "2026-03-22", home: "Paris Saint-Germain",  away: "Monaco",              competition: "FL1" },
  { date: "2026-03-22", home: "Lyon",                 away: "Marseille",           competition: "FL1" },
  { date: "2026-04-05", home: "Monaco",               away: "Lille",               competition: "FL1" },
  { date: "2026-03-25", home: "Arsenal",              away: "Bayern Munich",       competition: "CL" },
  { date: "2026-03-25", home: "Barcelona",            away: "Inter Milan",         competition: "CL" },
  { date: "2026-04-08", home: "Liverpool",            away: "Atletico Madrid",     competition: "CL" },
  { date: "2026-03-26", home: "Manchester United",    away: "Roma",                competition: "EL" },
  { date: "2026-03-26", home: "Lazio",                away: "Ajax",                competition: "EL" },
  { date: "2026-04-09", home: "Lyon",                 away: "Tottenham",           competition: "EL" },
];

export const TEAMS: TeamInfo[] = [
  { name: "Arsenal",              aliases: ["arsenal", "gunners", "afc"],                                   competition: "PL",  stadium: "Emirates Stadium",                  manager: "Mikel Arteta" },
  { name: "Chelsea",              aliases: ["chelsea", "blues", "cfc"],                                     competition: "PL",  stadium: "Stamford Bridge",                   manager: "Enzo Maresca" },
  { name: "Liverpool",            aliases: ["liverpool", "reds", "lfc"],                                    competition: "PL",  stadium: "Anfield",                           manager: "Arne Slot" },
  { name: "Manchester City",      aliases: ["manchester city", "man city", "mcfc"],                         competition: "PL",  stadium: "Etihad Stadium",                    manager: "Pep Guardiola" },
  { name: "Manchester United",    aliases: ["manchester united", "man united", "man utd", "mufc"],          competition: "PL",  stadium: "Old Trafford",                      manager: "Ruben Amorim" },
  { name: "Tottenham",            aliases: ["tottenham", "spurs", "thfc"],                                  competition: "PL",  stadium: "Tottenham Hotspur Stadium",         manager: "Ange Postecoglou" },
  { name: "Aston Villa",          aliases: ["aston villa", "villa", "avfc"],                                competition: "PL",  stadium: "Villa Park",                        manager: "Unai Emery" },
  { name: "Newcastle",            aliases: ["newcastle", "magpies", "nufc"],                                competition: "PL",  stadium: "St. James' Park",                   manager: "Eddie Howe" },
  { name: "Nottm Forest",         aliases: ["nottingham forest", "nottm forest", "forest", "nffc"],         competition: "PL",  stadium: "City Ground",                       manager: "Nuno Espirito Santo" },
  { name: "Fulham",               aliases: ["fulham", "cottagers", "ffc"],                                  competition: "PL",  stadium: "Craven Cottage",                    manager: "Marco Silva" },
  { name: "Barcelona",            aliases: ["barcelona", "barca", "fcb"],                                   competition: "PD",  stadium: "Estadi Olímpic Lluís Companys",    manager: "Hansi Flick" },
  { name: "Real Madrid",          aliases: ["real madrid", "madrid", "los blancos"],                        competition: "PD",  stadium: "Santiago Bernabéu",                 manager: "Carlo Ancelotti" },
  { name: "Atletico Madrid",      aliases: ["atletico madrid", "atletico", "atleti"],                       competition: "PD",  stadium: "Cívitas Metropolitano",             manager: "Diego Simeone" },
  { name: "Athletic Club",        aliases: ["athletic club", "athletic bilbao", "bilbao"],                  competition: "PD",  stadium: "San Mamés",                         manager: "Ernesto Valverde" },
  { name: "Villarreal",           aliases: ["villarreal", "yellow submarine"],                              competition: "PD",  stadium: "Estadio de la Cerámica",            manager: "Marcelino" },
  { name: "Real Sociedad",        aliases: ["real sociedad", "la real"],                                    competition: "PD",  stadium: "Reale Arena",                       manager: "Imanol Alguacil" },
  { name: "Betis",                aliases: ["betis", "real betis"],                                         competition: "PD",  stadium: "Benito Villamarín",                 manager: "Manuel Pellegrini" },
  { name: "Sevilla",              aliases: ["sevilla", "sfc"],                                              competition: "PD",  stadium: "Ramón Sánchez-Pizjuán",            manager: "Francisco Machado" },
  { name: "Bayern Munich",        aliases: ["bayern munich", "bayern", "die roten", "fcbayern"],            competition: "BL1", stadium: "Allianz Arena",                     manager: "Vincent Kompany" },
  { name: "Bayer Leverkusen",     aliases: ["bayer leverkusen", "leverkusen", "die werkself"],              competition: "BL1", stadium: "BayArena",                          manager: "Xabi Alonso" },
  { name: "Borussia Dortmund",    aliases: ["borussia dortmund", "dortmund", "bvb"],                        competition: "BL1", stadium: "Signal Iduna Park",                 manager: "Niko Kovac" },
  { name: "Eintracht Frankfurt",  aliases: ["eintracht frankfurt", "frankfurt", "sge"],                    competition: "BL1", stadium: "Deutsche Bank Park",                manager: "Dino Toppmoller" },
  { name: "RB Leipzig",           aliases: ["rb leipzig", "leipzig", "rbl"],                               competition: "BL1", stadium: "Red Bull Arena",                    manager: "Marco Rose" },
  { name: "Napoli",               aliases: ["napoli", "partenopei", "sscn"],                               competition: "SA",  stadium: "Stadio Diego Armando Maradona",     manager: "Antonio Conte" },
  { name: "Inter Milan",          aliases: ["inter milan", "inter", "nerazzurri", "fcim"],                 competition: "SA",  stadium: "San Siro",                          manager: "Simone Inzaghi" },
  { name: "Atalanta",             aliases: ["atalanta", "la dea"],                                         competition: "SA",  stadium: "Gewiss Stadium",                    manager: "Gian Piero Gasperini" },
  { name: "Juventus",             aliases: ["juventus", "juve", "la vecchia signora"],                     competition: "SA",  stadium: "Juventus Stadium",                  manager: "Thiago Motta" },
  { name: "Lazio",                aliases: ["lazio", "biancocelesti", "ss lazio"],                         competition: "SA",  stadium: "Stadio Olimpico",                   manager: "Marco Baroni" },
  { name: "AC Milan",             aliases: ["ac milan", "milan", "rossoneri"],                             competition: "SA",  stadium: "San Siro",                          manager: "Sergio Conceicao" },
  { name: "Roma",                 aliases: ["roma", "as roma", "giallorossi"],                             competition: "SA",  stadium: "Stadio Olimpico",                   manager: "Claudio Ranieri" },
  { name: "Fiorentina",           aliases: ["fiorentina", "viola", "acf"],                                 competition: "SA",  stadium: "Stadio Artemio Franchi",            manager: "Raffaele Palladino" },
  { name: "Paris Saint-Germain",  aliases: ["paris saint-germain", "psg", "paris"],                        competition: "FL1", stadium: "Parc des Princes",                  manager: "Luis Enrique" },
  { name: "Monaco",               aliases: ["monaco", "asm"],                                              competition: "FL1", stadium: "Stade Louis II",                    manager: "Adi Hutter" },
  { name: "Lille",                aliases: ["lille", "losc", "les dogues"],                                competition: "FL1", stadium: "Stade Pierre-Mauroy",               manager: "Bruno Genesio" },
  { name: "Lyon",                 aliases: ["lyon", "ol", "olympique lyonnais"],                           competition: "FL1", stadium: "Groupama Stadium",                  manager: "Paulo Fonseca" },
  { name: "Marseille",            aliases: ["marseille", "om", "olympique de marseille"],                  competition: "FL1", stadium: "Stade Vélodrome",                   manager: "Jean-Louis Gasset" },
  { name: "Ajax",                 aliases: ["ajax", "ajfc"],                                               competition: "EL",  stadium: "Johan Cruyff Arena",                manager: "Francesco Farioli" },
  { name: "Porto",                aliases: ["porto", "fcp", "os dragoes"],                                 competition: "EL",  stadium: "Estádio do Dragão",                 manager: "Vitor Bruno" },
  { name: "Fenerbahce",           aliases: ["fenerbahce", "fener", "fb"],                                  competition: "EL",  stadium: "Şükrü Saracoğlu",                  manager: "Jose Mourinho" },
];

export function findTeam(query: string): TeamInfo | undefined {
  const q = query.toLowerCase();
  return TEAMS.find(t => t.aliases.some(a => q.includes(a)));
}

export function fixturesByCompetition(code: CompetitionCode): Fixture[] {
  return FIXTURES.filter(f => f.competition === code);
}

export function fixturesByTeam(teamName: string): Fixture[] {
  const n = teamName.toLowerCase();
  return FIXTURES.filter(
    f => f.home.toLowerCase().includes(n) || f.away.toLowerCase().includes(n)
  );
}
