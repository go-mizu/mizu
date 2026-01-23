import { useMemo } from 'react'
import {
  Paper, Text, Group, Stack, Button, Select, ActionIcon, Badge, Divider, Box
} from '@mantine/core'
import { IconPlus, IconTrash, IconMathFunction, IconCategory } from '@tabler/icons-react'
import { useColumns, useTables } from '../../api/hooks'
import type { Aggregation } from '../../api/types'

interface GroupByColumn {
  id: string
  column: string
  temporalBucket?: 'minute' | 'hour' | 'day' | 'week' | 'month' | 'quarter' | 'year'
}

interface SummarizeBuilderProps {
  datasourceId: string | null
  sourceTable: string | null
  aggregations: Aggregation[]
  groupBy: GroupByColumn[]
  onAddAggregation: (aggregation: Aggregation) => void
  onRemoveAggregation: (id: string) => void
  onUpdateAggregation: (id: string, updates: Partial<Aggregation>) => void
  onAddGroupBy: (groupBy: GroupByColumn) => void
  onRemoveGroupBy: (id: string) => void
  onUpdateGroupBy: (id: string, updates: Partial<GroupByColumn>) => void
}

const aggregationFunctions: { value: Aggregation['function']; label: string; needsColumn: boolean }[] = [
  { value: 'count', label: 'Count', needsColumn: false },
  { value: 'distinct', label: 'Distinct count', needsColumn: true },
  { value: 'sum', label: 'Sum', needsColumn: true },
  { value: 'avg', label: 'Average', needsColumn: true },
  { value: 'min', label: 'Minimum', needsColumn: true },
  { value: 'max', label: 'Maximum', needsColumn: true },
]

const temporalBuckets: { value: string; label: string }[] = [
  { value: '', label: 'Don\'t bin' },
  { value: 'minute', label: 'Minute' },
  { value: 'hour', label: 'Hour' },
  { value: 'day', label: 'Day' },
  { value: 'week', label: 'Week' },
  { value: 'month', label: 'Month' },
  { value: 'quarter', label: 'Quarter' },
  { value: 'year', label: 'Year' },
]

export default function SummarizeBuilder({
  datasourceId,
  sourceTable,
  aggregations,
  groupBy,
  onAddAggregation,
  onRemoveAggregation,
  onUpdateAggregation,
  onAddGroupBy,
  onRemoveGroupBy,
  onUpdateGroupBy,
}: SummarizeBuilderProps) {
  const { data: tables } = useTables(datasourceId || '')
  const selectedTableId = useMemo(() => {
    const table = tables?.find(t => t.name === sourceTable || t.id === sourceTable)
    return table?.id || ''
  }, [tables, sourceTable])
  const { data: columns } = useColumns(datasourceId || '', selectedTableId)

  const numericColumns = useMemo(() => {
    return (columns || [])
      .filter(c => c.type === 'number')
      .map(c => ({ value: c.name, label: c.display_name || c.name }))
  }, [columns])

  const allColumns = useMemo(() => {
    return (columns || []).map(c => ({
      value: c.name,
      label: c.display_name || c.name,
      type: c.type,
    }))
  }, [columns])

  const isDateColumn = (columnName: string) => {
    const col = columns?.find(c => c.name === columnName)
    return col?.type === 'datetime' || col?.type === 'date'
  }

  const handleAddAggregation = () => {
    onAddAggregation({
      id: Math.random().toString(36).substring(2, 9),
      function: 'count',
    })
  }

  const handleAddGroupBy = () => {
    if (allColumns.length > 0) {
      onAddGroupBy({
        id: Math.random().toString(36).substring(2, 9),
        column: allColumns[0].value,
      })
    }
  }

  if (!sourceTable) {
    return null
  }

  const hasContent = aggregations.length > 0 || groupBy.length > 0

  return (
    <Paper withBorder radius="md" p={0} style={{ overflow: 'hidden' }}>
      <Group justify="space-between" p="sm" bg="summarize.0">
        <Group gap="xs">
          <IconMathFunction size={18} color="var(--mantine-color-summarize-5)" />
          <Text fw={500} size="sm" c="summarize.7">Summarize</Text>
          {hasContent && (
            <Badge size="sm" variant="filled" color="summarize">
              {aggregations.length + groupBy.length}
            </Badge>
          )}
        </Group>
      </Group>

      <Divider />

      <Stack gap="md" p="sm">
        {/* Aggregations */}
        <Box>
          <Group justify="space-between" mb="xs">
            <Text size="sm" fw={500}>Pick a metric</Text>
            <Button
              size="xs"
              variant="light"
              color="summarize"
              leftSection={<IconPlus size={14} />}
              onClick={handleAddAggregation}
            >
              Add
            </Button>
          </Group>

          {aggregations.length === 0 ? (
            <Text size="sm" c="dimmed">Count of rows (default)</Text>
          ) : (
            <Stack gap="xs">
              {aggregations.map(agg => (
                <AggregationRow
                  key={agg.id}
                  aggregation={agg}
                  columns={numericColumns}
                  onUpdate={(updates) => onUpdateAggregation(agg.id, updates)}
                  onRemove={() => onRemoveAggregation(agg.id)}
                />
              ))}
            </Stack>
          )}
        </Box>

        <Divider />

        {/* Group By */}
        <Box>
          <Group justify="space-between" mb="xs">
            <Group gap="xs">
              <IconCategory size={16} />
              <Text size="sm" fw={500}>Group by</Text>
            </Group>
            <Button
              size="xs"
              variant="light"
              color="summarize"
              leftSection={<IconPlus size={14} />}
              onClick={handleAddGroupBy}
            >
              Add
            </Button>
          </Group>

          {groupBy.length === 0 ? (
            <Text size="sm" c="dimmed">No grouping (total)</Text>
          ) : (
            <Stack gap="xs">
              {groupBy.map(gb => (
                <GroupByRow
                  key={gb.id}
                  groupBy={gb}
                  columns={allColumns}
                  isDateColumn={isDateColumn(gb.column)}
                  onUpdate={(updates) => onUpdateGroupBy(gb.id, updates)}
                  onRemove={() => onRemoveGroupBy(gb.id)}
                />
              ))}
            </Stack>
          )}
        </Box>
      </Stack>
    </Paper>
  )
}

