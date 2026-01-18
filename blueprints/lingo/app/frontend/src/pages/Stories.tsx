import { useState, useEffect } from 'react'
import { Stack, Text, Group, Card, Center, Loader, Badge, SimpleGrid, Box, Progress } from '@mantine/core'
import { IconBook, IconClock, IconStar, IconCheck, IconLock } from '@tabler/icons-react'
import { useNavigate } from 'react-router-dom'
import { storiesApi, Story } from '../api/client'
import { useAuthStore } from '../stores/auth'
import { colors } from '../styles/tokens'

// Group stories by set_id
function groupStoriesBySet(stories: Story[]): Map<number, Story[]> {
  const groups = new Map<number, Story[]>()
  stories.forEach((story) => {
    const existing = groups.get(story.set_id) || []
    existing.push(story)
    groups.set(story.set_id, existing)
  })
  return groups
}

// Get difficulty label and color
function getDifficultyInfo(difficulty: number): { label: string; color: string } {
  if (difficulty <= 1) return { label: 'Beginner', color: '#58CC02' }
  if (difficulty <= 2) return { label: 'Easy', color: '#1CB0F6' }
  if (difficulty <= 3) return { label: 'Medium', color: '#FFC800' }
  if (difficulty <= 4) return { label: 'Hard', color: '#FF9600' }
  return { label: 'Expert', color: '#FF4B4B' }
}

// Story card component
function StoryCard({ story, isUnlocked = true }: { story: Story; isUnlocked?: boolean }) {
  const navigate = useNavigate()
  const { label: diffLabel, color: diffColor } = getDifficultyInfo(story.difficulty)
  const isCompleted = false // TODO: Get from user progress

  return (
    <Card
      shadow="sm"
      p="lg"
      radius="lg"
      withBorder
      onClick={() => isUnlocked && navigate(`/story/${story.id}`)}
      style={{
        cursor: isUnlocked ? 'pointer' : 'not-allowed',
        opacity: isUnlocked ? 1 : 0.6,
        border: `2px solid ${isCompleted ? '#58CC02' : '#E5E5E5'}`,
        backgroundColor: '#FFFFFF',
        transition: 'all 0.2s ease',
        position: 'relative',
        overflow: 'hidden',
      }}
      styles={{
        root: {
          '&:hover': isUnlocked ? {
            transform: 'translateY(-2px)',
            boxShadow: '0 8px 16px rgba(0,0,0,0.1)',
          } : {},
        },
      }}
    >
      {/* Illustration area */}
      <Box
        mb="md"
        style={{
          height: 120,
          backgroundColor: '#F7F7F7',
          borderRadius: 12,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          position: 'relative',
        }}
      >
        {story.illustration_url ? (
          <img
            src={story.illustration_url}
            alt={story.title}
            style={{
              width: '100%',
              height: '100%',
              objectFit: 'cover',
              borderRadius: 12,
            }}
          />
        ) : (
          <IconBook size={48} style={{ color: '#AFAFAF' }} />
        )}

        {/* Completed badge */}
        {isCompleted && (
          <Box
            style={{
              position: 'absolute',
              top: 8,
              right: 8,
              backgroundColor: '#58CC02',
              borderRadius: '50%',
              width: 28,
              height: 28,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
            }}
          >
            <IconCheck size={16} color="white" />
          </Box>
        )}

        {/* Lock overlay */}
        {!isUnlocked && (
          <Box
            style={{
              position: 'absolute',
              inset: 0,
              backgroundColor: 'rgba(0,0,0,0.3)',
              borderRadius: 12,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
            }}
          >
            <IconLock size={32} color="white" />
          </Box>
        )}
      </Box>

      {/* Story info */}
      <Stack gap="xs">
        <Text fw={700} size="md" lineClamp={2} style={{ color: colors.text.primary, minHeight: 44 }}>
          {story.title}
        </Text>

        {story.title_translation && (
          <Text size="sm" c="dimmed" lineClamp={1}>
            {story.title_translation}
          </Text>
        )}

        <Group justify="space-between" mt="xs">
          <Badge color={diffColor.replace('#', '')} variant="light" size="sm">
            {diffLabel}
          </Badge>

          <Group gap="xs">
            <Group gap={4}>
              <IconClock size={14} style={{ color: '#AFAFAF' }} />
              <Text size="xs" c="dimmed">
                {Math.ceil(story.duration_seconds / 60)}m
              </Text>
            </Group>

            <Group gap={4}>
              <IconStar size={14} style={{ color: '#FFC800' }} />
              <Text size="xs" c="dimmed">
                {story.xp_reward} XP
              </Text>
            </Group>
          </Group>
        </Group>
      </Stack>
    </Card>
  )
}

