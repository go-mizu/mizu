import { useState, useEffect } from 'react'
import { Container, Title, Text, Paper, Group, Stack, Progress, Badge, ActionIcon, Tooltip, Loader, Center } from '@mantine/core'
import { IconLock, IconCheck, IconBook, IconFlame, IconChevronLeft, IconStar } from '@tabler/icons-react'
import { useNavigate } from 'react-router-dom'
import { motion } from 'framer-motion'
import { useAuthStore } from '../stores/auth'
import { colors } from '../styles/tokens'
import { coursesApi, Unit, Skill } from '../api/client'

interface SkillWithProgress extends Skill {
  crownLevel: number
  isLocked: boolean
  isComplete: boolean
}

interface UnitWithProgress extends Omit<Unit, 'skills'> {
  skills: SkillWithProgress[]
}

function SkillNode({ skill, onClick }: { skill: SkillWithProgress; onClick: () => void }) {
  const getBackgroundColor = () => {
    if (skill.isComplete) return colors.accent.yellow
    if (skill.isLocked) return colors.neutral.border
    if (skill.crownLevel > 0) return colors.primary.green
    return colors.primary.green
  }

  const getShadowColor = () => {
    if (skill.isComplete) return colors.shadows?.skill?.completed || '0 4px 0 #E5B400'
    if (skill.isLocked) return 'none'
    return '0 4px 0 #58A700'
  }

  const getIconForSkill = (iconName: string) => {
    switch (iconName) {
      case 'star':
        return <IconStar size={32} style={{ color: 'white' }} />
      case 'book':
        return <IconBook size={32} style={{ color: 'white' }} />
      default:
        return <Text size="1.5rem" style={{ color: 'white' }}>字</Text>
    }
  }

  return (
    <motion.div
      whileHover={{ scale: skill.isLocked ? 1 : 1.05 }}
      whileTap={{ scale: skill.isLocked ? 1 : 0.95 }}
    >
      <Tooltip label={skill.isLocked ? 'Complete previous skills to unlock' : skill.name}>
        <Paper
          onClick={skill.isLocked ? undefined : onClick}
          style={{
            width: 72,
            height: 72,
            borderRadius: '50%',
            backgroundColor: getBackgroundColor(),
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            cursor: skill.isLocked ? 'not-allowed' : 'pointer',
            position: 'relative',
            boxShadow: getShadowColor(),
            transition: 'all 0.15s ease',
          }}
        >
          {skill.isLocked ? (
            <IconLock size={32} style={{ color: colors.text.muted }} />
          ) : (
            getIconForSkill(skill.icon_name)
          )}

          {!skill.isLocked && skill.crownLevel > 0 && (
            <Badge
              size="sm"
              style={{
                position: 'absolute',
                bottom: -8,
                backgroundColor: colors.accent.yellow,
                color: colors.text.primary,
                fontWeight: 700,
                border: '2px solid white',
              }}
            >
              {skill.crownLevel}/{skill.levels}
            </Badge>
          )}

          {skill.isComplete && (
            <IconCheck
              size={20}
              style={{
                position: 'absolute',
                top: -4,
                right: -4,
                backgroundColor: colors.accent.yellow,
                borderRadius: '50%',
                padding: 2,
                color: 'white',
              }}
            />
          )}
        </Paper>
      </Tooltip>
      <Text ta="center" mt="sm" size="sm" fw={600} style={{ color: skill.isLocked ? colors.text.muted : colors.text.primary }}>
        {skill.name}
      </Text>
    </motion.div>
  )
}

