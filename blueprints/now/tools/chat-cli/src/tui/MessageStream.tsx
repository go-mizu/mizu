import React, { useState, useEffect, useMemo } from "react";
import { Box, Text, useInput } from "ink";
import type { Message } from "../api/types.js";
import { actorColor, formatTime } from "../utils/format.js";
import { Markdown } from "./Markdown.js";

interface Props {
  messages: Message[];
  currentActor: string;
  active: boolean;
}

const MemoMessage = React.memo(function MemoMessage({
  msg,
  isMe,
}: {
  msg: Message;
  isMe: boolean;
}) {
  const color = actorColor(msg.actor);
  return (
    <Box flexDirection="column" marginBottom={1}>
      <Box justifyContent="space-between">
        <Text color={color} bold={isMe}>{msg.actor}</Text>
        <Text dimColor>{formatTime(msg.created_at)}</Text>
      </Box>
      <Box paddingLeft={2}>
        <Markdown text={msg.text} />
      </Box>
    </Box>
  );
});

export const MessageStream = React.memo(function MessageStream({
  messages,
  currentActor,
  active,
}: Props) {
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

  const visible = useMemo(
    () =>
      messages.slice(
        Math.max(0, messages.length - 40 - scrollOffset),
        Math.max(0, messages.length - scrollOffset),
      ),
    [messages, scrollOffset],
  );

  if (messages.length === 0) {
    return (
      <Box flexDirection="column" alignItems="center" justifyContent="center" flexGrow={1} paddingY={2}>
        <Text dimColor>No messages yet. Say something!</Text>
      </Box>
    );
  }

  return (
    <Box flexDirection="column" flexGrow={1} paddingX={2} paddingY={1}>
      {scrollOffset > 0 && (
        <Text dimColor italic>  ↑ {scrollOffset} more messages above</Text>
      )}
      {visible.map((msg) => (
        <MemoMessage key={msg.id} msg={msg} isMe={msg.actor === currentActor} />
      ))}
    </Box>
  );
});
