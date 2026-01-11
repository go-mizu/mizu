import { useSortable } from '@dnd-kit/sortable';
import { CSS } from '@dnd-kit/utilities';
import type { TableRecord, Field, CellValue, Attachment } from '../../../types';
import { CARD_SIZES, type KanbanConfig } from './types';

interface KanbanCardProps {
  record: TableRecord;
  primaryField: Field | undefined;
  displayFields: Field[];
  coverField: Field | undefined;
  colorField: Field | undefined;
  config: KanbanConfig;
  isDragging?: boolean;
  hideEmptyFields?: boolean;
  onClick: () => void;
}

export function KanbanCard({
  record,
  primaryField,
  displayFields,
  coverField,
  colorField,
  config,
  isDragging,
  hideEmptyFields,
  onClick,
}: KanbanCardProps) {
  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging: isSortableDragging,
  } = useSortable({
    id: record.id,
    data: {
      type: 'card',
      record,
    },
  });

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isSortableDragging ? 0.5 : 1,
  };

  const sizeConfig = CARD_SIZES[config.cardSize];

  // Get title from primary field
  const title = primaryField
    ? (record.values[primaryField.id] as string) || 'Untitled'
    : 'Untitled';

  // Get cover image if configured
  const coverImage = getCoverImage(record, coverField);

  // Get card background color from color field
  const cardColor = getCardColor(record, colorField);

  // Get fields to display (limited by card size, optionally hiding empty fields)
  const fieldsToShow = (() => {
    let fields = displayFields.slice(0, sizeConfig.fieldCount);

    if (hideEmptyFields) {
      fields = fields.filter((field) => {
        const value = record.values[field.id];
        if (value === null || value === undefined || value === '') return false;
        if (Array.isArray(value) && value.length === 0) return false;
        return true;
      });
    }

    return fields;
  })();

  return (
    <div
      ref={setNodeRef}
      style={style}
      {...attributes}
      {...listeners}
      onClick={onClick}
      className={`
        relative bg-white rounded-lg shadow-sm border border-slate-200
        cursor-pointer hover:shadow-md transition-all duration-150
        ${sizeConfig.padding}
        ${isDragging ? 'ring-2 ring-primary shadow-lg' : ''}
      `}
    >
      {/* Cover image */}
      {coverImage && (
        <div className={`-mx-3 -mt-3 mb-2 ${sizeConfig.coverHeight} bg-gray-100 rounded-t-lg overflow-hidden`}>
          <img
            src={coverImage.thumbnail_url || coverImage.url}
            alt=""
            className={`w-full h-full ${config.cardCoverFit === 'contain' ? 'object-contain' : 'object-cover'}`}
            loading="lazy"
          />
        </div>
      )}

      {/* Card color indicator */}
      {cardColor && (
        <div
          className="absolute top-0 left-0 right-0 h-1 rounded-t-lg"
          style={{ backgroundColor: cardColor }}
        />
      )}

      {/* Title */}
      <h4 className={`font-medium text-gray-900 mb-1 line-clamp-2 ${sizeConfig.titleSize}`}>
        {title}
      </h4>

      {/* Display fields */}
      {fieldsToShow.length > 0 && (
        <div className="space-y-1 mt-2">
          {fieldsToShow.map((field) => {
            const value = record.values[field.id];
            if (value === null || value === undefined || value === '') return null;

            return (
              <div key={field.id} className="text-xs text-gray-500 truncate">
                <span className="font-medium text-gray-400">{field.name}:</span>{' '}
                {renderFieldValue(field, value)}
              </div>
            );
          })}
        </div>
      )}

      {/* Card color background (subtle) */}
      {cardColor && (
        <div
          className="absolute inset-0 rounded-lg pointer-events-none"
          style={{ backgroundColor: cardColor, opacity: 0.08 }}
        />
      )}
    </div>
  );
}

