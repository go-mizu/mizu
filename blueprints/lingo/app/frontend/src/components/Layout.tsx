import { AppShell, Group, Stack, UnstyledButton, Text, Avatar, Tooltip } from '@mantine/core'
import { IconFlame, IconHome, IconTrophy, IconUser, IconShoppingCart, IconMedal, IconHeart, IconBook } from '@tabler/icons-react'
import { useNavigate, useLocation } from 'react-router-dom'
import { useAuthStore } from '../stores/auth'
import { colors } from '../styles/tokens'

const navItems = [
  { icon: IconHome, label: 'LEARN', path: '/learn' },
  { icon: IconBook, label: 'LETTERS', path: '/letters' },
  { icon: IconTrophy, label: 'LEADERBOARDS', path: '/leaderboards' },
  { icon: IconMedal, label: 'QUESTS', path: '/quests' },
  { icon: IconShoppingCart, label: 'SHOP', path: '/shop' },
  { icon: IconUser, label: 'PROFILE', path: '/profile' },
]

export default function Layout({ children }: { children: React.ReactNode }) {
  const navigate = useNavigate()
  const location = useLocation()
  const { user } = useAuthStore()

  return (
    <AppShell
      header={{ height: 60 }}
      navbar={{ width: 256, breakpoint: 'sm' }}
      padding="md"
      styles={{
        main: {
          backgroundColor: colors.neutral.background,
          minHeight: '100vh',
        },
        header: {
          backgroundColor: colors.neutral.white,
          borderBottom: `2px solid ${colors.neutral.border}`,
        },
        navbar: {
          backgroundColor: colors.neutral.white,
          borderRight: `2px solid ${colors.neutral.border}`,
        },
      }}
    >
      <AppShell.Header>
        <Group h="100%" px="md" justify="space-between">
          <Group>
            <Text
              size="xl"
              fw={800}
              style={{ color: colors.primary.green, cursor: 'pointer', letterSpacing: '-0.5px' }}
              onClick={() => navigate('/learn')}
            >
              lingo
            </Text>
          </Group>

          <Group gap="lg">
            {/* Streak */}
            <Tooltip label={`${user?.streak_days || 0} day streak`}>
              <Group gap={4} className="stat-item streak">
                <IconFlame size={24} style={{ color: colors.accent.orange }} />
                <Text fw={700} style={{ color: colors.accent.orange }}>{user?.streak_days || 0}</Text>
              </Group>
            </Tooltip>

            {/* Gems */}
            <Tooltip label="Gems">
              <Group gap={4} className="stat-item gems">
                <Text size="lg">💎</Text>
                <Text fw={700} style={{ color: colors.secondary.blue }}>{user?.gems || 0}</Text>
              </Group>
            </Tooltip>

            {/* Hearts */}
            <Tooltip label="Hearts">
              <Group gap={4} className="stat-item hearts">
                <IconHeart size={24} style={{ color: colors.accent.pink, fill: colors.accent.pink }} />
                <Text fw={700} style={{ color: colors.accent.pink }}>{user?.hearts || 0}</Text>
              </Group>
            </Tooltip>
          </Group>
        </Group>
      </AppShell.Header>

      <AppShell.Navbar p="md">
        <Stack gap="xs">
          {navItems.map((item) => {
            const isActive = location.pathname === item.path
            return (
              <UnstyledButton
                key={item.path}
                onClick={() => navigate(item.path)}
                className={`nav-item ${isActive ? 'active' : ''}`}
                style={{
                  padding: '12px 16px',
                  borderRadius: '12px',
                  backgroundColor: isActive ? colors.secondary.blueLight : 'transparent',
                  border: isActive ? `2px solid ${colors.secondary.blue}` : '2px solid transparent',
                  transition: 'all 0.15s ease',
                }}
              >
                <Group gap="md">
                  <item.icon
                    size={28}
                    style={{
                      color: isActive ? colors.secondary.blue : colors.text.secondary,
                    }}
                  />
                  <Text
                    fw={700}
                    size="sm"
                    style={{
                      color: isActive ? colors.secondary.blue : colors.text.secondary,
                      letterSpacing: '0.5px',
                    }}
                  >
                    {item.label}
                  </Text>
                </Group>
              </UnstyledButton>
            )
          })}
        </Stack>

        {/* User info at bottom */}
        <Stack mt="auto" pt="md" style={{ borderTop: `1px solid ${colors.neutral.border}` }}>
          <Group>
            <Avatar size="md" radius="xl" color="green" style={{ backgroundColor: colors.primary.green }}>
              {user?.username?.charAt(0).toUpperCase()}
            </Avatar>
            <div>
              <Text size="sm" fw={700} style={{ color: colors.text.primary }}>
                {user?.display_name || user?.username}
              </Text>
              <Text size="xs" style={{ color: colors.text.secondary }}>
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
