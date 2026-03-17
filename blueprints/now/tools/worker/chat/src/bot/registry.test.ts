import { describe, it, expect, beforeEach } from "vitest";
// Each vitest test file gets its own module scope, so cross-file state is isolated.
// Within this file, `registry` is a module singleton — we call _resetForTesting()
// in beforeEach so every test starts with an empty registry.
import {
  registerBot, isBuiltInBot, getBotProfile,
  listBotActors, dispatchReply, _resetForTesting,
} from "./registry";

beforeEach(() => {
  _resetForTesting();
});

const testProfile = {
  bio: "A test bot.",
  examples: ["hello", "world"],
};

describe("registerBot / isBuiltInBot", () => {
  it("registers a bot and recognises it", () => {
    registerBot({ actor: "a/test1", profile: testProfile, reply: () => "hi" });
    expect(isBuiltInBot("a/test1")).toBe(true);
  });

  it("returns false for unknown actors", () => {
    expect(isBuiltInBot("a/nobody")).toBe(false);
  });

  it("throws when registering a duplicate actor", () => {
    registerBot({ actor: "a/dup", profile: testProfile, reply: () => "x" });
    expect(() =>
      registerBot({ actor: "a/dup", profile: testProfile, reply: () => "y" })
    ).toThrow("Bot already registered: a/dup");
  });
});

describe("getBotProfile", () => {
  it("returns the profile for a registered bot", () => {
    registerBot({ actor: "a/profiled", profile: testProfile, reply: () => "hi" });
    expect(getBotProfile("a/profiled")).toEqual(testProfile);
  });

  it("returns null for unknown actors", () => {
    expect(getBotProfile("a/ghost")).toBeNull();
  });
});

describe("listBotActors", () => {
  it("includes all registered actors", () => {
    registerBot({ actor: "a/botA", profile: testProfile, reply: () => "a" });
    registerBot({ actor: "a/botB", profile: testProfile, reply: () => "b" });
    const list = listBotActors();
    expect(list).toContain("a/botA");
    expect(list).toContain("a/botB");
  });

  it("returns empty array when no bots registered", () => {
    expect(listBotActors()).toEqual([]);
  });
});

describe("dispatchReply", () => {
  it("calls the bot reply and returns the string", async () => {
    registerBot({ actor: "a/pong", profile: testProfile, reply: () => "pong" });
    const result = await dispatchReply("a/pong", "ping", {} as D1Database);
    expect(result).toBe("pong");
  });

  it("returns null for unregistered actors", async () => {
    const result = await dispatchReply("a/unknown", "hi", {} as D1Database);
    expect(result).toBeNull();
  });

  it("awaits async reply functions", async () => {
    registerBot({
      actor: "a/async",
      profile: testProfile,
      reply: async (msg) => `async:${msg}`,
    });
    const result = await dispatchReply("a/async", "hello", {} as D1Database);
    expect(result).toBe("async:hello");
  });
});
