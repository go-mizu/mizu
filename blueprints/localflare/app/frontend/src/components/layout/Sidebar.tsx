import { NavLink, Stack, Text, Group, ThemeIcon, Divider, ScrollArea } from '@mantine/core'
import { useLocation, useNavigate } from 'react-router-dom'
import {
  IconWorld,
  IconBolt,
  IconDatabase,
  IconCloud,
  IconKey,
  IconChartBar,
  IconSettings,
  IconHome,
  IconServer,
} from '@tabler/icons-react'

const navItems = [
  { label: 'Dashboard', icon: IconHome, path: '/' },
  { label: 'Zones', icon: IconWorld, path: '/zones' },
  { label: 'Workers', icon: IconBolt, path: '/workers' },
  { label: 'KV', icon: IconKey, path: '/kv' },
  { label: 'R2', icon: IconCloud, path: '/r2' },
  { label: 'D1', icon: IconDatabase, path: '/d1' },
  { label: 'Analytics', icon: IconChartBar, path: '/analytics' },
]

export function Sidebar() {
  const location = useLocation()
  const navigate = useNavigate()

  return (
    <Stack h="100%" p="md">
      <Group gap="xs" mb="lg">
        <ThemeIcon size="lg" radius="md" color="orange">
          <IconServer size={20} />
        </ThemeIcon>
        <Text fw={700} size="lg" c="orange">Localflare</Text>
      </Group>

      <ScrollArea flex={1}>
        <Stack gap={4}>
          {navItems.map((item) => (
            <NavLink
              key={item.path}
              label={item.label}
              leftSection={<item.icon size={18} />}
              active={location.pathname === item.path ||
                (item.path !== '/' && location.pathname.startsWith(item.path))}
              onClick={() => navigate(item.path)}
              variant="filled"
              styles={{
                root: {
                  borderRadius: 'var(--mantine-radius-md)',
                },
              }}
            />
          ))}
        </Stack>
      </ScrollArea>

      <Divider />

      <NavLink
        label="Settings"
        leftSection={<IconSettings size={18} />}
        onClick={() => navigate('/settings')}
        variant="subtle"
        styles={{
          root: {
            borderRadius: 'var(--mantine-radius-md)',
          },
        }}
      />
    </Stack>
  )
}
