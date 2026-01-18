import { useState, useEffect } from 'react'
import { Title, Text, Paper, Group, Stack, Badge, ActionIcon, Tooltip, Loader, Center, Button, Progress } from '@mantine/core'
import { IconLock, IconCheck, IconBook, IconFlame, IconStar, IconCrown, IconChevronRight, IconPlayerPlay } from '@tabler/icons-react'
import { useNavigate } from 'react-router-dom'
import { motion } from 'framer-motion'
import { useAuthStore } from '../stores/auth'
import { colors } from '../styles/tokens'
import { coursesApi, Unit, Skill } from '../api/client'

interface SkillWithProgress extends Skill {
  crownLevel: number
  isLocked: boolean
  isComplete: boolean
  isLegendary: boolean
}

interface UnitWithProgress extends Omit<Unit, 'skills'> {
  skills: SkillWithProgress[]
}

// Winding path positions - creates a serpentine pattern
const getSkillPosition = (index: number): 'left' | 'center' | 'right' => {
  const pattern: ('left' | 'center' | 'right')[] = ['center', 'right', 'center', 'left', 'center', 'right']
  return pattern[index % pattern.length]
}

const getHorizontalOffset = (position: 'left' | 'center' | 'right'): number => {
  switch (position) {
    case 'left': return -80
    case 'right': return 80
    default: return 0
  }
}

interface SkillNodeProps {
  skill: SkillWithProgress
  index: number
  onClick: () => void
  isFirst: boolean
}

function SkillNode({ skill, index, onClick, isFirst }: SkillNodeProps) {
  const position = getSkillPosition(index)
  const offsetX = getHorizontalOffset(position)

  const getBackgroundColor = () => {
    if (skill.isLegendary) return colors.accent.yellow
    if (skill.isComplete) return colors.accent.yellow
    if (skill.isLocked) return '#E5E5E5'
    return colors.primary.green
  }

  const getShadowColor = () => {
    if (skill.isLegendary) return '0 6px 0 #C9A000'
    if (skill.isComplete) return '0 6px 0 #C9A000'
    if (skill.isLocked) return 'none'
    return '0 6px 0 #58A700'
  }

  const getIconColor = () => {
    if (skill.isLocked) return '#AFAFAF'
    return 'white'
  }

  return (
    <motion.div
      initial={{ opacity: 0, scale: 0.8, y: 20 }}
      animate={{ opacity: 1, scale: 1, y: 0 }}
      transition={{ delay: index * 0.05, duration: 0.3 }}
      style={{
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        transform: `translateX(${offsetX}px)`,
        marginBottom: 16,
      }}
    >
      <Tooltip
        label={skill.isLocked ? 'Complete previous skills to unlock' : skill.name}
        withArrow
        position="right"
      >
        <motion.div
          whileHover={{ scale: skill.isLocked ? 1 : 1.08 }}
          whileTap={{ scale: skill.isLocked ? 1 : 0.95 }}
          style={{ position: 'relative' }}
        >
          <Paper
            onClick={skill.isLocked ? undefined : onClick}
            style={{
              width: 80,
              height: 80,
              borderRadius: '50%',
              backgroundColor: getBackgroundColor(),
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              cursor: skill.isLocked ? 'not-allowed' : 'pointer',
              boxShadow: getShadowColor(),
              transition: 'all 0.15s ease',
              border: skill.isLocked ? '4px solid #CDCDCD' : 'none',
            }}
          >
            {skill.isLocked ? (
              <IconLock size={36} style={{ color: getIconColor() }} />
            ) : skill.isLegendary ? (
              <IconCrown size={36} style={{ color: getIconColor() }} />
            ) : (
              <IconStar size={36} style={{ color: getIconColor() }} />
            )}
          </Paper>

          {/* Crown level indicator */}
          {!skill.isLocked && skill.crownLevel > 0 && (
            <Badge
              size="md"
              style={{
                position: 'absolute',
                bottom: -10,
                left: '50%',
                transform: 'translateX(-50%)',
                backgroundColor: skill.isComplete ? colors.accent.yellow : colors.primary.green,
                color: 'white',
                fontWeight: 700,
                border: '2px solid white',
                boxShadow: '0 2px 4px rgba(0,0,0,0.1)',
              }}
            >
              {skill.crownLevel}/{skill.levels}
            </Badge>
          )}

          {/* Completion checkmark */}
          {skill.isComplete && (
            <IconCheck
              size={22}
              style={{
                position: 'absolute',
                top: -6,
                right: -6,
                backgroundColor: colors.accent.yellow,
                borderRadius: '50%',
                padding: 3,
                color: 'white',
                border: '2px solid white',
              }}
            />
          )}
        </motion.div>
      </Tooltip>

      {/* Skill name */}
      <Text
        ta="center"
        mt="md"
        size="sm"
        fw={700}
        style={{
          color: skill.isLocked ? colors.text.muted : colors.text.primary,
          maxWidth: 100,
        }}
      >
        {skill.name}
      </Text>

      {/* Start button for first unlocked skill */}
      {isFirst && !skill.isLocked && !skill.isComplete && (
        <motion.div
          initial={{ opacity: 0, y: 10 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.3 }}
        >
          <Button
            onClick={onClick}
            color="green"
            radius="xl"
            size="md"
            mt="sm"
            leftSection={<IconPlayerPlay size={18} />}
            style={{
              boxShadow: '0 4px 0 #58A700',
              fontWeight: 700,
              textTransform: 'uppercase',
              letterSpacing: '0.5px',
            }}
          >
            Start
          </Button>
        </motion.div>
      )}
    </motion.div>
  )
}

