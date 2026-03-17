export interface BotProfile {
  bio: string;
  examples: string[];
}

export interface BotDef {
  actor: string;
  profile: BotProfile;
  reply: (msg: string, db: D1Database) => Promise<string> | string;
}

const registry = new Map<string, BotDef>();

export function registerBot(def: BotDef): void {
  if (registry.has(def.actor)) {
    throw new Error(`Bot already registered: ${def.actor}`);
  }
  registry.set(def.actor, def);
}

/** Only for use in tests — clears all registered bots. */
export function _resetForTesting(): void {
  registry.clear();
}

export function isBuiltInBot(actor: string): boolean {
  return registry.has(actor);
}

export function getBotProfile(actor: string): BotProfile | null {
  return registry.get(actor)?.profile ?? null;
}

export function listBotActors(): string[] {
  return Array.from(registry.keys());
}

export async function dispatchReply(
  actor: string,
  msg: string,
  db: D1Database
): Promise<string | null> {
  const bot = registry.get(actor);
  if (!bot) return null;
  return await bot.reply(msg, db);
}
