import { useState, useEffect, useRef, useCallback } from 'react'
import { Container, Paper, Title, Text, Button, Group, Stack, Progress, ActionIcon, Loader, Badge, TextInput } from '@mantine/core'
import { IconX, IconHeart, IconCheck, IconVolume, IconVolume2 } from '@tabler/icons-react'
import { useNavigate, useParams } from 'react-router-dom'
import { motion, AnimatePresence } from 'framer-motion'
import { useAuthStore } from '../stores/auth'
import { colors } from '../styles/tokens'
import { lessonsApi, Exercise, Lesson as LessonType } from '../api/client'
import { sounds, playSound } from '../utils/sounds'

// Audio playback hook
function useAudio() {
  const audioRef = useRef<HTMLAudioElement | null>(null)
  const [isPlaying, setIsPlaying] = useState(false)

  const playAudio = useCallback((url: string, slow = false) => {
    if (!url) return

    // Stop any currently playing audio
    if (audioRef.current) {
      audioRef.current.pause()
      audioRef.current = null
    }

    // Apply slow mode for Google TTS if requested
    let audioUrl = url
    if (slow && url.includes('translate.google.com')) {
      audioUrl = url.includes('?') ? `${url}&ttsspeed=0.3` : `${url}?ttsspeed=0.3`
    }

    const audio = new Audio(audioUrl)
    audioRef.current = audio
    setIsPlaying(true)

    audio.onended = () => setIsPlaying(false)
    audio.onerror = () => setIsPlaying(false)

    audio.play().catch(() => setIsPlaying(false))
  }, [])

  return { playAudio, isPlaying }
}

// Word Bank Exercise Component - Duolingo style
function WordBankExercise({
  exercise,
  selectedWords,
  onSelectWord,
  onRemoveWord,
  isChecked,
  isCorrect,
}: {
  exercise: Exercise
  selectedWords: string[]
  onSelectWord: (word: string) => void
  onRemoveWord: (index: number) => void
  isChecked: boolean
  isCorrect: boolean
}) {
  const availableWords = exercise.choices || []

  return (
    <Stack gap="lg">
      {/* Answer area - where selected words appear */}
      <Paper
        p="lg"
        radius="lg"
        style={{
          backgroundColor: isChecked
            ? isCorrect
              ? colors.semantic.successLight
              : colors.semantic.errorLight
            : colors.neutral.background,
          border: `2px ${isChecked ? 'solid' : 'dashed'} ${
            isChecked
              ? isCorrect
                ? colors.semantic.success
                : colors.semantic.error
              : colors.neutral.border
          }`,
          minHeight: 70,
        }}
      >
        <Group gap="sm" wrap="wrap">
          {selectedWords.length === 0 ? (
            <Text c={colors.text.muted} size="sm">Tap the words to form the answer</Text>
          ) : (
            selectedWords.map((word, index) => (
              <motion.div
                key={`${word}-${index}`}
                initial={{ scale: 0.8, opacity: 0 }}
                animate={{ scale: 1, opacity: 1 }}
                transition={{ duration: 0.15 }}
              >
                <Badge
                  size="xl"
                  variant="filled"
                  color={isChecked ? (isCorrect ? 'green' : 'red') : 'blue'}
                  style={{
                    cursor: isChecked ? 'default' : 'pointer',
                    padding: '12px 16px',
                    fontSize: '1rem',
                  }}
                  onClick={() => !isChecked && onRemoveWord(index)}
                >
                  {word}
                </Badge>
              </motion.div>
            ))
          )}
        </Group>
      </Paper>

      {/* Word choices */}
      <Group gap="sm" wrap="wrap" justify="center">
        {availableWords.map((word, index) => {
          const usedCount = selectedWords.filter(w => w === word).length
          const totalCount = availableWords.filter(w => w === word).length
          const isFullyUsed = usedCount >= totalCount

          return (
            <motion.div
              key={`${word}-${index}`}
              whileTap={{ scale: 0.95 }}
            >
              <Paper
                p="md"
                radius="lg"
                onClick={() => {
                  if (!isChecked && !isFullyUsed) {
                    playSound('click', 0.2)
                    onSelectWord(word)
                  }
                }}
                style={{
                  backgroundColor: isFullyUsed ? colors.neutral.background : colors.neutral.white,
                  border: `2px solid ${colors.neutral.border}`,
                  cursor: isChecked || isFullyUsed ? 'default' : 'pointer',
                  opacity: isFullyUsed ? 0.4 : 1,
                  boxShadow: isFullyUsed ? 'none' : '0 2px 0 #E5E5E5',
                  transition: 'all 0.1s ease',
                }}
              >
                <Text fw={600} style={{ color: isFullyUsed ? colors.text.muted : colors.text.primary }}>
                  {word}
                </Text>
              </Paper>
            </motion.div>
          )
        })}
      </Group>

      {/* Show correct answer if wrong */}
      {isChecked && !isCorrect && (
        <Text size="sm" c={colors.semantic.error}>
          Correct answer: {exercise.correct_answer}
        </Text>
      )}
    </Stack>
  )
}

