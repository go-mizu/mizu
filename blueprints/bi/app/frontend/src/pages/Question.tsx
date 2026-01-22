import { useEffect, useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import {
  Box, Group, Text, Button, ActionIcon, Menu, Loader,
  Modal, TextInput, Textarea, Badge, Divider, Tooltip, Stack,
  ThemeIcon, rem
} from '@mantine/core'
import { useDisclosure } from '@mantine/hooks'
import { notifications } from '@mantine/notifications'
import {
  IconPlayerPlay, IconDeviceFloppy, IconDots, IconDownload, IconShare,
  IconTrash, IconCopy, IconBell, IconChartBar,
  IconRefresh, IconBookmark, IconBookmarkFilled,
  IconStar, IconStarFilled,
  IconChevronLeft, IconChevronRight, IconDatabase
} from '@tabler/icons-react'
import { QueryBuilder } from '../components/query-builder'
import Visualization from '../components/visualizations'
import VisualizationPicker, { VisualizationTypeSelect } from '../components/visualizations/VisualizationPicker'
import { useQueryStore } from '../stores/queryStore'
import { useBookmarkStore, useBookmarkActions, usePinActions } from '../stores/bookmarkStore'
import {
  useQuestion, useCreateQuestion, useUpdateQuestion, useDeleteQuestion,
  useExecuteQuery, useExecuteNativeQuery
} from '../api/hooks'
import type { QueryResult, VisualizationType } from '../api/types'
import { semanticColors } from '../theme'

// =============================================================================
// STYLES
// =============================================================================

const styles = {
  container: {
    display: 'flex',
    height: '100vh',
    overflow: 'hidden',
    backgroundColor: semanticColors.bgLight,
  },
  sidebar: {
    width: 320,
    flexShrink: 0,
    borderRight: `1px solid ${semanticColors.borderMedium}`,
    backgroundColor: '#ffffff',
    overflow: 'auto',
    transition: 'width 0.2s ease',
    display: 'flex',
    flexDirection: 'column' as const,
  },
  sidebarCollapsed: {
    width: 0,
    borderRight: 'none',
  },
  sidebarHeader: {
    padding: rem(16),
    borderBottom: `1px solid ${semanticColors.borderLight}`,
    backgroundColor: '#ffffff',
  },
  sidebarContent: {
    flex: 1,
    overflow: 'auto',
    padding: rem(16),
  },
  main: {
    flex: 1,
    display: 'flex',
    flexDirection: 'column' as const,
    overflow: 'hidden',
  },
  header: {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'space-between',
    padding: `${rem(12)} ${rem(16)}`,
    backgroundColor: '#ffffff',
    borderBottom: `1px solid ${semanticColors.borderMedium}`,
    minHeight: 56,
  },
  results: {
    flex: 1,
    overflow: 'auto',
    padding: rem(20),
  },
  resultCard: {
    backgroundColor: '#ffffff',
    border: `1px solid ${semanticColors.borderMedium}`,
    borderRadius: rem(8),
    overflow: 'hidden',
  },
  emptyState: {
    backgroundColor: '#ffffff',
    border: `1px solid ${semanticColors.borderMedium}`,
    borderRadius: rem(8),
    padding: rem(48),
    textAlign: 'center' as const,
  },
  statsBar: {
    display: 'flex',
    alignItems: 'center',
    gap: rem(12),
    marginBottom: rem(16),
  },
}

// =============================================================================
// MAIN COMPONENT
// =============================================================================

interface QuestionProps {
  mode?: 'view' | 'edit'
}

export default function Question({ mode: _pageMode = 'view' }: QuestionProps) {
  const { id } = useParams()
  const navigate = useNavigate()
  const isNew = !id || id === 'new'

  // Bookmark store
  const { addRecentItem } = useBookmarkStore()

  // Store state
  const queryStore = useQueryStore()
  const {
    mode: queryMode,
    datasourceId,
    sourceTable,
    nativeSql,
    columns,
    filters,
    aggregations,
    groupBy,
    orderBy,
    limit,
    visualization,
    setVisualizationType,
    isExecuting,
    setIsExecuting,
    reset,
    loadQuestion,
  } = queryStore

  // Local state
  const [result, setResult] = useState<QueryResult | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [saveModalOpened, { open: openSaveModal, close: closeSaveModal }] = useDisclosure(false)
  const [vizPickerOpened, { open: openVizPicker, close: closeVizPicker }] = useDisclosure(false)
  const [sidebarOpened, setSidebarOpened] = useState(true)
  const [questionName, setQuestionName] = useState('')
  const [questionDescription, setQuestionDescription] = useState('')

  // Queries
  const { data: existingQuestion, isLoading: loadingQuestion } = useQuestion(isNew ? '' : id!)
  const createQuestion = useCreateQuestion()
  const updateQuestion = useUpdateQuestion()
  const deleteQuestion = useDeleteQuestion()
  const executeQuery = useExecuteQuery()
  const executeNativeQuery = useExecuteNativeQuery()

  // Load existing question
  useEffect(() => {
    if (existingQuestion) {
      loadQuestion({
        mode: existingQuestion.query_type as 'query' | 'native',
        datasourceId: existingQuestion.datasource_id,
        query: existingQuestion.query,
        visualization: existingQuestion.visualization,
      })
      setQuestionName(existingQuestion.name)
      setQuestionDescription(existingQuestion.description || '')

      // Track in recents
      addRecentItem({
        id: existingQuestion.id,
        type: 'question',
        name: existingQuestion.name,
      })

      // Auto-execute on load
      handleExecute()
    }
  }, [existingQuestion])

  // Reset on new question
  useEffect(() => {
    if (isNew) {
      reset()
      setResult(null)
      setError(null)
      setQuestionName('')
      setQuestionDescription('')
    }
  }, [isNew])

  // Build query from store state
  const buildQuery = () => {
    if (queryMode === 'native') {
      return {
        sql: nativeSql,
      }
    }

    return {
      table: sourceTable || undefined,
      columns: columns.map(c => c.column),
      filters: filters.length > 0 ? filters : undefined,
      aggregations: aggregations.length > 0 ? aggregations : undefined,
      group_by: groupBy.length > 0 ? groupBy.map(g => g.column) : undefined,
      order_by: orderBy.length > 0 ? orderBy : undefined,
      limit: limit || undefined,
    }
  }

  // Execute query
  const handleExecute = async () => {
    if (!datasourceId) {
      notifications.show({
        title: 'No data source',
        message: 'Please select a data source first',
        color: 'yellow',
      })
      return
    }

    if (queryMode === 'native' && !nativeSql.trim()) {
      notifications.show({
        title: 'No query',
        message: 'Please enter a SQL query',
        color: 'yellow',
      })
      return
    }

    if (queryMode === 'query' && !sourceTable) {
      notifications.show({
        title: 'No table',
        message: 'Please select a table first',
        color: 'yellow',
      })
      return
    }

    setIsExecuting(true)
    setError(null)

    try {
      let res: QueryResult
      if (queryMode === 'native') {
        res = await executeNativeQuery.mutateAsync({
          datasource_id: datasourceId,
          query: nativeSql,
        })
      } else {
        res = await executeQuery.mutateAsync({
          datasource_id: datasourceId,
          query: buildQuery(),
        })
      }
      setResult(res)
    } catch (err: any) {
      setError(err.message || 'Query execution failed')
      setResult(null)
    } finally {
      setIsExecuting(false)
    }
  }

  // Save question
  const handleSave = async () => {
    if (!questionName.trim()) {
      notifications.show({
        title: 'Name required',
        message: 'Please enter a name for this question',
        color: 'yellow',
      })
      return
    }

    const questionData = {
      name: questionName,
      description: questionDescription || undefined,
      datasource_id: datasourceId!,
      query_type: queryMode,
      query: buildQuery(),
      visualization,
    }

    try {
      if (isNew) {
        const newQuestion = await createQuestion.mutateAsync(questionData)
        notifications.show({
          title: 'Question saved',
          message: 'Your question has been saved',
          color: 'green',
        })
        closeSaveModal()
        navigate(`/question/${newQuestion.id}`)
      } else {
        await updateQuestion.mutateAsync({ id: id!, ...questionData })
        notifications.show({
          title: 'Question updated',
          message: 'Your changes have been saved',
          color: 'green',
        })
        closeSaveModal()
      }
    } catch (err: any) {
      notifications.show({
        title: 'Error',
        message: err.message || 'Failed to save question',
        color: 'red',
      })
    }
  }

  // Delete question
  const handleDelete = async () => {
    if (!id || isNew) return

    if (!confirm('Are you sure you want to delete this question?')) return

    try {
      await deleteQuestion.mutateAsync(id)
      notifications.show({
        title: 'Question deleted',
        message: 'The question has been deleted',
        color: 'green',
      })
      navigate('/browse')
    } catch (err: any) {
      notifications.show({
        title: 'Error',
        message: err.message || 'Failed to delete question',
        color: 'red',
      })
    }
  }

  // Change visualization type
  const handleVizTypeChange = (type: VisualizationType) => {
    setVisualizationType(type)
    closeVizPicker()
  }

  if (loadingQuestion && !isNew) {
    return (
      <Box style={styles.container}>
        <Box style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
          <Stack align="center" gap="md">
            <Loader size="lg" color="brand" />
            <Text c="dimmed">Loading question...</Text>
          </Stack>
        </Box>
      </Box>
    )
  }

  return (
    <Box style={styles.container}>
      {/* Query Builder Sidebar */}
      <Box
        style={{
          ...styles.sidebar,
          ...(sidebarOpened ? {} : styles.sidebarCollapsed),
        }}
      >
        {sidebarOpened && (
          <>
            <Box style={styles.sidebarHeader}>
              <Group justify="space-between" mb="md">
                <Group gap="xs">
                  <IconDatabase size={18} color={semanticColors.brand} />
                  <Text fw={600} size="sm" style={{ color: semanticColors.textPrimary }}>
                    Query Builder
                  </Text>
                </Group>
                <ActionIcon
                  variant="subtle"
                  color="gray"
                  onClick={() => setSidebarOpened(false)}
                >
                  <IconChevronLeft size={16} />
                </ActionIcon>
              </Group>
              <Button
                fullWidth
                leftSection={<IconPlayerPlay size={16} />}
                onClick={handleExecute}
                loading={isExecuting}
              >
                Get Answer
              </Button>
            </Box>
            <Box style={styles.sidebarContent}>
              <QueryBuilder onRun={handleExecute} isExecuting={isExecuting} />
            </Box>
          </>
        )}
      </Box>

      {/* Main Content */}
      <Box style={styles.main}>
        {/* Header */}
        <Box style={styles.header}>
          <Group gap="md">
            {!sidebarOpened && (
              <Tooltip label="Show query builder">
                <ActionIcon variant="subtle" color="gray" onClick={() => setSidebarOpened(true)}>
                  <IconChevronRight size={18} />
                </ActionIcon>
              </Tooltip>
            )}

            <Box>
              <Group gap="xs" mb={2}>
                {existingQuestion ? (
                  <Text fw={600} size="lg" style={{ color: semanticColors.textPrimary }}>
                    {existingQuestion.name}
                  </Text>
                ) : (
                  <Text fw={600} size="lg" style={{ color: semanticColors.textTertiary }}>
                    New Question
                  </Text>
                )}
                {result?.cached && (
                  <Badge size="xs" variant="light" color="gray">cached</Badge>
                )}
              </Group>
              {existingQuestion?.description && (
                <Text size="xs" c="dimmed">{existingQuestion.description}</Text>
              )}
            </Box>
          </Group>

          <Group gap="sm">
            {/* Visualization type quick selector */}
            <VisualizationTypeSelect
              value={visualization.type}
              onChange={handleVizTypeChange}
            />

            <Button
              variant="light"
              size="sm"
              onClick={openVizPicker}
              color="gray"
            >
              Settings
            </Button>

            <Divider orientation="vertical" />

            {!isNew && existingQuestion && (
              <QuestionActions
                questionId={existingQuestion.id}
                questionName={existingQuestion.name}
              />
            )}

            <Button
              leftSection={<IconDeviceFloppy size={16} />}
              onClick={openSaveModal}
              variant="light"
            >
              Save
            </Button>

            <Menu position="bottom-end">
              <Menu.Target>
                <ActionIcon variant="subtle" size="lg" color="gray">
                  <IconDots size={20} />
                </ActionIcon>
              </Menu.Target>
              <Menu.Dropdown>
                <Menu.Item leftSection={<IconRefresh size={14} />} onClick={handleExecute}>
                  Refresh results
                </Menu.Item>
                <Menu.Item leftSection={<IconDownload size={14} />}>
                  Download results
                </Menu.Item>
                <Menu.Item leftSection={<IconCopy size={14} />}>
                  Duplicate
                </Menu.Item>
                <Menu.Item leftSection={<IconShare size={14} />}>
                  Share
                </Menu.Item>
                <Menu.Item leftSection={<IconBell size={14} />}>
                  Create alert
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
        </Box>

        {/* Results Area */}
        <Box style={styles.results}>
          {isExecuting ? (
            <Box style={styles.emptyState}>
              <Stack align="center" gap="md">
                <Loader size="lg" color="brand" />
                <Text c="dimmed">Running query...</Text>
              </Stack>
            </Box>
          ) : error ? (
            <Box style={{
              ...styles.emptyState,
              backgroundColor: semanticColors.errorLight,
              borderColor: semanticColors.error,
            }}>
              <Group gap="md" justify="center">
                <ThemeIcon color="red" variant="light" size="lg">
                  <IconChartBar size={20} />
                </ThemeIcon>
                <Box style={{ textAlign: 'left' }}>
                  <Text fw={600} style={{ color: semanticColors.error }}>Query Error</Text>
                  <Text size="sm" style={{ color: semanticColors.error }}>{error}</Text>
                </Box>
              </Group>
            </Box>
          ) : result ? (
            <Stack gap="md">
              {/* Result stats */}
              <Box style={styles.statsBar}>
                <Badge variant="light" color="brand" radius="sm">
                  {result.row_count.toLocaleString()} rows
                </Badge>
                <Badge variant="light" color="gray" radius="sm">
                  {result.duration_ms.toFixed(1)} ms
                </Badge>
                <Badge variant="light" color="gray" radius="sm">
                  {result.columns.length} columns
                </Badge>
              </Box>

              {/* Visualization */}
              <Box style={styles.resultCard}>
                <Box p="md">
                  <Visualization
                    result={result}
                    visualization={visualization}
                    height={500}
                  />
                </Box>
              </Box>
            </Stack>
          ) : (
            <Box style={styles.emptyState}>
              <Stack align="center" gap="lg">
                <ThemeIcon size={64} radius="xl" variant="light" color="brand">
                  <IconChartBar size={32} />
                </ThemeIcon>
                <Box>
                  <Text fw={600} size="lg" mb={4} style={{ color: semanticColors.textPrimary }}>
                    Ready to explore
                  </Text>
                  <Text c="dimmed" size="sm" maw={300} mx="auto">
                    Select a data source and table in the query builder, then click "Get Answer" to see results
                  </Text>
                </Box>
                <Button
                  size="lg"
                  leftSection={<IconPlayerPlay size={20} />}
                  onClick={handleExecute}
                  disabled={!datasourceId}
                >
                  Get Answer
                </Button>
              </Stack>
            </Box>
          )}
        </Box>
      </Box>

      {/* Save Modal */}
      <Modal
        opened={saveModalOpened}
        onClose={closeSaveModal}
        title={isNew ? 'Save Question' : 'Update Question'}
        size="md"
      >
        <Stack gap="md">
          <TextInput
            label="Name"
            placeholder="What does this question answer?"
            value={questionName}
            onChange={(e) => setQuestionName(e.target.value)}
            required
          />
          <Textarea
            label="Description"
            placeholder="Add a description to help others understand this question"
            value={questionDescription}
            onChange={(e) => setQuestionDescription(e.target.value)}
            rows={3}
          />
          <Group justify="flex-end" mt="md">
            <Button variant="subtle" onClick={closeSaveModal} color="gray">
              Cancel
            </Button>
            <Button onClick={handleSave} loading={createQuestion.isPending || updateQuestion.isPending}>
              {isNew ? 'Save' : 'Update'}
            </Button>
          </Group>
        </Stack>
      </Modal>

      {/* Visualization Picker Modal */}
      <VisualizationPicker
        opened={vizPickerOpened}
        onClose={closeVizPicker}
        value={visualization.type}
        onChange={handleVizTypeChange}
      />
    </Box>
  )
}

// =============================================================================
// QUESTION ACTIONS COMPONENT
// =============================================================================

function QuestionActions({
  questionId,
  questionName,
}: {
  questionId: string
  questionName: string
}) {
  const { bookmarked, toggle: toggleBookmark } = useBookmarkActions(questionId, 'question', questionName)
  const { pinned, toggle: togglePin } = usePinActions(questionId, 'question')

  return (
    <>
      <Tooltip label={bookmarked ? 'Remove bookmark' : 'Bookmark'}>
        <ActionIcon
          variant="subtle"
          color={bookmarked ? 'yellow' : 'gray'}
          onClick={toggleBookmark}
        >
          {bookmarked ? <IconBookmarkFilled size={18} /> : <IconBookmark size={18} />}
        </ActionIcon>
      </Tooltip>
      <Tooltip label={pinned ? 'Unpin from home' : 'Pin to home'}>
        <ActionIcon
          variant="subtle"
          color={pinned ? 'yellow' : 'gray'}
          onClick={togglePin}
        >
          {pinned ? <IconStarFilled size={18} /> : <IconStar size={18} />}
        </ActionIcon>
      </Tooltip>
    </>
  )
}
