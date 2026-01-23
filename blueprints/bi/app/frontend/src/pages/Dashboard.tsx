import { useEffect, useState, useMemo, useCallback, useRef } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import {
  Box, Group, Text, Button, ActionIcon, Menu, Paper, Loader,
  Modal, TextInput, Textarea, Select, Stack, Title, ThemeIcon,
  Tooltip, Tabs, Switch, Badge
} from '@mantine/core'
import { useDisclosure } from '@mantine/hooks'
import { notifications } from '@mantine/notifications'
import GridLayout, { Layout } from 'react-grid-layout'
import 'react-grid-layout/css/styles.css'
import {
  IconPlus, IconDeviceFloppy, IconDots, IconDownload, IconShare,
  IconTrash, IconRefresh, IconLayoutDashboard, IconChartBar,
  IconMaximize, IconFilter, IconClock, IconLink,
  IconGripVertical, IconLetterCase, IconBookmark, IconBookmarkFilled,
  IconArrowsMaximize, IconX
} from '@tabler/icons-react'
import Visualization from '../components/visualizations'
import DashboardFilters from '../components/dashboard/DashboardFilters'
import {
  useDashboard, useCreateDashboard, useUpdateDashboard, useDeleteDashboard,
  useDashboardCards, useAddDashboardCard, useUpdateDashboardCard, useRemoveDashboardCard,
  useQuestions, useExecuteQuestion
} from '../api/hooks'
import { useBookmarkStore } from '../stores/bookmarkStore'
import type { DashboardCard, QueryResult, Question, DashboardFilter, DashboardTab } from '../api/types'
import { LoadingState, EmptyState } from '../components/ui'

// Use GridLayout directly

interface DashboardProps {
  mode?: 'view' | 'edit'
}

// Card results cache
const cardResultsCache = new Map<string, QueryResult>()

// Sample filter options based on common Northwind columns
// In production, these would be fetched via useScanColumn
function getSampleFilterOptions(filterName: string): string[] {
  const normalizedName = filterName.toLowerCase()
  if (normalizedName.includes('category')) {
    return ['Beverages', 'Condiments', 'Confections', 'Dairy Products', 'Grains/Cereals', 'Meat/Poultry', 'Produce', 'Seafood']
  }
  if (normalizedName.includes('country')) {
    return ['Argentina', 'Austria', 'Belgium', 'Brazil', 'Canada', 'Denmark', 'Finland', 'France', 'Germany', 'Ireland', 'Italy', 'Mexico', 'Norway', 'Poland', 'Portugal', 'Spain', 'Sweden', 'Switzerland', 'UK', 'USA', 'Venezuela']
  }
  if (normalizedName.includes('region')) {
    return ['North America', 'South America', 'Western Europe', 'Eastern Europe', 'Northern Europe', 'Southern Europe', 'Scandinavia']
  }
  if (normalizedName.includes('shipper') || normalizedName.includes('shipping')) {
    return ['Federal Shipping', 'Speedy Express', 'United Package']
  }
  if (normalizedName.includes('employee') || normalizedName.includes('sales rep')) {
    return ['Nancy Davolio', 'Andrew Fuller', 'Janet Leverling', 'Margaret Peacock', 'Steven Buchanan', 'Michael Suyama', 'Robert King', 'Laura Callahan', 'Anne Dodsworth']
  }
  if (normalizedName.includes('supplier')) {
    return ['Exotic Liquids', 'New Orleans Cajun Delights', 'Grandma Kelly\'s Homestead', 'Tokyo Traders', 'Cooperativa de Quesos', 'Mayumi\'s', 'Pavlova Ltd.', 'Specialty Biscuits Ltd.', 'PB Knäckebröd AB', 'Refrescos Americanas']
  }
  if (normalizedName.includes('status')) {
    return ['Active', 'Pending', 'Shipped', 'Delivered', 'Cancelled']
  }
  return []
}

