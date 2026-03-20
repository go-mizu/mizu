import { defineConfig } from "vitest/config";
import { cloudflareTest } from "@cloudflare/vitest-pool-workers";

export default defineConfig({
  plugins: [
    cloudflareTest({
      wrangler: { configPath: "./wrangler.toml" },
      miniflare: {
        d1Databases: ["DB"],
        r2Buckets: ["BUCKET"],
        bindings: {
          R2_ENDPOINT: "https://test.r2.cloudflarestorage.com",
          R2_ACCESS_KEY_ID: "test-key-id",
          R2_SECRET_ACCESS_KEY: "test-secret-key",
        },
      },
    }),
  ],
});
