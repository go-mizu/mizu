import { useState } from 'react'
import {
  UnstyledButton,
  Group,
  Text,
  Menu,
  ThemeIcon,
  Box,
  TextInput,
  Stack,
  Badge,
  Divider,
} from '@mantine/core'
import {
  IconChevronDown,
  IconChevronRight,
  IconSearch,
  IconWorld,
  IconPlus,
  IconCheck,
} from '@tabler/icons-react'

export interface Zone {
  id: string
  name: string
  status: 'active' | 'pending' | 'moved'
  plan: 'free' | 'pro' | 'business' | 'enterprise'
}

interface ZoneSelectorProps {
  zones: Zone[]
  currentZone: Zone | null
  onSelect: (zone: Zone) => void
  onCreateNew?: () => void
}

const planColors: Record<string, string> = {
  free: 'gray',
  pro: 'blue',
  business: 'violet',
  enterprise: 'orange',
}

export function ZoneSelector({ zones, currentZone, onSelect, onCreateNew }: ZoneSelectorProps) {
  const [opened, setOpened] = useState(false)
  const [search, setSearch] = useState('')

  const filteredZones = zones.filter((zone) =>
    zone.name.toLowerCase().includes(search.toLowerCase())
  )

  if (zones.length === 0) {
    return null
  }

  return (
    <Menu
      opened={opened}
      onChange={setOpened}
      width={280}
      position="bottom-start"
      shadow="lg"
      offset={4}
    >
      <Menu.Target>
        <UnstyledButton
          w="100%"
          p="sm"
          style={{
            borderRadius: 'var(--mantine-radius-md)',
            backgroundColor: opened ? 'var(--mantine-color-dark-6)' : 'transparent',
            transition: 'background-color 150ms ease',
          }}
          styles={{
            root: {
              '&:hover': {
                backgroundColor: 'var(--mantine-color-dark-6)',
              },
            },
          }}
        >
          <Group justify="space-between" wrap="nowrap">
            <Group gap="sm" wrap="nowrap">
              <ThemeIcon size="md" variant="light" color="orange" radius="md">
                <IconWorld size={16} />
              </ThemeIcon>
              <Box style={{ overflow: 'hidden' }}>
                <Text size="xs" c="dimmed" fw={500}>
                  Zone
                </Text>
                <Text size="sm" fw={600} truncate style={{ maxWidth: 140 }}>
                  {currentZone?.name || 'Select zone'}
                </Text>
              </Box>
            </Group>
            {opened ? (
              <IconChevronDown size={16} style={{ flexShrink: 0 }} />
            ) : (
              <IconChevronRight size={16} style={{ flexShrink: 0 }} />
            )}
          </Group>
        </UnstyledButton>
      </Menu.Target>

      <Menu.Dropdown>
        <Box p="xs">
          <TextInput
            placeholder="Search zones..."
            size="xs"
            leftSection={<IconSearch size={14} />}
            value={search}
            onChange={(e) => setSearch(e.currentTarget.value)}
            styles={{
              input: {
                backgroundColor: 'var(--mantine-color-dark-7)',
              },
            }}
          />
        </Box>

        <Divider />

        <Box style={{ maxHeight: 250, overflowY: 'auto' }} py="xs">
          <Stack gap={2} px="xs">
            {filteredZones.length === 0 ? (
              <Text size="sm" c="dimmed" ta="center" py="md">
                No zones found
              </Text>
            ) : (
              filteredZones.map((zone) => (
                <UnstyledButton
                  key={zone.id}
                  w="100%"
                  p="xs"
                  onClick={() => {
                    onSelect(zone)
                    setOpened(false)
                    setSearch('')
                  }}
                  style={{
                    borderRadius: 'var(--mantine-radius-sm)',
                    backgroundColor:
                      currentZone?.id === zone.id
                        ? 'var(--mantine-color-orange-9)'
                        : 'transparent',
                  }}
                >
                  <Group justify="space-between" wrap="nowrap">
                    <Box style={{ overflow: 'hidden', flex: 1 }}>
                      <Text size="sm" fw={500} truncate>
                        {zone.name}
                      </Text>
                    </Box>
                    <Group gap="xs" wrap="nowrap">
                      <Badge size="xs" variant="light" color={planColors[zone.plan]}>
                        {zone.plan}
                      </Badge>
                      {currentZone?.id === zone.id && (
                        <IconCheck size={14} color="var(--mantine-color-orange-5)" />
                      )}
                    </Group>
                  </Group>
                </UnstyledButton>
              ))
            )}
          </Stack>
        </Box>

        {onCreateNew && (
          <>
            <Divider />
            <Box p="xs">
              <UnstyledButton
                w="100%"
                p="xs"
                onClick={() => {
                  onCreateNew()
                  setOpened(false)
                }}
                style={{
                  borderRadius: 'var(--mantine-radius-sm)',
                }}
              >
                <Group gap="xs">
                  <IconPlus size={14} />
                  <Text size="sm">Add new zone</Text>
                </Group>
              </UnstyledButton>
            </Box>
          </>
        )}
      </Menu.Dropdown>
    </Menu>
  )
}
