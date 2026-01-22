import { useMemo } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  Box, Title, Text, Group, Stack, Button, SimpleGrid,
  Skeleton, ThemeIcon, ActionIcon, Menu, Tooltip, rem,
  UnstyledButton, Badge, Paper
} from '@mantine/core'
import {
  IconChartLine, IconLayoutDashboard, IconFolder, IconDatabase,
  IconPlus, IconDots, IconStar, IconClock,
  IconArrowRight, IconSearch, IconBookmark, IconStarFilled,
  IconSparkles, IconPencil
} from '@tabler/icons-react'
import { useQuestions, useDashboards, useCollections, useDataSources } from '../api/hooks'
import { useUIStore } from '../stores/uiStore'
import { useBookmarkStore, usePinActions } from '../stores/bookmarkStore'
import { chartColors, semanticColors } from '../theme'

// =============================================================================
// STYLES - Metabase Light Theme
// =============================================================================

const styles = {
  container: {
    padding: rem(32),
    maxWidth: 1200,
    margin: '0 auto',
    backgroundColor: semanticColors.bgLight,
    minHeight: '100vh',
  },
  header: {
    marginBottom: rem(32),
  },
  section: {
    marginBottom: rem(40),
  },
  sectionHeader: {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'space-between',
    marginBottom: rem(16),
  },
  sectionTitle: {
    fontSize: rem(13),
    fontWeight: 700,
    color: semanticColors.textSecondary,
    textTransform: 'uppercase' as const,
    letterSpacing: '0.05em',
    display: 'flex',
    alignItems: 'center',
    gap: rem(8),
  },
  card: {
    backgroundColor: '#ffffff',
    border: `1px solid ${semanticColors.borderMedium}`,
    borderRadius: rem(8),
    padding: rem(20),
    cursor: 'pointer',
    transition: 'all 0.15s ease',
    height: '100%',
  },
  cardHover: {
    boxShadow: '0 4px 12px rgba(0, 0, 0, 0.08)',
    transform: 'translateY(-2px)',
  },
  startCard: {
    backgroundColor: '#ffffff',
    border: `1px solid ${semanticColors.borderMedium}`,
    borderRadius: rem(8),
    padding: rem(24),
    display: 'flex',
    alignItems: 'flex-start',
    gap: rem(16),
    cursor: 'pointer',
    transition: 'all 0.15s ease',
  },
  emptyState: {
    backgroundColor: '#ffffff',
    border: `1px solid ${semanticColors.borderMedium}`,
    borderRadius: rem(8),
    padding: rem(64),
    textAlign: 'center' as const,
  },
  statCard: {
    backgroundColor: '#ffffff',
    border: `1px solid ${semanticColors.borderMedium}`,
    borderRadius: rem(8),
    padding: rem(20),
    cursor: 'pointer',
    transition: 'all 0.15s ease',
  },
}

// =============================================================================
// MAIN COMPONENT
// =============================================================================