export default function Dashboard({ mode: _pageMode = 'view' }: DashboardProps) {
  const { id } = useParams()
  const navigate = useNavigate()
  const isNew = !id || id === 'new'

  // Bookmark store
  const { addBookmark, removeBookmark, isBookmarked, addRecentItem } = useBookmarkStore()
  const bookmarked = isBookmarked(id || '')

  // State
  const [editMode, setEditMode] = useState(isNew)
  const [_fullscreen, setFullscreen] = useState(false)
  const [autoRefresh, setAutoRefresh] = useState<number | null>(null)
  const [saveModalOpened, { open: openSaveModal, close: closeSaveModal }] = useDisclosure(false)
  const [addCardModalOpened, { open: openAddCardModal, close: closeAddCardModal }] = useDisclosure(false)
  const [dashboardName, setDashboardName] = useState('')
  const [dashboardDescription, setDashboardDescription] = useState('')
  const [selectedQuestionId, setSelectedQuestionId] = useState<string | null>(null)
  const [cardType, setCardType] = useState<'question' | 'text' | 'heading' | 'link'>('question')
  const [cardResults, setCardResults] = useState<Record<string, QueryResult>>({})
  const [loadingCards, setLoadingCards] = useState<Record<string, boolean>>({})

  // Dashboard filters state
  const [dashboardFilters, setDashboardFilters] = useState<DashboardFilter[]>([])
  const [filterValues, setFilterValues] = useState<Record<string, any>>({})
  // Filter dropdown options (map of filterId -> array of string values)
  const [filterOptions, setFilterOptions] = useState<Record<string, string[]>>({})

  // Dashboard tabs state
  const [dashboardTabs, setDashboardTabs] = useState<DashboardTab[]>([])
  const [activeTab, setActiveTab] = useState<string | null>(null)
  const [editingTabId, setEditingTabId] = useState<string | null>(null)
  const [editingTabName, setEditingTabName] = useState('')

  // Responsive grid width
  const gridContainerRef = useRef<HTMLDivElement>(null)
  const [gridWidth, setGridWidth] = useState(1200)

  // Queries
  const { data: dashboard, isLoading: loadingDashboard } = useDashboard(isNew ? '' : id!)
  const { data: cards, refetch: refetchCards } = useDashboardCards(isNew ? '' : id!)
  const { data: questions } = useQuestions()
  const createDashboard = useCreateDashboard()
  const updateDashboard = useUpdateDashboard()
  const deleteDashboard = useDeleteDashboard()
  const addDashboardCard = useAddDashboardCard()
  const updateDashboardCard = useUpdateDashboardCard()
  const removeDashboardCard = useRemoveDashboardCard()
  const executeQuestion = useExecuteQuestion()

  // Load dashboard data
  useEffect(() => {
    if (dashboard) {
      setDashboardName(dashboard.name)
      setDashboardDescription(dashboard.description || '')
      setDashboardFilters(dashboard.filters || [])
      setDashboardTabs(dashboard.tabs || [])
      // Track in recents
      addRecentItem({
        id: dashboard.id,
        type: 'dashboard',
        name: dashboard.name,
      })
    }
  }, [dashboard])

  // Auto-refresh interval
  useEffect(() => {
    if (!autoRefresh || isNew) return
    const interval = setInterval(() => {
      handleRefreshAll()
    }, autoRefresh * 1000)
    return () => clearInterval(interval)
  }, [autoRefresh, isNew])

  // Responsive grid width measurement
  useEffect(() => {
    const container = gridContainerRef.current
    if (!container) return

    const updateWidth = () => {
      const width = container.offsetWidth
      if (width > 0) {
        setGridWidth(width - 32) // Account for padding
      }
    }

    updateWidth()
    const resizeObserver = new ResizeObserver(updateWidth)
    resizeObserver.observe(container)

    return () => resizeObserver.disconnect()
  }, [])

  // Handle filter value change
  const handleFilterChange = (filterId: string, value: any) => {
    setFilterValues(prev => ({ ...prev, [filterId]: value }))
    // Refresh cards when filter changes
    setTimeout(() => {
      handleRefreshAll()
    }, 300)
  }

  // Toggle bookmark
  const toggleBookmark = () => {
    if (!dashboard) return
    if (bookmarked) {
      removeBookmark(dashboard.id)
    } else {
      addBookmark({
        id: dashboard.id,
        type: 'dashboard',
        name: dashboard.name,
      })
    }
  }

  // Toggle fullscreen
  const toggleFullscreen = () => {
    if (!document.fullscreenElement) {
      document.documentElement.requestFullscreen()
      setFullscreen(true)
    } else {
      document.exitFullscreen()
      setFullscreen(false)
    }
  }

  // Load card results
  useEffect(() => {
    if (cards && cards.length > 0) {
      cards.forEach(card => {
        if (card.question_id && !cardResults[card.id]) {
          loadCardResult(card)
        }
      })
    }
  }, [cards])

  const loadCardResult = async (card: DashboardCard) => {
    if (!card.question_id) return

    // Check cache first
    const cached = cardResultsCache.get(card.question_id)
    if (cached) {
      setCardResults(prev => ({ ...prev, [card.id]: cached }))
      return
    }

    setLoadingCards(prev => ({ ...prev, [card.id]: true }))
    try {
      const result = await executeQuestion.mutateAsync(card.question_id)
      cardResultsCache.set(card.question_id, result)
      setCardResults(prev => ({ ...prev, [card.id]: result }))
    } catch (err) {
      console.error('Failed to load card result:', err)
    } finally {
      setLoadingCards(prev => ({ ...prev, [card.id]: false }))
    }
  }

  // Refresh all cards
  const handleRefreshAll = () => {
    cardResultsCache.clear()
    setCardResults({})
    cards?.forEach(card => loadCardResult(card))
  }

  // Create/save dashboard
  const handleSave = async () => {
    if (!dashboardName.trim()) {
      notifications.show({
        title: 'Name required',
        message: 'Please enter a name for this dashboard',
        color: 'yellow',
      })
      return
    }

    try {
      if (isNew) {
        const newDashboard = await createDashboard.mutateAsync({
          name: dashboardName,
          description: dashboardDescription || undefined,
        })
        notifications.show({
          title: 'Dashboard created',
          message: 'Your dashboard has been created',
          color: 'green',
        })
        closeSaveModal()
        navigate(`/dashboard/${newDashboard.id}`)
      } else {
        await updateDashboard.mutateAsync({
          id: id!,
          name: dashboardName,
          description: dashboardDescription || undefined,
        })
        notifications.show({
          title: 'Dashboard saved',
          message: 'Your changes have been saved',
          color: 'green',
        })
        setEditMode(false)
      }
    } catch (err: any) {
      notifications.show({
        title: 'Error',
        message: err.message || 'Failed to save dashboard',
        color: 'red',
      })
    }
  }

  // Delete dashboard
  const handleDelete = async () => {
    if (!id || isNew) return

    if (!confirm('Are you sure you want to delete this dashboard?')) return

    try {
      await deleteDashboard.mutateAsync(id)
      notifications.show({
        title: 'Dashboard deleted',
        message: 'The dashboard has been deleted',
        color: 'green',
      })
      navigate('/browse')
    } catch (err: any) {
      notifications.show({
        title: 'Error',
        message: err.message || 'Failed to delete dashboard',
        color: 'red',
      })
    }
  }

  // Add card
  const handleAddCard = async () => {
    if (isNew || !id) {
      notifications.show({
        title: 'Save first',
        message: 'Please save the dashboard before adding cards',
        color: 'yellow',
      })
      return
    }

    const maxY = Math.max(0, ...(cards || []).map(c => c.row + c.height))

    try {
      await addDashboardCard.mutateAsync({
        dashboardId: id,
        question_id: cardType === 'question' ? selectedQuestionId || undefined : undefined,
        card_type: cardType,
        row: maxY,
        col: 0,
        width: cardType === 'question' ? 6 : 4,
        height: cardType === 'question' ? 4 : 1,
        title: cardType !== 'question' ? 'New Card' : undefined,
      })
      notifications.show({
        title: 'Card added',
        message: 'The card has been added to the dashboard',
        color: 'green',
      })
      refetchCards()
      closeAddCardModal()
      setSelectedQuestionId(null)
    } catch (err: any) {
      notifications.show({
        title: 'Error',
        message: err.message || 'Failed to add card',
        color: 'red',
      })
    }
  }

  // Remove card
  const handleRemoveCard = async (cardId: string) => {
    if (!id) return

    try {
      await removeDashboardCard.mutateAsync({ dashboardId: id, cardId })
      refetchCards()
    } catch (err: any) {
      notifications.show({
        title: 'Error',
        message: err.message || 'Failed to remove card',
        color: 'red',
      })
    }
  }

  // Handle grid layout change
  const handleLayoutChange = useCallback(async (newLayout: Layout) => {
    if (!id || !editMode) return

    // Update each card position
    for (const item of newLayout) {
      const card = cards?.find(c => c.id === item.i)
      if (card && (card.row !== item.y || card.col !== item.x || card.width !== item.w || card.height !== item.h)) {
        try {
          await updateDashboardCard.mutateAsync({
            dashboardId: id,
            id: item.i,
            row: item.y,
            col: item.x,
            width: item.w,
            height: item.h,
          })
        } catch (err) {
          console.error('Failed to update card position:', err)
        }
      }
    }
  }, [id, cards, editMode, updateDashboardCard])

  // Get question for card
  const getQuestionForCard = (card: DashboardCard): Question | undefined => {
    return questions?.find(q => q.id === card.question_id)
  }

  // Filter cards by active tab
  const filteredCards = useMemo(() => {
    if (!cards) return []
    if (!activeTab) return cards
    return cards.filter(card => card.tab_id === activeTab)
  }, [cards, activeTab])

  // Filter layout to match filtered cards
  const filteredLayout: Layout = useMemo(() => {
    return filteredCards.map(card => ({
      i: card.id,
      x: card.col,
      y: card.row,
      w: card.width,
      h: card.height,
      minW: 2,
      minH: 2,
    }))
  }, [filteredCards])

  // Add new tab
  const handleAddTab = () => {
    const newTab: DashboardTab = {
      id: Math.random().toString(36).substring(2, 9),
      dashboard_id: id!,
      name: `Tab ${dashboardTabs.length + 1}`,
      position: dashboardTabs.length,
    }
    setDashboardTabs(prev => [...prev, newTab])
    setActiveTab(newTab.id)
  }

  // Rename tab
  const handleRenameTab = (tabId: string) => {
    if (!editingTabName.trim()) return
    setDashboardTabs(prev => prev.map(tab =>
      tab.id === tabId ? { ...tab, name: editingTabName } : tab
    ))
    setEditingTabId(null)
    setEditingTabName('')
  }

  // Remove tab
  const handleRemoveTab = (tabId: string) => {
    setDashboardTabs(prev => prev.filter(tab => tab.id !== tabId))
    if (activeTab === tabId) {
      setActiveTab(null)
    }
    // Unassign cards from removed tab
    cards?.forEach(card => {
      if (card.tab_id === tabId && id) {
        updateDashboardCard.mutate({
          dashboardId: id,
          id: card.id,
          tab_id: undefined,
        })
      }
    })
  }

  if (loadingDashboard && !isNew) {
    return (
      <Box px="lg" py="lg" style={{ width: '100%', backgroundColor: 'var(--color-background-muted)', minHeight: '100vh' }}>
        <LoadingState message="Loading dashboard..." />
      </Box>
    )
  }

  return (
    <Box style={{ minHeight: '100vh', backgroundColor: 'var(--color-background-muted)' }}>
      {/* Header */}
      <Box
        p="md"
        style={{
          backgroundColor: 'var(--color-background)',
          borderBottom: '1px solid var(--color-border)',
        }}
      >
        <Group justify="space-between">
          <Group gap="md">
            <ThemeIcon
              size={40}
              radius="md"
              variant="light"
              style={{ backgroundColor: 'var(--color-success)15', color: 'var(--color-success)' }}
            >
              <IconLayoutDashboard size={20} />
            </ThemeIcon>
            <div>
              {editMode && !isNew ? (
                <TextInput
                  value={dashboardName}
                  onChange={(e) => setDashboardName(e.target.value)}
                  placeholder="Dashboard name"
                  size="lg"
                  variant="unstyled"
                  styles={{ input: { fontSize: '1.5rem', fontWeight: 600, color: 'var(--color-foreground)' } }}
                />
              ) : (
                <Title order={2} style={{ color: 'var(--color-foreground)' }}>{dashboard?.name || 'New Dashboard'}</Title>
              )}
              <Text size="sm" style={{ color: 'var(--color-foreground-muted)' }}>
                {cards?.length || 0} cards
              </Text>
            </div>
          </Group>

          <Group gap="sm">
            {!isNew && (
              <>
                <Tooltip label={bookmarked ? 'Remove bookmark' : 'Bookmark'}>
                  <ActionIcon
                    variant="subtle"
                    size="lg"
                    color={bookmarked ? 'yellow' : 'gray'}
                    onClick={toggleBookmark}
                  >
                    {bookmarked ? <IconBookmarkFilled size={20} /> : <IconBookmark size={20} />}
                  </ActionIcon>
                </Tooltip>
                <Tooltip label="Refresh all cards">
                  <ActionIcon variant="subtle" size="lg" onClick={handleRefreshAll}>
                    <IconRefresh size={20} />
                  </ActionIcon>
                </Tooltip>
                <Tooltip label="Fullscreen">
                  <ActionIcon variant="subtle" size="lg" onClick={toggleFullscreen}>
                    <IconArrowsMaximize size={20} />
                  </ActionIcon>
                </Tooltip>
              </>
            )}

            {autoRefresh && (
              <Badge variant="light" color="brand" leftSection={<IconClock size={12} />}>
                {autoRefresh}s
              </Badge>
            )}

            {editMode && !isNew && (
              <>
                <Button
                  variant="light"
                  leftSection={<IconFilter size={16} />}
                  onClick={() => {
                    const newFilter: DashboardFilter = {
                      id: Math.random().toString(36).substring(2, 9),
                      dashboard_id: id!,
                      name: 'New Filter',
                      type: 'text',
                      required: false,
                      targets: [],
                    }
                    setDashboardFilters(prev => [...prev, newFilter])
                  }}
                >
                  Add Filter
                </Button>
                <Button
                  variant="light"
                  leftSection={<IconPlus size={16} />}
                  onClick={openAddCardModal}
                >
                  Add Card
                </Button>
              </>
            )}

            <Switch
              label="Edit"
              checked={editMode}
              onChange={(e) => setEditMode(e.currentTarget.checked)}
              disabled={isNew}
            />

            <Button
              leftSection={<IconDeviceFloppy size={16} />}
              onClick={isNew ? openSaveModal : handleSave}
              loading={createDashboard.isPending || updateDashboard.isPending}
            >
              Save
            </Button>

            <Menu position="bottom-end">
              <Menu.Target>
                <ActionIcon variant="subtle" size="lg">
                  <IconDots size={20} />
                </ActionIcon>
              </Menu.Target>
              <Menu.Dropdown>
                <Menu.Item leftSection={<IconDownload size={14} />}>
                  Download as PDF
                </Menu.Item>
                <Menu.Item leftSection={<IconShare size={14} />}>
                  Share
                </Menu.Item>
                <Menu.Label>Auto-refresh</Menu.Label>
                <Menu.Item onClick={() => setAutoRefresh(null)}>
                  Off {!autoRefresh && '✓'}
                </Menu.Item>
                <Menu.Item onClick={() => setAutoRefresh(30)}>
                  30 seconds {autoRefresh === 30 && '✓'}
                </Menu.Item>
                <Menu.Item onClick={() => setAutoRefresh(60)}>
                  1 minute {autoRefresh === 60 && '✓'}
                </Menu.Item>
                <Menu.Item onClick={() => setAutoRefresh(300)}>
                  5 minutes {autoRefresh === 300 && '✓'}
                </Menu.Item>
                <Menu.Divider />
                {!isNew && (
                  <Menu.Item leftSection={<IconTrash size={14} />} color="red" onClick={handleDelete}>
                    Delete
                  </Menu.Item>
                )}
              </Menu.Dropdown>
            </Menu>
          </Group>
        </Group>
      </Box>

      {/* Dashboard Grid */}
      <Box ref={gridContainerRef} px="lg" py="lg" style={{ width: '100%' }}>
        {/* Dashboard Filters */}
        {(dashboardFilters.length > 0 || editMode) && !isNew && (
          <DashboardFilters
            filters={dashboardFilters}
            filterValues={filterValues}
            filterOptions={filterOptions}
            onFilterChange={handleFilterChange}
            onAddFilter={editMode ? (filter) => {
              const newFilter: DashboardFilter = {
                ...filter,
                id: Math.random().toString(36).substring(2, 9),
                dashboard_id: id!,
              }
              setDashboardFilters(prev => [...prev, newFilter])
              // Initialize with sample options for category filters
              if (filter.type === 'category' || filter.display_type === 'dropdown') {
                // These would typically come from column scan results
                // For now, use sample data based on common Northwind columns
                const sampleOptions = getSampleFilterOptions(filter.name)
                if (sampleOptions.length > 0) {
                  setFilterOptions(prev => ({ ...prev, [newFilter.id]: sampleOptions }))
                }
              }
            } : undefined}
            onRemoveFilter={editMode ? (filterId) => {
              setDashboardFilters(prev => prev.filter(f => f.id !== filterId))
              setFilterValues(prev => {
                const next = { ...prev }
                delete next[filterId]
                return next
              })
              setFilterOptions(prev => {
                const next = { ...prev }
                delete next[filterId]
                return next
              })
            } : undefined}
            editMode={editMode}
          />
        )}

        {/* Dashboard Tabs */}
        {(dashboardTabs.length > 0 || editMode) && !isNew && (
          <Paper withBorder radius="sm" mb="md" p={0} style={{ backgroundColor: 'white' }}>
            <Group gap={0} wrap="nowrap">
              <Tabs
                value={activeTab || ''}
                onChange={(value) => setActiveTab(value === '' ? null : value)}
                style={{ flex: 1 }}
              >
                <Tabs.List style={{ borderBottom: 'none' }}>
                  <Tabs.Tab value="" key="all">
                    All
                  </Tabs.Tab>
                  {dashboardTabs.map((tab) => (
                    <Tabs.Tab
                      key={tab.id}
                      value={tab.id}
                      onDoubleClick={() => {
                        if (editMode) {
                          setEditingTabId(tab.id)
                          setEditingTabName(tab.name)
                        }
                      }}
                      rightSection={editMode && (
                        <ActionIcon
                          size="xs"
                          variant="subtle"
                          color="gray"
                          onClick={(e) => {
                            e.stopPropagation()
                            handleRemoveTab(tab.id)
                          }}
                        >
                          <IconX size={12} />
                        </ActionIcon>
                      )}
                    >
                      {editingTabId === tab.id ? (
                        <TextInput
                          size="xs"
                          value={editingTabName}
                          onChange={(e) => setEditingTabName(e.target.value)}
                          onBlur={() => handleRenameTab(tab.id)}
                          onKeyDown={(e) => {
                            if (e.key === 'Enter') handleRenameTab(tab.id)
                            if (e.key === 'Escape') {
                              setEditingTabId(null)
                              setEditingTabName('')
                            }
                          }}
                          autoFocus
                          styles={{ input: { minWidth: 60, padding: '2px 6px', height: 24 } }}
                          onClick={(e) => e.stopPropagation()}
                        />
                      ) : (
                        tab.name
                      )}
                    </Tabs.Tab>
                  ))}
                </Tabs.List>
              </Tabs>
              {editMode && (
                <ActionIcon
                  variant="subtle"
                  color="gray"
                  size="md"
                  mr="xs"
                  onClick={handleAddTab}
                >
                  <IconPlus size={16} />
                </ActionIcon>
              )}
            </Group>
          </Paper>
        )}

        {filteredCards && filteredCards.length > 0 ? (
          <GridLayout
            width={gridWidth}
            className="layout"
            layout={filteredLayout}
            data-testid="dashboard-grid"
            gridConfig={{ cols: 18, rowHeight: 80, margin: [16, 16], containerPadding: null, maxRows: Infinity }}
            dragConfig={{ enabled: editMode, bounded: false, handle: '.drag-handle', threshold: 3 }}
            resizeConfig={{ enabled: editMode }}
            onLayoutChange={handleLayoutChange}
          >
            {filteredCards.map(card => (
              <div key={card.id} data-testid="dashboard-card">
                <DashboardCardComponent
                  card={card}
                  question={getQuestionForCard(card)}
                  result={cardResults[card.id]}
                  isLoading={loadingCards[card.id]}
                  editMode={editMode}
                  onRefresh={() => loadCardResult(card)}
                  onRemove={() => handleRemoveCard(card.id)}
                  onNavigate={() => card.question_id && navigate(`/question/${card.question_id}`)}
                  onUpdateContent={(content, title) => {
                    if (!id) return
                    const update: any = { dashboardId: id, id: card.id }
                    if (card.card_type === 'link') {
                      update.title = title || content
                      update.settings = { ...card.settings, url: content }
                    } else {
                      update.content = content
                    }
                    updateDashboardCard.mutate(update)
                  }}
                />
              </div>
            ))}
          </GridLayout>
        ) : (
          <EmptyState
            icon={<IconLayoutDashboard size={40} strokeWidth={1.5} />}
            iconColor="var(--color-success)"
            title={isNew ? 'Create your dashboard' : 'Add your first card'}
            description={
              isNew
                ? 'Give your dashboard a name and save it, then add cards to visualize your data.'
                : activeTab
                  ? 'No cards in this tab. Add cards and assign them to this tab.'
                  : 'Click "Add Card" to add questions to this dashboard.'
            }
            size="lg"
            action={
              isNew ? (
                <Button
                  size="lg"
                  leftSection={<IconDeviceFloppy size={20} />}
                  onClick={openSaveModal}
                >
                  Save Dashboard
                </Button>
              ) : (
                <Button
                  size="lg"
                  leftSection={<IconPlus size={20} />}
                  onClick={openAddCardModal}
                >
                  Add Card
                </Button>
              )
            }
          />
        )}
      </Box>

      {/* Save Modal */}
      <Modal opened={saveModalOpened} onClose={closeSaveModal} title="Save Dashboard">
        <Stack gap="md">
          <TextInput
            label="Name"
            placeholder="My Dashboard"
            value={dashboardName}
            onChange={(e) => setDashboardName(e.target.value)}
            required
          />
          <Textarea
            label="Description"
            placeholder="Optional description..."
            value={dashboardDescription}
            onChange={(e) => setDashboardDescription(e.target.value)}
            rows={3}
          />
          <Group justify="flex-end" mt="md">
            <Button variant="light" onClick={closeSaveModal}>Cancel</Button>
            <Button onClick={handleSave} loading={createDashboard.isPending}>
              {isNew ? 'Create' : 'Save'}
            </Button>
          </Group>
        </Stack>
      </Modal>

      {/* Add Card Modal */}
      <Modal opened={addCardModalOpened} onClose={closeAddCardModal} title="Add Card">
        <Stack gap="md">
          <Tabs value={cardType} onChange={(v) => setCardType(v as any)}>
            <Tabs.List>
              <Tabs.Tab value="question" leftSection={<IconChartBar size={14} />}>
                Question
              </Tabs.Tab>
              <Tabs.Tab value="text" leftSection={<IconLetterCase size={14} />}>
                Text
              </Tabs.Tab>
              <Tabs.Tab value="heading" leftSection={<IconLetterCase size={14} />}>
                Heading
              </Tabs.Tab>
              <Tabs.Tab value="link" leftSection={<IconLink size={14} />}>
                Link
              </Tabs.Tab>
            </Tabs.List>
          </Tabs>

          {cardType === 'question' && (
            <Select
              label="Select a question"
              placeholder="Choose a saved question"
              data={(questions || []).map(q => ({
                value: q.id,
                label: q.name,
                description: q.description,
              }))}
              value={selectedQuestionId}
              onChange={setSelectedQuestionId}
              searchable
              nothingFoundMessage="No questions found"
            />
          )}

          {cardType === 'text' && (
            <Text size="sm" c="dimmed">
              Add a text box to add context to your dashboard.
            </Text>
          )}

          {cardType === 'heading' && (
            <Text size="sm" c="dimmed">
              Add a heading to organize your dashboard sections.
            </Text>
          )}

          {cardType === 'link' && (
            <Text size="sm" c="dimmed">
              Add a link button to navigate to another page or dashboard.
            </Text>
          )}

          <Group justify="flex-end" mt="md">
            <Button variant="light" onClick={closeAddCardModal}>Cancel</Button>
            <Button
              onClick={handleAddCard}
              disabled={cardType === 'question' && !selectedQuestionId}
              loading={addDashboardCard.isPending}
            >
              Add Card
            </Button>
          </Group>
        </Stack>
      </Modal>
    </Box>
  )
}