function AggregationRow({
  aggregation,
  columns,
  onUpdate,
  onRemove,
}: {
  aggregation: Aggregation
  columns: { value: string; label: string }[]
  onUpdate: (updates: Partial<Aggregation>) => void
  onRemove: () => void
}) {
  const funcInfo = aggregationFunctions.find(f => f.value === aggregation.function)
  const needsColumn = funcInfo?.needsColumn ?? true

  return (
    <Group gap="xs">
      <Select
        size="xs"
        data={aggregationFunctions.map(f => ({ value: f.value, label: f.label }))}
        value={aggregation.function}
        onChange={(value) => onUpdate({ function: value as Aggregation['function'] })}
        style={{ minWidth: 130 }}
      />

      {needsColumn && (
        <>
          <Text size="sm" c="dimmed">of</Text>
          <Select
            size="xs"
            data={columns}
            value={aggregation.column || ''}
            onChange={(value) => onUpdate({ column: value || undefined })}
            placeholder="Select column"
            style={{ minWidth: 150 }}
            searchable
          />
        </>
      )}

      <ActionIcon variant="subtle" color="gray" size="sm" onClick={onRemove}>
        <IconTrash size={14} />
      </ActionIcon>
    </Group>
  )
}

function GroupByRow({
  groupBy,
  columns,
  isDateColumn,
  onUpdate,
  onRemove,
}: {
  groupBy: GroupByColumn
  columns: { value: string; label: string; type: string }[]
  isDateColumn: boolean
  onUpdate: (updates: Partial<GroupByColumn>) => void
  onRemove: () => void
}) {
  return (
    <Group gap="xs">
      <Select
        size="xs"
        data={columns}
        value={groupBy.column}
        onChange={(value) => onUpdate({ column: value || '', temporalBucket: undefined })}
        style={{ minWidth: 150 }}
        searchable
      />

      {isDateColumn && (
        <>
          <Text size="sm" c="dimmed">by</Text>
          <Select
            size="xs"
            data={temporalBuckets}
            value={groupBy.temporalBucket || ''}
            onChange={(value) => onUpdate({ temporalBucket: value as any || undefined })}
            style={{ minWidth: 100 }}
          />
        </>
      )}

      <ActionIcon variant="subtle" color="gray" size="sm" onClick={onRemove}>
        <IconTrash size={14} />
      </ActionIcon>
    </Group>
  )
}

// Compact summarize display
export function SummarizeBadges({
  aggregations,
  groupBy,
}: {
  aggregations: Aggregation[]
  groupBy: GroupByColumn[]
}) {
  if (aggregations.length === 0 && groupBy.length === 0) return null

  return (
    <Group gap="xs">
      {aggregations.map(agg => (
        <Badge key={agg.id} variant="light" color="summarize">
          {agg.function}{agg.column ? ` of ${agg.column}` : ''}
        </Badge>
      ))}
      {groupBy.map(gb => (
        <Badge key={gb.id} variant="outline" color="summarize">
          by {gb.column}{gb.temporalBucket ? ` (${gb.temporalBucket})` : ''}
        </Badge>
      ))}
    </Group>
  )
}
