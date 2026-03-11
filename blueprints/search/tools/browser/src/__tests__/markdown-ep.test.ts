import { describe, it, expect } from "vitest";
import { htmlToMarkdown } from "../markdown";

describe("htmlToMarkdown (used by /api/markdown fallback)", () => {
  it("converts h1 to # heading", () => {
    expect(htmlToMarkdown("<h1>Hello</h1>")).toContain("# Hello");
  });

  it("strips script tags", () => {
    const out = htmlToMarkdown("<script>alert(1)</script><p>text</p>");
    expect(out).not.toContain("alert");
    expect(out).toContain("text");
  });

  it("converts links", () => {
    const out = htmlToMarkdown('<a href="https://example.com">click</a>');
    expect(out).toContain("[click](https://example.com)");
  });
});
