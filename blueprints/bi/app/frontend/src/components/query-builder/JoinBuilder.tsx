import { useState } from 'react'
import {
  Paper, Stack, Group, Text, ActionIcon, Select, Button,
  ThemeIcon, Collapse, Badge
} from '@mantine/core'
import { useDisclosure } from '@mantine/hooks'
import {
  IconPlus, IconTrash, IconLink, IconChevronDown, IconChevronUp
} from '@tabler/icons-react'
import type { Column, Table } from '../../api/types'

export interface JoinCondition {
  source_column: string
  target_column: string
  operator?: '=' | '!=' | '>' | '<' | '>=' | '<='
}

export interface JoinConfig {
  id: string
  source_table?: string
  type: 'left' | 'inner' | 'right' | 'full'
  target_table: string
  conditions: JoinCondition[]
}

interface JoinBuilderProps {
  joins: JoinConfig[]
  sourceTable: string
  sourceColumns: Column[]
  availableTables: Table[]
  onAddJoin: (join: JoinConfig) => void
  onRemoveJoin: (id: string) => void
  onUpdateJoin: (id: string, updates: Partial<JoinConfig>) => void
  getColumnsForTable: (tableId: string) => Column[]
}

const JOIN_TYPES = [
  { value: 'left', label: 'Left Join', description: 'All rows from source, matching from target' },
  { value: 'inner', label: 'Inner Join', description: 'Only matching rows from both tables' },
  { value: 'right', label: 'Right Join', description: 'All rows from target, matching from source' },
  { value: 'full', label: 'Full Outer Join', description: 'All rows from both tables' },
]

const generateId = () => Math.random().toString(36).substring(2, 9)

export default function JoinBuilder({
  joins,
  sourceTable,
  sourceColumns,
  availableTables,
  onAddJoin,
  onRemoveJoin,
  onUpdateJoin,
  getColumnsForTable,
}: JoinBuilderProps) {
  const [expanded, { toggle: toggleExpanded }] = useDisclosure(joins.length > 0)

  const handleAddJoin = () => {
    // Find a table that hasn't been joined yet
    const usedTables = [sourceTable, ...joins.map(j => j.target_table)]
    const availableTable = availableTables.find(t => !usedTables.includes(t.name))

    if (!availableTable) return

    const newJoin: JoinConfig = {
      id: generateId(),
      type: 'left',
      target_table: availableTable.name,
      conditions: [],
    }
    onAddJoin(newJoin)
  }

  const handleAddCondition = (joinId: string) => {
    const join = joins.find(j => j.id === joinId)
    if (!join) return

    const targetColumns = getColumnsForTable(join.target_table)

    // Auto-detect potential join columns
    let sourceCol = sourceColumns[0]?.name || ''
    let targetCol = targetColumns[0]?.name || ''

    // Try to find matching column names or FK relationships
    for (const sc of sourceColumns) {
      // Look for foreign key pattern: target_table_id
      if (sc.name === `${join.target_table}_id` || sc.name === `${join.target_table}Id`) {
        sourceCol = sc.name
        targetCol = 'id'
        break
      }
      // Look for matching column names
      const matchingTarget = targetColumns.find(tc => tc.name === sc.name)
      if (matchingTarget) {
        sourceCol = sc.name
        targetCol = matchingTarget.name
        break
      }
    }

    onUpdateJoin(joinId, {
      conditions: [...join.conditions, { source_column: sourceCol, target_column: targetCol }],
    })
  }

  const handleRemoveCondition = (joinId: string, conditionIndex: number) => {
    const join = joins.find(j => j.id === joinId)
    if (!join) return

    onUpdateJoin(joinId, {
      conditions: join.conditions.filter((_, i) => i !== conditionIndex),
    })
  }

  const handleUpdateCondition = (
    joinId: string,
    conditionIndex: number,
    field: 'source_column' | 'target_column',
    value: string
  ) => {
    const join = joins.find(j => j.id === joinId)
    if (!join) return

    const newConditions = [...join.conditions]
    newConditions[conditionIndex] = { ...newConditions[conditionIndex], [field]: value }
    onUpdateJoin(joinId, { conditions: newConditions })
  }

  // Get tables that can still be joined
  const usedTables = [sourceTable, ...joins.map(j => j.target_table)]
  const canAddJoin = availableTables.some(t => !usedTables.includes(t.name))

  return (
    <Paper withBorder radius="sm" p={0}>
      {/* Header */}
      <Group
        px="sm"
        py="xs"
        justify="space-between"
        style={{ cursor: 'pointer', backgroundColor: 'var(--mantine-color-gray-0)' }}
        onClick={toggleExpanded}
      >
        <Group gap="xs">
          <ThemeIcon size="sm" variant="light" color="blue">
            <IconLink size={14} />
          </ThemeIcon>
          <Text size="sm" fw={500}>Join Tables</Text>
          {joins.length > 0 && (
            <Badge size="xs" variant="light" color="blue">
              {joins.length}
            </Badge>
          )}
        </Group>
        <ActionIcon variant="subtle" size="sm">
          {expanded ? <IconChevronUp size={14} /> : <IconChevronDown size={14} />}
        </ActionIcon>
      </Group>

      <Collapse in={expanded}>
        <Stack gap="sm" p="sm">
          {/* Existing joins */}
          {joins.map((join) => (
            <JoinRow
              key={join.id}
              join={join}
              sourceTable={sourceTable}
              sourceColumns={sourceColumns}
              availableTables={availableTables.filter(t => t.name !== sourceTable)}
              getColumnsForTable={getColumnsForTable}
              onUpdate={(updates) => onUpdateJoin(join.id, updates)}
              onRemove={() => onRemoveJoin(join.id)}
              onAddCondition={() => handleAddCondition(join.id)}
              onRemoveCondition={(idx) => handleRemoveCondition(join.id, idx)}
              onUpdateCondition={(idx, field, value) => handleUpdateCondition(join.id, idx, field, value)}
            />
          ))}

          {/* Add join button */}
          {canAddJoin && (
            <Button
              variant="subtle"
              size="xs"
              leftSection={<IconPlus size={14} />}
              onClick={handleAddJoin}
            >
              Add join
            </Button>
          )}

          {/* Empty state */}
          {joins.length === 0 && (
            <Text size="sm" c="dimmed" ta="center" py="md">
              Join other tables to combine data
            </Text>
          )}
        </Stack>
      </Collapse>
    </Paper>
  )
}

