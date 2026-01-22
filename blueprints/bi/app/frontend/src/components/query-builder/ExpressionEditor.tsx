import { useState, useMemo, useRef, useEffect } from 'react'
import {
  Box, Paper, Stack, Group, Text, Button, ActionIcon, Modal,
  TextInput, Tooltip, Tabs, ScrollArea, UnstyledButton, Badge,
  Divider, Code, Kbd
} from '@mantine/core'
import { useDisclosure } from '@mantine/hooks'
import {
  IconPlus, IconX, IconFunction, IconHelp, IconSparkles,
  IconBraces, IconMath, IconCalendar, IconLetterCase,
  IconEqual, IconArrowRight
} from '@tabler/icons-react'
import type { Column } from '../../api/types'

interface ExpressionEditorProps {
  value: string
  onChange: (value: string) => void
  columns?: Column[]
  placeholder?: string
  label?: string
  mode?: 'filter' | 'column' | 'aggregation'
}

// Available functions categorized
const FUNCTIONS = {
  math: [
    { name: 'abs', args: ['number'], desc: 'Absolute value' },
    { name: 'ceil', args: ['number'], desc: 'Round up to nearest integer' },
    { name: 'floor', args: ['number'], desc: 'Round down to nearest integer' },
    { name: 'round', args: ['number'], desc: 'Round to nearest integer' },
    { name: 'sqrt', args: ['number'], desc: 'Square root' },
    { name: 'power', args: ['base', 'exponent'], desc: 'Raise to power' },
    { name: 'log', args: ['number'], desc: 'Natural logarithm' },
    { name: 'exp', args: ['number'], desc: 'e raised to power' },
  ],
  string: [
    { name: 'concat', args: ['string1', 'string2', '...'], desc: 'Join strings together' },
    { name: 'substring', args: ['string', 'start', 'length'], desc: 'Extract part of string' },
    { name: 'trim', args: ['string'], desc: 'Remove leading/trailing whitespace' },
    { name: 'upper', args: ['string'], desc: 'Convert to uppercase' },
    { name: 'lower', args: ['string'], desc: 'Convert to lowercase' },
    { name: 'replace', args: ['string', 'find', 'replace'], desc: 'Replace text' },
    { name: 'length', args: ['string'], desc: 'String length' },
    { name: 'contains', args: ['string', 'substring'], desc: 'Check if contains text' },
    { name: 'startsWith', args: ['string', 'prefix'], desc: 'Check if starts with' },
    { name: 'endsWith', args: ['string', 'suffix'], desc: 'Check if ends with' },
    { name: 'regexextract', args: ['string', 'pattern'], desc: 'Extract using regex' },
  ],
  date: [
    { name: 'now', args: [], desc: 'Current date and time' },
    { name: 'today', args: [], desc: 'Current date' },
    { name: 'getYear', args: ['date'], desc: 'Extract year' },
    { name: 'getMonth', args: ['date'], desc: 'Extract month (1-12)' },
    { name: 'getDay', args: ['date'], desc: 'Extract day of month' },
    { name: 'getHour', args: ['datetime'], desc: 'Extract hour (0-23)' },
    { name: 'getMinute', args: ['datetime'], desc: 'Extract minute' },
    { name: 'getSecond', args: ['datetime'], desc: 'Extract second' },
    { name: 'getQuarter', args: ['date'], desc: 'Extract quarter (1-4)' },
    { name: 'getDayOfWeek', args: ['date'], desc: 'Day of week (1=Mon, 7=Sun)' },
    { name: 'datetimeAdd', args: ['datetime', 'amount', 'unit'], desc: 'Add time to date' },
    { name: 'datetimeDiff', args: ['datetime1', 'datetime2', 'unit'], desc: 'Difference between dates' },
    { name: 'convertTimezone', args: ['datetime', 'from_tz', 'to_tz'], desc: 'Convert timezone' },
  ],
  logic: [
    { name: 'case', args: ['condition', 'true_val', 'false_val'], desc: 'If/then/else logic' },
    { name: 'coalesce', args: ['value1', 'value2', '...'], desc: 'First non-null value' },
    { name: 'isNull', args: ['value'], desc: 'Check if null' },
    { name: 'isEmpty', args: ['value'], desc: 'Check if empty string' },
    { name: 'between', args: ['value', 'min', 'max'], desc: 'Check if between values' },
  ],
  aggregation: [
    { name: 'count', args: [], desc: 'Count rows' },
    { name: 'countIf', args: ['condition'], desc: 'Count rows matching condition' },
    { name: 'sum', args: ['column'], desc: 'Sum of values' },
    { name: 'sumIf', args: ['column', 'condition'], desc: 'Conditional sum' },
    { name: 'avg', args: ['column'], desc: 'Average value' },
    { name: 'min', args: ['column'], desc: 'Minimum value' },
    { name: 'max', args: ['column'], desc: 'Maximum value' },
    { name: 'distinct', args: ['column'], desc: 'Distinct count' },
    { name: 'standardDeviation', args: ['column'], desc: 'Standard deviation' },
    { name: 'variance', args: ['column'], desc: 'Statistical variance' },
    { name: 'cumSum', args: ['column'], desc: 'Cumulative sum' },
    { name: 'percentile', args: ['column', 'percentile'], desc: 'Percentile value' },
  ],
}

