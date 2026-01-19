import { useState, useEffect, useRef } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Stack, Text, Group, Card, Center, Loader, Box, Button, Progress, ActionIcon, Tooltip } from '@mantine/core'
import { IconChevronLeft, IconVolume, IconCheck, IconTrophy, IconStar } from '@tabler/icons-react'
import { motion, AnimatePresence } from 'framer-motion'
import { storiesApi, Story, StoryElement, StoryCharacter, WordToken } from '../api/client'
import { sounds, playTTS, stopSpeaking } from '../utils/sounds'
import { colors } from '../styles/tokens'

// Audio button component - Duolingo style speaker icon
function AudioButton({ onClick, isPlaying = false, size = 24 }: { onClick: () => void; isPlaying?: boolean; size?: number }) {
  return (
    <ActionIcon
      variant="transparent"
      color="blue"
      size={size + 8}
      onClick={onClick}
      style={{ flexShrink: 0 }}
    >
      <IconVolume
        size={size}
        style={{
          color: '#1CB0F6',
          opacity: isPlaying ? 0.6 : 1,
        }}
      />
    </ActionIcon>
  )
}

// Tappable word component with translation tooltip
function TappableWord({ token }: { token: WordToken }) {
  const [showHint, setShowHint] = useState(false)

  if (!token.is_tappable || !token.translation) {
    return <span>{token.word} </span>
  }

  return (
    <Tooltip
      label={token.translation}
      opened={showHint}
      position="top"
      withArrow
      color="dark"
    >
      <span
        onClick={() => setShowHint(!showHint)}
        style={{
          textDecoration: 'underline',
          textDecorationStyle: 'dotted',
          textDecorationColor: '#AFAFAF',
          cursor: 'pointer',
        }}
      >
        {token.word}{' '}
      </span>
    </Tooltip>
  )
}

// Story line with audio button on left (Duolingo style - no cards)
function StoryLine({
  element,
}: {
  element: StoryElement
}) {
  const [isPlaying, setIsPlaying] = useState(false)

  const playAudio = () => {
    if (element.text) {
      stopSpeaking()
      setIsPlaying(true)

      playTTS(
        element.audio_url,
        element.text,
        undefined,
        false,
        () => setIsPlaying(false),
        () => setIsPlaying(false)
      )
    }
  }

  return (
    <motion.div
      initial={{ opacity: 0, y: 10 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.2 }}
    >
      <Group align="flex-start" gap="sm" wrap="nowrap" py={8}>
        <AudioButton onClick={playAudio} isPlaying={isPlaying} />
        <Box style={{ flex: 1 }}>
          <Text
            size="lg"
            style={{
              color: colors.text.primary,
              lineHeight: 1.6,
            }}
          >
            {element.tokens && element.tokens.length > 0 ? (
              element.tokens.map((token, i) => (
                <TappableWord key={i} token={token} />
              ))
            ) : (
              element.text
            )}
          </Text>
          {element.translation && (
            <Text size="sm" c="dimmed" mt={4}>
              {element.translation}
            </Text>
          )}
        </Box>
      </Group>
    </motion.div>
  )
}

// Narration line (no speaker, same style as line)
function NarrationLine({ element }: { element: StoryElement }) {
  const [isPlaying, setIsPlaying] = useState(false)

  const playAudio = () => {
    if (element.text) {
      stopSpeaking()
      setIsPlaying(true)

      playTTS(
        element.audio_url,
        element.text,
        undefined,
        false,
        () => setIsPlaying(false),
        () => setIsPlaying(false)
      )
    }
  }

  return (
    <motion.div
      initial={{ opacity: 0, y: 10 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.2 }}
    >
      <Group align="flex-start" gap="sm" wrap="nowrap" py={8}>
        <AudioButton onClick={playAudio} isPlaying={isPlaying} />
        <Box style={{ flex: 1 }}>
          <Text
            size="lg"
            style={{
              color: colors.text.primary,
              lineHeight: 1.6,
            }}
          >
            {element.tokens && element.tokens.length > 0 ? (
              element.tokens.map((token, i) => (
                <TappableWord key={i} token={token} />
              ))
            ) : (
              element.text
            )}
          </Text>
        </Box>
      </Group>
    </motion.div>
  )
}

