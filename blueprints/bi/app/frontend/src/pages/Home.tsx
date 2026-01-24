import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  Box, Title, Text, Group, Button, Skeleton, rem, Paper
} from '@mantine/core'
import {
  IconChartLine, IconLayoutDashboard, IconFolder, IconDatabase,
  IconPlus, IconClock, IconArrowRight, IconStarFilled,
  IconSparkles, IconPencil, IconBolt, IconBook2, IconDeviceDesktop
} from '@tabler/icons-react'
import { useQuestions, useDashboards, useCollections, useDataSources, useTables } from '../api/hooks'
import { useBookmarkStore, usePinActions } from '../stores/bookmarkStore'
import {
  CardGrid, Section, DataCard, StatCard as UIStatCard, EmptyState, PageContainer
} from '../components/ui'

// =============================================================================
// MAIN COMPONENT
// =============================================================================

export default function Home() {
  const navigate = useNavigate()
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
      <PageContainer>
        <Box mb="xl">
          <Skeleton height={36} width={200} mb="sm" radius="md" />
          <Skeleton height={20} width={300} radius="md" />
        </Box>
        <CardGrid cols={{ base: 1, sm: 2, md: 3, lg: 4 }}>
          {[1, 2, 3, 4].map((i) => (
            <Skeleton key={i} height={140} radius="lg" />
          ))}
        </CardGrid>
      </PageContainer>
    )
  }

  return (
    <PageContainer>
      {/* Header */}
      <Box mb="xl" pb="lg" style={{ borderBottom: '1px solid var(--color-border)' }}>
        <Group justify="space-between" align="flex-start">
          <Group gap="lg" align="center">
            <Box
              style={{
                width: 52,
                height: 52,
                borderRadius: 'var(--radius-xl)',
                background: 'linear-gradient(135deg, var(--color-primary) 0%, var(--color-info) 100%)',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                boxShadow: '0 4px 12px rgba(37, 99, 235, 0.2)',
              }}
            >
              <IconDeviceDesktop size={26} color="white" strokeWidth={1.5} />
            </Box>
            <div>
              <Title
                order={2}
                style={{
                  color: 'var(--color-foreground)',
                  fontWeight: 700,
                  fontSize: rem(28),
                  letterSpacing: '-0.02em',
                }}
              >
                Welcome back
              </Title>
              <Text size="sm" style={{ color: 'var(--color-foreground-muted)' }}>
                Your analytics overview
              </Text>
            </div>
          </Group>
          <Button
            variant="subtle"
            leftSection={<IconPencil size={14} strokeWidth={1.75} />}
            color="gray"
            size="sm"
            radius="md"
          >
            Customize
          </Button>
        </Group>
      </Box>

        {/* X-Ray Sample Cards */}
        {(datasources?.length || 0) > 0 && (
          <Section showHeader={false} mb="xl">
            <Text c="dimmed" mb="lg" style={{ fontSize: rem(15) }}>
              Try out these sample x-rays to see what the app can do.
            </Text>
            <XRayCards datasources={datasources || []} navigate={navigate} />
          </Section>
        )}

        {/* Start Here Section (for new users) */}
        {!hasData && (datasources?.length || 0) === 0 && (
          <Section
            title="Start here"
            icon={<IconSparkles size={16} color="var(--color-primary)" strokeWidth={1.75} />}
            mb="xl"
          >
            <CardGrid cols={{ base: 1, sm: 2, md: 3 }}>
              <StartCard
                icon={IconDatabase}
                color="var(--color-warning)"
                title="Add your data"
                description="Connect to a database to start exploring your data"
                onClick={() => navigate('/admin/datamodel')}
              />
              <StartCard
                icon={IconPencil}
                color="var(--color-primary)"
                title="Ask a question"
                description="Create visualizations and insights from your data"
                onClick={() => navigate('/question/new')}
              />
              <StartCard
                icon={IconLayoutDashboard}
                color="var(--color-success)"
                title="Create a dashboard"
                description="Combine multiple questions into a dashboard"
                onClick={() => navigate('/dashboard/new')}
              />
            </CardGrid>
          </Section>
        )}

        {/* Pinned Items Section */}
        {hasPinnedItems && (
          <Section
            title="Pinned"
            icon={<IconStarFilled size={14} color="var(--color-warning)" />}
            mb="xl"
          >
            <CardGrid cols={{ base: 1, sm: 2, md: 4 }}>
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
            </CardGrid>
          </Section>
        )}

        {/* Pick Up Where You Left Off Section */}
        {hasRecentItems && (
          <Section
            title="Pick up where you left off"
            icon={<IconClock size={14} color="var(--color-foreground-muted)" strokeWidth={1.75} />}
            actions={
              <Button
                variant="subtle"
                size="xs"
                rightSection={<IconArrowRight size={14} />}
                onClick={() => navigate('/browse')}
                color="gray"
              >
                View all
              </Button>
            }
            mb="xl"
          >
            <CardGrid cols={{ base: 1, sm: 2, md: 3, lg: 4 }}>
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
            </CardGrid>
          </Section>
        )}

        {/* Our Analytics Section */}
        {hasData && (
          <Section
            title="Our analytics"
            icon={<IconChartLine size={14} color="var(--color-primary)" strokeWidth={1.75} />}
            actions={
              <Button
                variant="subtle"
                size="xs"
                rightSection={<IconArrowRight size={14} />}
                onClick={() => navigate('/browse')}
                color="gray"
              >
                Browse all
              </Button>
            }
            mb="xl"
          >
            <CardGrid cols={{ base: 2, sm: 4 }}>
              <UIStatCard
                label="Questions"
                value={questions?.length || 0}
                icon={<IconChartLine size={22} strokeWidth={1.75} />}
                iconColor="var(--color-primary)"
                onClick={() => navigate('/browse')}
              />
              <UIStatCard
                label="Dashboards"
                value={dashboards?.length || 0}
                icon={<IconLayoutDashboard size={22} strokeWidth={1.75} />}
                iconColor="var(--color-success)"
                onClick={() => navigate('/browse')}
              />
              <UIStatCard
                label="Collections"
                value={collections?.length || 0}
                icon={<IconFolder size={22} strokeWidth={1.75} />}
                iconColor="var(--color-info)"
                onClick={() => navigate('/browse')}
              />
              <UIStatCard
                label="Databases"
                value={datasources?.length || 0}
                icon={<IconDatabase size={22} strokeWidth={1.75} />}
                iconColor="var(--color-warning)"
                onClick={() => navigate('/admin/datamodel')}
              />
            </CardGrid>
          </Section>
        )}

        {/* Empty State */}
        {!hasData && !isLoading && (datasources?.length || 0) === 0 && (
          <EmptyState
            icon={<IconSparkles size={40} strokeWidth={1.5} />}
            iconColor="var(--color-primary)"
            title="Ready to explore your data?"
            description={
              <>
                Connect to a database to start creating questions and dashboards, or run{' '}
                <code style={{
                  background: 'var(--color-background-subtle)',
                  padding: '2px 6px',
                  borderRadius: 'var(--radius-sm)',
                  fontFamily: 'var(--font-mono)',
                  fontSize: '0.9em',
                }}>
                  bi seed
                </code>{' '}
                to add sample data.
              </>
            }
            action={
              <Button
                size="lg"
                leftSection={<IconDatabase size={20} strokeWidth={1.75} />}
                onClick={() => navigate('/admin/datamodel')}
              >
                Add a database
              </Button>
            }
            secondaryAction={
              <Button
                size="lg"
                variant="light"
                leftSection={<IconPlus size={20} strokeWidth={2} />}
                onClick={() => navigate('/question/new')}
              >
                New question
              </Button>
            }
            size="lg"
          />
        )}
    </PageContainer>
  )
}

