import type { ChatClient } from "./client.js";
import type { Message, Chat } from "./types.js";

export type Unsubscribe = () => void;

export interface Transport {
  subscribeMessages(chatId: string, onMessages: (msgs: Message[]) => void): Unsubscribe;
  subscribeRooms(onRooms: (rooms: Chat[]) => void): Unsubscribe;
}

export class PollingTransport implements Transport {
  private timers = new Map<string, ReturnType<typeof setInterval>>();

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
        onMessages(msgs);
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
        onRooms([...rooms, ...dms]);
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

  destroy() {
    for (const key of this.timers.keys()) this.clearTimer(key);
  }
}
