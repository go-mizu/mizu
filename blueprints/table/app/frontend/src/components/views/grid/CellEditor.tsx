import { useRef, useEffect, memo } from 'react';
import type { Field, CellValue } from '../../../types';

type RowHeightKey = 'short' | 'medium' | 'tall' | 'extra_tall';

const ROW_HEIGHT_CLASSES: Record<RowHeightKey, string> = {
  short: 'h-9',
  medium: 'h-14',
  tall: 'h-24',
  extra_tall: 'h-36',
};

interface CellEditorProps {
  field: Field;
  value: CellValue;
  isEditing: boolean;
  onChange: (value: CellValue) => void;
  onCancel: () => void;
  rowHeight?: RowHeightKey;
}

// Wrap with React.memo to prevent unnecessary re-renders.
// Each cell in the grid has a CellEditor, so preventing re-renders is critical for performance.
// Custom comparison function to check only relevant props (field.id, value, isEditing, rowHeight)
export const CellEditor = memo(function CellEditor({ field, value, isEditing, onChange, onCancel, rowHeight: _rowHeight = 'short' }: CellEditorProps) {
  void _rowHeight; // Future use for taller row layouts
  void ROW_HEIGHT_CLASSES; // Future use
  const inputRef = useRef<HTMLInputElement | HTMLTextAreaElement>(null);

  useEffect(() => {
    if (isEditing && inputRef.current) {
      inputRef.current.focus();
      if ('select' in inputRef.current) {
        inputRef.current.select();
      }
    }
  }, [isEditing]);

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      onChange((e.target as HTMLInputElement).value);
    } else if (e.key === 'Escape') {
      e.preventDefault();
      onCancel();
    }
  };

  // Render based on field type
  switch (field.type) {
    case 'text':
    case 'single_line_text':
    case 'email':
    case 'url':
    case 'phone':
      if (isEditing) {
        return (
          <input
            ref={inputRef as React.RefObject<HTMLInputElement>}
            type={field.type === 'email' ? 'email' : field.type === 'url' ? 'url' : 'text'}
            defaultValue={value as string || ''}
            onKeyDown={handleKeyDown}
            onBlur={(e) => onChange(e.target.value)}
            className="w-full h-9 px-2 border-0 focus:outline-none focus:ring-0 text-[13px]"
          />
        );
      }
      return (
        <div className="h-9 px-2 flex items-center text-[13px] truncate">
          {field.type === 'url' && value ? (
            <a href={value as string} target="_blank" rel="noopener noreferrer" className="text-primary hover:underline">
              {value as string}
            </a>
          ) : field.type === 'email' && value ? (
            <a href={`mailto:${value}`} className="text-primary hover:underline">
              {value as string}
            </a>
          ) : (
            value as string || ''
          )}
        </div>
      );

    case 'long_text':
    case 'rich_text':
      if (isEditing) {
        return (
          <textarea
            ref={inputRef as React.RefObject<HTMLTextAreaElement>}
            defaultValue={value as string || ''}
            onKeyDown={(e) => {
              if (e.key === 'Escape') {
                e.preventDefault();
                onCancel();
              }
            }}
            onBlur={(e) => onChange(e.target.value)}
            className="w-full min-h-[100px] px-2 py-1 border-0 focus:outline-none focus:ring-0 resize-none text-[13px]"
          />
        );
      }
      return (
        <div className="h-9 px-2 flex items-center text-[13px] truncate">
          {(value as string || '').split('\n')[0]}
        </div>
      );

    case 'number':
    case 'currency':
    case 'percent':
      if (isEditing) {
        return (
          <input
            ref={inputRef as React.RefObject<HTMLInputElement>}
            type="number"
            defaultValue={value as number || ''}
            onKeyDown={handleKeyDown}
            onBlur={(e) => onChange(e.target.value ? parseFloat(e.target.value) : null)}
            className="w-full h-9 px-2 border-0 focus:outline-none focus:ring-0 text-right text-[13px]"
          />
        );
      }
      const numValue = value as number;
      let displayValue = '';
      if (numValue !== null && numValue !== undefined) {
        if (field.type === 'currency') {
          const currencySymbol = field.options?.currency_symbol || '$';
          const precision = field.options?.precision ?? 2;
          displayValue = `${currencySymbol}${numValue.toLocaleString(undefined, { minimumFractionDigits: precision, maximumFractionDigits: precision })}`;
        } else if (field.type === 'percent') {
          displayValue = `${numValue}%`;
        } else {
          const precision = field.options?.precision;
          displayValue = precision !== undefined
            ? numValue.toLocaleString(undefined, { minimumFractionDigits: precision, maximumFractionDigits: precision })
            : numValue.toLocaleString();
        }
      }
      return (
        <div className="h-9 px-2 flex items-center justify-end text-[13px]">
          {displayValue}
        </div>
      );

    case 'checkbox':
      return (
        <div className="h-9 flex items-center justify-center">
          <input
            type="checkbox"
            checked={Boolean(value)}
            onChange={(e) => onChange(e.target.checked)}
            className="w-4 h-4 rounded border-gray-300 text-primary focus:ring-primary"
          />
        </div>
      );

    case 'date':
    case 'datetime':
      if (isEditing) {
        return (
          <input
            ref={inputRef as React.RefObject<HTMLInputElement>}
            type={field.type === 'datetime' ? 'datetime-local' : 'date'}
            defaultValue={value as string || ''}
            onKeyDown={handleKeyDown}
            onBlur={(e) => onChange(e.target.value || null)}
            className="w-full h-9 px-2 border-0 focus:outline-none focus:ring-0 text-[13px]"
          />
        );
      }
      return (
        <div className="h-9 px-2 flex items-center text-[13px]">
          {value ? new Date(value as string).toLocaleDateString() : ''}
        </div>
      );

    case 'single_select':
      const options = field.options?.choices || [];
      if (isEditing) {
        return (
          <select
            defaultValue={value as string || ''}
            onChange={(e) => onChange(e.target.value || null)}
            onBlur={() => onCancel()}
            autoFocus
            className="w-full h-9 px-2 border-0 focus:outline-none focus:ring-0 text-[13px]"
          >
            <option value="">Select...</option>
            {options.map((opt: { id: string; name: string; color: string }) => (
              <option key={opt.id} value={opt.id}>{opt.name}</option>
            ))}
          </select>
        );
      }
      const selectedOption = options.find((opt: { id: string }) => opt.id === value);
      return (
        <div className="h-9 px-2 flex items-center">
          {selectedOption && (
            <span
              className="px-2 py-0.5 rounded-full text-xs font-semibold"
              style={{ backgroundColor: selectedOption.color + '20', color: selectedOption.color }}
            >
              {selectedOption.name}
            </span>
          )}
        </div>
      );

    case 'multi_select':
      const multiOptions = field.options?.choices || [];
      const selectedValues = (value as string[]) || [];
      if (isEditing) {
        return (
          <div className="p-2 space-y-1">
            {multiOptions.map((opt: { id: string; name: string; color: string }) => (
              <label key={opt.id} className="flex items-center gap-2 text-sm cursor-pointer">
                <input
                  type="checkbox"
                  checked={selectedValues.includes(opt.id)}
                  onChange={(e) => {
                    const newValues = e.target.checked
                      ? [...selectedValues, opt.id]
                      : selectedValues.filter(v => v !== opt.id);
                    onChange(newValues.length > 0 ? newValues : null);
                  }}
                  className="w-4 h-4 rounded border-gray-300"
                />
                <span
                  className="px-2 py-0.5 rounded-full text-xs"
                  style={{ backgroundColor: opt.color + '20', color: opt.color }}
                >
                  {opt.name}
                </span>
              </label>
            ))}
          </div>
        );
      }
      return (
        <div className="h-9 px-2 flex items-center gap-1 overflow-hidden">
          {selectedValues.slice(0, 3).map((valId) => {
            const opt = multiOptions.find((o: { id: string }) => o.id === valId);
            return opt ? (
              <span
                key={opt.id}
                className="px-2 py-0.5 rounded-full text-xs font-semibold"
                style={{ backgroundColor: opt.color + '20', color: opt.color }}
              >
                {opt.name}
              </span>
            ) : null;
          })}
          {selectedValues.length > 3 && (
            <span className="text-xs text-gray-500">+{selectedValues.length - 3}</span>
          )}
        </div>
      );

    case 'rating':
      const maxRating = field.options?.max || 5;
      const currentRating = (value as number) || 0;
      return (
        <div className="h-9 px-2 flex items-center gap-0.5">
          {Array.from({ length: maxRating }, (_, i) => (
            <button
              key={i}
              onClick={() => onChange(i + 1 === currentRating ? null : i + 1)}
              className="text-yellow-400 hover:scale-110 transition-transform"
            >
              <svg className="w-4 h-4" fill={i < currentRating ? 'currentColor' : 'none'} stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z" />
              </svg>
            </button>
          ))}
        </div>
      );

    case 'duration':
      const durationSeconds = (value as number) || 0;
      const durationHours = Math.floor(durationSeconds / 3600);
      const durationMinutes = Math.floor((durationSeconds % 3600) / 60);
      const durationSecs = durationSeconds % 60;
      const durationFormat = field.options?.format || 'h:mm';

      if (isEditing) {
        return (
          <div className="flex items-center gap-1 px-1 h-9">
            <input
              ref={inputRef as React.RefObject<HTMLInputElement>}
              type="number"
              min="0"
              defaultValue={durationHours || ''}
              placeholder="0"
              onKeyDown={(e) => {
                if (e.key === 'Enter') {
                  e.preventDefault();
                  const h = parseInt((e.target as HTMLInputElement).value) || 0;
                  const parent = (e.target as HTMLInputElement).parentElement;
                  const minInput = parent?.querySelectorAll('input')[1] as HTMLInputElement;
                  const m = parseInt(minInput?.value) || 0;
                  onChange(h * 3600 + m * 60);
                } else if (e.key === 'Escape') {
                  onCancel();
                }
              }}
              onBlur={(e) => {
                const h = parseInt(e.target.value) || 0;
                const parent = e.target.parentElement;
                const minInput = parent?.querySelectorAll('input')[1] as HTMLInputElement;
                const m = parseInt(minInput?.value) || 0;
                onChange(h * 3600 + m * 60);
              }}
              className="w-12 h-7 px-1 border border-gray-300 rounded text-center text-[13px] focus:outline-none focus:ring-1 focus:ring-primary"
            />
            <span className="text-gray-400 text-xs">h</span>
            <input
              type="number"
              min="0"
              max="59"
              defaultValue={durationMinutes || ''}
              placeholder="0"
              className="w-12 h-7 px-1 border border-gray-300 rounded text-center text-[13px] focus:outline-none focus:ring-1 focus:ring-primary"
            />
            <span className="text-gray-400 text-xs">m</span>
          </div>
        );
      }

      let durationDisplay = '';
      if (durationSeconds > 0) {
        if (durationFormat === 'h:mm:ss') {
          durationDisplay = `${durationHours}:${durationMinutes.toString().padStart(2, '0')}:${durationSecs.toString().padStart(2, '0')}`;
        } else {
          durationDisplay = `${durationHours}:${durationMinutes.toString().padStart(2, '0')}`;
        }
      }
      return (
        <div className="h-9 px-2 flex items-center text-[13px] text-gray-600">
          {durationDisplay || '—'}
        </div>
      );

    case 'barcode':
      const barcodeValue = value as string || '';
      const barcodeType = field.options?.barcode_type || 'CODE128';
      if (isEditing) {
        return (
          <input
            ref={inputRef as React.RefObject<HTMLInputElement>}
            type="text"
            defaultValue={barcodeValue}
            onKeyDown={handleKeyDown}
            onBlur={(e) => onChange(e.target.value || null)}
            className="w-full h-9 px-2 border-0 focus:outline-none focus:ring-0 text-[13px] font-mono"
            placeholder="Enter barcode..."
          />
        );
      }
      return (
        <div className="h-9 px-2 flex items-center gap-2 text-[13px]">
          {barcodeValue ? (
            <>
              <svg className="w-4 h-4 text-gray-500 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v1m6 11h2m-6 0h-2v4m0-11v3m0 0h.01M12 12h4.01M16 20h2M4 12h2m10 0h.01M5 8h2a1 1 0 001-1V5a1 1 0 00-1-1H5a1 1 0 00-1 1v2a1 1 0 001 1zm12 0h2a1 1 0 001-1V5a1 1 0 00-1-1h-2a1 1 0 00-1 1v2a1 1 0 001 1zM5 20h2a1 1 0 001-1v-2a1 1 0 00-1-1H5a1 1 0 00-1 1v2a1 1 0 001 1z" />
              </svg>
              <span className="font-mono truncate" title={`${barcodeType}: ${barcodeValue}`}>{barcodeValue}</span>
            </>
          ) : (
            <span className="text-gray-400">—</span>
          )}
        </div>
      );

    case 'button':
      const buttonLabel = field.options?.label || field.name || 'Click';
      const buttonUrl = field.options?.url;
      const buttonColor = field.options?.color || '#2563eb';
      return (
        <div className="h-9 px-2 flex items-center">
          <button
            onClick={(e) => {
              e.stopPropagation();
              if (buttonUrl) {
                window.open(buttonUrl, '_blank', 'noopener,noreferrer');
              }
            }}
            className="px-3 py-1 rounded text-xs font-medium text-white transition-opacity hover:opacity-80"
            style={{ backgroundColor: buttonColor }}
          >
            {buttonLabel}
          </button>
        </div>
      );

    case 'attachment':
      const attachments = (value as { id?: string; filename?: string; url: string; mime_type?: string }[]) || [];
      const imageAttachments = attachments.filter(a => a.mime_type?.startsWith('image/') || a.url?.match(/\.(jpg|jpeg|png|gif|webp)/i));

      return (
        <div className="h-9 px-2 flex items-center gap-1">
          {attachments.length > 0 ? (
            <>
              {imageAttachments.length > 0 ? (
                <div className="flex -space-x-1">
                  {imageAttachments.slice(0, 3).map((att, idx) => (
                    <img
                      key={att.id || idx}
                      src={att.url}
                      alt={att.filename || 'attachment'}
                      className="w-7 h-7 rounded object-cover border-2 border-white"
                    />
                  ))}
                </div>
              ) : (
                <svg className="w-4 h-4 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15.172 7l-6.586 6.586a2 2 0 102.828 2.828l6.414-6.586a4 4 0 00-5.656-5.656l-6.415 6.585a6 6 0 108.486 8.486L20.5 13" />
                </svg>
              )}
              <span className="text-xs text-gray-500 ml-1">
                {attachments.length > 3 ? `+${attachments.length - 3}` : attachments.length > 1 ? attachments.length : ''}
              </span>
            </>
          ) : (
            <span className="text-sm text-gray-400">+ Add</span>
          )}
        </div>
      );

    case 'user':
      const users = (value as { id: string; name: string }[]) || [];
      return (
        <div className="h-9 px-2 flex items-center gap-1">
          {users.map((user) => (
            <div
              key={user.id}
              className="w-6 h-6 rounded-full bg-primary-100 flex items-center justify-center text-xs text-primary font-medium"
              title={user.name}
            >
              {user.name.charAt(0).toUpperCase()}
            </div>
          ))}
        </div>
      );

    case 'formula':
    case 'rollup':
    case 'count':
    case 'lookup':
      return (
        <div className="h-9 px-2 flex items-center text-[13px] text-gray-500 italic">
          {value !== null && value !== undefined ? String(value) : '—'}
        </div>
      );

    case 'autonumber':
      return (
        <div className="h-9 px-2 flex items-center text-[13px] text-gray-500">
          {value as number || ''}
        </div>
      );

    case 'created_time':
    case 'last_modified_time':
      return (
        <div className="h-9 px-2 flex items-center text-[13px] text-gray-500">
          {value ? new Date(value as string).toLocaleString() : ''}
        </div>
      );

    case 'created_by':
    case 'last_modified_by':
      const userValue = value as { name: string } | null;
      return (
        <div className="h-9 px-2 flex items-center text-[13px] text-gray-500">
          {userValue?.name || ''}
        </div>
      );

    case 'link':
      // Link field - displays linked records
      const linkedRecords = (value as unknown as { id: string; primary_value: string }[]) || [];
      return (
        <div className="h-9 px-2 flex items-center gap-1 overflow-hidden">
          {linkedRecords.length > 0 ? (
            <>
              {linkedRecords.slice(0, 3).map((record) => (
                <span
                  key={record.id}
                  className="px-2 py-0.5 bg-blue-50 text-blue-700 rounded text-xs font-medium truncate max-w-[100px]"
                  title={record.primary_value}
                >
                  {record.primary_value}
                </span>
              ))}
              {linkedRecords.length > 3 && (
                <span className="text-xs text-gray-500">+{linkedRecords.length - 3}</span>
              )}
            </>
          ) : (
            <span className="text-sm text-gray-400">+ Link record</span>
          )}
        </div>
      );

    default:
      return (
        <div className="h-9 px-2 flex items-center text-[13px]">
          {value !== null && value !== undefined ? String(value) : ''}
        </div>
      );
  }
}, (prevProps, nextProps) => {
  // Custom comparison: only re-render if these specific props change
  // This avoids re-renders when callback references change but values are the same
  return (
    prevProps.field.id === nextProps.field.id &&
    prevProps.value === nextProps.value &&
    prevProps.isEditing === nextProps.isEditing &&
    prevProps.rowHeight === nextProps.rowHeight
  );
});
