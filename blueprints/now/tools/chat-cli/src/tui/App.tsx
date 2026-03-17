import React, { useEffect, useCallback, useRef, useState, useSyncExternalStore } from "react";
import { render, Box, useApp, useInput, useStdout } from "ink";
import TextInput from "ink-text-input";
import type { Config } from "../auth/config.js";
import { signRequest, base64urlDecode } from "../auth/signer.js";
import { ChatClient } from "../api/client.js";
import { PollingTransport } from "../api/transport.js";
import { createChatStore, type ChatState } from "../store/chat.js";
import { Header } from "./Header.js";
import { MessageStream } from "./MessageStream.js";
import { Composer } from "./Composer.js";
import { RoomSwitcher } from "./RoomSwitcher.js";
import { Overlay } from "./Overlay.js";
import { roomLabel } from "../utils/format.js";

type OverlayMode = null | "switcher" | "create" | "join";

interface AppProps {
  config: Config;
  serverOverride?: string;
}

// Selector hook — only re-renders when selected value changes
function useStore<T>(
  store: ReturnType<typeof createChatStore>,
  selector: (s: ChatState) => T,
): T {
  return useSyncExternalStore(
    store.subscribe,
    () => selector(store.getState()),
  );
}

function App({ config, serverOverride }: AppProps) {
  const { exit } = useApp();
  const { stdout } = useStdout();
  const storeRef = useRef(createChatStore());
  const store = storeRef.current;

  // Granular subscriptions — each only re-renders when its value changes
  const rooms = useStore(store, (s) => s.rooms);
  const activeRoomId = useStore(store, (s) => s.activeRoomId);
  const connected = useStore(store, (s) => s.connected);
  const error = useStore(store, (s) => s.error);
  const allMessages = useStore(store, (s) => s.messages);

  const activeMessages = activeRoomId ? allMessages[activeRoomId] || [] : [];
  const activeRoom = rooms.find((r) => r.id === activeRoomId);

  const [overlay, setOverlay] = useState<OverlayMode>(null);
  const [overlayInput, setOverlayInput] = useState("");

  // Client + transport setup
  const clientRef = useRef<ChatClient | null>(null);
  const transportRef = useRef<PollingTransport | null>(null);

  useEffect(() => {
    const cfg = serverOverride ? { ...config, server: serverOverride } : config;
    const signer = (method: string, path: string, query: string, body: string) =>
      signRequest({
        actor: cfg.actor,
        privateKey: base64urlDecode(cfg.private_key),
        method,
        path,
        query,
        body,
      });
    const client = new ChatClient(cfg, signer);
    clientRef.current = client;

    const transport = new PollingTransport(client);
    transportRef.current = transport;

    // First poll always fires; subsequent polls only fire if data changed
    const unsubRooms = transport.subscribeRooms((newRooms) => {
      store.getState().setRooms(newRooms);
      store.getState().setConnected(true);
      store.getState().setError(null);
      if (!store.getState().activeRoomId && newRooms.length > 0) {
        store.getState().setActiveRoom(newRooms[0].id);
      }
    });

    return () => {
      unsubRooms();
      transport.destroy();
    };
  }, [config, serverOverride]);

  // Subscribe to active room messages
  const unsubMsgRef = useRef<(() => void) | null>(null);
  useEffect(() => {
    if (unsubMsgRef.current) unsubMsgRef.current();
    if (!activeRoomId || !transportRef.current) return;

    const chatId = activeRoomId;
    unsubMsgRef.current = transportRef.current.subscribeMessages(chatId, (msgs) => {
      store.getState().setMessages(chatId, msgs);
    });

    return () => {
      if (unsubMsgRef.current) unsubMsgRef.current();
    };
  }, [activeRoomId]);

  // Room cycling helper
  const cycleRoom = useCallback((direction: number) => {
    const { rooms: r, activeRoomId: id } = store.getState();
    if (r.length === 0) return;
    const idx = r.findIndex((room) => room.id === id);
    const next = (idx + direction + r.length) % r.length;
    store.getState().setActiveRoom(r[next].id);
  }, []);

  // Global keybindings
  useInput((input, key) => {
    if (overlay) return;

    if (key.ctrl && input === "q") { exit(); return; }
    if (key.ctrl && input === "c") { exit(); return; }
    if (key.ctrl && input === "k") { setOverlay("switcher"); return; }
    if (key.ctrl && input === "n") { setOverlay("create"); setOverlayInput(""); return; }
    if (key.ctrl && input === "j") { setOverlay("join"); setOverlayInput(""); return; }
    if (key.ctrl && key.leftArrow) { cycleRoom(-1); return; }
    if (key.ctrl && key.rightArrow) { cycleRoom(1); return; }
  });

  const handleSend = useCallback(async (text: string) => {
    if (!clientRef.current || !activeRoomId) return;
    const chatId = activeRoomId;
    try {
      // Auto-join first (idempotent if already member)
      try { await clientRef.current.joinChat(chatId); } catch { /* ignore */ }

      // Optimistic insert — show message immediately
      const optimistic = {
        id: `opt_${Date.now()}`,
        chat: chatId,
        actor: config.actor,
        text,
        created_at: new Date().toISOString(),
      };
      store.getState().setMessages(chatId, [optimistic]);

      // Actually send
      const msg = await clientRef.current.sendMessage(chatId, text);

      // Replace optimistic with real message
      store.getState().replaceOptimistic(chatId, optimistic.id, msg);

      // Reset transport fingerprint so next poll doesn't flicker
      if (transportRef.current) {
        transportRef.current.resetFingerprint(chatId);
      }
    } catch (e: unknown) {
      store.getState().setError(e instanceof Error ? e.message : String(e));
    }
  }, [activeRoomId, config.actor]);

  const handleSelectRoom = useCallback((id: string) => {
    store.getState().setActiveRoom(id);
    setOverlay(null);
  }, []);

  const handleCreateRoom = useCallback(async (title: string) => {
    setOverlay(null);
    if (!clientRef.current) return;
    try {
      const chat = await clientRef.current.createChat({ title });
      store.getState().setActiveRoom(chat.id);
    } catch (e: unknown) {
      store.getState().setError(e instanceof Error ? e.message : String(e));
    }
  }, []);

  const handleJoinRoom = useCallback(async (id: string) => {
    setOverlay(null);
    if (!clientRef.current) return;
    try {
      await clientRef.current.joinChat(id);
      store.getState().setActiveRoom(id);
    } catch (e: unknown) {
      store.getState().setError(e instanceof Error ? e.message : String(e));
    }
  }, []);

  const height = stdout?.rows || 24;

  return (
    <Box flexDirection="column" height={height} width="100%">
      <Header
        room={activeRoom ? roomLabel(activeRoom) : null}
        actor={config.actor}
        connected={connected}
        error={error}
      />

      <MessageStream
        messages={activeMessages}
        currentActor={config.actor}
        active={!overlay}
      />

      <Composer active={!overlay} onSubmit={handleSend} />

      {overlay === "switcher" && (
        <Box position="absolute" marginTop={3} marginLeft={4}>
          <RoomSwitcher
            rooms={rooms}
            activeId={activeRoomId}
            onSelect={handleSelectRoom}
            onCancel={() => setOverlay(null)}
          />
        </Box>
      )}

      {overlay === "create" && (
        <Box position="absolute" marginTop={3} marginLeft={4}>
          <Overlay title="Create Room">
            <Box gap={1}>
              <TextInput
                value={overlayInput}
                onChange={setOverlayInput}
                onSubmit={handleCreateRoom}
                placeholder="room title..."
                showCursor
              />
            </Box>
          </Overlay>
        </Box>
      )}

      {overlay === "join" && (
        <Box position="absolute" marginTop={3} marginLeft={4}>
          <Overlay title="Join Room">
            <Box gap={1}>
              <TextInput
                value={overlayInput}
                onChange={setOverlayInput}
                onSubmit={handleJoinRoom}
                placeholder="chat_..."
                showCursor
              />
            </Box>
          </Overlay>
        </Box>
      )}
    </Box>
  );
}

export async function launchTui(config: Config, serverOverride?: string) {
  const { waitUntilExit } = render(
    <App config={config} serverOverride={serverOverride} />,
    { exitOnCtrlC: false },
  );
  await waitUntilExit();
}
