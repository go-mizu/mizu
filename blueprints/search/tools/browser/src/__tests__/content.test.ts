import { describe, it, expect } from "vitest";
import { buildContentResult } from "../content";

describe("buildContentResult", () => {
  it("returns html string for valid HTML", () => {
    const html = "<html><head><title>Test</title></head><body>Hello</body></html>";
    const result = buildContentResult(html);
    expect(result.html).toBe(html);
    expect(result.title).toBe("Test");
  });

  it("returns empty title when no <title> tag", () => {
    const html = "<html><body>No title</body></html>";
    expect(buildContentResult(html).title).toBe("");
  });
});