function UnitHeader({ unit, unitIndex, onJump }: { unit: UnitWithProgress; unitIndex: number; onJump: () => void }) {
  const completedSkills = unit.skills.filter(s => s.isComplete).length
  const totalSkills = unit.skills.length
  const progress = (completedSkills / totalSkills) * 100

  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ delay: unitIndex * 0.1 }}
    >
      <Paper
        p="lg"
        radius="xl"
        mb="xl"
        style={{
          backgroundColor: unitIndex === 0 ? colors.primary.green : colors.neutral.white,
          border: unitIndex === 0 ? 'none' : `2px solid ${colors.neutral.border}`,
          overflow: 'hidden',
        }}
      >
        <Group justify="space-between" align="center" wrap="nowrap">
          <div style={{ flex: 1 }}>
            <Group gap="xs" mb={4}>
              <Badge
                size="sm"
                variant={unitIndex === 0 ? 'filled' : 'light'}
                color={unitIndex === 0 ? 'white' : 'gray'}
                style={{
                  backgroundColor: unitIndex === 0 ? 'rgba(255,255,255,0.2)' : undefined,
                  color: unitIndex === 0 ? 'white' : colors.text.secondary,
                }}
              >
                UNIT {unitIndex + 1}
              </Badge>
            </Group>
            <Title
              order={4}
              style={{ color: unitIndex === 0 ? 'white' : colors.text.primary }}
              mb={4}
            >
              {unit.title}
            </Title>
            <Text size="sm" style={{ color: unitIndex === 0 ? 'rgba(255,255,255,0.8)' : colors.text.secondary }}>
              {unit.description || `${totalSkills} skills`}
            </Text>

            {/* Progress bar */}
            <Progress
              value={progress}
              size="sm"
              radius="xl"
              mt="md"
              color={unitIndex === 0 ? 'white' : 'green'}
              style={{
                backgroundColor: unitIndex === 0 ? 'rgba(255,255,255,0.2)' : colors.neutral.border,
              }}
            />
            <Text size="xs" mt={4} style={{ color: unitIndex === 0 ? 'rgba(255,255,255,0.8)' : colors.text.muted }}>
              {completedSkills}/{totalSkills} completed
            </Text>
          </div>

          {/* Guidebook / Jump button */}
          <Stack gap="xs" align="center">
            {unit.guidebook_content && (
              <ActionIcon
                variant="light"
                size="lg"
                radius="xl"
                style={{
                  backgroundColor: unitIndex === 0 ? 'rgba(255,255,255,0.2)' : colors.secondary.blueLight,
                  color: unitIndex === 0 ? 'white' : colors.secondary.blue,
                }}
              >
                <IconBook size={20} />
              </ActionIcon>
            )}
            {unitIndex > 0 && (
              <ActionIcon
                variant="light"
                size="lg"
                radius="xl"
                onClick={onJump}
                style={{
                  backgroundColor: colors.secondary.blueLight,
                  color: colors.secondary.blue,
                }}
              >
                <IconChevronRight size={20} />
              </ActionIcon>
            )}
          </Stack>
        </Group>
      </Paper>
    </motion.div>
  )
}

