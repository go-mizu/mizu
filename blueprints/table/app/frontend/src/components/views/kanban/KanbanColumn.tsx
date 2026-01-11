import { useDroppable } from '@dnd-kit/core';
import { SortableContext, verticalListSortingStrategy } from '@dnd-kit/sortable';
import type { TableRecord, Field } from '../../../types';
import { KanbanCard } from './KanbanCard';
import type { KanbanColumn as KanbanColumnType, KanbanConfig } from './types';

interface KanbanColumnProps {
  column: KanbanColumnType;
  primaryField: Field | undefined;
  displayFields: Field[];
  coverField: Field | undefined;
  colorField: Field | undefined;
  config: KanbanConfig;
  onAddCard: (columnId: string) => void;
  onCardClick: (record: TableRecord) => void;
  onToggleCollapse: (columnId: string) => void;
  isOver: boolean;
}

export function KanbanColumn({
  column,
  primaryField,
  displayFields,
  coverField,
  colorField,
  config,
  onAddCard,
  onCardClick,
  onToggleCollapse,
  isOver,
}: KanbanColumnProps) {
  const { setNodeRef } = useDroppable({
    id: column.id,
    data: {
      type: 'column',
      column,
    },
  });

  // Collapsed column view
  if (column.isCollapsed) {
    return (
      <div
        onClick={() => onToggleCollapse(column.id)}
        className="flex-shrink-0 w-10 bg-slate-100 rounded-xl border border-slate-200 flex flex-col items-center py-3 cursor-pointer hover:bg-slate-200 transition-colors"
      >
        <span
          className="w-3 h-3 rounded-full mb-3"
          style={{ backgroundColor: column.color }}
        />
        <span className="text-xs font-medium text-gray-600 [writing-mode:vertical-rl] rotate-180">
          {column.name}
        </span>
        <span className="text-xs text-slate-500 mt-2 bg-white border border-slate-200 rounded-full px-1.5 py-0.5">
          {column.records.length}
        </span>
      </div>
    );
  }

  // Full column view
  return (
    <div
      ref={setNodeRef}
      className={`
        flex-shrink-0 w-72 bg-slate-50 rounded-xl border border-slate-200 flex flex-col
        ${isOver ? 'ring-2 ring-primary bg-slate-100' : ''}
        transition-all duration-150
      `}
    >
      {/* Column header */}
      <div className="p-3 flex items-center gap-2 border-b border-slate-200">
        <span
          className="w-3 h-3 rounded-full flex-shrink-0"
          style={{ backgroundColor: column.color }}
        />
        <span className="font-semibold text-gray-900 truncate flex-1">{column.name}</span>
        <span className="text-xs text-slate-500 bg-white border border-slate-200 rounded-full px-2 py-0.5 flex-shrink-0">
          {column.records.length}
        </span>
        <button
          onClick={(e) => {
            e.stopPropagation();
            onToggleCollapse(column.id);
          }}
          className="p-1 text-slate-400 hover:text-slate-600 hover:bg-slate-200 rounded transition-colors"
          title="Collapse column"
        >
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 19l-7-7 7-7m8 14l-7-7 7-7" />
          </svg>
        </button>
      </div>

      {/* Cards container */}
      <div className="flex-1 overflow-y-auto p-2">
        <SortableContext
          items={column.records.map((r) => r.id)}
          strategy={verticalListSortingStrategy}
        >
          <div className="space-y-2">
            {column.records.map((record) => (
              <KanbanCard
                key={record.id}
                record={record}
                primaryField={primaryField}
                displayFields={displayFields}
                coverField={coverField}
                colorField={colorField}
                config={config}
                hideEmptyFields={config.hideEmptyFields}
                onClick={() => onCardClick(record)}
              />
            ))}
          </div>
        </SortableContext>

        {/* Drop zone indicator when empty */}
        {column.records.length === 0 && (
          <div
            className={`
              h-24 border-2 border-dashed rounded-lg flex items-center justify-center
              ${isOver ? 'border-primary bg-primary/5' : 'border-slate-200'}
              transition-colors
            `}
          >
            <span className="text-sm text-slate-400">
              {isOver ? 'Drop here' : 'No cards'}
            </span>
          </div>
        )}

        {/* Add card button */}
        <button
          onClick={() => onAddCard(column.id)}
          className="w-full mt-2 p-2 text-sm text-slate-500 hover:bg-white rounded-lg flex items-center justify-center gap-1 transition-colors border border-transparent hover:border-slate-200"
        >
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
          </svg>
          Add card
        </button>
      </div>
    </div>
  );
}
