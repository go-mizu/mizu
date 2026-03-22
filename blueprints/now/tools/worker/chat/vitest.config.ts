import { defineConfig } from "vitest/config";

export default defineConfig({
  test: {
    include: ["src/bot/**/*.test.ts"],
    environment: "node",
  },
});
