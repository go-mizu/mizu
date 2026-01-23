import { useMemo, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import {
  Group, TextInput, Tabs, Menu, ActionIcon, Button, Breadcrumbs, Anchor,
  Modal, Table, SegmentedControl, Stack, ThemeIcon
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
import {
  SectionHeader, DataCard, EmptyState, CardGrid, PageContainer, LoadingState
} from '../components/ui'

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
      return { title: 'Our analytics', icon: IconFolder, iconColor: 'var(--color-info)' }
    }
    if (type === 'personal') {
      return { title: 'Your personal collection', icon: IconUser, iconColor: 'var(--color-primary)' }
    }
    if (type === 'trash') {
      return { title: 'Trash', icon: IconTrash, iconColor: 'var(--color-error)' }
    }
    return {
      title: currentCollection?.name || 'Collection',
      icon: IconFolder,
      iconColor: currentCollection?.color || 'var(--color-info)'
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
      <PageContainer>
        <LoadingState message="Loading collection..." />
      </PageContainer>
    )
  }

  return (
    <PageContainer>
      {/* Header */}
      <div style={{ marginBottom: 'var(--mantine-spacing-xl)' }}>
        {breadcrumbs.length > 1 && (
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
        )}
        <Group justify="space-between" align="flex-start">
          <Group gap="sm">
            <ThemeIcon
              size={40}
              radius="md"
              variant="light"
              style={{ backgroundColor: `${iconColor}15`, color: iconColor }}
            >
              <TitleIcon size={24} />
            </ThemeIcon>
            <div>
              <h2 style={{ margin: 0, color: 'var(--color-foreground)', fontSize: '1.5rem', fontWeight: 600 }}>
                {title}
              </h2>
              {currentCollection?.description && (
                <p style={{ margin: 0, color: 'var(--color-foreground-muted)', fontSize: '0.875rem' }}>
                  {currentCollection.description}
                </p>
              )}
            </div>
          </Group>
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
        </Group>
      </div>

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
                  onClick={() => navigate(`/collection/${collection.id}`)}
                />
              ))}
            </CardGrid>
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
      )}

      {(activeTab === 'all' || activeTab === 'questions') && filteredItems.questions.length > 0 && (
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
      )}

      {/* Empty state */}
      {totalItems === 0 && (
        <EmptyState
          icon={<TitleIcon size={32} strokeWidth={1.5} />}
          iconColor={iconColor}
          title={search ? 'No results found' : 'This collection is empty'}
          description={
            search
              ? 'Try adjusting your search terms'
              : type === 'trash'
                ? 'Items you delete will appear here'
                : 'Create a new question or dashboard to get started'
          }
          action={
            !search && type !== 'trash' && (
              <Button
                leftSection={<IconChartBar size={16} />}
                onClick={() => navigate('/question/new')}
              >
                New Question
              </Button>
            )
          }
          secondaryAction={
            !search && type !== 'trash' && (
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
