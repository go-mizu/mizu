import { useState, useEffect } from 'react'
import { useParams } from 'react-router-dom'
import { Stack, Group, Select, Pagination, Text } from '@mantine/core'
import { PageHeader, DataTable, LoadingState, StatusBadge, type Column } from '../components/common'
import { api } from '../api/client'
import type { AIGatewayLog } from '../types'

export function AIGatewayLogs() {
  const { id } = useParams<{ id: string }>()
  const [logs, setLogs] = useState<AIGatewayLog[]>([])
  const [loading, setLoading] = useState(true)
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [filters, setFilters] = useState({
    provider: '',
    status: '',
    model: '',
    timeRange: '1h',
  })

  const perPage = 20

  useEffect(() => {
    if (id) loadLogs()
  }, [id, page, filters])

  const loadLogs = async () => {
    setLoading(true)
    try {
      const res = await api.aiGateway.getLogs(id!, {
        provider: filters.provider || undefined,
        status: filters.status || undefined,
        model: filters.model || undefined,
        limit: perPage,
        offset: (page - 1) * perPage,
      })
      if (res.result) {
        setLogs(res.result.logs)
        setTotal(res.result.total)
      }
    } catch (error) {
      // Mock data
      const mockLogs: AIGatewayLog[] = Array.from({ length: 20 }, (_, i) => ({
        id: `${i + 1}`,
        gateway_id: id!,
        timestamp: new Date(Date.now() - i * 30000).toISOString(),
        provider: ['OpenAI', 'Anthropic', 'Cohere', 'OpenAI'][i % 4],
        model: ['gpt-4', 'claude-3', 'command', 'gpt-3.5'][i % 4],
        status: i % 7 === 0 ? 429 : i % 5 === 0 ? 'CACHED' : 200,
        latency_ms: i % 5 === 0 ? 12 : Math.floor(Math.random() * 2000) + 200,
        tokens: i % 5 === 0 ? 0 : Math.floor(Math.random() * 2000) + 100,
        cost: i % 5 === 0 ? 0 : Math.random() * 0.1,
        cached: i % 5 === 0,
      }))
      setLogs(mockLogs)
      setTotal(450)
    } finally {
      setLoading(false)
    }
  }

  const columns: Column<AIGatewayLog>[] = [
    { key: 'timestamp', label: 'Time', render: (row) => new Date(row.timestamp).toLocaleString() },
    { key: 'provider', label: 'Provider' },
    { key: 'model', label: 'Model' },
    {
      key: 'status',
      label: 'Status',
      render: (row) =>
        row.cached ? (
          <StatusBadge status="cached" />
        ) : (
          <StatusBadge status={row.status === 200 ? 'success' : 'error'} label={String(row.status)} />
        ),
    },
    { key: 'tokens', label: 'Tokens', render: (row) => row.tokens.toLocaleString() },
    { key: 'cost', label: 'Cost', render: (row) => `$${row.cost.toFixed(4)}` },
  ]

  const totalPages = Math.ceil(total / perPage)

  return (
    <Stack gap="lg">
      <PageHeader
        title="Request Logs"
        breadcrumbs={[
          { label: 'AI Gateway', path: '/ai-gateway' },
          { label: 'prod-gateway', path: `/ai-gateway/${id}` },
          { label: 'Logs' },
        ]}
        backPath={`/ai-gateway/${id}`}
      />

      <Group gap="md">
        <Select
          placeholder="All Providers"
          data={[
            { value: '', label: 'All Providers' },
            { value: 'OpenAI', label: 'OpenAI' },
            { value: 'Anthropic', label: 'Anthropic' },
            { value: 'Cohere', label: 'Cohere' },
          ]}
          value={filters.provider}
          onChange={(v) => setFilters((f) => ({ ...f, provider: v || '' }))}
          clearable
          w={150}
        />
        <Select
          placeholder="All Status"
          data={[
            { value: '', label: 'All Status' },
            { value: '200', label: '200 OK' },
            { value: 'CACHED', label: 'Cached' },
            { value: '429', label: '429 Rate Limited' },
            { value: '500', label: '500 Error' },
          ]}
          value={filters.status}
          onChange={(v) => setFilters((f) => ({ ...f, status: v || '' }))}
          clearable
          w={150}
        />
        <Select
          placeholder="All Models"
          data={[
            { value: '', label: 'All Models' },
            { value: 'gpt-4', label: 'GPT-4' },
            { value: 'gpt-3.5', label: 'GPT-3.5' },
            { value: 'claude-3', label: 'Claude 3' },
            { value: 'command', label: 'Command' },
          ]}
          value={filters.model}
          onChange={(v) => setFilters((f) => ({ ...f, model: v || '' }))}
          clearable
          w={150}
        />
        <Select
          data={[
            { value: '1h', label: 'Last 1 hour' },
            { value: '24h', label: 'Last 24 hours' },
            { value: '7d', label: 'Last 7 days' },
          ]}
          value={filters.timeRange}
          onChange={(v) => setFilters((f) => ({ ...f, timeRange: v || '1h' }))}
          w={150}
        />
      </Group>

      {loading ? (
        <LoadingState />
      ) : (
        <>
          <DataTable
            data={logs}
            columns={columns}
            getRowKey={(row) => row.id}
            searchable={false}
            emptyState={{ title: 'No logs found', description: 'Try adjusting your filters' }}
          />

          <Group justify="space-between">
            <Text size="sm" c="dimmed">
              Showing {(page - 1) * perPage + 1}-{Math.min(page * perPage, total)} of {total}
            </Text>
            <Pagination
              total={totalPages}
              value={page}
              onChange={setPage}
              size="sm"
            />
          </Group>
        </>
      )}
    </Stack>
  )
}
