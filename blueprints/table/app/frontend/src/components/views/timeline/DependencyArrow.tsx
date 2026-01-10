import type { DependencyType } from '../../../types';

interface DependencyArrowProps {
  sourceX: number;
  sourceY: number;
  sourceWidth: number;
  targetX: number;
  targetY: number;
  targetWidth: number;
  type: DependencyType;
  isHighlighted?: boolean;
  onClick?: () => void;
}

export function DependencyArrow({
  sourceX,
  sourceY,
  sourceWidth,
  targetX,
  targetY,
  targetWidth,
  type,
  isHighlighted = false,
  onClick,
}: DependencyArrowProps) {
  // Calculate start and end points based on dependency type
  let startX: number;
  let endX: number;

  switch (type) {
    case 'finish_to_start':
      startX = sourceX + sourceWidth; // Right side of source
      endX = targetX; // Left side of target
      break;
    case 'start_to_start':
      startX = sourceX; // Left side of source
      endX = targetX; // Left side of target
      break;
    case 'finish_to_finish':
      startX = sourceX + sourceWidth; // Right side of source
      endX = targetX + targetWidth; // Right side of target
      break;
    case 'start_to_finish':
      startX = sourceX; // Left side of source
      endX = targetX + targetWidth; // Right side of target
      break;
    default:
      startX = sourceX + sourceWidth;
      endX = targetX;
  }

  const startY = sourceY;
  const endY = targetY;

  // Determine path based on relative positions
  let path: string;

  if (endX >= startX) {
    // Target is to the right or same position
    const cx1 = startX + Math.max(20, (endX - startX) * 0.3);
    const cx2 = endX - Math.max(20, (endX - startX) * 0.3);
    path = `M ${startX} ${startY} C ${cx1} ${startY}, ${cx2} ${endY}, ${endX} ${endY}`;
  } else {
    // Target is to the left - need to go around
    const offset = 30;
    path = `M ${startX} ${startY}
            L ${startX + offset} ${startY}
            Q ${startX + offset + 10} ${startY}, ${startX + offset + 10} ${startY + (endY > startY ? 10 : -10)}
            L ${startX + offset + 10} ${(startY + endY) / 2}
            Q ${startX + offset + 10} ${endY + (endY > startY ? -10 : 10)}, ${startX + offset} ${endY}
            L ${endX + offset} ${endY}
            Q ${endX + offset - 10} ${endY}, ${endX + offset - 10} ${endY}
            L ${endX} ${endY}`;
  }

  // Arrow head points
  const arrowSize = 6;
  const arrowAngle = Math.atan2(endY - startY, endX - startX);
  const arrowPoints = [
    [endX, endY],
    [endX - arrowSize * Math.cos(arrowAngle - Math.PI / 6), endY - arrowSize * Math.sin(arrowAngle - Math.PI / 6)],
    [endX - arrowSize * Math.cos(arrowAngle + Math.PI / 6), endY - arrowSize * Math.sin(arrowAngle + Math.PI / 6)],
  ];

  const color = isHighlighted ? '#3B82F6' : '#94A3B8';
  const strokeWidth = isHighlighted ? 2 : 1.5;

  return (
    <g
      className={`transition-colors ${onClick ? 'cursor-pointer' : ''}`}
      onClick={onClick}
    >
      {/* Shadow/hover area for easier clicking */}
      <path
        d={path}
        fill="none"
        stroke="transparent"
        strokeWidth={10}
        className="cursor-pointer"
      />

      {/* Main path */}
      <path
        d={path}
        fill="none"
        stroke={color}
        strokeWidth={strokeWidth}
        strokeDasharray={type === 'start_to_start' || type === 'finish_to_finish' ? '4,2' : undefined}
        className="transition-all"
      />

      {/* Arrow head */}
      <polygon
        points={arrowPoints.map(p => p.join(',')).join(' ')}
        fill={color}
        className="transition-colors"
      />

      {/* Connection points */}
      <circle cx={startX} cy={startY} r={3} fill={color} />
    </g>
  );
}

// Component for creating new dependencies by dragging
interface DependencyCreatorProps {
  startX: number;
  startY: number;
  endX: number;
  endY: number;
}

export function DependencyCreator({ startX, startY, endX, endY }: DependencyCreatorProps) {
  return (
    <g className="pointer-events-none">
      <line
        x1={startX}
        y1={startY}
        x2={endX}
        y2={endY}
        stroke="#3B82F6"
        strokeWidth={2}
        strokeDasharray="4,4"
      />
      <circle cx={startX} cy={startY} r={4} fill="#3B82F6" />
      <circle cx={endX} cy={endY} r={4} fill="#3B82F6" stroke="white" strokeWidth={2} />
    </g>
  );
}
