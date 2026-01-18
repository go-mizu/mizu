import { useState, useEffect } from 'react'
import { Text, Paper, Group, Stack, Badge, ActionIcon, Tooltip, Loader, Center, Button, Progress } from '@mantine/core'
import { IconLock, IconCheck, IconBook, IconStar, IconCrown, IconChevronLeft, IconTrophy } from '@tabler/icons-react'
import { useNavigate } from 'react-router-dom'
import { motion } from 'framer-motion'
import { colors } from '../styles/tokens'
import { coursesApi, Unit, Skill } from '../api/client'
import { useAuthStore } from '../stores/auth'

interface SkillWithProgress extends Skill {
  crownLevel: number
  isLocked: boolean
  isComplete: boolean
  isLegendary: boolean
}

interface UnitWithProgress extends Omit<Unit, 'skills'> {
  skills: SkillWithProgress[]
}

// Winding path positions - creates a serpentine pattern like Duolingo
const getSkillPosition = (index: number): number => {
  // Pattern: center, slight-right, center, slight-left, center, slight-right
  const offsets = [0, 60, 0, -60, 0, 60]
  return offsets[index % offsets.length]
}

interface SkillNodeProps {
  skill: SkillWithProgress
  index: number
  onClick: () => void
  showStart: boolean
}

