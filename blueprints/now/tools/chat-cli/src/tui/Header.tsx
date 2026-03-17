import React from "react";
import { Box, Text } from "ink";

interface Props {
  room: string | null;
  actor: string;
  connected: boolean;
  error: string | null;
}

export const Header = React.memo(function Header({ room, actor, connected, error }: Props) {
  return (
    <Box
      borderStyle="round"
      borderColor="gray"
      paddingX={1}
      justifyContent="space-between"
    >
      <Box gap={1}>
        <Text color="magenta" bold>{"✦ chat-now"}</Text>
        {!connected && !error && <Text color="yellow">•••</Text>}
      </Box>
      <Box gap={2}>
        {room && <Text bold color="white">{room}</Text>}
        {error ? (
          <Text color="red">{error}</Text>
        ) : connected ? (
          <Text dimColor>{actor}</Text>
        ) : (
          <Text color="yellow">connecting</Text>
        )}
      </Box>
    </Box>
  );
});
