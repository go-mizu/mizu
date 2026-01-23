import { Group, TextInput, NumberInput, Badge, ActionIcon, Tooltip, Box, Text } from '@mantine/core'
import { DateInput } from '@mantine/dates'
import { IconX, IconVariable } from '@tabler/icons-react'
import type { VariableType } from './useQueryVariables'

interface Variable {
  name: string
  type: VariableType
  start: number
  end: number
}

interface VariableWidgetsProps {
  variables: Variable[]
  values: Record<string, string | number | Date | null>
  onValueChange: (name: string, value: string | number | Date | null) => void
  onClearValue: (name: string) => void
}

// Metabase-style variable colors
const variableColors = {
  bg: '#EFE6F7',
  border: '#7172AD',
  text: '#7172AD',
}

export default function VariableWidgets({
  variables,
  values,
  onValueChange,
  onClearValue,
}: VariableWidgetsProps) {
  if (variables.length === 0) {
    return null
  }

  return (
    <Box
      p="sm"
      style={{
        backgroundColor: '#FAFAFA',
        borderBottom: '1px solid var(--mantine-color-gray-2)',
      }}
    >
      <Group gap="sm" wrap="wrap" align="flex-start">
        <Group gap={4}>
          <IconVariable size={16} color={variableColors.text} />
          <Text size="xs" c="dimmed" fw={500}>
            Variables:
          </Text>
        </Group>

        {variables.map(variable => (
          <VariableWidget
            key={variable.name}
            variable={variable}
            value={values[variable.name]}
            onChange={(value) => onValueChange(variable.name, value)}
            onClear={() => onClearValue(variable.name)}
          />
        ))}
      </Group>
    </Box>
  )
}

interface VariableWidgetProps {
  variable: Variable
  value: string | number | Date | null | undefined
  onChange: (value: string | number | Date | null) => void
  onClear: () => void
}

function VariableWidget({ variable, value, onChange, onClear }: VariableWidgetProps) {
  const hasValue = value != null && value !== ''

  return (
    <Badge
      variant="outline"
      size="lg"
      radius="sm"
      style={{
        backgroundColor: hasValue ? variableColors.bg : 'white',
        borderColor: variableColors.border,
        color: variableColors.text,
        paddingLeft: 0,
        paddingRight: 0,
        height: 'auto',
        textTransform: 'none',
      }}
      leftSection={
        <Box
          px={8}
          py={4}
          style={{
            borderRight: `1px solid ${variableColors.border}`,
          }}
        >
          <Text size="xs" fw={600}>
            {formatVariableName(variable.name)}
          </Text>
        </Box>
      }
      rightSection={
        hasValue ? (
          <Tooltip label="Clear value">
            <ActionIcon
              size="xs"
              variant="subtle"
              color="gray"
              onClick={onClear}
              mr={4}
            >
              <IconX size={12} />
            </ActionIcon>
          </Tooltip>
        ) : null
      }
    >
      <Box px={4}>
        {variable.type === 'date' ? (
          <DateInput
            size="xs"
            placeholder="Select date..."
            value={value instanceof Date ? value : value ? new Date(value as string) : null}
            onChange={(date) => onChange(date)}
            clearable={false}
            variant="unstyled"
            styles={{
              input: {
                padding: '4px 8px',
                minWidth: 120,
                height: 24,
                minHeight: 24,
                fontSize: 12,
              },
            }}
          />
        ) : variable.type === 'number' ? (
          <NumberInput
            size="xs"
            placeholder="Enter number..."
            value={typeof value === 'number' ? value : value ? Number(value) : ''}
            onChange={(val) => onChange(typeof val === 'number' ? val : null)}
            variant="unstyled"
            hideControls
            styles={{
              input: {
                padding: '4px 8px',
                minWidth: 80,
                height: 24,
                minHeight: 24,
                fontSize: 12,
              },
            }}
          />
        ) : (
          <TextInput
            size="xs"
            placeholder="Enter value..."
            value={typeof value === 'string' ? value : value?.toString() || ''}
            onChange={(e) => onChange(e.target.value || null)}
            variant="unstyled"
            styles={{
              input: {
                padding: '4px 8px',
                minWidth: 100,
                height: 24,
                minHeight: 24,
                fontSize: 12,
              },
            }}
          />
        )}
      </Box>
    </Badge>
  )
}

/**
 * Format variable name for display
 * e.g., "start_date" -> "Start Date"
 */
function formatVariableName(name: string): string {
  return name
    .split('_')
    .map(word => word.charAt(0).toUpperCase() + word.slice(1))
    .join(' ')
}
