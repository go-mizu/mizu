import { Container, Title, Text, Paper, Group, Stack, Avatar, Badge, Button, Grid, Progress, ActionIcon } from '@mantine/core'
import { IconFlame, IconStar, IconSettings, IconLogout, IconEdit } from '@tabler/icons-react'
import { useNavigate } from 'react-router-dom'
import { useAuthStore } from '../stores/auth'

const mockStats = {
  wordsLearned: 450,
  lessonsCompleted: 120,
  totalXP: 5000,
  currentStreak: 30,
  longestStreak: 45,
  coursesStarted: 2,
  achievementsUnlocked: 12,
  timeSpentMinutes: 1800,
}

const recentActivity = [
  { day: 'Mon', xp: 150 },
  { day: 'Tue', xp: 200 },
  { day: 'Wed', xp: 180 },
  { day: 'Thu', xp: 220 },
  { day: 'Fri', xp: 100 },
  { day: 'Sat', xp: 300 },
  { day: 'Sun', xp: 250 },
]

export default function Profile() {
  const navigate = useNavigate()
  const { user, logout } = useAuthStore()

  const handleLogout = () => {
    logout()
    navigate('/')
  }

  return (
    <Container size="md">
      {/* Profile Header */}
      <Paper p="xl" radius="lg" mb="xl" style={{ backgroundColor: '#FFFFFF' }}>
        <Group justify="space-between" align="start">
          <Group>
            <Avatar size={100} radius="xl" color="green" style={{ fontSize: 36 }}>
              {user?.username?.charAt(0).toUpperCase()}
            </Avatar>
            <div>
              <Group gap="xs">
                <Title order={2} style={{ color: '#4B4B4B' }}>
                  {user?.display_name || user?.username}
                </Title>
                {user?.is_premium && (
                  <Badge color="yellow" variant="filled">Super</Badge>
                )}
              </Group>
              <Text style={{ color: '#777777' }}>@{user?.username}</Text>
              <Group gap="lg" mt="md">
                <Group gap={4}>
                  <IconFlame size={20} style={{ color: '#ff9600' }} />
                  <Text fw={600} style={{ color: '#ff9600' }}>{user?.streak_days || 0}</Text>
                </Group>
                <Group gap={4}>
                  <IconStar size={20} style={{ color: '#ffc800' }} />
                  <Text fw={600} style={{ color: '#ffc800' }}>{user?.xp_total || 0} XP</Text>
                </Group>
              </Group>
            </div>
          </Group>
          <Group>
            <ActionIcon variant="subtle" size="lg" radius="xl">
              <IconEdit size={20} style={{ color: '#777777' }} />
            </ActionIcon>
            <ActionIcon variant="subtle" size="lg" radius="xl">
              <IconSettings size={20} style={{ color: '#777777' }} />
            </ActionIcon>
          </Group>
        </Group>
      </Paper>

      {/* Stats Grid */}
      <Grid mb="xl">
        <Grid.Col span={4}>
          <Paper p="lg" radius="lg" style={{ backgroundColor: '#FFFFFF', textAlign: 'center' }}>
            <Text size="2rem" fw={800} style={{ color: '#58cc02' }}>
              {mockStats.wordsLearned}
            </Text>
            <Text size="sm" style={{ color: '#777777' }}>Words Learned</Text>
          </Paper>
        </Grid.Col>
        <Grid.Col span={4}>
          <Paper p="lg" radius="lg" style={{ backgroundColor: '#FFFFFF', textAlign: 'center' }}>
            <Text size="2rem" fw={800} style={{ color: '#1cb0f6' }}>
              {mockStats.lessonsCompleted}
            </Text>
            <Text size="sm" style={{ color: '#777777' }}>Lessons Completed</Text>
          </Paper>
        </Grid.Col>
        <Grid.Col span={4}>
          <Paper p="lg" radius="lg" style={{ backgroundColor: '#FFFFFF', textAlign: 'center' }}>
            <Text size="2rem" fw={800} style={{ color: '#ffc800' }}>
              {Math.floor(mockStats.timeSpentMinutes / 60)}h
            </Text>
            <Text size="sm" style={{ color: '#777777' }}>Time Learning</Text>
          </Paper>
        </Grid.Col>
      </Grid>

      {/* Weekly Activity */}
      <Paper p="xl" radius="lg" mb="xl" style={{ backgroundColor: '#FFFFFF' }}>
        <Title order={4} mb="lg" style={{ color: '#4B4B4B' }}>This Week</Title>
        <Group justify="space-between">
          {recentActivity.map((day) => (
            <Stack key={day.day} gap="xs" align="center">
              <div style={{ height: 100, display: 'flex', alignItems: 'flex-end' }}>
                <div
                  style={{
                    width: 30,
                    height: `${(day.xp / 300) * 100}%`,
                    backgroundColor: day.xp > 0 ? '#58cc02' : '#E5E5E5',
                    borderRadius: 4,
                    minHeight: 4,
                  }}
                />
              </div>
              <Text size="xs" fw={600} style={{ color: '#777777' }}>{day.day}</Text>
              <Text size="xs" style={{ color: '#58cc02' }}>{day.xp}</Text>
            </Stack>
          ))}
        </Group>
      </Paper>

      {/* Achievements Preview */}
      <Paper p="xl" radius="lg" mb="xl" style={{ backgroundColor: '#FFFFFF' }}>
        <Group justify="space-between" mb="lg">
          <Title order={4} style={{ color: '#4B4B4B' }}>Achievements</Title>
          <Button variant="subtle" color="blue" onClick={() => navigate('/achievements')}>
            View All
          </Button>
        </Group>
        <Group>
          {['🔥', '⭐', '🏆', '📚', '💎'].map((emoji, i) => (
            <Paper
              key={i}
              p="md"
              radius="lg"
              style={{
                backgroundColor: '#F7F7F7',
                width: 60,
                height: 60,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
              }}
            >
              <Text size="xl">{emoji}</Text>
            </Paper>
          ))}
        </Group>
      </Paper>

      {/* Streak Info */}
      <Paper p="xl" radius="lg" mb="xl" style={{ backgroundColor: '#FFFFFF' }}>
        <Group justify="space-between">
          <div>
            <Title order={4} style={{ color: '#4B4B4B' }}>Current Streak</Title>
            <Text style={{ color: '#777777' }}>Keep learning every day!</Text>
          </div>
          <Group>
            <IconFlame size={40} style={{ color: '#ff9600' }} />
            <Text size="2rem" fw={800} style={{ color: '#ff9600' }}>
              {user?.streak_days || 0}
            </Text>
          </Group>
        </Group>
        <Progress value={((user?.streak_days || 0) / 365) * 100} mt="lg" size="lg" radius="xl" color="orange" />
        <Text size="sm" ta="center" mt="sm" style={{ color: '#777777' }}>
          {365 - (user?.streak_days || 0)} days until 1 year streak!
        </Text>
      </Paper>

      {/* Logout Button */}
      <Button
        fullWidth
        variant="outline"
        color="red"
        size="lg"
        leftSection={<IconLogout size={20} />}
        onClick={handleLogout}
      >
        Log Out
      </Button>
    </Container>
  )
}