// Overlay component for drag preview
export function KanbanCardOverlay({
  record,
  primaryField,
  displayFields,
  coverField,
  config,
}: Omit<KanbanCardProps, 'onClick' | 'isDragging' | 'colorField'>) {
  const sizeConfig = CARD_SIZES[config.cardSize];
  const title = primaryField
    ? (record.values[primaryField.id] as string) || 'Untitled'
    : 'Untitled';
  const coverImage = getCoverImage(record, coverField);
  const fieldsToShow = displayFields.slice(0, sizeConfig.fieldCount);

  return (
    <div
      className={`
        bg-white rounded-lg shadow-xl border-2 border-primary
        ${sizeConfig.padding}
        w-64
      `}
    >
      {coverImage && (
        <div className={`-mx-3 -mt-3 mb-2 ${sizeConfig.coverHeight} bg-gray-100 rounded-t-lg overflow-hidden`}>
          <img
            src={coverImage.thumbnail_url || coverImage.url}
            alt=""
            className={`w-full h-full ${config.cardCoverFit === 'contain' ? 'object-contain' : 'object-cover'}`}
          />
        </div>
      )}

      <h4 className={`font-medium text-gray-900 mb-1 line-clamp-2 ${sizeConfig.titleSize}`}>
        {title}
      </h4>

      {fieldsToShow.length > 0 && (
        <div className="space-y-1 mt-2">
          {fieldsToShow.map((field) => {
            const value = record.values[field.id];
            if (value === null || value === undefined || value === '') return null;
            return (
              <div key={field.id} className="text-xs text-gray-500 truncate">
                <span className="font-medium text-gray-400">{field.name}:</span>{' '}
                {renderFieldValue(field, value)}
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}

function getCoverImage(record: TableRecord, coverField: Field | undefined): Attachment | null {
  if (!coverField || coverField.type !== 'attachment') return null;
  const attachments = record.values[coverField.id] as Attachment[] | undefined;
  if (!attachments || attachments.length === 0) return null;

  // Find first image attachment
  const imageAttachment = attachments.find((a) =>
    a.mime_type?.startsWith('image/')
  );
  return imageAttachment || null;
}

function getCardColor(record: TableRecord, colorField: Field | undefined): string | null {
  if (!colorField) return null;

  const value = record.values[colorField.id];
  if (!value) return null;

  if (colorField.type === 'single_select') {
    const choices = colorField.options?.choices || [];
    const choice = choices.find((c: { id: string }) => c.id === value);
    return choice?.color || null;
  }

  if (colorField.type === 'multi_select') {
    const values = value as string[];
    if (!values || values.length === 0) return null;
    const choices = colorField.options?.choices || [];
    const choice = choices.find((c: { id: string }) => values.includes(c.id));
    return choice?.color || null;
  }

  return null;
}

function renderFieldValue(field: Field, value: CellValue): string {
  if (value === null || value === undefined) return '';

  switch (field.type) {
    case 'checkbox':
      return value ? 'Yes' : 'No';
    case 'date':
      return new Date(value as string).toLocaleDateString();
    case 'datetime':
      return new Date(value as string).toLocaleString();
    case 'number':
      return (value as number).toLocaleString();
    case 'currency':
      const symbol = field.options?.currency_symbol || '$';
      return `${symbol}${(value as number).toLocaleString()}`;
    case 'percent':
      return `${value}%`;
    case 'rating':
      return '\u2605'.repeat(value as number);
    case 'single_select': {
      const choices = field.options?.choices || [];
      const choice = choices.find((c: { id: string }) => c.id === value);
      return choice?.name || String(value);
    }
    case 'multi_select': {
      const vals = value as string[];
      const choices = field.options?.choices || [];
      const names = vals
        .map((v) => choices.find((c: { id: string }) => c.id === v)?.name || v)
        .join(', ');
      return names;
    }
    case 'user':
      if (value && typeof value === 'object' && !Array.isArray(value) && 'name' in value) {
        return (value as { name: string }).name;
      }
      return String(value);
    case 'attachment':
      const attachments = value as Attachment[];
      return `${attachments.length} file${attachments.length !== 1 ? 's' : ''}`;
    default:
      return String(value);
  }
}
