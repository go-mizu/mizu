import { Container, Text, Paper, Group, Stack, Progress, Badge, Tabs } from '@mantine/core'
import { useState } from 'react'

interface Achievement {
  id: string
  name: string
  description: string
  icon: string
  category: string
  maxLevel: number
  currentLevel: number
  progress: number
  thresholds: number[]
}

const achievements: Achievement[] = [
  // Streak
  { id: 'wildfire', name: 'Wildfire', description: 'Reach a streak', icon: '🔥', category: 'streak', maxLevel: 10, currentLevel: 4, progress: 30, thresholds: [3, 7, 14, 30, 60, 90, 180, 365, 500, 1000] },
  { id: 'dedicated', name: 'Dedicated', description: 'Practice multiple days in a row', icon: '📅', category: 'streak', maxLevel: 5, currentLevel: 3, progress: 5, thresholds: [2, 3, 5, 7, 14] },

  // XP
  { id: 'xp_olympian', name: 'XP Olympian', description: 'Earn XP', icon: '⭐', category: 'xp', maxLevel: 10, currentLevel: 5, progress: 5000, thresholds: [100, 500, 1000, 2500, 5000, 10000, 15000, 20000, 25000, 30000] },
  { id: 'overtime', name: 'Overtime', description: 'Complete 7+ lessons in a day', icon: '⚡', category: 'xp', maxLevel: 5, currentLevel: 2, progress: 10, thresholds: [1, 5, 10, 25, 50] },

  // Learning
  { id: 'scholar', name: 'Scholar', description: 'Learn words', icon: '📚', category: 'learning', maxLevel: 10, currentLevel: 3, progress: 450, thresholds: [50, 100, 250, 500, 750, 1000, 1500, 2000, 3000, 5000] },
  { id: 'sage', name: 'Sage', description: 'Complete lessons', icon: '🎓', category: 'learning', maxLevel: 10, currentLevel: 4, progress: 120, thresholds: [10, 25, 50, 100, 200, 350, 500, 750, 1000, 1500] },
  { id: 'perfect', name: 'Perfectionist', description: 'Perfect lessons', icon: '💯', category: 'learning', maxLevel: 10, currentLevel: 2, progress: 15, thresholds: [1, 5, 10, 25, 50, 100, 200, 350, 500, 750] },
  { id: 'legendary', name: 'Legendary', description: 'Complete legendary levels', icon: '👑', category: 'learning', maxLevel: 10, currentLevel: 1, progress: 2, thresholds: [1, 5, 10, 25, 50, 100, 150, 200, 300, 500] },

  // Social
  { id: 'social_butterfly', name: 'Social Butterfly', description: 'Follow friends', icon: '🦋', category: 'social', maxLevel: 5, currentLevel: 2, progress: 5, thresholds: [1, 3, 5, 10, 20] },
  { id: 'quest_master', name: 'Quest Master', description: 'Complete friend quests', icon: '🎯', category: 'social', maxLevel: 10, currentLevel: 1, progress: 3, thresholds: [1, 5, 10, 25, 50, 100, 150, 200, 300, 500] },

  // League
  { id: 'winner', name: 'Winner', description: 'Win leagues', icon: '🏆', category: 'league', maxLevel: 10, currentLevel: 1, progress: 1, thresholds: [1, 3, 5, 10, 25, 50, 75, 100, 150, 200] },
  { id: 'champion', name: 'Champion', description: 'Reach Diamond', icon: '💎', category: 'league', maxLevel: 1, currentLevel: 0, progress: 0, thresholds: [1] },

  // Special
  { id: 'early_bird', name: 'Early Bird', description: 'Practice before 7 AM', icon: '🌅', category: 'special', maxLevel: 5, currentLevel: 1, progress: 3, thresholds: [1, 7, 30, 100, 365] },
  { id: 'night_owl', name: 'Night Owl', description: 'Practice after 10 PM', icon: '🦉', category: 'special', maxLevel: 5, currentLevel: 2, progress: 15, thresholds: [1, 7, 30, 100, 365] },
  { id: 'weekend_warrior', name: 'Weekend Warrior', description: 'Practice on weekends', icon: '🗓️', category: 'special', maxLevel: 10, currentLevel: 3, progress: 12, thresholds: [1, 4, 8, 16, 26, 52, 100, 150, 200, 300] },
]

