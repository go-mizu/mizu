import { createStore } from "zustand/vanilla";
import type { Chat, Message } from "../api/types.js";

export interface ChatState {
  rooms: Chat[];
  activeRoomId: string | null;
  setActiveRoom: (id: string) => void;

  messages: Record<string, Message[]>;
  setMessages: (chatId: string, msgs: Message[]) => void;
  replaceOptimistic: (chatId: string, optimisticId: string, real: Message) => void;

  membersFor: (chatId: string) => string[];

  connected: boolean;
  error: string | null;

  /** Single set() call for the rooms-poll callback — prevents multi-render. */
  applyRoomsPoll: (rooms: Chat[]) => void;
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

    applyRoomsPoll: (rooms) => {
      const s = get();
      const roomsSame =
        s.rooms.length === rooms.length &&
        s.rooms.every((r, i) => r.id === rooms[i].id);
      const alreadyConnected = s.connected;
      const alreadyNoError = s.error === null;

      // Nothing changed — skip set() entirely
      if (roomsSame && alreadyConnected && alreadyNoError) return;

      // Batch all changes into one set() → one subscriber notification → one Ink render
      const patch: Partial<ChatState> = {};
      if (!roomsSame) patch.rooms = rooms;
      if (!alreadyConnected) patch.connected = true;
      if (!alreadyNoError) patch.error = null;
      if (!s.activeRoomId && rooms.length > 0) patch.activeRoomId = rooms[0].id;
      set(patch);
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
        if (!existing.some((m) => m.id === optimisticId)) return state;
        const msgs = existing.map((m) => m.id === optimisticId ? real : m);
        return { messages: { ...state.messages, [chatId]: msgs } };
      }),

    membersFor: (chatId) => {
      const msgs = get().messages[chatId] || [];
      return [...new Set(msgs.map((m) => m.actor))];
    },

    setError: (e) => {
      if (get().error === e) return;
      set({ error: e });
    },
  }));
}