// Fill in the Blank Exercise Component
function FillBlankExercise({
  exercise,
  selectedAnswer,
  onSelect,
  isChecked,
  isCorrect,
}: {
  exercise: Exercise
  selectedAnswer: string | null
  onSelect: (answer: string) => void
  isChecked: boolean
  isCorrect: boolean
}) {
  // Parse the prompt to find the blank (marked with ___ or [blank])
  const promptParts = exercise.prompt.split(/___|\[blank\]/)
  const hasBlank = promptParts.length > 1

  return (
    <Stack gap="lg">
      {/* Sentence with blank */}
      <Paper
        p="lg"
        radius="lg"
        style={{
          backgroundColor: colors.neutral.background,
          border: `2px solid ${colors.neutral.border}`,
        }}
      >
        <Text size="lg" style={{ color: colors.text.primary, lineHeight: 2 }}>
          {hasBlank ? (
            <>
              {promptParts[0]}
              <Badge
                size="xl"
                variant={selectedAnswer ? 'filled' : 'outline'}
                color={
                  isChecked
                    ? isCorrect
                      ? 'green'
                      : 'red'
                    : selectedAnswer
                    ? 'blue'
                    : 'gray'
                }
                style={{
                  margin: '0 8px',
                  padding: selectedAnswer ? '8px 16px' : '8px 32px',
                  minWidth: 80,
                }}
              >
                {selectedAnswer || '___'}
              </Badge>
              {promptParts[1]}
            </>
          ) : (
            exercise.prompt
          )}
        </Text>
      </Paper>

      {/* Choices */}
      <Group gap="md" justify="center" wrap="wrap">
        {exercise.choices?.map((choice) => {
          const isSelected = selectedAnswer === choice
          const showCorrect = isChecked && choice === exercise.correct_answer
          const showIncorrect = isChecked && isSelected && !isCorrect

          return (
            <motion.div key={choice} whileTap={{ scale: 0.95 }}>
              <Paper
                p="md"
                px="xl"
                radius="xl"
                onClick={() => {
                  if (!isChecked) {
                    playSound('click', 0.2)
                    onSelect(choice)
                  }
                }}
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
                  boxShadow: isChecked ? 'none' : '0 3px 0 #E5E5E5',
                }}
              >
                <Text
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
              </Paper>
            </motion.div>
          )
        })}
      </Group>
    </Stack>
  )
}

