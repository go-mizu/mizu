import { useState, useEffect, useMemo } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  Modal, TextInput, Stack, Group, Text, UnstyledButton, Box, Kbd, Badge, Divider
} from '@mantine/core'
import {
  IconSearch, IconHome, IconFolder, IconPlus, IconLayoutDashboard, IconDatabase,
  IconSettings, IconUsers, IconFileAnalytics, IconChartBar, IconChevronRight,
  IconCommand
} from '@tabler/icons-react'
import { useUIStore } from '../../stores/uiStore'
import { useQuestions, useDashboards, useCollections } from '../../api/hooks'

interface CommandItem {
  id: string
  label: string
  description?: string
  icon: typeof IconSearch
  action: () => void
  category: 'action' | 'navigation' | 'recent'
  shortcut?: string
}

export default function CommandPalette() {
  const navigate = useNavigate()
  const { commandPaletteOpen, closeCommandPalette } = useUIStore()
  const [search, setSearch] = useState('')
  const [selectedIndex, setSelectedIndex] = useState(0)

  const { data: questions } = useQuestions()
  const { data: dashboards } = useDashboards()
  const { data: collections } = useCollections()

  // Reset state when opening
  useEffect(() => {
    if (commandPaletteOpen) {
      setSearch('')
      setSelectedIndex(0)
    }
  }, [commandPaletteOpen])

  const allItems: CommandItem[] = useMemo(() => {
    const actions: CommandItem[] = [
      {
        id: 'new-question',
        label: 'New Question',
        description: 'Create a new question',
        icon: IconPlus,
        action: () => navigate('/question/new'),
        category: 'action',
      },
      {
        id: 'new-dashboard',
        label: 'New Dashboard',
        description: 'Create a new dashboard',
        icon: IconLayoutDashboard,
        action: () => navigate('/dashboard/new'),
        category: 'action',
      },
      {
        id: 'new-collection',
        label: 'New Collection',
        description: 'Create a new collection',
        icon: IconFolder,
        action: () => navigate('/collection/new'),
        category: 'action',
      },
    ]

    const navigation: CommandItem[] = [
      {
        id: 'nav-home',
        label: 'Home',
        icon: IconHome,
        action: () => navigate('/'),
        category: 'navigation',
      },
      {
        id: 'nav-browse',
        label: 'Browse',
        icon: IconFolder,
        action: () => navigate('/browse'),
        category: 'navigation',
      },
      {
        id: 'nav-databases',
        label: 'Databases',
        icon: IconDatabase,
        action: () => navigate('/browse/databases'),
        category: 'navigation',
      },
      {
        id: 'nav-models',
        label: 'Models',
        icon: IconFileAnalytics,
        action: () => navigate('/browse/models'),
        category: 'navigation',
      },
      {
        id: 'nav-metrics',
        label: 'Metrics',
        icon: IconChartBar,
        action: () => navigate('/browse/metrics'),
        category: 'navigation',
      },
      {
        id: 'nav-datamodel',
        label: 'Data Model',
        icon: IconDatabase,
        action: () => navigate('/admin/datamodel'),
        category: 'navigation',
      },
      {
        id: 'nav-people',
        label: 'People',
        icon: IconUsers,
        action: () => navigate('/admin/people'),
        category: 'navigation',
      },
      {
        id: 'nav-settings',
        label: 'Settings',
        icon: IconSettings,
        action: () => navigate('/admin/settings'),
        category: 'navigation',
      },
    ]

    const recentQuestions: CommandItem[] = (questions || []).slice(0, 5).map(q => ({
      id: `question-${q.id}`,
      label: q.name,
      description: 'Question',
      icon: IconChartBar,
      action: () => navigate(`/question/${q.id}`),
      category: 'recent' as const,
    }))

    const recentDashboards: CommandItem[] = (dashboards || []).slice(0, 5).map(d => ({
      id: `dashboard-${d.id}`,
      label: d.name,
      description: 'Dashboard',
      icon: IconLayoutDashboard,
      action: () => navigate(`/dashboard/${d.id}`),
      category: 'recent' as const,
    }))

    return [...actions, ...navigation, ...recentQuestions, ...recentDashboards]
  }, [questions, dashboards, navigate])

  const filteredItems = useMemo(() => {
    if (!search.trim()) {
      return allItems
    }
    const searchLower = search.toLowerCase()
    return allItems.filter(item =>
      item.label.toLowerCase().includes(searchLower) ||
      item.description?.toLowerCase().includes(searchLower)
    )
  }, [allItems, search])

  // Group items by category
  const groupedItems = useMemo(() => {
    const groups: Record<string, CommandItem[]> = {
      action: [],
      navigation: [],
      recent: [],
    }
    filteredItems.forEach(item => {
      groups[item.category].push(item)
    })
    return groups
  }, [filteredItems])

  const handleSelect = (item: CommandItem) => {
    closeCommandPalette()
    item.action()
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      setSelectedIndex(i => Math.min(i + 1, filteredItems.length - 1))
    } else if (e.key === 'ArrowUp') {
      e.preventDefault()
      setSelectedIndex(i => Math.max(i - 1, 0))
    } else if (e.key === 'Enter' && filteredItems[selectedIndex]) {
      e.preventDefault()
      handleSelect(filteredItems[selectedIndex])
    }
  }

  return (
    <Modal
      opened={commandPaletteOpen}
      onClose={closeCommandPalette}
      withCloseButton={false}
      padding={0}
      radius="lg"
      size="lg"
      styles={{
        body: { padding: 0 },
        content: { overflow: 'hidden' },
      }}
    >
      <Box>
        <Box p="md" style={{ borderBottom: '1px solid var(--mantine-color-gray-2)' }}>
          <TextInput
            placeholder="Type a command or search..."
            leftSection={<IconSearch size={18} />}
            rightSection={
              <Group gap={4}>
                <Kbd size="xs">Esc</Kbd>
              </Group>
            }
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            onKeyDown={handleKeyDown}
            autoFocus
            variant="unstyled"
            size="md"
          />
        </Box>

        <Box style={{ maxHeight: 400, overflow: 'auto' }}>
          {filteredItems.length === 0 ? (
            <Box p="xl" ta="center">
              <Text c="dimmed">No results found</Text>
            </Box>
          ) : (
            <Stack gap={0}>
              {/* Actions */}
              {groupedItems.action.length > 0 && (
                <>
                  <Text size="xs" fw={600} c="dimmed" px="md" py="xs">
                    Actions
                  </Text>
                  {groupedItems.action.map((item, i) => (
                    <CommandItemRow
                      key={item.id}
                      item={item}
                      selected={filteredItems.indexOf(item) === selectedIndex}
                      onSelect={() => handleSelect(item)}
                    />
                  ))}
                </>
              )}

              {/* Navigation */}
              {groupedItems.navigation.length > 0 && (
                <>
                  <Text size="xs" fw={600} c="dimmed" px="md" py="xs">
                    Go to
                  </Text>
                  {groupedItems.navigation.map((item, i) => (
                    <CommandItemRow
                      key={item.id}
                      item={item}
                      selected={filteredItems.indexOf(item) === selectedIndex}
                      onSelect={() => handleSelect(item)}
                    />
                  ))}
                </>
              )}

              {/* Recent */}
              {groupedItems.recent.length > 0 && (
                <>
                  <Text size="xs" fw={600} c="dimmed" px="md" py="xs">
                    Recent
                  </Text>
                  {groupedItems.recent.map((item, i) => (
                    <CommandItemRow
                      key={item.id}
                      item={item}
                      selected={filteredItems.indexOf(item) === selectedIndex}
                      onSelect={() => handleSelect(item)}
                    />
                  ))}
                </>
              )}
            </Stack>
          )}
        </Box>

        <Box p="sm" style={{ borderTop: '1px solid var(--mantine-color-gray-2)' }}>
          <Group gap="lg" justify="center">
            <Group gap={4}>
              <Kbd size="xs">↑</Kbd>
              <Kbd size="xs">↓</Kbd>
              <Text size="xs" c="dimmed">to navigate</Text>
            </Group>
            <Group gap={4}>
              <Kbd size="xs">↵</Kbd>
              <Text size="xs" c="dimmed">to select</Text>
            </Group>
            <Group gap={4}>
              <Kbd size="xs">esc</Kbd>
              <Text size="xs" c="dimmed">to close</Text>
            </Group>
          </Group>
        </Box>
      </Box>
    </Modal>
  )
}

interface CommandItemRowProps {
  item: CommandItem
  selected: boolean
  onSelect: () => void
}

function CommandItemRow({ item, selected, onSelect }: CommandItemRowProps) {
  const Icon = item.icon
  return (
    <UnstyledButton
      onClick={onSelect}
      px="md"
      py="sm"
      style={{
        backgroundColor: selected ? 'var(--mantine-color-blue-0)' : 'transparent',
        display: 'flex',
        alignItems: 'center',
        gap: 12,
        width: '100%',
      }}
    >
      <Icon size={18} color="var(--mantine-color-gray-6)" />
      <Box style={{ flex: 1 }}>
        <Text size="sm" fw={500}>{item.label}</Text>
        {item.description && (
          <Text size="xs" c="dimmed">{item.description}</Text>
        )}
      </Box>
      <IconChevronRight size={14} color="var(--mantine-color-gray-5)" />
    </UnstyledButton>
  )
}
