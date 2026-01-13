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
  ActionIcon,
  useMantineColorScheme,
  useComputedColorScheme,
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
  IconCode,
  IconKey,
  IconCloud,
  IconTable,
  IconWorld,
  IconPhoto,
  IconVideo,
  IconActivity,
  IconSun,
  IconMoon,
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
    title: 'WORKERS & SCRIPTS',
    collapsible: true,
    items: [
      { label: 'Workers', icon: IconCode, path: '/workers' },
      { label: 'Durable Objects', icon: IconDatabase, path: '/durable-objects' },
      { label: 'Cron Triggers', icon: IconClock, path: '/cron' },
    ],
  },
  {
    title: 'STORAGE',
    collapsible: true,
    items: [
      { label: 'KV', icon: IconKey, path: '/kv' },
      { label: 'R2', icon: IconCloud, path: '/r2' },
      { label: 'D1', icon: IconTable, path: '/d1' },
      { label: 'Queues', icon: IconMailbox, path: '/queues' },
      { label: 'Vectorize', icon: IconVectorTriangle, path: '/vectorize' },
    ],
  },
  {
    title: 'MEDIA',
    collapsible: true,
    items: [
      { label: 'Pages', icon: IconWorld, path: '/pages' },
      { label: 'Images', icon: IconPhoto, path: '/images' },
      { label: 'Stream', icon: IconVideo, path: '/stream' },
    ],
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
  {
    title: 'ANALYTICS',
    collapsible: true,
    items: [
      { label: 'Analytics Engine', icon: IconChartLine, path: '/analytics-engine' },
      { label: 'Observability', icon: IconActivity, path: '/observability' },
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
  const { setColorScheme } = useMantineColorScheme()
  const computedColorScheme = useComputedColorScheme('light', { getInitialValueInEffect: true })
  const [expandedSections, setExpandedSections] = useState<string[]>([
    'WORKERS & SCRIPTS',
    'STORAGE',
    'MEDIA',
    'AI',
    'ANALYTICS',
  ])
  const [currentZone, setCurrentZone] = useState<Zone>(mockZones[0])

  const toggleColorScheme = () => {
    setColorScheme(computedColorScheme === 'light' ? 'dark' : 'light')
  }

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
        <Group justify="space-between" px="xs" mb="xs">
          <Text size="xs" c="dimmed" fw={500}>Theme</Text>
          <Tooltip label={computedColorScheme === 'light' ? 'Switch to dark mode' : 'Switch to light mode'} position="right">
            <ActionIcon
              onClick={toggleColorScheme}
              variant="default"
              size="md"
              aria-label="Toggle color scheme"
            >
              {computedColorScheme === 'light' ? <IconMoon size={16} stroke={1.5} /> : <IconSun size={16} stroke={1.5} />}
            </ActionIcon>
          </Tooltip>
        </Group>
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
