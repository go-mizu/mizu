import React from "react";
import { Box, Text } from "ink";

interface Props {
  actor: string;
  room: string | null;
  memberCount: number;
  connected: boolean;
  error: string | null;
}

export function StatusBar({ actor, room, memberCount, connected, error }: Props) {
  return (
    <Box paddingX={1}>
      <Text dimColor>{actor}</Text>
      {room && <Text dimColor> · {room}</Text>}
      <Text dimColor> · {memberCount} members</Text>
      <Text dimColor> · </Text>
      {error ? (
        <Text color="red">{error}</Text>
      ) : connected ? (
        <Text color="green">connected</Text>
      ) : (
        <Text color="yellow">connecting...</Text>
      )}
      <Text dimColor>  [Tab]focus [^N]new [^J]join [^Q]quit</Text>
    </Box>
  );
}
