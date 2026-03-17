import { describe, it } from "node:test";
import assert from "node:assert/strict";
import { createChatStore } from "../src/store/chat.ts";

describe("ChatStore", () => {
  it("sets active room", () => {
    const store = createChatStore();
    store.getState().setActiveRoom("chat_1");
    assert.equal(store.getState().activeRoomId, "chat_1");
  });

  it("adds messages deduplicating by ID", () => {
    const store = createChatStore();
    store.getState().setMessages("chat_1", [
      { id: "m_1", chat: "chat_1", actor: "u/alice", text: "hi", created_at: "2026-03-17T00:00:00Z" },
    ]);
    store.getState().setMessages("chat_1", [
      { id: "m_1", chat: "chat_1", actor: "u/alice", text: "hi", created_at: "2026-03-17T00:00:00Z" },
      { id: "m_2", chat: "chat_1", actor: "u/bob", text: "hey", created_at: "2026-03-17T00:01:00Z" },
    ]);
    assert.equal(store.getState().messages["chat_1"].length, 2);
  });

  it("sorts messages by created_at", () => {
    const store = createChatStore();
    store.getState().setMessages("chat_1", [
      { id: "m_2", chat: "chat_1", actor: "u/bob", text: "hey", created_at: "2026-03-17T00:01:00Z" },
      { id: "m_1", chat: "chat_1", actor: "u/alice", text: "hi", created_at: "2026-03-17T00:00:00Z" },
    ]);
    const msgs = store.getState().messages["chat_1"];
    assert.equal(msgs[0].id, "m_1");
    assert.equal(msgs[1].id, "m_2");
  });

  it("derives members from messages", () => {
    const store = createChatStore();
    store.getState().setMessages("chat_1", [
      { id: "m_1", chat: "chat_1", actor: "u/alice", text: "hi", created_at: "2026-03-17T00:00:00Z" },
      { id: "m_2", chat: "chat_1", actor: "u/bob", text: "hey", created_at: "2026-03-17T00:01:00Z" },
      { id: "m_3", chat: "chat_1", actor: "u/alice", text: "yo", created_at: "2026-03-17T00:02:00Z" },
    ]);
    const members = store.getState().membersFor("chat_1");
    assert.deepEqual(members.sort(), ["u/alice", "u/bob"]);
  });

  it("sets rooms", () => {
    const store = createChatStore();
    store.getState().setRooms([
      { id: "chat_1", kind: "room", title: "general", creator: "u/alice", created_at: "2026-03-17T00:00:00Z" },
    ]);
    assert.equal(store.getState().rooms.length, 1);
  });
});