// Individual join row component
function JoinRow({
  join,
  sourceTable,
  sourceColumns,
  availableTables,
  getColumnsForTable,
  onUpdate,
  onRemove,
  onAddCondition,
  onRemoveCondition,
  onUpdateCondition,
}: {
  join: JoinConfig
  sourceTable: string
  sourceColumns: Column[]
  availableTables: Table[]
  getColumnsForTable: (tableId: string) => Column[]
  onUpdate: (updates: Partial<JoinConfig>) => void
  onRemove: () => void
  onAddCondition: () => void
  onRemoveCondition: (index: number) => void
  onUpdateCondition: (index: number, field: 'source_column' | 'target_column', value: string) => void
}) {
  const [showDetails] = useState(true)
  const targetColumns = getColumnsForTable(join.target_table)

  return (
    <Paper withBorder radius="sm" p="sm" bg="gray.0">
      <Stack gap="sm">
        {/* Join type and table selection */}
        <Group gap="sm" align="flex-end">
          <Select
            label="Join Type"
            size="xs"
            data={JOIN_TYPES.map(jt => ({ value: jt.value, label: jt.label }))}
            value={join.type}
            onChange={(value) => value && onUpdate({ type: value as JoinConfig['type'] })}
            style={{ flex: 1 }}
          />
          <Select
            label="Target Table"
            size="xs"
            data={availableTables.map(t => ({ value: t.name, label: t.display_name || t.name }))}
            value={join.target_table}
            onChange={(value) => value && onUpdate({ target_table: value, conditions: [] })}
            style={{ flex: 2 }}
          />
          <ActionIcon variant="subtle" color="red" size="sm" onClick={onRemove}>
            <IconTrash size={14} />
          </ActionIcon>
        </Group>

        {/* Join conditions */}
        <Collapse in={showDetails}>
          <Stack gap="xs">
            <Group justify="space-between">
              <Text size="xs" fw={500} c="dimmed">Join Conditions</Text>
              <Button
                variant="subtle"
                size="xs"
                leftSection={<IconPlus size={12} />}
                onClick={onAddCondition}
              >
                Add condition
              </Button>
            </Group>

            {join.conditions.map((condition, index) => (
              <Group key={index} gap="xs" align="flex-end">
                <Select
                  size="xs"
                  placeholder={`${sourceTable} column`}
                  data={sourceColumns.map(c => ({ value: c.name, label: c.display_name || c.name }))}
                  value={condition.source_column}
                  onChange={(value) => value && onUpdateCondition(index, 'source_column', value)}
                  style={{ flex: 1 }}
                />
                <Text size="xs" c="dimmed">=</Text>
                <Select
                  size="xs"
                  placeholder={`${join.target_table} column`}
                  data={targetColumns.map(c => ({ value: c.name, label: c.display_name || c.name }))}
                  value={condition.target_column}
                  onChange={(value) => value && onUpdateCondition(index, 'target_column', value)}
                  style={{ flex: 1 }}
                />
                <ActionIcon
                  variant="subtle"
                  color="gray"
                  size="xs"
                  onClick={() => onRemoveCondition(index)}
                >
                  <IconTrash size={12} />
                </ActionIcon>
              </Group>
            ))}

            {join.conditions.length === 0 && (
              <Text size="xs" c="dimmed" ta="center" py="xs">
                Add at least one join condition
              </Text>
            )}
          </Stack>
        </Collapse>
      </Stack>
    </Paper>
  )
}
