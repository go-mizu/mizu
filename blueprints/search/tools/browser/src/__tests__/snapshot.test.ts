import { describe, it, expect } from "vitest";
import { buildSnapshotFallback } from "../snapshot";

describe("buildSnapshotFallback", () => {
  it("returns content and null screenshot", () => {
    const html = "<html><body>Hello</body></html>";
    const result = buildSnapshotFallback(html);
    expect(result.content).toBe(html);
    expect(result.screenshot).toBeNull();
  });
});
