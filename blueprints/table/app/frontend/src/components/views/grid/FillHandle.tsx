import { useRef, useCallback } from 'react';

interface FillHandleProps {
  onFillStart: () => void;
  onFillMove: (deltaRows: number, deltaCols: number) => void;
  onFillEnd: (deltaRows: number, deltaCols: number) => void;
}

export function FillHandle({ onFillStart, onFillMove, onFillEnd }: FillHandleProps) {
  const startRef = useRef<{ x: number; y: number } | null>(null);
  const cellSizeRef = useRef({ width: 200, height: 36 });

  const handleMouseDown = useCallback(
    (e: React.MouseEvent) => {
      e.preventDefault();
      e.stopPropagation();

      startRef.current = { x: e.clientX, y: e.clientY };
      onFillStart();

      // Estimate cell dimensions from parent
      const cell = (e.target as HTMLElement).closest('td');
      if (cell) {
        cellSizeRef.current = {
          width: cell.offsetWidth,
          height: cell.offsetHeight,
        };
      }

      const handleMouseMove = (moveEvent: MouseEvent) => {
        if (!startRef.current) return;

        const deltaX = moveEvent.clientX - startRef.current.x;
        const deltaY = moveEvent.clientY - startRef.current.y;

        const deltaCols = Math.round(deltaX / cellSizeRef.current.width);
        const deltaRows = Math.round(deltaY / cellSizeRef.current.height);

        onFillMove(deltaRows, deltaCols);
      };

      const handleMouseUp = (upEvent: MouseEvent) => {
        if (!startRef.current) return;

        const deltaX = upEvent.clientX - startRef.current.x;
        const deltaY = upEvent.clientY - startRef.current.y;

        const deltaCols = Math.round(deltaX / cellSizeRef.current.width);
        const deltaRows = Math.round(deltaY / cellSizeRef.current.height);

        onFillEnd(deltaRows, deltaCols);
        startRef.current = null;

        window.removeEventListener('mousemove', handleMouseMove);
        window.removeEventListener('mouseup', handleMouseUp);
      };

      window.addEventListener('mousemove', handleMouseMove);
      window.addEventListener('mouseup', handleMouseUp);
    },
    [onFillStart, onFillMove, onFillEnd]
  );

  return (
    <div
      className="absolute bottom-0 right-0 w-2 h-2 bg-primary border border-white cursor-crosshair z-20 hover:scale-125 transition-transform"
      style={{ transform: 'translate(50%, 50%)' }}
      onMouseDown={handleMouseDown}
    />
  );
}
