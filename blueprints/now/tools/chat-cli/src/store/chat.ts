import { createStore } from "zustand/vanilla";
import type { Chat, Message } from "../api/types.js";

export interface ChatState {
  rooms: Chat[];
  activeRoomId: string | null;
  setActiveRoom: (id: string) => void;
  setRooms: (rooms: Chat[]) => void;

  messages: Record<string, Message[]>;
  setMessages: (chatId: string, msgs: Message[]) => void;

  membersFor: (chatId: string) => string[];

  connected: boolean;
  error: string | null;
  setConnected: (v: boolean) => void;
  setError: (e: string | null) => void;
}

export function createChatStore() {
  return createStore<ChatState>((set, get) => ({
    rooms: [],
    activeRoomId: null,
    messages: {},
    connected: false,
    error: null,

    setActiveRoom: (id) => set({ activeRoomId: id }),

    setRooms: (rooms) => set({ rooms }),

    setMessages: (chatId, msgs) =>
      set((state) => {
        const existing = state.messages[chatId] || [];
        const seen = new Set(existing.map((m) => m.id));
        const merged = [...existing];
        for (const m of msgs) {
          if (!seen.has(m.id)) {
            merged.push(m);
            seen.add(m.id);
          }
        }
        merged.sort((a, b) => a.created_at.localeCompare(b.created_at));
        return { messages: { ...state.messages, [chatId]: merged } };
      }),

    membersFor: (chatId) => {
      const msgs = get().messages[chatId] || [];
      return [...new Set(msgs.map((m) => m.actor))];
    },

    setConnected: (v) => set({ connected: v }),
    setError: (e) => set({ error: e }),
  }));
}