// Story set section
function StorySetSection({ setId, stories }: { setId: number; stories: Story[] }) {
  const completedCount = 0 // TODO: Get from user progress
  const totalCount = stories.length
  const progressPercent = totalCount > 0 ? (completedCount / totalCount) * 100 : 0

  return (
    <Box mb="xl">
      {/* Set header */}
      <Group justify="space-between" mb="md">
        <Group gap="md">
          <Box
            style={{
              width: 48,
              height: 48,
              backgroundColor: '#FF4B4B',
              borderRadius: 12,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
            }}
          >
            <Text fw={800} size="lg" c="white">
              {setId}
            </Text>
          </Box>
          <Stack gap={2}>
            <Text fw={700} size="lg" style={{ color: colors.text.primary }}>
              Set {setId}
            </Text>
            <Text size="sm" c="dimmed">
              {completedCount}/{totalCount} completed
            </Text>
          </Stack>
        </Group>

        {/* Progress bar */}
        <Box style={{ width: 120 }}>
          <Progress
            value={progressPercent}
            color="green"
            size="sm"
            radius="xl"
          />
        </Box>
      </Group>

      {/* Stories grid */}
      <SimpleGrid
        cols={{ base: 1, xs: 2, sm: 3, md: 4 }}
        spacing="md"
      >
        {stories.map((story) => (
          <StoryCard key={story.id} story={story} />
        ))}
      </SimpleGrid>
    </Box>
  )
}

export default function Stories() {
  const [stories, setStories] = useState<Story[]>([])
  const [loading, setLoading] = useState(true)
  const { user } = useAuthStore()

  useEffect(() => {
    loadStories()
  }, [user?.active_course_id])

  const loadStories = async () => {
    if (!user?.active_course_id) {
      setLoading(false)
      return
    }

    try {
      setLoading(true)
      const data = await storiesApi.getStories(user.active_course_id)
      setStories(data || [])
    } catch (err) {
      console.error('Failed to load stories:', err)
      setStories([])
    } finally {
      setLoading(false)
    }
  }

  // Group stories by set
  const storyGroups = groupStoriesBySet(stories)
  const sortedSetIds = Array.from(storyGroups.keys()).sort((a, b) => a - b)

  if (loading) {
    return (
      <Center h={400}>
        <Loader color="green" size="lg" />
      </Center>
    )
  }

  if (stories.length === 0) {
    return (
      <Center h={400}>
        <Stack align="center" gap="md">
          <IconBook size={64} style={{ color: '#AFAFAF' }} />
          <Text fw={700} size="xl" style={{ color: colors.text.primary }}>
            No stories available yet
          </Text>
          <Text c="dimmed" ta="center" maw={400}>
            Stories will be available as you progress through your course. Keep learning to unlock them!
          </Text>
        </Stack>
      </Center>
    )
  }

  return (
    <Box maw={1200} mx="auto">
      {/* Header */}
      <Stack gap="xs" mb="xl">
        <Text
          size="xl"
          fw={800}
          style={{
            color: colors.text.primary,
            fontFamily: '"Nunito", "DIN Round Pro", sans-serif',
          }}
        >
          Stories
        </Text>
        <Text c="dimmed" size="md">
          Practice reading and listening with fun, interactive stories
        </Text>
      </Stack>

      {/* Story sets */}
      {sortedSetIds.map((setId) => (
        <StorySetSection
          key={setId}
          setId={setId}
          stories={storyGroups.get(setId) || []}
        />
      ))}
    </Box>
  )
}
