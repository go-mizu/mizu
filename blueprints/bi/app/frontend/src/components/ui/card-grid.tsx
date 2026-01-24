import { SimpleGrid, rem, Box, Stack, Group, type StackProps } from '@mantine/core'
import type { ReactNode } from 'react'
import type { MantineSize } from '@mantine/core'
import { SectionHeader, type SectionHeaderProps } from './page-header'

// =============================================================================
// CARD GRID - Responsive grid for cards (modern, full-width friendly)
// =============================================================================

export interface CardGridProps {
  /** Grid content (cards) */
  children: ReactNode
  /** Number of columns at different breakpoints */
  cols?: {
    base?: number
    xs?: number
    sm?: number
    md?: number
    lg?: number
    xl?: number
  }
  /** Gap between cards */
  gap?: 'xs' | 'sm' | 'md' | 'lg' | 'xl' | number
  /** Minimum card width (for auto-fit grid) */
  minCardWidth?: number
}

export function CardGrid({
  children,
  cols = { base: 1, sm: 2, md: 3, lg: 4 },
  gap = 'md',
}: CardGridProps) {
  return (
    <SimpleGrid
      cols={cols}
      spacing={gap}
      verticalSpacing={gap}
    >
      {children}
    </SimpleGrid>
  )
}

// =============================================================================
// PAGE CONTAINER - Full-width wrapper with consistent padding (shadcn-inspired)
// =============================================================================

export interface PageContainerProps {
  /** Page content */
  children: ReactNode
  /** Container max width - 'full' (default), 'xl', 'lg', 'md' */
  size?: 'full' | 'xl' | 'lg' | 'md'
  /** Vertical padding */
  py?: 'none' | 'sm' | 'md' | 'lg' | 'xl'
  /** Horizontal padding */
  px?: 'none' | 'sm' | 'md' | 'lg' | 'xl'
  /** Background color */
  bg?: 'default' | 'muted' | 'white'
  /** Min height */
  minHeight?: string
}

const sizesMap: Record<string, string> = {
  full: 'none',
  xl: '1400px',
  lg: '1200px',
  md: '960px',
}

const paddingMap: Record<string, string | number> = {
  none: 0,
  sm: rem(16),
  md: rem(24),
  lg: rem(32),
  xl: rem(48),
}

const bgMap: Record<string, string> = {
  default: 'var(--color-background-muted)',
  muted: 'var(--color-background-muted)',
  white: 'var(--color-background)',
}

export function PageContainer({
  children,
  size = 'full',
  py = 'lg',
  px = 'lg',
  bg = 'default',
  minHeight = '100vh',
}: PageContainerProps) {
  return (
    <Box
      style={{
        width: '100%',
        maxWidth: sizesMap[size],
        marginLeft: size !== 'full' ? 'auto' : undefined,
        marginRight: size !== 'full' ? 'auto' : undefined,
        paddingTop: paddingMap[py],
        paddingBottom: paddingMap[py],
        paddingLeft: paddingMap[px],
        paddingRight: paddingMap[px],
        backgroundColor: bgMap[bg],
        minHeight,
      }}
    >
      {children}
    </Box>
  )
}

// =============================================================================
// SECTION - Content section with consistent spacing (modern)
// =============================================================================

export interface SectionProps extends Partial<SectionHeaderProps> {
  /** Section content */
  children: ReactNode
  /** Whether to show header */
  showHeader?: boolean
  /** Bottom margin */
  mb?: MantineSize | number
}

export function Section({
  children,
  showHeader = true,
  title,
  icon,
  count,
  actions,
  size,
  mb = 'xl',
}: SectionProps) {
  return (
    <Box mb={mb}>
      {showHeader && title && (
        <SectionHeader
          title={title}
          icon={icon}
          count={count}
          actions={actions}
          size={size}
        />
      )}
      {children}
    </Box>
  )
}

// =============================================================================
// CONTENT STACK - Vertical stack with consistent spacing
// =============================================================================

export interface ContentStackProps extends Omit<StackProps, 'gap'> {
  /** Gap size */
  gap?: 'xs' | 'sm' | 'md' | 'lg' | 'xl' | number
}

export function ContentStack({
  children,
  gap = 'lg',
  ...props
}: ContentStackProps) {
  return (
    <Stack gap={gap} {...props}>
      {children}
    </Stack>
  )
}

// =============================================================================
// SPLIT PANEL - Two-column layout
// =============================================================================

export interface SplitPanelProps {
  /** Left panel content */
  left: ReactNode
  /** Right panel content */
  right: ReactNode
  /** Left panel width */
  leftWidth?: number | string
  /** Gap between panels */
  gap?: MantineSize | number
  /** Collapse on mobile */
  collapseOnMobile?: boolean
}

export function SplitPanel({
  left,
  right,
  leftWidth = 320,
  gap = 'md',
  collapseOnMobile = true,
}: SplitPanelProps) {
  return (
    <Group
      align="flex-start"
      gap={gap}
      wrap={collapseOnMobile ? 'wrap' : 'nowrap'}
      style={{ width: '100%' }}
    >
      <Box
        style={{
          width: typeof leftWidth === 'number' ? rem(leftWidth) : leftWidth,
          flexShrink: 0,
        }}
        w={collapseOnMobile ? { base: '100%', md: leftWidth } : leftWidth}
      >
        {left}
      </Box>
      <Box style={{ flex: 1, minWidth: 0 }}>
        {right}
      </Box>
    </Group>
  )
}