export default function Home() {
  const navigate = useNavigate()
  const { openCommandPalette } = useUIStore()
  const { pinnedItems, recentItems } = useBookmarkStore()

  const { data: questions, isLoading: questionsLoading } = useQuestions()
  const { data: dashboards, isLoading: dashboardsLoading } = useDashboards()
  const { data: collections, isLoading: collectionsLoading } = useCollections()
  const { data: datasources, isLoading: datasourcesLoading } = useDataSources()

  const isLoading = questionsLoading || dashboardsLoading || collectionsLoading || datasourcesLoading
  const hasData = (questions?.length || 0) > 0 || (dashboards?.length || 0) > 0

  // Get pinned dashboards and questions
  const pinnedDashboards = useMemo(() => {
    return pinnedItems
      .filter(p => p.type === 'dashboard')
      .map(p => dashboards?.find(d => d.id === p.id))
      .filter(Boolean)
  }, [pinnedItems, dashboards])

  const pinnedQuestions = useMemo(() => {
    return pinnedItems
      .filter(p => p.type === 'question')
      .map(p => questions?.find(q => q.id === p.id))
      .filter(Boolean)
  }, [pinnedItems, questions])

  const hasPinnedItems = pinnedDashboards.length > 0 || pinnedQuestions.length > 0

  // Recent items from the store
  const recentDashboards = useMemo(() => {
    return recentItems
      .filter(item => item.type === 'dashboard')
      .slice(0, 4)
      .map(item => dashboards?.find(d => d.id === item.id))
      .filter(Boolean)
  }, [recentItems, dashboards])

  const recentQuestions = useMemo(() => {
    return recentItems
      .filter(item => item.type === 'question')
      .slice(0, 6)
      .map(item => questions?.find(q => q.id === item.id))
      .filter(Boolean)
  }, [recentItems, questions])

  const hasRecentItems = recentDashboards.length > 0 || recentQuestions.length > 0

  // Loading state
  if (isLoading && !hasData) {
    return (
      <Box style={styles.container}>
        <Box style={styles.header}>
          <Skeleton height={32} width={200} mb="sm" />
          <Skeleton height={20} width={300} />
        </Box>
        <SimpleGrid cols={{ base: 1, sm: 2, md: 3 }}>
          {[1, 2, 3].map((i) => (
            <Skeleton key={i} height={120} radius="md" />
          ))}
        </SimpleGrid>
      </Box>
    )
  }

  return (
    <Box style={styles.container}>
      {/* Header */}
      <Box style={styles.header}>
        <Group justify="space-between" align="flex-start">
          <Box>
            <Title order={2} style={{ color: semanticColors.textPrimary, fontWeight: 700 }}>
              Home
            </Title>
            <Text c="dimmed" size="sm" mt={4}>
              Welcome back! Here's what's happening with your data.
            </Text>
          </Box>
          <Group gap="sm">
            <Button
              variant="subtle"
              leftSection={<IconSearch size={16} strokeWidth={1.75} />}
              onClick={openCommandPalette}
              color="gray"
              styles={{
                root: {
                  fontWeight: 500,
                }
              }}
            >
              Search
            </Button>
            <Menu position="bottom-end" shadow="md">
              <Menu.Target>
                <Button leftSection={<IconPlus size={16} strokeWidth={2} />}>
                  New
                </Button>
              </Menu.Target>
              <Menu.Dropdown>
                <Menu.Item
                  leftSection={<IconPencil size={16} color={semanticColors.brand} strokeWidth={1.75} />}
                  onClick={() => navigate('/question/new')}
                >
                  Question
                </Menu.Item>
                <Menu.Item
                  leftSection={<IconLayoutDashboard size={16} color={semanticColors.summarize} strokeWidth={1.75} />}
                  onClick={() => navigate('/dashboard/new')}
                >
                  Dashboard
                </Menu.Item>
                <Menu.Divider />
                <Menu.Item
                  leftSection={<IconFolder size={16} color="#F9D45C" strokeWidth={1.75} />}
                  onClick={() => navigate('/collection/new')}
                >
                  Collection
                </Menu.Item>
              </Menu.Dropdown>
            </Menu>
          </Group>
        </Group>
      </Box>

      {/* Start Here Section (for new users) */}
      {!hasData && (
        <Box style={styles.section}>
          <Box style={styles.sectionHeader}>
            <Text style={styles.sectionTitle}>
              <IconSparkles size={16} color={semanticColors.brand} strokeWidth={1.75} />
              Start here
            </Text>
          </Box>
          <SimpleGrid cols={{ base: 1, sm: 2, md: 3 }}>
            <StartCard
              icon={IconDatabase}
              color="#F9D45C"
              title="Add your data"
              description="Connect to a database to start exploring your data"
              onClick={() => navigate('/admin/datamodel')}
            />
            <StartCard
              icon={IconPencil}
              color={semanticColors.brand}
              title="Ask a question"
              description="Create visualizations and insights from your data"
              onClick={() => navigate('/question/new')}
            />
            <StartCard
              icon={IconLayoutDashboard}
              color={semanticColors.summarize}
              title="Create a dashboard"
              description="Combine multiple questions into a dashboard"
              onClick={() => navigate('/dashboard/new')}
            />
          </SimpleGrid>
        </Box>
      )}

      {/* Pinned Items Section */}
      {hasPinnedItems && (
        <Box style={styles.section}>
          <Box style={styles.sectionHeader}>
            <Text style={styles.sectionTitle}>
              <IconStarFilled size={14} color="#F9D45C" />
              Pinned
            </Text>
          </Box>
          <SimpleGrid cols={{ base: 1, sm: 2, md: 4 }}>
            {pinnedDashboards.map((dashboard: any, index: number) => (
              <ItemCard
                key={dashboard.id}
                id={dashboard.id}
                type="dashboard"
                name={dashboard.name}
                description={dashboard.description}
                cardCount={dashboard.cards?.length}
                colorIndex={index}
                onClick={() => navigate(`/dashboard/${dashboard.id}`)}
              />
            ))}
            {pinnedQuestions.map((question: any, index: number) => (
              <ItemCard
                key={question.id}
                id={question.id}
                type="question"
                name={question.name}
                description={question.description}
                vizType={question.visualization?.type}
                colorIndex={pinnedDashboards.length + index}
                onClick={() => navigate(`/question/${question.id}`)}
              />
            ))}
          </SimpleGrid>
        </Box>
      )}

      {/* Pick Up Where You Left Off Section */}
      {hasRecentItems && (
        <Box style={styles.section}>
          <Box style={styles.sectionHeader}>
            <Text style={styles.sectionTitle}>
              <IconClock size={14} color={semanticColors.textSecondary} strokeWidth={1.75} />
              Pick up where you left off
            </Text>
            <Button
              variant="subtle"
              size="xs"
              rightSection={<IconArrowRight size={14} />}
              onClick={() => navigate('/browse')}
              color="gray"
            >
              View all
            </Button>
          </Box>
          <SimpleGrid cols={{ base: 1, sm: 2, md: 3, lg: 4 }}>
            {[...recentDashboards, ...recentQuestions].slice(0, 8).map((item: any, index: number) => {
              const isDashboard = 'cards' in item
              return (
                <ItemCard
                  key={item.id}
                  id={item.id}
                  type={isDashboard ? 'dashboard' : 'question'}
                  name={item.name}
                  description={item.description}
                  cardCount={isDashboard ? item.cards?.length : undefined}
                  vizType={!isDashboard ? item.visualization?.type : undefined}
                  colorIndex={index}
                  onClick={() => navigate(`/${isDashboard ? 'dashboard' : 'question'}/${item.id}`)}
                  compact
                />
              )
            })}
          </SimpleGrid>
        </Box>
      )}

      {/* Our Analytics Section */}
      {hasData && (
        <Box style={styles.section}>
          <Box style={styles.sectionHeader}>
            <Text style={styles.sectionTitle}>
              <IconChartLine size={14} color={semanticColors.brand} strokeWidth={1.75} />
              Our analytics
            </Text>
            <Button
              variant="subtle"
              size="xs"
              rightSection={<IconArrowRight size={14} />}
              onClick={() => navigate('/browse')}
              color="gray"
            >
              Browse all
            </Button>
          </Box>
          <SimpleGrid cols={{ base: 2, sm: 4 }}>
            <StatCard
              label="Questions"
              value={questions?.length || 0}
              icon={IconChartLine}
              color={semanticColors.brand}
              onClick={() => navigate('/browse')}
            />
            <StatCard
              label="Dashboards"
              value={dashboards?.length || 0}
              icon={IconLayoutDashboard}
              color={semanticColors.summarize}
              onClick={() => navigate('/browse')}
            />
            <StatCard
              label="Collections"
              value={collections?.length || 0}
              icon={IconFolder}
              color={semanticColors.filter}
              onClick={() => navigate('/browse')}
            />
            <StatCard
              label="Databases"
              value={datasources?.length || 0}
              icon={IconDatabase}
              color="#F9D45C"
              onClick={() => navigate('/admin/datamodel')}
            />
          </SimpleGrid>
        </Box>
      )}

      {/* Empty State */}
      {!hasData && !isLoading && (datasources?.length || 0) === 0 && (
        <Box style={styles.emptyState}>
          <Stack align="center" gap="lg">
            <ThemeIcon size={80} radius="xl" variant="light" color="brand">
              <IconSparkles size={40} strokeWidth={1.5} />
            </ThemeIcon>
            <Box>
              <Title order={3} mb="xs" style={{ color: semanticColors.textPrimary }}>
                Ready to explore your data?
              </Title>
              <Text c="dimmed" maw={400} mx="auto">
                Connect to a database to start creating questions and dashboards, or run <code style={{ background: '#F0F0F0', padding: '2px 6px', borderRadius: 4 }}>bi seed</code> to add sample data.
              </Text>
            </Box>
            <Group>
              <Button
                size="lg"
                leftSection={<IconDatabase size={20} strokeWidth={1.75} />}
                onClick={() => navigate('/admin/datamodel')}
              >
                Add a database
              </Button>
              <Button
                size="lg"
                variant="light"
                leftSection={<IconPlus size={20} strokeWidth={2} />}
                onClick={() => navigate('/question/new')}
              >
                New question
              </Button>
            </Group>
          </Stack>
        </Box>
      )}
    </Box>
  )
}

