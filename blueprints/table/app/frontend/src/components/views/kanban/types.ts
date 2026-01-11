import type { TableRecord, Field, CellValue, Attachment } from '../../../types';

export interface KanbanColumn {
  id: string;
  name: string;
  color: string;
  records: TableRecord[];
  isCollapsed: boolean;
}

export interface KanbanCard {
  record: TableRecord;
  title: string;
  coverImage?: Attachment;
  displayFields: { field: Field; value: CellValue }[];
  color?: string;
}

export interface KanbanConfig {
  groupBy: string | null;
  cardFields: string[];
  coverField: string | null;
  cardSize: 'small' | 'medium' | 'large';
  cardCoverFit: 'contain' | 'cover';
  hideEmptyColumns: boolean;
  collapsedColumns: string[];
  cardColorField: string | null;
  // Airtable-compatible features
  keepSorted: boolean;                       // When true, use view sorts; when false, allow manual reordering
  hideEmptyFields: boolean;                  // Hide fields without values on cards
  cardPositions: Record<string, string[]>;   // Manual card positions per column (columnId -> recordIds)
}

export interface DragState {
  activeId: string | null;
  activeRecord: TableRecord | null;
  overColumnId: string | null;
}

export const DEFAULT_KANBAN_CONFIG: KanbanConfig = {
  groupBy: null,
  cardFields: [],
  coverField: null,
  cardSize: 'medium',
  cardCoverFit: 'cover',
  hideEmptyColumns: false,
  collapsedColumns: [],
  cardColorField: null,
  keepSorted: false,
  hideEmptyFields: false,
  cardPositions: {},
};

export const CARD_SIZES = {
  small: {
    height: 'auto',
    padding: 'p-2',
    titleSize: 'text-sm',
    fieldCount: 1,
    coverHeight: 'h-16',
  },
  medium: {
    height: 'auto',
    padding: 'p-3',
    titleSize: 'text-sm',
    fieldCount: 3,
    coverHeight: 'h-24',
  },
  large: {
    height: 'auto',
    padding: 'p-4',
    titleSize: 'text-base',
    fieldCount: 5,
    coverHeight: 'h-32',
  },
} as const;
