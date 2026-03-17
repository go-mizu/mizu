import React, { useEffect, useCallback, useRef, useState } from "react";
import { render, Box, Text, useApp, useInput, useStdout } from "ink";
import type { Config } from "../auth/config.js";
import { signRequest, base64urlDecode } from "../auth/signer.js";
import { ChatClient } from "../api/client.js";
import { PollingTransport } from "../api/transport.js";
import { createChatStore, type ChatState, type Panel } from "../store/chat.js";
import { RoomList } from "./RoomList.js";
import { MessageView } from "./MessageView.js";
import { MemberList } from "./MemberList.js";
import { InputBar } from "./InputBar.js";
import { StatusBar } from "./StatusBar.js";
import { Prompt } from "./Prompt.js";
import { roomLabel } from "../utils/format.js";

type PromptMode = null | "create" | "join";

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
  const [promptMode, setPromptMode] = useState<PromptMode>(null);

  useEffect(() => {
    return store.subscribe(setState);
  }, [store]);

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

  // Global keybindings
  useInput((input, key) => {
    if (promptMode) return; // Let prompt handle input
    if (key.ctrl && input === "q") { exit(); return; }
    if (key.ctrl && input === "c") { exit(); return; }
    if (key.ctrl && input === "n") { setPromptMode("create"); return; }
    if (key.ctrl && input === "j") { setPromptMode("join"); return; }
    if (key.tab && key.shift) {
      const panels: Panel[] = ["input", "rooms", "messages", "members"];
      const idx = panels.indexOf(state.focusedPanel);
      store.getState().setFocus(panels[(idx - 1 + panels.length) % panels.length]);
      return;
    }
    if (key.tab) { store.getState().cycleFocus(); return; }
  });

  const activeRoom = state.rooms.find((r) => r.id === state.activeRoomId);
  const activeMessages = state.activeRoomId ? state.messages[state.activeRoomId] || [] : [];
  const members = state.activeRoomId ? state.membersFor(state.activeRoomId) : [];

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
    store.getState().setFocus("input");
  }, []);

  const handlePromptCreate = useCallback(async (title: string) => {
    setPromptMode(null);
    if (!clientRef.current) return;
    try {
      const chat = await clientRef.current.createChat({ title });
      store.getState().setActiveRoom(chat.id);
    } catch (e: unknown) {
      store.getState().setError(e instanceof Error ? e.message : String(e));
    }
  }, []);

  const handlePromptJoin = useCallback(async (id: string) => {
    setPromptMode(null);
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
      {/* Three-panel layout */}
      <Box flexGrow={1}>
        {/* Room list */}
        <Box width={20} flexShrink={0} borderStyle="single" borderRight borderLeft={false} borderTop={false} borderBottom={false}>
          <RoomList
            rooms={state.rooms}
            activeId={state.activeRoomId}
            focused={state.focusedPanel === "rooms"}
            onSelect={handleSelectRoom}
          />
        </Box>
        {/* Messages */}
        <Box flexGrow={1} flexDirection="column">
          <MessageView
            messages={activeMessages}
            currentActor={config.actor}
            focused={state.focusedPanel === "messages"}
          />
        </Box>
        {/* Member list */}
        <Box width={18} flexShrink={0} borderStyle="single" borderLeft borderRight={false} borderTop={false} borderBottom={false}>
          <MemberList
            members={members}
            currentActor={config.actor}
            focused={state.focusedPanel === "members"}
          />
        </Box>
      </Box>

      {/* Input or Prompt */}
      <Box borderStyle="single" borderTop borderBottom={false} borderLeft={false} borderRight={false}>
        {promptMode === "create" ? (
          <Prompt label="Room title" onSubmit={handlePromptCreate} onCancel={() => setPromptMode(null)} />
        ) : promptMode === "join" ? (
          <Prompt label="Room ID" onSubmit={handlePromptJoin} onCancel={() => setPromptMode(null)} />
        ) : (
          <InputBar focused={state.focusedPanel === "input"} onSubmit={handleSend} />
        )}
      </Box>

      {/* Status bar */}
      <StatusBar
        actor={config.actor}
        room={activeRoom ? roomLabel(activeRoom) : null}
        memberCount={members.length}
        connected={state.connected}
        error={state.error}
      />
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
