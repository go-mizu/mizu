import React, { useState } from "react";
import { Box, Text, useInput } from "ink";
import { TextInput } from "./TextInput.js";

interface Props {
  active: boolean;
  onSubmit: (text: string) => void;
}

export const Composer = React.memo(function Composer({ active, onSubmit }: Props) {
  const [value, setValue] = useState("");

  useInput((input, key) => {
    if (!active) return;

    // Escape: clear
    if (key.escape) {
      setValue("");
      return;
    }
  });

  const handleSubmit = (val: string) => {
    const text = val.trim();
    if (text) {
      onSubmit(text);
      setValue("");
    }
  };

  return (
    <Box flexDirection="column">
      <Box paddingX={2}>
        <Text dimColor>{"─".repeat(60)}</Text>
      </Box>
      <Box paddingX={2}>
        <Text color={active ? "green" : "gray"}>{"› "}</Text>
        <TextInput
          value={value}
          onChange={setValue}
          onSubmit={handleSubmit}
          placeholder="type a message..."
          isActive={active}
        />
      </Box>
    </Box>
  );
});
