import { NavLink, Stack, Text, Group, ThemeIcon, Divider, ScrollArea, Box } from '@mantine/core'
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
} from '@tabler/icons-react'

interface NavItem {
  label: string
  icon: typeof IconLayoutDashboard
  path: string
}

interface NavSection {
  title: string
  items: NavItem[]
}

const navSections: NavSection[] = [
  {
    title: '',
    items: [
      { label: 'Overview', icon: IconLayoutDashboard, path: '/' },
    ],
  },
  {
    title: 'COMPUTE',
    items: [
      { label: 'Durable Objects', icon: IconDatabase, path: '/durable-objects' },
      { label: 'Cron Triggers', icon: IconClock, path: '/cron' },
    ],
  },
  {
    title: 'STORAGE',
    items: [
      { label: 'Queues', icon: IconMailbox, path: '/queues' },
      { label: 'Vectorize', icon: IconVectorTriangle, path: '/vectorize' },
    ],
  },
  {
    title: 'ANALYTICS',
    items: [
      { label: 'Analytics Engine', icon: IconChartLine, path: '/analytics-engine' },
    ],
  },
  {
    title: 'AI',
    items: [
      { label: 'Workers AI', icon: IconRobot, path: '/ai' },
      { label: 'AI Gateway', icon: IconApiApp, path: '/ai-gateway' },
      { label: 'Hyperdrive', icon: IconBolt, path: '/hyperdrive' },
    ],
  },
]

export function Sidebar() {
  const location = useLocation()
  const navigate = useNavigate()

  const isActive = (path: string) => {
    if (path === '/') {
      return location.pathname === '/'
    }
    return location.pathname === path || location.pathname.startsWith(path + '/')
  }

  return (
    <Stack h="100%" p="md" gap={0}>
      <Group gap="xs" mb="xl" px="xs">
        <ThemeIcon size="lg" radius="md" variant="gradient" gradient={{ from: 'orange', to: 'red' }}>
          <IconFlame size={20} />
        </ThemeIcon>
        <Text fw={700} size="lg" c="orange">Localflare</Text>
      </Group>

      <ScrollArea flex={1} scrollbarSize={6}>
        <Stack gap="lg">
          {navSections.map((section, idx) => (
            <Box key={idx}>
              {section.title && (
                <Text size="xs" c="dimmed" fw={600} mb="xs" px="xs" tt="uppercase">
                  {section.title}
                </Text>
              )}
              <Stack gap={4}>
                {section.items.map((item) => (
                  <NavLink
                    key={item.path}
                    label={item.label}
                    leftSection={<item.icon size={18} stroke={1.5} />}
                    active={isActive(item.path)}
                    onClick={() => navigate(item.path)}
                    variant="filled"
                    styles={{
                      root: {
                        borderRadius: 'var(--mantine-radius-md)',
                      },
                      label: {
                        fontWeight: 500,
                      },
                    }}
                  />
                ))}
              </Stack>
            </Box>
          ))}
        </Stack>
      </ScrollArea>

      <Divider my="md" />

      <Stack gap={4}>
        <NavLink
          label="Settings"
          leftSection={<IconSettings size={18} stroke={1.5} />}
          onClick={() => navigate('/settings')}
          variant="subtle"
          styles={{
            root: {
              borderRadius: 'var(--mantine-radius-md)',
            },
          }}
        />
        <NavLink
          label="Documentation"
          leftSection={<IconBook size={18} stroke={1.5} />}
          component="a"
          href="https://developers.cloudflare.com"
          target="_blank"
          variant="subtle"
          styles={{
            root: {
              borderRadius: 'var(--mantine-radius-md)',
            },
          }}
        />
      </Stack>
    </Stack>
  )
}
