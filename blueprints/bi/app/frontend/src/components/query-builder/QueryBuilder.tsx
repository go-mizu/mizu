import { useState } from 'react'
import {
  Box, Paper, Stack, Group, Text, Button, Divider, ActionIcon,
  Collapse, Badge, SegmentedControl
} from '@mantine/core'
import {
  IconDatabase, IconColumns, IconFilter, IconMathFunction,
  IconArrowsSort, IconCode, IconPlayerPlay, IconChevronDown,
  IconChevronRight
} from '@tabler/icons-react'
import DataSourcePicker from './DataSourcePicker'
import TablePicker from './TablePicker'
import ColumnSelector from './ColumnSelector'
import FilterBuilder from './FilterBuilder'
import SummarizeBuilder from './SummarizeBuilder'
import { useQueryStore } from '../../stores/queryStore'

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

  const [expandedSections, setExpandedSections] = useState<Record<string, boolean>>({
    source: true,
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
        <NativeQueryBuilder
          datasourceId={datasourceId}
          onDatasourceChange={setDatasource}
          sql={nativeSql}
          onSqlChange={setNativeSql}
        />
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

          {/* Visualize Button - Metabase Green */}
          <Button
            fullWidth
            size="md"
            leftSection={<IconPlayerPlay size={18} />}
            onClick={onRun}
            loading={isExecuting}
            data-testid="btn-run-query"
            style={{
              backgroundColor: '#84BB4C',
              fontWeight: 700,
            }}
            styles={{
              root: {
                '&:hover': {
                  backgroundColor: '#6FA83D',
                },
              },
            }}
          >
            Visualize
          </Button>
        </Stack>
      )}
    </Box>
  )
}

// Metabase-style section colors
const sectionColors = {
  brand: {
    bg: '#E6F2FF',
    text: '#509EE3',
    pillBg: '#509EE3',
    pillText: '#ffffff',
  },
  filter: {
    bg: '#EFEEF5',
    text: '#7172AD',
    pillBg: '#7172AD',
    pillText: '#ffffff',
  },
  summarize: {
    bg: '#EDF7E4',
    text: '#84BB4C',
    pillBg: '#84BB4C',
    pillText: '#ffffff',
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
        <Box p="sm" pt="sm" style={{ backgroundColor: '#ffffff' }}>
          {children}
        </Box>
      </Collapse>
    </Paper>
  )
}

// Native SQL query builder
function NativeQueryBuilder({
  datasourceId,
  onDatasourceChange,
  sql,
  onSqlChange,
}: {
  datasourceId: string | null
  onDatasourceChange: (id: string | null) => void
  sql: string
  onSqlChange: (sql: string) => void
}) {
  return (
    <Stack gap="md">
      <DataSourcePicker
        value={datasourceId}
        onChange={onDatasourceChange}
      />

      <Paper withBorder radius="md" style={{ overflow: 'hidden' }}>
        <Group justify="space-between" p="sm" bg="gray.0">
          <Group gap="sm">
            <IconCode size={18} />
            <Text fw={500} size="sm">SQL Query</Text>
          </Group>
        </Group>
        <Divider />
        <textarea
          value={sql}
          onChange={(e) => onSqlChange(e.target.value)}
          placeholder="SELECT * FROM table LIMIT 100"
          data-testid="sql-editor"
          style={{
            width: '100%',
            minHeight: 200,
            padding: 12,
            border: 'none',
            fontFamily: 'var(--mantine-font-family-monospace)',
            fontSize: 14,
            lineHeight: 1.5,
            resize: 'vertical',
            outline: 'none',
          }}
        />
      </Paper>
    </Stack>
  )
}

// Sort/Order builder
import { Select, NumberInput } from '@mantine/core'
import { useTables, useColumns } from '../../api/hooks'
import { useMemo } from 'react'
import type { OrderBy } from '../../api/types'
import { IconPlus, IconTrash } from '@tabler/icons-react'

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
  const { data: columns } = useColumns(selectedTableId)

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
