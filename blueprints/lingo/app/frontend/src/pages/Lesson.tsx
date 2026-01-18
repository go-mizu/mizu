import { useState } from 'react'
import { Container, Paper, Title, Text, Button, Group, Stack, Progress, ActionIcon } from '@mantine/core'
import { IconX, IconHeart, IconCheck, IconVolume } from '@tabler/icons-react'
import { useNavigate, useParams } from 'react-router-dom'
import { motion, AnimatePresence } from 'framer-motion'
import { useAuthStore } from '../stores/auth'

interface Exercise {
  id: string
  type: 'translation' | 'multiple_choice' | 'listening'
  prompt: string
  correctAnswer: string
  choices?: string[]
}

// Mock exercises
const mockExercises: Exercise[] = [
  {
    id: '1',
    type: 'multiple_choice',
    prompt: 'What does "Hola" mean?',
    correctAnswer: 'Hello',
    choices: ['Hello', 'Goodbye', 'Thank you', 'Please'],
  },
  {
    id: '2',
    type: 'translation',
    prompt: 'Translate: Buenos días',
    correctAnswer: 'Good morning',
    choices: ['Good morning', 'Good night', 'Good afternoon', 'Good evening'],
  },
  {
    id: '3',
    type: 'multiple_choice',
    prompt: 'What does "Gracias" mean?',
    correctAnswer: 'Thank you',
    choices: ['Hello', 'Goodbye', 'Thank you', 'Please'],
  },
  {
    id: '4',
    type: 'translation',
    prompt: 'Translate: Adiós',
    correctAnswer: 'Goodbye',
    choices: ['Hello', 'Goodbye', 'Thank you', 'Please'],
  },
]

