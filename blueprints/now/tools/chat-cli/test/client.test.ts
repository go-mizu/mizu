import { describe, it } from "node:test";
import assert from "node:assert/strict";
import { ChatClient } from "../src/api/client.ts";
import type { Config } from "../src/auth/config.ts";

const mockSigner = async () =>
  "CHAT-ED25519 Credential=u/test, Timestamp=1710000000, Signature=dGVzdA";

function mockFetch(response: {
  status: number;
  body: unknown;
  headers?: Record<string, string>;
}) {
  return async (_url: string | URL | Request, _init?: RequestInit) => {
    return {
      ok: response.status >= 200 && response.status < 300,
      status: response.status,
      headers: new Headers(response.headers || {}),
      json: async () => response.body,
      text: async () => JSON.stringify(response.body),
    } as Response;
  };
}

const testConfig: Config = {
  actor: "u/test",
  public_key: "dGVzdA",
  private_key: "dGVzdA",
  fingerprint: "1234567890abcdef",
  server: "https://chat.go-mizu.workers.dev",
};

describe("ChatClient", () => {
  it("register sends correct request", async () => {
    let capturedUrl = "";
    const client = new ChatClient(testConfig, mockSigner, async (url, _init) => {
      capturedUrl = url as string;
      return {
        ok: true,
        status: 201,
        json: async () => ({ actor: "u/test", recovery_code: "abc" }),
        text: async () => "",
        headers: new Headers(),
      } as Response;
    });
    const result = await client.register("u/test", new Uint8Array([1, 2, 3]));
    assert.ok(capturedUrl.endsWith("/api/register"));
    assert.equal(result.actor, "u/test");
  });

  it("createChat sends POST /api/chat", async () => {
    let capturedInit: RequestInit | undefined;
    const client = new ChatClient(testConfig, mockSigner, async (_url, init) => {
      capturedInit = init;
      return {
        ok: true,
        status: 201,
        json: async () => ({
          id: "chat_abc",
          kind: "room",
          title: "test",
          creator: "u/test",
          created_at: "2026-03-17T00:00:00Z",
        }),
        text: async () => "",
        headers: new Headers(),
      } as Response;
    });
    const chat = await client.createChat({ title: "test" });
    assert.equal(chat.id, "chat_abc");
    assert.equal(capturedInit?.method, "POST");
  });

  it("listChats unwraps items envelope", async () => {
    const client = new ChatClient(
      testConfig,
      mockSigner,
      mockFetch({ status: 200, body: { items: [{ id: "chat_1" }, { id: "chat_2" }] } }),
    );
    const chats = await client.listChats();
    assert.equal(chats.length, 2);
  });

  it("throws AuthError on 401", async () => {
    const client = new ChatClient(
      testConfig,
      mockSigner,
      mockFetch({ status: 401, body: { error: "unauthorized" } }),
    );
    await assert.rejects(() => client.listChats(), { name: "AuthError" });
  });

  it("throws RateLimitError on 429", async () => {
    const client = new ChatClient(
      testConfig,
      mockSigner,
      mockFetch({
        status: 429,
        body: { error: "rate limited" },
        headers: { "retry-after": "60" },
      }),
    );
    await assert.rejects(() => client.listChats(), { name: "RateLimitError" });
  });

  it("sendMessage sends POST with text body", async () => {
    let capturedBody = "";
    const client = new ChatClient(testConfig, mockSigner, async (_url, init) => {
      capturedBody = init?.body as string;
      return {
        ok: true,
        status: 201,
        json: async () => ({
          id: "msg_abc",
          chat: "chat_1",
          actor: "u/test",
          text: "hello",
          created_at: "2026-03-17T00:00:00Z",
        }),
        text: async () => "",
        headers: new Headers(),
      } as Response;
    });
    const msg = await client.sendMessage("chat_1", "hello");
    assert.equal(msg.id, "msg_abc");
    assert.deepEqual(JSON.parse(capturedBody), { text: "hello" });
  });

  it("joinChat sends POST and handles 204", async () => {
    const client = new ChatClient(testConfig, mockSigner, async () => {
      return {
        ok: true,
        status: 204,
        json: async () => undefined,
        text: async () => "",
        headers: new Headers(),
      } as Response;
    });
    await client.joinChat("chat_1"); // Should not throw
  });
});
