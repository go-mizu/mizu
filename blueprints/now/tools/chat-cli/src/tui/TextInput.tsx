import React, { useState } from "react";
import { Text, useInput } from "ink";
import chalk from "chalk";

interface Props {
  value: string;
  onChange: (value: string) => void;
  onSubmit?: (value: string) => void;
  placeholder?: string;
  isActive?: boolean;
}

/**
 * Minimal controlled text input — avoids the double-render bug
 * in ink-text-input@6 (its useEffect creates a new state object
 * on every keystroke even when values are identical).
 */
export function TextInput({
  value,
  onChange,
  onSubmit,
  placeholder = "",
  isActive = true,
}: Props) {
  const [cursorOffset, setCursorOffset] = useState(value.length);

  useInput(
    (input, key) => {
      if (
        key.upArrow || key.downArrow || key.tab ||
        (key.ctrl && input === "c") ||
        (key.shift && key.tab)
      ) {
        return;
      }

      if (key.return) {
        onSubmit?.(value);
        return;
      }

      let nextValue = value;
      let nextOffset = cursorOffset;

      if (key.leftArrow) {
        nextOffset = Math.max(0, cursorOffset - 1);
      } else if (key.rightArrow) {
        nextOffset = Math.min(value.length, cursorOffset + 1);
      } else if (key.backspace || key.delete) {
        if (cursorOffset > 0) {
          nextValue = value.slice(0, cursorOffset - 1) + value.slice(cursorOffset);
          nextOffset = cursorOffset - 1;
        }
      } else {
        nextValue = value.slice(0, cursorOffset) + input + value.slice(cursorOffset);
        nextOffset = cursorOffset + input.length;
      }

      // Clamp
      nextOffset = Math.max(0, Math.min(nextValue.length, nextOffset));

      // Only update cursor if it actually changed
      if (nextOffset !== cursorOffset) {
        setCursorOffset(nextOffset);
      }

      if (nextValue !== value) {
        onChange(nextValue);
        // Keep cursor in sync with new value length
        setCursorOffset(Math.min(nextOffset, nextValue.length));
      }
    },
    { isActive },
  );

  // Render with fake cursor
  if (value.length === 0) {
    if (placeholder) {
      return <Text>{chalk.inverse(placeholder[0])}{chalk.gray(placeholder.slice(1))}</Text>;
    }
    return <Text>{chalk.inverse(" ")}</Text>;
  }

  let rendered = "";
  for (let i = 0; i < value.length; i++) {
    rendered += i === cursorOffset ? chalk.inverse(value[i]!) : value[i];
  }
  if (cursorOffset === value.length) {
    rendered += chalk.inverse(" ");
  }

  return <Text>{rendered}</Text>;
}
