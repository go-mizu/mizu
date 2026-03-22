import React, { useState } from "react";
import { Box, Text, useInput } from "ink";
import { TextInput } from "./TextInput.js";
import type { Chat } from "../api/types.js";
import { roomLabel } from "../utils/format.js";
import { Overlay } from "./Overlay.js";

interface Props {
  rooms: Chat[];
  activeId: string | null;
  onSelect: (id: string) => void;
  onCancel: () => void;
}

export function RoomSwitcher({ rooms, activeId, onSelect, onCancel }: Props) {
  const [query, setQuery] = useState("");
  const [cursor, setCursor] = useState(0);

  const filtered = rooms.filter((r) => {
    const label = roomLabel(r).toLowerCase();
    return label.includes(query.toLowerCase());
  });

  useInput((input, key) => {
    if (key.escape) {
      onCancel();
      return;
    }
    if (key.upArrow) {
      setCursor((c) => Math.max(0, c - 1));
      return;
    }
    if (key.downArrow) {
      setCursor((c) => Math.min(filtered.length - 1, c + 1));
      return;
    }
  });

  const handleSubmit = () => {
    if (filtered[cursor]) {
      onSelect(filtered[cursor].id);
    }
  };

  return (
    <Overlay title="Switch Room" width={40}>
      <Box gap={1} marginBottom={1}>
        <Text color="cyan">{"›"}</Text>
        <TextInput
          value={query}
          onChange={(v) => { setQuery(v); setCursor(0); }}
          onSubmit={handleSubmit}
          placeholder="filter..."
        />
      </Box>
      <Box flexDirection="column">
        {filtered.length === 0 ? (
          <Text dimColor>  no matches</Text>
        ) : (
          filtered.slice(0, 10).map((room, i) => {
            const isActive = room.id === activeId;
            const isCursor = i === cursor;
            const label = roomLabel(room);
            return (
              <Box key={room.id} paddingX={1}>
                <Text
                  inverse={isCursor}
                  bold={isActive}
                  color={isCursor ? "cyan" : isActive ? "white" : undefined}
                >
                  {isCursor ? "› " : "  "}{label}
                </Text>
              </Box>
            );
          })
        )}
      </Box>
      <Box marginTop={1}>
        <Text dimColor>  ↑↓ navigate · enter select · esc cancel</Text>
      </Box>
    </Overlay>
  );
}
