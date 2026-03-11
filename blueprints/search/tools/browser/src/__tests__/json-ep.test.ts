import { describe, it, expect } from "vitest";
import { jsonParamsObj } from "../json-ep";

describe("jsonParamsObj", () => {
  it("returns prompt and schema from request", () => {
    const req = {
      prompt: "extract the title",
      response_format: { type: "json_schema" as const, schema: { type: "object" } },
    };
    const obj = jsonParamsObj(req);
    expect(obj.prompt).toBe("extract the title");
    expect(obj.schema).toEqual({ type: "object" });
  });

  it("handles missing prompt and schema", () => {
    const obj = jsonParamsObj({});
    expect(obj.prompt).toBeUndefined();
    expect(obj.schema).toBeUndefined();
  });
});
