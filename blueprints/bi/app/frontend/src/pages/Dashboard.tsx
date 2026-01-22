import { useEffect, useState, useMemo, useCallback } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import {
  Container, Box, Group, Text, Button, ActionIcon, Menu, Paper, Loader,
  Modal, TextInput, Textarea, Select, Badge, Stack, Title, ThemeIcon,
  Tooltip, Tabs, Switch
} from '@mantine/core'
import { useDisclosure } from '@mantine/hooks'
import { notifications } from '@mantine/notifications'
import GridLayout, { Layout, WidthProvider } from 'react-grid-layout'
import 'react-grid-layout/css/styles.css'
import 'react-resizable/css/styles.css'
import {
  IconPlus, IconDeviceFloppy, IconDots, IconDownload, IconShare,
  IconPencil, IconTrash, IconRefresh, IconLayoutDashboard, IconChartBar,
  IconMaximize, IconFilter, IconClock, IconSettings, IconText, IconLink,
  IconGripVertical
} from '@tabler/icons-react'
import Visualization from '../components/visualizations'
import {
  useDashboard, useCreateDashboard, useUpdateDashboard, useDeleteDashboard,
  useDashboardCards, useAddDashboardCard, useUpdateDashboardCard, useRemoveDashboardCard,
  useQuestions, useExecuteQuestion
} from '../api/hooks'
import type { DashboardCard, QueryResult, Question } from '../api/types'

const ResponsiveGridLayout = WidthProvider(GridLayout)

interface DashboardProps {
  mode?: 'view' | 'edit'
}

// Card results cache
const cardResultsCache = new Map<string, QueryResult>()

