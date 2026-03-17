import React, { useState } from "react";
import { Box, Text, useInput } from "ink";

interface Props {
  focused: boolean;
  onSubmit: (text: string) => void;
}

export function InputBar({ focused, onSubmit }: Props) {
  const [value, setValue] = useState("");

  useInput(
    (input, key) => {
      if (!focused) return;

      if (key.return && value.trim()) {
        onSubmit(value.trim());
        setValue("");
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
    },
  );

  return (
    <Box paddingX={1}>
      <Text color={focused ? "green" : "gray"}>{"› "}</Text>
      <Text>{value}</Text>
      {focused && <Text color="green">▎</Text>}
      {!focused && !value && <Text dimColor>press Tab to focus</Text>}
    </Box>
  );
}
