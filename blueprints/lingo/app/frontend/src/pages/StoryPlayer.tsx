import { useState, useEffect, useRef } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Stack, Text, Group, Card, Center, Loader, Box, Button, Progress, ActionIcon, UnstyledButton } from '@mantine/core'
import { IconX, IconVolume, IconCheck, IconArrowRight, IconStar, IconTrophy } from '@tabler/icons-react'
import { motion, AnimatePresence } from 'framer-motion'
import { storiesApi, Story, StoryElement, StoryCharacter } from '../api/client'
import { sounds, playTTS, stopSpeaking } from '../utils/sounds'
import { colors } from '../styles/tokens'

// Character avatar component
function CharacterAvatar({ character, isActive = false }: { character?: StoryCharacter | null; isActive?: boolean }) {
  if (!character) return null

  return (
    <Box
      style={{
        width: 56,
        height: 56,
        borderRadius: '50%',
        backgroundColor: isActive ? '#E8F5E9' : '#F5F5F5',
        border: `3px solid ${isActive ? '#58CC02' : '#E5E5E5'}`,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        overflow: 'hidden',
        flexShrink: 0,
      }}
    >
      {character.avatar_url ? (
        <img
          src={character.avatar_url}
          alt={character.display_name || character.name}
          style={{ width: '100%', height: '100%', objectFit: 'cover' }}
        />
      ) : (
        <Text fw={700} size="lg" style={{ color: isActive ? '#58CC02' : '#AFAFAF' }}>
          {(character.display_name || character.name).charAt(0).toUpperCase()}
        </Text>
      )}
    </Box>
  )
}

// Story line component (dialogue)
function StoryLine({
  element,
  character,
  onAudioPlay,
}: {
  element: StoryElement
  character?: StoryCharacter | null
  onAudioPlay?: () => void
}) {
  const [isPlaying, setIsPlaying] = useState(false)

  const playAudio = () => {
    if (element.text) {
      stopSpeaking()
      setIsPlaying(true)
      onAudioPlay?.()

      // Use Web Speech API - try audio_url first to extract lang, fallback to element text
      playTTS(
        element.audio_url,
        element.text,
        undefined, // will try to extract from audio_url
        false,
        () => setIsPlaying(false),
        () => setIsPlaying(false)
      )
    }
  }

  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.3 }}
    >
      <Group align="flex-start" gap="md" wrap="nowrap">
        <CharacterAvatar character={character} isActive={isPlaying} />

        <Card
          shadow="sm"
          radius="lg"
          p="md"
          style={{
            flex: 1,
            backgroundColor: '#FFFFFF',
            border: '2px solid #E5E5E5',
          }}
        >
          <Stack gap="xs">
            {character && (
              <Text fw={700} size="sm" style={{ color: '#58CC02' }}>
                {character.display_name || character.name}
              </Text>
            )}

            <Group justify="space-between" align="flex-start">
              <Box style={{ flex: 1 }}>
                <Text fw={600} size="md" style={{ color: colors.text.primary }}>
                  {element.text}
                </Text>
                {element.translation && (
                  <Text size="sm" c="dimmed" mt={4}>
                    {element.translation}
                  </Text>
                )}
              </Box>

              {element.text && (
                <ActionIcon
                  variant="light"
                  color="blue"
                  size="lg"
                  onClick={playAudio}
                  loading={isPlaying}
                >
                  <IconVolume size={20} />
                </ActionIcon>
              )}
            </Group>
          </Stack>
        </Card>
      </Group>
    </motion.div>
  )
}

// Header element
function StoryHeader({ element }: { element: StoryElement }) {
  return (
    <motion.div
      initial={{ opacity: 0, scale: 0.95 }}
      animate={{ opacity: 1, scale: 1 }}
      transition={{ duration: 0.3 }}
    >
      <Center py="xl">
        <Text
          fw={800}
          size="xl"
          ta="center"
          style={{
            color: colors.text.primary,
            fontFamily: '"Nunito", "DIN Round Pro", sans-serif',
          }}
        >
          {element.text}
        </Text>
      </Center>
    </motion.div>
  )
}

