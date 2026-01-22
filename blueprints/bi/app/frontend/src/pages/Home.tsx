import { useMemo } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  Container, Title, Text, Card, Group, Stack, Button, SimpleGrid,
  Badge, Paper, Skeleton, Avatar, ThemeIcon, ActionIcon, Menu, Tooltip
} from '@mantine/core'
import {
  IconChartBar, IconLayoutDashboard, IconFolder, IconDatabase,
  IconPlus, IconDots, IconStar, IconStarFilled, IconClock,
  IconArrowRight, IconBolt, IconSearch
} from '@tabler/icons-react'
import { useQuestions, useDashboards, useCollections, useDataSources } from '../api/hooks'
import { useUIStore } from '../stores/uiStore'
import { chartColors } from '../theme'

export default function Home() {
  const navigate = useNavigate()
  const { openCommandPalette } = useUIStore()

  const { data: questions, isLoading: questionsLoading } = useQuestions()
  const { data: dashboards, isLoading: dashboardsLoading } = useDashboards()
  const { data: collections, isLoading: collectionsLoading } = useCollections()
  const { data: datasources, isLoading: datasourcesLoading } = useDataSources()

  const isLoading = questionsLoading || dashboardsLoading || collectionsLoading || datasourcesLoading

  const stats = useMemo(() => ({
    questions: questions?.length || 0,
    dashboards: dashboards?.length || 0,
    collections: collections?.length || 0,
    datasources: datasources?.length || 0,
  }), [questions, dashboards, collections, datasources])

  const recentDashboards = useMemo(() => {
    return (dashboards || [])
      .sort((a, b) => new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime())
      .slice(0, 4)
  }, [dashboards])

  const recentQuestions = useMemo(() => {
    return (questions || [])
      .sort((a, b) => new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime())
      .slice(0, 6)
  }, [questions])

  const statCards = [
    { label: 'Questions', value: stats.questions, icon: IconChartBar, color: 'brand', path: '/browse' },
    { label: 'Dashboards', value: stats.dashboards, icon: IconLayoutDashboard, color: 'summarize', path: '/browse' },
    { label: 'Collections', value: stats.collections, icon: IconFolder, color: 'filter', path: '/browse' },
    { label: 'Data Sources', value: stats.datasources, icon: IconDatabase, color: 'warning', path: '/admin/datamodel' },
  ]

  const quickActions = [
    { label: 'New question', icon: IconChartBar, color: 'brand', action: () => navigate('/question/new') },
    { label: 'New dashboard', icon: IconLayoutDashboard, color: 'summarize', action: () => navigate('/dashboard/new') },
    { label: 'Browse data', icon: IconDatabase, color: 'filter', action: () => navigate('/browse/databases') },
  ]

  const getVizIcon = (type: string) => {
    switch (type) {
      case 'line': return '📈'
      case 'bar': return '📊'
      case 'pie': return '🥧'
      case 'number': return '#'
      default: return '📋'
    }
  }

  if (isLoading && stats.questions === 0 && stats.dashboards === 0) {
    return (
      <Container size="xl" py="lg">
        <Group justify="space-between" mb="xl">
          <div>
            <Skeleton height={32} width={200} mb="sm" />
            <Skeleton height={20} width={300} />
          </div>
        </Group>
        <SimpleGrid cols={{ base: 1, sm: 2, md: 4 }} mb="xl">
          {[1, 2, 3, 4].map((i) => (
            <Skeleton key={i} height={100} radius="md" />
          ))}
        </SimpleGrid>
      </Container>
    )
  }

  return (
    <Container size="xl" py="lg">
      {/* Header */}
      <Group justify="space-between" mb="xl">
        <div>
          <Title order={2}>Home</Title>
          <Text c="dimmed">Your analytics at a glance</Text>
        </div>
        <Group>
          <Button
            variant="light"
            leftSection={<IconSearch size={16} />}
            onClick={openCommandPalette}
            styles={{ root: { fontWeight: 400 } }}
          >
            Search or jump to...
          </Button>
          <Button leftSection={<IconPlus size={16} />} onClick={() => navigate('/question/new')}>
            New Question
          </Button>
        </Group>
      </Group>

      {/* Quick Actions */}
      <Card withBorder radius="md" p="lg" mb="xl" bg="gray.0">
        <Group gap="lg">
          <Text fw={500} c="dimmed">Quick actions</Text>
          {quickActions.map((action) => (
            <Button
              key={action.label}
              variant="white"
              leftSection={<action.icon size={18} />}
              onClick={action.action}
              styles={{
                root: {
                  boxShadow: '0 1px 3px rgba(0,0,0,0.1)',
                }
              }}
            >
              {action.label}
            </Button>
          ))}
        </Group>
      </Card>

      {/* Stats */}
      <SimpleGrid cols={{ base: 1, sm: 2, md: 4 }} mb="xl">
        {statCards.map((stat) => (
          <Card
            key={stat.label}
            withBorder
            radius="md"
            padding="lg"
            style={{ cursor: 'pointer' }}
            onClick={() => navigate(stat.path)}
          >
            <Group justify="space-between" align="flex-start">
              <div>
                <Text size="2rem" fw={700} c={`${stat.color}.5`}>
                  {stat.value}
                </Text>
                <Text c="dimmed" size="sm" mt={4}>
                  {stat.label}
                </Text>
              </div>
              <ThemeIcon
                size={48}
                radius="md"
                variant="light"
                color={stat.color}
              >
                <stat.icon size={24} />
              </ThemeIcon>
            </Group>
          </Card>
        ))}
      </SimpleGrid>

      {/* Recent Dashboards */}
      {recentDashboards.length > 0 && (
        <>
          <Group justify="space-between" mb="md">
            <Group gap="xs">
              <IconClock size={20} color="var(--mantine-color-gray-6)" />
              <Title order={4}>Recent Dashboards</Title>
            </Group>
            <Button
              variant="subtle"
              size="sm"
              rightSection={<IconArrowRight size={14} />}
              onClick={() => navigate('/browse')}
            >
              View all
            </Button>
          </Group>
          <SimpleGrid cols={{ base: 1, sm: 2, md: 4 }} mb="xl">
            {recentDashboards.map((dashboard, index) => (
              <Card
                key={dashboard.id}
                withBorder
                radius="md"
                padding="lg"
                style={{ cursor: 'pointer' }}
                onClick={() => navigate(`/dashboard/${dashboard.id}`)}
              >
                <Group justify="space-between" mb="sm">
                  <ThemeIcon
                    size={40}
                    radius="md"
                    variant="light"
                    color="summarize"
                    style={{
                      backgroundColor: chartColors[index % chartColors.length] + '20',
                    }}
                  >
                    <IconLayoutDashboard size={20} color={chartColors[index % chartColors.length]} />
                  </ThemeIcon>
                  <Menu position="bottom-end" withinPortal>
                    <Menu.Target>
                      <ActionIcon variant="subtle" color="gray" onClick={(e) => e.stopPropagation()}>
                        <IconDots size={16} />
                      </ActionIcon>
                    </Menu.Target>
                    <Menu.Dropdown>
                      <Menu.Item leftSection={<IconStar size={14} />}>
                        Pin to Home
                      </Menu.Item>
                    </Menu.Dropdown>
                  </Menu>
                </Group>
                <Text fw={500} lineClamp={1}>{dashboard.name}</Text>
                {dashboard.description && (
                  <Text size="sm" c="dimmed" lineClamp={1} mt={4}>
                    {dashboard.description}
                  </Text>
                )}
                <Text size="xs" c="dimmed" mt="sm">
                  {dashboard.cards?.length || 0} cards
                </Text>
              </Card>
            ))}
          </SimpleGrid>
        </>
      )}

      {/* Recent Questions */}
      {recentQuestions.length > 0 && (
        <>
          <Group justify="space-between" mb="md">
            <Group gap="xs">
              <IconClock size={20} color="var(--mantine-color-gray-6)" />
              <Title order={4}>Recent Questions</Title>
            </Group>
            <Button
              variant="subtle"
              size="sm"
              rightSection={<IconArrowRight size={14} />}
              onClick={() => navigate('/browse')}
            >
              View all
            </Button>
          </Group>
          <SimpleGrid cols={{ base: 1, sm: 2, md: 3 }} mb="xl">
            {recentQuestions.map((question, index) => (
              <Card
                key={question.id}
                withBorder
                radius="md"
                padding="lg"
                style={{ cursor: 'pointer' }}
                onClick={() => navigate(`/question/${question.id}`)}
              >
                <Group justify="space-between" mb="sm">
                  <Group gap="sm">
                    <Avatar
                      size="sm"
                      radius="sm"
                      color="brand"
                      style={{
                        backgroundColor: chartColors[index % chartColors.length] + '20',
                        color: chartColors[index % chartColors.length],
                      }}
                    >
                      {getVizIcon(question.visualization?.type || 'table')}
                    </Avatar>
                    <Badge size="sm" variant="light" color="brand">
                      {question.visualization?.type || 'table'}
                    </Badge>
                  </Group>
                  <Menu position="bottom-end" withinPortal>
                    <Menu.Target>
                      <ActionIcon variant="subtle" color="gray" onClick={(e) => e.stopPropagation()}>
                        <IconDots size={16} />
                      </ActionIcon>
                    </Menu.Target>
                    <Menu.Dropdown>
                      <Menu.Item leftSection={<IconStar size={14} />}>
                        Pin to Home
                      </Menu.Item>
                    </Menu.Dropdown>
                  </Menu>
                </Group>
                <Text fw={500} lineClamp={2}>{question.name}</Text>
                {question.description && (
                  <Text size="sm" c="dimmed" lineClamp={1} mt={4}>
                    {question.description}
                  </Text>
                )}
              </Card>
            ))}
          </SimpleGrid>
        </>
      )}

      {/* Empty state */}
      {!isLoading && stats.questions === 0 && stats.dashboards === 0 && (
        <Paper withBorder radius="md" p="xl" ta="center">
          <Stack align="center" gap="lg">
            <ThemeIcon size={80} radius="xl" variant="light" color="brand">
              <IconBolt size={40} />
            </ThemeIcon>
            <div>
              <Title order={3} mb="xs">Get started with BI</Title>
              <Text c="dimmed" maw={400} mx="auto">
                Connect to a data source, or run 'bi seed' to add sample data and start exploring.
              </Text>
            </div>
            <Group>
              <Button
                size="lg"
                leftSection={<IconDatabase size={20} />}
                onClick={() => navigate('/admin/datamodel')}
              >
                Add Data Source
              </Button>
              <Button
                size="lg"
                variant="light"
                leftSection={<IconPlus size={20} />}
                onClick={() => navigate('/question/new')}
              >
                New Question
              </Button>
            </Group>
          </Stack>
        </Paper>
      )}
    </Container>
  )
}
