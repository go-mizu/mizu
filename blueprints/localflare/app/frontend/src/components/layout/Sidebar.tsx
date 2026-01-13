import { useState } from 'react'
import {
  NavLink,
  Stack,
  Text,
  Group,
  ThemeIcon,
  Divider,
  ScrollArea,
  Box,
  Collapse,
  UnstyledButton,
  Tooltip,
} from '@mantine/core'
import { useLocation, useNavigate } from 'react-router-dom'
import {
  IconLayoutDashboard,
  IconDatabase,
  IconClock,
  IconMailbox,
  IconVectorTriangle,
  IconChartLine,
  IconRobot,
  IconApiApp,
  IconBolt,
  IconSettings,
  IconBook,
  IconFlame,
  IconChevronDown,
  IconChevronRight,
  IconExternalLink,
} from '@tabler/icons-react'
import { ZoneSelector, type Zone } from './ZoneSelector'

interface NavItem {
  label: string
  icon: typeof IconLayoutDashboard
  path: string
  badge?: string
}

interface NavSection {
  title: string
  items: NavItem[]
  collapsible?: boolean
}

const navSections: NavSection[] = [
  {
    title: '',
    items: [{ label: 'Overview', icon: IconLayoutDashboard, path: '/' }],
  },
  {
    title: 'COMPUTE',
    collapsible: true,
    items: [
      { label: 'Durable Objects', icon: IconDatabase, path: '/durable-objects' },
      { label: 'Cron Triggers', icon: IconClock, path: '/cron' },
    ],
  },
  {
    title: 'STORAGE',
    collapsible: true,
    items: [
      { label: 'Queues', icon: IconMailbox, path: '/queues' },
      { label: 'Vectorize', icon: IconVectorTriangle, path: '/vectorize' },
    ],
  },
  {
    title: 'ANALYTICS',
    collapsible: true,
    items: [{ label: 'Analytics Engine', icon: IconChartLine, path: '/analytics-engine' }],
  },
  {
    title: 'AI',
    collapsible: true,
    items: [
      { label: 'Workers AI', icon: IconRobot, path: '/ai' },
      { label: 'AI Gateway', icon: IconApiApp, path: '/ai-gateway' },
      { label: 'Hyperdrive', icon: IconBolt, path: '/hyperdrive' },
    ],
  },
]

// Mock zones for demonstration
const mockZones: Zone[] = [
  { id: '1', name: 'example.com', status: 'active', plan: 'pro' },
  { id: '2', name: 'myapp.dev', status: 'active', plan: 'free' },
  { id: '3', name: 'enterprise-app.io', status: 'active', plan: 'enterprise' },
]

export function Sidebar() {
  const location = useLocation()
  const navigate = useNavigate()
  const [expandedSections, setExpandedSections] = useState<string[]>([
    'COMPUTE',
    'STORAGE',
    'ANALYTICS',
    'AI',
  ])
  const [currentZone, setCurrentZone] = useState<Zone>(mockZones[0])

  const isActive = (path: string) => {
    if (path === '/') {
      return location.pathname === '/'
    }
    return location.pathname === path || location.pathname.startsWith(path + '/')
  }

  const toggleSection = (title: string) => {
    setExpandedSections((prev) =>
      prev.includes(title) ? prev.filter((s) => s !== title) : [...prev, title]
    )
  }

  const isSectionExpanded = (title: string) => expandedSections.includes(title)

  return (
    <Stack h="100%" gap={0}>
      {/* Brand Header */}
      <Box p="md" pb="xs">
        <Group gap="xs" mb="md">
          <ThemeIcon
            size="lg"
            radius="md"
            variant="gradient"
            gradient={{ from: 'orange', to: 'red' }}
          >
            <IconFlame size={20} />
          </ThemeIcon>
          <Text fw={700} size="lg" c="orange">
            Localflare
          </Text>
        </Group>

        {/* Zone Selector - Cloudflare style */}
        <ZoneSelector
          zones={mockZones}
          currentZone={currentZone}
          onSelect={setCurrentZone}
          onCreateNew={() => console.log('Create new zone')}
        />
      </Box>

      <Divider my="xs" />

      {/* Navigation Sections */}
      <ScrollArea flex={1} scrollbarSize={6} px="md" py="xs">
        <Stack gap="md">
          {navSections.map((section, idx) => (
            <Box key={idx}>
              {section.title && (
                <UnstyledButton
                  w="100%"
                  onClick={() => section.collapsible && toggleSection(section.title)}
                  style={{ cursor: section.collapsible ? 'pointer' : 'default' }}
                >
                  <Group gap="xs" mb="xs" px="xs">
                    {section.collapsible && (
                      isSectionExpanded(section.title) ? (
                        <IconChevronDown size={12} color="var(--mantine-color-dimmed)" />
                      ) : (
                        <IconChevronRight size={12} color="var(--mantine-color-dimmed)" />
                      )
                    )}
                    <Text size="xs" c="dimmed" fw={600} tt="uppercase">
                      {section.title}
                    </Text>
                  </Group>
                </UnstyledButton>
              )}

              <Collapse in={!section.collapsible || isSectionExpanded(section.title)}>
                <Stack gap={2}>
                  {section.items.map((item) => (
                    <NavLink
                      key={item.path}
                      label={
                        <Group justify="space-between" wrap="nowrap">
                          <Text size="sm" fw={500}>
                            {item.label}
                          </Text>
                          {item.badge && (
                            <Text size="xs" c="orange" fw={600}>
                              {item.badge}
                            </Text>
                          )}
                        </Group>
                      }
                      leftSection={<item.icon size={20} stroke={1.5} />}
                      active={isActive(item.path)}
                      onClick={() => navigate(item.path)}
                      variant="filled"
                      styles={{
                        root: {
                          borderRadius: 'var(--mantine-radius-md)',
                          padding: '10px 12px',
                        },
                        section: {
                          marginRight: '12px',
                        },
                      }}
                    />
                  ))}
                </Stack>
              </Collapse>
            </Box>
          ))}
        </Stack>
      </ScrollArea>

      <Divider />

      {/* Footer Links */}
      <Stack gap={2} p="md">
        <NavLink
          label="Settings"
          leftSection={<IconSettings size={20} stroke={1.5} />}
          onClick={() => navigate('/settings')}
          variant="subtle"
          styles={{
            root: {
              borderRadius: 'var(--mantine-radius-md)',
              padding: '10px 12px',
            },
            section: {
              marginRight: '12px',
            },
          }}
        />
        <Tooltip label="Opens in new tab" position="right">
          <NavLink
            label={
              <Group gap="xs" wrap="nowrap">
                <Text size="sm">Documentation</Text>
                <IconExternalLink size={12} color="var(--mantine-color-dimmed)" />
              </Group>
            }
            leftSection={<IconBook size={20} stroke={1.5} />}
            onClick={() => window.open('https://developers.cloudflare.com', '_blank', 'noopener,noreferrer')}
            variant="subtle"
            styles={{
              root: {
                borderRadius: 'var(--mantine-radius-md)',
                padding: '10px 12px',
              },
              section: {
                marginRight: '12px',
              },
            }}
          />
        </Tooltip>
      </Stack>
    </Stack>
  )
}