// Multiple choice challenge
function MultipleChoiceChallenge({
  element,
  onAnswer,
}: {
  element: StoryElement
  onAnswer: (correct: boolean) => void
}) {
  const [selected, setSelected] = useState<number | null>(null)
  const [showResult, setShowResult] = useState(false)
  const challenge = element.challenge_data

  if (!challenge?.options) return null

  const handleSelect = (index: number) => {
    if (showResult) return
    setSelected(index)
    sounds.buttonClick()
  }

  const handleCheck = () => {
    if (selected === null) return
    setShowResult(true)

    const isCorrect = selected === challenge.correct_index
    if (isCorrect) {
      sounds.correctAnswer()
    } else {
      sounds.wrongAnswer()
    }

    setTimeout(() => {
      onAnswer(isCorrect)
    }, 1500)
  }

  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
    >
      <Card
        shadow="md"
        radius="lg"
        p="xl"
        style={{
          backgroundColor: '#FFFFFF',
          border: '2px solid #E5E5E5',
        }}
      >
        <Stack gap="lg">
          <Text fw={700} size="lg" ta="center" style={{ color: colors.text.primary }}>
            {challenge.question || element.text}
          </Text>

          {challenge.question_translation && (
            <Text size="sm" c="dimmed" ta="center">
              {challenge.question_translation}
            </Text>
          )}

          <Stack gap="sm">
            {challenge.options.map((option, index) => {
              const isSelected = selected === index
              const isCorrect = index === challenge.correct_index
              const showCorrect = showResult && isCorrect
              const showWrong = showResult && isSelected && !isCorrect

              return (
                <UnstyledButton
                  key={index}
                  onClick={() => handleSelect(index)}
                  style={{
                    padding: '16px 20px',
                    borderRadius: 12,
                    border: `2px solid ${
                      showCorrect ? '#58CC02' :
                      showWrong ? '#FF4B4B' :
                      isSelected ? '#1CB0F6' :
                      '#E5E5E5'
                    }`,
                    backgroundColor: showCorrect ? '#E8F5E9' :
                      showWrong ? '#FFEBEE' :
                      isSelected ? '#E3F2FD' :
                      '#FFFFFF',
                    transition: 'all 0.15s ease',
                  }}
                >
                  <Group>
                    <Box
                      style={{
                        width: 28,
                        height: 28,
                        borderRadius: '50%',
                        border: `2px solid ${
                          showCorrect ? '#58CC02' :
                          showWrong ? '#FF4B4B' :
                          isSelected ? '#1CB0F6' :
                          '#E5E5E5'
                        }`,
                        backgroundColor: isSelected || showResult ? (
                          showCorrect ? '#58CC02' :
                          showWrong ? '#FF4B4B' :
                          '#1CB0F6'
                        ) : 'transparent',
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                      }}
                    >
                      {(isSelected || showResult) && (showCorrect || (isSelected && !showResult)) && (
                        <IconCheck size={16} color="white" />
                      )}
                    </Box>
                    <Text fw={600} style={{ color: colors.text.primary }}>
                      {option}
                    </Text>
                  </Group>
                </UnstyledButton>
              )
            })}
          </Stack>

          {!showResult && (
            <Button
              size="lg"
              color="green"
              radius="xl"
              disabled={selected === null}
              onClick={handleCheck}
              fullWidth
            >
              CHECK
            </Button>
          )}

          {showResult && (
            <Box
              p="md"
              style={{
                borderRadius: 12,
                backgroundColor: selected === challenge.correct_index ? '#E8F5E9' : '#FFEBEE',
              }}
            >
              <Text fw={600} style={{ color: selected === challenge.correct_index ? '#58CC02' : '#FF4B4B' }}>
                {selected === challenge.correct_index
                  ? (challenge.feedback_correct || 'Correct!')
                  : (challenge.feedback_incorrect || 'Not quite...')}
              </Text>
            </Box>
          )}
        </Stack>
      </Card>
    </motion.div>
  )
}