export default function Lesson() {
  const navigate = useNavigate()
  useParams() // Skill ID from route params (unused for now, using mock data)
  const { user, updateUser } = useAuthStore()

  const [currentIndex, setCurrentIndex] = useState(0)
  const [selectedAnswer, setSelectedAnswer] = useState<string | null>(null)
  const [isChecked, setIsChecked] = useState(false)
  const [isCorrect, setIsCorrect] = useState(false)
  const [hearts, setHearts] = useState(user?.hearts || 5)
  const [xpEarned, setXpEarned] = useState(0)
  const [mistakes, setMistakes] = useState(0)

  const currentExercise = mockExercises[currentIndex]
  const progress = ((currentIndex + 1) / mockExercises.length) * 100

  const handleCheck = () => {
    if (!selectedAnswer) return

    const correct = selectedAnswer === currentExercise.correctAnswer
    setIsCorrect(correct)
    setIsChecked(true)

    if (correct) {
      setXpEarned((prev) => prev + 3)
    } else {
      setMistakes((prev) => prev + 1)
      setHearts((prev) => Math.max(0, prev - 1))
    }
  }

  const handleContinue = () => {
    if (currentIndex < mockExercises.length - 1) {
      setCurrentIndex((prev) => prev + 1)
      setSelectedAnswer(null)
      setIsChecked(false)
    } else {
      // Lesson complete
      const finalXP = xpEarned + (mistakes === 0 ? 5 : 0) // Bonus for perfect
      updateUser({
        xp_total: (user?.xp_total || 0) + finalXP,
        hearts: hearts,
      })
      navigate('/learn')
    }
  }

  const handleQuit = () => {
    if (window.confirm('Are you sure you want to quit? Your progress will be lost.')) {
      navigate('/learn')
    }
  }

  return (
    <div style={{ minHeight: '100vh', backgroundColor: '#131f24' }}>
      {/* Header */}
      <Paper
        p="md"
        radius={0}
        style={{
          backgroundColor: '#1a2c33',
          borderBottom: '2px solid #3d5a68',
          position: 'sticky',
          top: 0,
          zIndex: 100,
        }}
      >
        <Container size="md">
          <Group justify="space-between">
            <ActionIcon variant="subtle" size="lg" onClick={handleQuit}>
              <IconX size={24} style={{ color: '#8fa8b2' }} />
            </ActionIcon>

            <Progress
              value={progress}
              size="lg"
              radius="xl"
              color="green"
              style={{ flex: 1, margin: '0 20px' }}
            />

            <Group gap={4}>
              {Array.from({ length: 5 }).map((_, i) => (
                <IconHeart
                  key={i}
                  size={24}
                  style={{
                    color: i < hearts ? '#ff4b4b' : '#3d5a68',
                    fill: i < hearts ? '#ff4b4b' : 'transparent',
                  }}
                />
              ))}
            </Group>
          </Group>
        </Container>
      </Paper>

      {/* Exercise Content */}
      <Container size="sm" py="xl">
        <AnimatePresence mode="wait">
          <motion.div
            key={currentIndex}
            initial={{ opacity: 0, x: 20 }}
            animate={{ opacity: 1, x: 0 }}
            exit={{ opacity: 0, x: -20 }}
          >
            <Stack gap="xl">
              {/* Prompt */}
              <div>
                <Text size="sm" tt="uppercase" fw={700} style={{ color: '#8fa8b2' }} mb="xs">
                  {currentExercise.type === 'translation' ? 'Translate this phrase' : 'Select the correct answer'}
                </Text>
                <Group gap="md">
                  <ActionIcon variant="filled" color="blue" size="xl" radius="xl">
                    <IconVolume size={20} />
                  </ActionIcon>
                  <Title order={2} style={{ color: 'white' }}>
                    {currentExercise.prompt}
                  </Title>
                </Group>
              </div>

              {/* Choices */}
              <Stack gap="md">
                {currentExercise.choices?.map((choice) => {
                  const isSelected = selectedAnswer === choice
                  const showCorrect = isChecked && choice === currentExercise.correctAnswer
                  const showIncorrect = isChecked && isSelected && !isCorrect

                  return (
                    <Paper
                      key={choice}
                      p="lg"
                      radius="lg"
                      onClick={() => !isChecked && setSelectedAnswer(choice)}
                      style={{
                        backgroundColor: showCorrect
                          ? '#58cc02'
                          : showIncorrect
                          ? '#ff4b4b'
                          : isSelected
                          ? '#233a42'
                          : '#1a2c33',
                        border: `2px solid ${
                          showCorrect
                            ? '#58cc02'
                            : showIncorrect
                            ? '#ff4b4b'
                            : isSelected
                            ? '#58cc02'
                            : '#3d5a68'
                        }`,
                        cursor: isChecked ? 'default' : 'pointer',
                        transition: 'all 0.2s ease',
                      }}
                    >
                      <Group justify="space-between">
                        <Text
                          size="lg"
                          fw={600}
                          style={{
                            color: showCorrect || showIncorrect ? 'white' : isSelected ? '#58cc02' : 'white',
                          }}
                        >
                          {choice}
                        </Text>
                        {showCorrect && <IconCheck size={24} style={{ color: 'white' }} />}
                      </Group>
                    </Paper>
                  )
                })}
              </Stack>
            </Stack>
          </motion.div>
        </AnimatePresence>
      </Container>

      {/* Footer */}
      <Paper
        p="lg"
        radius={0}
        style={{
          backgroundColor: isChecked ? (isCorrect ? '#58cc02' : '#ff4b4b') : '#1a2c33',
          borderTop: '2px solid #3d5a68',
          position: 'fixed',
          bottom: 0,
          left: 0,
          right: 0,
        }}
      >
        <Container size="sm">
          {isChecked ? (
            <Group justify="space-between">
              <div>
                <Text size="lg" fw={700} style={{ color: 'white' }}>
                  {isCorrect ? 'Correct!' : 'Incorrect'}
                </Text>
                {!isCorrect && (
                  <Text size="sm" style={{ color: 'rgba(255,255,255,0.8)' }}>
                    Correct answer: {currentExercise.correctAnswer}
                  </Text>
                )}
              </div>
              <Button size="lg" color="white" variant="white" onClick={handleContinue}>
                Continue
              </Button>
            </Group>
          ) : (
            <Group justify="space-between">
              <Button size="lg" variant="subtle" color="gray" onClick={handleQuit}>
                Skip
              </Button>
              <Button
                size="lg"
                color="green"
                disabled={!selectedAnswer}
                onClick={handleCheck}
              >
                Check
              </Button>
            </Group>
          )}
        </Container>
      </Paper>
    </div>
  )
}