// =============================================================================
// SUB-COMPONENTS
// =============================================================================

// Start Card for onboarding (modern shadcn-inspired)
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
  const [isHovered, setIsHovered] = useState(false)

  return (
    <Paper
      p={rem(20)}
      radius="lg"
      onClick={onClick}
      style={{
        cursor: 'pointer',
        display: 'flex',
        alignItems: 'flex-start',
        gap: rem(16),
        transition: 'all 150ms cubic-bezier(0.4, 0, 0.2, 1)',
        border: '1px solid var(--color-border)',
        backgroundColor: 'var(--color-background)',
        boxShadow: isHovered ? 'var(--shadow-md)' : 'var(--shadow-xs)',
        transform: isHovered ? 'translateY(-2px)' : 'none',
      }}
      onMouseEnter={() => setIsHovered(true)}
      onMouseLeave={() => setIsHovered(false)}
    >
      <Box
        style={{
          width: 52,
          height: 52,
          borderRadius: 'var(--radius-xl)',
          backgroundColor: `${color}10`,
          border: `1px solid ${color}18`,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          transition: 'transform 150ms ease',
          transform: isHovered ? 'scale(1.05)' : 'none',
        }}
      >
        <Icon size={26} color={color} strokeWidth={1.75} />
      </Box>
      <Box style={{ flex: 1 }}>
        <Text fw={600} size="md" style={{ color: 'var(--color-foreground)', letterSpacing: '-0.01em' }} mb={rem(4)}>
          {title}
        </Text>
        <Text size="sm" style={{ color: 'var(--color-foreground-muted)', lineHeight: 1.5 }}>
          {description}
        </Text>
      </Box>
    </Paper>
  )
}