// =============================================================================
// SUB-COMPONENTS
// =============================================================================

// Start Card for onboarding
function StartCard({
  icon: Icon,
  color,
  title,
  description,
  onClick,
}: {
  icon: typeof IconDatabase
  color: string
  title: string
  description: string
  onClick: () => void
}) {
  return (
    <UnstyledButton
      onClick={onClick}
      style={styles.startCard}
      onMouseEnter={(e) => {
        e.currentTarget.style.boxShadow = '0 4px 12px rgba(0, 0, 0, 0.08)'
        e.currentTarget.style.transform = 'translateY(-2px)'
      }}
      onMouseLeave={(e) => {
        e.currentTarget.style.boxShadow = 'none'
        e.currentTarget.style.transform = 'translateY(0)'
      }}
    >
      <ThemeIcon size={48} radius="md" style={{ backgroundColor: `${color}15` }}>
        <Icon size={24} color={color} strokeWidth={1.75} />
      </ThemeIcon>
      <Box style={{ flex: 1 }}>
        <Text fw={600} size="md" style={{ color: semanticColors.textPrimary }} mb={4}>
          {title}
        </Text>
        <Text size="sm" c="dimmed" lh={1.4}>
          {description}
        </Text>
      </Box>
    </UnstyledButton>
  )
}