export default function ExpressionEditor({
  value,
  onChange,
  columns = [],
  placeholder = 'Enter expression...',
  label,
  mode = 'column',
}: ExpressionEditorProps) {
  const [focused, setFocused] = useState(false)
  const [helpOpen, { open: openHelp, close: closeHelp }] = useDisclosure(false)
  const inputRef = useRef<HTMLTextAreaElement>(null)

  // Insert text at cursor position
  const insertAtCursor = (text: string) => {
    if (inputRef.current) {
      const start = inputRef.current.selectionStart
      const end = inputRef.current.selectionEnd
      const newValue = value.slice(0, start) + text + value.slice(end)
      onChange(newValue)
      // Set cursor after inserted text
      setTimeout(() => {
        if (inputRef.current) {
          inputRef.current.selectionStart = start + text.length
          inputRef.current.selectionEnd = start + text.length
          inputRef.current.focus()
        }
      }, 0)
    } else {
      onChange(value + text)
    }
  }

  // Insert column reference
  const insertColumn = (columnName: string) => {
    insertAtCursor(`[${columnName}]`)
  }

  // Insert function template
  const insertFunction = (fn: { name: string; args: string[] }) => {
    const argsStr = fn.args.length > 0 ? fn.args.join(', ') : ''
    insertAtCursor(`${fn.name}(${argsStr})`)
  }

  // Auto-format expression
  const formatExpression = () => {
    // Basic formatting: ensure spaces around operators
    let formatted = value
      .replace(/([+\-*/=<>!&|])/g, ' $1 ')
      .replace(/\s+/g, ' ')
      .trim()
    onChange(formatted)
  }

  return (
    <Box>
      {label && (
        <Text size="sm" fw={500} mb={4}>{label}</Text>
      )}
      <Paper
        withBorder
        radius="sm"
        style={{
          borderColor: focused ? 'var(--mantine-color-brand-5)' : undefined,
          transition: 'border-color 0.15s ease',
        }}
      >
        {/* Toolbar */}
        <Group gap="xs" px="sm" py={6} bg="gray.0" style={{ borderBottom: '1px solid var(--mantine-color-gray-2)' }}>
          <Tooltip label="Insert column">
            <ActionIcon
              variant="subtle"
              size="sm"
              onClick={() => {
                // Show column picker - simplified for now
                if (columns.length > 0) {
                  insertColumn(columns[0].name)
                }
              }}
            >
              <IconBraces size={14} />
            </ActionIcon>
          </Tooltip>
          <Tooltip label="Function browser">
            <ActionIcon variant="subtle" size="sm" onClick={openHelp}>
              <IconFunction size={14} />
            </ActionIcon>
          </Tooltip>
          <Tooltip label="Auto-format">
            <ActionIcon variant="subtle" size="sm" onClick={formatExpression}>
              <IconSparkles size={14} />
            </ActionIcon>
          </Tooltip>
          <Box style={{ flex: 1 }} />
          <Tooltip label="Expression help">
            <ActionIcon variant="subtle" size="sm" color="gray" onClick={openHelp}>
              <IconHelp size={14} />
            </ActionIcon>
          </Tooltip>
        </Group>

        {/* Editor */}
        <textarea
          ref={inputRef}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          placeholder={placeholder}
          onFocus={() => setFocused(true)}
          onBlur={() => setFocused(false)}
          style={{
            width: '100%',
            minHeight: 80,
            padding: 12,
            border: 'none',
            fontFamily: 'var(--mantine-font-family-monospace)',
            fontSize: 13,
            lineHeight: 1.5,
            resize: 'vertical',
            outline: 'none',
            backgroundColor: 'transparent',
          }}
        />

        {/* Column quick insert */}
        {columns.length > 0 && (
          <Box px="sm" py="xs" bg="gray.0" style={{ borderTop: '1px solid var(--mantine-color-gray-2)' }}>
            <Group gap={6}>
              <Text size="xs" c="dimmed">Columns:</Text>
              {columns.slice(0, 5).map((col) => (
                <Badge
                  key={col.id}
                  size="xs"
                  variant="light"
                  style={{ cursor: 'pointer' }}
                  onClick={() => insertColumn(col.name)}
                >
                  {col.display_name || col.name}
                </Badge>
              ))}
              {columns.length > 5 && (
                <Text size="xs" c="dimmed">+{columns.length - 5} more</Text>
              )}
            </Group>
          </Box>
        )}
      </Paper>

      {/* Function Browser Modal */}
      <FunctionBrowserModal
        opened={helpOpen}
        onClose={closeHelp}
        onInsert={insertFunction}
        mode={mode}
      />
    </Box>
  )
}

