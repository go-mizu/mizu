import { createStore } from "zustand/vanilla";
import type { Chat, Message } from "../api/types.js";

export interface ChatState {
  rooms: Chat[];
  activeRoomId: string | null;
  setActiveRoom: (id: string) => void;
  setRooms: (rooms: Chat[]) => void;

  messages: Record<string, Message[]>;
  setMessages: (chatId: string, msgs: Message[]) => void;
  replaceOptimistic: (chatId: string, optimisticId: string, real: Message) => void;

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

    setActiveRoom: (id) => {
      if (get().activeRoomId === id) return;
      set({ activeRoomId: id });
    },

    setRooms: (rooms) => {
      const prev = get().rooms;
      // Skip if room list hasn't changed (compare IDs)
      if (prev.length === rooms.length && prev.every((r, i) => r.id === rooms[i].id)) return;
      set({ rooms });
    },

    setMessages: (chatId, msgs) => {
      const state = get();
      const existing = state.messages[chatId] || [];
      const seen = new Set(existing.map((m) => m.id));
      let added = false;
      for (const m of msgs) {
        if (!seen.has(m.id)) { added = true; break; }
      }
      // No new messages — skip state update entirely
      if (!added) return;

      const merged = [...existing];
      for (const m of msgs) {
        if (!seen.has(m.id)) {
          merged.push(m);
          seen.add(m.id);
        }
      }
      merged.sort((a, b) => a.created_at.localeCompare(b.created_at));
      set({ messages: { ...state.messages, [chatId]: merged } });
    },

    replaceOptimistic: (chatId, optimisticId, real) =>
      set((state) => {
        const existing = state.messages[chatId] || [];
        // If optimistic message already gone (e.g. poll replaced it), skip
        if (!existing.some((m) => m.id === optimisticId)) return state;
        const msgs = existing.map((m) => m.id === optimisticId ? real : m);
        return { messages: { ...state.messages, [chatId]: msgs } };
      }),

    membersFor: (chatId) => {
      const msgs = get().messages[chatId] || [];
      return [...new Set(msgs.map((m) => m.actor))];
    },

    setConnected: (v) => {
      if (get().connected === v) return;
      set({ connected: v });
    },
    setError: (e) => {
      if (get().error === e) return;
      set({ error: e });
    },
  }));
}
