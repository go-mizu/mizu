import React, { useEffect, useCallback, useRef, useState } from "react";
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

function App({ config, serverOverride }: AppProps) {
  const { exit } = useApp();
  const { stdout } = useStdout();
  const storeRef = useRef(createChatStore());
  const store = storeRef.current;
  const [state, setState] = useState<ChatState>(store.getState());
  const [overlay, setOverlay] = useState<OverlayMode>(null);
  const [overlayInput, setOverlayInput] = useState("");

  useEffect(() => {
    return store.subscribe(setState);
  }, [store]);

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

    const unsubRooms = transport.subscribeRooms((rooms) => {
      store.getState().setRooms(rooms);
      store.getState().setConnected(true);
      store.getState().setError(null);
      if (!store.getState().activeRoomId && rooms.length > 0) {
        store.getState().setActiveRoom(rooms[0].id);
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
    if (!state.activeRoomId || !transportRef.current) return;

    const chatId = state.activeRoomId;
    unsubMsgRef.current = transportRef.current.subscribeMessages(chatId, (msgs) => {
      store.getState().setMessages(chatId, msgs);
    });

    return () => {
      if (unsubMsgRef.current) unsubMsgRef.current();
    };
  }, [state.activeRoomId]);

  // Room cycling helper
  const cycleRoom = useCallback((direction: number) => {
    const { rooms, activeRoomId } = store.getState();
    if (rooms.length === 0) return;
    const idx = rooms.findIndex((r) => r.id === activeRoomId);
    const next = (idx + direction + rooms.length) % rooms.length;
    store.getState().setActiveRoom(rooms[next].id);
  }, []);

  // Global keybindings
  useInput((input, key) => {
    if (overlay) return; // Overlays handle their own input

    if (key.ctrl && input === "q") { exit(); return; }
    if (key.ctrl && input === "c") { exit(); return; }
    if (key.ctrl && input === "k") { setOverlay("switcher"); return; }
    if (key.ctrl && input === "n") { setOverlay("create"); setOverlayInput(""); return; }
    if (key.ctrl && input === "j") { setOverlay("join"); setOverlayInput(""); return; }
    if (key.ctrl && input === "r") {
      // Force refresh
      if (transportRef.current) {
        transportRef.current.destroy();
        const cfg = serverOverride ? { ...config, server: serverOverride } : config;
        const signer = (m: string, p: string, q: string, b: string) =>
          signRequest({ actor: cfg.actor, privateKey: base64urlDecode(cfg.private_key), method: m, path: p, query: q, body: b });
        const client = new ChatClient(cfg, signer);
        clientRef.current = client;
        const transport = new PollingTransport(client);
        transportRef.current = transport;
        transport.subscribeRooms((rooms) => {
          store.getState().setRooms(rooms);
          store.getState().setConnected(true);
        });
        if (state.activeRoomId) {
          transport.subscribeMessages(state.activeRoomId, (msgs) => {
            store.getState().setMessages(state.activeRoomId!, msgs);
          });
        }
      }
      return;
    }
    // Ctrl+Left / Ctrl+Right to cycle rooms
    if (key.ctrl && key.leftArrow) { cycleRoom(-1); return; }
    if (key.ctrl && key.rightArrow) { cycleRoom(1); return; }
  });

  const activeRoom = state.rooms.find((r) => r.id === state.activeRoomId);
  const activeMessages = state.activeRoomId ? state.messages[state.activeRoomId] || [] : [];

  const handleSend = useCallback(async (text: string) => {
    if (!clientRef.current || !state.activeRoomId) return;
    try {
      await clientRef.current.sendMessage(state.activeRoomId, text);
    } catch (e: unknown) {
      store.getState().setError(e instanceof Error ? e.message : String(e));
    }
  }, [state.activeRoomId]);

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
    <Box flexDirection="column" height={height}>
      {/* Header */}
      <Header
        room={activeRoom ? roomLabel(activeRoom) : null}
        actor={config.actor}
        connected={state.connected}
        error={state.error}
      />

      {/* Message stream */}
      <MessageStream
        messages={activeMessages}
        currentActor={config.actor}
        active={!overlay}
      />

      {/* Composer */}
      <Composer active={!overlay} onSubmit={handleSend} />

      {/* Overlays */}
      {overlay === "switcher" && (
        <Box position="absolute" marginTop={3} marginLeft={4}>
          <RoomSwitcher
            rooms={state.rooms}
            activeId={state.activeRoomId}
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