// Arrange words challenge
function ArrangeChallenge({
  element,
  onAnswer,
}: {
  element: StoryElement
  onAnswer: (correct: boolean) => void
}) {
  const challenge = element.challenge_data
  const [arranged, setArranged] = useState<string[]>([])
  const [available, setAvailable] = useState<string[]>(challenge?.arrange_words || [])
  const [showResult, setShowResult] = useState(false)

  if (!challenge?.arrange_words) return null

  const handleWordClick = (word: string, fromArranged: boolean) => {
    if (showResult) return
    sounds.wordSelect()

    if (fromArranged) {
      setArranged(arranged.filter((w) => w !== word))
      setAvailable([...available, word])
    } else {
      setAvailable(available.filter((w) => w !== word))
      setArranged([...arranged, word])
    }
  }

  const handleCheck = () => {
    setShowResult(true)
    // Simple check - compare with correct answer
    const userAnswer = arranged.join(' ')
    const correctAnswer = element.text || ''
    const isCorrect = userAnswer.toLowerCase().trim() === correctAnswer.toLowerCase().trim()

    if (isCorrect) {
      sounds.correctAnswer()
    } else {
      sounds.wrongAnswer()
    }

    setTimeout(() => {
      onAnswer(isCorrect)
    }, 1500)
  }

  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
    >
      <Card shadow="md" radius="lg" p="xl" style={{ border: '2px solid #E5E5E5' }}>
        <Stack gap="lg">
          <Text fw={700} size="lg" ta="center" style={{ color: colors.text.primary }}>
            {challenge.question || 'Arrange the words'}
          </Text>

          {/* Arranged words area */}
          <Box
            style={{
              minHeight: 60,
              padding: 16,
              borderRadius: 12,
              border: '2px dashed #E5E5E5',
              backgroundColor: '#F7F7F7',
            }}
          >
            <Group gap="xs">
              <AnimatePresence mode="popLayout">
                {arranged.map((word, index) => (
                  <motion.div
                    key={`${word}-${index}`}
                    initial={{ scale: 0 }}
                    animate={{ scale: 1 }}
                    exit={{ scale: 0 }}
                    layout
                  >
                    <Button
                      variant="filled"
                      color="blue"
                      radius="md"
                      onClick={() => handleWordClick(word, true)}
                    >
                      {word}
                    </Button>
                  </motion.div>
                ))}
              </AnimatePresence>
            </Group>
          </Box>

          {/* Available words */}
          <Group gap="xs" justify="center">
            <AnimatePresence mode="popLayout">
              {available.map((word, index) => (
                <motion.div
                  key={`${word}-${index}`}
                  initial={{ scale: 0 }}
                  animate={{ scale: 1 }}
                  exit={{ scale: 0 }}
                  layout
                >
                  <Button
                    variant="outline"
                    color="gray"
                    radius="md"
                    onClick={() => handleWordClick(word, false)}
                  >
                    {word}
                  </Button>
                </motion.div>
              ))}
            </AnimatePresence>
          </Group>

          {!showResult && (
            <Button
              size="lg"
              color="green"
              radius="xl"
              disabled={arranged.length === 0}
              onClick={handleCheck}
              fullWidth
            >
              CHECK
            </Button>
          )}
        </Stack>
      </Card>
    </motion.div>
  )
}

