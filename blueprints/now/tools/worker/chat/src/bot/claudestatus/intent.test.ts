import { describe, it, expect } from "vitest";
import { detectIntent } from "./intent";

describe("detectIntent — status", () => {
  it("detects 'is claude down'", () => {
    expect(detectIntent("is claude down?").intent).toBe("status");
  });
  it("detects 'is it up'", () => {
    expect(detectIntent("is it up?").intent).toBe("status");
  });
  it("detects 'outage'", () => {
    expect(detectIntent("any outage today?").intent).toBe("status");
  });
  it("detects 'operational'", () => {
    expect(detectIntent("is everything operational?").intent).toBe("status");
  });
  it("detects 'working'", () => {
    expect(detectIntent("is claude working").intent).toBe("status");
  });
});

describe("detectIntent — components", () => {
  it("detects 'api'", () => {
    expect(detectIntent("is the api up?").intent).toBe("components");
  });
  it("detects 'claude code'", () => {
    expect(detectIntent("how is claude code doing?").intent).toBe("components");
  });
  it("detects 'platform'", () => {
    expect(detectIntent("check platform status").intent).toBe("components");
  });
  it("detects 'component'", () => {
    expect(detectIntent("show me component status").intent).toBe("components");
  });
});

describe("detectIntent — incidents", () => {
  it("detects 'incident'", () => {
    expect(detectIntent("any incidents recently?").intent).toBe("incidents");
  });
  it("detects 'what happened'", () => {
    expect(detectIntent("what happened today?").intent).toBe("incidents");
  });
  it("detects 'past issues'", () => {
    expect(detectIntent("show me past issues").intent).toBe("incidents");
  });
});

describe("detectIntent — incident_detail", () => {
  it("detects 'latest incident'", () => {
    expect(detectIntent("latest incident details").intent).toBe("incident_detail");
  });
  it("detects 'last incident'", () => {
    expect(detectIntent("what was the last incident?").intent).toBe("incident_detail");
  });
  it("detects 'incident detail'", () => {
    expect(detectIntent("incident detail").intent).toBe("incident_detail");
  });
  it("'tell me about the api' does NOT hit incident_detail", () => {
    expect(detectIntent("tell me about the api").intent).toBe("components");
  });
});

describe("detectIntent — uptime", () => {
  it("detects 'uptime'", () => {
    expect(detectIntent("what is the uptime?").intent).toBe("uptime");
  });
  it("detects 'availability'", () => {
    expect(detectIntent("check availability").intent).toBe("uptime");
  });
  it("detects 'sla'", () => {
    expect(detectIntent("what is your sla?").intent).toBe("uptime");
  });
  it("detects 'reliability'", () => {
    expect(detectIntent("how reliable is claude?").intent).toBe("uptime");
  });
});

describe("detectIntent — help fallback", () => {
  it("returns help for unknown messages", () => {
    expect(detectIntent("hello there").intent).toBe("help");
  });
  it("returns help for empty string", () => {
    expect(detectIntent("").intent).toBe("help");
  });
});

describe("priority order", () => {
  it("incident_detail beats incidents", () => {
    expect(detectIntent("show latest incident history").intent).toBe("incident_detail");
  });
  it("incidents beats uptime", () => {
    expect(detectIntent("past issues availability").intent).toBe("incidents");
  });
});
