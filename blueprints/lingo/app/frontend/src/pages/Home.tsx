import { useState } from 'react'
import { Container, Title, Text, Paper, Group, Stack, Progress, Badge, ActionIcon, Tooltip } from '@mantine/core'
import { IconLock, IconCheck, IconBook, IconFlame } from '@tabler/icons-react'
import { useNavigate } from 'react-router-dom'
import { motion } from 'framer-motion'
import { useAuthStore } from '../stores/auth'

interface Skill {
  id: string
  name: string
  icon: string
  level: number
  maxLevel: number
  isLocked: boolean
  isComplete: boolean
}

interface Unit {
  id: string
  title: string
  description: string
  skills: Skill[]
}

// Mock data - in production, fetch from API
const mockUnits: Unit[] = [
  {
    id: '1',
    title: 'Basics 1',
    description: 'Learn basic greetings and introductions',
    skills: [
      { id: '1', name: 'Greetings', icon: '👋', level: 3, maxLevel: 5, isLocked: false, isComplete: false },
      { id: '2', name: 'Introduction', icon: '🙋', level: 0, maxLevel: 5, isLocked: false, isComplete: false },
      { id: '3', name: 'Common Phrases', icon: '💬', level: 0, maxLevel: 5, isLocked: true, isComplete: false },
    ],
  },
  {
    id: '2',
    title: 'Basics 2',
    description: 'Learn more fundamental vocabulary',
    skills: [
      { id: '4', name: 'Family', icon: '👨‍👩‍👧', level: 0, maxLevel: 5, isLocked: true, isComplete: false },
      { id: '5', name: 'Numbers', icon: '🔢', level: 0, maxLevel: 5, isLocked: true, isComplete: false },
      { id: '6', name: 'Colors', icon: '🎨', level: 0, maxLevel: 5, isLocked: true, isComplete: false },
    ],
  },
  {
    id: '3',
    title: 'Food',
    description: 'Learn food vocabulary',
    skills: [
      { id: '7', name: 'Fruits', icon: '🍎', level: 0, maxLevel: 5, isLocked: true, isComplete: false },
      { id: '8', name: 'Vegetables', icon: '🥕', level: 0, maxLevel: 5, isLocked: true, isComplete: false },
      { id: '9', name: 'Drinks', icon: '🥤', level: 0, maxLevel: 5, isLocked: true, isComplete: false },
    ],
  },
]

function SkillNode({ skill, onClick }: { skill: Skill; onClick: () => void }) {
  const getColor = () => {
    if (skill.isComplete) return '#ffc800'
    if (skill.isLocked) return '#3d5a68'
    if (skill.level > 0) return '#58cc02'
    return '#58cc02'
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
            width: 80,
            height: 80,
            borderRadius: '50%',
            backgroundColor: getColor(),
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            cursor: skill.isLocked ? 'not-allowed' : 'pointer',
            position: 'relative',
            boxShadow: skill.isLocked ? 'none' : '0 4px 0 #3f9a02',
          }}
        >
          {skill.isLocked ? (
            <IconLock size={32} style={{ color: '#8fa8b2' }} />
          ) : (
            <Text size="2rem">{skill.icon}</Text>
          )}

          {!skill.isLocked && skill.level > 0 && (
            <Badge
              size="sm"
              color="yellow"
              style={{
                position: 'absolute',
                bottom: -5,
                fontWeight: 700,
              }}
            >
              {skill.level}/{skill.maxLevel}
            </Badge>
          )}

          {skill.isComplete && (
            <IconCheck
              size={20}
              style={{
                position: 'absolute',
                top: -5,
                right: -5,
                backgroundColor: '#ffc800',
                borderRadius: '50%',
                padding: 2,
              }}
            />
          )}
        </Paper>
      </Tooltip>
      <Text ta="center" mt="xs" size="sm" fw={600} style={{ color: skill.isLocked ? '#8fa8b2' : 'white' }}>
        {skill.name}
      </Text>
    </motion.div>
  )
}

export default function Home() {
  const navigate = useNavigate()
  const { user } = useAuthStore()
  const [units] = useState<Unit[]>(mockUnits)

  const handleSkillClick = (skillId: string) => {
    navigate(`/lesson/${skillId}`)
  }

  return (
    <Container size="md">
      {/* Daily Goal Progress */}
      <Paper p="lg" radius="lg" mb="xl" style={{ backgroundColor: '#1a2c33' }}>
        <Group justify="space-between" mb="md">
          <div>
            <Text size="lg" fw={700} style={{ color: 'white' }}>Daily Goal</Text>
            <Text size="sm" style={{ color: '#8fa8b2' }}>
              {user?.daily_goal_minutes || 10} minutes per day
            </Text>
          </div>
          <Group gap="xs">
            <IconFlame size={24} style={{ color: '#ff9600' }} />
            <Text fw={700} style={{ color: '#ff9600' }}>{user?.streak_days || 0} day streak</Text>
          </Group>
        </Group>
        <Progress value={35} size="lg" radius="xl" color="green" />
        <Text ta="center" mt="sm" size="sm" style={{ color: '#8fa8b2' }}>
          3.5 / 10 minutes completed today
        </Text>
      </Paper>

      {/* Learning Path */}
      <Stack gap="xl">
        {units.map((unit, unitIndex) => (
          <motion.div
            key={unit.id}
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: unitIndex * 0.1 }}
          >
            {/* Unit Header */}
            <Paper p="md" radius="lg" mb="md" style={{ backgroundColor: '#1a2c33' }}>
              <Group justify="space-between">
                <div>
                  <Badge color="blue" mb="xs">{`Unit ${unitIndex + 1}`}</Badge>
                  <Title order={3} style={{ color: 'white' }}>{unit.title}</Title>
                  <Text size="sm" style={{ color: '#8fa8b2' }}>{unit.description}</Text>
                </div>
                <ActionIcon variant="subtle" size="lg" radius="xl">
                  <IconBook size={20} style={{ color: '#8fa8b2' }} />
                </ActionIcon>
              </Group>
            </Paper>

            {/* Skills Path */}
            <Group justify="center" gap="xl" style={{ position: 'relative' }}>
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
          </motion.div>
        ))}
      </Stack>

      {/* Start Button for Current Skill */}
      <Paper
        p="lg"
        radius="lg"
        mt="xl"
        style={{
          backgroundColor: '#58cc02',
          textAlign: 'center',
          cursor: 'pointer',
        }}
        onClick={() => handleSkillClick('1')}
      >
        <Title order={3} style={{ color: 'white' }}>Start Learning</Title>
        <Text style={{ color: 'rgba(255,255,255,0.9)' }}>Continue with Greetings</Text>
      </Paper>
    </Container>
  )
}