// Match Pairs Exercise Component - Two column matching
function MatchPairsExercise({
  exercise,
  matchedPairs,
  selectedLeft,
  selectedRight,
  onSelectLeft,
  onSelectRight,
  isChecked,
}: {
  exercise: Exercise
  matchedPairs: Map<string, string>
  selectedLeft: string | null
  selectedRight: string | null
  onSelectLeft: (word: string) => void
  onSelectRight: (word: string) => void
  isChecked: boolean
}) {
  // Parse choices into left and right columns
  // Format: "word|translation" or just split choices into two groups
  const pairs = exercise.choices?.map(c => c.split('|')) || []
  const leftWords = pairs.map(p => p[0])
  const rightWords = pairs.map(p => p[1] || p[0]).sort(() => Math.random() - 0.5)

  return (
    <Stack gap="lg">
      <Text size="sm" c={colors.text.secondary} ta="center">
        Match the words with their translations
      </Text>

      <Group gap="xl" justify="center" align="flex-start">
        {/* Left column */}
        <Stack gap="sm">
          {leftWords.map((word) => {
            const isMatched = matchedPairs.has(word)
            const isSelected = selectedLeft === word

            return (
              <motion.div key={word} whileTap={{ scale: 0.95 }}>
                <Paper
                  p="md"
                  radius="lg"
                  onClick={() => !isChecked && !isMatched && onSelectLeft(word)}
                  style={{
                    backgroundColor: isMatched
                      ? colors.semantic.successLight
                      : isSelected
                      ? colors.secondary.blueLight
                      : colors.neutral.white,
                    border: `2px solid ${
                      isMatched
                        ? colors.semantic.success
                        : isSelected
                        ? colors.secondary.blue
                        : colors.neutral.border
                    }`,
                    cursor: isChecked || isMatched ? 'default' : 'pointer',
                    minWidth: 120,
                    textAlign: 'center',
                    opacity: isMatched ? 0.7 : 1,
                  }}
                >
                  <Text fw={600} style={{ color: colors.text.primary }}>
                    {word}
                  </Text>
                </Paper>
              </motion.div>
            )
          })}
        </Stack>

        {/* Right column */}
        <Stack gap="sm">
          {rightWords.map((word) => {
            const isMatched = Array.from(matchedPairs.values()).includes(word)
            const isSelected = selectedRight === word

            return (
              <motion.div key={word} whileTap={{ scale: 0.95 }}>
                <Paper
                  p="md"
                  radius="lg"
                  onClick={() => !isChecked && !isMatched && onSelectRight(word)}
                  style={{
                    backgroundColor: isMatched
                      ? colors.semantic.successLight
                      : isSelected
                      ? colors.secondary.blueLight
                      : colors.neutral.white,
                    border: `2px solid ${
                      isMatched
                        ? colors.semantic.success
                        : isSelected
                        ? colors.secondary.blue
                        : colors.neutral.border
                    }`,
                    cursor: isChecked || isMatched ? 'default' : 'pointer',
                    minWidth: 120,
                    textAlign: 'center',
                    opacity: isMatched ? 0.7 : 1,
                  }}
                >
                  <Text fw={600} style={{ color: colors.text.primary }}>
                    {word}
                  </Text>
                </Paper>
              </motion.div>
            )
          })}
        </Stack>
      </Group>
    </Stack>
  )
}

// Multiple Choice Exercise Component
function MultipleChoiceExercise({
  exercise,
  selectedAnswer,
  onSelect,
  isChecked,
  isCorrect,
}: {
  exercise: Exercise
  selectedAnswer: string | null
  onSelect: (answer: string) => void
  isChecked: boolean
  isCorrect: boolean
}) {
  return (
    <Stack gap="md">
      {exercise.choices?.map((choice) => {
        const isSelected = selectedAnswer === choice
        const showCorrect = isChecked && choice === exercise.correct_answer
        const showIncorrect = isChecked && isSelected && !isCorrect

        return (
          <motion.div key={choice} whileTap={{ scale: 0.98 }}>
            <Paper
              p="lg"
              radius="lg"
              onClick={() => {
                if (!isChecked) {
                  playSound('click', 0.2)
                  onSelect(choice)
                }
              }}
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
                boxShadow: isChecked
                  ? 'none'
                  : isSelected
                  ? '0 4px 0 #1899D6'
                  : '0 4px 0 #E5E5E5',
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
          </motion.div>
        )
      })}
    </Stack>
  )
}

