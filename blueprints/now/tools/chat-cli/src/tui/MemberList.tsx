import React from "react";
import { Box, Text } from "ink";
import { actorColor } from "../utils/format.js";

interface Props {
  members: string[];
  currentActor: string;
  focused: boolean;
}

export function MemberList({ members, currentActor, focused }: Props) {
  return (
    <Box flexDirection="column">
      <Box paddingX={1}>
        <Text bold color={focused ? "cyan" : undefined}>Members</Text>
      </Box>
      {members.length === 0 ? (
        <Box paddingX={1}>
          <Text dimColor>—</Text>
        </Box>
      ) : (
        members.map((actor) => (
          <Box key={actor} paddingX={1}>
            <Text color={actorColor(actor)} bold={actor === currentActor}>
              {actor}
            </Text>
          </Box>
        ))
      )}
    </Box>
  );
}