export default function Home() {
  const navigate = useNavigate()
  const { user } = useAuthStore()
  const [units, setUnits] = useState<UnitWithProgress[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [currentCourseId, setCurrentCourseId] = useState<string | null>(null)

  useEffect(() => {
    async function loadCourseData() {
      try {
        setLoading(true)
        // Get available courses for English speakers
        const courses = await coursesApi.listCourses('en')

        if (courses.length === 0) {
          setError('No courses available')
          return
        }

        // Use first course for now (Spanish)
        const course = courses[0]
        setCurrentCourseId(course.id)

        // Get course path (units with skills)
        const coursePath = await coursesApi.getCoursePath(course.id)

        // Transform to include progress (mock for now until progress API is connected)
        const unitsWithProgress: UnitWithProgress[] = coursePath.map((unit, unitIndex) => ({
          ...unit,
          skills: unit.skills.map((skill, skillIndex) => ({
            ...skill,
            crownLevel: unitIndex === 0 && skillIndex === 0 ? 3 : 0,
            isLocked: !(unitIndex === 0 && skillIndex <= 1),
            isComplete: false,
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
    <Container size="md" py="lg">
      {/* Back button */}
      <Group mb="lg">
        <ActionIcon variant="subtle" size="lg" onClick={() => navigate(-1)}>
          <IconChevronLeft size={24} style={{ color: colors.text.secondary }} />
        </ActionIcon>
        <Text size="sm" fw={600} c={colors.text.secondary}>Back</Text>
      </Group>

      {/* Daily Goal Progress */}
      <Paper p="lg" radius="lg" mb="xl" style={{ backgroundColor: colors.neutral.white, border: `2px solid ${colors.neutral.border}` }}>
        <Group justify="space-between" mb="md">
          <div>
            <Text size="lg" fw={700} style={{ color: colors.text.primary }}>Daily Goal</Text>
            <Text size="sm" style={{ color: colors.text.secondary }}>
              {user?.daily_goal_minutes || 10} minutes per day
            </Text>
          </div>
          <Group gap="xs">
            <IconFlame size={24} style={{ color: colors.accent.orange }} />
            <Text fw={700} style={{ color: colors.accent.orange }}>{user?.streak_days || 0} day streak</Text>
          </Group>
        </Group>
        <Progress value={35} size="lg" radius="xl" color="green" />
        <Text ta="center" mt="sm" size="sm" style={{ color: colors.text.secondary }}>
          3.5 / 10 minutes completed today
        </Text>
      </Paper>

      {/* Learning Path - Sections */}
      <Stack gap="xl">
        {units.map((unit, unitIndex) => (
          <motion.div
            key={unit.id}
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: unitIndex * 0.1 }}
          >
            {/* Section Card */}
            <Paper
              p="lg"
              radius="lg"
              mb="lg"
              style={{
                backgroundColor: colors.neutral.white,
                border: `2px solid ${colors.neutral.border}`,
                position: 'relative',
                overflow: 'hidden',
              }}
            >
              <Group justify="space-between" align="flex-start">
                <div style={{ flex: 1 }}>
                  <Group gap="xs" mb="xs">
                    <Badge
                      variant="light"
                      color="gray"
                      size="sm"
                      style={{ fontWeight: 700, textTransform: 'uppercase' }}
                    >
                      Section {unitIndex + 1}
                    </Badge>
                    <Text size="xs" c={colors.text.muted}>{unit.skills.length} UNITS</Text>
                  </Group>
                  <Title order={3} style={{ color: colors.text.primary }} mb="xs">{unit.title}</Title>
                  <Text size="sm" style={{ color: colors.text.secondary }}>{unit.description}</Text>
                </div>

                {/* Guidebook button */}
                {unit.guidebook_content && (
                  <ActionIcon
                    variant="light"
                    size="lg"
                    radius="xl"
                    color="blue"
                    style={{ border: `2px solid ${colors.secondary.blue}` }}
                  >
                    <IconBook size={20} style={{ color: colors.secondary.blue }} />
                  </ActionIcon>
                )}
              </Group>

              {/* Continue/Start button */}
              {unitIndex === 0 && (
                <Paper
                  p="md"
                  mt="lg"
                  radius="lg"
                  style={{
                    backgroundColor: colors.primary.green,
                    textAlign: 'center',
                    cursor: 'pointer',
                    boxShadow: '0 4px 0 #58A700',
                  }}
                  onClick={() => handleSkillClick(unit.skills[0]?.id)}
                >
                  <Text fw={700} style={{ color: 'white', textTransform: 'uppercase', letterSpacing: '0.5px' }}>
                    Continue
                  </Text>
                </Paper>
              )}

              {unitIndex > 0 && (
                <Paper
                  p="md"
                  mt="lg"
                  radius="lg"
                  style={{
                    backgroundColor: colors.neutral.white,
                    textAlign: 'center',
                    cursor: 'pointer',
                    border: `2px solid ${colors.secondary.blue}`,
                  }}
                  onClick={() => handleSkillClick(unit.skills[0]?.id)}
                >
                  <Text fw={700} style={{ color: colors.secondary.blue, textTransform: 'uppercase', letterSpacing: '0.5px' }}>
                    Jump to Section {unitIndex + 1}
                  </Text>
                </Paper>
              )}
            </Paper>

            {/* Skills Path - only show for first section or expanded */}
            {unitIndex === 0 && (
              <Group justify="center" gap="xl" mt="lg" style={{ position: 'relative' }}>
                {unit.skills.map((skill, skillIndex) => (
                  <div
                    key={skill.id}
                    style={{
                      marginTop: skillIndex % 2 === 1 ? 40 : 0,
                    }}
                  >
                    <SkillNode skill={skill} onClick={() => handleSkillClick(skill.id)} />
                  </div>
                ))}
              </Group>
            )}
          </motion.div>
        ))}
      </Stack>
    </Container>
  )
}
