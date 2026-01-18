import { Container, Title, Text, Paper, Group, Stack, Avatar, Badge, SegmentedControl } from '@mantine/core'
import { IconTrophy, IconChevronUp, IconChevronDown, IconMinus } from '@tabler/icons-react'
import { useState } from 'react'
import { useAuthStore } from '../stores/auth'

interface LeaderboardEntry {
  rank: number
  username: string
  displayName: string
  xp: number
  streak: number
  change: 'up' | 'down' | 'same'
}

const mockLeaderboard: LeaderboardEntry[] = [
  { rank: 1, username: 'polyglot_master', displayName: 'Polyglot Master', xp: 2500, streak: 120, change: 'same' },
  { rank: 2, username: 'lingua_lover', displayName: 'Lingua Lover', xp: 2350, streak: 90, change: 'up' },
  { rank: 3, username: 'word_wizard', displayName: 'Word Wizard', xp: 2200, streak: 75, change: 'up' },
  { rank: 4, username: 'demo', displayName: 'Demo User', xp: 1800, streak: 30, change: 'down' },
  { rank: 5, username: 'spanish_star', displayName: 'Spanish Star', xp: 1650, streak: 45, change: 'up' },
  { rank: 6, username: 'french_fan', displayName: 'French Fan', xp: 1500, streak: 28, change: 'same' },
  { rank: 7, username: 'german_guru', displayName: 'German Guru', xp: 1400, streak: 35, change: 'down' },
  { rank: 8, username: 'japanese_joy', displayName: 'Japanese Joy', xp: 1300, streak: 22, change: 'up' },
]

const leagues = [
  { id: 1, name: 'Bronze', color: '#CD7F32' },
  { id: 2, name: 'Silver', color: '#C0C0C0' },
  { id: 3, name: 'Gold', color: '#FFD700' },
  { id: 4, name: 'Sapphire', color: '#0F52BA' },
  { id: 5, name: 'Ruby', color: '#E0115F' },
  { id: 6, name: 'Emerald', color: '#50C878' },
  { id: 7, name: 'Amethyst', color: '#9966CC' },
  { id: 8, name: 'Pearl', color: '#FDEEF4' },
  { id: 9, name: 'Obsidian', color: '#3D3D3D' },
  { id: 10, name: 'Diamond', color: '#B9F2FF' },
]

export default function Leaderboards() {
  const { user } = useAuthStore()
  const [tab, setTab] = useState<string>('league')
  const currentLeague = leagues[2] // Gold for demo

  return (
    <Container size="md">
      {/* Current League */}
      <Paper
        p="xl"
        radius="lg"
        mb="xl"
        style={{
          backgroundColor: '#FFFFFF',
          textAlign: 'center',
        }}
      >
        <Stack align="center" gap="md">
          <div
            style={{
              width: 100,
              height: 100,
              borderRadius: '50%',
              backgroundColor: currentLeague.color,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              boxShadow: `0 0 20px ${currentLeague.color}40`,
            }}
          >
            <IconTrophy size={48} style={{ color: '#4B4B4B' }} />
          </div>
          <div>
            <Title order={2} style={{ color: currentLeague.color }}>
              {currentLeague.name} League
            </Title>
            <Text style={{ color: '#777777' }}>
              Top 10 advance to Sapphire League
            </Text>
          </div>
          <Badge size="lg" color="blue">
            4 days remaining
          </Badge>
        </Stack>
      </Paper>

      {/* Tab Selection */}
      <SegmentedControl
        fullWidth
        mb="xl"
        value={tab}
        onChange={setTab}
        data={[
          { label: 'League', value: 'league' },
          { label: 'Friends', value: 'friends' },
        ]}
        styles={{
          root: { backgroundColor: '#FFFFFF' },
          indicator: { backgroundColor: '#58cc02' },
          label: { color: '#777777', fontWeight: 600 },
        }}
      />

      {/* Leaderboard */}
      <Stack gap="md">
        {mockLeaderboard.map((entry) => {
          const isCurrentUser = entry.username === user?.username || entry.username === 'demo'
          const isTop3 = entry.rank <= 3
          const rankColors = ['#FFD700', '#C0C0C0', '#CD7F32']

          return (
            <Paper
              key={entry.username}
              p="md"
              radius="lg"
              style={{
                backgroundColor: isCurrentUser ? '#F7F7F7' : '#FFFFFF',
                border: isCurrentUser ? '2px solid #58cc02' : 'none',
              }}
            >
              <Group justify="space-between">
                <Group>
                  {/* Rank */}
                  <div
                    style={{
                      width: 40,
                      height: 40,
                      borderRadius: '50%',
                      backgroundColor: isTop3 ? rankColors[entry.rank - 1] : '#E5E5E5',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                    }}
                  >
                    <Text fw={700} style={{ color: '#4B4B4B' }}>
                      {entry.rank}
                    </Text>
                  </div>

                  {/* Avatar & Name */}
                  <Avatar size="md" radius="xl" color="green">
                    {entry.displayName.charAt(0)}
                  </Avatar>
                  <div>
                    <Text fw={700} style={{ color: '#4B4B4B' }}>
                      {entry.displayName}
                      {isCurrentUser && <Badge ml="xs" size="xs" color="green">You</Badge>}
                    </Text>
                    <Text size="sm" style={{ color: '#777777' }}>
                      @{entry.username}
                    </Text>
                  </div>
                </Group>

                {/* XP & Change */}
                <Group>
                  <Text fw={700} style={{ color: '#ffc800' }}>
                    {entry.xp.toLocaleString()} XP
                  </Text>
                  {entry.change === 'up' && <IconChevronUp size={20} style={{ color: '#58cc02' }} />}
                  {entry.change === 'down' && <IconChevronDown size={20} style={{ color: '#ff4b4b' }} />}
                  {entry.change === 'same' && <IconMinus size={20} style={{ color: '#777777' }} />}
                </Group>
              </Group>
            </Paper>
          )
        })}
      </Stack>

      {/* Promotion Zone Info */}
      <Paper
        p="md"
        radius="lg"
        mt="xl"
        style={{
          backgroundColor: '#F7F7F7',
          border: '2px dashed #58cc02',
        }}
      >
        <Group justify="center" gap="xl">
          <div style={{ textAlign: 'center' }}>
            <IconChevronUp size={24} style={{ color: '#58cc02' }} />
            <Text size="sm" fw={600} style={{ color: '#58cc02' }}>Top 10</Text>
            <Text size="xs" style={{ color: '#777777' }}>Promotion Zone</Text>
          </div>
          <div style={{ textAlign: 'center' }}>
            <IconChevronDown size={24} style={{ color: '#ff4b4b' }} />
            <Text size="sm" fw={600} style={{ color: '#ff4b4b' }}>Bottom 5</Text>
            <Text size="xs" style={{ color: '#777777' }}>Demotion Zone</Text>
          </div>
        </Group>
      </Paper>
    </Container>
  )
}