// Story header with title and audio
function StoryHeader({
  story,
  character,
}: {
  story: Story
  character?: StoryCharacter | null
}) {
  const [isPlaying, setIsPlaying] = useState(false)

  const playAudio = () => {
    if (story.title) {
      stopSpeaking()
      setIsPlaying(true)

      playTTS(
        undefined,
        story.title,
        undefined,
        false,
        () => setIsPlaying(false),
        () => setIsPlaying(false)
      )
    }
  }

  return (
    <Stack align="center" gap="md" mb="xl">
      {/* Character illustration */}
      {character?.avatar_url ? (
        <Box
          style={{
            width: 180,
            height: 180,
            borderRadius: '50%',
            overflow: 'hidden',
          }}
        >
          <img
            src={character.avatar_url}
            alt={character.display_name || character.name}
            style={{ width: '100%', height: '100%', objectFit: 'cover' }}
          />
        </Box>
      ) : story.illustration_url ? (
        <Box
          style={{
            width: 180,
            height: 180,
            borderRadius: 16,
            overflow: 'hidden',
          }}
        >
          <img
            src={story.illustration_url}
            alt={story.title}
            style={{ width: '100%', height: '100%', objectFit: 'cover' }}
          />
        </Box>
      ) : null}

      {/* Title with audio */}
      <Group gap="sm" justify="center">
        <AudioButton onClick={playAudio} isPlaying={isPlaying} size={28} />
        <Text
          fw={700}
          size="xl"
          style={{
            color: colors.text.primary,
            fontFamily: '"Nunito", "DIN Round Pro", sans-serif',
          }}
        >
          {story.title}
          {story.title_translation && (
            <Text component="span" c="dimmed" fw={400}>
              {' '}{story.title_translation}
            </Text>
          )}
        </Text>
      </Group>

      {/* Divider */}
      <Box w="100%" h={1} bg="#E5E5E5" />
    </Stack>
  )
}