// Right sidebar components
function DailyQuestsCard() {
  return (
    <Paper p="lg" radius="lg" style={{ backgroundColor: colors.neutral.white }}>
      <Group justify="space-between" mb="md">
        <Text fw={700} style={{ color: colors.text.primary }}>Daily Quests</Text>
        <Badge color="orange">3 active</Badge>
      </Group>
      <Stack gap="sm">
        <Paper p="sm" radius="md" style={{ backgroundColor: colors.neutral.background }}>
          <Group justify="space-between">
            <Group gap="xs">
              <Text>⚡</Text>
              <Text size="sm" style={{ color: colors.text.primary }}>Earn 20 XP</Text>
            </Group>
            <Badge color="green" size="sm">12/20</Badge>
          </Group>
          <Progress value={60} size="xs" radius="xl" color="green" mt="xs" />
        </Paper>
        <Paper p="sm" radius="md" style={{ backgroundColor: colors.neutral.background }}>
          <Group justify="space-between">
            <Group gap="xs">
              <Text>🎯</Text>
              <Text size="sm" style={{ color: colors.text.primary }}>Complete 3 lessons</Text>
            </Group>
            <Badge color="green" size="sm">1/3</Badge>
          </Group>
          <Progress value={33} size="xs" radius="xl" color="green" mt="xs" />
        </Paper>
        <Paper p="sm" radius="md" style={{ backgroundColor: colors.neutral.background }}>
          <Group justify="space-between">
            <Group gap="xs">
              <Text>🔥</Text>
              <Text size="sm" style={{ color: colors.text.primary }}>Perfect lesson</Text>
            </Group>
            <Badge color="gray" size="sm">0/1</Badge>
          </Group>
          <Progress value={0} size="xs" radius="xl" color="gray" mt="xs" />
        </Paper>
      </Stack>
    </Paper>
  )
}

function LeaderboardPreview() {
  const leaders = [
    { name: 'Maria', xp: 2500, avatar: 'M' },
    { name: 'John', xp: 2350, avatar: 'J' },
    { name: 'You', xp: 1800, avatar: 'Y', isYou: true },
  ]

  return (
    <Paper p="lg" radius="lg" style={{ backgroundColor: colors.neutral.white }}>
      <Group justify="space-between" mb="md">
        <Text fw={700} style={{ color: colors.text.primary }}>Gold League</Text>
        <Badge color="yellow">4 days left</Badge>
      </Group>
      <Stack gap="xs">
        {leaders.map((leader, i) => (
          <Group key={i} justify="space-between" p="xs" style={{
            backgroundColor: leader.isYou ? colors.primary.greenLight : 'transparent',
            borderRadius: 8,
          }}>
            <Group gap="xs">
              <Text fw={700} size="sm" style={{ color: colors.text.secondary, width: 20 }}>
                {i + 1}
              </Text>
              <div style={{
                width: 32,
                height: 32,
                borderRadius: '50%',
                backgroundColor: colors.primary.green,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                color: 'white',
                fontWeight: 700,
                fontSize: 14,
              }}>
                {leader.avatar}
              </div>
              <Text size="sm" fw={600} style={{ color: colors.text.primary }}>
                {leader.name}
                {leader.isYou && <Badge ml="xs" size="xs" color="green">You</Badge>}
              </Text>
            </Group>
            <Text size="sm" fw={700} style={{ color: colors.accent.yellow }}>
              {leader.xp} XP
            </Text>
          </Group>
        ))}
      </Stack>
      <Button fullWidth variant="subtle" color="blue" mt="md" size="sm">
        View Full Leaderboard
      </Button>
    </Paper>
  )
}

