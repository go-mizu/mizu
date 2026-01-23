import { useState, useMemo } from 'react'
import {
  Box, Stack, Tabs, TextInput, Textarea, Switch, Text, Group, ActionIcon,
  Menu, Select, MultiSelect, ColorSwatch, Button, Paper, NumberInput, Divider,
  Tooltip, Badge
} from '@mantine/core'
import {
  IconGripVertical, IconEye, IconEyeOff, IconDots, IconPlus, IconTrash,
  IconKey, IconLink, IconHash, IconLetterT, IconCalendar, IconToggleLeft,
  IconCategory, IconCurrencyDollar, IconMail, IconPhoto,
  IconChevronUp, IconChevronDown
} from '@tabler/icons-react'
import { DragDropContext, Droppable, Draggable, DropResult } from '@hello-pangea/dnd'
import type { ResultColumn } from '../../../api/types'
import type { TableSettings, TableColumnConfig, ConditionalFormattingRule, FormatCondition } from './types'
import classes from './TableSettingsPanel.module.css'

interface TableSettingsPanelProps {
  columns: ResultColumn[]
  settings: TableSettings
  onSettingsChange: (settings: TableSettings) => void
}

const CONDITION_OPERATORS = [
  { value: 'equals', label: 'is equal to' },
  { value: 'not-equals', label: 'is not equal to' },
  { value: 'greater-than', label: 'is greater than' },
  { value: 'less-than', label: 'is less than' },
  { value: 'greater-or-equal', label: 'is greater than or equal to' },
  { value: 'less-or-equal', label: 'is less than or equal to' },
  { value: 'between', label: 'is between' },
  { value: 'is-null', label: 'is empty' },
  { value: 'is-not-null', label: 'is not empty' },
  { value: 'contains', label: 'contains' },
  { value: 'starts-with', label: 'starts with' },
  { value: 'ends-with', label: 'ends with' },
]

const DEFAULT_COLORS = [
  '#ed6e6e', // Red
  '#f2a86f', // Orange
  '#f9d45c', // Yellow
  '#84bb4c', // Green
  '#509ee3', // Blue
  '#7172ad', // Purple
]