const categories = [
  { value: 'all', label: 'All' },
  { value: 'streak', label: 'Streak' },
  { value: 'xp', label: 'XP' },
  { value: 'learning', label: 'Learning' },
  { value: 'social', label: 'Social' },
  { value: 'league', label: 'League' },
  { value: 'special', label: 'Special' },
]

function AchievementCard({ achievement }: { achievement: Achievement }) {
  const isMaxed = achievement.currentLevel >= achievement.maxLevel
  const nextThreshold = achievement.thresholds[achievement.currentLevel] || achievement.thresholds[achievement.thresholds.length - 1]
  const progressPercent = Math.min(100, (achievement.progress / nextThreshold) * 100)

  return (
    <Paper
      p="lg"
      radius="lg"
      style={{
        backgroundColor: isMaxed ? '#3f9a02' : '#1a2c33',
        border: isMaxed ? '2px solid #ffc800' : 'none',
      }}
    >
      <Group>
        <div
          style={{
            width: 60,
            height: 60,
            borderRadius: '50%',
            backgroundColor: isMaxed ? '#ffc800' : '#233a42',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            fontSize: 28,
          }}
        >
          {achievement.icon}
        </div>
        <div style={{ flex: 1 }}>
          <Group justify="space-between" mb={4}>
            <Text fw={700} style={{ color: 'white' }}>{achievement.name}</Text>
            <Badge
              color={isMaxed ? 'yellow' : 'gray'}
              variant={isMaxed ? 'filled' : 'outline'}
            >
              Level {achievement.currentLevel}/{achievement.maxLevel}
            </Badge>
          </Group>
          <Text size="sm" style={{ color: '#8fa8b2' }} mb="sm">
            {achievement.description}
          </Text>
          {!isMaxed && (
            <>
              <Progress
                value={progressPercent}
                size="sm"
                radius="xl"
                color={progressPercent >= 100 ? 'green' : 'blue'}
              />
              <Text size="xs" style={{ color: '#8fa8b2' }} mt={4}>
                {achievement.progress} / {nextThreshold}
              </Text>
            </>
          )}
          {isMaxed && (
            <Badge color="yellow" variant="light">
              ✨ MAXED OUT
            </Badge>
          )}
        </div>
      </Group>
    </Paper>
  )
}

export default function Achievements() {
  const [category, setCategory] = useState('all')

  const filteredAchievements = category === 'all'
    ? achievements
    : achievements.filter((a) => a.category === category)

  const totalUnlocked = achievements.filter((a) => a.currentLevel > 0).length
  const totalMaxed = achievements.filter((a) => a.currentLevel >= a.maxLevel).length

  return (
    <Container size="md">
      {/* Stats Overview */}
      <Paper
        p="xl"
        radius="lg"
        mb="xl"
        style={{
          backgroundColor: '#1a2c33',
        }}
      >
        <Group justify="space-around">
          <Stack align="center" gap="xs">
            <Text size="2.5rem" fw={800} style={{ color: '#58cc02' }}>
              {totalUnlocked}
            </Text>
            <Text style={{ color: '#8fa8b2' }}>Unlocked</Text>
          </Stack>
          <Stack align="center" gap="xs">
            <Text size="2.5rem" fw={800} style={{ color: '#ffc800' }}>
              {totalMaxed}
            </Text>
            <Text style={{ color: '#8fa8b2' }}>Maxed Out</Text>
          </Stack>
          <Stack align="center" gap="xs">
            <Text size="2.5rem" fw={800} style={{ color: '#1cb0f6' }}>
              {achievements.length}
            </Text>
            <Text style={{ color: '#8fa8b2' }}>Total</Text>
          </Stack>
        </Group>
      </Paper>

      {/* Category Tabs */}
      <Tabs value={category} onChange={(v) => setCategory(v || 'all')} mb="xl">
        <Tabs.List grow>
          {categories.map((cat) => (
            <Tabs.Tab
              key={cat.value}
              value={cat.value}
              style={{
                color: category === cat.value ? '#58cc02' : '#8fa8b2',
                fontWeight: 600,
              }}
            >
              {cat.label}
            </Tabs.Tab>
          ))}
        </Tabs.List>
      </Tabs>

      {/* Achievements List */}
      <Stack gap="md">
        {filteredAchievements.map((achievement) => (
          <AchievementCard key={achievement.id} achievement={achievement} />
        ))}
      </Stack>
    </Container>
  )
}
