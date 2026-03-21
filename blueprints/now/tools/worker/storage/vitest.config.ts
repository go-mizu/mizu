import { defineConfig } from "vitest/config";
import { cloudflareTest } from "@cloudflare/vitest-pool-workers";
import { readFileSync } from "fs";
import { join } from "path";

/** Load env vars from $HOME/data/.local.env (for POSTGRES_DSN). */
function loadLocalEnv(): Record<string, string> {
  try {
    const content = readFileSync(join(process.env.HOME!, "data/.local.env"), "utf8");
    const env: Record<string, string> = {};
    for (const line of content.split("\n")) {
      const match = line.match(/^(\w+)="([^"]*)"/);
      if (match) env[match[1]] = match[2];
    }
    return env;
  } catch {
    return {};
  }
}

const localEnv = loadLocalEnv();

export default defineConfig({
  plugins: [
    cloudflareTest({
      wrangler: { configPath: "./wrangler.toml" },
      miniflare: {
        d1Databases: ["DB"],
        r2Buckets: ["BUCKET"],
        hyperdrives: {
          HYPERDRIVE: localEnv.POSTGRES_DSN || "postgresql://user:pass@localhost:5432/db",
        },
        bindings: {
          R2_ENDPOINT: "https://test.r2.cloudflarestorage.com",
          R2_ACCESS_KEY_ID: "test-key-id",
          R2_SECRET_ACCESS_KEY: "test-secret-key",
          POSTGRES_DSN: localEnv.POSTGRES_DSN || "",
        },
      },
    }),
  ],
});
