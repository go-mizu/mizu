import { Group, ActionIcon, Tooltip, Divider, Text, Kbd, Menu } from '@mantine/core'
import {
  IconPlayerPlay, IconWand, IconCopy, IconClipboard,
  IconDatabase, IconHistory, IconDotsVertical, IconDownload,
  IconBraces
} from '@tabler/icons-react'

interface EditorToolbarProps {
  onRun: () => void
  onFormat: () => void
  onCopy: () => void
  onToggleSchema?: () => void
  onShowHistory?: () => void
  isRunning?: boolean
  schemaVisible?: boolean
  hasSelection?: boolean
}

export default function EditorToolbar({
  onRun,
  onFormat,
  onCopy,
  onToggleSchema,
  onShowHistory,
  isRunning = false,
  schemaVisible = false,
  hasSelection = false,
}: EditorToolbarProps) {
  const isMac = navigator.platform.toUpperCase().indexOf('MAC') >= 0
  const modKey = isMac ? 'Cmd' : 'Ctrl'

  return (
    <Group
      px="sm"
      py={6}
      justify="space-between"
      style={{
        backgroundColor: '#FAFAFA',
        borderBottom: '1px solid var(--mantine-color-gray-2)',
      }}
    >
      {/* Left side - Run button */}
      <Group gap="xs">
        <Tooltip
          label={
            <Group gap={4}>
              <Text size="xs">{hasSelection ? 'Run selected' : 'Run query'}</Text>
              <Kbd size="xs">{modKey}</Kbd>
              <Text size="xs">+</Text>
              <Kbd size="xs">Enter</Kbd>
            </Group>
          }
        >
          <ActionIcon
            variant="filled"
            color="blue"
            size="md"
            onClick={onRun}
            loading={isRunning}
            data-testid="btn-run-sql"
          >
            <IconPlayerPlay size={16} />
          </ActionIcon>
        </Tooltip>

        {hasSelection && (
          <Text size="xs" c="dimmed">
            Running selection
          </Text>
        )}
      </Group>

      {/* Right side - Tools */}
      <Group gap={4}>
        <Tooltip
          label={
            <Group gap={4}>
              <Text size="xs">Format SQL</Text>
              <Kbd size="xs">{modKey}</Kbd>
              <Text size="xs">+</Text>
              <Kbd size="xs">Shift</Kbd>
              <Text size="xs">+</Text>
              <Kbd size="xs">F</Kbd>
            </Group>
          }
        >
          <ActionIcon
            variant="subtle"
            color="gray"
            size="md"
            onClick={onFormat}
          >
            <IconWand size={16} />
          </ActionIcon>
        </Tooltip>

        <Tooltip label="Copy query">
          <ActionIcon
            variant="subtle"
            color="gray"
            size="md"
            onClick={onCopy}
          >
            <IconCopy size={16} />
          </ActionIcon>
        </Tooltip>

        <Divider orientation="vertical" mx={4} />

        {onToggleSchema && (
          <Tooltip label={schemaVisible ? 'Hide schema' : 'Show schema'}>
            <ActionIcon
              variant={schemaVisible ? 'light' : 'subtle'}
              color={schemaVisible ? 'blue' : 'gray'}
              size="md"
              onClick={onToggleSchema}
            >
              <IconDatabase size={16} />
            </ActionIcon>
          </Tooltip>
        )}

        <Tooltip label="Variables">
          <ActionIcon
            variant="subtle"
            color="gray"
            size="md"
          >
            <IconBraces size={16} />
          </ActionIcon>
        </Tooltip>

        {onShowHistory && (
          <Tooltip label="Query history">
            <ActionIcon
              variant="subtle"
              color="gray"
              size="md"
              onClick={onShowHistory}
            >
              <IconHistory size={16} />
            </ActionIcon>
          </Tooltip>
        )}

        <Menu position="bottom-end" withArrow>
          <Menu.Target>
            <ActionIcon variant="subtle" color="gray" size="md">
              <IconDotsVertical size={16} />
            </ActionIcon>
          </Menu.Target>
          <Menu.Dropdown>
            <Menu.Label>Actions</Menu.Label>
            <Menu.Item leftSection={<IconDownload size={14} />}>
              Export to SQL file
            </Menu.Item>
            <Menu.Item leftSection={<IconClipboard size={14} />}>
              Copy as formatted
            </Menu.Item>
          </Menu.Dropdown>
        </Menu>
      </Group>
    </Group>
  )
}

/**
 * Keyboard shortcut hints component
 */
export function KeyboardShortcuts() {
  const isMac = navigator.platform.toUpperCase().indexOf('MAC') >= 0
  const modKey = isMac ? '\u2318' : 'Ctrl'

  const shortcuts = [
    { keys: [`${modKey}`, 'Enter'], action: 'Run query' },
    { keys: [`${modKey}`, 'Shift', 'F'], action: 'Format SQL' },
    { keys: [`${modKey}`, 'Space'], action: 'Autocomplete' },
    { keys: [`${modKey}`, '/'], action: 'Toggle comment' },
    { keys: [`${modKey}`, 'D'], action: 'Duplicate line' },
  ]

  return (
    <Group gap="md" wrap="wrap">
      {shortcuts.map((shortcut, i) => (
        <Group key={i} gap={4}>
          {shortcut.keys.map((key, j) => (
            <span key={j}>
              <Kbd size="xs">{key}</Kbd>
              {j < shortcut.keys.length - 1 && <Text span size="xs" mx={2}>+</Text>}
            </span>
          ))}
          <Text size="xs" c="dimmed" ml={4}>{shortcut.action}</Text>
        </Group>
      ))}
    </Group>
  )
}
