import { useState, useMemo } from 'react'
import { Box, Stack, TextInput, Switch, Text, Checkbox, Group, Anchor } from '@mantine/core'
import {
  IconSearch, IconKey, IconLink, IconHash, IconLetterT, IconCalendar,
  IconToggleLeft, IconCategory, IconCurrencyDollar, IconMail, IconPhoto
} from '@tabler/icons-react'
import type { ResultColumn } from '../../../api/types'
import type { TableSettings, TableColumnConfig } from './types'
import classes from './ColumnPicker.module.css'

interface ColumnPickerProps {
  columns: ResultColumn[]
  settings: TableSettings
  onSettingsChange: (settings: TableSettings) => void
  onClose?: () => void
}

export default function ColumnPicker({
  columns,
  settings,
  onSettingsChange,
  onClose,
}: ColumnPickerProps) {
  const [search, setSearch] = useState('')

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

  // Filtered columns based on search
  const filteredColumns = columnConfigs.filter(({ column }) =>
    column.name.toLowerCase().includes(search.toLowerCase()) ||
    column.display_name?.toLowerCase().includes(search.toLowerCase())
  )

  // Check if all visible
  const allSelected = filteredColumns.every(({ config }) => config.visible)

  // Toggle column visibility
  const toggleColumn = (name: string) => {
    const currentConfigs = settings.columns || columns.map((col, i) => ({
      name: col.name,
      displayName: col.display_name || col.name,
      visible: true,
      position: i,
    }))

    const index = currentConfigs.findIndex(c => c.name === name)
    if (index >= 0) {
      currentConfigs[index] = {
        ...currentConfigs[index],
        visible: !currentConfigs[index].visible,
      }
    }

    onSettingsChange({ ...settings, columns: [...currentConfigs] })
  }

  // Toggle all columns
  const toggleAll = () => {
    const currentConfigs = settings.columns || columns.map((col, i) => ({
      name: col.name,
      displayName: col.display_name || col.name,
      visible: true,
      position: i,
    }))

    const updated = currentConfigs.map(config => ({
      ...config,
      visible: !allSelected,
    }))

    onSettingsChange({ ...settings, columns: updated })
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
    <Box className={classes.container}>
      <Stack gap="md">
        {/* Show row index toggle */}
        <Switch
          label="Show row index"
          checked={settings.showRowIndex || false}
          onChange={(e) => onSettingsChange({ ...settings, showRowIndex: e.currentTarget.checked })}
        />

        {/* Done link */}
        {onClose && (
          <Anchor size="sm" c="brand" onClick={onClose}>
            Done picking columns
          </Anchor>
        )}

        {/* Search */}
        <TextInput
          placeholder="Search for a column..."
          leftSection={<IconSearch size={16} />}
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />

        {/* Table name (if available) */}
        <Text size="sm" fw={700} c="dimmed">
          Columns
        </Text>

        {/* Add all checkbox */}
        <Box className={classes.addAllRow}>
          <Checkbox
            label="Add all"
            checked={allSelected}
            indeterminate={!allSelected && filteredColumns.some(({ config }) => config.visible)}
            onChange={toggleAll}
          />
        </Box>

        {/* Column list */}
        <Stack gap={2}>
          {filteredColumns.map(({ column, config }) => {
            const Icon = getColumnIcon(column)
            return (
              <Group
                key={column.name}
                className={classes.columnRow}
                gap="sm"
                wrap="nowrap"
              >
                <Checkbox
                  checked={config.visible}
                  onChange={() => toggleColumn(column.name)}
                />
                <Icon size={16} color="var(--mantine-color-gray-6)" />
                <Text size="sm" truncate style={{ flex: 1 }}>
                  {config.displayName || column.display_name || column.name}
                </Text>
              </Group>
            )
          })}
        </Stack>

        {filteredColumns.length === 0 && (
          <Text size="sm" c="dimmed" ta="center" py="md">
            No columns match your search
          </Text>
        )}
      </Stack>
    </Box>
  )
}