// Item Card (Question/Dashboard) - Using new DataCard
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

  const badge = vizType
    ? getVizLabel(vizType)
    : cardCount !== undefined
      ? `${cardCount} ${cardCount === 1 ? 'card' : 'cards'}`
      : type === 'dashboard' ? 'Dashboard' : 'Question'

  return (
    <DataCard
      id={id}
      type={type}
      name={name}
      description={compact ? undefined : description}
      badge={badge}
      colorIndex={colorIndex}
      pinned={pinned}
      onTogglePin={togglePin}
      onClick={onClick}
      compact={compact}
    />
  )
}

// X-Ray Cards Component (modern shadcn-inspired)
function XRayCards({
  datasources,
  navigate,
}: {
  datasources: any[]
  navigate: (path: string) => void
}) {
  const firstDatasource = datasources[0]
  const { data: tables } = useTables(firstDatasource?.id || '')

  const xrayPhrases = [
    { prefix: 'A glance at', suffix: '' },
    { prefix: 'A summary of', suffix: '' },
    { prefix: 'Some insights about', suffix: '' },
    { prefix: 'A look at', suffix: '' },
  ]

  const xrayCards = useMemo(() => {
    if (!tables || tables.length === 0) return []
    return tables.slice(0, 8).map((table, index) => ({
      table,
      phrase: xrayPhrases[index % xrayPhrases.length],
    }))
  }, [tables])

  if (xrayCards.length === 0) return null

  return (
    <CardGrid cols={{ base: 1, sm: 2, md: 3, lg: 4 }}>
      {xrayCards.map(({ table, phrase }) => (
        <XRayCard
          key={table.id}
          icon={<IconBolt size={18} color="var(--color-warning)" strokeWidth={2} />}
          onClick={() => navigate(`/xray/${firstDatasource.id}/table/${table.id}`)}
        >
          <Text size="sm" style={{ color: 'var(--color-foreground)' }}>
            {phrase.prefix}{' '}
            <Text span fw={600} inherit>
              {table.display_name || table.name}
            </Text>
            {phrase.suffix}
          </Text>
        </XRayCard>
      ))}
      {/* Tips Card */}
      <XRayCard
        icon={<IconBook2 size={18} color="var(--color-foreground-muted)" strokeWidth={1.75} />}
        onClick={() => window.open('https://www.metabase.com/docs', '_blank')}
      >
        <Text size="sm" fw={500} style={{ color: 'var(--color-foreground)' }}>
          Documentation & tips
        </Text>
      </XRayCard>
    </CardGrid>
  )
}

// Individual X-Ray card component with hover state
function XRayCard({
  icon,
  children,
  onClick,
}: {
  icon: React.ReactNode
  children: React.ReactNode
  onClick: () => void
}) {
  const [isHovered, setIsHovered] = useState(false)

  return (
    <Paper
      p={rem(16)}
      radius="lg"
      onClick={onClick}
      style={{
        cursor: 'pointer',
        display: 'flex',
        alignItems: 'center',
        gap: rem(12),
        transition: 'all 150ms cubic-bezier(0.4, 0, 0.2, 1)',
        border: '1px solid var(--color-border)',
        backgroundColor: 'var(--color-background)',
        boxShadow: isHovered ? 'var(--shadow-md)' : 'var(--shadow-xs)',
        transform: isHovered ? 'translateY(-1px)' : 'none',
      }}
      onMouseEnter={() => setIsHovered(true)}
      onMouseLeave={() => setIsHovered(false)}
    >
      <Box
        style={{
          width: 36,
          height: 36,
          borderRadius: 'var(--radius-lg)',
          backgroundColor: 'var(--color-background-subtle)',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          flexShrink: 0,
        }}
      >
        {icon}
      </Box>
      {children}
    </Paper>
  )
}