// Function Browser Modal
function FunctionBrowserModal({
  opened,
  onClose,
  onInsert,
  mode,
}: {
  opened: boolean
  onClose: () => void
  onInsert: (fn: { name: string; args: string[] }) => void
  mode: 'filter' | 'column' | 'aggregation'
}) {
  const [search, setSearch] = useState('')
  const [selectedCategory, setSelectedCategory] = useState<string | null>('math')

  const categories = [
    { key: 'math', label: 'Math', icon: IconMath },
    { key: 'string', label: 'String', icon: IconLetterCase },
    { key: 'date', label: 'Date', icon: IconCalendar },
    { key: 'logic', label: 'Logic', icon: IconEqual },
    { key: 'aggregation', label: 'Aggregation', icon: IconFunction },
  ]

  const filteredFunctions = useMemo(() => {
    const fns = selectedCategory ? FUNCTIONS[selectedCategory as keyof typeof FUNCTIONS] || [] : []
    if (!search) return fns
    const searchLower = search.toLowerCase()
    return fns.filter(fn =>
      fn.name.toLowerCase().includes(searchLower) ||
      fn.desc.toLowerCase().includes(searchLower)
    )
  }, [selectedCategory, search])

  const handleInsert = (fn: { name: string; args: string[] }) => {
    onInsert(fn)
    onClose()
  }

  return (
    <Modal
      opened={opened}
      onClose={onClose}
      title="Function Browser"
      size="lg"
    >
      <Stack gap="md">
        <TextInput
          placeholder="Search functions..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          leftSection={<IconFunction size={16} />}
        />

        <Group align="flex-start" gap="md">
          {/* Category tabs */}
          <Paper withBorder radius="sm" p="xs" style={{ width: 140 }}>
            <Stack gap={4}>
              {categories.map((cat) => (
                <UnstyledButton
                  key={cat.key}
                  onClick={() => setSelectedCategory(cat.key)}
                  px="sm"
                  py={6}
                  style={{
                    borderRadius: 4,
                    backgroundColor: selectedCategory === cat.key ? 'var(--mantine-color-brand-0)' : 'transparent',
                    color: selectedCategory === cat.key ? 'var(--mantine-color-brand-7)' : undefined,
                  }}
                >
                  <Group gap="xs">
                    <cat.icon size={14} />
                    <Text size="sm">{cat.label}</Text>
                  </Group>
                </UnstyledButton>
              ))}
            </Stack>
          </Paper>

          {/* Function list */}
          <ScrollArea style={{ flex: 1, height: 300 }}>
            <Stack gap={4}>
              {filteredFunctions.map((fn) => (
                <Paper
                  key={fn.name}
                  withBorder
                  radius="sm"
                  p="sm"
                  style={{ cursor: 'pointer' }}
                  onClick={() => handleInsert(fn)}
                >
                  <Group justify="space-between" mb={4}>
                    <Code fw={600}>{fn.name}</Code>
                    <ActionIcon variant="subtle" size="sm">
                      <IconArrowRight size={14} />
                    </ActionIcon>
                  </Group>
                  <Text size="xs" c="dimmed">{fn.desc}</Text>
                  {fn.args.length > 0 && (
                    <Text size="xs" mt={4}>
                      <Text span c="dimmed">Arguments: </Text>
                      <Code size="xs">{fn.args.join(', ')}</Code>
                    </Text>
                  )}
                </Paper>
              ))}
              {filteredFunctions.length === 0 && (
                <Text size="sm" c="dimmed" ta="center" py="xl">
                  No functions found
                </Text>
              )}
            </Stack>
          </ScrollArea>
        </Group>

        <Divider />

        <Box>
          <Text size="sm" fw={500} mb="xs">Expression Syntax</Text>
          <Group gap="md">
            <Box>
              <Text size="xs" c="dimmed">Column reference</Text>
              <Code>[Column Name]</Code>
            </Box>
            <Box>
              <Text size="xs" c="dimmed">String literal</Text>
              <Code>"text"</Code>
            </Box>
            <Box>
              <Text size="xs" c="dimmed">Number</Text>
              <Code>123</Code>
            </Box>
            <Box>
              <Text size="xs" c="dimmed">Operators</Text>
              <Code>+ - * / = != &gt; &lt;</Code>
            </Box>
          </Group>
        </Box>
      </Stack>
    </Modal>
  )
}

// Simple expression preview/validation
export function ExpressionPreview({
  expression,
  valid,
  error,
}: {
  expression: string
  valid: boolean
  error?: string
}) {
  if (!expression) return null

  return (
    <Paper
      withBorder
      radius="sm"
      p="xs"
      bg={valid ? 'green.0' : 'red.0'}
      style={{ borderColor: valid ? 'var(--mantine-color-green-3)' : 'var(--mantine-color-red-3)' }}
    >
      <Group gap="xs">
        {valid ? (
          <Badge size="xs" color="green">Valid</Badge>
        ) : (
          <Badge size="xs" color="red">Error</Badge>
        )}
        <Text size="xs" c={valid ? 'green.7' : 'red.7'}>
          {error || 'Expression is valid'}
        </Text>
      </Group>
    </Paper>
  )
}
