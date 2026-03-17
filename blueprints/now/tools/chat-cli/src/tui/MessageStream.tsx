import React, { useState, useEffect } from "react";
import { Box, Text, useInput } from "ink";
import type { Message } from "../api/types.js";
import { actorColor, formatTime } from "../utils/format.js";
import { Markdown } from "./Markdown.js";

interface Props {
  messages: Message[];
  currentActor: string;
  active: boolean;
}

export function MessageStream({ messages, currentActor, active }: Props) {
  const [scrollOffset, setScrollOffset] = useState(0);

  useEffect(() => {
    setScrollOffset(0);
  }, [messages.length]);

  useInput((input, key) => {
    if (!active) return;
    if (key.upArrow) setScrollOffset((o) => Math.min(messages.length - 1, o + 1));
    if (key.downArrow) setScrollOffset((o) => Math.max(0, o - 1));
    if (key.pageUp) setScrollOffset((o) => Math.min(messages.length - 1, o + 10));
    if (key.pageDown) setScrollOffset((o) => Math.max(0, o - 10));
  });

  if (messages.length === 0) {
    return (
      <Box flexDirection="column" alignItems="center" justifyContent="center" flexGrow={1} paddingY={2}>
        <Text dimColor>No messages yet. Say something!</Text>
      </Box>
    );
  }

  const visible = messages.slice(
    Math.max(0, messages.length - 40 - scrollOffset),
    Math.max(0, messages.length - scrollOffset),
  );

  return (
    <Box flexDirection="column" flexGrow={1} paddingX={2} paddingY={1}>
      {scrollOffset > 0 && (
        <Text dimColor italic>  ↑ {scrollOffset} more messages above</Text>
      )}
      {visible.map((msg) => {
        const isMe = msg.actor === currentActor;
        const color = actorColor(msg.actor);
        return (
          <Box key={msg.id} flexDirection="column" marginBottom={1}>
            <Box justifyContent="space-between">
              <Text color={color} bold={isMe}>{msg.actor}</Text>
              <Text dimColor>{formatTime(msg.created_at)}</Text>
            </Box>
            <Box paddingLeft={2}>
              <Markdown text={msg.text} />
            </Box>
          </Box>
        );
      })}
    </Box>
  );
}
