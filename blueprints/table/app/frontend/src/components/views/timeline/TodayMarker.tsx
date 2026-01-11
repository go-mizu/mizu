interface TodayMarkerProps {
  left: number;
  height: number;
  visible: boolean;
}

export function TodayMarker({ left, height, visible }: TodayMarkerProps) {
  if (!visible || left < 0) return null;

  return (
    <div
      className="absolute top-0 pointer-events-none z-30"
      style={{ left, height }}
    >
      {/* Marker line */}
      <div className="w-0.5 h-full bg-red-500 opacity-75" />

      {/* Today label at top */}
      <div className="absolute -top-6 left-1/2 -translate-x-1/2 whitespace-nowrap">
        <span className="text-xs font-medium text-red-500 bg-white px-1.5 py-0.5 rounded shadow-sm border border-red-200">
          Today
        </span>
      </div>

      {/* Small triangle at top */}
      <div
        className="absolute -top-1 left-1/2 -translate-x-1/2 w-0 h-0"
        style={{
          borderLeft: '4px solid transparent',
          borderRight: '4px solid transparent',
          borderTop: '6px solid #EF4444',
        }}
      />
    </div>
  );
}