// Dashboard Card Component
function DashboardCardComponent({
  card,
  question,
  result,
  isLoading,
  editMode,
  onRefresh,
  onRemove,
  onNavigate,
  onUpdateContent,
}: {
  card: DashboardCard
  question: Question | undefined
  result: QueryResult | undefined
  isLoading: boolean
  editMode: boolean
  onRefresh: () => void
  onRemove: () => void
  onNavigate: () => void
  onUpdateContent?: (content: string, title?: string) => void
}) {
  const [hovered, setHovered] = useState(false)
  const [isEditing, setIsEditing] = useState(false)
  const [editContent, setEditContent] = useState(card.content || card.title || '')
  const [editTitle, setEditTitle] = useState(card.title || '')
  const [editUrl, setEditUrl] = useState(card.settings?.url || '')

  // Text or heading card
  if (card.card_type === 'text' || card.card_type === 'heading') {
    const handleSaveContent = () => {
      if (onUpdateContent && editContent !== (card.content || card.title)) {
        onUpdateContent(editContent)
      }
      setIsEditing(false)
    }

    return (
      <Paper
        withBorder
        radius="md"
        p="md"
        h="100%"
        onMouseEnter={() => setHovered(true)}
        onMouseLeave={() => setHovered(false)}
      >
        {editMode && (
          <Group
            justify="space-between"
            mb="xs"
            className="drag-handle"
            style={{ cursor: 'grab' }}
          >
            <IconGripVertical size={16} color="var(--mantine-color-gray-5)" />
            {hovered && (
              <ActionIcon variant="subtle" size="sm" color="red" onClick={onRemove}>
                <IconTrash size={14} />
              </ActionIcon>
            )}
          </Group>
        )}
        {isEditing && editMode ? (
          card.card_type === 'heading' ? (
            <TextInput
              value={editContent}
              onChange={(e) => setEditContent(e.target.value)}
              onBlur={handleSaveContent}
              onKeyDown={(e) => {
                if (e.key === 'Enter') handleSaveContent()
                if (e.key === 'Escape') {
                  setEditContent(card.content || card.title || '')
                  setIsEditing(false)
                }
              }}
              autoFocus
              variant="unstyled"
              styles={{ input: { fontWeight: 600, fontSize: 'var(--mantine-font-size-lg)' } }}
            />
          ) : (
            <Textarea
              value={editContent}
              onChange={(e) => setEditContent(e.target.value)}
              onBlur={handleSaveContent}
              onKeyDown={(e) => {
                if (e.key === 'Escape') {
                  setEditContent(card.content || card.title || '')
                  setIsEditing(false)
                }
              }}
              autoFocus
              variant="unstyled"
              autosize
              minRows={2}
              styles={{ input: { fontSize: 'var(--mantine-font-size-sm)' } }}
            />
          )
        ) : (
          <Text
            fw={card.card_type === 'heading' ? 600 : 400}
            size={card.card_type === 'heading' ? 'lg' : 'sm'}
            onClick={() => editMode && setIsEditing(true)}
            style={{ cursor: editMode ? 'text' : 'default' }}
          >
            {card.content || card.title || (editMode ? 'Click to edit...' : '')}
          </Text>
        )}
      </Paper>
    )
  }

  // Link card
  if (card.card_type === 'link') {
    const handleSaveLinkContent = () => {
      if (onUpdateContent) {
        onUpdateContent(editUrl, editTitle)
      }
      setIsEditing(false)
    }

    const handleLinkClick = () => {
      if (!editMode && card.settings?.url) {
        window.open(card.settings.url, '_blank')
      }
    }

    return (
      <Paper
        withBorder
        radius="md"
        p="md"
        h="100%"
        style={{ cursor: editMode ? 'default' : 'pointer' }}
        onMouseEnter={() => setHovered(true)}
        onMouseLeave={() => setHovered(false)}
        onClick={handleLinkClick}
      >
        {editMode && (
          <Group
            justify="space-between"
            mb="xs"
            className="drag-handle"
            style={{ cursor: 'grab' }}
          >
            <IconGripVertical size={16} color="var(--mantine-color-gray-5)" />
            {hovered && (
              <ActionIcon variant="subtle" size="sm" color="red" onClick={onRemove}>
                <IconTrash size={14} />
              </ActionIcon>
            )}
          </Group>
        )}
        {isEditing && editMode ? (
          <Stack gap="xs">
            <TextInput
              size="sm"
              placeholder="Link title"
              value={editTitle}
              onChange={(e) => setEditTitle(e.target.value)}
              leftSection={<IconLink size={14} />}
            />
            <TextInput
              size="sm"
              placeholder="URL (e.g., https://example.com)"
              value={editUrl}
              onChange={(e) => setEditUrl(e.target.value)}
            />
            <Group gap="xs">
              <Button size="xs" onClick={handleSaveLinkContent}>Save</Button>
              <Button size="xs" variant="light" onClick={() => {
                setEditTitle(card.title || '')
                setEditUrl(card.settings?.url || '')
                setIsEditing(false)
              }}>Cancel</Button>
            </Group>
          </Stack>
        ) : (
          <Group gap="sm" onClick={(e) => {
            if (editMode) {
              e.stopPropagation()
              setIsEditing(true)
            }
          }}>
            <IconLink size={16} />
            <Text>{card.title || (editMode ? 'Click to configure link...' : 'Link')}</Text>
          </Group>
        )}
      </Paper>
    )
  }

  // Question card
  return (
    <Paper
      withBorder
      radius="md"
      h="100%"
      style={{
        display: 'flex',
        flexDirection: 'column',
        overflow: 'hidden',
        backgroundColor: 'var(--color-background)',
        borderColor: 'var(--color-border)',
      }}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
    >
      {/* Card Header */}
      <Group
        justify="space-between"
        p="sm"
        style={{ borderBottom: '1px solid var(--color-border)' }}
        className={editMode ? 'drag-handle' : ''}
      >
        <Group gap="sm">
          {editMode && <IconGripVertical size={16} style={{ color: 'var(--color-foreground-subtle)', cursor: 'grab' }} />}
          <Text
            fw={500}
            size="sm"
            style={{ cursor: 'pointer', color: 'var(--color-foreground)' }}
            onClick={onNavigate}
          >
            {card.title || question?.name || 'Untitled'}
          </Text>
        </Group>
        {(hovered || editMode) && (
          <Group gap={4}>
            <Tooltip label="Refresh">
              <ActionIcon variant="subtle" size="sm" onClick={onRefresh}>
                <IconRefresh size={14} />
              </ActionIcon>
            </Tooltip>
            <Tooltip label="Fullscreen">
              <ActionIcon variant="subtle" size="sm" onClick={onNavigate}>
                <IconMaximize size={14} />
              </ActionIcon>
            </Tooltip>
            {editMode && (
              <Tooltip label="Remove">
                <ActionIcon variant="subtle" size="sm" color="red" onClick={onRemove}>
                  <IconTrash size={14} />
                </ActionIcon>
              </Tooltip>
            )}
          </Group>
        )}
      </Group>

      {/* Card Content */}
      <Box style={{ flex: 1, overflow: 'auto', padding: 8 }}>
        {isLoading ? (
          <Group justify="center" align="center" h="100%">
            <Loader size="sm" />
          </Group>
        ) : result ? (
          <Visualization
            result={result}
            visualization={question?.visualization || { type: 'table' }}
            height={Math.max(200, (card.height * 80) - 100)}
            showLegend={false}
          />
        ) : (
          <Group justify="center" align="center" h="100%">
            <Text c="dimmed" size="sm">No data</Text>
          </Group>
        )}
      </Box>
    </Paper>
  )
}