// Completion screen
function CompletionScreen({
  story,
  mistakes,
  onClose,
}: {
  story: Story
  mistakes: number
  onClose: () => void
}) {
  const navigate = useNavigate()

  useEffect(() => {
    sounds.lessonComplete()
  }, [])

  const xpEarned = story.xp_reward
  const isPerfect = mistakes === 0

  return (
    <motion.div
      initial={{ opacity: 0, scale: 0.9 }}
      animate={{ opacity: 1, scale: 1 }}
      transition={{ duration: 0.5 }}
      style={{
        position: 'fixed',
        inset: 0,
        backgroundColor: '#FFFFFF',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        zIndex: 1000,
      }}
    >
      <Stack align="center" gap="xl" p="xl">
        {/* Trophy icon */}
        <motion.div
          initial={{ scale: 0 }}
          animate={{ scale: 1 }}
          transition={{ delay: 0.2, type: 'spring', stiffness: 200 }}
        >
          <Box
            style={{
              width: 120,
              height: 120,
              backgroundColor: '#FFC800',
              borderRadius: '50%',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
            }}
          >
            <IconTrophy size={64} color="white" />
          </Box>
        </motion.div>

        <Text
          fw={800}
          size="xl"
          style={{
            color: colors.text.primary,
            fontFamily: '"Nunito", "DIN Round Pro", sans-serif',
          }}
        >
          {isPerfect ? 'Perfect!' : 'Story Complete!'}
        </Text>

        <Text c="dimmed" ta="center">
          You finished "{story.title}"
        </Text>

        {/* XP earned */}
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.4 }}
        >
          <Group gap="lg">
            <Card p="lg" radius="lg" style={{ border: '2px solid #FFC800', textAlign: 'center' }}>
              <Group gap="xs">
                <IconStar size={28} style={{ color: '#FFC800' }} />
                <Text fw={700} size="xl" style={{ color: '#FFC800' }}>
                  +{xpEarned} XP
                </Text>
              </Group>
            </Card>

            {isPerfect && (
              <Card p="lg" radius="lg" style={{ border: '2px solid #58CC02', textAlign: 'center' }}>
                <Group gap="xs">
                  <IconCheck size={28} style={{ color: '#58CC02' }} />
                  <Text fw={700} size="lg" style={{ color: '#58CC02' }}>
                    No mistakes!
                  </Text>
                </Group>
              </Card>
            )}
          </Group>
        </motion.div>

        <Button
          size="lg"
          color="green"
          radius="xl"
          px={48}
          mt="xl"
          onClick={() => {
            onClose()
            navigate('/stories')
          }}
        >
          CONTINUE
        </Button>
      </Stack>
    </motion.div>
  )
}

