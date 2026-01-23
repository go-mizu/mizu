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
  IconDots, IconPencil, IconTrash, IconStar, IconFolderPlus,
  IconGridDots, IconList, IconArrowRight, IconUser
} from '@tabler/icons-react'
import {
  useCollection, useCollectionItems, useCreateCollection,
  useCollections,
  useRootCollection, useRootCollectionItems,
  usePersonalCollection, usePersonalCollectionItems
} from '../api/hooks'
import { chartColors } from '../theme'

interface CollectionProps {
  type?: 'root' | 'personal' | 'trash'
}

export default function Collection({ type }: CollectionProps) {
  const navigate = useNavigate()
  const { id: collectionId } = useParams()

  // State
  const [search, setSearch] = useState('')
  const [viewMode, setViewMode] = useState<'grid' | 'list'>('grid')
  const [activeTab, setActiveTab] = useState<string | null>('all')
  const [newCollectionModalOpened, { open: openNewCollectionModal, close: closeNewCollectionModal }] = useDisclosure(false)
  const [newCollectionName, setNewCollectionName] = useState('')

  // Queries - use specialized hooks for root/personal collections
  const { data: allCollections } = useCollections()

  // Root collection queries
  const { data: rootCollection, isLoading: isLoadingRoot } = useRootCollection()
  const { data: rootItems, isLoading: isLoadingRootItems } = useRootCollectionItems()

  // Personal collection queries
  const { data: personalCollection, isLoading: isLoadingPersonal } = usePersonalCollection()
  const { data: personalItems, isLoading: isLoadingPersonalItems } = usePersonalCollectionItems()

  // Regular collection queries (only when viewing a specific collection)
  const { data: regularCollection, isLoading: isLoadingRegular } = useCollection(
    type ? '' : collectionId || ''
  )
  const { data: regularItems, isLoading: isLoadingRegularItems } = useCollectionItems(
    type ? '' : collectionId || ''
  )

  const createCollection = useCreateCollection()

  // Determine which collection/items to use based on type
  const currentCollection = useMemo(() => {
    if (type === 'root') return rootCollection
    if (type === 'personal') return personalCollection
    return regularCollection
  }, [type, rootCollection, personalCollection, regularCollection])

  const collectionItems = useMemo(() => {
    if (type === 'root') return rootItems
    if (type === 'personal') return personalItems
    return regularItems
  }, [type, rootItems, personalItems, regularItems])

  const isLoadingCollection = type === 'root' ? isLoadingRoot :
    type === 'personal' ? isLoadingPersonal : isLoadingRegular

  const isLoadingItems = type === 'root' ? isLoadingRootItems :
    type === 'personal' ? isLoadingPersonalItems : isLoadingRegularItems

  // Effective ID for creating subcollections
  const effectiveId = useMemo(() => {
    if (type === 'root') return currentCollection?.id || 'root'
    if (type === 'personal') return currentCollection?.id || ''
    if (type === 'trash') return 'trash'
    return collectionId || ''
  }, [type, collectionId, currentCollection?.id])

  // Build breadcrumbs for collection navigation
  const breadcrumbs = useMemo(() => {
    const crumbs: { label: string; path: string }[] = []

    if (type === 'root') {
      crumbs.push({ label: 'Our analytics', path: '/collection/root' })
    } else if (type === 'personal') {
      crumbs.push({ label: 'Your personal collection', path: '/collection/personal' })
    } else if (type === 'trash') {
      crumbs.push({ label: 'Trash', path: '/collection/trash' })
    } else if (currentCollection) {
      // Build parent chain
      const buildParentChain = (collection: typeof currentCollection): typeof crumbs => {
        if (!collection) return []
        const parent = allCollections?.find(c => c.id === collection.parent_id)
        const parentCrumbs = parent ? buildParentChain(parent) : []
        return [
          ...parentCrumbs,
          { label: collection.name, path: `/collection/${collection.id}` }
        ]
      }
      crumbs.push(...buildParentChain(currentCollection))
    }

    return crumbs
  }, [type, currentCollection, allCollections])

  // Get collection title and icon
  const { title, icon: TitleIcon, iconColor } = useMemo(() => {
    if (type === 'root') {
      return { title: 'Our analytics', icon: IconFolder, iconColor: '#7172AD' }
    }
    if (type === 'personal') {
      return { title: 'Your personal collection', icon: IconUser, iconColor: '#509EE3' }
    }
    if (type === 'trash') {
      return { title: 'Trash', icon: IconTrash, iconColor: '#E85D54' }
    }
    return {
      title: currentCollection?.name || 'Collection',
      icon: IconFolder,
      iconColor: currentCollection?.color || '#7172AD'
    }
  }, [type, currentCollection])

  // Filter items based on search
  const filteredItems = useMemo(() => {
    const searchLower = search.toLowerCase()

    return {
      collections: (collectionItems?.subcollections || []).filter(c =>
        c.name.toLowerCase().includes(searchLower)
      ),
      dashboards: (collectionItems?.dashboards || []).filter(d =>
        d.name.toLowerCase().includes(searchLower)
      ),
      questions: (collectionItems?.questions || []).filter(q =>
        q.name.toLowerCase().includes(searchLower)
      ),
    }
  }, [collectionItems, search])

  // Handle create collection
  const handleCreateCollection = async () => {
    if (!newCollectionName.trim()) return

    try {
      const newCollection = await createCollection.mutateAsync({
        name: newCollectionName,
        parent_id: effectiveId !== 'root' ? effectiveId : undefined,
      })
      notifications.show({
        title: 'Collection created',
        message: `${newCollectionName} has been created`,
        color: 'green',
      })
      closeNewCollectionModal()
      setNewCollectionName('')
      navigate(`/collection/${newCollection.id}`)
    } catch (err: any) {
      notifications.show({
        title: 'Error',
        message: err.message || 'Failed to create collection',
        color: 'red',
      })
    }
  }

  const totalItems =
    filteredItems.collections.length +
    filteredItems.dashboards.length +
    filteredItems.questions.length

  // Loading state
  if (isLoadingCollection || isLoadingItems) {
    return (
      <Container size="xl" py="lg">
        <Paper withBorder radius="md" p="xl" ta="center">
          <Text>Loading...</Text>
        </Paper>
      </Container>
    )
  }

  return (
    <Container size="xl" py="lg">
      {/* Header */}
      <Group justify="space-between" mb="xl">
        <div>
          {breadcrumbs.length > 1 && (
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
          )}
          <Group gap="sm">
            <ThemeIcon size={40} radius="md" variant="light" style={{ backgroundColor: iconColor + '20', color: iconColor }}>
              <TitleIcon size={24} />
            </ThemeIcon>
            <div>
              <Title order={2}>{title}</Title>
              {currentCollection?.description && (
                <Text c="dimmed" size="sm">{currentCollection.description}</Text>
              )}
            </div>
          </Group>
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
      {(activeTab === 'all' || activeTab === 'collections') && filteredItems.collections.length > 0 && (
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
                  onClick={() => navigate(`/collection/${collection.id}`)}
                />
              ))}
            </SimpleGrid>
          ) : (
            <ItemList
              items={filteredItems.collections}
              type="collection"
              onClick={(id) => navigate(`/collection/${id}`)}
            />
          )}
        </div>
      )}

      {(activeTab === 'all' || activeTab === 'dashboards') && filteredItems.dashboards.length > 0 && (
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
      )}

      {(activeTab === 'all' || activeTab === 'questions') && filteredItems.questions.length > 0 && (
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
      )}

      {/* Empty state */}
      {totalItems === 0 && (
        <Paper withBorder radius="md" p="xl" ta="center">
          <Stack align="center" gap="lg">
            <ThemeIcon size={60} radius="xl" variant="light" style={{ backgroundColor: iconColor + '20', color: iconColor }}>
              <TitleIcon size={30} />
            </ThemeIcon>
            <div>
              <Title order={3} mb="xs">
                {search ? 'No results found' : 'This collection is empty'}
              </Title>
              <Text c="dimmed" maw={400} mx="auto">
                {search
                  ? 'Try adjusting your search terms'
                  : type === 'trash'
                    ? 'Items you delete will appear here'
                    : 'Create a new question or dashboard to get started'}
              </Text>
            </div>
            {!search && type !== 'trash' && (
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
