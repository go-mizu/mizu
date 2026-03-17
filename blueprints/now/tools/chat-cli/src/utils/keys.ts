export const KEYBINDINGS = {
  quit: { key: "q", ctrl: true },
  cycleFocus: { key: "tab" },
  cycleFocusReverse: { key: "tab", shift: true },
  createRoom: { key: "n", ctrl: true },
  joinRoom: { key: "j", ctrl: true },
  quickSwitch: { key: "k", ctrl: true },
  refresh: { key: "r", ctrl: true },
} as const;
