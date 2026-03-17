import React, { useState } from "react";
import { Box, Text, useInput } from "ink";
import TextInput from "ink-text-input";

interface Props {
  active: boolean;
  onSubmit: (text: string) => void;
}

export function Composer({ active, onSubmit }: Props) {
  const [lines, setLines] = useState<string[]>([""]);
  const [activeLine, setActiveLine] = useState(0);

  useInput((input, key) => {
    if (!active) return;

    // Shift+Enter or Ctrl+Enter: new line
    if (key.return && (key.shift || key.ctrl)) {
      setLines((prev) => {
        const next = [...prev];
        next.splice(activeLine + 1, 0, "");
        return next;
      });
      setActiveLine((l) => l + 1);
      return;
    }

    // Escape: clear
    if (key.escape) {
      setLines([""]);
      setActiveLine(0);
      return;
    }
  });

  const handleSubmit = (value: string) => {
    // Update current line then send
    const all = [...lines];
    all[activeLine] = value;
    const text = all.join("\n").trim();
    if (text) {
      onSubmit(text);
      setLines([""]);
      setActiveLine(0);
    }
  };

  const handleChange = (value: string) => {
    setLines((prev) => {
      const next = [...prev];
      next[activeLine] = value;
      return next;
    });
  };

  return (
    <Box flexDirection="column">
      <Box paddingX={2}>
        <Text dimColor>{"─".repeat(60)}</Text>
      </Box>
      <Box flexDirection="column" paddingX={2} paddingY={0}>
        {lines.map((line, i) => (
          <Box key={i} gap={1}>
            <Text color={active ? "green" : "gray"}>{"›"}</Text>
            {i === activeLine && active ? (
              <TextInput
                value={line}
                onChange={handleChange}
                onSubmit={handleSubmit}
                placeholder={i === 0 ? "type a message..." : ""}
                showCursor
              />
            ) : (
              <Text>{line || (i === 0 ? <Text dimColor>type a message...</Text> : "")}</Text>
            )}
          </Box>
        ))}
        {lines.length > 1 && (
          <Text dimColor>  [{lines.length} lines]</Text>
        )}
      </Box>
    </Box>
  );
}
