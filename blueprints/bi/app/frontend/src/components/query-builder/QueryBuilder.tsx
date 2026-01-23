import { useState, useMemo } from 'react'
import {
  Box, Paper, Stack, Group, Text, Button, Divider, ActionIcon,
  Collapse, Badge, SegmentedControl, Select, NumberInput
} from '@mantine/core'
import { IconPlus, IconTrash } from '@tabler/icons-react'
import {
  IconDatabase, IconColumns, IconFilter, IconMathFunction,
  IconArrowsSort, IconPlayerPlay, IconChevronDown,
  IconChevronRight, IconLink
} from '@tabler/icons-react'
import DataSourcePicker from './DataSourcePicker'
import TablePicker from './TablePicker'
import ColumnSelector from './ColumnSelector'
import FilterBuilder from './FilterBuilder'
import SummarizeBuilder from './SummarizeBuilder'
import JoinBuilder from './JoinBuilder'
import { SqlEditor } from '../query-editor'
import { useQueryStore } from '../../stores/queryStore'
import { useTables, useColumns } from '../../api/hooks'

interface QueryBuilderProps {
  onRun: () => void
  isExecuting?: boolean
}

export default function QueryBuilder({ onRun, isExecuting }: QueryBuilderProps) {
  const {
    mode,
    setMode,
    datasourceId,
    setDatasource,
    sourceTable,
    setSourceTable,
    nativeSql,
    setNativeSql,
    columns,
    addColumn,
    removeColumn,
    clearColumns,
    joins,
    addJoin,
    removeJoin,
    updateJoin,
    filters,
    addFilter,
    removeFilter,
    updateFilter,
    clearFilters,
    aggregations,
    addAggregation,
    removeAggregation,
    updateAggregation,
    groupBy,
    addGroupBy,
    removeGroupBy,
    updateGroupBy,
    orderBy,
    addOrderBy,
    removeOrderBy,
    limit,
    setLimit,
  } = useQueryStore()

  // Get tables and columns for the JoinBuilder
  const { data: tables } = useTables(datasourceId || '')
  const selectedTableId = useMemo(() => {
    const foundTable = tables?.find(t => t.name === sourceTable || t.id === sourceTable)
    return foundTable?.id || ''
  }, [tables, sourceTable])
  const { data: sourceColumns } = useColumns(datasourceId || '', selectedTableId)

  // Function to get columns for any table
  const getColumnsForTable = (_tableName: string) => {
    // For now, return source columns for the joined table
    // In a real implementation, we'd fetch columns for each table
    return sourceColumns || []
  }

  const [expandedSections, setExpandedSections] = useState<Record<string, boolean>>({
    source: true,
    joins: false,
    columns: true,
    filter: false,
    summarize: false,
    sort: false,
  })

  const toggleSection = (section: string) => {
    setExpandedSections(prev => ({
      ...prev,
      [section]: !prev[section],
    }))
  }

  const handleToggleColumn = (column: { id: string; table: string; column: string }) => {
    const existing = columns.find(c => c.id === column.id)
    if (existing) {
      removeColumn(column.id)
    } else {
      addColumn(column)
    }
  }

  return (
    <Box>
      {/* Mode selector */}
      <Group justify="space-between" mb="md">
        <SegmentedControl
          value={mode}
          onChange={(value) => setMode(value as 'query' | 'native')}
          data={[
            { value: 'query', label: 'Simple' },
            { value: 'native', label: 'Native query' },
          ]}
          size="sm"
          data-testid="mode-toggle"
        />
      </Group>

      {mode === 'native' ? (
        <Stack gap="md">
          <DataSourcePicker
            value={datasourceId}
            onChange={setDatasource}
          />
          <Paper withBorder radius="md" style={{ overflow: 'hidden', minHeight: 400 }}>
            <SqlEditor
              value={nativeSql}
              onChange={setNativeSql}
              onRun={() => onRun()}
              datasourceId={datasourceId}
              isRunning={isExecuting}
              minHeight={350}
              showSchema={false}
            />
          </Paper>
        </Stack>
      ) : (
        <Stack gap="md">
          {/* Data Source Section */}
          <BuilderSection
            icon={IconDatabase}
            title="Data"
            expanded={expandedSections.source}
            onToggle={() => toggleSection('source')}
            badge={datasourceId && sourceTable ? '1 table' : undefined}
          >
            <Stack gap="md">
              <DataSourcePicker
                value={datasourceId}
                onChange={setDatasource}
              />
              <TablePicker
                datasourceId={datasourceId}
                value={sourceTable}
                onChange={(_tableId, tableName) => setSourceTable(tableName)}
              />
            </Stack>
          </BuilderSection>

          {/* Join Section */}
          {sourceTable && tables && tables.length > 1 && (
            <BuilderSection
              icon={IconLink}
              title="Join"
              expanded={expandedSections.joins}
              onToggle={() => toggleSection('joins')}
              badge={joins.length > 0 ? `${joins.length} join${joins.length > 1 ? 's' : ''}` : undefined}
            >
              <JoinBuilder
                joins={joins.map(j => ({
                  id: j.id,
                  type: j.type,
                  target_table: j.target_table,
                  conditions: j.conditions.map(c => ({
                    source_column: c.source_column,
                    target_column: c.target_column,
                  })),
                }))}
                sourceTable={sourceTable}
                sourceColumns={sourceColumns || []}
                availableTables={tables}
                onAddJoin={(join) => addJoin({
                  ...join,
                  source_table: sourceTable,
                  conditions: join.conditions.map(c => ({
                    ...c,
                    operator: '=' as const,
                  })),
                })}
                onRemoveJoin={removeJoin}
                onUpdateJoin={(id, updates) => updateJoin(id, {
                  ...updates,
                  conditions: updates.conditions?.map(c => ({
                    ...c,
                    operator: '=' as const,
                  })),
                })}
                getColumnsForTable={getColumnsForTable}
              />
            </BuilderSection>
          )}

          {/* Columns Section */}
          <BuilderSection
            icon={IconColumns}
            title="Columns"
            expanded={expandedSections.columns}
            onToggle={() => toggleSection('columns')}
            badge={columns.length > 0 ? `${columns.length} selected` : undefined}
          >
            <ColumnSelector
              datasourceId={datasourceId}
              sourceTable={sourceTable}
              selectedColumns={columns}
              onToggleColumn={handleToggleColumn}
              onClearColumns={clearColumns}
            />
          </BuilderSection>

          {/* Filter Section */}
          <BuilderSection
            icon={IconFilter}
            title="Filter"
            expanded={expandedSections.filter}
            onToggle={() => toggleSection('filter')}
            badge={filters.length > 0 ? `${filters.length} active` : undefined}
            color="filter"
          >
            <FilterBuilder
              datasourceId={datasourceId}
              sourceTable={sourceTable}
              filters={filters}
              onAddFilter={addFilter}
              onRemoveFilter={removeFilter}
              onUpdateFilter={updateFilter}
              onClearFilters={clearFilters}
            />
          </BuilderSection>

          {/* Summarize Section */}
          <BuilderSection
            icon={IconMathFunction}
            title="Summarize"
            expanded={expandedSections.summarize}
            onToggle={() => toggleSection('summarize')}
            badge={aggregations.length > 0 || groupBy.length > 0
              ? `${aggregations.length + groupBy.length} items`
              : undefined}
            color="summarize"
          >
            <SummarizeBuilder
              datasourceId={datasourceId}
              sourceTable={sourceTable}
              aggregations={aggregations}
              groupBy={groupBy}
              onAddAggregation={addAggregation}
              onRemoveAggregation={removeAggregation}
              onUpdateAggregation={updateAggregation}
              onAddGroupBy={addGroupBy}
              onRemoveGroupBy={removeGroupBy}
              onUpdateGroupBy={updateGroupBy}
            />
          </BuilderSection>

          {/* Sort Section */}
          <BuilderSection
            icon={IconArrowsSort}
            title="Sort"
            expanded={expandedSections.sort}
            onToggle={() => toggleSection('sort')}
            badge={orderBy.length > 0 ? `${orderBy.length} sorts` : undefined}
          >
            <SortBuilder
              datasourceId={datasourceId}
              sourceTable={sourceTable}
              orderBy={orderBy}
              onAdd={addOrderBy}
              onRemove={removeOrderBy}
              limit={limit}
              onLimitChange={setLimit}
            />
          </BuilderSection>

          {/* Visualize Button - Success/Summarize Green */}
          <Button
            fullWidth
            size="md"
            leftSection={<IconPlayerPlay size={18} />}
            onClick={onRun}
            loading={isExecuting}
            data-testid="btn-run-query"
            className="btn-visualize"
          >
            Visualize
          </Button>
        </Stack>
      )}
    </Box>
  )
}

