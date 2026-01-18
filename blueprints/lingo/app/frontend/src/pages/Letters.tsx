import { Container, Title, Text, Paper, Group, Stack, Progress, Badge, Button, SimpleGrid } from '@mantine/core'
import { IconVolume, IconCheck, IconLock } from '@tabler/icons-react'
import { colors } from '../styles/tokens'

interface LetterUnit {
  id: string
  title: string
  letters: string[]
  isUnlocked: boolean
  isComplete: boolean
  progress: number
}

const letterUnits: LetterUnit[] = [
  {
    id: '1',
    title: 'Basic Vowels',
    letters: ['A', 'E', 'I', 'O', 'U'],
    isUnlocked: true,
    isComplete: true,
    progress: 100,
  },
  {
    id: '2',
    title: 'Common Consonants',
    letters: ['B', 'C', 'D', 'F', 'G'],
    isUnlocked: true,
    isComplete: false,
    progress: 60,
  },
  {
    id: '3',
    title: 'More Consonants',
    letters: ['H', 'J', 'K', 'L', 'M'],
    isUnlocked: true,
    isComplete: false,
    progress: 20,
  },
  {
    id: '4',
    title: 'Final Consonants',
    letters: ['N', 'P', 'Q', 'R', 'S'],
    isUnlocked: false,
    isComplete: false,
    progress: 0,
  },
  {
    id: '5',
    title: 'Remaining Letters',
    letters: ['T', 'V', 'W', 'X', 'Y', 'Z'],
    isUnlocked: false,
    isComplete: false,
    progress: 0,
  },
  {
    id: '6',
    title: 'Special Characters',
    letters: ['Ñ', 'Ü', 'É', 'Á', 'Í'],
    isUnlocked: false,
    isComplete: false,
    progress: 0,
  },
]

function LetterCard({ letter, isLearned }: { letter: string; isLearned: boolean }) {
  return (
    <Paper
      p="md"
      radius="lg"
      style={{
        backgroundColor: isLearned ? colors.primary.greenLight : colors.neutral.white,
        border: `2px solid ${isLearned ? colors.primary.green : colors.neutral.border}`,
        textAlign: 'center',
        cursor: 'pointer',
        transition: 'all 0.15s ease',
      }}
    >
      <Text size="2rem" fw={800} style={{ color: isLearned ? colors.primary.green : colors.text.primary }}>
        {letter}
      </Text>
      {isLearned && (
        <IconCheck size={16} style={{ color: colors.primary.green }} />
      )}
    </Paper>
  )
}

function UnitCard({ unit }: { unit: LetterUnit }) {
  return (
    <Paper
      p="lg"
      radius="lg"
      style={{
        backgroundColor: unit.isComplete ? colors.primary.greenLight : colors.neutral.white,
        border: `2px solid ${unit.isComplete ? colors.primary.green : unit.isUnlocked ? colors.neutral.border : colors.neutral.border}`,
        opacity: unit.isUnlocked ? 1 : 0.6,
      }}
    >
      <Group justify="space-between" mb="md">
        <div>
          <Group gap="xs">
            <Title order={4} style={{ color: colors.text.primary }}>{unit.title}</Title>
            {unit.isComplete && (
              <Badge color="green" size="sm">Complete</Badge>
            )}
            {!unit.isUnlocked && (
              <Badge color="gray" size="sm" leftSection={<IconLock size={12} />}>Locked</Badge>
            )}
          </Group>
          {unit.isUnlocked && !unit.isComplete && (
            <>
              <Progress value={unit.progress} size="sm" radius="xl" color="green" mt="sm" />
              <Text size="xs" mt={4} style={{ color: colors.text.muted }}>
                {unit.progress}% complete
              </Text>
            </>
          )}
        </div>
        {unit.isUnlocked && (
          <Button
            color={unit.isComplete ? 'green' : 'blue'}
            variant={unit.isComplete ? 'light' : 'filled'}
            radius="xl"
          >
            {unit.isComplete ? 'Review' : 'Practice'}
          </Button>
        )}
      </Group>

      {/* Letter grid */}
      <SimpleGrid cols={5} spacing="xs">
        {unit.letters.map((letter, i) => (
          <LetterCard
            key={letter}
            letter={letter}
            isLearned={unit.isUnlocked && (unit.isComplete || (unit.progress / 100) * unit.letters.length > i)}
          />
        ))}
      </SimpleGrid>
    </Paper>
  )
}

export default function Letters() {
  const totalLetters = letterUnits.reduce((acc, u) => acc + u.letters.length, 0)
  const learnedLetters = letterUnits.reduce((acc, u) => {
    if (u.isComplete) return acc + u.letters.length
    if (u.isUnlocked) return acc + Math.floor((u.progress / 100) * u.letters.length)
    return acc
  }, 0)

  return (
    <Container size="md">
      {/* Header */}
      <Paper
        p="xl"
        radius="lg"
        mb="xl"
        style={{
          backgroundColor: colors.accent.purple,
          textAlign: 'center',
        }}
      >
        <Text size="3rem" mb="md">🔤</Text>
        <Title order={2} style={{ color: 'white' }}>Letters</Title>
        <Text style={{ color: 'rgba(255,255,255,0.8)' }}>
          Master the alphabet with pronunciation practice
        </Text>
        <Group justify="center" mt="md">
          <Badge size="xl" color="white" style={{ backgroundColor: 'rgba(255,255,255,0.2)', color: 'white' }}>
            {learnedLetters}/{totalLetters} letters learned
          </Badge>
        </Group>
      </Paper>

      {/* Progress overview */}
      <Paper p="lg" radius="lg" mb="xl" style={{ backgroundColor: colors.neutral.white }}>
        <Group justify="space-between" mb="md">
          <Text fw={700} style={{ color: colors.text.primary }}>Your Progress</Text>
          <Text fw={600} style={{ color: colors.primary.green }}>
            {Math.round((learnedLetters / totalLetters) * 100)}%
          </Text>
        </Group>
        <Progress
          value={(learnedLetters / totalLetters) * 100}
          size="lg"
          radius="xl"
          color="purple"
        />
      </Paper>

      {/* Letter units */}
      <Stack gap="lg">
        {letterUnits.map((unit) => (
          <UnitCard key={unit.id} unit={unit} />
        ))}
      </Stack>

      {/* Audio practice tip */}
      <Paper
        p="lg"
        radius="lg"
        mt="xl"
        style={{
          backgroundColor: colors.secondary.blueLight,
          border: `2px solid ${colors.secondary.blue}`,
        }}
      >
        <Group gap="md">
          <IconVolume size={32} style={{ color: colors.secondary.blue }} />
          <div>
            <Text fw={700} style={{ color: colors.secondary.blue }}>Audio Practice</Text>
            <Text size="sm" style={{ color: colors.text.secondary }}>
              Tap on any letter to hear its pronunciation. Practice until you can recognize each sound!
            </Text>
          </div>
        </Group>
      </Paper>
    </Container>
  )
}
