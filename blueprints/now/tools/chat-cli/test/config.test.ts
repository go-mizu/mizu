import { describe, it, before, after } from "node:test";
import assert from "node:assert/strict";
import { mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import { tmpdir } from "node:os";
import { loadConfig, saveConfig, importGoConfig, type Config } from "../src/auth/config.ts";

describe("config", () => {
  let dir: string;

  before(() => {
    dir = mkdtempSync(join(tmpdir(), "chat-now-test-"));
  });

  after(() => {
    rmSync(dir, { recursive: true });
  });

  it("returns null when config does not exist", async () => {
    const cfg = await loadConfig(join(dir, "nonexistent.json"));
    assert.equal(cfg, null);
  });

  it("saves and loads config", async () => {
    const path = join(dir, "config.json");
    const config: Config = {
      actor: "u/alice",
      public_key: "dGVzdHB1YmtleQ",
      private_key: "dGVzdHByaXZrZXk",
      fingerprint: "a1b2c3d4e5f67890",
      server: "https://chat.go-mizu.workers.dev",
    };
    await saveConfig(path, config);
    const loaded = await loadConfig(path);
    assert.deepEqual(loaded, config);
  });

  it("imports Go CLI config stripping padding", async () => {
    const goPath = join(dir, "go-config.json");
    writeFileSync(
      goPath,
      JSON.stringify({
        actor: "u/bob",
        public_key: "dGVzdA==",
        private_key: "cHJpdg==",
        fingerprint: "deadbeef12345678",
      }),
    );
    const cfg = await importGoConfig(goPath);
    assert.equal(cfg!.actor, "u/bob");
    assert.equal(cfg!.public_key, "dGVzdA");
    assert.equal(cfg!.private_key, "cHJpdg");
    assert.equal(cfg!.server, "https://chat.go-mizu.workers.dev");
  });
});
