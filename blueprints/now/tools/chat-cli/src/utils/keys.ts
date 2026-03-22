export const KEYBINDINGS = {
  quit: { key: "q", ctrl: true },
  roomSwitcher: { key: "k", ctrl: true },
  createRoom: { key: "n", ctrl: true },
  joinRoom: { key: "j", ctrl: true },
  refresh: { key: "r", ctrl: true },
  prevRoom: { key: "left", ctrl: true },
  nextRoom: { key: "right", ctrl: true },
} as const;