// Section color mappings using CSS variables
const sectionColors = {
  brand: {
    bg: 'var(--color-primary-light)',
    text: 'var(--color-primary)',
    pillBg: 'var(--color-primary)',
    pillText: 'var(--color-primary-foreground)',
  },
  filter: {
    bg: 'var(--color-info-light)',
    text: 'var(--color-info)',
    pillBg: 'var(--color-info)',
    pillText: '#ffffff',
  },
  summarize: {
    bg: 'var(--color-success-light)',
    text: 'var(--color-success)',
    pillBg: 'var(--color-success)',
    pillText: 'var(--color-success-foreground)',
  },
}

// Collapsible section wrapper - Metabase Notebook Style
function BuilderSection({
  icon: Icon,
  title,
  expanded,
  onToggle,
  badge,
  color = 'brand',
  children,
}: {
  icon: typeof IconDatabase
  title: string
  expanded: boolean
  onToggle: () => void
  badge?: string
  color?: 'brand' | 'filter' | 'summarize'
  children: React.ReactNode
}) {
  const colors = sectionColors[color] || sectionColors.brand

  return (
    <Paper
      radius="md"
      style={{
        overflow: 'hidden',
        border: `1px solid ${colors.bg}`,
      }}
    >
      <Group
        justify="space-between"
        p="sm"
        style={{
          backgroundColor: colors.bg,
          cursor: 'pointer',
        }}
        onClick={onToggle}
      >
        <Group gap="sm">
          {expanded ? (
            <IconChevronDown size={16} color={colors.text} />
          ) : (
            <IconChevronRight size={16} color={colors.text} />
          )}
          <Icon size={18} color={colors.text} />
          <Text fw={600} size="sm" style={{ color: colors.text }}>
            {title}
          </Text>
          {badge && (
            <Badge
              size="sm"
              radius="xl"
              style={{
                backgroundColor: colors.pillBg,
                color: colors.pillText,
              }}
            >
              {badge}
            </Badge>
          )}
        </Group>
      </Group>
      <Collapse in={expanded}>
        <Box p="sm" pt="sm" style={{ backgroundColor: 'var(--color-background)' }}>
          {children}
        </Box>
      </Collapse>
    </Paper>
  )
}

