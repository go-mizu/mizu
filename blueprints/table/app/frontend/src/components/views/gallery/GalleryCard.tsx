import type { TableRecord, Field, Attachment } from '../../../types';
import type { GalleryConfig } from './types';
import { CARD_SIZES, ASPECT_RATIOS } from './types';

interface GalleryCardProps {
  record: TableRecord;
  primaryField?: Field;
  displayFields: Field[];
  coverField?: Field;
  colorField?: Field;
  config: GalleryConfig;
  onClick: () => void;
  isSelected?: boolean;
}

export function GalleryCard({
  record,
  primaryField,
  displayFields,
  coverField,
  colorField,
  config,
  onClick,
  isSelected,
}: GalleryCardProps) {
  const sizeConfig = CARD_SIZES[config.cardSize];
  const aspectClass = ASPECT_RATIOS[config.aspectRatio];

  // Get cover image
  const getCoverImage = (): Attachment | null => {
    if (!coverField) return null;
    const attachments = record.values[coverField.id] as Attachment[] | undefined;
    return attachments?.[0] || null;
  };

  // Get card title
  const getTitle = (): string => {
    if (!primaryField) return 'Untitled';
    const value = record.values[primaryField.id];
    return value ? String(value) : 'Untitled';
  };

  // Get color from color field
  const getCardColor = (): string | null => {
    if (!colorField) return null;
    const value = record.values[colorField.id];
    if (!value) return null;

    if (colorField.type === 'single_select') {
      const choice = colorField.options?.choices?.find(
        (c: { id: string }) => c.id === value
      );
      return choice?.color || null;
    }
    if (colorField.type === 'multi_select' && Array.isArray(value)) {
      const firstValue = value[0];
      if (typeof firstValue === 'string') {
        const choice = colorField.options?.choices?.find(
          (c: { id: string }) => c.id === firstValue
        );
        return choice?.color || null;
      }
    }
    return null;
  };

  // Get attachment count
  const getAttachmentCount = (): number => {
    if (!coverField) return 0;
    const attachments = record.values[coverField.id] as Attachment[] | undefined;
    return attachments?.length || 0;
  };

  const coverImage = getCoverImage();
  const cardColor = getCardColor();
  const attachmentCount = getAttachmentCount();
  const visibleFields = displayFields.slice(0, sizeConfig.fieldCount);

  return (
    <div
      onClick={onClick}
      className={`
        bg-white rounded-xl border overflow-hidden cursor-pointer
        hover:shadow-lg transition-all duration-200 group
        ${isSelected ? 'ring-2 ring-primary border-primary' : 'border-slate-200'}
      `}
    >
      {/* Color accent bar */}
      {cardColor && (
        <div
          className="h-1"
          style={{ backgroundColor: cardColor }}
        />
      )}

      {/* Cover image */}
      <div className={`${aspectClass} bg-slate-100 relative overflow-hidden`}>
        {coverImage ? (
          <img
            src={coverImage.url}
            alt=""
            className={`w-full h-full transition-transform duration-300 group-hover:scale-105 ${
              config.cardCoverFit === 'contain' ? 'object-contain' : 'object-cover'
            }`}
          />
        ) : (
          <div className="w-full h-full flex items-center justify-center">
            <svg className="w-12 h-12 text-gray-300" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
            </svg>
          </div>
        )}

        {/* Attachment count badge */}
        {attachmentCount > 1 && (
          <div className="absolute bottom-2 right-2 bg-black/60 text-white text-xs px-2 py-0.5 rounded-full flex items-center gap-1">
            <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14" />
            </svg>
            {attachmentCount}
          </div>
        )}
      </div>

      {/* Card content */}
      <div className={sizeConfig.padding}>
        <h3 className={`font-semibold text-gray-900 mb-2 line-clamp-2 ${sizeConfig.titleSize}`}>
          {getTitle()}
        </h3>

        {visibleFields.length > 0 && (
          <div className="space-y-1.5">
            {visibleFields.map((field) => {
              const value = record.values[field.id];
              if (value === null || value === undefined) return null;

              return (
                <div key={field.id} className="flex items-start gap-2 text-sm">
                  <span className="text-slate-500 flex-shrink-0 truncate max-w-[80px]">
                    {field.name}:
                  </span>
                  <span className="text-gray-900 flex-1 min-w-0">
                    <FieldValue field={field} value={value} />
                  </span>
                </div>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}

interface FieldValueProps {
  field: Field;
  value: unknown;
}

function FieldValue({ field, value }: FieldValueProps) {
  switch (field.type) {
    case 'checkbox':
      return value ? (
        <span className="inline-flex items-center text-green-600">
          <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
            <path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd" />
          </svg>
        </span>
      ) : (
        <span className="text-slate-300">-</span>
      );

    case 'date':
    case 'datetime':
      try {
        return <span className="truncate">{new Date(value as string).toLocaleDateString()}</span>;
      } catch {
        return <span className="truncate">{String(value)}</span>;
      }

    case 'number':
      return <span className="truncate">{(value as number).toLocaleString()}</span>;

    case 'currency':
      return <span className="truncate">${(value as number).toLocaleString()}</span>;

    case 'percent':
      const percentValue = value as number;
      return (
        <span className="inline-flex items-center gap-1.5">
          <div className="flex-1 h-1.5 bg-slate-200 rounded-full overflow-hidden max-w-[60px]">
            <div
              className="h-full bg-primary rounded-full"
              style={{ width: `${Math.min(100, Math.max(0, percentValue))}%` }}
            />
          </div>
          <span className="text-xs text-slate-500">{percentValue}%</span>
        </span>
      );

    case 'rating':
      const rating = value as number;
      const max = field.options?.max || 5;
      return (
        <span className="text-amber-400">
          {'★'.repeat(Math.min(rating, max))}
          <span className="text-slate-200">{'★'.repeat(Math.max(0, max - rating))}</span>
        </span>
      );

    case 'single_select':
      const choice = field.options?.choices?.find(
        (c: { id: string }) => c.id === value
      );
      if (choice) {
        return (
          <span
            className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium"
            style={{
              backgroundColor: `${choice.color}20`,
              color: choice.color,
            }}
          >
            {choice.name}
          </span>
        );
      }
      return <span className="truncate">{String(value)}</span>;

    case 'multi_select':
      if (Array.isArray(value)) {
        const selectedChoices = value
          .slice(0, 2)
          .map((v) => {
            if (typeof v === 'string') {
              return field.options?.choices?.find((c: { id: string }) => c.id === v);
            }
            return null;
          })
          .filter((c): c is { id: string; name: string; color: string } => c !== null && c !== undefined);

        return (
          <span className="flex flex-wrap gap-1">
            {selectedChoices.map((choice) => (
              <span
                key={choice.id}
                className="inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium"
                style={{
                  backgroundColor: `${choice.color}20`,
                  color: choice.color,
                }}
              >
                {choice.name}
              </span>
            ))}
            {value.length > 2 && (
              <span className="text-xs text-slate-400">+{value.length - 2}</span>
            )}
          </span>
        );
      }
      return <span className="truncate">{String(value)}</span>;

    case 'email':
      return (
        <span className="truncate text-primary hover:underline">
          {String(value)}
        </span>
      );

    case 'url':
      return (
        <span className="truncate text-primary hover:underline">
          {String(value).replace(/^https?:\/\//, '')}
        </span>
      );

    case 'phone':
      return <span className="truncate">{String(value)}</span>;

    case 'user':
      return (
        <span className="inline-flex items-center gap-1">
          <span className="w-5 h-5 rounded-full bg-primary/20 text-primary text-xs flex items-center justify-center font-medium">
            {String(value).charAt(0).toUpperCase()}
          </span>
          <span className="truncate text-xs">{String(value).split('@')[0]}</span>
        </span>
      );

    default:
      return <span className="truncate">{String(value)}</span>;
  }
}
