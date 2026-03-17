import React, { useState, useEffect } from "react";
import { Box, Text, useInput } from "ink";
import type { Message } from "../api/types.js";
import { actorColor, formatTime } from "../utils/format.js";

interface Props {
  messages: Message[];
  currentActor: string;
  focused: boolean;
}

export function MessageView({ messages, currentActor, focused }: Props) {
  const [scrollOffset, setScrollOffset] = useState(0);

  useEffect(() => {
    setScrollOffset(0);
  }, [messages.length]);

  useInput((input, key) => {
    if (!focused) return;
    if (key.upArrow) setScrollOffset((o) => Math.min(messages.length - 1, o + 1));
    if (key.downArrow) setScrollOffset((o) => Math.max(0, o - 1));
    if (key.pageUp) setScrollOffset((o) => Math.min(messages.length - 1, o + 10));
    if (key.pageDown) setScrollOffset((o) => Math.max(0, o - 10));
  });

  if (messages.length === 0) {
    return (
      <Box flexDirection="column" justifyContent="center" alignItems="center" flexGrow={1}>
        <Text dimColor>No messages yet</Text>
      </Box>
    );
  }

  const visibleMessages = messages.slice(
    Math.max(0, messages.length - 30 - scrollOffset),
    Math.max(0, messages.length - scrollOffset),
  );

  return (
    <Box flexDirection="column" paddingX={1} flexGrow={1}>
      {visibleMessages.map((msg) => {
        const isMe = msg.actor === currentActor;
        const color = actorColor(msg.actor);
        return (
          <Box key={msg.id} flexDirection="column">
            <Box gap={1}>
              <Text color={color} bold={isMe}>
                {msg.actor}
              </Text>
              <Text dimColor>{formatTime(msg.created_at)}</Text>
            </Box>
            <Box paddingLeft={2}>
              <Text>{msg.text}</Text>
            </Box>
          </Box>
        );
      })}
      {scrollOffset > 0 && (
        <Text dimColor italic>↓ {scrollOffset} more below</Text>
      )}
    </Box>
  );
}
