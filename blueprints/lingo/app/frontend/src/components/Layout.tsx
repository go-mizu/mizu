import { useState, useEffect } from 'react'
import { AppShell, Group, Stack, UnstyledButton, Text, Tooltip } from '@mantine/core'
import { IconFlame, IconHome, IconTrophy, IconUser, IconShoppingCart, IconMedal, IconHeart, IconBook, IconDots, IconNotebook } from '@tabler/icons-react'
import { useNavigate, useLocation } from 'react-router-dom'
import { useAuthStore } from '../stores/auth'
import { colors } from '../styles/tokens'
import { coursesApi, Language } from '../api/client'

// Duolingo-style colorful navigation icons
const navItems = [
  { icon: IconHome, label: 'LEARN', path: '/learn', color: '#58CC02' },
  { icon: IconNotebook, label: 'STORIES', path: '/stories', color: '#FF4B4B' },
  { icon: IconBook, label: 'LETTERS', path: '/letters', color: '#CE82FF' },
  { icon: IconTrophy, label: 'LEADERBOARDS', path: '/leaderboards', color: '#FFC800' },
  { icon: IconMedal, label: 'QUESTS', path: '/quests', color: '#FF9600' },
  { icon: IconShoppingCart, label: 'SHOP', path: '/shop', color: '#1CB0F6' },
  { icon: IconUser, label: 'PROFILE', path: '/profile', color: '#58CC02' },
  { icon: IconDots, label: 'MORE', path: '/more', color: '#CE82FF' },
]

export default function Layout({ children }: { children: React.ReactNode }) {
  const navigate = useNavigate()
  const location = useLocation()
  const { user } = useAuthStore()
  const [learningLanguage, setLearningLanguage] = useState<Language | null>(null)

  useEffect(() => {
    async function loadLanguage() {
      if (!user?.active_course_id) return

      try {
        const course = await coursesApi.getCourse(user.active_course_id)
        const languages = await coursesApi.listLanguages()
        const lang = languages.find((l) => l.id === course.learning_language_id)
        if (lang) {
          setLearningLanguage(lang)
        }
      } catch (err) {
        console.error('Failed to load language:', err)
      }
    }

    loadLanguage()
  }, [user?.active_course_id])

  return (
    <AppShell
      navbar={{ width: 220, breakpoint: 'sm' }}
      padding={0}
      styles={{
        main: {
          backgroundColor: '#FFFFFF',
          minHeight: '100vh',
        },
        navbar: {
          backgroundColor: '#FFFFFF',
          borderRight: 'none',
          padding: '24px 16px',
        },
      }}
    >
      <AppShell.Navbar>
        {/* Logo */}
        <Text
          size="xl"
          fw={800}
          mb="xl"
          style={{
            color: colors.primary.green,
            cursor: 'pointer',
            letterSpacing: '-0.5px',
            fontSize: '1.75rem',
            fontFamily: '"Nunito", "DIN Round Pro", sans-serif',
          }}
          onClick={() => navigate('/learn')}
        >
          lingo
        </Text>

        <Stack gap={4}>
          {navItems.map((item) => {
            const isActive = location.pathname === item.path
            return (
              <UnstyledButton
                key={item.path}
                onClick={() => navigate(item.path)}
                style={{
                  padding: '12px 16px',
                  borderRadius: '12px',
                  backgroundColor: isActive ? `${item.color}20` : 'transparent',
                  border: isActive ? `2px solid ${item.color}` : '2px solid transparent',
                  transition: 'all 0.15s ease',
                }}
              >
                <Group gap="md">
                  <item.icon
                    size={28}
                    style={{
                      color: item.color,
                    }}
                  />
                  <Text
                    fw={700}
                    size="sm"
                    style={{
                      color: isActive ? item.color : colors.text.secondary,
                      letterSpacing: '0.5px',
                      textTransform: 'uppercase',
                    }}
                  >
                    {item.label}
                  </Text>
                </Group>
              </UnstyledButton>
            )
          })}
        </Stack>
      </AppShell.Navbar>

      <AppShell.Main>
        {/* Top stats bar */}
        <div style={{
          position: 'sticky',
          top: 0,
          backgroundColor: '#FFFFFF',
          zIndex: 100,
          padding: '16px 24px',
          display: 'flex',
          justifyContent: 'flex-end',
        }}>
          <Group gap="lg">
            {/* Language flag - clickable to change course */}
            <Tooltip label={learningLanguage ? `Learning ${learningLanguage.name}` : 'Select a course'}>
              <UnstyledButton
                onClick={() => navigate('/courses')}
                style={{
                  padding: '4px 8px',
                  borderRadius: 8,
                  transition: 'background-color 0.15s ease',
                }}
              >
                <Text size="xl">{learningLanguage?.flag_emoji || '🌐'}</Text>
              </UnstyledButton>
            </Tooltip>

            {/* Streak */}
            <Tooltip label={`${user?.streak_days || 0} day streak`}>
              <Group gap={4}>
                <IconFlame size={24} style={{ color: user?.streak_days ? colors.accent.orange : '#E5E5E5' }} />
                <Text fw={700} style={{ color: user?.streak_days ? colors.accent.orange : '#AFAFAF' }}>
                  {user?.streak_days || 0}
                </Text>
              </Group>
            </Tooltip>

            {/* Gems */}
            <Tooltip label="Gems">
              <Group gap={4}>
                <Text size="lg">💎</Text>
                <Text fw={700} style={{ color: colors.secondary.blue }}>{user?.gems || 500}</Text>
              </Group>
            </Tooltip>

            {/* Hearts */}
            <Tooltip label="Hearts">
              <Group gap={4}>
                <IconHeart size={24} style={{ color: colors.accent.pink, fill: colors.accent.pink }} />
                <Text fw={700} style={{ color: colors.accent.pink }}>{user?.hearts || 5}</Text>
              </Group>
            </Tooltip>
          </Group>
        </div>

        {/* Main content */}
        <div style={{ padding: '0 24px 24px' }}>
          {children}
        </div>
      </AppShell.Main>
    </AppShell>
  )
}
