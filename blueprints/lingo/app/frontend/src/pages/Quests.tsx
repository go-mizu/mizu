import { Container, Title, Text, Paper, Group, Stack, Progress, Badge, Button } from '@mantine/core'
import { IconTarget, IconClock, IconGift, IconChevronRight } from '@tabler/icons-react'
import { colors } from '../styles/tokens'

interface Quest {
  id: string
  title: string
  description: string
  icon: string
  progress: number
  target: number
  xpReward: number
  gemsReward?: number
  type: 'daily' | 'weekly' | 'friend'
  expiresIn?: string
}

const dailyQuests: Quest[] = [
  {
    id: '1',
    title: 'Earn 20 XP',
    description: 'Complete lessons to earn XP',
    icon: '⚡',
    progress: 12,
    target: 20,
    xpReward: 10,
    type: 'daily',
    expiresIn: '8 hours',
  },
  {
    id: '2',
    title: 'Complete 3 lessons',
    description: 'Finish any 3 lessons',
    icon: '📚',
    progress: 1,
    target: 3,
    xpReward: 15,
    type: 'daily',
    expiresIn: '8 hours',
  },
  {
    id: '3',
    title: 'Perfect lesson',
    description: 'Complete a lesson with no mistakes',
    icon: '🎯',
    progress: 0,
    target: 1,
    xpReward: 20,
    gemsReward: 5,
    type: 'daily',
    expiresIn: '8 hours',
  },
]

const weeklyQuests: Quest[] = [
  {
    id: '4',
    title: 'Earn 200 XP',
    description: 'Earn XP this week',
    icon: '🌟',
    progress: 85,
    target: 200,
    xpReward: 50,
    gemsReward: 20,
    type: 'weekly',
    expiresIn: '4 days',
  },
  {
    id: '5',
    title: '5 day streak',
    description: 'Practice 5 days in a row',
    icon: '🔥',
    progress: 3,
    target: 5,
    xpReward: 30,
    gemsReward: 15,
    type: 'weekly',
    expiresIn: '4 days',
  },
]

const friendQuests: Quest[] = [
  {
    id: '6',
    title: 'Challenge a friend',
    description: 'Beat a friend in XP this week',
    icon: '👥',
    progress: 0,
    target: 1,
    xpReward: 50,
    gemsReward: 100,
    type: 'friend',
  },
]

function QuestCard({ quest }: { quest: Quest }) {
  const progressPercent = (quest.progress / quest.target) * 100
  const isComplete = quest.progress >= quest.target

  return (
    <Paper
      p="lg"
      radius="lg"
      style={{
        backgroundColor: isComplete ? colors.primary.greenLight : colors.neutral.white,
        border: isComplete ? `2px solid ${colors.primary.green}` : `2px solid ${colors.neutral.border}`,
      }}
    >
      <Group justify="space-between" wrap="nowrap">
        <Group gap="md" wrap="nowrap">
          <div
            style={{
              width: 56,
              height: 56,
              borderRadius: '50%',
              backgroundColor: isComplete ? colors.primary.green : colors.neutral.background,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              fontSize: 28,
            }}
          >
            {quest.icon}
          </div>
          <div style={{ flex: 1 }}>
            <Text fw={700} style={{ color: colors.text.primary }}>{quest.title}</Text>
            <Text size="sm" style={{ color: colors.text.secondary }}>{quest.description}</Text>
            {!isComplete && (
              <>
                <Progress
                  value={progressPercent}
                  size="sm"
                  radius="xl"
                  color="green"
                  mt="xs"
                />
                <Text size="xs" mt={4} style={{ color: colors.text.muted }}>
                  {quest.progress}/{quest.target}
                </Text>
              </>
            )}
          </div>
        </Group>

        <Stack align="center" gap="xs">
          {isComplete ? (
            <Button color="green" radius="xl" size="sm">
              Claim
            </Button>
          ) : (
            <Badge color="yellow" size="lg">
              +{quest.xpReward} XP
              {quest.gemsReward && ` +${quest.gemsReward} 💎`}
            </Badge>
          )}
          {quest.expiresIn && !isComplete && (
            <Group gap={4}>
              <IconClock size={14} style={{ color: colors.text.muted }} />
              <Text size="xs" style={{ color: colors.text.muted }}>{quest.expiresIn}</Text>
            </Group>
          )}
        </Stack>
      </Group>
    </Paper>
  )
}

export default function Quests() {
  return (
    <Container size="md">
      {/* Header */}
      <Paper
        p="xl"
        radius="lg"
        mb="xl"
        style={{
          backgroundColor: colors.accent.orange,
          textAlign: 'center',
        }}
      >
        <IconTarget size={48} style={{ color: 'white' }} />
        <Title order={2} mt="md" style={{ color: 'white' }}>Daily Quests</Title>
        <Text style={{ color: 'rgba(255,255,255,0.8)' }}>
          Complete quests to earn XP and gems
        </Text>
      </Paper>

      {/* Daily Quests */}
      <Group justify="space-between" mb="md">
        <Title order={3} style={{ color: colors.text.primary }}>Daily Quests</Title>
        <Badge color="orange" size="lg" leftSection={<IconClock size={14} />}>
          Resets in 8 hours
        </Badge>
      </Group>
      <Stack gap="md" mb="xl">
        {dailyQuests.map((quest) => (
          <QuestCard key={quest.id} quest={quest} />
        ))}
      </Stack>

      {/* Weekly Quests */}
      <Group justify="space-between" mb="md">
        <Title order={3} style={{ color: colors.text.primary }}>Weekly Quests</Title>
        <Badge color="blue" size="lg" leftSection={<IconClock size={14} />}>
          4 days left
        </Badge>
      </Group>
      <Stack gap="md" mb="xl">
        {weeklyQuests.map((quest) => (
          <QuestCard key={quest.id} quest={quest} />
        ))}
      </Stack>

      {/* Friend Quests */}
      <Group justify="space-between" mb="md">
        <Title order={3} style={{ color: colors.text.primary }}>Friend Quests</Title>
        <Badge color="purple" size="lg" leftSection={<IconGift size={14} />}>
          Bonus rewards
        </Badge>
      </Group>
      <Stack gap="md" mb="xl">
        {friendQuests.map((quest) => (
          <QuestCard key={quest.id} quest={quest} />
        ))}
      </Stack>

      {/* Invite friends CTA */}
      <Paper
        p="lg"
        radius="lg"
        style={{
          backgroundColor: colors.secondary.blueLight,
          border: `2px solid ${colors.secondary.blue}`,
        }}
      >
        <Group justify="space-between">
          <div>
            <Text fw={700} style={{ color: colors.secondary.blue }}>Invite Friends</Text>
            <Text size="sm" style={{ color: colors.text.secondary }}>
              Complete friend quests together for bonus rewards!
            </Text>
          </div>
          <Button
            color="blue"
            radius="xl"
            rightSection={<IconChevronRight size={16} />}
          >
            Invite
          </Button>
        </Group>
      </Paper>
    </Container>
  )
}
