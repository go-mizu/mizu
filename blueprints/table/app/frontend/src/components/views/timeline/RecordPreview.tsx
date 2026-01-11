import type { TableRecord, Field } from '../../../types';

interface RecordPreviewProps {
  record: TableRecord;
  fields: Field[];
  position: { x: number; y: number };
  onClose: () => void;
  onEdit: () => void;
}

export function RecordPreview({ record, fields, position, onEdit }: RecordPreviewProps) {
  // Get primary field (first text field or first field)
  const primaryField = fields.find(f => f.type === 'text' || (f.type as string) === 'single_line_text') || fields[0];
  const title = primaryField ? (record.values[primaryField.id] as string) || 'Untitled' : 'Untitled';

  // Get visible fields (first 5 non-primary fields)
  const visibleFields = fields
    .filter(f => f.id !== primaryField?.id && !f.is_hidden)
    .slice(0, 5);

  return (
    <div
      className="fixed bg-white rounded-lg shadow-2xl border border-slate-200 w-72 z-50 pointer-events-auto"
      style={{
        left: Math.min(position.x, window.innerWidth - 300),
        top: Math.min(position.y + 10, window.innerHeight - 300),
      }}
      onClick={(e) => e.stopPropagation()}
    >
      {/* Header */}
      <div className="p-3 border-b border-slate-100">
        <h4 className="font-semibold text-slate-900 truncate">{title}</h4>
      </div>

      {/* Fields preview */}
      <div className="p-3 space-y-2 max-h-48 overflow-y-auto">
        {visibleFields.map(field => {
          const value = record.values[field.id];
          if (value === null || value === undefined || value === '') return null;

          return (
            <div key={field.id} className="flex items-start gap-2">
              <span className="text-xs text-slate-500 w-20 flex-shrink-0 truncate">
                {field.name}
              </span>
              <span className="text-sm text-slate-900 flex-1 min-w-0">
                <FieldValue value={value} field={field} />
              </span>
            </div>
          );
        })}
      </div>

      {/* Actions */}
      <div className="p-2 border-t border-slate-100 flex justify-end gap-2">
        <button
          onClick={onEdit}
          className="text-xs px-3 py-1.5 bg-primary text-white rounded hover:bg-primary/90 transition-colors"
        >
          Open record
        </button>
      </div>
    </div>
  );
}

function FieldValue({ value, field }: { value: unknown; field: Field }) {
  if (value === null || value === undefined) return null;

  switch (field.type) {
    case 'single_select':
      const choice = field.options?.choices?.find(c => c.id === value);
      if (choice) {
        return (
          <span
            className="inline-block px-2 py-0.5 rounded-full text-xs text-white"
            style={{ backgroundColor: choice.color }}
          >
            {choice.name}
          </span>
        );
      }
      return <span>{String(value)}</span>;

    case 'multi_select':
      const values = Array.isArray(value) ? value : [];
      return (
        <div className="flex flex-wrap gap-1">
          {values.slice(0, 3).map((v, i) => {
            const opt = field.options?.choices?.find(c => c.id === v);
            return (
              <span
                key={i}
                className="inline-block px-2 py-0.5 rounded-full text-xs text-white"
                style={{ backgroundColor: opt?.color || '#6B7280' }}
              >
                {opt?.name || v}
              </span>
            );
          })}
          {values.length > 3 && (
            <span className="text-xs text-slate-500">+{values.length - 3}</span>
          )}
        </div>
      );

    case 'date':
    case 'datetime':
      return <span>{new Date(value as string).toLocaleDateString()}</span>;

    case 'checkbox':
      return (
        <span className={`text-lg ${value ? 'text-green-500' : 'text-slate-300'}`}>
          {value ? '✓' : '○'}
        </span>
      );

    case 'rating':
      const rating = Number(value) || 0;
      const max = field.options?.max || 5;
      return (
        <span className="text-amber-400">
          {'★'.repeat(rating)}
          <span className="text-slate-200">{'★'.repeat(max - rating)}</span>
        </span>
      );

    case 'percent':
      return <span>{String(value)}%</span>;

    case 'currency':
      return (
        <span>
          {field.options?.currency_symbol || '$'}
          {Number(value).toLocaleString()}
        </span>
      );

    case 'url':
      return (
        <a
          href={value as string}
          target="_blank"
          rel="noopener noreferrer"
          className="text-primary hover:underline truncate block"
        >
          {value as string}
        </a>
      );

    case 'email':
      return (
        <a href={`mailto:${value}`} className="text-primary hover:underline">
          {value as string}
        </a>
      );

    case 'attachment':
      const attachments = Array.isArray(value) ? value : [];
      if (attachments.length === 0) return null;
      const first = attachments[0] as { url?: string; filename?: string };
      if (first.url?.match(/\.(jpg|jpeg|png|gif|webp)$/i)) {
        return (
          <img
            src={first.url}
            alt={first.filename || 'Attachment'}
            className="w-16 h-16 object-cover rounded"
          />
        );
      }
      return <span>{first.filename || 'Attachment'}</span>;

    case 'long_text':
      const text = String(value);
      return <span className="truncate block">{text.slice(0, 100)}{text.length > 100 ? '...' : ''}</span>;

    default:
      return <span className="truncate block">{String(value)}</span>;
  }
}
