import React, { useState } from "react";
import { Box, Text, useInput } from "ink";

interface Props {
  label: string;
  onSubmit: (value: string) => void;
  onCancel: () => void;
}

export function Prompt({ label, onSubmit, onCancel }: Props) {
  const [value, setValue] = useState("");

  useInput((input, key) => {
    if (key.escape) {
      onCancel();
      return;
    }
    if (key.return && value.trim()) {
      onSubmit(value.trim());
      return;
    }
    if (key.backspace || key.delete) {
      setValue((v) => v.slice(0, -1));
      return;
    }
    if (key.ctrl || key.meta) return;
    if (key.upArrow || key.downArrow || key.leftArrow || key.rightArrow || key.tab) return;

    if (input) {
      setValue((v) => v + input);
    }
  });

  return (
    <Box paddingX={1}>
      <Text color="yellow">{label}: </Text>
      <Text>{value}</Text>
      <Text color="yellow">▎</Text>
      <Text dimColor>  (Enter to confirm, Esc to cancel)</Text>
    </Box>
  );
}