// Stat Card
function StatCard({
  label,
  value,
  icon: Icon,
  color,
  onClick,
}: {
  label: string
  value: number
  icon: typeof IconChartLine
  color: string
  onClick: () => void
}) {
  return (
    <UnstyledButton
      onClick={onClick}
      style={styles.statCard}
      onMouseEnter={(e) => {
        e.currentTarget.style.boxShadow = '0 4px 12px rgba(0, 0, 0, 0.08)'
      }}
      onMouseLeave={(e) => {
        e.currentTarget.style.boxShadow = 'none'
      }}
    >
      <Group gap="md">
        <ThemeIcon size={44} radius="md" style={{ backgroundColor: `${color}15` }}>
          <Icon size={22} color={color} strokeWidth={1.75} />
        </ThemeIcon>
        <Box>
          <Text size="xl" fw={700} style={{ color: semanticColors.textPrimary, lineHeight: 1.2 }}>
            {value}
          </Text>
          <Text size="sm" c="dimmed" mt={2}>
            {label}
          </Text>
        </Box>
      </Group>
    </UnstyledButton>
  )
}

// Item Card (Question/Dashboard)
function ItemCard({
  id,
  type,
  name,
  description,
  vizType,
  cardCount,
  colorIndex,
  onClick,
  compact = false,
}: {
  id: string
  type: 'dashboard' | 'question'
  name: string
  description?: string
  vizType?: string
  cardCount?: number
  colorIndex: number
  onClick: () => void
  compact?: boolean
}) {
  const { pinned, toggle: togglePin } = usePinActions(id, type)
  const Icon = type === 'dashboard' ? IconLayoutDashboard : IconChartLine
  const accentColor = chartColors[colorIndex % chartColors.length]

  const getVizLabel = (vizType: string) => {
    switch (vizType) {
      case 'line': return 'Line'
      case 'bar': return 'Bar'
      case 'pie': return 'Pie'
      case 'number': return 'Number'
      case 'table': return 'Table'
      case 'area': return 'Area'
      default: return vizType
    }
  }

  return (
    <Paper
      withBorder
      p={compact ? 'sm' : 'md'}
      radius="md"
      style={{
        cursor: 'pointer',
        transition: 'all 0.15s ease',
        height: '100%',
      }}
      onClick={onClick}
      onMouseEnter={(e) => {
        e.currentTarget.style.boxShadow = '0 4px 12px rgba(0, 0, 0, 0.08)'
        e.currentTarget.style.transform = 'translateY(-2px)'
      }}
      onMouseLeave={(e) => {
        e.currentTarget.style.boxShadow = 'none'
        e.currentTarget.style.transform = 'translateY(0)'
      }}
    >
      <Group justify="space-between" mb={compact ? 'xs' : 'sm'}>
        <ThemeIcon
          size={compact ? 32 : 40}
          radius="md"
          style={{
            backgroundColor: `${accentColor}15`,
          }}
        >
          <Icon size={compact ? 16 : 20} color={accentColor} strokeWidth={1.75} />
        </ThemeIcon>
        <Group gap={4}>
          <Tooltip label={pinned ? 'Unpin' : 'Pin to home'}>
            <ActionIcon
              variant="subtle"
              color={pinned ? 'yellow' : 'gray'}
              size="sm"
              onClick={(e) => {
                e.stopPropagation()
                togglePin()
              }}
            >
              {pinned ? <IconStarFilled size={14} /> : <IconStar size={14} strokeWidth={1.75} />}
            </ActionIcon>
          </Tooltip>
          <Menu position="bottom-end" withinPortal>
            <Menu.Target>
              <ActionIcon variant="subtle" color="gray" size="sm" onClick={(e) => e.stopPropagation()}>
                <IconDots size={14} />
              </ActionIcon>
            </Menu.Target>
            <Menu.Dropdown>
              <Menu.Item leftSection={<IconBookmark size={14} strokeWidth={1.75} />}>
                Bookmark
              </Menu.Item>
              <Menu.Item leftSection={<IconStar size={14} strokeWidth={1.75} />} onClick={togglePin}>
                {pinned ? 'Unpin from home' : 'Pin to home'}
              </Menu.Item>
            </Menu.Dropdown>
          </Menu>
        </Group>
      </Group>
      <Text fw={600} lineClamp={compact ? 1 : 2} size={compact ? 'sm' : 'md'} style={{ color: semanticColors.textPrimary }}>
        {name}
      </Text>
      {!compact && description && (
        <Text size="sm" c="dimmed" lineClamp={1} mt={4}>
          {description}
        </Text>
      )}
      <Group gap="xs" mt={compact ? 'xs' : 'sm'}>
        {vizType && (
          <Badge size="xs" variant="light" color="gray" radius="sm">
            {getVizLabel(vizType)}
          </Badge>
        )}
        {cardCount !== undefined && (
          <Badge size="xs" variant="light" color="gray" radius="sm">
            {cardCount} {cardCount === 1 ? 'card' : 'cards'}
          </Badge>
        )}
        {type === 'dashboard' && !cardCount && (
          <Badge size="xs" variant="light" color="gray" radius="sm">
            Dashboard
          </Badge>
        )}
        {type === 'question' && !vizType && (
          <Badge size="xs" variant="light" color="gray" radius="sm">
            Question
          </Badge>
        )}
      </Group>
    </Paper>
  )
}
