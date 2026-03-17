import React, { useState } from "react";
import { Box, Text, useInput } from "ink";
import type { Chat } from "../api/types.js";
import { roomLabel } from "../utils/format.js";

interface Props {
  rooms: Chat[];
  activeId: string | null;
  focused: boolean;
  onSelect: (id: string) => void;
}

export function RoomList({ rooms, activeId, focused, onSelect }: Props) {
  const [cursor, setCursor] = useState(0);

  useInput(
    (input, key) => {
      if (!focused) return;
      if (key.upArrow) setCursor((c) => Math.max(0, c - 1));
      if (key.downArrow) setCursor((c) => Math.min(rooms.length - 1, c + 1));
      if (key.return && rooms[cursor]) onSelect(rooms[cursor].id);
    },
  );

  if (rooms.length === 0) {
    return (
      <Box flexDirection="column" padding={1}>
        <Text dimColor>No rooms</Text>
        <Text dimColor>^N create</Text>
      </Box>
    );
  }

  return (
    <Box flexDirection="column">
      <Box paddingX={1}>
        <Text bold color={focused ? "cyan" : undefined}>Rooms</Text>
      </Box>
      {rooms.map((room, i) => {
        const isActive = room.id === activeId;
        const isCursor = i === cursor && focused;
        return (
          <Box key={room.id} paddingX={1}>
            <Text
              bold={isActive}
              inverse={isCursor}
              color={isActive ? "cyan" : undefined}
            >
              {roomLabel(room)}
            </Text>
          </Box>
        );
      })}
    </Box>
  );
}