// Inline multiple choice challenge (gray box style)
function InlineMultipleChoice({
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

  const correctIndex = challenge.correct_index ?? 0

  const handleSelect = (index: number) => {
    if (showResult) return
    setSelected(index)
    sounds.buttonClick()

    // Show result immediately
    setShowResult(true)
    const isCorrect = index === correctIndex

    if (isCorrect) {
      sounds.correctAnswer()
    } else {
      sounds.wrongAnswer()
    }

    setTimeout(() => {
      onAnswer(isCorrect)
    }, 1200)
  }

  return (
    <motion.div
      initial={{ opacity: 0, y: 10 }}
      animate={{ opacity: 1, y: 0 }}
    >
      <Box
        py="md"
        px="lg"
        my="md"
        style={{
          backgroundColor: '#F7F7F7',
          borderRadius: 8,
        }}
      >
        <Text size="md" c="dimmed" mb="sm">
          {challenge.question || challenge.prompt || element.text}
        </Text>
        <Stack gap={4}>
          {challenge.options.map((option, index) => {
            const isSelected = selected === index
            const isCorrect = index === correctIndex
            const showCorrect = showResult && isCorrect
            const showWrong = showResult && isSelected && !isCorrect

            return (
              <Group
                key={index}
                gap="xs"
                onClick={() => handleSelect(index)}
                style={{
                  cursor: showResult ? 'default' : 'pointer',
                  padding: '4px 0',
                }}
              >
                <Text c="dimmed" size="md">•</Text>
                <Text
                  size="md"
                  style={{
                    color: showCorrect ? '#58CC02' : showWrong ? '#FF4B4B' : '#1CB0F6',
                    textDecoration: 'underline',
                    textDecorationStyle: 'dotted',
                    fontWeight: isSelected ? 600 : 400,
                  }}
                >
                  {option}
                </Text>
                {showCorrect && <IconCheck size={16} style={{ color: '#58CC02' }} />}
              </Group>
            )
          })}
        </Stack>
      </Box>
    </motion.div>
  )
}

// Word selection challenge (tap the word that means X)
function SelectWordChallenge({
  element,
  onAnswer,
}: {
  element: StoryElement
  onAnswer: (correct: boolean) => void
}) {
  const [selected, setSelected] = useState<number | null>(null)
  const [showResult, setShowResult] = useState(false)
  const challenge = element.challenge_data

  if (!challenge?.sentence_words && !element.tokens) return null

  const words = challenge?.sentence_words || element.tokens || []
  const targetIndex = challenge?.target_word_index ?? words.findIndex(w => w.is_target)

  const handleSelect = (index: number) => {
    if (showResult) return

    const word = words[index]
    if (!word) return

    setSelected(index)
    sounds.buttonClick()

    setShowResult(true)
    const isCorrect = word.is_target || index === targetIndex

    if (isCorrect) {
      sounds.correctAnswer()
    } else {
      sounds.wrongAnswer()
    }

    setTimeout(() => {
      onAnswer(isCorrect)
    }, 1200)
  }

  return (
    <motion.div
      initial={{ opacity: 0, y: 10 }}
      animate={{ opacity: 1, y: 0 }}
    >
      <Box
        py="md"
        px="lg"
        my="md"
        style={{
          backgroundColor: '#F7F7F7',
          borderRadius: 8,
        }}
      >
        <Text size="md" c="dimmed" mb="md">
          {challenge?.question || challenge?.prompt || `Choose the option that means "${challenge?.target_meaning || 'this'}".`}
        </Text>
        <Group gap={8}>
          {words.map((word, index) => {
            const isSelected = selected === index
            const isCorrect = word.is_target || index === targetIndex
            const showCorrect = showResult && isCorrect
            const showWrong = showResult && isSelected && !isCorrect

            return (
              <Button
                key={index}
                variant="outline"
                color={showCorrect ? 'green' : showWrong ? 'red' : isSelected ? 'blue' : 'gray'}
                radius="md"
                size="sm"
                onClick={() => handleSelect(index)}
                style={{
                  borderWidth: 2,
                  backgroundColor: showCorrect ? '#E8F5E9' : showWrong ? '#FFEBEE' : isSelected ? '#E3F2FD' : 'white',
                }}
              >
                {word.word}
              </Button>
            )
          })}
        </Group>
      </Box>
    </motion.div>
  )
}

// "What comes next?" prompt
function WhatNextPrompt() {
  return (
    <motion.div
      initial={{ opacity: 0, y: 10 }}
      animate={{ opacity: 1, y: 0 }}
    >
      <Box
        py="md"
        px="lg"
        my="md"
        style={{
          backgroundColor: '#F7F7F7',
          borderRadius: 8,
        }}
      >
        <Text size="md" c="dimmed">
          What comes next?
        </Text>
      </Box>
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
  const initialWords = challenge?.arrange_words || []
  const [arranged, setArranged] = useState<string[]>([])
  const [available, setAvailable] = useState<string[]>([...initialWords])
  const [showResult, setShowResult] = useState(false)

  if (!initialWords.length) return null

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
    const userAnswer = arranged.join(' ')
    const correctAnswer = element.text || challenge?.correct_answer || ''
    const isCorrect = userAnswer.toLowerCase().trim() === correctAnswer.toLowerCase().trim()

    if (isCorrect) {
      sounds.correctAnswer()
    } else {
      sounds.wrongAnswer()
    }

    setTimeout(() => {
      onAnswer(isCorrect)
    }, 1200)
  }

  return (
    <motion.div
      initial={{ opacity: 0, y: 10 }}
      animate={{ opacity: 1, y: 0 }}
    >
      <Box
        py="md"
        px="lg"
        my="md"
        style={{
          backgroundColor: '#F7F7F7',
          borderRadius: 8,
        }}
      >
        <Text size="md" c="dimmed" mb="md">
          {challenge?.question || challenge?.prompt || 'Arrange the words'}
        </Text>

        {/* Arranged words area */}
        <Box
          mb="md"
          p="md"
          style={{
            minHeight: 50,
            borderRadius: 8,
            border: '2px dashed #E5E5E5',
            backgroundColor: 'white',
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
                    size="sm"
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
        <Group gap="xs">
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
                  size="sm"
                  onClick={() => handleWordClick(word, false)}
                >
                  {word}
                </Button>
              </motion.div>
            ))}
          </AnimatePresence>
        </Group>

        {!showResult && available.length === 0 && arranged.length > 0 && (
          <Button
            mt="md"
            size="md"
            color="green"
            radius="xl"
            onClick={handleCheck}
            fullWidth
          >
            CHECK
          </Button>
        )}
      </Box>
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

  const getMainCharacter = (): StoryCharacter | null => {
    if (!story?.characters || story.characters.length === 0) return null
    return story.characters[0]
  }

  const renderElement = (element: StoryElement, index: number) => {
    const isLatest = index === visibleElements.length - 1

    switch (element.element_type) {
      case 'header':
        return null // Header is handled separately

      case 'line':
        return (
          <StoryLine key={element.id} element={element} />
        )

      case 'narration':
        return (
          <NarrationLine
            key={element.id}
            element={element}
          />
        )

      case 'multiple_choice':
        if (isLatest) {
          return (
            <InlineMultipleChoice
              key={element.id}
              element={element}
              onAnswer={handleChallengeAnswer}
            />
          )
        }
        return null

      case 'select_word':
        if (isLatest) {
          return (
            <SelectWordChallenge
              key={element.id}
              element={element}
              onAnswer={handleChallengeAnswer}
            />
          )
        }
        return null

      case 'select_phrase':
        if (isLatest) {
          return (
            <InlineMultipleChoice
              key={element.id}
              element={element}
              onAnswer={handleChallengeAnswer}
            />
          )
        }
        return null

      case 'what_next':
        return (
          <WhatNextPrompt key={element.id} />
        )

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
          <StoryLine key={element.id} element={element} />
        )
    }
  }

  // Check if current element is a challenge
  const currentElement = story?.elements?.[currentIndex]
  const isChallenge = currentElement && [
    'multiple_choice',
    'select_phrase',
    'select_word',
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
        backgroundColor: '#FFFFFF',
      }}
    >
      {/* Header with language info */}
      <Box
        px="md"
        py="sm"
        style={{
          borderBottom: '1px solid #E5E5E5',
        }}
      >
        <Group justify="space-between" align="center">
          <Group gap="sm">
            <ActionIcon
              variant="subtle"
              color="gray"
              size="lg"
              onClick={() => navigate('/stories')}
            >
              <IconChevronLeft size={24} />
            </ActionIcon>
            <Text fw={600} style={{ color: '#1CB0F6' }}>
              English Stories
            </Text>
            <Text c="dimmed">from English</Text>
          </Group>
        </Group>
      </Box>

      {/* Progress bar */}
      <Progress
        value={progress}
        color="green"
        size="xs"
        radius={0}
      />

      {/* Story content */}
      <Box
        ref={scrollRef}
        style={{
          flex: 1,
          overflowY: 'auto',
          padding: '24px 16px',
        }}
      >
        <Box maw={700} mx="auto">
          {/* Story header with illustration and title */}
          <StoryHeader story={story} character={getMainCharacter()} />

          {/* Story elements */}
          <Stack gap={0}>
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
          <Box maw={700} mx="auto">
            <Button
              size="lg"
              color="green"
              radius="xl"
              fullWidth
              onClick={handleContinue}
            >
              CONTINUE
            </Button>
          </Box>
        </Box>
      )}
    </Box>
  )
}
