import { Select, Group, Text, Box, Loader } from '@mantine/core'
import { IconDatabase } from '@tabler/icons-react'
import { useDataSources } from '../../api/hooks'
import type { DataSource } from '../../api/types'

interface DataSourcePickerProps {
  value: string | null
  onChange: (value: string | null) => void
  disabled?: boolean
}

export default function DataSourcePicker({ value, onChange, disabled }: DataSourcePickerProps) {
  const { data: datasources, isLoading } = useDataSources()

  if (isLoading) {
    return (
      <Group gap="xs">
        <Loader size="sm" />
        <Text size="sm" c="dimmed">Loading data sources...</Text>
      </Group>
    )
  }

  const options = (datasources || []).map(ds => ({
    value: ds.id,
    label: ds.name,
    engine: ds.engine,
  }))

  return (
    <Select
      label="Data Source"
      placeholder="Select a data source"
      data={options}
      value={value}
      onChange={onChange}
      disabled={disabled}
      leftSection={<IconDatabase size={16} />}
      searchable
      clearable
      renderOption={({ option }) => (
        <Group gap="sm">
          <IconDatabase size={16} color="var(--mantine-color-gray-6)" />
          <div>
            <Text size="sm">{option.label}</Text>
            <Text size="xs" c="dimmed">{(option as any).engine}</Text>
          </div>
        </Group>
      )}
    />
  )
}

// Engine icons
export function EngineIcon({ engine }: { engine: DataSource['engine'] }) {
  const colors: Record<string, string> = {
    postgres: '#336791',
    mysql: '#4479A1',
    sqlite: '#003B57',
  }
  return (
    <Box
      style={{
        width: 24,
        height: 24,
        borderRadius: 4,
        backgroundColor: colors[engine] || '#888',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        color: 'white',
        fontSize: 10,
        fontWeight: 700,
      }}
    >
      {engine.charAt(0).toUpperCase()}
    </Box>
  )
}
