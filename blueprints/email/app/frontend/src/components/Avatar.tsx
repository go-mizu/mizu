const AVATAR_COLORS = [
  "#1A73E8",
  "#EA4335",
  "#FBBC04",
  "#34A853",
  "#FF6D01",
  "#46BDC6",
  "#7BAAF7",
  "#F07B72",
  "#FCD04F",
  "#71C287",
  "#AF5CF7",
  "#E8710A",
  "#F439A0",
  "#24C1E0",
  "#9334E6",
  "#A0C3FF",
];

function hashString(str: string): number {
  let hash = 0;
  for (let i = 0; i < str.length; i++) {
    const char = str.charCodeAt(i);
    hash = (hash << 5) - hash + char;
    hash |= 0;
  }
  return Math.abs(hash);
}

interface AvatarProps {
  name: string;
  email?: string;
  size?: number;
  className?: string;
}

export default function Avatar({
  name,
  email,
  size = 40,
  className = "",
}: AvatarProps) {
  const displayStr = name || email || "?";
  const initial = displayStr.charAt(0).toUpperCase();
  const colorIndex = hashString(displayStr) % AVATAR_COLORS.length;
  const bgColor = AVATAR_COLORS[colorIndex]!;

  const fontSize = size <= 32 ? 14 : size <= 40 ? 16 : 20;

  return (
    <div
      className={`flex flex-shrink-0 items-center justify-center rounded-full font-medium text-white ${className}`}
      style={{
        width: size,
        height: size,
        backgroundColor: bgColor,
        fontSize,
        fontFamily: "'Google Sans', 'Roboto', sans-serif",
        lineHeight: 1,
      }}
      aria-label={displayStr}
    >
      {initial}
    </div>
  );
}
