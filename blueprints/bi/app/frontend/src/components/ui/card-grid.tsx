import { SimpleGrid, rem } from '@mantine/core'
import type { ReactNode } from 'react'

// =============================================================================
// CARD GRID - Responsive grid for cards
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
// PAGE CONTAINER - Standard page wrapper with consistent padding
// =============================================================================

import { Container, type MantineSize } from '@mantine/core'

export interface PageContainerProps {
  /** Page content */
  children: ReactNode
  /** Container max width */
  size?: MantineSize | 'fluid'
  /** Custom padding */
  padding?: MantineSize | number
  /** Remove horizontal padding on mobile */
  fluidOnMobile?: boolean
}

export function PageContainer({
  children,
  size = 'xl',
  padding = 'lg',
  fluidOnMobile = false,
}: PageContainerProps) {
  if (size === 'fluid') {
    return (
      <div
        style={{
          padding: typeof padding === 'number' ? rem(padding) : undefined,
          paddingLeft: fluidOnMobile ? undefined : rem(24),
          paddingRight: fluidOnMobile ? undefined : rem(24),
        }}
        data-padding={typeof padding === 'string' ? padding : undefined}
      >
        {children}
      </div>
    )
  }

  return (
    <Container
      size={size}
      py={padding}
      px={fluidOnMobile ? { base: 'xs', sm: padding } : padding}
    >
      {children}
    </Container>
  )
}

// =============================================================================
// SECTION - Content section with consistent spacing
// =============================================================================

import { Box } from '@mantine/core'
import { SectionHeader, type SectionHeaderProps } from './page-header'

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
        />
      )}
      {children}
    </Box>
  )
}

// =============================================================================
// CONTENT STACK - Vertical stack with consistent spacing
// =============================================================================

import { Stack, type StackProps } from '@mantine/core'

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

import { Group } from '@mantine/core'

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
