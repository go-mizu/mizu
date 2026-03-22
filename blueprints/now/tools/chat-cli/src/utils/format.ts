const COLORS = ["cyan", "green", "yellow", "blue", "magenta", "red", "gray", "white"] as const;

export type ActorColor = (typeof COLORS)[number];

export function actorColor(actor: string): ActorColor {
  let hash = 0;
  for (let i = 0; i < actor.length; i++) {
    hash = ((hash << 5) - hash + actor.charCodeAt(i)) | 0;
  }
  return COLORS[Math.abs(hash) % COLORS.length];
}

export function formatTime(iso: string): string {
  const date = new Date(iso);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));

  const time = date.toLocaleTimeString(undefined, { hour: "2-digit", minute: "2-digit" });

  if (diffDays === 0) return time;
  if (diffDays === 1) return `yesterday ${time}`;
  if (diffDays < 7) {
    const day = date.toLocaleDateString(undefined, { weekday: "short" });
    return `${day} ${time}`;
  }
  const short = date.toLocaleDateString(undefined, { month: "short", day: "numeric" });
  return `${short} ${time}`;
}

export function roomLabel(chat: { kind: string; title: string; peer?: string }): string {
  if (chat.kind === "direct" && chat.peer) return `@${chat.peer.replace(/^[ua]\//, "")}`;
  return `#${chat.title || "untitled"}`;
}
