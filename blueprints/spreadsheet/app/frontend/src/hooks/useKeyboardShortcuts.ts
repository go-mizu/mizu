import { useCallback, useEffect } from 'react';

export interface KeyboardShortcutHandlers {
  // Navigation
  onMoveUp?: () => void;
  onMoveDown?: () => void;
  onMoveLeft?: () => void;
  onMoveRight?: () => void;
  onMoveToStart?: () => void;
  onMoveToEnd?: () => void;
  onTab?: () => void;
  onShiftTab?: () => void;
  onPageUp?: () => void;
  onPageDown?: () => void;

  // Selection
  onExtendUp?: () => void;
  onExtendDown?: () => void;
  onExtendLeft?: () => void;
  onExtendRight?: () => void;
  onSelectAll?: () => void;

  // Editing
  onEdit?: () => void;
  onDelete?: () => void;
  onEscape?: () => void;
  onEnter?: () => void;

  // Clipboard
  onCopy?: () => void;
  onCut?: () => void;
  onPaste?: () => void;
  onPasteValues?: () => void;

  // Undo/Redo
  onUndo?: () => void;
  onRedo?: () => void;

  // Formatting
  onBold?: () => void;
  onItalic?: () => void;
  onUnderline?: () => void;
  onStrikethrough?: () => void;

  // Find/Replace
  onFind?: () => void;
  onReplace?: () => void;

  // Check if editing mode
  isEditing?: () => boolean;
}

export function useKeyboardShortcuts(
  handlers: KeyboardShortcutHandlers,
  enabled: boolean = true
) {
  const handleKeyDown = useCallback(
    (event: KeyboardEvent) => {
      if (!enabled) return;

      const { key, ctrlKey, metaKey, shiftKey, altKey } = event;
      const ctrl = ctrlKey || metaKey;
      const isEditing = handlers.isEditing?.() ?? false;

      // Ignore keyboard shortcuts when typing in input fields
      const target = event.target as HTMLElement;
      const isInputField =
        target.tagName === 'INPUT' ||
        target.tagName === 'TEXTAREA' ||
        target.isContentEditable;

      // Always allow Escape
      if (key === 'Escape') {
        event.preventDefault();
        handlers.onEscape?.();
        return;
      }

      // Allow Ctrl shortcuts even in input fields (for undo/redo, etc.)
      if (ctrl && !altKey) {
        switch (key.toLowerCase()) {
          case 'z':
            event.preventDefault();
            if (shiftKey) {
              handlers.onRedo?.();
            } else {
              handlers.onUndo?.();
            }
            return;
          case 'y':
            event.preventDefault();
            handlers.onRedo?.();
            return;
          case 'c':
            if (!isInputField) {
              event.preventDefault();
              handlers.onCopy?.();
            }
            return;
          case 'x':
            if (!isInputField) {
              event.preventDefault();
              handlers.onCut?.();
            }
            return;
          case 'v':
            if (!isInputField) {
              event.preventDefault();
              if (shiftKey) {
                handlers.onPasteValues?.();
              } else {
                handlers.onPaste?.();
              }
            }
            return;
          case 'b':
            event.preventDefault();
            handlers.onBold?.();
            return;
          case 'i':
            event.preventDefault();
            handlers.onItalic?.();
            return;
          case 'u':
            event.preventDefault();
            handlers.onUnderline?.();
            return;
          case '5':
            event.preventDefault();
            handlers.onStrikethrough?.();
            return;
          case 'f':
            event.preventDefault();
            handlers.onFind?.();
            return;
          case 'h':
            event.preventDefault();
            handlers.onReplace?.();
            return;
          case 'a':
            if (!isInputField) {
              event.preventDefault();
              handlers.onSelectAll?.();
            }
            return;
          case 'home':
            event.preventDefault();
            handlers.onMoveToStart?.();
            return;
          case 'end':
            event.preventDefault();
            handlers.onMoveToEnd?.();
            return;
        }
      }

      // Don't handle other keys when editing or in input fields
      if (isEditing || isInputField) return;

      // Navigation and editing shortcuts
      switch (key) {
        case 'ArrowUp':
          event.preventDefault();
          if (shiftKey) {
            handlers.onExtendUp?.();
          } else {
            handlers.onMoveUp?.();
          }
          break;
        case 'ArrowDown':
          event.preventDefault();
          if (shiftKey) {
            handlers.onExtendDown?.();
          } else {
            handlers.onMoveDown?.();
          }
          break;
        case 'ArrowLeft':
          event.preventDefault();
          if (shiftKey) {
            handlers.onExtendLeft?.();
          } else {
            handlers.onMoveLeft?.();
          }
          break;
        case 'ArrowRight':
          event.preventDefault();
          if (shiftKey) {
            handlers.onExtendRight?.();
          } else {
            handlers.onMoveRight?.();
          }
          break;
        case 'Tab':
          event.preventDefault();
          if (shiftKey) {
            handlers.onShiftTab?.();
          } else {
            handlers.onTab?.();
          }
          break;
        case 'Enter':
          event.preventDefault();
          if (shiftKey) {
            handlers.onMoveUp?.();
          } else {
            handlers.onEnter?.();
          }
          break;
        case 'F2':
          event.preventDefault();
          handlers.onEdit?.();
          break;
        case 'Delete':
        case 'Backspace':
          event.preventDefault();
          handlers.onDelete?.();
          break;
        case 'PageUp':
          handlers.onPageUp?.();
          break;
        case 'PageDown':
          handlers.onPageDown?.();
          break;
        case 'Home':
          event.preventDefault();
          handlers.onMoveToStart?.();
          break;
        case 'End':
          event.preventDefault();
          handlers.onMoveToEnd?.();
          break;
      }
    },
    [handlers, enabled]
  );

  useEffect(() => {
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [handleKeyDown]);
}
