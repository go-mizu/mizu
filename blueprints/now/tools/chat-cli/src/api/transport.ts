import type { ChatClient } from "./client.js";
import type { Message, Chat } from "./types.js";

export type Unsubscribe = () => void;

export interface Transport {
  subscribeMessages(chatId: string, onMessages: (msgs: Message[]) => void): Unsubscribe;
  subscribeRooms(onRooms: (rooms: Chat[]) => void): Unsubscribe;
}

// Sentinel — can never equal any real fingerprint string
const UNSET = Symbol("unset");

export class PollingTransport implements Transport {
  private timers = new Map<string, ReturnType<typeof setInterval>>();
  private lastMessageIds = new Map<string, string>();
  private lastRoomFingerprint: string | typeof UNSET = UNSET;
  private onError?: (e: Error) => void;

  constructor(
    private client: ChatClient,
    private messageInterval = 3000,
    private roomInterval = 30000,
    opts?: { onError?: (e: Error) => void },
  ) {
    this.onError = opts?.onError;
  }

  subscribeMessages(chatId: string, onMessages: (msgs: Message[]) => void): Unsubscribe {
    const key = `msg:${chatId}`;
    this.clearTimer(key);

    const poll = async () => {
      try {
        const msgs = await this.client.listMessages(chatId, { limit: 50 });
        const fingerprint = msgs.map((m) => m.id).join(",");
        if (fingerprint !== this.lastMessageIds.get(chatId)) {
          this.lastMessageIds.set(chatId, fingerprint);
          onMessages(msgs);
        }
      } catch (e) {
        this.onError?.(e instanceof Error ? e : new Error(String(e)));
      }
    };

    poll();
    this.timers.set(key, setInterval(poll, this.messageInterval));
    return () => this.clearTimer(key);
  }

  subscribeRooms(onRooms: (rooms: Chat[]) => void): Unsubscribe {
    const key = "rooms";
    this.clearTimer(key);

    const poll = async () => {
      try {
        // Fetch independently — one failing shouldn't block the other
        const [rooms, dms] = await Promise.all([
          this.client.listChats().catch(() => [] as Chat[]),
          this.client.listDms().catch(() => [] as Chat[]),
        ]);
        const all = [...rooms, ...dms];
        const fingerprint = all.map((r) => r.id).join(",");
        // UNSET sentinel ensures first poll always fires callback
        if (fingerprint !== this.lastRoomFingerprint) {
          this.lastRoomFingerprint = fingerprint;
          onRooms(all);
        }
      } catch (e) {
        this.onError?.(e instanceof Error ? e : new Error(String(e)));
      }
    };

    poll();
    this.timers.set(key, setInterval(poll, this.roomInterval));
    return () => this.clearTimer(key);
  }

  private clearTimer(key: string) {
    const existing = this.timers.get(key);
    if (existing) {
      clearInterval(existing);
      this.timers.delete(key);
    }
  }

  destroy() {
    for (const key of this.timers.keys()) this.clearTimer(key);
  }
}