// Main story player component
export default function StoryPlayer() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [story, setStory] = useState<Story | null>(null)
  const [loading, setLoading] = useState(true)
  const [currentIndex, setCurrentIndex] = useState(0)
  const [visibleElements, setVisibleElements] = useState<StoryElement[]>([])
  const [mistakes, setMistakes] = useState(0)
  const [showCompletion, setShowCompletion] = useState(false)
  const scrollRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (id) {
      loadStory(id)
    }
  }, [id])

  useEffect(() => {
    // Auto-scroll to bottom when new elements appear
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight
    }
  }, [visibleElements])

  const loadStory = async (storyId: string) => {
    try {
      setLoading(true)
      const data = await storiesApi.getStory(storyId)
      setStory(data)
      // Show first element
      if (data.elements && data.elements.length > 0) {
        setVisibleElements([data.elements[0]])
        setCurrentIndex(0)
      }
    } catch (err) {
      console.error('Failed to load story:', err)
    } finally {
      setLoading(false)
    }
  }

  const handleContinue = () => {
    if (!story?.elements) return

    const nextIndex = currentIndex + 1
    if (nextIndex >= story.elements.length) {
      // Story complete
      setShowCompletion(true)
      storiesApi.completeStory(story.id).catch(console.error)
    } else {
      setCurrentIndex(nextIndex)
      setVisibleElements([...visibleElements, story.elements[nextIndex]])
    }
  }

  const handleChallengeAnswer = (correct: boolean) => {
    if (!correct) {
      setMistakes(mistakes + 1)
    }
    // Auto continue after challenge
    setTimeout(handleContinue, 500)
  }

  const getCharacter = (speakerId?: string): StoryCharacter | null => {
    if (!speakerId || !story?.characters) return null
    return story.characters.find((c) => c.id === speakerId) || null
  }

  const renderElement = (element: StoryElement, index: number) => {
    const isLatest = index === visibleElements.length - 1
    const character = getCharacter(element.speaker_id)

    switch (element.element_type) {
      case 'header':
        return <StoryHeader key={element.id} element={element} />

      case 'line':
        return (
          <StoryLine
            key={element.id}
            element={element}
            character={character}
          />
        )

      case 'multiple_choice':
      case 'select_phrase':
        if (isLatest) {
          return (
            <MultipleChoiceChallenge
              key={element.id}
              element={element}
              onAnswer={handleChallengeAnswer}
            />
          )
        }
        return null

      case 'arrange':
        if (isLatest) {
          return (
            <ArrangeChallenge
              key={element.id}
              element={element}
              onAnswer={handleChallengeAnswer}
            />
          )
        }
        return null

      default:
        return (
          <StoryLine
            key={element.id}
            element={element}
            character={character}
          />
        )
    }
  }

  // Check if current element is a challenge
  const currentElement = story?.elements?.[currentIndex]
  const isChallenge = currentElement && [
    'multiple_choice',
    'select_phrase',
    'arrange',
    'match',
    'point_to_phrase',
    'tap_complete',
  ].includes(currentElement.element_type)

  if (loading) {
    return (
      <Center h="100vh">
        <Loader color="green" size="lg" />
      </Center>
    )
  }

  if (!story) {
    return (
      <Center h="100vh">
        <Text>Story not found</Text>
      </Center>
    )
  }

  if (showCompletion) {
    return (
      <CompletionScreen
        story={story}
        mistakes={mistakes}
        onClose={() => setShowCompletion(false)}
      />
    )
  }

  const progress = story.elements ? ((currentIndex + 1) / story.elements.length) * 100 : 0

  return (
    <Box
      style={{
        height: '100vh',
        display: 'flex',
        flexDirection: 'column',
        backgroundColor: '#F7F7F7',
      }}
    >
      {/* Header */}
      <Box
        px="md"
        py="sm"
        style={{
          backgroundColor: '#FFFFFF',
          borderBottom: '1px solid #E5E5E5',
        }}
      >
        <Group justify="space-between">
          <ActionIcon
            variant="subtle"
            color="gray"
            size="lg"
            onClick={() => navigate('/stories')}
          >
            <IconX size={24} />
          </ActionIcon>

          <Progress
            value={progress}
            color="green"
            size="md"
            radius="xl"
            style={{ flex: 1, maxWidth: 400, margin: '0 16px' }}
          />

          <Text fw={600} size="sm" c="dimmed">
            {currentIndex + 1}/{story.elements?.length || 0}
          </Text>
        </Group>
      </Box>

      {/* Story content */}
      <Box
        ref={scrollRef}
        style={{
          flex: 1,
          overflowY: 'auto',
          padding: '24px 16px',
        }}
      >
        <Box maw={600} mx="auto">
          <Stack gap="lg">
            {visibleElements.map((element, index) => renderElement(element, index))}
          </Stack>
        </Box>
      </Box>

      {/* Continue button (only for non-challenge elements) */}
      {!isChallenge && (
        <Box
          p="md"
          style={{
            backgroundColor: '#FFFFFF',
            borderTop: '1px solid #E5E5E5',
          }}
        >
          <Box maw={600} mx="auto">
            <Button
              size="lg"
              color="green"
              radius="xl"
              fullWidth
              onClick={handleContinue}
              rightSection={<IconArrowRight size={20} />}
            >
              CONTINUE
            </Button>
          </Box>
        </Box>
      )}
    </Box>
  )
}