// Native SQL query builder is now handled by SqlEditor component

// Sort/Order builder
import type { OrderBy } from '../../api/types'

function SortBuilder({
  datasourceId,
  sourceTable,
  orderBy,
  onAdd,
  onRemove,
  limit,
  onLimitChange,
}: {
  datasourceId: string | null
  sourceTable: string | null
  orderBy: OrderBy[]
  onAdd: (orderBy: OrderBy) => void
  onRemove: (column: string) => void
  limit: number | null
  onLimitChange: (limit: number | null) => void
}) {
  const { data: tables } = useTables(datasourceId || '')
  const selectedTableId = useMemo(() => {
    const table = tables?.find(t => t.name === sourceTable || t.id === sourceTable)
    return table?.id || ''
  }, [tables, sourceTable])
  const { data: columns } = useColumns(datasourceId || '', selectedTableId)

  const columnOptions = useMemo(() => {
    return (columns || []).map(c => ({
      value: c.name,
      label: c.display_name || c.name,
    }))
  }, [columns])

  const handleAddSort = () => {
    if (columnOptions.length > 0) {
      const unusedColumn = columnOptions.find(c => !orderBy.some(o => o.column === c.value))
      if (unusedColumn) {
        onAdd({ column: unusedColumn.value, direction: 'asc' })
      }
    }
  }

  if (!sourceTable) {
    return (
      <Text size="sm" c="dimmed">Select a table first</Text>
    )
  }

  return (
    <Stack gap="md">
      <Box>
        <Group justify="space-between" mb="xs">
          <Text size="sm" fw={500}>Sort by</Text>
          <Button
            size="xs"
            variant="light"
            leftSection={<IconPlus size={14} />}
            onClick={handleAddSort}
            disabled={orderBy.length >= columnOptions.length}
          >
            Add sort
          </Button>
        </Group>

        {orderBy.length === 0 ? (
          <Text size="sm" c="dimmed">No sorting applied</Text>
        ) : (
          <Stack gap="xs">
            {orderBy.map((sort) => (
              <Group key={sort.column} gap="xs">
                <Select
                  size="xs"
                  data={columnOptions}
                  value={sort.column}
                  onChange={(value) => {
                    if (value) {
                      onRemove(sort.column)
                      onAdd({ column: value, direction: sort.direction })
                    }
                  }}
                  style={{ flex: 1 }}
                />
                <SegmentedControl
                  size="xs"
                  data={[
                    { value: 'asc', label: 'Asc' },
                    { value: 'desc', label: 'Desc' },
                  ]}
                  value={sort.direction}
                  onChange={(value) => {
                    onRemove(sort.column)
                    onAdd({ column: sort.column, direction: value as 'asc' | 'desc' })
                  }}
                />
                <ActionIcon variant="subtle" color="gray" size="sm" onClick={() => onRemove(sort.column)}>
                  <IconTrash size={14} />
                </ActionIcon>
              </Group>
            ))}
          </Stack>
        )}
      </Box>

      <Divider />

      <Group gap="md">
        <Text size="sm" fw={500}>Row limit</Text>
        <NumberInput
          size="xs"
          value={limit || ''}
          onChange={(value) => onLimitChange(typeof value === 'number' ? value : null)}
          placeholder="No limit"
          min={1}
          style={{ width: 100 }}
        />
      </Group>
    </Stack>
  )
}
