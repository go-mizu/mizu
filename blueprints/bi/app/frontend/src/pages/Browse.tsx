import { useMemo, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import {
  Container, Title, Text, Card, Group, Stack, SimpleGrid, Badge, TextInput,
  Tabs, Paper, Menu, ActionIcon, Button, Breadcrumbs, Anchor, ThemeIcon,
  Modal, Table, SegmentedControl
} from '@mantine/core'
import { useDisclosure } from '@mantine/hooks'
import { notifications } from '@mantine/notifications'
import {
  IconChartBar, IconLayoutDashboard, IconFolder, IconSearch, IconPlus,
  IconDots, IconPencil, IconTrash, IconStar, IconDatabase,
  IconFileAnalytics, IconChartLine, IconArrowRight, IconFolderPlus,
  IconGridDots, IconList
} from '@tabler/icons-react'
import {
  useCollections, useCollection, useCollectionItems, useCreateCollection,
  useDashboards, useQuestions, useDataSources,
  useModels, useMetrics
} from '../api/hooks'
import { chartColors } from '../theme'

interface BrowseProps {
  view?: 'all' | 'databases' | 'models' | 'metrics'
}

export default function Browse({ view = 'all' }: BrowseProps) {
  const navigate = useNavigate()
  const { id: collectionId } = useParams()

  // State
  const [search, setSearch] = useState('')
  const [viewMode, setViewMode] = useState<'grid' | 'list'>('grid')
  const [activeTab, setActiveTab] = useState<string | null>(view === 'all' ? 'all' : view)
  const [newCollectionModalOpened, { open: openNewCollectionModal, close: closeNewCollectionModal }] = useDisclosure(false)
  const [newCollectionName, setNewCollectionName] = useState('')

  // Queries
  const { data: collections } = useCollections()
  const { data: currentCollection } = useCollection(collectionId || '')
  const { data: collectionItems } = useCollectionItems(collectionId || '')
  const { data: dashboards } = useDashboards()
  const { data: questions } = useQuestions()
  useDataSources() // Pre-fetch datasources
  useModels() // Pre-fetch models
  useMetrics() // Pre-fetch metrics
  const createCollection = useCreateCollection()

  // Build breadcrumbs for collection navigation
  const breadcrumbs = useMemo(() => {
    const crumbs: { label: string; path: string }[] = [
      { label: 'Browse', path: '/browse' },
    ]
    if (currentCollection) {
      crumbs.push({
        label: currentCollection.name,
        path: `/browse/${currentCollection.id}`,
      })
    }
    return crumbs
  }, [currentCollection])

  // Filter items based on search
  const filteredItems = useMemo(() => {
    const searchLower = search.toLowerCase()

    // If viewing a collection, use collection items
    if (collectionId && collectionItems) {
      return {
        collections: collectionItems.subcollections?.filter(c =>
          c.name.toLowerCase().includes(searchLower)
        ) || [],
        dashboards: collectionItems.dashboards?.filter(d =>
          d.name.toLowerCase().includes(searchLower)
        ) || [],
        questions: collectionItems.questions?.filter(q =>
          q.name.toLowerCase().includes(searchLower)
        ) || [],
      }
    }

    // Otherwise, show all items (not in any collection for root)
    return {
      collections: (collections || []).filter(c =>
        !c.parent_id && c.name.toLowerCase().includes(searchLower)
      ),
      dashboards: (dashboards || []).filter(d =>
        !d.collection_id && d.name.toLowerCase().includes(searchLower)
      ),
      questions: (questions || []).filter(q =>
        !q.collection_id && q.name.toLowerCase().includes(searchLower)
      ),
    }
  }, [collectionId, collectionItems, collections, dashboards, questions, search])

  // Handle create collection
  const handleCreateCollection = async () => {
    if (!newCollectionName.trim()) return

    try {
      const newCollection = await createCollection.mutateAsync({
        name: newCollectionName,
        parent_id: collectionId || undefined,
      })
      notifications.show({
        title: 'Collection created',
        message: `${newCollectionName} has been created`,
        color: 'green',
      })
      closeNewCollectionModal()
      setNewCollectionName('')
      navigate(`/browse/${newCollection.id}`)
    } catch (err: any) {
      notifications.show({
        title: 'Error',
        message: err.message || 'Failed to create collection',
        color: 'red',
      })
    }
  }

  // Render based on view type
  if (view === 'databases') {
    return <DatabasesView search={search} setSearch={setSearch} />
  }

  if (view === 'models') {
    return <ModelsView search={search} setSearch={setSearch} />
  }

  if (view === 'metrics') {
    return <MetricsView search={search} setSearch={setSearch} />
  }

  const totalItems =
    filteredItems.collections.length +
    filteredItems.dashboards.length +
    filteredItems.questions.length

  return (
    <Container size="xl" py="lg">
      {/* Header */}
      <Group justify="space-between" mb="xl">
        <div>
          {collectionId ? (
            <>
              <Breadcrumbs mb="xs">
                {breadcrumbs.map((crumb, i) => (
                  <Anchor
                    key={crumb.path}
                    onClick={() => navigate(crumb.path)}
                    c={i === breadcrumbs.length - 1 ? 'dark' : 'dimmed'}
                    fw={i === breadcrumbs.length - 1 ? 600 : 400}
                  >
                    {crumb.label}
                  </Anchor>
                ))}
              </Breadcrumbs>
              <Title order={2}>{currentCollection?.name || 'Collection'}</Title>
            </>
          ) : (
            <>
              <Title order={2}>Browse</Title>
              <Text c="dimmed">Explore your data</Text>
            </>
          )}
        </div>
        <Group gap="sm">
          <TextInput
            placeholder="Search..."
            leftSection={<IconSearch size={16} />}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            w={250}
          />
          <SegmentedControl
            value={viewMode}
            onChange={(v) => setViewMode(v as 'grid' | 'list')}
            data={[
              { value: 'grid', label: <IconGridDots size={16} /> },
              { value: 'list', label: <IconList size={16} /> },
            ]}
            size="sm"
          />
          <Menu position="bottom-end">
            <Menu.Target>
              <Button leftSection={<IconPlus size={16} />}>New</Button>
            </Menu.Target>
            <Menu.Dropdown>
              <Menu.Item
                leftSection={<IconChartBar size={14} />}
                onClick={() => navigate('/question/new')}
              >
                New question
              </Menu.Item>
              <Menu.Item
                leftSection={<IconLayoutDashboard size={14} />}
                onClick={() => navigate('/dashboard/new')}
              >
                New dashboard
              </Menu.Item>
              <Menu.Divider />
              <Menu.Item
                leftSection={<IconFolderPlus size={14} />}
                onClick={openNewCollectionModal}
              >
                New collection
              </Menu.Item>
            </Menu.Dropdown>
          </Menu>
        </Group>
      </Group>

      {/* Tabs */}
      <Tabs value={activeTab} onChange={setActiveTab} mb="lg">
        <Tabs.List>
          <Tabs.Tab value="all">
            All ({totalItems})
          </Tabs.Tab>
          <Tabs.Tab value="collections" leftSection={<IconFolder size={14} />}>
            Collections ({filteredItems.collections.length})
          </Tabs.Tab>
          <Tabs.Tab value="dashboards" leftSection={<IconLayoutDashboard size={14} />}>
            Dashboards ({filteredItems.dashboards.length})
          </Tabs.Tab>
          <Tabs.Tab value="questions" leftSection={<IconChartBar size={14} />}>
            Questions ({filteredItems.questions.length})
          </Tabs.Tab>
        </Tabs.List>
      </Tabs>

      {/* Content */}
      {activeTab === 'all' || activeTab === 'collections' ? (
        filteredItems.collections.length > 0 && (
          <div style={{ marginBottom: 32 }}>
            {activeTab === 'all' && (
              <Group justify="space-between" mb="md">
                <Title order={4}>Collections</Title>
                <Button
                  variant="subtle"
                  size="sm"
                  rightSection={<IconArrowRight size={14} />}
                  onClick={() => setActiveTab('collections')}
                >
                  View all
                </Button>
              </Group>
            )}
            {viewMode === 'grid' ? (
              <SimpleGrid cols={{ base: 1, sm: 2, md: 4 }}>
                {filteredItems.collections.map((collection, i) => (
                  <CollectionCard
                    key={collection.id}
                    collection={collection}
                    colorIndex={i}
                    onClick={() => navigate(`/browse/${collection.id}`)}
                  />
                ))}
              </SimpleGrid>
            ) : (
              <ItemList
                items={filteredItems.collections}
                type="collection"
                onClick={(id) => navigate(`/browse/${id}`)}
              />
            )}
          </div>
        )
      ) : null}

      {activeTab === 'all' || activeTab === 'dashboards' ? (
        filteredItems.dashboards.length > 0 && (
          <div style={{ marginBottom: 32 }}>
            {activeTab === 'all' && (
              <Group justify="space-between" mb="md">
                <Title order={4}>Dashboards</Title>
                <Button
                  variant="subtle"
                  size="sm"
                  rightSection={<IconArrowRight size={14} />}
                  onClick={() => setActiveTab('dashboards')}
                >
                  View all
                </Button>
              </Group>
            )}
            {viewMode === 'grid' ? (
              <SimpleGrid cols={{ base: 1, sm: 2, md: 3 }}>
                {filteredItems.dashboards.map((dashboard) => (
                  <DashboardCard
                    key={dashboard.id}
                    dashboard={dashboard}
                    onClick={() => navigate(`/dashboard/${dashboard.id}`)}
                  />
                ))}
              </SimpleGrid>
            ) : (
              <ItemList
                items={filteredItems.dashboards}
                type="dashboard"
                onClick={(id) => navigate(`/dashboard/${id}`)}
              />
            )}
          </div>
        )
      ) : null}

      {activeTab === 'all' || activeTab === 'questions' ? (
        filteredItems.questions.length > 0 && (
          <div style={{ marginBottom: 32 }}>
            {activeTab === 'all' && (
              <Group justify="space-between" mb="md">
                <Title order={4}>Questions</Title>
                <Button
                  variant="subtle"
                  size="sm"
                  rightSection={<IconArrowRight size={14} />}
                  onClick={() => setActiveTab('questions')}
                >
                  View all
                </Button>
              </Group>
            )}
            {viewMode === 'grid' ? (
              <SimpleGrid cols={{ base: 1, sm: 2, md: 3 }}>
                {filteredItems.questions.map((question) => (
                  <QuestionCard
                    key={question.id}
                    question={question}
                    onClick={() => navigate(`/question/${question.id}`)}
                  />
                ))}
              </SimpleGrid>
            ) : (
              <ItemList
                items={filteredItems.questions}
                type="question"
                onClick={(id) => navigate(`/question/${id}`)}
              />
            )}
          </div>
        )
      ) : null}

      {/* Empty state */}
      {totalItems === 0 && (
        <Paper withBorder radius="md" p="xl" ta="center">
          <Stack align="center" gap="lg">
            <ThemeIcon size={60} radius="xl" variant="light" color="brand">
              <IconFolder size={30} />
            </ThemeIcon>
            <div>
              <Title order={3} mb="xs">
                {search ? 'No results found' : 'This collection is empty'}
              </Title>
              <Text c="dimmed" maw={400} mx="auto">
                {search
                  ? 'Try adjusting your search terms'
                  : 'Create a new question or dashboard to get started'}
              </Text>
            </div>
            {!search && (
              <Group>
                <Button
                  leftSection={<IconChartBar size={16} />}
                  onClick={() => navigate('/question/new')}
                >
                  New Question
                </Button>
                <Button
                  variant="light"
                  leftSection={<IconLayoutDashboard size={16} />}
                  onClick={() => navigate('/dashboard/new')}
                >
                  New Dashboard
                </Button>
              </Group>
            )}
          </Stack>
        </Paper>
      )}

      {/* New Collection Modal */}
      <Modal
        opened={newCollectionModalOpened}
        onClose={closeNewCollectionModal}
        title="New Collection"
      >
        <Stack gap="md">
          <TextInput
            label="Collection name"
            placeholder="My Collection"
            value={newCollectionName}
            onChange={(e) => setNewCollectionName(e.target.value)}
            required
          />
          <Group justify="flex-end" mt="md">
            <Button variant="light" onClick={closeNewCollectionModal}>Cancel</Button>
            <Button onClick={handleCreateCollection} loading={createCollection.isPending}>
              Create
            </Button>
          </Group>
        </Stack>
      </Modal>
    </Container>
  )
}

// Collection Card Component
function CollectionCard({
  collection,
  colorIndex,
  onClick,
}: {
  collection: { id: string; name: string; color?: string }
  colorIndex: number
  onClick: () => void
}) {
  const color = collection.color || chartColors[colorIndex % chartColors.length]

  return (
    <Card
      withBorder
      radius="md"
      padding="lg"
      style={{ cursor: 'pointer', borderLeft: `4px solid ${color}` }}
      onClick={onClick}
    >
      <Group>
        <ThemeIcon size={40} radius="md" variant="light" style={{ backgroundColor: color + '20', color }}>
          <IconFolder size={20} />
        </ThemeIcon>
        <div style={{ flex: 1 }}>
          <Text fw={500}>{collection.name}</Text>
        </div>
      </Group>
    </Card>
  )
}

// Dashboard Card Component
function DashboardCard({
  dashboard,
  onClick,
}: {
  dashboard: { id: string; name: string; description?: string; cards?: any[] }
  onClick: () => void
}) {
  return (
    <Card
      withBorder
      radius="md"
      padding="lg"
      style={{ cursor: 'pointer' }}
      onClick={onClick}
    >
      <Group mb="sm">
        <ThemeIcon size={40} radius="md" variant="light" color="summarize">
          <IconLayoutDashboard size={20} />
        </ThemeIcon>
        <div style={{ flex: 1 }}>
          <Text fw={500}>{dashboard.name}</Text>
          {dashboard.description && (
            <Text size="sm" c="dimmed" lineClamp={1}>
              {dashboard.description}
            </Text>
          )}
        </div>
      </Group>
      <Badge size="sm" variant="light" color="gray">
        {dashboard.cards?.length || 0} cards
      </Badge>
    </Card>
  )
}

// Question Card Component
function QuestionCard({
  question,
  onClick,
}: {
  question: { id: string; name: string; description?: string; visualization?: { type: string } }
  onClick: () => void
}) {
  return (
    <Card
      withBorder
      radius="md"
      padding="lg"
      style={{ cursor: 'pointer' }}
      onClick={onClick}
    >
      <Group mb="sm">
        <ThemeIcon size={40} radius="md" variant="light" color="brand">
          <IconChartBar size={20} />
        </ThemeIcon>
        <div style={{ flex: 1 }}>
          <Text fw={500}>{question.name}</Text>
          {question.description && (
            <Text size="sm" c="dimmed" lineClamp={1}>
              {question.description}
            </Text>
          )}
        </div>
      </Group>
      <Badge size="sm" variant="light" color="brand">
        {question.visualization?.type || 'table'}
      </Badge>
    </Card>
  )
}

// List View Component
function ItemList({
  items,
  type,
  onClick,
}: {
  items: any[]
  type: 'collection' | 'dashboard' | 'question'
  onClick: (id: string) => void
}) {
  const Icon = type === 'collection' ? IconFolder : type === 'dashboard' ? IconLayoutDashboard : IconChartBar
  const color = type === 'collection' ? 'filter' : type === 'dashboard' ? 'summarize' : 'brand'

  return (
    <Table striped highlightOnHover>
      <Table.Thead>
        <Table.Tr>
          <Table.Th>Name</Table.Th>
          <Table.Th>Type</Table.Th>
          <Table.Th w={100}></Table.Th>
        </Table.Tr>
      </Table.Thead>
      <Table.Tbody>
        {items.map((item) => (
          <Table.Tr key={item.id} style={{ cursor: 'pointer' }} onClick={() => onClick(item.id)}>
            <Table.Td>
              <Group gap="sm">
                <ThemeIcon size="sm" variant="light" color={color}>
                  <Icon size={14} />
                </ThemeIcon>
                <Text fw={500}>{item.name}</Text>
              </Group>
            </Table.Td>
            <Table.Td>
              <Badge size="sm" variant="light" color={color}>
                {type}
              </Badge>
            </Table.Td>
            <Table.Td>
              <Menu position="bottom-end">
                <Menu.Target>
                  <ActionIcon variant="subtle" onClick={(e) => e.stopPropagation()}>
                    <IconDots size={16} />
                  </ActionIcon>
                </Menu.Target>
                <Menu.Dropdown>
                  <Menu.Item leftSection={<IconStar size={14} />}>Pin</Menu.Item>
                  <Menu.Item leftSection={<IconPencil size={14} />}>Edit</Menu.Item>
                  <Menu.Divider />
                  <Menu.Item leftSection={<IconTrash size={14} />} color="red">Delete</Menu.Item>
                </Menu.Dropdown>
              </Menu>
            </Table.Td>
          </Table.Tr>
        ))}
      </Table.Tbody>
    </Table>
  )
}

// Databases View Component
function DatabasesView({
  search,
  setSearch,
}: {
  search: string
  setSearch: (value: string) => void
}) {
  const navigate = useNavigate()
  const { data: datasources } = useDataSources()

  const filteredDatasources = useMemo(() => {
    if (!search.trim()) return datasources || []
    const searchLower = search.toLowerCase()
    return (datasources || []).filter(ds =>
      ds.name.toLowerCase().includes(searchLower) ||
      ds.database.toLowerCase().includes(searchLower)
    )
  }, [datasources, search])

  return (
    <Container size="xl" py="lg">
      <Group justify="space-between" mb="xl">
        <div>
          <Title order={2}>Databases</Title>
          <Text c="dimmed">Explore your connected data sources</Text>
        </div>
        <Group gap="sm">
          <TextInput
            placeholder="Search databases..."
            leftSection={<IconSearch size={16} />}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            w={250}
          />
          <Button leftSection={<IconPlus size={16} />} onClick={() => navigate('/admin/datamodel')}>
            Add Database
          </Button>
        </Group>
      </Group>

      {filteredDatasources.length > 0 ? (
        <SimpleGrid cols={{ base: 1, sm: 2, md: 3 }}>
          {filteredDatasources.map((ds) => (
            <Card
              key={ds.id}
              withBorder
              radius="md"
              padding="lg"
              style={{ cursor: 'pointer' }}
              onClick={() => navigate(`/admin/datamodel/${ds.id}`)}
            >
              <Group mb="md">
                <ThemeIcon size={48} radius="md" variant="light" color="warning">
                  <IconDatabase size={24} />
                </ThemeIcon>
                <div style={{ flex: 1 }}>
                  <Text fw={500}>{ds.name}</Text>
                  <Text size="sm" c="dimmed">{ds.engine}</Text>
                </div>
              </Group>
              <Badge size="sm" variant="light" color="gray">
                {ds.database}
              </Badge>
            </Card>
          ))}
        </SimpleGrid>
      ) : (
        <Paper withBorder radius="md" p="xl" ta="center">
          <Stack align="center" gap="md">
            <ThemeIcon size={60} radius="xl" variant="light" color="warning">
              <IconDatabase size={30} />
            </ThemeIcon>
            <Title order={3}>No databases connected</Title>
            <Text c="dimmed">Connect a database to start exploring your data</Text>
            <Button leftSection={<IconPlus size={16} />} onClick={() => navigate('/admin/datamodel')}>
              Add Database
            </Button>
          </Stack>
        </Paper>
      )}
    </Container>
  )
}

// Models View Component
function ModelsView({
  search,
  setSearch,
}: {
  search: string
  setSearch: (value: string) => void
}) {
  const { data: models } = useModels()

  const filteredModels = useMemo(() => {
    if (!search.trim()) return models || []
    const searchLower = search.toLowerCase()
    return (models || []).filter(m =>
      m.name.toLowerCase().includes(searchLower)
    )
  }, [models, search])

  return (
    <Container size="xl" py="lg">
      <Group justify="space-between" mb="xl">
        <div>
          <Title order={2}>Models</Title>
          <Text c="dimmed">Curated views of your data</Text>
        </div>
        <Group gap="sm">
          <TextInput
            placeholder="Search models..."
            leftSection={<IconSearch size={16} />}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            w={250}
          />
          <Button leftSection={<IconPlus size={16} />}>
            New Model
          </Button>
        </Group>
      </Group>

      {filteredModels.length > 0 ? (
        <SimpleGrid cols={{ base: 1, sm: 2, md: 3 }}>
          {filteredModels.map((model) => (
            <Card
              key={model.id}
              withBorder
              radius="md"
              padding="lg"
              style={{ cursor: 'pointer' }}
            >
              <Group mb="md">
                <ThemeIcon size={48} radius="md" variant="light" color="filter">
                  <IconFileAnalytics size={24} />
                </ThemeIcon>
                <div style={{ flex: 1 }}>
                  <Text fw={500}>{model.name}</Text>
                  {model.description && (
                    <Text size="sm" c="dimmed" lineClamp={1}>{model.description}</Text>
                  )}
                </div>
              </Group>
            </Card>
          ))}
        </SimpleGrid>
      ) : (
        <Paper withBorder radius="md" p="xl" ta="center">
          <Stack align="center" gap="md">
            <ThemeIcon size={60} radius="xl" variant="light" color="filter">
              <IconFileAnalytics size={30} />
            </ThemeIcon>
            <Title order={3}>No models yet</Title>
            <Text c="dimmed">Models help you define curated views of your data</Text>
            <Button leftSection={<IconPlus size={16} />}>
              Create Model
            </Button>
          </Stack>
        </Paper>
      )}
    </Container>
  )
}

// Metrics View Component
function MetricsView({
  search,
  setSearch,
}: {
  search: string
  setSearch: (value: string) => void
}) {
  const { data: metrics } = useMetrics()

  const filteredMetrics = useMemo(() => {
    if (!search.trim()) return metrics || []
    const searchLower = search.toLowerCase()
    return (metrics || []).filter(m =>
      m.name.toLowerCase().includes(searchLower)
    )
  }, [metrics, search])

  return (
    <Container size="xl" py="lg">
      <Group justify="space-between" mb="xl">
        <div>
          <Title order={2}>Metrics</Title>
          <Text c="dimmed">Key business metrics</Text>
        </div>
        <Group gap="sm">
          <TextInput
            placeholder="Search metrics..."
            leftSection={<IconSearch size={16} />}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            w={250}
          />
          <Button leftSection={<IconPlus size={16} />}>
            New Metric
          </Button>
        </Group>
      </Group>

      {filteredMetrics.length > 0 ? (
        <SimpleGrid cols={{ base: 1, sm: 2, md: 3 }}>
          {filteredMetrics.map((metric) => (
            <Card
              key={metric.id}
              withBorder
              radius="md"
              padding="lg"
              style={{ cursor: 'pointer' }}
            >
              <Group mb="md">
                <ThemeIcon size={48} radius="md" variant="light" color="brand">
                  <IconChartLine size={24} />
                </ThemeIcon>
                <div style={{ flex: 1 }}>
                  <Text fw={500}>{metric.name}</Text>
                  {metric.description && (
                    <Text size="sm" c="dimmed" lineClamp={1}>{metric.description}</Text>
                  )}
                </div>
              </Group>
              <Badge size="sm" variant="light" color="brand">
                {metric.definition?.aggregation || 'count'}
              </Badge>
            </Card>
          ))}
        </SimpleGrid>
      ) : (
        <Paper withBorder radius="md" p="xl" ta="center">
          <Stack align="center" gap="md">
            <ThemeIcon size={60} radius="xl" variant="light" color="brand">
              <IconChartLine size={30} />
            </ThemeIcon>
            <Title order={3}>No metrics defined</Title>
            <Text c="dimmed">Define metrics to track key business numbers</Text>
            <Button leftSection={<IconPlus size={16} />}>
              Create Metric
            </Button>
          </Stack>
        </Paper>
      )}
    </Container>
  )
}
