import { AppShell, Group, Stack, UnstyledButton, Text, Avatar, Badge, Tooltip } from '@mantine/core'
import { IconFlame, IconHome, IconTrophy, IconUser, IconShoppingCart, IconMedal, IconHeart } from '@tabler/icons-react'
import { useNavigate, useLocation } from 'react-router-dom'
import { useAuthStore } from '../stores/auth'

const navItems = [
  { icon: IconHome, label: 'Learn', path: '/learn' },
  { icon: IconTrophy, label: 'Leaderboards', path: '/leaderboards' },
  { icon: IconMedal, label: 'Achievements', path: '/achievements' },
  { icon: IconShoppingCart, label: 'Shop', path: '/shop' },
  { icon: IconUser, label: 'Profile', path: '/profile' },
]

export default function Layout({ children }: { children: React.ReactNode }) {
  const navigate = useNavigate()
  const location = useLocation()
  const { user } = useAuthStore()

  return (
    <AppShell
      header={{ height: 60 }}
      navbar={{ width: 250, breakpoint: 'sm' }}
      padding="md"
      styles={{
        main: {
          backgroundColor: '#131f24',
          minHeight: '100vh',
        },
        header: {
          backgroundColor: '#1a2c33',
          borderBottom: '2px solid #3d5a68',
        },
        navbar: {
          backgroundColor: '#1a2c33',
          borderRight: '2px solid #3d5a68',
        },
      }}
    >
      <AppShell.Header>
        <Group h="100%" px="md" justify="space-between">
          <Group>
            <Text
              size="xl"
              fw={800}
              style={{ color: '#58cc02', cursor: 'pointer' }}
              onClick={() => navigate('/learn')}
            >
              Lingo
            </Text>
          </Group>

          <Group gap="lg">
            {/* Streak */}
            <Tooltip label={`${user?.streak_days || 0} day streak`}>
              <Group gap={4}>
                <IconFlame size={24} style={{ color: '#ff9600' }} />
                <Text fw={700} style={{ color: '#ff9600' }}>{user?.streak_days || 0}</Text>
              </Group>
            </Tooltip>

            {/* Gems */}
            <Tooltip label="Gems">
              <Group gap={4}>
                <Text size="lg">💎</Text>
                <Text fw={700} style={{ color: '#1cb0f6' }}>{user?.gems || 0}</Text>
              </Group>
            </Tooltip>

            {/* Hearts */}
            <Tooltip label="Hearts">
              <Group gap={4}>
                <IconHeart size={24} style={{ color: '#ff4b4b', fill: '#ff4b4b' }} />
                <Text fw={700} style={{ color: '#ff4b4b' }}>{user?.hearts || 0}</Text>
              </Group>
            </Tooltip>

            {/* XP */}
            <Badge
              size="lg"
              variant="filled"
              color="yellow"
              style={{ fontWeight: 700 }}
            >
              {user?.xp_total || 0} XP
            </Badge>
          </Group>
        </Group>
      </AppShell.Header>

      <AppShell.Navbar p="md">
        <Stack gap="xs">
          {navItems.map((item) => (
            <UnstyledButton
              key={item.path}
              onClick={() => navigate(item.path)}
              style={{
                padding: '12px 16px',
                borderRadius: '12px',
                backgroundColor: location.pathname === item.path ? '#58cc02' : 'transparent',
                transition: 'all 0.2s ease',
              }}
              onMouseEnter={(e) => {
                if (location.pathname !== item.path) {
                  e.currentTarget.style.backgroundColor = '#233a42'
                }
              }}
              onMouseLeave={(e) => {
                if (location.pathname !== item.path) {
                  e.currentTarget.style.backgroundColor = 'transparent'
                }
              }}
            >
              <Group>
                <item.icon
                  size={24}
                  style={{
                    color: location.pathname === item.path ? '#131f24' : '#8fa8b2',
                  }}
                />
                <Text
                  fw={700}
                  size="md"
                  style={{
                    color: location.pathname === item.path ? '#131f24' : '#8fa8b2',
                  }}
                >
                  {item.label}
                </Text>
              </Group>
            </UnstyledButton>
          ))}
        </Stack>

        {/* User info at bottom */}
        <Stack mt="auto" pt="md" style={{ borderTop: '1px solid #3d5a68' }}>
          <Group>
            <Avatar size="md" radius="xl" color="green">
              {user?.username?.charAt(0).toUpperCase()}
            </Avatar>
            <div>
              <Text size="sm" fw={700} style={{ color: 'white' }}>
                {user?.display_name || user?.username}
              </Text>
              <Text size="xs" style={{ color: '#8fa8b2' }}>
                @{user?.username}
              </Text>
            </div>
          </Group>
        </Stack>
      </AppShell.Navbar>

      <AppShell.Main>
        {children}
      </AppShell.Main>
    </AppShell>
  )
}
