import { useEffect, useState } from 'react'
import { Container, Title, Text, Card, Group, Stack, Button, Select, Textarea, Table, Paper, Loader, Badge } from '@mantine/core'
import { IconPlayerPlay, IconDeviceFloppy, IconChartBar, IconTable } from '@tabler/icons-react'
import { useParams, useNavigate } from 'react-router-dom'
import { LineChart, Line, BarChart, Bar, PieChart, Pie, Cell, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts'
import { api } from '../api/client'

interface DataSource {
  id: string
  name: string
}

interface Question {
  id: string
  name: string
  description: string
  datasource_id: string
  query_type: string
  query: { sql?: string }
  visualization: { type: string; settings?: Record<string, unknown> }
}

interface QueryResult {
  columns: { name: string; type: string }[]
  rows: Record<string, unknown>[]
  row_count: number
  duration_ms: number
}

const COLORS = ['#509EE3', '#88BF4D', '#A989C5', '#F9CF48', '#EF8C8C', '#98D9D9', '#F2A86F', '#7172AD']

export default function Question() {
  const { id } = useParams()
  const navigate = useNavigate()
  const [question, setQuestion] = useState<Question | null>(null)
  const [datasources, setDatasources] = useState<DataSource[]>([])
  const [selectedDatasource, setSelectedDatasource] = useState<string | null>(null)
  const [sql, setSql] = useState('')
  const [vizType, setVizType] = useState('table')
  const [result, setResult] = useState<QueryResult | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    loadDatasources()
    if (id && id !== 'new') {
      loadQuestion(id)
    }
  }, [id])

  const loadDatasources = async () => {
    try {
      const res = await api.get<DataSource[]>('/datasources')
      setDatasources(res || [])
      if (res?.length > 0 && !selectedDatasource) {
        setSelectedDatasource(res[0].id)
      }
    } catch (error) {
      console.error('Failed to load datasources:', error)
    }
  }

  const loadQuestion = async (questionId: string) => {
    try {
      const q = await api.get<Question>(`/questions/${questionId}`)
      setQuestion(q)
      setSelectedDatasource(q.datasource_id)
      setSql(q.query.sql || '')
      setVizType(q.visualization?.type || 'table')
      // Execute the query
      executeQuery(q.datasource_id, q.query.sql || '')
    } catch (error) {
      console.error('Failed to load question:', error)
    }
  }

  const executeQuery = async (datasourceId: string, query: string) => {
    if (!datasourceId || !query) return

    setLoading(true)
    setError(null)
    try {
      const res = await api.post<QueryResult>('/query/native', {
        datasource_id: datasourceId,
        query: query,
      })
      setResult(res)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Query failed')
      setResult(null)
    } finally {
      setLoading(false)
    }
  }

  const handleRun = () => {
    if (selectedDatasource && sql) {
      executeQuery(selectedDatasource, sql)
    }
  }

  const handleSave = async () => {
    const name = prompt('Question name:')
    if (!name) return

    try {
      const q = await api.post<Question>('/questions', {
        name,
        datasource_id: selectedDatasource,
        query_type: 'native',
        query: { sql },
        visualization: { type: vizType },
      })
      navigate(`/question/${q.id}`)
    } catch (err) {
      alert('Failed to save question')
    }
  }

  const renderVisualization = () => {
    if (!result || result.rows.length === 0) return null

    const data = result.rows as Record<string, unknown>[]

    switch (vizType) {
      case 'line':
        return (
          <ResponsiveContainer width="100%" height={400}>
            <LineChart data={data}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey={result.columns[0]?.name} />
              <YAxis />
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
          <ResponsiveContainer width="100%" height={400}>
            <BarChart data={data}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey={result.columns[0]?.name} />
              <YAxis />
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
          <ResponsiveContainer width="100%" height={400}>
            <PieChart>
              <Pie
                data={data}
                dataKey={result.columns[1]?.name}
                nameKey={result.columns[0]?.name}
                cx="50%"
                cy="50%"
                outerRadius={150}
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
            <Text size="3rem" fw={700} c="blue">
              {typeof value === 'number' ? value.toLocaleString() : String(value)}
            </Text>
            <Text c="dimmed">{result.columns[0]?.name}</Text>
          </Paper>
        )

      default:
        return (
          <Table striped highlightOnHover>
            <Table.Thead>
              <Table.Tr>
                {result.columns.map((col) => (
                  <Table.Th key={col.name}>{col.name}</Table.Th>
                ))}
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>
              {result.rows.slice(0, 100).map((row, i) => (
                <Table.Tr key={i}>
                  {result.columns.map((col) => (
                    <Table.Td key={col.name}>{String(row[col.name] ?? '')}</Table.Td>
                  ))}
                </Table.Tr>
              ))}
            </Table.Tbody>
          </Table>
        )
    }
  }

  return (
    <Container size="xl" py="lg">
      {/* Header */}
      <Group justify="space-between" mb="xl">
        <div>
          <Title order={2}>{question?.name || 'New Question'}</Title>
          <Text c="dimmed">Write SQL to explore your data</Text>
        </div>
        <Group>
          <Button leftSection={<IconPlayerPlay size={16} />} onClick={handleRun} loading={loading}>
            Run
          </Button>
          <Button variant="light" leftSection={<IconDeviceFloppy size={16} />} onClick={handleSave}>
            Save
          </Button>
        </Group>
      </Group>

      {/* Query Editor */}
      <Card withBorder mb="lg">
        <Stack>
          <Group>
            <Select
              label="Data Source"
              placeholder="Select data source"
              data={datasources.map((ds) => ({ value: ds.id, label: ds.name }))}
              value={selectedDatasource}
              onChange={setSelectedDatasource}
              w={200}
            />
            <Select
              label="Visualization"
              data={[
                { value: 'table', label: 'Table' },
                { value: 'line', label: 'Line Chart' },
                { value: 'bar', label: 'Bar Chart' },
                { value: 'pie', label: 'Pie Chart' },
                { value: 'number', label: 'Number' },
              ]}
              value={vizType}
              onChange={(v) => setVizType(v || 'table')}
              w={150}
            />
          </Group>
          <Textarea
            label="SQL Query"
            placeholder="SELECT * FROM table LIMIT 100"
            value={sql}
            onChange={(e) => setSql(e.target.value)}
            minRows={5}
            styles={{
              input: {
                fontFamily: 'monospace',
              },
            }}
          />
        </Stack>
      </Card>

      {/* Results */}
      {loading && (
        <Paper withBorder p="xl" ta="center">
          <Loader size="lg" />
          <Text mt="md">Running query...</Text>
        </Paper>
      )}

      {error && (
        <Paper withBorder p="lg" bg="red.0">
          <Text c="red" fw={500}>Error: {error}</Text>
        </Paper>
      )}

      {result && !loading && (
        <Card withBorder>
          <Group justify="space-between" mb="md">
            <Group>
              <Badge color="blue">{result.row_count} rows</Badge>
              <Badge color="gray">{result.duration_ms.toFixed(1)} ms</Badge>
            </Group>
            <Group>
              <Button
                variant={vizType === 'table' ? 'filled' : 'light'}
                size="xs"
                leftSection={<IconTable size={14} />}
                onClick={() => setVizType('table')}
              >
                Table
              </Button>
              <Button
                variant={vizType !== 'table' ? 'filled' : 'light'}
                size="xs"
                leftSection={<IconChartBar size={14} />}
                onClick={() => setVizType('bar')}
              >
                Chart
              </Button>
            </Group>
          </Group>
          {renderVisualization()}
        </Card>
      )}
    </Container>
  )
}
