import React from "react";
import { Box, Text } from "ink";

interface Props {
  title: string;
  children: React.ReactNode;
  width?: number;
}

export function Overlay({ title, children, width = 50 }: Props) {
  return (
    <Box
      flexDirection="column"
      borderStyle="round"
      borderColor="cyan"
      width={width}
      paddingX={1}
      paddingY={0}
    >
      <Box marginBottom={1}>
        <Text bold color="cyan">{title}</Text>
      </Box>
      {children}
    </Box>
  );
}
