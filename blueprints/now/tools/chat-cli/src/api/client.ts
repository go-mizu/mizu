import type { Config } from "../auth/config.js";
import { base64url } from "../auth/signer.js";
import type { Chat, Message, RegisterResponse, ListResponse } from "./types.js";
import { ApiError, AuthError, RateLimitError } from "./types.js";

type Signer = (method: string, path: string, query: string, body: string) => Promise<string>;
type Fetcher = typeof globalThis.fetch;

export class ChatClient {
  private server: string;
  private sign: Signer;
  private fetch: Fetcher;

  constructor(
    config: Config,
    signFn: Signer,
    fetchFn?: Fetcher,
  ) {
    this.server = config.server.replace(/\/$/, "");
    this.sign = signFn;
    this.fetch = fetchFn || globalThis.fetch.bind(globalThis);
  }

  private async request<T>(
    method: string,
    path: string,
    opts?: { query?: string; body?: string; noAuth?: boolean },
  ): Promise<T> {
    const query = opts?.query || "";
    const body = opts?.body || "";
    const url = `${this.server}${path}${query ? "?" + query : ""}`;

    const headers: Record<string, string> = {};
    if (body) headers["Content-Type"] = "application/json";
    if (!opts?.noAuth) {
      headers["Authorization"] = await this.sign(method, path, query, body);
    }

    const res = await this.fetch(url, {
      method,
      headers,
      body: body || undefined,
    });

    if (!res.ok) {
      const text = await res.text();
      if (res.status === 401 || res.status === 403) throw new AuthError(res.status, text);
      if (res.status === 429) {
        const retryAfter = parseInt(res.headers.get("retry-after") || "60", 10);
        throw new RateLimitError(res.status, text, retryAfter);
      }
      throw new ApiError(res.status, text);
    }

    if (res.status === 204) return undefined as T;
    return res.json() as Promise<T>;
  }

  async register(actor: string, publicKey: Uint8Array): Promise<RegisterResponse> {
    return this.request<RegisterResponse>("POST", "/api/register", {
      body: JSON.stringify({ actor, public_key: base64url(publicKey) }),
      noAuth: true,
    });
  }

  async createChat(opts: { title?: string; visibility?: string } = {}): Promise<Chat> {
    return this.request<Chat>("POST", "/api/chat", {
      body: JSON.stringify({ kind: "room", ...opts }),
    });
  }

  async getChat(id: string): Promise<Chat> {
    return this.request<Chat>("GET", `/api/chat/${id}`);
  }

  async listChats(opts?: { limit?: number }): Promise<Chat[]> {
    const params = new URLSearchParams();
    if (opts?.limit) params.set("limit", String(opts.limit));
    const query = params.toString();
    const res = await this.request<ListResponse<Chat>>("GET", "/api/chat", { query });
    return res.items;
  }

  async joinChat(id: string): Promise<void> {
    await this.request<void>("POST", `/api/chat/${id}/join`);
  }

  async startDm(peer: string): Promise<Chat> {
    return this.request<Chat>("POST", "/api/chat/dm", {
      body: JSON.stringify({ peer }),
    });
  }

  async listDms(opts?: { limit?: number }): Promise<Chat[]> {
    const params = new URLSearchParams();
    if (opts?.limit) params.set("limit", String(opts.limit));
    const query = params.toString();
    const res = await this.request<ListResponse<Chat>>("GET", "/api/chat/dm", { query });
    return res.items;
  }

  async sendMessage(chatId: string, text: string): Promise<Message> {
    return this.request<Message>("POST", `/api/chat/${chatId}/messages`, {
      body: JSON.stringify({ text }),
    });
  }

  async listMessages(
    chatId: string,
    opts?: { limit?: number; before?: string },
  ): Promise<Message[]> {
    const params = new URLSearchParams();
    if (opts?.limit) params.set("limit", String(opts.limit));
    if (opts?.before) params.set("before", opts.before);
    const query = params.toString();
    const res = await this.request<ListResponse<Message>>(
      "GET",
      `/api/chat/${chatId}/messages`,
      { query },
    );
    return res.items;
  }
}