// Listening/Typing Exercise Component
function TypingExercise({
  exercise,
  typedAnswer,
  onType,
  isChecked,
  isCorrect,
}: {
  exercise: Exercise
  typedAnswer: string
  onType: (value: string) => void
  isChecked: boolean
  isCorrect: boolean
}) {
  return (
    <Stack gap="md">
      <TextInput
        placeholder="Type your answer..."
        size="lg"
        value={typedAnswer}
        onChange={(e) => !isChecked && onType(e.target.value)}
        disabled={isChecked}
        onKeyDown={(e) => e.key === 'Enter' && !isChecked}
        styles={{
          input: {
            backgroundColor: isChecked
              ? isCorrect
                ? colors.semantic.successLight
                : colors.semantic.errorLight
              : colors.neutral.white,
            borderColor: isChecked
              ? isCorrect
                ? colors.semantic.success
                : colors.semantic.error
              : colors.neutral.border,
            borderWidth: 2,
            fontSize: '1.1rem',
            padding: '1.5rem 1rem',
            borderRadius: 16,
          },
        }}
      />
      {isChecked && !isCorrect && (
        <Text size="sm" c={colors.semantic.error}>
          Correct answer: {exercise.correct_answer}
        </Text>
      )}
    </Stack>
  )
}

export default function Lesson() {
  const navigate = useNavigate()
  const { id: skillId } = useParams<{ id: string }>()
  const { user, updateUser } = useAuthStore()
  const { playAudio, isPlaying } = useAudio()

  const [loading, setLoading] = useState(true)
  const [lesson, setLesson] = useState<LessonType | null>(null)
  const [exercises, setExercises] = useState<Exercise[]>([])
  const [currentIndex, setCurrentIndex] = useState(0)
  const [selectedAnswer, setSelectedAnswer] = useState<string | null>(null)
  const [selectedWords, setSelectedWords] = useState<string[]>([])
  const [typedAnswer, setTypedAnswer] = useState('')
  const [isChecked, setIsChecked] = useState(false)
  const [isCorrect, setIsCorrect] = useState(false)
  const [hearts, setHearts] = useState(user?.hearts || 5)
  const [xpEarned, setXpEarned] = useState(0)
  const [mistakes, setMistakes] = useState(0)

  // Match pairs state
  const [matchedPairs, setMatchedPairs] = useState<Map<string, string>>(new Map())
  const [selectedLeft, setSelectedLeft] = useState<string | null>(null)
  const [selectedRight, setSelectedRight] = useState<string | null>(null)

  useEffect(() => {
    async function loadLessonData() {
      if (!skillId) return

      try {
        setLoading(true)

        // Get the first lesson for this skill with exercises
        const response = await lessonsApi.getLessonBySkill(skillId)

        if (response && response.lesson) {
          setLesson(response.lesson)

          if (response.exercises && response.exercises.length > 0) {
            console.log(`Loaded ${response.exercises.length} exercises from database`)
            setExercises(response.exercises)
          } else {
            console.log('No exercises found in database, using mock data')
            setExercises(generateMockExercises())
          }
        } else {
          console.log('No lesson data returned, using mock data')
          setExercises(generateMockExercises())
        }
      } catch (err) {
        console.error('Failed to load lesson:', err)
        setExercises(generateMockExercises())
      } finally {
        setLoading(false)
      }
    }

    loadLessonData()
  }, [skillId])

  // Handle match pairs selection
  useEffect(() => {
    if (selectedLeft && selectedRight) {
      // Check if this is a correct match
      const exercise = exercises[currentIndex]
      const pairs = exercise?.choices?.map(c => c.split('|')) || []
      const isMatch = pairs.some(p => p[0] === selectedLeft && p[1] === selectedRight)

      if (isMatch) {
        playSound('correct', 0.4)
        setMatchedPairs(prev => new Map(prev).set(selectedLeft, selectedRight))
      } else {
        playSound('wrong', 0.3)
      }

      setSelectedLeft(null)
      setSelectedRight(null)
    }
  }, [selectedLeft, selectedRight, currentIndex, exercises])

  function generateMockExercises(): Exercise[] {
    return [
      {
        id: '1',
        lesson_id: skillId || '',
        type: 'multiple_choice',
        prompt: 'What does "你好" mean?',
        correct_answer: 'Hello',
        choices: ['Hello', 'Goodbye', 'Thank you', 'Please'],
        difficulty: 1,
      },
      {
        id: '2',
        lesson_id: skillId || '',
        type: 'word_bank',
        prompt: 'Build: "Good morning"',
        correct_answer: 'Good morning',
        choices: ['morning', 'Good', 'night', 'Hello'],
        difficulty: 1,
      },
      {
        id: '3',
        lesson_id: skillId || '',
        type: 'fill_blank',
        prompt: 'The word for hello is ___',
        correct_answer: '你好',
        choices: ['你好', '再见', '谢谢', '对不起'],
        difficulty: 1,
      },
      {
        id: '4',
        lesson_id: skillId || '',
        type: 'translation',
        prompt: 'Translate: 再见',
        correct_answer: 'Goodbye',
        choices: ['Hello', 'Goodbye', 'Thank you', 'Please'],
        difficulty: 1,
      },
      {
        id: '5',
        lesson_id: skillId || '',
        type: 'multiple_choice',
        prompt: 'What does "谢谢" mean?',
        correct_answer: 'Thank you',
        choices: ['Hello', 'Goodbye', 'Thank you', 'Please'],
        difficulty: 1,
      },
    ]
  }

  const currentExercise = exercises[currentIndex]
  const progress = exercises.length > 0 ? ((currentIndex + 1) / exercises.length) * 100 : 0

  const handleCheck = async () => {
    if (!currentExercise) return

    let userAnswer: string
    const exerciseType = currentExercise.type

    if (exerciseType === 'word_bank') {
      userAnswer = selectedWords.join(' ')
    } else if (exerciseType === 'listening' || (exerciseType === 'translation' && (!currentExercise.choices || currentExercise.choices.length === 0))) {
      userAnswer = typedAnswer.trim()
    } else if (exerciseType === 'match_pairs') {
      // For match pairs, check if all pairs are matched
      const allPairs = currentExercise.choices?.length || 0
      const correct = matchedPairs.size === allPairs
      setIsCorrect(correct)
      setIsChecked(true)
      if (correct) {
        setXpEarned((prev) => prev + 3)
        sounds.correctAnswer()
      } else {
        setMistakes((prev) => prev + 1)
        setHearts((prev) => Math.max(0, prev - 1))
        sounds.wrongAnswer()
      }
      return
    } else {
      if (!selectedAnswer) return
      userAnswer = selectedAnswer
    }

    if (!userAnswer) return

    const correct = userAnswer.toLowerCase() === currentExercise.correct_answer.toLowerCase()
    setIsCorrect(correct)
    setIsChecked(true)

    try {
      await lessonsApi.answerExercise(currentExercise.id, userAnswer)
    } catch (err) {
      console.log('Failed to submit answer to API')
    }

    if (correct) {
      setXpEarned((prev) => prev + 3)
      sounds.correctAnswer()
    } else {
      setMistakes((prev) => prev + 1)
      setHearts((prev) => Math.max(0, prev - 1))
      sounds.wrongAnswer()
    }
  }

  const handleContinue = () => {
    if (currentIndex < exercises.length - 1) {
      setCurrentIndex((prev) => prev + 1)
      setSelectedAnswer(null)
      setSelectedWords([])
      setTypedAnswer('')
      setIsChecked(false)
      setMatchedPairs(new Map())
      setSelectedLeft(null)
      setSelectedRight(null)
      sounds.buttonClick()
    } else {
      const finalXP = xpEarned + (mistakes === 0 ? 5 : 0)
      sounds.lessonComplete()

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

  const hasAnswer = () => {
    if (!currentExercise) return false
    const exerciseType = currentExercise.type

    if (exerciseType === 'word_bank') {
      return selectedWords.length > 0
    }
    if (exerciseType === 'match_pairs') {
      const allPairs = currentExercise.choices?.length || 0
      return matchedPairs.size === allPairs
    }
    if (exerciseType === 'listening' ||
        (exerciseType === 'translation' && (!currentExercise.choices || currentExercise.choices.length === 0))) {
      return typedAnswer.trim().length > 0
    }
    return selectedAnswer !== null
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
      case 'match_pairs':
        return 'Match the pairs'
      default:
        return 'Answer the question'
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
      <Container size="sm" py="xl" style={{ paddingBottom: 140 }}>
        <AnimatePresence mode="wait">
          <motion.div
            key={currentIndex}
            initial={{ opacity: 0, x: 20 }}
            animate={{ opacity: 1, x: 0 }}
            exit={{ opacity: 0, x: -20 }}
          >
            <Stack gap="xl">
              {/* Exercise type label */}
              <Text size="sm" tt="uppercase" fw={700} style={{ color: colors.text.secondary }}>
                {getExerciseTypeLabel(currentExercise.type)}
              </Text>

              {/* Prompt with audio */}
              <Group gap="md" align="center">
                {currentExercise.audio_url && (
                  <Group gap="xs">
                    <ActionIcon
                      variant="filled"
                      color="blue"
                      size="xl"
                      radius="xl"
                      onClick={() => playAudio(currentExercise.audio_url!)}
                      style={{ opacity: isPlaying ? 0.7 : 1 }}
                    >
                      <IconVolume size={20} />
                    </ActionIcon>
                    <ActionIcon
                      variant="light"
                      color="blue"
                      size="lg"
                      radius="xl"
                      onClick={() => playAudio(currentExercise.audio_url!, true)}
                      title="Play slowly"
                    >
                      <IconVolume2 size={16} />
                    </ActionIcon>
                  </Group>
                )}
                {currentExercise.type === 'listening' ? (
                  <Title order={2} style={{ color: colors.text.muted }}>
                    Type what you hear
                  </Title>
                ) : currentExercise.type !== 'fill_blank' && (
                  <Title order={2} style={{ color: colors.text.primary }}>
                    {currentExercise.prompt}
                  </Title>
                )}
              </Group>

              {/* Hint */}
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

              {/* Exercise Component based on type */}
              {currentExercise.type === 'word_bank' ? (
                <WordBankExercise
                  exercise={currentExercise}
                  selectedWords={selectedWords}
                  onSelectWord={(word) => setSelectedWords([...selectedWords, word])}
                  onRemoveWord={(index) => setSelectedWords(selectedWords.filter((_, i) => i !== index))}
                  isChecked={isChecked}
                  isCorrect={isCorrect}
                />
              ) : currentExercise.type === 'fill_blank' ? (
                <FillBlankExercise
                  exercise={currentExercise}
                  selectedAnswer={selectedAnswer}
                  onSelect={setSelectedAnswer}
                  isChecked={isChecked}
                  isCorrect={isCorrect}
                />
              ) : currentExercise.type === 'match_pairs' ? (
                <MatchPairsExercise
                  exercise={currentExercise}
                  matchedPairs={matchedPairs}
                  selectedLeft={selectedLeft}
                  selectedRight={selectedRight}
                  onSelectLeft={setSelectedLeft}
                  onSelectRight={setSelectedRight}
                  isChecked={isChecked}
                />
              ) : currentExercise.type === 'listening' ||
                (currentExercise.type === 'translation' && (!currentExercise.choices || currentExercise.choices.length === 0)) ? (
                <TypingExercise
                  exercise={currentExercise}
                  typedAnswer={typedAnswer}
                  onType={setTypedAnswer}
                  isChecked={isChecked}
                  isCorrect={isCorrect}
                />
              ) : (
                <MultipleChoiceExercise
                  exercise={currentExercise}
                  selectedAnswer={selectedAnswer}
                  onSelect={setSelectedAnswer}
                  isChecked={isChecked}
                  isCorrect={isCorrect}
                />
              )}
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
                {!isCorrect && currentExercise.type !== 'match_pairs' && (
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
                disabled={!hasAnswer()}
                onClick={handleCheck}
                style={{
                  fontWeight: 700,
                  textTransform: 'uppercase',
                  boxShadow: hasAnswer() ? '0 4px 0 #58A700' : '0 4px 0 #CDCDCD',
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
