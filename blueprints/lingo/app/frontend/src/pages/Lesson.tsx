import { useState, useEffect } from 'react'
import { Container, Paper, Title, Text, Button, Group, Stack, Progress, ActionIcon, Loader, Center, Badge } from '@mantine/core'
import { IconX, IconHeart, IconCheck, IconVolume, IconSettings } from '@tabler/icons-react'
import { useNavigate, useParams } from 'react-router-dom'
import { motion, AnimatePresence } from 'framer-motion'
import { useAuthStore } from '../stores/auth'
import { colors } from '../styles/tokens'
import { lessonsApi, coursesApi, Exercise, Skill, Lesson as LessonType } from '../api/client'

export default function Lesson() {
  const navigate = useNavigate()
  const { id: skillId } = useParams<{ id: string }>()
  const { user, updateUser } = useAuthStore()

  const [loading, setLoading] = useState(true)
  const [skill, setSkill] = useState<Skill | null>(null)
  const [lesson, setLesson] = useState<LessonType | null>(null)
  const [exercises, setExercises] = useState<Exercise[]>([])
  const [currentIndex, setCurrentIndex] = useState(0)
  const [selectedAnswer, setSelectedAnswer] = useState<string | null>(null)
  const [isChecked, setIsChecked] = useState(false)
  const [isCorrect, setIsCorrect] = useState(false)
  const [hearts, setHearts] = useState(user?.hearts || 5)
  const [xpEarned, setXpEarned] = useState(0)
  const [mistakes, setMistakes] = useState(0)

  useEffect(() => {
    async function loadLessonData() {
      if (!skillId) return

      try {
        setLoading(true)

        // First get the skill details to find the first lesson
        const skillData = await coursesApi.getCoursePath(skillId).catch(() => null)

        // For now, we'll get the lesson directly using the skill ID as a lesson ID
        // In a real implementation, we'd have a proper skill -> lesson relationship
        const lessonData = await lessonsApi.getLesson(skillId)

        if (lessonData) {
          setLesson(lessonData)

          // Get exercises for this lesson
          // The lesson object should contain exercises
          if (lessonData.exercises && lessonData.exercises.length > 0) {
            setExercises(lessonData.exercises)
          } else {
            // Generate mock exercises if none exist
            setExercises(generateMockExercises())
          }
        } else {
          // Use mock exercises if API fails
          setExercises(generateMockExercises())
        }
      } catch (err) {
        console.error('Failed to load lesson:', err)
        // Fall back to mock data
        setExercises(generateMockExercises())
      } finally {
        setLoading(false)
      }
    }

    loadLessonData()
  }, [skillId])

  // Generate mock exercises as fallback
  function generateMockExercises(): Exercise[] {
    return [
      {
        id: '1',
        lesson_id: skillId || '',
        type: 'multiple_choice',
        prompt: 'What does "Hola" mean?',
        correct_answer: 'Hello',
        choices: ['Hello', 'Goodbye', 'Thank you', 'Please'],
        difficulty: 1,
      },
      {
        id: '2',
        lesson_id: skillId || '',
        type: 'translation',
        prompt: 'Translate: Buenos dias',
        correct_answer: 'Good morning',
        choices: ['Good morning', 'Good night', 'Good afternoon', 'Good evening'],
        difficulty: 1,
      },
      {
        id: '3',
        lesson_id: skillId || '',
        type: 'multiple_choice',
        prompt: 'What does "Gracias" mean?',
        correct_answer: 'Thank you',
        choices: ['Hello', 'Goodbye', 'Thank you', 'Please'],
        difficulty: 1,
      },
      {
        id: '4',
        lesson_id: skillId || '',
        type: 'translation',
        prompt: 'Translate: Adios',
        correct_answer: 'Goodbye',
        choices: ['Hello', 'Goodbye', 'Thank you', 'Please'],
        difficulty: 1,
      },
      {
        id: '5',
        lesson_id: skillId || '',
        type: 'multiple_choice',
        prompt: 'What does "Por favor" mean?',
        correct_answer: 'Please',
        choices: ['Hello', 'Goodbye', 'Thank you', 'Please'],
        difficulty: 1,
      },
    ]
  }

  const currentExercise = exercises[currentIndex]
  const progress = exercises.length > 0 ? ((currentIndex + 1) / exercises.length) * 100 : 0

  const handleCheck = async () => {
    if (!selectedAnswer || !currentExercise) return

    const correct = selectedAnswer === currentExercise.correct_answer
    setIsCorrect(correct)
    setIsChecked(true)

    // Try to submit answer to API
    try {
      await lessonsApi.answerExercise(currentExercise.id, selectedAnswer)
    } catch (err) {
      console.log('Failed to submit answer to API, continuing locally')
    }

    if (correct) {
      setXpEarned((prev) => prev + 3)
    } else {
      setMistakes((prev) => prev + 1)
      setHearts((prev) => Math.max(0, prev - 1))
    }
  }

  const handleContinue = () => {
    if (currentIndex < exercises.length - 1) {
      setCurrentIndex((prev) => prev + 1)
      setSelectedAnswer(null)
      setIsChecked(false)
    } else {
      // Lesson complete
      const finalXP = xpEarned + (mistakes === 0 ? 5 : 0) // Bonus for perfect

      // Try to complete lesson via API
      lessonsApi.completeLesson(lesson?.id || skillId || '', {
        mistakes_count: mistakes,
        hearts_lost: (user?.hearts || 5) - hearts,
      }).catch(() => {
        console.log('Failed to complete lesson via API')
      })

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

  if (loading) {
    return (
      <div style={{ minHeight: '100vh', backgroundColor: colors.neutral.white, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
        <Stack align="center" gap="md">
          <Loader size="lg" color="green" />
          <Text c={colors.text.secondary}>Loading lesson...</Text>
        </Stack>
      </div>
    )
  }

  if (!currentExercise) {
    return (
      <div style={{ minHeight: '100vh', backgroundColor: colors.neutral.white, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
        <Text c={colors.text.secondary}>No exercises found</Text>
      </div>
    )
  }

  const getExerciseTypeLabel = (type: string) => {
    switch (type) {
      case 'translation':
        return 'Translate this phrase'
      case 'multiple_choice':
        return 'Select the correct answer'
      case 'listening':
        return 'Type what you hear'
      case 'word_bank':
        return 'Build the sentence'
      case 'fill_blank':
        return 'Complete the sentence'
      default:
        return 'Answer the question'
    }
  }

  return (
    <div style={{ minHeight: '100vh', backgroundColor: colors.neutral.white }}>
      {/* Header */}
      <Paper
        p="md"
        radius={0}
        style={{
          backgroundColor: colors.neutral.white,
          borderBottom: `2px solid ${colors.neutral.border}`,
          position: 'sticky',
          top: 0,
          zIndex: 100,
        }}
      >
        <Container size="md">
          <Group justify="space-between">
            <ActionIcon variant="subtle" size="lg" onClick={handleQuit}>
              <IconX size={24} style={{ color: colors.text.secondary }} />
            </ActionIcon>

            <ActionIcon variant="subtle" size="lg">
              <IconSettings size={20} style={{ color: colors.text.secondary }} />
            </ActionIcon>

            <Progress
              value={progress}
              size="lg"
              radius="xl"
              color="green"
              style={{ flex: 1, margin: '0 20px' }}
            />

            <Group gap={4}>
              <IconHeart size={24} style={{ color: colors.accent.pink, fill: colors.accent.pink }} />
              <Text fw={700} style={{ color: colors.accent.pink }}>{hearts}</Text>
            </Group>
          </Group>
        </Container>
      </Paper>

      {/* Exercise Content */}
      <Container size="sm" py="xl" style={{ paddingBottom: 120 }}>
        <AnimatePresence mode="wait">
          <motion.div
            key={currentIndex}
            initial={{ opacity: 0, x: 20 }}
            animate={{ opacity: 1, x: 0 }}
            exit={{ opacity: 0, x: -20 }}
          >
            <Stack gap="xl">
              {/* Exercise type badge */}
              {currentExercise.type === 'translation' && (
                <Badge
                  size="lg"
                  variant="light"
                  style={{ backgroundColor: colors.accent.purpleLight, color: colors.accent.purple, alignSelf: 'flex-start' }}
                >
                  NEW WORD
                </Badge>
              )}

              {/* Prompt */}
              <div>
                <Text size="sm" tt="uppercase" fw={700} style={{ color: colors.text.secondary }} mb="xs">
                  {getExerciseTypeLabel(currentExercise.type)}
                </Text>
                <Group gap="md" align="center">
                  {currentExercise.audio_url && (
                    <ActionIcon variant="filled" color="blue" size="xl" radius="xl">
                      <IconVolume size={20} />
                    </ActionIcon>
                  )}
                  <Title order={2} style={{ color: colors.text.primary }}>
                    {currentExercise.prompt}
                  </Title>
                </Group>
              </div>

              {/* Hint tooltip */}
              {currentExercise.hints && currentExercise.hints.length > 0 && (
                <Paper
                  p="md"
                  radius="lg"
                  style={{
                    backgroundColor: colors.neutral.background,
                    border: `2px solid ${colors.neutral.border}`,
                  }}
                >
                  <Text size="sm" c={colors.text.secondary}>
                    {currentExercise.hints[0]}
                  </Text>
                </Paper>
              )}

              {/* Choices */}
              <Stack gap="md">
                {currentExercise.choices?.map((choice) => {
                  const isSelected = selectedAnswer === choice
                  const showCorrect = isChecked && choice === currentExercise.correct_answer
                  const showIncorrect = isChecked && isSelected && !isCorrect

                  return (
                    <Paper
                      key={choice}
                      p="lg"
                      radius="lg"
                      onClick={() => !isChecked && setSelectedAnswer(choice)}
                      className={`choice-button ${isSelected ? 'selected' : ''} ${showCorrect ? 'correct' : ''} ${showIncorrect ? 'incorrect' : ''}`}
                      style={{
                        backgroundColor: showCorrect
                          ? colors.semantic.successLight
                          : showIncorrect
                          ? colors.semantic.errorLight
                          : isSelected
                          ? colors.secondary.blueLight
                          : colors.neutral.white,
                        border: `2px solid ${
                          showCorrect
                            ? colors.semantic.success
                            : showIncorrect
                            ? colors.semantic.error
                            : isSelected
                            ? colors.secondary.blue
                            : colors.neutral.border
                        }`,
                        cursor: isChecked ? 'default' : 'pointer',
                        transition: 'all 0.15s ease',
                      }}
                    >
                      <Group justify="space-between">
                        <Text
                          size="lg"
                          fw={600}
                          style={{
                            color: showCorrect
                              ? colors.semantic.success
                              : showIncorrect
                              ? colors.semantic.error
                              : isSelected
                              ? colors.secondary.blue
                              : colors.text.primary,
                          }}
                        >
                          {choice}
                        </Text>
                        {showCorrect && <IconCheck size={24} style={{ color: colors.semantic.success }} />}
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
          backgroundColor: isChecked
            ? isCorrect
              ? colors.semantic.successLight
              : colors.semantic.errorLight
            : colors.neutral.white,
          borderTop: `2px solid ${colors.neutral.border}`,
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
                <Text size="lg" fw={700} style={{ color: isCorrect ? colors.semantic.success : colors.semantic.error }}>
                  {isCorrect ? 'Correct!' : 'Incorrect'}
                </Text>
                {!isCorrect && (
                  <Text size="sm" style={{ color: colors.semantic.error }}>
                    Correct answer: {currentExercise.correct_answer}
                  </Text>
                )}
              </div>
              <Button
                size="lg"
                color={isCorrect ? 'green' : 'red'}
                onClick={handleContinue}
                style={{
                  fontWeight: 700,
                  textTransform: 'uppercase',
                  boxShadow: isCorrect ? '0 4px 0 #58A700' : '0 4px 0 #EA2B2B',
                }}
              >
                Continue
              </Button>
            </Group>
          ) : (
            <Group justify="space-between">
              <Button
                size="lg"
                variant="subtle"
                color="gray"
                onClick={handleQuit}
                style={{ fontWeight: 700, textTransform: 'uppercase' }}
              >
                Skip
              </Button>
              <Button
                size="lg"
                color="green"
                disabled={!selectedAnswer}
                onClick={handleCheck}
                style={{
                  fontWeight: 700,
                  textTransform: 'uppercase',
                  boxShadow: selectedAnswer ? '0 4px 0 #58A700' : '0 4px 0 #CDCDCD',
                }}
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
