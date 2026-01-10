import { useRef, useEffect } from 'react';
import type { Field, CellValue } from '../../../types';

interface CellEditorProps {
  field: Field;
  value: CellValue;
  isEditing: boolean;
  onChange: (value: CellValue) => void;
  onCancel: () => void;
}

export function CellEditor({ field, value, isEditing, onChange, onCancel }: CellEditorProps) {
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
          displayValue = `$${numValue.toLocaleString()}`;
        } else if (field.type === 'percent') {
          displayValue = `${numValue}%`;
        } else {
          displayValue = numValue.toLocaleString();
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

    case 'attachment':
      const attachments = (value as { filename?: string; url: string }[]) || [];
      return (
        <div className="h-9 px-2 flex items-center gap-1">
          {attachments.length > 0 ? (
            <>
              <svg className="w-4 h-4 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15.172 7l-6.586 6.586a2 2 0 102.828 2.828l6.414-6.586a4 4 0 00-5.656-5.656l-6.415 6.585a6 6 0 108.486 8.486L20.5 13" />
              </svg>
              <span className="text-sm text-gray-600">{attachments.length} file(s)</span>
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

    default:
      return (
        <div className="h-9 px-2 flex items-center text-[13px]">
          {value !== null && value !== undefined ? String(value) : ''}
        </div>
      );
  }
}
