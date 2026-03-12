import { describe, it, expect } from "vitest";
import { filterLinksForEndpoint } from "../links-ep";

describe("filterLinksForEndpoint", () => {
  const links = [
    "https://example.com/a",
    "https://other.com/b",
    "https://example.com/c",
  ];

  it("returns all links when no filters set", () => {
    expect(filterLinksForEndpoint(links, "https://example.com", false, false)).toEqual(links);
  });

  it("excludes external links when excludeExternalLinks=true", () => {
    const result = filterLinksForEndpoint(links, "https://example.com", false, true);
    expect(result).toEqual(["https://example.com/a", "https://example.com/c"]);
  });

  it("returns all when visibleLinksOnly=true (fallback: same as all)", () => {
    const result = filterLinksForEndpoint(links, "https://example.com", true, false);
    expect(result).toEqual(links);
  });
});
