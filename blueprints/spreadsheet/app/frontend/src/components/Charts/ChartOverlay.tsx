import { useState, useCallback, useRef, useEffect } from 'react';
import { SpreadsheetChart } from './SpreadsheetChart';
import type { Chart, UpdateChartRequest } from '../../types';

interface ChartOverlayProps {
  charts: Chart[];
  selectedChart: Chart | null;
  onSelectChart: (chart: Chart | null) => void;
  onEditChart: (chart: Chart) => void;
  onDeleteChart: (chartId: string) => void;
  onUpdateChart: (chartId: string, updates: UpdateChartRequest) => Promise<void>;
  rowHeight: number;
  colWidth: number;
  scrollLeft: number;
  scrollTop: number;
  headerHeight: number;
  headerWidth: number;
}

export function ChartOverlay({
  charts,
  selectedChart,
  onSelectChart,
  onEditChart,
  onDeleteChart,
  onUpdateChart,
  rowHeight,
  colWidth,
  scrollLeft,
  scrollTop,
  headerHeight,
  headerWidth,
}: ChartOverlayProps) {
  const [dragging, setDragging] = useState<{
    chartId: string;
    startX: number;
    startY: number;
    startRow: number;
    startCol: number;
  } | null>(null);

  const [resizing, setResizing] = useState<{
    chartId: string;
    handle: 'nw' | 'ne' | 'sw' | 'se';
    startX: number;
    startY: number;
    startWidth: number;
    startHeight: number;
  } | null>(null);

  const overlayRef = useRef<HTMLDivElement>(null);

  // Calculate chart position in pixels
  const getChartPosition = useCallback(
    (chart: Chart) => {
      const x = headerWidth + chart.position.col * colWidth + chart.position.offsetX - scrollLeft;
      const y = headerHeight + chart.position.row * rowHeight + chart.position.offsetY - scrollTop;
      return { x, y };
    },
    [rowHeight, colWidth, scrollLeft, scrollTop, headerHeight, headerWidth]
  );

  // Handle mouse move for dragging/resizing
  useEffect(() => {
    if (!dragging && !resizing) return;

    const handleMouseMove = (e: MouseEvent) => {
      if (dragging) {
        const dx = e.clientX - dragging.startX;
        const dy = e.clientY - dragging.startY;

        // Calculate new row/col
        const newCol = Math.max(0, Math.round(dragging.startCol + dx / colWidth));
        const newRow = Math.max(0, Math.round(dragging.startRow + dy / rowHeight));

        // Update chart position temporarily (visual feedback)
        const chart = charts.find((c) => c.id === dragging.chartId);
        if (chart) {
          chart.position.row = newRow;
          chart.position.col = newCol;
        }
      }

      if (resizing) {
        const chart = charts.find((c) => c.id === resizing.chartId);
        if (!chart) return;

        const dx = e.clientX - resizing.startX;
        const dy = e.clientY - resizing.startY;

        let newWidth = resizing.startWidth;
        let newHeight = resizing.startHeight;

        if (resizing.handle.includes('e')) {
          newWidth = Math.max(200, resizing.startWidth + dx);
        }
        if (resizing.handle.includes('w')) {
          newWidth = Math.max(200, resizing.startWidth - dx);
        }
        if (resizing.handle.includes('s')) {
          newHeight = Math.max(150, resizing.startHeight + dy);
        }
        if (resizing.handle.includes('n')) {
          newHeight = Math.max(150, resizing.startHeight - dy);
        }

        chart.size.width = newWidth;
        chart.size.height = newHeight;
      }
    };

    const handleMouseUp = async () => {
      if (dragging) {
        const chart = charts.find((c) => c.id === dragging.chartId);
        if (chart) {
          await onUpdateChart(chart.id, {
            position: chart.position,
          });
        }
      }

      if (resizing) {
        const chart = charts.find((c) => c.id === resizing.chartId);
        if (chart) {
          await onUpdateChart(chart.id, {
            size: chart.size,
          });
        }
      }

      setDragging(null);
      setResizing(null);
    };

    document.addEventListener('mousemove', handleMouseMove);
    document.addEventListener('mouseup', handleMouseUp);

    return () => {
      document.removeEventListener('mousemove', handleMouseMove);
      document.removeEventListener('mouseup', handleMouseUp);
    };
  }, [dragging, resizing, charts, colWidth, rowHeight, onUpdateChart]);

  // Handle chart drag start
  const handleDragStart = useCallback(
    (e: React.MouseEvent, chart: Chart) => {
      if (e.button !== 0) return; // Only left click
      e.preventDefault();
      e.stopPropagation();

      setDragging({
        chartId: chart.id,
        startX: e.clientX,
        startY: e.clientY,
        startRow: chart.position.row,
        startCol: chart.position.col,
      });

      onSelectChart(chart);
    },
    [onSelectChart]
  );

  // Handle chart resize start
  const handleResizeStart = useCallback(
    (e: React.MouseEvent, chart: Chart, handle: 'nw' | 'ne' | 'sw' | 'se') => {
      e.preventDefault();
      e.stopPropagation();

      setResizing({
        chartId: chart.id,
        handle,
        startX: e.clientX,
        startY: e.clientY,
        startWidth: chart.size.width,
        startHeight: chart.size.height,
      });
    },
    []
  );

  // Handle context menu
  const handleContextMenu = useCallback(
    (e: React.MouseEvent, chart: Chart) => {
      e.preventDefault();
      e.stopPropagation();
      onSelectChart(chart);

      // Create context menu
      const menu = document.createElement('div');
      menu.className = 'chart-context-menu';
      menu.style.cssText = `
        position: fixed;
        left: ${e.clientX}px;
        top: ${e.clientY}px;
        background: white;
        border: 1px solid #ddd;
        border-radius: 4px;
        box-shadow: 0 2px 8px rgba(0,0,0,0.15);
        z-index: 1000;
        min-width: 150px;
      `;

      const items = [
        { label: 'Edit Chart', action: () => onEditChart(chart) },
        { label: 'Delete Chart', action: () => onDeleteChart(chart.id) },
      ];

      items.forEach((item) => {
        const itemEl = document.createElement('div');
        itemEl.textContent = item.label;
        itemEl.style.cssText = `
          padding: 8px 16px;
          cursor: pointer;
          font-size: 14px;
        `;
        itemEl.onmouseenter = () => {
          itemEl.style.backgroundColor = '#f5f5f5';
        };
        itemEl.onmouseleave = () => {
          itemEl.style.backgroundColor = 'transparent';
        };
        itemEl.onclick = () => {
          document.body.removeChild(menu);
          item.action();
        };
        menu.appendChild(itemEl);
      });

      document.body.appendChild(menu);

      const closeMenu = (ev: MouseEvent) => {
        if (!menu.contains(ev.target as Node)) {
          document.body.removeChild(menu);
          document.removeEventListener('click', closeMenu);
        }
      };

      setTimeout(() => {
        document.addEventListener('click', closeMenu);
      }, 0);
    },
    [onSelectChart, onEditChart, onDeleteChart]
  );

  return (
    <div
      ref={overlayRef}
      style={{
        position: 'absolute',
        top: 0,
        left: 0,
        width: '100%',
        height: '100%',
        pointerEvents: 'none',
        overflow: 'hidden',
      }}
    >
      {charts.map((chart) => {
        const pos = getChartPosition(chart);

        // Skip if chart is outside visible area
        if (
          pos.x + chart.size.width < 0 ||
          pos.y + chart.size.height < 0 ||
          pos.x > window.innerWidth ||
          pos.y > window.innerHeight
        ) {
          return null;
        }

        const isSelected = selectedChart?.id === chart.id;

        return (
          <div
            key={chart.id}
            style={{
              position: 'absolute',
              left: pos.x,
              top: pos.y,
              pointerEvents: 'auto',
              cursor: dragging?.chartId === chart.id ? 'grabbing' : 'grab',
            }}
            onMouseDown={(e) => handleDragStart(e, chart)}
            onContextMenu={(e) => handleContextMenu(e, chart)}
          >
            <SpreadsheetChart
              chart={chart}
              selected={isSelected}
              onSelect={() => onSelectChart(chart)}
              onEdit={() => onEditChart(chart)}
            />

            {/* Resize handles */}
            {isSelected && (
              <>
                <div
                  style={{
                    position: 'absolute',
                    top: -4,
                    left: -4,
                    width: 8,
                    height: 8,
                    backgroundColor: '#2196F3',
                    borderRadius: '50%',
                    cursor: 'nw-resize',
                    pointerEvents: 'auto',
                  }}
                  onMouseDown={(e) => {
                    e.stopPropagation();
                    handleResizeStart(e, chart, 'nw');
                  }}
                />
                <div
                  style={{
                    position: 'absolute',
                    top: -4,
                    right: -4,
                    width: 8,
                    height: 8,
                    backgroundColor: '#2196F3',
                    borderRadius: '50%',
                    cursor: 'ne-resize',
                    pointerEvents: 'auto',
                  }}
                  onMouseDown={(e) => {
                    e.stopPropagation();
                    handleResizeStart(e, chart, 'ne');
                  }}
                />
                <div
                  style={{
                    position: 'absolute',
                    bottom: -4,
                    left: -4,
                    width: 8,
                    height: 8,
                    backgroundColor: '#2196F3',
                    borderRadius: '50%',
                    cursor: 'sw-resize',
                    pointerEvents: 'auto',
                  }}
                  onMouseDown={(e) => {
                    e.stopPropagation();
                    handleResizeStart(e, chart, 'sw');
                  }}
                />
                <div
                  style={{
                    position: 'absolute',
                    bottom: -4,
                    right: -4,
                    width: 8,
                    height: 8,
                    backgroundColor: '#2196F3',
                    borderRadius: '50%',
                    cursor: 'se-resize',
                    pointerEvents: 'auto',
                  }}
                  onMouseDown={(e) => {
                    e.stopPropagation();
                    handleResizeStart(e, chart, 'se');
                  }}
                />
              </>
            )}
          </div>
        );
      })}
    </div>
  );
}

export default ChartOverlay;
