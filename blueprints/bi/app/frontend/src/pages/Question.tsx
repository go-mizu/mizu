import { useEffect, useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import {
  Container, Box, Group, Text, Button, ActionIcon, Menu, Paper, Loader,
  Modal, TextInput, Textarea, Badge, Divider, Tooltip, Stack,
  ThemeIcon
} from '@mantine/core'
import { useDisclosure } from '@mantine/hooks'
import { notifications } from '@mantine/notifications'
import {
  IconPlayerPlay, IconDeviceFloppy, IconDots, IconDownload, IconShare,
  IconTrash, IconCopy, IconBell, IconChartBar,
  IconRefresh, IconLayoutSidebar
} from '@tabler/icons-react'
import { QueryBuilder } from '../components/query-builder'
import Visualization from '../components/visualizations'
import VisualizationPicker, { VisualizationTypeSelect } from '../components/visualizations/VisualizationPicker'
import { useQueryStore } from '../stores/queryStore'
import {
  useQuestion, useCreateQuestion, useUpdateQuestion, useDeleteQuestion,
  useExecuteQuery, useExecuteNativeQuery
} from '../api/hooks'
import type { QueryResult, VisualizationType } from '../api/types'

interface QuestionProps {
  mode?: 'view' | 'edit'
}

export default function Question({ mode: _pageMode = 'view' }: QuestionProps) {
  const { id } = useParams()
  const navigate = useNavigate()
  const isNew = !id || id === 'new'

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
      <Container size="xl" py="lg">
        <Group justify="center" py="xl">
          <Loader size="lg" />
          <Text>Loading question...</Text>
        </Group>
      </Container>
    )
  }

  return (
    <Box style={{ display: 'flex', height: '100vh', overflow: 'hidden' }}>
      {/* Query Builder Sidebar */}
      <Box
        style={{
          width: sidebarOpened ? 400 : 0,
          flexShrink: 0,
          borderRight: sidebarOpened ? '1px solid var(--mantine-color-gray-3)' : 'none',
          backgroundColor: 'white',
          overflow: 'auto',
          transition: 'width 0.2s ease',
        }}
      >
        {sidebarOpened && (
          <Box p="md">
            <QueryBuilder onRun={handleExecute} isExecuting={isExecuting} />
          </Box>
        )}
      </Box>

      {/* Main Content */}
      <Box style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
        {/* Header */}
        <Group
          justify="space-between"
          p="md"
          style={{ borderBottom: '1px solid var(--mantine-color-gray-3)' }}
        >
          <Group gap="md">
            <Tooltip label={sidebarOpened ? 'Hide editor' : 'Show editor'}>
              <ActionIcon variant="subtle" onClick={() => setSidebarOpened(!sidebarOpened)}>
                <IconLayoutSidebar size={20} />
              </ActionIcon>
            </Tooltip>

            <div>
              <Group gap="xs">
                {existingQuestion ? (
                  <Text fw={600} size="lg">{existingQuestion.name}</Text>
                ) : (
                  <Text fw={600} size="lg" c="dimmed">New Question</Text>
                )}
                {result?.cached && (
                  <Badge size="sm" variant="light" color="gray">cached</Badge>
                )}
              </Group>
              {existingQuestion?.description && (
                <Text size="sm" c="dimmed">{existingQuestion.description}</Text>
              )}
            </div>
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
            >
              More charts
            </Button>

            <Divider orientation="vertical" />

            <Button
              leftSection={<IconDeviceFloppy size={16} />}
              onClick={openSaveModal}
              variant="light"
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
                <Menu.Item leftSection={<IconRefresh size={14} />} onClick={handleExecute}>
                  Refresh
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
        </Group>

        {/* Results Area */}
        <Box style={{ flex: 1, overflow: 'auto', padding: 16 }}>
          {isExecuting ? (
            <Paper withBorder p="xl" radius="md" ta="center">
              <Stack align="center" gap="md">
                <Loader size="lg" />
                <Text c="dimmed">Running query...</Text>
              </Stack>
            </Paper>
          ) : error ? (
            <Paper withBorder p="lg" radius="md" bg="red.0">
              <Group gap="md">
                <ThemeIcon color="red" variant="light" size="lg">
                  <IconChartBar size={20} />
                </ThemeIcon>
                <div>
                  <Text fw={500} c="red.7">Query Error</Text>
                  <Text size="sm" c="red.6">{error}</Text>
                </div>
              </Group>
            </Paper>
          ) : result ? (
            <Stack gap="md">
              {/* Result stats */}
              <Group gap="md">
                <Badge variant="light" color="brand">
                  {result.row_count.toLocaleString()} rows
                </Badge>
                <Badge variant="light" color="gray">
                  {result.duration_ms.toFixed(1)} ms
                </Badge>
                <Badge variant="light" color="gray">
                  {result.columns.length} columns
                </Badge>
              </Group>

              {/* Visualization */}
              <Paper withBorder p="md" radius="md">
                <Visualization
                  result={result}
                  visualization={visualization}
                  height={500}
                />
              </Paper>
            </Stack>
          ) : (
            <Paper withBorder p="xl" radius="md" ta="center" bg="gray.0">
              <Stack align="center" gap="md">
                <ThemeIcon size={60} radius="xl" variant="light" color="brand">
                  <IconChartBar size={30} />
                </ThemeIcon>
                <div>
                  <Text fw={500} size="lg">Ready to explore</Text>
                  <Text c="dimmed" size="sm">
                    Select a data source and table, then click "Get Answer" to see results
                  </Text>
                </div>
                <Button
                  size="lg"
                  leftSection={<IconPlayerPlay size={20} />}
                  onClick={handleExecute}
                  disabled={!datasourceId}
                >
                  Get Answer
                </Button>
              </Stack>
            </Paper>
          )}
        </Box>
      </Box>

      {/* Save Modal */}
      <Modal
        opened={saveModalOpened}
        onClose={closeSaveModal}
        title={isNew ? 'Save Question' : 'Update Question'}
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
            placeholder="Optional description..."
            value={questionDescription}
            onChange={(e) => setQuestionDescription(e.target.value)}
            rows={3}
          />
          <Group justify="flex-end" mt="md">
            <Button variant="light" onClick={closeSaveModal}>Cancel</Button>
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
