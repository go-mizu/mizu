import { useEffect, useState } from 'react'
import { Container, Title, Text, Card, Group, Stack, Button, Select, Paper, Loader, Badge, SimpleGrid, Modal, TextInput, Textarea, ActionIcon, Menu } from '@mantine/core'
import { IconPlus, IconDeviceFloppy, IconDotsVertical, IconTrash, IconEdit, IconRefresh } from '@tabler/icons-react'
import { useParams, useNavigate } from 'react-router-dom'
import { LineChart, Line, BarChart, Bar, PieChart, Pie, Cell, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts'
import { api } from '../api/client'

interface Question {
  id: string
  name: string
  visualization?: { type: string }
}

interface DashboardCard {
  id: string
  dashboard_id: string
  question_id?: string
  card_type: string
  title: string
  position_x: number
  position_y: number
  width: number
  height: number
  settings: Record<string, unknown>
}

interface Dashboard {
  id: string
  name: string
  description: string
  collection_id?: string
  cards?: DashboardCard[]
}

interface QueryResult {
  columns: { name: string; type: string }[]
  rows: Record<string, unknown>[]
  row_count: number
  duration_ms: number
}

const COLORS = ['#509EE3', '#88BF4D', '#A989C5', '#F9CF48', '#EF8C8C', '#98D9D9', '#F2A86F', '#7172AD']

export default function Dashboard() {
  const { id } = useParams()
  const navigate = useNavigate()
  const [dashboard, setDashboard] = useState<Dashboard | null>(null)
  const [questions, setQuestions] = useState<Question[]>([])
  const [cardResults, setCardResults] = useState<Record<string, QueryResult>>({})
  const [loading, setLoading] = useState(true)
  const [addCardOpen, setAddCardOpen] = useState(false)
  const [selectedQuestion, setSelectedQuestion] = useState<string | null>(null)
  const [editMode, setEditMode] = useState(false)
  const [nameModalOpen, setNameModalOpen] = useState(false)
  const [dashboardName, setDashboardName] = useState('')
  const [dashboardDesc, setDashboardDesc] = useState('')

  const isNew = id === 'new'

  useEffect(() => {
    loadQuestions()
    if (!isNew && id) {
      loadDashboard(id)
    } else {
      setLoading(false)
      setEditMode(true)
    }
  }, [id])

  const loadQuestions = async () => {
    try {
      const res = await api.get<Question[]>('/questions')
      setQuestions(res || [])
    } catch (error) {
      console.error('Failed to load questions:', error)
    }
  }

  const loadDashboard = async (dashboardId: string) => {
    try {
      const d = await api.get<Dashboard>(`/dashboards/${dashboardId}`)
      setDashboard(d)
      setDashboardName(d.name)
      setDashboardDesc(d.description || '')

      // Load results for each card
      if (d.cards) {
        for (const card of d.cards) {
          if (card.question_id) {
            loadCardResult(card.id, card.question_id)
          }
        }
      }
    } catch (error) {
      console.error('Failed to load dashboard:', error)
    } finally {
      setLoading(false)
    }
  }

  const loadCardResult = async (cardId: string, questionId: string) => {
    try {
      const q = await api.get<{ datasource_id: string; query: { sql?: string } }>(`/questions/${questionId}`)
      if (q.datasource_id && q.query.sql) {
        const result = await api.post<QueryResult>('/query/native', {
          datasource_id: q.datasource_id,
          query: q.query.sql,
        })
        setCardResults(prev => ({ ...prev, [cardId]: result }))
      }
    } catch (error) {
      console.error('Failed to load card result:', error)
    }
  }

  const handleSave = async () => {
    if (isNew) {
      setNameModalOpen(true)
    } else if (dashboard) {
      try {
        await api.put(`/dashboards/${dashboard.id}`, {
          name: dashboardName,
          description: dashboardDesc,
        })
        setEditMode(false)
      } catch (err) {
        alert('Failed to save dashboard')
      }
    }
  }

  const handleCreate = async () => {
    if (!dashboardName) return
    try {
      const d = await api.post<Dashboard>('/dashboards', {
        name: dashboardName,
        description: dashboardDesc,
      })
      navigate(`/dashboard/${d.id}`)
    } catch (err) {
      alert('Failed to create dashboard')
    }
  }

  const handleAddCard = async () => {
    if (!selectedQuestion || !dashboard) return
    try {
      const card = await api.post<DashboardCard>(`/dashboards/${dashboard.id}/cards`, {
        question_id: selectedQuestion,
        card_type: 'question',
        position_x: 0,
        position_y: (dashboard.cards?.length || 0) * 4,
        width: 6,
        height: 4,
      })
      setDashboard(prev => prev ? {
        ...prev,
        cards: [...(prev.cards || []), card],
      } : null)
      loadCardResult(card.id, selectedQuestion)
      setAddCardOpen(false)
      setSelectedQuestion(null)
    } catch (err) {
      alert('Failed to add card')
    }
  }

  const handleDeleteCard = async (cardId: string) => {
    if (!dashboard) return
    try {
      await api.delete(`/dashboards/${dashboard.id}/cards/${cardId}`)
      setDashboard(prev => prev ? {
        ...prev,
        cards: prev.cards?.filter(c => c.id !== cardId),
      } : null)
    } catch (err) {
      alert('Failed to delete card')
    }
  }

  const handleRefreshCard = async (cardId: string, questionId: string) => {
    loadCardResult(cardId, questionId)
  }

  const renderCardVisualization = (card: DashboardCard, result: QueryResult | undefined) => {
    if (!result || result.rows.length === 0) {
      return (
        <Paper p="xl" ta="center" h="100%">
          <Loader size="sm" />
        </Paper>
      )
    }

    const question = questions.find(q => q.id === card.question_id)
    const vizType = question?.visualization?.type || 'table'
    const data = result.rows as Record<string, unknown>[]

    switch (vizType) {
      case 'line':
        return (
          <ResponsiveContainer width="100%" height={250}>
            <LineChart data={data}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey={result.columns[0]?.name} tick={{ fontSize: 12 }} />
              <YAxis tick={{ fontSize: 12 }} />
              <Tooltip />
              <Legend />
              {result.columns.slice(1).map((col, i) => (
                <Line key={col.name} type="monotone" dataKey={col.name} stroke={COLORS[i % COLORS.length]} />
              ))}
            </LineChart>
          </ResponsiveContainer>
        )

      case 'bar':
        return (
          <ResponsiveContainer width="100%" height={250}>
            <BarChart data={data}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey={result.columns[0]?.name} tick={{ fontSize: 12 }} />
              <YAxis tick={{ fontSize: 12 }} />
              <Tooltip />
              <Legend />
              {result.columns.slice(1).map((col, i) => (
                <Bar key={col.name} dataKey={col.name} fill={COLORS[i % COLORS.length]} />
              ))}
            </BarChart>
          </ResponsiveContainer>
        )

      case 'pie':
        return (
          <ResponsiveContainer width="100%" height={250}>
            <PieChart>
              <Pie
                data={data}
                dataKey={result.columns[1]?.name}
                nameKey={result.columns[0]?.name}
                cx="50%"
                cy="50%"
                outerRadius={80}
                label
              >
                {data.map((_, index) => (
                  <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                ))}
              </Pie>
              <Tooltip />
              <Legend />
            </PieChart>
          </ResponsiveContainer>
        )

      case 'number':
        const value = result.rows[0]?.[result.columns[0]?.name]
        return (
          <Paper p="xl" ta="center">
            <Text size="2.5rem" fw={700} c="blue">
              {typeof value === 'number' ? value.toLocaleString() : String(value)}
            </Text>
            <Text c="dimmed" size="sm">{result.columns[0]?.name}</Text>
          </Paper>
        )

      default:
        return (
          <Stack gap="xs" p="sm">
            <Group gap="xs">
              {result.columns.map(col => (
                <Badge key={col.name} size="xs" variant="light">{col.name}</Badge>
              ))}
            </Group>
            <Text size="sm" c="dimmed">
              {result.row_count} rows
            </Text>
          </Stack>
        )
    }
  }

  if (loading) {
    return (
      <Container size="xl" py="lg">
        <Paper withBorder p="xl" ta="center">
          <Loader size="lg" />
          <Text mt="md">Loading dashboard...</Text>
        </Paper>
      </Container>
    )
  }

  return (
    <Container size="xl" py="lg">
      {/* Header */}
      <Group justify="space-between" mb="xl">
        <div>
          {editMode ? (
            <TextInput
              value={dashboardName}
              onChange={(e) => setDashboardName(e.target.value)}
              placeholder="Dashboard name"
              size="lg"
              styles={{ input: { fontSize: '1.5rem', fontWeight: 600 } }}
            />
          ) : (
            <Title order={2}>{dashboard?.name || 'New Dashboard'}</Title>
          )}
          <Text c="dimmed">
            {dashboard?.cards?.length || 0} cards
          </Text>
        </div>
        <Group>
          {!isNew && (
            <Button
              variant="light"
              leftSection={<IconRefresh size={16} />}
              onClick={() => dashboard && loadDashboard(dashboard.id)}
            >
              Refresh
            </Button>
          )}
          {editMode && (
            <Button
              leftSection={<IconPlus size={16} />}
              onClick={() => setAddCardOpen(true)}
              disabled={isNew}
            >
              Add Card
            </Button>
          )}
          <Button
            variant={editMode ? 'filled' : 'light'}
            leftSection={editMode ? <IconDeviceFloppy size={16} /> : <IconEdit size={16} />}
            onClick={editMode ? handleSave : () => setEditMode(true)}
          >
            {editMode ? 'Save' : 'Edit'}
          </Button>
        </Group>
      </Group>

      {/* Cards Grid */}
      {dashboard?.cards && dashboard.cards.length > 0 ? (
        <SimpleGrid cols={{ base: 1, sm: 2 }} spacing="lg">
          {dashboard.cards.map((card) => {
            const question = questions.find(q => q.id === card.question_id)
            return (
              <Card key={card.id} withBorder radius="md" padding="lg">
                <Group justify="space-between" mb="md">
                  <Text fw={500}>{card.title || question?.name || 'Untitled'}</Text>
                  {editMode && (
                    <Menu shadow="md" width={200}>
                      <Menu.Target>
                        <ActionIcon variant="subtle" size="sm">
                          <IconDotsVertical size={16} />
                        </ActionIcon>
                      </Menu.Target>
                      <Menu.Dropdown>
                        {card.question_id && (
                          <Menu.Item
                            leftSection={<IconRefresh size={14} />}
                            onClick={() => handleRefreshCard(card.id, card.question_id!)}
                          >
                            Refresh
                          </Menu.Item>
                        )}
                        <Menu.Item
                          leftSection={<IconTrash size={14} />}
                          color="red"
                          onClick={() => handleDeleteCard(card.id)}
                        >
                          Delete
                        </Menu.Item>
                      </Menu.Dropdown>
                    </Menu>
                  )}
                </Group>
                {renderCardVisualization(card, cardResults[card.id])}
              </Card>
            )
          })}
        </SimpleGrid>
      ) : (
        <Paper withBorder radius="md" p="xl" ta="center">
          <Stack align="center" gap="md">
            <IconPlus size={48} color="var(--mantine-color-gray-5)" />
            <Title order={3}>Add your first card</Title>
            <Text c="dimmed">
              {isNew
                ? 'Save this dashboard first, then add cards'
                : 'Click "Add Card" to add a question to this dashboard'}
            </Text>
            {!isNew && (
              <Button leftSection={<IconPlus size={16} />} onClick={() => setAddCardOpen(true)}>
                Add Card
              </Button>
            )}
          </Stack>
        </Paper>
      )}

      {/* Add Card Modal */}
      <Modal opened={addCardOpen} onClose={() => setAddCardOpen(false)} title="Add Card">
        <Stack>
          <Select
            label="Select a question"
            placeholder="Choose a question"
            data={questions.map(q => ({ value: q.id, label: q.name }))}
            value={selectedQuestion}
            onChange={setSelectedQuestion}
            searchable
          />
          <Group justify="flex-end">
            <Button variant="light" onClick={() => setAddCardOpen(false)}>
              Cancel
            </Button>
            <Button onClick={handleAddCard} disabled={!selectedQuestion}>
              Add
            </Button>
          </Group>
        </Stack>
      </Modal>

      {/* Name Modal for New Dashboard */}
      <Modal opened={nameModalOpen} onClose={() => setNameModalOpen(false)} title="Create Dashboard">
        <Stack>
          <TextInput
            label="Dashboard name"
            placeholder="My Dashboard"
            value={dashboardName}
            onChange={(e) => setDashboardName(e.target.value)}
            required
          />
          <Textarea
            label="Description"
            placeholder="Optional description"
            value={dashboardDesc}
            onChange={(e) => setDashboardDesc(e.target.value)}
          />
          <Group justify="flex-end">
            <Button variant="light" onClick={() => setNameModalOpen(false)}>
              Cancel
            </Button>
            <Button onClick={handleCreate} disabled={!dashboardName}>
              Create
            </Button>
          </Group>
        </Stack>
      </Modal>
    </Container>
  )
}
