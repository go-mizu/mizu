import { describe, it, expect } from "vitest";
import { makeCacheKey, hashParams } from "../cache";

describe("makeCacheKey", () => {
  it("joins url, endpoint, hash with null byte", () => {
    expect(makeCacheKey("https://example.com", "content", "")).toBe(
      "https://example.com\0content\0"
    );
  });

  it("includes params hash when provided", () => {
    expect(makeCacheKey("https://x.com", "links", "abc123")).toBe(
      "https://x.com\0links\0abc123"
    );
  });
});

describe("hashParams", () => {
  it("returns empty string for null/undefined", async () => {
    expect(await hashParams(null)).toBe("");
    expect(await hashParams(undefined)).toBe("");
  });

  it("returns 16-char hex for an object", async () => {
    const h = await hashParams({ a: 1, b: 2 });
    expect(h).toMatch(/^[0-9a-f]{16}$/);
  });

  it("is deterministic (same input → same output)", async () => {
    const a = await hashParams({ x: true, y: false });
    const b = await hashParams({ y: false, x: true });
    expect(a).toBe(b);
  });

  it("produces different hashes for different inputs", async () => {
    const a = await hashParams({ selector: "h1" });
    const b = await hashParams({ selector: "h2" });
    expect(a).not.toBe(b);
  });
});