export default function Home() {
  const navigate = useNavigate()
  const { user } = useAuthStore()
  const [units, setUnits] = useState<UnitWithProgress[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [expandedUnits, setExpandedUnits] = useState<Set<number>>(new Set([0]))

  useEffect(() => {
    async function loadCourseData() {
      try {
        setLoading(true)
        const courses = await coursesApi.listCourses('en')

        if (courses.length === 0) {
          setError('No courses available')
          return
        }

        const course = courses[0]
        const coursePath = await coursesApi.getCoursePath(course.id)

        // Transform to include progress
        const unitsWithProgress: UnitWithProgress[] = coursePath.map((unit, unitIndex) => ({
          ...unit,
          skills: unit.skills.map((skill, skillIndex) => ({
            ...skill,
            crownLevel: unitIndex === 0 && skillIndex === 0 ? 3 : unitIndex === 0 && skillIndex === 1 ? 1 : 0,
            isLocked: !(unitIndex === 0 && skillIndex <= 2),
            isComplete: unitIndex === 0 && skillIndex === 0,
            isLegendary: false,
          })),
        }))

        setUnits(unitsWithProgress)
      } catch (err) {
        console.error('Failed to load course data:', err)
        setError('Failed to load course data')
      } finally {
        setLoading(false)
      }
    }

    loadCourseData()
  }, [])

  const handleSkillClick = (skillId: string) => {
    navigate(`/lesson/${skillId}`)
  }

  const toggleUnitExpanded = (unitIndex: number) => {
    setExpandedUnits(prev => {
      const next = new Set(prev)
      if (next.has(unitIndex)) {
        next.delete(unitIndex)
      } else {
        next.add(unitIndex)
      }
      return next
    })
  }

  if (loading) {
    return (
      <Center h="60vh">
        <Stack align="center" gap="md">
          <Loader size="lg" color="green" />
          <Text c={colors.text.secondary}>Loading your lessons...</Text>
        </Stack>
      </Center>
    )
  }

  if (error) {
    return (
      <Center h="60vh">
        <Stack align="center" gap="md">
          <Text c={colors.semantic.error} fw={600}>{error}</Text>
          <Text c={colors.text.secondary}>Please try again later</Text>
        </Stack>
      </Center>
    )
  }

  return (
    <div style={{ display: 'flex', gap: 24, padding: '24px 0' }}>
      {/* Main content - skill tree */}
      <div style={{ flex: 1, maxWidth: 500, margin: '0 auto' }}>
        {/* Streak banner */}
        <Paper
          p="md"
          radius="xl"
          mb="xl"
          style={{
            backgroundColor: colors.accent.orangeLight,
            border: `2px solid ${colors.accent.orange}`,
          }}
        >
          <Group justify="space-between">
            <Group gap="xs">
              <IconFlame size={28} style={{ color: colors.accent.orange }} />
              <div>
                <Text fw={700} style={{ color: colors.accent.orange }}>
                  {user?.streak_days || 0} day streak!
                </Text>
                <Text size="xs" style={{ color: colors.text.secondary }}>
                  Keep learning to extend your streak
                </Text>
              </div>
            </Group>
            <Badge size="lg" color="orange" variant="light">
              🔥
            </Badge>
          </Group>
        </Paper>

        {/* Units and skill tree */}
        <Stack gap="xl">
          {units.map((unit, unitIndex) => (
            <div key={unit.id}>
              <UnitHeader
                unit={unit}
                unitIndex={unitIndex}
                onJump={() => toggleUnitExpanded(unitIndex)}
              />

              {/* Skill tree path - only show for expanded units */}
              {expandedUnits.has(unitIndex) && (
                <div style={{
                  display: 'flex',
                  flexDirection: 'column',
                  alignItems: 'center',
                  paddingTop: 16,
                }}>
                  {unit.skills.map((skill, skillIndex) => {
                    // Find the first unlocked, incomplete skill
                    const firstActiveIndex = unit.skills.findIndex(s => !s.isLocked && !s.isComplete)
                    const isFirst = skillIndex === firstActiveIndex

                    return (
                      <SkillNode
                        key={skill.id}
                        skill={skill}
                        index={skillIndex}
                        onClick={() => handleSkillClick(skill.id)}
                        isFirst={isFirst}
                      />
                    )
                  })}
                </div>
              )}
            </div>
          ))}
        </Stack>
      </div>

      {/* Right sidebar - quests & leaderboard (hidden on mobile) */}
      <div style={{ width: 320, flexShrink: 0, display: 'none' }} className="right-sidebar">
        <Stack gap="lg" style={{ position: 'sticky', top: 80 }}>
          <DailyQuestsCard />
          <LeaderboardPreview />
        </Stack>
      </div>

      {/* CSS for responsive right sidebar */}
      <style>{`
        @media (min-width: 1200px) {
          .right-sidebar {
            display: block !important;
          }
        }
      `}</style>
    </div>
  )
}
