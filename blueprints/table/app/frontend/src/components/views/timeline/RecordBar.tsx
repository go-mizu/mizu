import { useState, useRef, useCallback } from 'react';
import type { TableRecord, Field } from '../../../types';

interface RecordBarProps {
  record: TableRecord;
  left: number;
  width: number;
  title: string;
  color: string;
  fields: Field[];
  labelFieldIds?: string[];
  rowHeight: number;
  stackIndex?: number;
  msPerPixel: number;
  startField: Field;
  endField?: Field;
  onRecordClick: (record: TableRecord) => void;
  onDateChange: (recordId: string, startDate: string, endDate?: string) => void;
  onHover: (record: TableRecord | null, event?: React.MouseEvent) => void;
}

export function RecordBar({
  record,
  left,
  width,
  title,
  color,
  fields,
  labelFieldIds,
  rowHeight,
  stackIndex = 0,
  msPerPixel,
  startField,
  endField,
  onRecordClick,
  onDateChange,
  onHover,
}: RecordBarProps) {
  const barRef = useRef<HTMLDivElement>(null);
  const [isDragging, setIsDragging] = useState(false);
  const [isResizing, setIsResizing] = useState<'left' | 'right' | null>(null);
  const [dragOffset, setDragOffset] = useState(0);
  const [resizeWidth, setResizeWidth] = useState(width);
  const [resizeLeft, setResizeLeft] = useState(left);

  const barHeight = rowHeight === 32 ? 24 : rowHeight === 48 ? 36 : 28;
  const stackOffset = stackIndex * (barHeight + 4);

  // Get additional labels to show
  const additionalLabels = labelFieldIds
    ?.filter(id => id !== startField.id && id !== endField?.id)
    .map(id => {
      const field = fields.find(f => f.id === id);
      if (!field) return null;
      const value = record.values[id];
      if (value === null || value === undefined || value === '') return null;
      return { field, value: formatValue(value, field) };
    })
    .filter(Boolean)
    .slice(0, 2);

  // Drag handlers
  const handleMouseDown = useCallback((e: React.MouseEvent) => {
    if (isResizing) return;
    e.preventDefault();
    e.stopPropagation();

    const startX = e.clientX;

    const handleMouseMove = (moveEvent: MouseEvent) => {
      const deltaX = moveEvent.clientX - startX;
      setDragOffset(deltaX);
      setIsDragging(true);
    };

    const handleMouseUp = (upEvent: MouseEvent) => {
      document.removeEventListener('mousemove', handleMouseMove);
      document.removeEventListener('mouseup', handleMouseUp);
      setIsDragging(false);

      const finalDelta = upEvent.clientX - startX;
      if (Math.abs(finalDelta) > 5) {
        // Calculate new dates
        const deltaMs = finalDelta * msPerPixel;
        const currentStart = new Date(record.values[startField.id] as string);
        const newStart = new Date(currentStart.getTime() + deltaMs);
        const newStartStr = newStart.toISOString().split('T')[0];

        let newEndStr: string | undefined;
        if (endField && record.values[endField.id]) {
          const currentEnd = new Date(record.values[endField.id] as string);
          const newEnd = new Date(currentEnd.getTime() + deltaMs);
          newEndStr = newEnd.toISOString().split('T')[0];
        }

        onDateChange(record.id, newStartStr, newEndStr);
      }
      setDragOffset(0);
    };

    document.addEventListener('mousemove', handleMouseMove);
    document.addEventListener('mouseup', handleMouseUp);
  }, [left, msPerPixel, record, startField, endField, onDateChange, isResizing]);

  // Resize handlers
  const handleResizeStart = useCallback((e: React.MouseEvent, side: 'left' | 'right') => {
    e.preventDefault();
    e.stopPropagation();

    const startX = e.clientX;
    const initialWidth = width;
    const initialLeft = left;

    setIsResizing(side);
    setResizeWidth(width);
    setResizeLeft(left);

    const handleMouseMove = (moveEvent: MouseEvent) => {
      const deltaX = moveEvent.clientX - startX;

      if (side === 'right') {
        const newWidth = Math.max(30, initialWidth + deltaX);
        setResizeWidth(newWidth);
      } else {
        const newLeft = initialLeft + deltaX;
        const newWidth = Math.max(30, initialWidth - deltaX);
        setResizeLeft(newLeft);
        setResizeWidth(newWidth);
      }
    };

    const handleMouseUp = (upEvent: MouseEvent) => {
      document.removeEventListener('mousemove', handleMouseMove);
      document.removeEventListener('mouseup', handleMouseUp);
      setIsResizing(null);

      const deltaX = upEvent.clientX - startX;
      if (Math.abs(deltaX) > 5) {
        const currentStart = new Date(record.values[startField.id] as string);
        const currentEnd = endField && record.values[endField.id]
          ? new Date(record.values[endField.id] as string)
          : new Date(currentStart.getTime() + 86400000);

        if (side === 'right') {
          // Extend/shrink end date
          const deltaMs = deltaX * msPerPixel;
          const newEnd = new Date(currentEnd.getTime() + deltaMs);
          onDateChange(record.id, record.values[startField.id] as string, newEnd.toISOString().split('T')[0]);
        } else {
          // Extend/shrink start date
          const deltaMs = deltaX * msPerPixel;
          const newStart = new Date(currentStart.getTime() + deltaMs);
          onDateChange(record.id, newStart.toISOString().split('T')[0], endField ? record.values[endField.id] as string : undefined);
        }
      }
    };

    document.addEventListener('mousemove', handleMouseMove);
    document.addEventListener('mouseup', handleMouseUp);
  }, [width, left, msPerPixel, record, startField, endField, onDateChange]);

  const currentLeft = isResizing ? resizeLeft : (isDragging ? left + dragOffset : left);
  const currentWidth = isResizing ? resizeWidth : width;

  return (
    <div
      ref={barRef}
      className={`absolute rounded cursor-pointer flex items-center group transition-shadow ${
        isDragging || isResizing ? 'shadow-lg z-20' : 'hover:shadow-md z-10'
      }`}
      style={{
        left: currentLeft,
        width: currentWidth,
        height: barHeight,
        top: 4 + stackOffset,
        backgroundColor: color,
        opacity: isDragging ? 0.9 : 1,
      }}
      onMouseDown={handleMouseDown}
      onClick={(e) => {
        e.stopPropagation();
        if (!isDragging && !isResizing) {
          onRecordClick(record);
        }
      }}
      onMouseEnter={(e) => onHover(record, e)}
      onMouseLeave={() => onHover(null)}
    >
      {/* Left resize handle */}
      <div
        className="absolute left-0 top-0 bottom-0 w-2 cursor-ew-resize opacity-0 group-hover:opacity-100 hover:bg-black/20 rounded-l"
        onMouseDown={(e) => handleResizeStart(e, 'left')}
      />

      {/* Bar content */}
      <div className="flex-1 px-2 overflow-hidden flex items-center gap-2 min-w-0">
        <span className="text-xs text-white font-medium truncate">
          {currentWidth > 60 ? title : ''}
        </span>
        {currentWidth > 120 && additionalLabels && additionalLabels.length > 0 && (
          <span className="text-xs text-white/70 truncate">
            {additionalLabels.map((l, i) => l && (
              <span key={i} className="mr-2">{l.value}</span>
            ))}
          </span>
        )}
      </div>

      {/* Right resize handle */}
      <div
        className="absolute right-0 top-0 bottom-0 w-2 cursor-ew-resize opacity-0 group-hover:opacity-100 hover:bg-black/20 rounded-r"
        onMouseDown={(e) => handleResizeStart(e, 'right')}
      />

      {/* Progress indicator (if progress field exists) */}
      {record.values['progress'] !== undefined && (
        <div
          className="absolute bottom-0 left-0 h-1 bg-white/40 rounded-b"
          style={{ width: `${record.values['progress']}%` }}
        />
      )}
    </div>
  );
}

function formatValue(value: unknown, field: Field): string {
  if (value === null || value === undefined) return '';

  switch (field.type) {
    case 'single_select':
      const choice = field.options?.choices?.find(c => c.id === value);
      return choice?.name || String(value);
    case 'date':
    case 'datetime':
      return new Date(value as string).toLocaleDateString();
    case 'number':
    case 'currency':
    case 'percent':
      return String(value);
    case 'checkbox':
      return value ? '✓' : '';
    case 'rating':
      return '★'.repeat(value as number);
    default:
      return String(value);
  }
}
