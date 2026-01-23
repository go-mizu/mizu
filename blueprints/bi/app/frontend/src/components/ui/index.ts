// =============================================================================
// UI COMPONENTS - Modern shadcn-inspired component library
// =============================================================================

// Page Layout
export { PageHeader, SectionHeader } from './page-header'
export type { PageHeaderProps, SectionHeaderProps } from './page-header'

// Grid & Layout
export {
  CardGrid,
  PageContainer,
  Section,
  ContentStack,
  SplitPanel,
} from './card-grid'
export type {
  CardGridProps,
  PageContainerProps,
  SectionProps,
  ContentStackProps,
  SplitPanelProps,
} from './card-grid'

// Data Display Cards
export { DataCard, DataCardSkeleton } from './data-card'
export type { DataCardProps, DataCardType } from './data-card'

// Stat Cards
export { StatCard, StatCardSkeleton, MiniStat } from './stat-card'
export type { StatCardProps, MiniStatProps } from './stat-card'

// Empty & Loading States
export {
  EmptyState,
  InlineEmptyState,
  LoadingState,
  ErrorState,
} from './empty-state'
export type {
  EmptyStateProps,
  InlineEmptyStateProps,
  LoadingStateProps,
  ErrorStateProps,
} from './empty-state'

// Dialogs
export {
  ConfirmDialog,
  useConfirmDialog,
} from './confirm-dialog'
export type {
  ConfirmDialogProps,
  ConfirmDialogVariant,
  UseConfirmDialogOptions,
} from './confirm-dialog'