export default function TableSettingsPanel({
  columns,
  settings,
  onSettingsChange,
}: TableSettingsPanelProps) {
  const [activeTab, setActiveTab] = useState<string | null>('columns')

  // Get or create column config
  const getColumnConfig = (col: ResultColumn, index: number): TableColumnConfig => {
    const existing = settings.columns?.find(c => c.name === col.name)
    return existing || {
      name: col.name,
      displayName: col.display_name || col.name,
      visible: true,
      position: index,
    }
  }

  // Column configs with full data
  const columnConfigs = useMemo(() => {
    return columns.map((col, index) => ({
      column: col,
      config: getColumnConfig(col, index),
    }))
  }, [columns, settings.columns])

  // Update a column config
  const updateColumnConfig = (name: string, updates: Partial<TableColumnConfig>) => {
    const currentConfigs: TableColumnConfig[] = settings.columns
      ? [...settings.columns]
      : columns.map((col, i) => ({
          name: col.name,
          displayName: col.display_name || col.name,
          visible: true as boolean,
          position: i,
        }))

    const index = currentConfigs.findIndex(c => c.name === name)
    if (index >= 0) {
      currentConfigs[index] = { ...currentConfigs[index], ...updates }
    } else {
      const colIndex = columns.findIndex(c => c.name === name)
      currentConfigs.push({
        name,
        displayName: columns[colIndex]?.display_name || name,
        visible: true as boolean,
        position: colIndex,
        ...updates,
      })
    }

    onSettingsChange({ ...settings, columns: currentConfigs })
  }

  // Handle column reorder via drag-drop
  const handleDragEnd = (result: DropResult) => {
    if (!result.destination) return

    const currentConfigs: TableColumnConfig[] = settings.columns
      ? [...settings.columns]
      : columns.map((col, i) => ({
          name: col.name,
          displayName: col.display_name || col.name,
          visible: true as boolean,
          position: i,
        }))

    const sorted = [...currentConfigs].sort((a, b) => a.position - b.position)
    const [removed] = sorted.splice(result.source.index, 1)
    sorted.splice(result.destination.index, 0, removed)

    // Update positions
    const updated = sorted.map((config, i) => ({ ...config, position: i }))
    onSettingsChange({ ...settings, columns: updated })
  }

  // Toggle column visibility
  const toggleColumnVisibility = (name: string) => {
    const config = columnConfigs.find(c => c.column.name === name)?.config
    updateColumnConfig(name, { visible: !config?.visible })
  }

  // Conditional formatting helpers
  const addRule = () => {
    const newRule: ConditionalFormattingRule = {
      id: Math.random().toString(36).substring(2, 9),
      columns: [],
      style: 'single',
      condition: { operator: 'greater-than', value: 0 },
      color: '#ed6e6e',
      highlightWholeRow: false,
    }
    onSettingsChange({
      ...settings,
      conditionalFormatting: [...(settings.conditionalFormatting || []), newRule],
    })
  }

  const updateRule = (id: string, updates: Partial<ConditionalFormattingRule>) => {
    const rules = settings.conditionalFormatting || []
    const index = rules.findIndex(r => r.id === id)
    if (index >= 0) {
      rules[index] = { ...rules[index], ...updates }
      onSettingsChange({ ...settings, conditionalFormatting: [...rules] })
    }
  }

  const removeRule = (id: string) => {
    onSettingsChange({
      ...settings,
      conditionalFormatting: (settings.conditionalFormatting || []).filter(r => r.id !== id),
    })
  }

  // Move rule up/down
  const moveRule = (id: string, direction: 'up' | 'down') => {
    const rules = [...(settings.conditionalFormatting || [])]
    const index = rules.findIndex(r => r.id === id)
    if (index < 0) return

    const newIndex = direction === 'up' ? index - 1 : index + 1
    if (newIndex < 0 || newIndex >= rules.length) return

    const [removed] = rules.splice(index, 1)
    rules.splice(newIndex, 0, removed)
    onSettingsChange({ ...settings, conditionalFormatting: rules })
  }

  // Column type icon
  const getColumnIcon = (col: ResultColumn) => {
    const semantic = (col as any).semantic
    if (semantic === 'type/PK') return IconKey
    if (semantic === 'type/FK') return IconLink
    if (semantic === 'type/Email') return IconMail
    if (semantic === 'type/URL') return IconLink
    if (semantic === 'type/ImageURL') return IconPhoto
    if (semantic === 'type/Category') return IconCategory
    if (semantic === 'type/Price' || semantic === 'type/Currency') return IconCurrencyDollar

    if (col.type === 'number' || col.type === 'integer' || col.type === 'float') return IconHash
    if (col.type === 'datetime' || col.type === 'date' || col.type === 'timestamp') return IconCalendar
    if (col.type === 'boolean') return IconToggleLeft

    return IconLetterT
  }

  return (
    <Box className={classes.panel}>
      <Tabs value={activeTab} onChange={setActiveTab}>
        <Tabs.List>
          <Tabs.Tab value="columns">Columns</Tabs.Tab>
          <Tabs.Tab value="formatting">
            Conditional Formatting
            {(settings.conditionalFormatting?.length ?? 0) > 0 && (
              <Badge size="xs" ml={4} variant="filled">
                {settings.conditionalFormatting?.length}
              </Badge>
            )}
          </Tabs.Tab>
        </Tabs.List>

        <Tabs.Panel value="columns" pt="md">
          <Stack gap="md">
            {/* Table title */}
            <TextInput
              label="Title"
              placeholder="Table title"
              value={settings.title || ''}
              onChange={(e) => onSettingsChange({ ...settings, title: e.target.value })}
            />

            {/* Description */}
            <Textarea
              label="Description"
              placeholder="Optional description"
              value={settings.description || ''}
              onChange={(e) => onSettingsChange({ ...settings, description: e.target.value })}
              rows={2}
            />

            <Divider />

            {/* Toggles */}
            <Switch
              label="Hide this card if there are no results"
              checked={settings.hideIfNoResults || false}
              onChange={(e) => onSettingsChange({ ...settings, hideIfNoResults: e.currentTarget.checked })}
            />

            <Switch
              label="Paginate results"
              checked={settings.paginateResults || false}
              onChange={(e) => onSettingsChange({ ...settings, paginateResults: e.currentTarget.checked })}
            />

            <Switch
              label="Show row index"
              checked={settings.showRowIndex || false}
              onChange={(e) => onSettingsChange({ ...settings, showRowIndex: e.currentTarget.checked })}
            />

            <Divider />

            {/* Column list */}
            <DragDropContext onDragEnd={handleDragEnd}>
              <Droppable droppableId="columns">
                {(provided) => (
                  <Stack gap={0} {...provided.droppableProps} ref={provided.innerRef}>
                    {columnConfigs
                      .sort((a, b) => a.config.position - b.config.position)
                      .map(({ column, config }, index) => {
                        const Icon = getColumnIcon(column)
                        return (
                          <Draggable key={column.name} draggableId={column.name} index={index}>
                            {(provided, snapshot) => (
                              <Paper
                                ref={provided.innerRef}
                                {...provided.draggableProps}
                                className={`${classes.columnRow} ${snapshot.isDragging ? classes.dragging : ''}`}
                              >
                                <Group gap="sm" wrap="nowrap">
                                  <div {...provided.dragHandleProps} className={classes.dragHandle}>
                                    <IconGripVertical size={16} color="var(--mantine-color-gray-5)" />
                                  </div>
                                  <Icon size={16} color="var(--mantine-color-gray-6)" />
                                  <Text size="sm" style={{ flex: 1 }} truncate>
                                    {config.displayName || column.display_name || column.name}
                                  </Text>
                                  <Menu position="bottom-end" withinPortal>
                                    <Menu.Target>
                                      <ActionIcon variant="subtle" size="sm">
                                        <IconDots size={14} />
                                      </ActionIcon>
                                    </Menu.Target>
                                    <Menu.Dropdown>
                                      <Menu.Label>Column Settings</Menu.Label>
                                      <Menu.Item>Edit display name</Menu.Item>
                                      <Menu.Item>Format values</Menu.Item>
                                      <Menu.Item>Click behavior</Menu.Item>
                                    </Menu.Dropdown>
                                  </Menu>
                                  <Tooltip label={config.visible ? 'Hide column' : 'Show column'}>
                                    <ActionIcon
                                      variant="subtle"
                                      size="sm"
                                      color={config.visible ? 'gray' : 'red'}
                                      onClick={() => toggleColumnVisibility(column.name)}
                                    >
                                      {config.visible ? <IconEye size={16} /> : <IconEyeOff size={16} />}
                                    </ActionIcon>
                                  </Tooltip>
                                </Group>
                              </Paper>
                            )}
                          </Draggable>
                        )
                      })}
                    {provided.placeholder}
                  </Stack>
                )}
              </Droppable>
            </DragDropContext>
          </Stack>
        </Tabs.Panel>

        <Tabs.Panel value="formatting" pt="md">
          <Stack gap="md">
            {(settings.conditionalFormatting || []).map((rule, index) => (
              <Paper key={rule.id} withBorder p="sm" radius="md">
                <Stack gap="sm">
                  {/* Rule header */}
                  <Group justify="space-between">
                    <Text size="sm" fw={500}>Rule {index + 1}</Text>
                    <Group gap={4}>
                      <ActionIcon
                        variant="subtle"
                        size="xs"
                        disabled={index === 0}
                        onClick={() => moveRule(rule.id, 'up')}
                      >
                        <IconChevronUp size={14} />
                      </ActionIcon>
                      <ActionIcon
                        variant="subtle"
                        size="xs"
                        disabled={index === (settings.conditionalFormatting?.length || 0) - 1}
                        onClick={() => moveRule(rule.id, 'down')}
                      >
                        <IconChevronDown size={14} />
                      </ActionIcon>
                      <ActionIcon
                        variant="subtle"
                        size="xs"
                        color="red"
                        onClick={() => removeRule(rule.id)}
                      >
                        <IconTrash size={14} />
                      </ActionIcon>
                    </Group>
                  </Group>

                  {/* Columns selector */}
                  <MultiSelect
                    label="Which columns should be affected?"
                    placeholder="Select columns"
                    data={columns.map(c => ({ value: c.name, label: c.display_name || c.name }))}
                    value={rule.columns}
                    onChange={(value) => updateRule(rule.id, { columns: value })}
                    searchable
                  />

                  {/* Formatting style */}
                  <Box>
                    <Text size="sm" fw={500} mb="xs">Formatting style</Text>
                    <Group gap="md">
                      <label className={classes.radioLabel}>
                        <input
                          type="radio"
                          name={`style-${rule.id}`}
                          checked={rule.style === 'single'}
                          onChange={() => updateRule(rule.id, { style: 'single' })}
                        />
                        <Text size="sm">Single color</Text>
                      </label>
                      <label className={classes.radioLabel}>
                        <input
                          type="radio"
                          name={`style-${rule.id}`}
                          checked={rule.style === 'range'}
                          onChange={() => updateRule(rule.id, { style: 'range' })}
                        />
                        <Text size="sm">Color range</Text>
                      </label>
                    </Group>
                  </Box>

                  {/* Condition (for single color) */}
                  {rule.style === 'single' && (
                    <Box>
                      <Text size="sm" fw={500} mb="xs">When a cell in this column...</Text>
                      <Stack gap="xs">
                        <Select
                          data={CONDITION_OPERATORS}
                          value={rule.condition?.operator || 'greater-than'}
                          onChange={(value) => updateRule(rule.id, {
                            condition: { ...rule.condition!, operator: value as FormatCondition['operator'] }
                          })}
                        />
                        {!['is-null', 'is-not-null'].includes(rule.condition?.operator || '') && (
                          <Group gap="xs">
                            <NumberInput
                              placeholder="Value"
                              value={rule.condition?.value ?? ''}
                              onChange={(value) => updateRule(rule.id, {
                                condition: { ...rule.condition!, value }
                              })}
                              style={{ flex: 1 }}
                            />
                            {rule.condition?.operator === 'between' && (
                              <>
                                <Text size="sm">and</Text>
                                <NumberInput
                                  placeholder="Value"
                                  value={rule.condition?.valueEnd ?? ''}
                                  onChange={(value) => updateRule(rule.id, {
                                    condition: { ...rule.condition!, valueEnd: value }
                                  })}
                                  style={{ flex: 1 }}
                                />
                              </>
                            )}
                          </Group>
                        )}
                      </Stack>
                    </Box>
                  )}

                  {/* Color picker */}
                  <Box>
                    <Text size="sm" fw={500} mb="xs">
                      {rule.style === 'single' ? '...turn its background this color:' : 'Color range:'}
                    </Text>
                    {rule.style === 'single' ? (
                      <Group gap="xs">
                        {DEFAULT_COLORS.map((color) => (
                          <ColorSwatch
                            key={color}
                            color={color}
                            size={24}
                            radius="xl"
                            style={{
                              cursor: 'pointer',
                              border: rule.color === color ? '2px solid #509ee3' : '2px solid transparent'
                            }}
                            onClick={() => updateRule(rule.id, { color })}
                          />
                        ))}
                      </Group>
                    ) : (
                      <Group gap="xs">
                        <ColorSwatch color={rule.colorRange?.colors[0] || '#84bb4c'} size={24} radius="xl" />
                        <Text size="sm">to</Text>
                        <ColorSwatch color={rule.colorRange?.colors[1] || '#ed6e6e'} size={24} radius="xl" />
                      </Group>
                    )}
                  </Box>

                  {/* Highlight whole row */}
                  <Switch
                    label="Highlight the whole row"
                    checked={rule.highlightWholeRow || false}
                    onChange={(e) => updateRule(rule.id, { highlightWholeRow: e.currentTarget.checked })}
                  />
                </Stack>
              </Paper>
            ))}

            <Button
              variant="light"
              leftSection={<IconPlus size={16} />}
              onClick={addRule}
            >
              Add rule
            </Button>
          </Stack>
        </Tabs.Panel>
      </Tabs>
    </Box>
  )
}