export default function Dashboard({ mode: pageMode = 'view' }: DashboardProps) {
  const { id } = useParams()
  const navigate = useNavigate()
  const isNew = !id || id === 'new'

  // State
  const [editMode, setEditMode] = useState(isNew)
  const [saveModalOpened, { open: openSaveModal, close: closeSaveModal }] = useDisclosure(false)
  const [addCardModalOpened, { open: openAddCardModal, close: closeAddCardModal }] = useDisclosure(false)
  const [dashboardName, setDashboardName] = useState('')
  const [dashboardDescription, setDashboardDescription] = useState('')
  const [selectedQuestionId, setSelectedQuestionId] = useState<string | null>(null)
  const [cardType, setCardType] = useState<'question' | 'text' | 'heading' | 'link'>('question')
  const [cardResults, setCardResults] = useState<Record<string, QueryResult>>({})
  const [loadingCards, setLoadingCards] = useState<Record<string, boolean>>({})

  // Queries
  const { data: dashboard, isLoading: loadingDashboard } = useDashboard(isNew ? '' : id!)
  const { data: cards, isLoading: loadingCards2, refetch: refetchCards } = useDashboardCards(isNew ? '' : id!)
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
    }
  }, [dashboard])

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
  const handleLayoutChange = useCallback(async (layout: Layout[]) => {
    if (!id || !editMode) return

    // Update each card position
    for (const item of layout) {
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

  // Generate layout from cards
  const layout: Layout[] = useMemo(() => {
    return (cards || []).map(card => ({
      i: card.id,
      x: card.col,
      y: card.row,
      w: card.width,
      h: card.height,
      minW: 2,
      minH: 2,
    }))
  }, [cards])

  // Get question for card
  const getQuestionForCard = (card: DashboardCard): Question | undefined => {
    return questions?.find(q => q.id === card.question_id)
  }

  if (loadingDashboard && !isNew) {
    return (
      <Container size="xl" py="lg">
        <Group justify="center" py="xl">
          <Loader size="lg" />
          <Text>Loading dashboard...</Text>
        </Group>
      </Container>
    )
  }

  return (
    <Box style={{ minHeight: '100vh', backgroundColor: '#f9fbfc' }}>
      {/* Header */}
      <Box
        p="md"
        bg="white"
        style={{ borderBottom: '1px solid var(--mantine-color-gray-3)' }}
      >
        <Group justify="space-between">
          <Group gap="md">
            <ThemeIcon size={40} radius="md" variant="light" color="summarize">
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
                  styles={{ input: { fontSize: '1.5rem', fontWeight: 600 } }}
                />
              ) : (
                <Title order={2}>{dashboard?.name || 'New Dashboard'}</Title>
              )}
              <Text size="sm" c="dimmed">
                {cards?.length || 0} cards
              </Text>
            </div>
          </Group>

          <Group gap="sm">
            {!isNew && (
              <Tooltip label="Refresh all cards">
                <ActionIcon variant="subtle" size="lg" onClick={handleRefreshAll}>
                  <IconRefresh size={20} />
                </ActionIcon>
              </Tooltip>
            )}

            {editMode && !isNew && (
              <Button
                variant="light"
                leftSection={<IconPlus size={16} />}
                onClick={openAddCardModal}
              >
                Add Card
              </Button>
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
                <Menu.Item leftSection={<IconClock size={14} />}>
                  Auto-refresh settings
                </Menu.Item>
                <Menu.Item leftSection={<IconFilter size={14} />}>
                  Add filter
                </Menu.Item>
                {!isNew && (
                  <>
                    <Menu.Divider />
                    <Menu.Item leftSection={<IconTrash size={14} />} color="red" onClick={handleDelete}>
                      Delete
                    </Menu.Item>
                  </>
                )}
              </Menu.Dropdown>
            </Menu>
          </Group>
        </Group>
      </Box>

      {/* Dashboard Grid */}
      <Container size="xl" py="lg">
        {cards && cards.length > 0 ? (
          <ResponsiveGridLayout
            className="layout"
            layout={layout}
            cols={18}
            rowHeight={80}
            isDraggable={editMode}
            isResizable={editMode}
            onLayoutChange={handleLayoutChange}
            draggableHandle=".drag-handle"
            margin={[16, 16]}
          >
            {cards.map(card => (
              <div key={card.id}>
                <DashboardCardComponent
                  card={card}
                  question={getQuestionForCard(card)}
                  result={cardResults[card.id]}
                  isLoading={loadingCards[card.id]}
                  editMode={editMode}
                  onRefresh={() => loadCardResult(card)}
                  onRemove={() => handleRemoveCard(card.id)}
                  onNavigate={() => card.question_id && navigate(`/question/${card.question_id}`)}
                />
              </div>
            ))}
          </ResponsiveGridLayout>
        ) : (
          <Paper withBorder radius="md" p="xl" ta="center">
            <Stack align="center" gap="lg">
              <ThemeIcon size={80} radius="xl" variant="light" color="summarize">
                <IconLayoutDashboard size={40} />
              </ThemeIcon>
              <div>
                <Title order={3} mb="xs">
                  {isNew ? 'Create your dashboard' : 'Add your first card'}
                </Title>
                <Text c="dimmed" maw={400} mx="auto">
                  {isNew
                    ? 'Give your dashboard a name and save it, then add cards to visualize your data.'
                    : 'Click "Add Card" to add questions to this dashboard.'}
                </Text>
              </div>
              {isNew ? (
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
              )}
            </Stack>
          </Paper>
        )}
      </Container>

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
              <Tabs.Tab value="text" leftSection={<IconText size={14} />}>
                Text
              </Tabs.Tab>
              <Tabs.Tab value="heading" leftSection={<IconText size={14} />}>
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
}: {
  card: DashboardCard
  question: Question | undefined
  result: QueryResult | undefined
  isLoading: boolean
  editMode: boolean
  onRefresh: () => void
  onRemove: () => void
  onNavigate: () => void
}) {
  const [hovered, setHovered] = useState(false)

  // Text or heading card
  if (card.card_type === 'text' || card.card_type === 'heading') {
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
        <Text
          fw={card.card_type === 'heading' ? 600 : 400}
          size={card.card_type === 'heading' ? 'lg' : 'sm'}
        >
          {card.content || card.title || 'Click to edit...'}
        </Text>
      </Paper>
    )
  }

  // Link card
  if (card.card_type === 'link') {
    return (
      <Paper
        withBorder
        radius="md"
        p="md"
        h="100%"
        style={{ cursor: 'pointer' }}
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
        <Group gap="sm">
          <IconLink size={16} />
          <Text>{card.title || 'Link'}</Text>
        </Group>
      </Paper>
    )
  }

  // Question card
  return (
    <Paper
      withBorder
      radius="md"
      h="100%"
      style={{ display: 'flex', flexDirection: 'column', overflow: 'hidden' }}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
    >
      {/* Card Header */}
      <Group
        justify="space-between"
        p="sm"
        style={{ borderBottom: '1px solid var(--mantine-color-gray-2)' }}
        className={editMode ? 'drag-handle' : ''}
      >
        <Group gap="sm">
          {editMode && <IconGripVertical size={16} color="var(--mantine-color-gray-5)" style={{ cursor: 'grab' }} />}
          <Text
            fw={500}
            size="sm"
            style={{ cursor: 'pointer' }}
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
