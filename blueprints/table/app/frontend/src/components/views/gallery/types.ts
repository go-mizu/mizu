import type { TableRecord, Field, CellValue, Attachment } from '../../../types';

export interface GalleryConfig {
  coverField: string | null;
  titleField: string | null;
  cardFields: string[];
  cardSize: 'small' | 'medium' | 'large';
  cardCoverFit: 'cover' | 'contain';
  cardColorField: string | null;
  showEmptyCards: boolean;
  aspectRatio: '16:9' | '4:3' | '1:1' | '3:2';
}

export interface GalleryCard {
  record: TableRecord;
  title: string;
  coverImage?: Attachment;
  displayFields: { field: Field; value: CellValue }[];
  color?: string;
}

export const DEFAULT_GALLERY_CONFIG: GalleryConfig = {
  coverField: null,
  titleField: null,
  cardFields: [],
  cardSize: 'medium',
  cardCoverFit: 'cover',
  cardColorField: null,
  showEmptyCards: true,
  aspectRatio: '16:9',
};

export const CARD_SIZES = {
  small: {
    gridCols: 'grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6',
    fieldCount: 2,
    titleSize: 'text-sm',
    padding: 'p-2',
    gap: 'gap-3',
  },
  medium: {
    gridCols: 'grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5',
    fieldCount: 4,
    titleSize: 'text-base',
    padding: 'p-3',
    gap: 'gap-4',
  },
  large: {
    gridCols: 'grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4',
    fieldCount: 6,
    titleSize: 'text-lg',
    padding: 'p-4',
    gap: 'gap-5',
  },
} as const;

export const ASPECT_RATIOS = {
  '16:9': 'aspect-video',
  '4:3': 'aspect-[4/3]',
  '1:1': 'aspect-square',
  '3:2': 'aspect-[3/2]',
} as const;

export type CardSize = keyof typeof CARD_SIZES;
export type AspectRatio = keyof typeof ASPECT_RATIOS;
