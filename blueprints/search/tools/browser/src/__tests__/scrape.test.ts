import { describe, it, expect } from "vitest";
import { scrapeHtml } from "../scrape";

describe("scrapeHtml", () => {
  const html = `<html><body>
    <h1 class="title">Main Title</h1>
    <h2>Sub One</h2>
    <h2>Sub Two</h2>
    <p id="intro">Intro text</p>
  </body></html>`;

  it("extracts text from a single selector", async () => {
    const results = await scrapeHtml(html, [{ selector: "h1" }]);
    expect(results).toHaveLength(1);
    expect(results[0].selector).toBe("h1");
    expect(results[0].results).toHaveLength(1);
    expect(results[0].results[0].text).toContain("Main Title");
  });

  it("extracts multiple matches for the same selector", async () => {
    const results = await scrapeHtml(html, [{ selector: "h2" }]);
    expect(results[0].results).toHaveLength(2);
  });

  it("extracts attributes", async () => {
    const results = await scrapeHtml(html, [{ selector: "h1" }]);
    const attrs = results[0].results[0].attributes;
    expect(attrs).toContainEqual({ name: "class", value: "title" });
  });

  it("returns zero dimensions (fallback, no layout engine)", async () => {
    const results = await scrapeHtml(html, [{ selector: "h1" }]);
    expect(results[0].results[0].height).toBe(0);
    expect(results[0].results[0].width).toBe(0);
  });

  it("returns empty results for non-matching selector", async () => {
    const results = await scrapeHtml(html, [{ selector: "table" }]);
    expect(results[0].results).toHaveLength(0);
  });

  it("handles multiple selectors in one call", async () => {
    const results = await scrapeHtml(html, [{ selector: "h1" }, { selector: "p" }]);
    expect(results).toHaveLength(2);
    expect(results[1].results[0].text).toContain("Intro text");
  });
});