function SkillNode({ skill, index, onClick, showStart }: SkillNodeProps) {
  const offsetX = getSkillPosition(index)

  const getBackgroundColor = () => {
    if (skill.isLegendary || skill.isComplete) return colors.accent.yellow
    if (skill.isLocked) return '#E5E5E5'
    return colors.primary.green
  }

  const getShadowColor = () => {
    if (skill.isLegendary || skill.isComplete) return '0 6px 0 #C9A000'
    if (skill.isLocked) return 'none'
    return '0 6px 0 #58A700'
  }

  const getIcon = () => {
    if (skill.isLocked) {
      return <IconLock size={32} style={{ color: '#AFAFAF' }} />
    }
    if (skill.isLegendary) {
      return <IconCrown size={32} style={{ color: 'white' }} />
    }
    // Show Chinese character for some skills, star for others
    if (skill.icon_name === 'hanzi' || skill.name?.includes('Character')) {
      return <Text size="xl" fw={700} style={{ color: skill.isLocked ? '#AFAFAF' : 'white' }}>字</Text>
    }
    return <IconStar size={32} style={{ color: 'white' }} />
  }

  return (
    <motion.div
      initial={{ opacity: 0, scale: 0.8 }}
      animate={{ opacity: 1, scale: 1 }}
      transition={{ delay: index * 0.05, duration: 0.2 }}
      style={{
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        transform: `translateX(${offsetX}px)`,
        marginBottom: 24,
        position: 'relative',
      }}
    >
      {/* START button above the first active skill */}
      {showStart && (
        <motion.div
          initial={{ opacity: 0, y: 10 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.3 }}
          style={{ marginBottom: 8 }}
        >
          <Button
            color="green"
            radius="xl"
            size="sm"
            onClick={onClick}
            style={{
              fontWeight: 700,
              textTransform: 'uppercase',
              letterSpacing: '1px',
              paddingLeft: 24,
              paddingRight: 24,
            }}
          >
            START
          </Button>
        </motion.div>
      )}

      <Tooltip
        label={skill.isLocked ? 'Complete previous skills to unlock' : skill.name}
        withArrow
        position="right"
      >
        <motion.div
          whileHover={{ scale: skill.isLocked ? 1 : 1.05 }}
          whileTap={{ scale: skill.isLocked ? 1 : 0.95 }}
          style={{ position: 'relative' }}
        >
          <Paper
            onClick={skill.isLocked ? undefined : onClick}
            style={{
              width: 70,
              height: 70,
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
            {getIcon()}
          </Paper>

          {/* Crown level badge */}
          {!skill.isLocked && skill.crownLevel > 0 && (
            <Badge
              size="sm"
              style={{
                position: 'absolute',
                bottom: -8,
                left: '50%',
                transform: 'translateX(-50%)',
                backgroundColor: skill.isComplete ? colors.accent.yellow : colors.primary.green,
                color: 'white',
                fontWeight: 700,
                border: '2px solid white',
                fontSize: 10,
              }}
            >
              {skill.crownLevel}/{skill.levels}
            </Badge>
          )}

          {/* Completion checkmark */}
          {skill.isComplete && (
            <IconCheck
              size={18}
              style={{
                position: 'absolute',
                top: -4,
                right: -4,
                backgroundColor: colors.accent.yellow,
                borderRadius: '50%',
                padding: 2,
                color: 'white',
                border: '2px solid white',
              }}
            />
          )}
        </motion.div>
      </Tooltip>
    </motion.div>
  )
}

// Section header like Duolingo's green bar
function SectionHeader({ unit, unitIndex, onBack, onGuidebook }: { unit: UnitWithProgress; unitIndex: number; onBack?: () => void; onGuidebook?: () => void }) {
  return (
    <Paper
      p="md"
      radius="xl"
      mb="xl"
      style={{
        backgroundColor: colors.primary.green,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
      }}
    >
      <Group gap="xs">
        {onBack && (
          <ActionIcon
            variant="transparent"
            onClick={onBack}
            style={{ color: 'white' }}
          >
            <IconChevronLeft size={20} />
          </ActionIcon>
        )}
        <div>
          <Text size="xs" fw={700} style={{ color: 'rgba(255,255,255,0.8)', textTransform: 'uppercase' }}>
            Section {unitIndex + 1}, Unit 1
          </Text>
          <Text fw={700} style={{ color: 'white' }}>
            {unit.title}
          </Text>
        </div>
      </Group>
      <Button
        variant="white"
        color="dark"
        radius="xl"
        size="sm"
        leftSection={<IconBook size={16} />}
        style={{ fontWeight: 700 }}
        onClick={onGuidebook}
      >
        GUIDEBOOK
      </Button>
    </Paper>
  )
}

// Right sidebar cards
function UnlockLeaderboardsCard() {
  return (
    <Paper p="lg" radius="lg" style={{ backgroundColor: '#FFFFFF', border: '2px solid #E5E5E5' }}>
      <Text fw={700} size="lg" mb="md" style={{ color: colors.text.primary }}>
        Unlock Leaderboards!
      </Text>
      <Group gap="md" align="flex-start">
        <div style={{
          width: 48,
          height: 48,
          borderRadius: '50%',
          backgroundColor: '#E5E5E5',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
        }}>
          <IconTrophy size={24} style={{ color: '#AFAFAF' }} />
        </div>
        <Text size="sm" style={{ color: colors.text.secondary, flex: 1 }}>
          Complete 10 more lessons to start competing
        </Text>
      </Group>
    </Paper>
  )
}

function DailyQuestsCard() {
  const navigate = useNavigate()

  return (
    <Paper p="lg" radius="lg" style={{ backgroundColor: '#FFFFFF', border: '2px solid #E5E5E5' }}>
      <Group justify="space-between" mb="md">
        <Text fw={700} size="lg" style={{ color: colors.text.primary }}>Daily Quests</Text>
        <Button variant="subtle" color="blue" size="xs" onClick={() => navigate('/quests')}>
          VIEW ALL
        </Button>
      </Group>
      <Group gap="md" align="center">
        <Text size="xl">⚡</Text>
        <div style={{ flex: 1 }}>
          <Text fw={600} size="sm" style={{ color: colors.text.primary }}>Earn 10 XP</Text>
          <Progress value={0} size="sm" radius="xl" color="gray" mt={4} />
          <Text size="xs" style={{ color: colors.text.muted }} mt={2}>0 / 10</Text>
        </div>
        <div style={{
          width: 36,
          height: 36,
          borderRadius: 8,
          backgroundColor: colors.neutral.background,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
        }}>
          <Text>🎁</Text>
        </div>
      </Group>
    </Paper>
  )
}

function CreateProfileCard() {
  const navigate = useNavigate()

  return (
    <Paper p="lg" radius="lg" style={{ backgroundColor: '#FFFFFF', border: '2px solid #E5E5E5' }}>
      <Text fw={700} size="lg" mb="lg" ta="center" style={{ color: colors.text.primary }}>
        Create a profile to save your progress!
      </Text>
      <Stack gap="sm">
        <Button
          fullWidth
          color="green"
          radius="xl"
          size="md"
          style={{ fontWeight: 700, textTransform: 'uppercase' }}
          onClick={() => navigate('/signup')}
        >
          Create a Profile
        </Button>
        <Button
          fullWidth
          color="blue"
          radius="xl"
          size="md"
          style={{ fontWeight: 700, textTransform: 'uppercase' }}
          onClick={() => navigate('/login')}
        >
          Sign In
        </Button>
      </Stack>
    </Paper>
  )
}

// Locked skill tooltip/card
function LockedSkillCard({ skill }: { skill: SkillWithProgress }) {
  if (!skill.isLocked) return null

  return (
    <Paper
      p="lg"
      radius="lg"
      style={{
        backgroundColor: '#FFFFFF',
        border: '2px solid #E5E5E5',
        textAlign: 'center',
        marginTop: -16,
        marginBottom: 24,
        maxWidth: 280,
        marginLeft: 'auto',
        marginRight: 'auto',
      }}
    >
      <Text fw={700} style={{ color: colors.text.primary }}>{skill.name}</Text>
      <Text size="sm" style={{ color: colors.text.secondary }} mt="xs">
        Complete all levels above to unlock this!
      </Text>
      <Button
        fullWidth
        variant="light"
        color="gray"
        radius="xl"
        mt="md"
        disabled
        style={{ fontWeight: 700, textTransform: 'uppercase' }}
      >
        LOCKED
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

  useEffect(() => {
    async function loadCourseData() {
      try {
        setLoading(true)

        // If user has no active course, redirect to course selection
        if (!user?.active_course_id) {
          // Try to get courses and auto-select first one, or redirect to course selection
          const courses = await coursesApi.listCourses('en')
          if (courses.length === 0) {
            setError('No courses available')
            return
          }
          // Redirect to course selection page
          navigate('/courses')
          return
        }

        const coursePath = await coursesApi.getCoursePath(user.active_course_id)

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
  }, [user?.active_course_id, navigate])

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
    <div style={{
      display: 'flex',
      gap: 48,
      maxWidth: 1000,
      margin: '0 auto',
      justifyContent: 'center',
    }}>
      {/* Main content - skill tree */}
      <div style={{ flex: '0 0 600px', maxWidth: 600 }}>
        {units.map((unit, unitIndex) => (
          <div key={unit.id}>
            {/* Section header */}
            <SectionHeader
              unit={unit}
              unitIndex={unitIndex}
              onBack={unitIndex > 0 ? () => {} : undefined}
              onGuidebook={() => navigate(`/guidebook/${unit.id}`)}
            />

            {/* Skill tree */}
            <div style={{
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
              paddingBottom: 40,
            }}>
              {unit.skills.map((skill, skillIndex) => {
                const firstActiveIndex = unit.skills.findIndex(s => !s.isLocked && !s.isComplete)
                const showStart = skillIndex === firstActiveIndex

                return (
                  <div key={skill.id}>
                    <SkillNode
                      skill={skill}
                      index={skillIndex}
                      onClick={() => handleSkillClick(skill.id)}
                      showStart={showStart}
                    />
                    {/* Show locked card for first locked skill after active ones */}
                    {skill.isLocked && skillIndex === unit.skills.findIndex(s => s.isLocked) && (
                      <LockedSkillCard skill={skill} />
                    )}
                  </div>
                )
              })}
            </div>

            {/* Section divider */}
            {unitIndex < units.length - 1 && (
              <div style={{
                display: 'flex',
                alignItems: 'center',
                gap: 16,
                marginBottom: 32,
              }}>
                <div style={{ flex: 1, height: 2, backgroundColor: '#E5E5E5' }} />
                <Text size="sm" fw={600} style={{ color: colors.text.muted }}>
                  {units[unitIndex + 1]?.title || 'Next section'}
                </Text>
                <div style={{ flex: 1, height: 2, backgroundColor: '#E5E5E5' }} />
              </div>
            )}
          </div>
        ))}
      </div>

      {/* Right sidebar */}
      <div style={{ flex: '0 0 330px', width: 330 }} className="right-sidebar">
        <Stack gap="lg" style={{ position: 'sticky', top: 80 }}>
          <UnlockLeaderboardsCard />
          <DailyQuestsCard />
          <CreateProfileCard />

          {/* Footer links */}
          <Group justify="center" gap="md" mt="lg">
            {['ABOUT', 'BLOG', 'STORE', 'EFFICACY', 'CAREERS'].map((link) => (
              <Text key={link} size="xs" fw={600} style={{ color: colors.text.muted, cursor: 'pointer' }}>
                {link}
              </Text>
            ))}
          </Group>
          <Group justify="center" gap="md">
            {['INVESTORS', 'TERMS', 'PRIVACY'].map((link) => (
              <Text key={link} size="xs" fw={600} style={{ color: colors.text.muted, cursor: 'pointer' }}>
                {link}
              </Text>
            ))}
          </Group>
        </Stack>
      </div>

      {/* Hide right sidebar on smaller screens */}
      <style>{`
        @media (max-width: 900px) {
          .right-sidebar {
            display: none;
          }
        }
      `}</style>
    </div>
  )
}
