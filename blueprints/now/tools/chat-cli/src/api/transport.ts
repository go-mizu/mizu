import type { ChatClient } from "./client.js";
import type { Message, Chat } from "./types.js";

export type Unsubscribe = () => void;

export interface Transport {
  subscribeMessages(chatId: string, onMessages: (msgs: Message[]) => void): Unsubscribe;
  subscribeRooms(onRooms: (rooms: Chat[]) => void): Unsubscribe;
}

export class PollingTransport implements Transport {
  private timers = new Map<string, ReturnType<typeof setInterval>>();
  private lastMessageIds = new Map<string, string>();
  private lastRoomIds = "";

  constructor(
    private client: ChatClient,
    private messageInterval = 3000,
    private roomInterval = 30000,
  ) {}

  subscribeMessages(chatId: string, onMessages: (msgs: Message[]) => void): Unsubscribe {
    const key = `msg:${chatId}`;
    this.clearTimer(key);

    const poll = async () => {
      try {
        const msgs = await this.client.listMessages(chatId, { limit: 50 });
        // Only notify if messages actually changed
        const fingerprint = msgs.map((m) => m.id).join(",");
        if (fingerprint !== this.lastMessageIds.get(chatId)) {
          this.lastMessageIds.set(chatId, fingerprint);
          onMessages(msgs);
        }
      } catch {
        // Swallow — TUI handles via status bar
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
        const rooms = await this.client.listChats();
        const dms = await this.client.listDms();
        const all = [...rooms, ...dms];
        // Only notify if rooms actually changed
        const fingerprint = all.map((r) => r.id).join(",");
        if (fingerprint !== this.lastRoomIds) {
          this.lastRoomIds = fingerprint;
          onRooms(all);
        }
      } catch {
        // Swallow
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

  resetFingerprint(chatId: string) {
    this.lastMessageIds.delete(chatId);
  }

  destroy() {
    for (const key of this.timers.keys()) this.clearTimer(key);
  }
}
