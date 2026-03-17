import { readFile, writeFile, mkdir } from "node:fs/promises";
import { dirname, join } from "node:path";
import { homedir } from "node:os";

export interface Config {
  actor: string;
  public_key: string;
  private_key: string;
  fingerprint: string;
  server: string;
}

const DEFAULT_SERVER = "https://chat.go-mizu.workers.dev";
const DEFAULT_PATH = join(homedir(), ".config", "chat-now", "config.json");
const GO_CLI_PATH = join(homedir(), ".config", "now", "config.json");

export function defaultConfigPath(): string {
  return DEFAULT_PATH;
}

export function goCliConfigPath(): string {
  return GO_CLI_PATH;
}

export async function loadConfig(path: string): Promise<Config | null> {
  try {
    const raw = await readFile(path, "utf-8");
    return JSON.parse(raw) as Config;
  } catch {
    return null;
  }
}

export async function saveConfig(path: string, config: Config): Promise<void> {
  await mkdir(dirname(path), { recursive: true, mode: 0o700 });
  await writeFile(path, JSON.stringify(config, null, 2) + "\n", { mode: 0o600 });
}

function stripPadding(s: string): string {
  return s.replace(/=+$/, "");
}

export async function importGoConfig(path: string): Promise<Config | null> {
  const raw = await loadConfig(path);
  if (!raw) return null;
  return {
    actor: raw.actor,
    public_key: stripPadding(raw.public_key),
    private_key: stripPadding(raw.private_key),
    fingerprint: raw.fingerprint,
    server: raw.server || DEFAULT_SERVER,
  };
}

export async function resolveConfig(overridePath?: string): Promise<Config | null> {
  if (overridePath) return loadConfig(overridePath);
  const cfg = await loadConfig(DEFAULT_PATH);
  if (cfg) return cfg;
  return importGoConfig(GO_CLI_PATH);
}
