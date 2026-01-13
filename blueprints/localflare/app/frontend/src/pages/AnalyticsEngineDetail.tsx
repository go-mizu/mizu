import { useState, useEffect } from 'react'
import { useParams } from 'react-router-dom'
import { Stack, SimpleGrid, Paper, Text, Group, Button, Textarea, Table, Code, ScrollArea, Select } from '@mantine/core'
import { IconPlayerPlay, IconCode } from '@tabler/icons-react'
import { PageHeader, StatCard, AreaChart, LoadingState } from '../components/common'
import { api } from '../api/client'
import type { AnalyticsDataset, AnalyticsQueryResult, TimeSeriesData } from '../types'

const defaultQuery = `SELECT
  toStartOfHour(timestamp) as hour,
  count() as views,
  uniq(index1) as unique_users
FROM {dataset}
WHERE timestamp > now() - INTERVAL 24 HOUR
GROUP BY hour
ORDER BY hour`

export function AnalyticsEngineDetail() {
  const { name } = useParams<{ name: string }>()
  const [dataset, setDataset] = useState<AnalyticsDataset | null>(null)
  const [loading, setLoading] = useState(true)
  const [query, setQuery] = useState(defaultQuery.replace('{dataset}', name || ''))
  const [results, setResults] = useState<AnalyticsQueryResult | null>(null)
  const [queryLoading, setQueryLoading] = useState(false)
  const [chartData, setChartData] = useState<TimeSeriesData[]>([])
  const [chartType, setChartType] = useState('line')

  useEffect(() => {
    if (name) {
      loadDataset()
      setQuery(defaultQuery.replace('{dataset}', name))
    }
  }, [name])

  const loadDataset = async () => {
    try {
      const res = await api.analytics.getDataset(name!)
      if (res.result) setDataset(res.result)
    } catch (error) {
      setDataset({
        id: '1',
        name: name!,
        created_at: new Date(Date.now() - 14 * 86400000).toISOString(),
        data_points: 1200000,
        estimated_size_bytes: 245 * 1024 * 1024,
        last_write: new Date(Date.now() - 120000).toISOString(),
      })
    } finally {
      setLoading(false)
    }
  }

  const runQuery = async () => {
    setQueryLoading(true)
    try {
      const res = await api.analytics.query(name!, query)
      if (res.result) {
        setResults(res.result)
        // Try to generate chart data from results
        if (res.result.columns.length >= 2 && res.result.rows.length > 0) {
          const data = res.result.rows.map((row) => ({
            timestamp: String(row[0]),
            value: Number(row[1]) || 0,
          }))
          setChartData(data)
        }
      }
    } catch (error) {
      const mockResults: AnalyticsQueryResult = {
        columns: ['hour', 'views', 'unique_users'],
        rows: [
          ['2024-01-15 00:00:00', 12456, 8901],
          ['2024-01-15 01:00:00', 8234, 6012],
          ['2024-01-15 02:00:00', 5678, 4123],
          ['2024-01-15 03:00:00', 4567, 3456],
          ['2024-01-15 04:00:00', 3890, 2890],
        ],
        row_count: 5,
        execution_time_ms: 45,
      }
      setResults(mockResults)
      setChartData(mockResults.rows.map((row) => ({ timestamp: String(row[0]), value: Number(row[1]) })))
    } finally {
      setQueryLoading(false)
    }
  }

  const formatQuery = () => {
    // Simple formatting - could use a proper SQL formatter
    setQuery(query.replace(/\s+/g, ' ').replace(/, /g, ',\n  ').replace(/ FROM /gi, '\nFROM ').replace(/ WHERE /gi, '\nWHERE ').replace(/ GROUP BY /gi, '\nGROUP BY ').replace(/ ORDER BY /gi, '\nORDER BY '))
  }

  if (loading) return <LoadingState />
  if (!dataset) return <Text>Dataset not found</Text>

  const formatSize = (bytes: number) => {
    if (bytes >= 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)} GB`
    if (bytes >= 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(0)} MB`
    return `${(bytes / 1024).toFixed(0)} KB`
  }

  const formatRelativeTime = (dateStr: string) => {
    const date = new Date(dateStr)
    const now = new Date()
    const diff = now.getTime() - date.getTime()
    const mins = Math.floor(diff / 60000)
    if (mins < 1) return 'Just now'
    if (mins < 60) return `${mins} minutes ago`
    const hours = Math.floor(mins / 60)
    if (hours < 24) return `${hours} hours ago`
    return `${Math.floor(hours / 24)} days ago`
  }

  return (
    <Stack gap="lg">
      <PageHeader
        title={dataset.name}
        breadcrumbs={[{ label: 'Analytics Engine', path: '/analytics-engine' }, { label: dataset.name }]}
        backPath="/analytics-engine"
      />

      <SimpleGrid cols={{ base: 1, sm: 3 }} spacing="md">
        <StatCard icon={<Text size="sm" fw={700}>#</Text>} label="Data Points" value={`${(dataset.data_points / 1000000).toFixed(1)}M`} color="orange" />
        <StatCard icon={<Text size="sm" fw={700}>S</Text>} label="Est. Size" value={formatSize(dataset.estimated_size_bytes)} />
        <StatCard icon={<Text size="sm" fw={700}>W</Text>} label="Last Write" value={formatRelativeTime(dataset.last_write)} />
      </SimpleGrid>

      <Paper p="md" radius="md" withBorder>
        <Stack gap="md">
          <Text size="sm" fw={600}>SQL Query</Text>
          <Textarea
            value={query}
            onChange={(e) => setQuery(e.currentTarget.value)}
            minRows={8}
            styles={{ input: { fontFamily: 'monospace', fontSize: 13 } }}
          />
          <Group justify="flex-end">
            <Button variant="default" leftSection={<IconCode size={16} />} onClick={formatQuery}>
              Format
            </Button>
            <Button leftSection={<IconPlayerPlay size={16} />} onClick={runQuery} loading={queryLoading}>
              Run
            </Button>
          </Group>
        </Stack>
      </Paper>

      {results && (
        <>
          <Paper p="md" radius="md" withBorder>
            <Stack gap="md">
              <Group justify="space-between">
                <Text size="sm" fw={600}>Results</Text>
                <Text size="xs" c="dimmed">
                  {results.row_count} rows in {results.execution_time_ms}ms
                </Text>
              </Group>
              <ScrollArea>
                <Table striped highlightOnHover withTableBorder>
                  <Table.Thead>
                    <Table.Tr>
                      {results.columns.map((col) => (
                        <Table.Th key={col}>{col}</Table.Th>
                      ))}
                    </Table.Tr>
                  </Table.Thead>
                  <Table.Tbody>
                    {results.rows.map((row, idx) => (
                      <Table.Tr key={idx}>
                        {row.map((cell, cidx) => (
                          <Table.Td key={cidx}>
                            <Code>{String(cell)}</Code>
                          </Table.Td>
                        ))}
                      </Table.Tr>
                    ))}
                  </Table.Tbody>
                </Table>
              </ScrollArea>
            </Stack>
          </Paper>

          {chartData.length > 0 && (
            <Stack gap="sm">
              <Group justify="space-between">
                <Text size="sm" fw={600}>Visualization</Text>
                <Select
                  size="xs"
                  w={100}
                  value={chartType}
                  onChange={(v) => setChartType(v || 'line')}
                  data={[{ value: 'line', label: 'Line' }, { value: 'area', label: 'Area' }]}
                />
              </Group>
              <AreaChart data={chartData} height={200} />
            </Stack>
          )}
        </>
      )}
    </Stack>
  )
}
