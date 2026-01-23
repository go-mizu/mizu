import { useMemo, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import {
  Group, TextInput, Tabs, Menu, ActionIcon, Button, Breadcrumbs, Anchor,
  Modal, Table, SegmentedControl, Stack
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
import {
  PageHeader, SectionHeader, DataCard, EmptyState, CardGrid, PageContainer
} from '../components/ui'

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
    <PageContainer>
      {/* Header */}
      {collectionId ? (
        <div style={{ marginBottom: 'var(--mantine-spacing-xl)' }}>
          <Breadcrumbs mb="xs">
            {breadcrumbs.map((crumb, i) => (
              <Anchor
                key={crumb.path}
                onClick={() => navigate(crumb.path)}
                style={{
                  color: i === breadcrumbs.length - 1 ? 'var(--color-foreground)' : 'var(--color-foreground-muted)',
                  fontWeight: i === breadcrumbs.length - 1 ? 600 : 400,
                }}
              >
                {crumb.label}
              </Anchor>
            ))}
          </Breadcrumbs>
          <PageHeader
            title={currentCollection?.name || 'Collection'}
            actions={
              <Group gap="sm">
                <TextInput
                  placeholder="Search..."
                  leftSection={<IconSearch size={16} style={{ color: 'var(--color-foreground-subtle)' }} />}
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
                      leftSection={<IconChartBar size={14} style={{ color: 'var(--color-primary)' }} />}
                      onClick={() => navigate('/question/new')}
                    >
                      New question
                    </Menu.Item>
                    <Menu.Item
                      leftSection={<IconLayoutDashboard size={14} style={{ color: 'var(--color-success)' }} />}
                      onClick={() => navigate('/dashboard/new')}
                    >
                      New dashboard
                    </Menu.Item>
                    <Menu.Divider />
                    <Menu.Item
                      leftSection={<IconFolderPlus size={14} style={{ color: 'var(--color-warning)' }} />}
                      onClick={openNewCollectionModal}
                    >
                      New collection
                    </Menu.Item>
                  </Menu.Dropdown>
                </Menu>
              </Group>
            }
          />
        </div>
      ) : (
        <PageHeader
          title="Browse"
          subtitle="Explore your data"
          actions={
            <Group gap="sm">
              <TextInput
                placeholder="Search..."
                leftSection={<IconSearch size={16} style={{ color: 'var(--color-foreground-subtle)' }} />}
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
                    leftSection={<IconChartBar size={14} style={{ color: 'var(--color-primary)' }} />}
                    onClick={() => navigate('/question/new')}
                  >
                    New question
                  </Menu.Item>
                  <Menu.Item
                    leftSection={<IconLayoutDashboard size={14} style={{ color: 'var(--color-success)' }} />}
                    onClick={() => navigate('/dashboard/new')}
                  >
                    New dashboard
                  </Menu.Item>
                  <Menu.Divider />
                  <Menu.Item
                    leftSection={<IconFolderPlus size={14} style={{ color: 'var(--color-warning)' }} />}
                    onClick={openNewCollectionModal}
                  >
                    New collection
                  </Menu.Item>
                </Menu.Dropdown>
              </Menu>
            </Group>
          }
        />
      )}

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
              <SectionHeader
                title="Collections"
                icon={<IconFolder size={18} />}
                count={filteredItems.collections.length}
                actions={
                  <Button
                    variant="subtle"
                    size="sm"
                    rightSection={<IconArrowRight size={14} />}
                    onClick={() => setActiveTab('collections')}
                  >
                    View all
                  </Button>
                }
              />
            )}
            {viewMode === 'grid' ? (
              <CardGrid cols={{ base: 1, sm: 2, md: 4 }}>
                {filteredItems.collections.map((collection, i) => (
                  <DataCard
                    key={collection.id}
                    id={collection.id}
                    type="collection"
                    name={collection.name}
                    colorIndex={i}
                    onClick={() => navigate(`/browse/${collection.id}`)}
                  />
                ))}
              </CardGrid>
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
              <SectionHeader
                title="Dashboards"
                icon={<IconLayoutDashboard size={18} />}
                count={filteredItems.dashboards.length}
                actions={
                  <Button
                    variant="subtle"
                    size="sm"
                    rightSection={<IconArrowRight size={14} />}
                    onClick={() => setActiveTab('dashboards')}
                  >
                    View all
                  </Button>
                }
              />
            )}
            {viewMode === 'grid' ? (
              <CardGrid cols={{ base: 1, sm: 2, md: 3 }}>
                {filteredItems.dashboards.map((dashboard, i) => (
                  <DataCard
                    key={dashboard.id}
                    id={dashboard.id}
                    type="dashboard"
                    name={dashboard.name}
                    description={dashboard.description}
                    badge={`${dashboard.cards?.length || 0} cards`}
                    colorIndex={i}
                    onClick={() => navigate(`/dashboard/${dashboard.id}`)}
                  />
                ))}
              </CardGrid>
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
              <SectionHeader
                title="Questions"
                icon={<IconChartBar size={18} />}
                count={filteredItems.questions.length}
                actions={
                  <Button
                    variant="subtle"
                    size="sm"
                    rightSection={<IconArrowRight size={14} />}
                    onClick={() => setActiveTab('questions')}
                  >
                    View all
                  </Button>
                }
              />
            )}
            {viewMode === 'grid' ? (
              <CardGrid cols={{ base: 1, sm: 2, md: 3 }}>
                {filteredItems.questions.map((question, i) => (
                  <DataCard
                    key={question.id}
                    id={question.id}
                    type="question"
                    name={question.name}
                    description={question.description}
                    badge={question.visualization?.type || 'table'}
                    colorIndex={i}
                    onClick={() => navigate(`/question/${question.id}`)}
                  />
                ))}
              </CardGrid>
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
        <EmptyState
          icon={<IconFolder size={32} strokeWidth={1.5} />}
          iconColor="var(--color-info)"
          title={search ? 'No results found' : 'This collection is empty'}
          description={
            search
              ? 'Try adjusting your search terms'
              : 'Create a new question or dashboard to get started'
          }
          action={
            !search && (
              <Button
                leftSection={<IconChartBar size={16} />}
                onClick={() => navigate('/question/new')}
              >
                New Question
              </Button>
            )
          }
          secondaryAction={
            !search && (
              <Button
                variant="light"
                leftSection={<IconLayoutDashboard size={16} />}
                onClick={() => navigate('/dashboard/new')}
              >
                New Dashboard
              </Button>
            )
          }
        />
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
    </PageContainer>
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
  const colorVar = type === 'collection' ? 'var(--color-info)' : type === 'dashboard' ? 'var(--color-success)' : 'var(--color-primary)'

  return (
    <Table striped highlightOnHover className="data-table">
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
                <div
                  style={{
                    width: 24,
                    height: 24,
                    borderRadius: 'var(--radius-sm)',
                    backgroundColor: `${colorVar}15`,
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                  }}
                >
                  <Icon size={14} style={{ color: colorVar }} />
                </div>
                <span style={{ fontWeight: 500, color: 'var(--color-foreground)' }}>{item.name}</span>
              </Group>
            </Table.Td>
            <Table.Td>
              <span
                style={{
                  fontSize: '0.75rem',
                  fontWeight: 500,
                  padding: '0.125rem 0.5rem',
                  borderRadius: 'var(--radius-full)',
                  backgroundColor: `${colorVar}15`,
                  color: colorVar,
                  textTransform: 'capitalize',
                }}
              >
                {type}
              </span>
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
    <PageContainer>
      <PageHeader
        title="Databases"
        subtitle="Explore your connected data sources"
        actions={
          <Group gap="sm">
            <TextInput
              placeholder="Search databases..."
              leftSection={<IconSearch size={16} style={{ color: 'var(--color-foreground-subtle)' }} />}
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              w={250}
            />
            <Button leftSection={<IconPlus size={16} />} onClick={() => navigate('/admin/datamodel')}>
              Add Database
            </Button>
          </Group>
        }
      />

      {filteredDatasources.length > 0 ? (
        <CardGrid cols={{ base: 1, sm: 2, md: 3 }}>
          {filteredDatasources.map((ds, i) => (
            <DataCard
              key={ds.id}
              id={ds.id}
              type="database"
              name={ds.name}
              description={ds.engine}
              badge={ds.database}
              colorIndex={i}
              onClick={() => navigate(`/browse/database/${ds.id}`)}
            />
          ))}
        </CardGrid>
      ) : (
        <EmptyState
          icon={<IconDatabase size={32} strokeWidth={1.5} />}
          iconColor="var(--color-warning)"
          title="No databases connected"
          description="Connect a database to start exploring your data"
          action={
            <Button leftSection={<IconPlus size={16} />} onClick={() => navigate('/admin/datamodel')}>
              Add Database
            </Button>
          }
        />
      )}
    </PageContainer>
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
    <PageContainer>
      <PageHeader
        title="Models"
        subtitle="Curated views of your data"
        actions={
          <Group gap="sm">
            <TextInput
              placeholder="Search models..."
              leftSection={<IconSearch size={16} style={{ color: 'var(--color-foreground-subtle)' }} />}
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              w={250}
            />
            <Button leftSection={<IconPlus size={16} />}>
              New Model
            </Button>
          </Group>
        }
      />

      {filteredModels.length > 0 ? (
        <CardGrid cols={{ base: 1, sm: 2, md: 3 }}>
          {filteredModels.map((model, i) => (
            <DataCard
              key={model.id}
              id={model.id}
              type="question"
              name={model.name}
              description={model.description}
              colorIndex={i}
            />
          ))}
        </CardGrid>
      ) : (
        <EmptyState
          icon={<IconFileAnalytics size={32} strokeWidth={1.5} />}
          iconColor="var(--color-info)"
          title="No models yet"
          description="Models help you define curated views of your data"
          action={
            <Button leftSection={<IconPlus size={16} />}>
              Create Model
            </Button>
          }
        />
      )}
    </PageContainer>
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
    <PageContainer>
      <PageHeader
        title="Metrics"
        subtitle="Key business metrics"
        actions={
          <Group gap="sm">
            <TextInput
              placeholder="Search metrics..."
              leftSection={<IconSearch size={16} style={{ color: 'var(--color-foreground-subtle)' }} />}
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              w={250}
            />
            <Button leftSection={<IconPlus size={16} />}>
              New Metric
            </Button>
          </Group>
        }
      />

      {filteredMetrics.length > 0 ? (
        <CardGrid cols={{ base: 1, sm: 2, md: 3 }}>
          {filteredMetrics.map((metric, i) => (
            <DataCard
              key={metric.id}
              id={metric.id}
              type="question"
              name={metric.name}
              description={metric.description}
              badge={metric.definition?.aggregation || 'count'}
              colorIndex={i}
            />
          ))}
        </CardGrid>
      ) : (
        <EmptyState
          icon={<IconChartLine size={32} strokeWidth={1.5} />}
          iconColor="var(--color-primary)"
          title="No metrics defined"
          description="Define metrics to track key business numbers"
          action={
            <Button leftSection={<IconPlus size={16} />}>
              Create Metric
            </Button>
          }
        />
      )}
    </PageContainer>
  )
}
