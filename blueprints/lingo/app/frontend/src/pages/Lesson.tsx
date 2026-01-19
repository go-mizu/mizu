import { useState, useEffect, useCallback, useRef } from 'react'
import { Container, Paper, Title, Text, Button, Group, Stack, Progress, ActionIcon, Loader, Badge, TextInput } from '@mantine/core'
import { IconX, IconHeart, IconCheck, IconVolume, IconVolume2, IconFlag, IconFlame } from '@tabler/icons-react'
import { useNavigate, useParams } from 'react-router-dom'
import { motion, AnimatePresence } from 'framer-motion'
import { useAuthStore } from '../stores/auth'
import { colors } from '../styles/tokens'
import { lessonsApi, Exercise, Lesson as LessonType } from '../api/client'
import { sounds, playSound, playTTS, stopSpeaking } from '../utils/sounds'

// Celebratory messages for correct answers - organized by streak level
const FEEDBACK_MESSAGES = {
  correct: {
    basic: ['Correct!', 'Nice!', 'Great!', 'Good!', 'Right!'],
    streak3: ['Amazing!', 'On fire!', 'Fantastic!', 'Excellent!', 'Superb!'],
    streak5: ['Incredible!', 'Unstoppable!', 'Brilliant!', 'Outstanding!', 'Magnificent!'],
    streak10: ['LEGENDARY!', 'UNBELIEVABLE!', 'PERFECT RUN!', 'GODLIKE!', 'FLAWLESS!'],
    perfect: ['Perfect lesson!', 'No mistakes!', 'Flawless!', 'Master!'],
  },
  incorrect: {
    first: ['Not quite', 'Almost!', 'Try again', 'Keep going!'],
    repeated: ['Keep trying!', "You'll get it!", 'Practice makes perfect', 'Don\'t give up!'],
  },
  encouragement: [
    'You\'re doing great!',
    'Keep it up!',
    'Almost there!',
    'Great progress!',
  ],
}

// Get appropriate message based on streak
function getCorrectMessage(streak: number): string {
  const messages = FEEDBACK_MESSAGES.correct
  let pool: string[]

  if (streak >= 10) {
    pool = messages.streak10
  } else if (streak >= 5) {
    pool = messages.streak5
  } else if (streak >= 3) {
    pool = messages.streak3
  } else {
    pool = messages.basic
  }

  return pool[Math.floor(Math.random() * pool.length)]
}

// Web Speech API type declarations for TypeScript
interface SpeechRecognitionEvent extends Event {
  results: SpeechRecognitionResultList
  resultIndex: number
}

interface SpeechRecognitionResultList {
  length: number
  item(index: number): SpeechRecognitionResult
  [index: number]: SpeechRecognitionResult
}

interface SpeechRecognitionResult {
  isFinal: boolean
  length: number
  item(index: number): SpeechRecognitionAlternative
  [index: number]: SpeechRecognitionAlternative
}

interface SpeechRecognitionAlternative {
  transcript: string
  confidence: number
}

interface SpeechRecognition extends EventTarget {
  continuous: boolean
  interimResults: boolean
  lang: string
  maxAlternatives: number
  onresult: ((event: SpeechRecognitionEvent) => void) | null
  onerror: ((event: Event) => void) | null
  onend: (() => void) | null
  start(): void
  stop(): void
  abort(): void
}

declare global {
  interface Window {
    SpeechRecognition: new () => SpeechRecognition
    webkitSpeechRecognition: new () => SpeechRecognition
  }
}

// Physics-based Confetti component using Canvas for celebrations
function Confetti({ show }: { show: boolean }) {
  const canvasRef = useRef<HTMLCanvasElement>(null)

  useEffect(() => {
    if (!show || !canvasRef.current) return

    const canvas = canvasRef.current
    const ctx = canvas.getContext('2d')
    if (!ctx) return

    // Set canvas size
    canvas.width = window.innerWidth
    canvas.height = window.innerHeight

    const COLORS = ['#58CC02', '#1CB0F6', '#FF9600', '#FFC800', '#CE82FF', '#FF4B4B', '#FFE066', '#87CEEB']
    const PARTICLE_COUNT = 80
    const GRAVITY = 0.25
    const DRAG = 0.98
    const ROTATION_SPEED = 0.15

    interface Particle {
      x: number
      y: number
      vx: number
      vy: number
      rotation: number
      rotationSpeed: number
      color: string
      size: number
      shape: 'circle' | 'square' | 'triangle' | 'ribbon'
      opacity: number
    }

    // Create particles with varied initial velocities for burst effect
    const particles: Particle[] = Array.from({ length: PARTICLE_COUNT }, () => {
      const angle = Math.random() * Math.PI * 2
      const velocity = Math.random() * 12 + 6
      return {
        x: canvas.width / 2 + (Math.random() - 0.5) * 100,
        y: canvas.height * 0.3,
        vx: Math.cos(angle) * velocity * (Math.random() + 0.5),
        vy: Math.sin(angle) * velocity * 0.5 - Math.random() * 8,
        rotation: Math.random() * Math.PI * 2,
        rotationSpeed: (Math.random() - 0.5) * ROTATION_SPEED,
        color: COLORS[Math.floor(Math.random() * COLORS.length)],
        size: Math.random() * 10 + 6,
        shape: (['circle', 'square', 'triangle', 'ribbon'] as const)[Math.floor(Math.random() * 4)],
        opacity: 1,
      }
    })

    let animationId: number
    let startTime = Date.now()

    function drawParticle(ctx: CanvasRenderingContext2D, p: Particle) {
      ctx.save()
      ctx.translate(p.x, p.y)
      ctx.rotate(p.rotation)
      ctx.globalAlpha = p.opacity
      ctx.fillStyle = p.color

      switch (p.shape) {
        case 'circle':
          ctx.beginPath()
          ctx.arc(0, 0, p.size / 2, 0, Math.PI * 2)
          ctx.fill()
          break
        case 'square':
          ctx.fillRect(-p.size / 2, -p.size / 2, p.size, p.size)
          break
        case 'triangle':
          ctx.beginPath()
          ctx.moveTo(0, -p.size / 2)
          ctx.lineTo(-p.size / 2, p.size / 2)
          ctx.lineTo(p.size / 2, p.size / 2)
          ctx.closePath()
          ctx.fill()
          break
        case 'ribbon':
          ctx.fillRect(-p.size / 6, -p.size, p.size / 3, p.size * 2)
          break
      }
      ctx.restore()
    }

    function animate() {
      if (!ctx || !canvas) return

      ctx.clearRect(0, 0, canvas.width, canvas.height)

      const elapsed = Date.now() - startTime
      let activeParticles = 0

      for (const p of particles) {
        // Physics
        p.vy += GRAVITY
        p.vx *= DRAG
        p.vy *= DRAG
        p.x += p.vx
        p.y += p.vy
        p.rotation += p.rotationSpeed

        // Fade out after 2 seconds
        if (elapsed > 2000) {
          p.opacity = Math.max(0, p.opacity - 0.03)
        }

        // Draw if still visible
        if (p.opacity > 0 && p.y < canvas.height + 50) {
          drawParticle(ctx, p)
          activeParticles++
        }
      }

      // Stop animation when all particles are done
      if (activeParticles > 0 && elapsed < 4000) {
        animationId = requestAnimationFrame(animate)
      }
    }

    animate()

    return () => {
      if (animationId) {
        cancelAnimationFrame(animationId)
      }
    }
  }, [show])

  if (!show) return null

  return (
    <canvas
      ref={canvasRef}
      style={{
        position: 'fixed',
        top: 0,
        left: 0,
        width: '100%',
        height: '100%',
        pointerEvents: 'none',
        zIndex: 9999,
      }}
    />
  )
}

// Mascot expressions and animations
type MascotExpression = 'neutral' | 'happy' | 'sad' | 'excited' | 'thinking'

const MASCOT_CONFIG = {
  neutral: { body: '#1CB0F6', accent: '#1899D6', eyeOffset: { x: 0, y: 0 }, mouthType: 'neutral' },
  happy: { body: '#58CC02', accent: '#58A700', eyeOffset: { x: 0, y: -1 }, mouthType: 'smile' },
  sad: { body: '#FF4B4B', accent: '#EA2B2B', eyeOffset: { x: 0, y: 2 }, mouthType: 'sad' },
  excited: { body: '#FFC800', accent: '#E5B400', eyeOffset: { x: 0, y: -2 }, mouthType: 'open' },
  thinking: { body: '#CE82FF', accent: '#A855F7', eyeOffset: { x: 2, y: -1 }, mouthType: 'neutral' },
} as const

// Mascot character component with idle animation and expressions
function Mascot({ message, variant = 'neutral' }: { message?: string; variant?: MascotExpression }) {
  const config = MASCOT_CONFIG[variant]
  const { body, accent, eyeOffset, mouthType } = config

  // Mouth paths for different expressions
  const mouthPaths = {
    neutral: "M36 54 Q40 56 44 54",
    smile: "M34 52 Q40 60 46 52",
    sad: "M34 56 Q40 52 46 56",
    open: "M36 52 Q40 58 44 52 Q40 62 36 52",
  }

  // Eye sizes for expressions
  const eyeSize = variant === 'excited' ? 7 : variant === 'sad' ? 5 : 6

  return (
    <Group gap="md" align="flex-end">
      {/* Animated owl mascot */}
      <motion.div
        // Idle breathing animation
        animate={{
          scale: [1, 1.02, 1],
          y: [0, -2, 0],
        }}
        transition={{
          duration: 3,
          repeat: Infinity,
          ease: "easeInOut",
        }}
      >
        <motion.svg
          width="80"
          height="80"
          viewBox="0 0 80 80"
          fill="none"
          // Celebration bounce for happy/excited
          animate={
            variant === 'happy' || variant === 'excited'
              ? { rotate: [0, -5, 5, -5, 0], y: [0, -5, 0] }
              : variant === 'sad'
              ? { rotate: [0, -2, 2, 0] }
              : {}
          }
          transition={{
            duration: 0.6,
            ease: "easeOut",
          }}
        >
          {/* Shadow */}
          <ellipse cx="40" cy="76" rx="20" ry="4" fill="rgba(0,0,0,0.1)" />

          {/* Body with gradient effect */}
          <defs>
            <radialGradient id={`bodyGrad-${variant}`} cx="40%" cy="30%" r="70%">
              <stop offset="0%" stopColor={body} stopOpacity="1" />
              <stop offset="100%" stopColor={accent} stopOpacity="1" />
            </radialGradient>
          </defs>
          <ellipse cx="40" cy="50" rx="28" ry="25" fill={`url(#bodyGrad-${variant})`} />

          {/* Belly highlight */}
          <ellipse cx="40" cy="55" rx="18" ry="15" fill="rgba(255,255,255,0.15)" />

          {/* Eyes with expression-based positioning */}
          <motion.g
            animate={{ x: eyeOffset.x, y: eyeOffset.y }}
            transition={{ duration: 0.2 }}
          >
            {/* Eye whites */}
            <circle cx="30" cy="42" r="12" fill="white" />
            <circle cx="50" cy="42" r="12" fill="white" />

            {/* Pupils - animated blink */}
            <motion.circle
              cx="32"
              cy="43"
              r={eyeSize}
              fill="#4B4B4B"
              animate={variant === 'thinking' ? { cx: [32, 34, 32] } : {}}
              transition={{ duration: 2, repeat: Infinity }}
            />
            <motion.circle
              cx="52"
              cy="43"
              r={eyeSize}
              fill="#4B4B4B"
              animate={variant === 'thinking' ? { cx: [52, 54, 52] } : {}}
              transition={{ duration: 2, repeat: Infinity }}
            />

            {/* Eye highlights */}
            <circle cx="34" cy="41" r="2.5" fill="white" />
            <circle cx="54" cy="41" r="2.5" fill="white" />

            {/* Eyebrows for expressions */}
            {variant === 'sad' && (
              <>
                <path d="M22 36 L34 38" stroke="#4B4B4B" strokeWidth="2" strokeLinecap="round" />
                <path d="M58 36 L46 38" stroke="#4B4B4B" strokeWidth="2" strokeLinecap="round" />
              </>
            )}
            {variant === 'excited' && (
              <>
                <path d="M22 38 L34 36" stroke="#4B4B4B" strokeWidth="2" strokeLinecap="round" />
                <path d="M58 38 L46 36" stroke="#4B4B4B" strokeWidth="2" strokeLinecap="round" />
              </>
            )}
          </motion.g>

          {/* Beak with expression-based mouth */}
          <path d="M36 50 L40 54 L44 50 Z" fill="#FFC800" stroke="#E5B400" strokeWidth="1" />
          <motion.path
            d={mouthPaths[mouthType as keyof typeof mouthPaths]}
            stroke="#E5B400"
            strokeWidth="2"
            fill={mouthType === 'open' ? '#FF9500' : 'none'}
            strokeLinecap="round"
          />

          {/* Ear tufts with subtle animation */}
          <motion.path
            d="M18 30 Q22 18 28 28"
            fill={accent}
            animate={{ d: ["M18 30 Q22 18 28 28", "M18 30 Q22 20 28 28", "M18 30 Q22 18 28 28"] }}
            transition={{ duration: 2, repeat: Infinity, ease: "easeInOut" }}
          />
          <motion.path
            d="M62 30 Q58 18 52 28"
            fill={accent}
            animate={{ d: ["M62 30 Q58 18 52 28", "M62 30 Q58 20 52 28", "M62 30 Q58 18 52 28"] }}
            transition={{ duration: 2, repeat: Infinity, ease: "easeInOut", delay: 0.5 }}
          />

          {/* Wings (subtle movement) */}
          <motion.ellipse
            cx="14"
            cy="52"
            rx="6"
            ry="12"
            fill={accent}
            animate={variant === 'happy' ? { rotate: [0, -10, 0] } : {}}
            transition={{ duration: 0.3 }}
          />
          <motion.ellipse
            cx="66"
            cy="52"
            rx="6"
            ry="12"
            fill={accent}
            animate={variant === 'happy' ? { rotate: [0, 10, 0] } : {}}
            transition={{ duration: 0.3 }}
          />

          {/* Feet */}
          <ellipse cx="32" cy="73" rx="8" ry="4" fill="#FFC800" />
          <ellipse cx="48" cy="73" rx="8" ry="4" fill="#FFC800" />

          {/* Sparkles for excited state */}
          {variant === 'excited' && (
            <>
              <motion.circle
                cx="10"
                cy="25"
                r="3"
                fill="#FFC800"
                animate={{ scale: [0, 1, 0], opacity: [0, 1, 0] }}
                transition={{ duration: 0.8, repeat: Infinity, delay: 0 }}
              />
              <motion.circle
                cx="70"
                cy="30"
                r="2"
                fill="#FFC800"
                animate={{ scale: [0, 1, 0], opacity: [0, 1, 0] }}
                transition={{ duration: 0.8, repeat: Infinity, delay: 0.3 }}
              />
              <motion.circle
                cx="65"
                cy="20"
                r="2.5"
                fill="#FFC800"
                animate={{ scale: [0, 1, 0], opacity: [0, 1, 0] }}
                transition={{ duration: 0.8, repeat: Infinity, delay: 0.6 }}
              />
            </>
          )}
        </motion.svg>
      </motion.div>

      {/* Speech bubble with animation */}
      {message && (
        <motion.div
          className="character-bubble"
          initial={{ opacity: 0, scale: 0.8, y: 10 }}
          animate={{ opacity: 1, scale: 1, y: 0 }}
          transition={{ duration: 0.3, type: 'spring', stiffness: 300 }}
        >
          <Text fw={600} size="lg" style={{ color: colors.text.primary }}>
            {message}
          </Text>
        </motion.div>
      )}
    </Group>
  )
}

// Audio playback hook using Web Speech API
function useAudio() {
  const [isPlaying, setIsPlaying] = useState(false)

  const playAudio = useCallback((url: string, slow = false) => {
    if (!url) return

    // Stop any currently playing audio
    stopSpeaking()
    setIsPlaying(true)

    // Use Web Speech API via playTTS (extracts text/lang from Google TTS URL)
    playTTS(
      url,
      undefined, // text will be extracted from URL
      undefined, // lang will be extracted from URL
      slow,
      () => setIsPlaying(false), // onEnd
      () => setIsPlaying(false)  // onError
    )
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
    <Stack gap="xl">
      {/* Answer area - where selected words appear (like a text line) */}
      <motion.div
        animate={
          isChecked && !isCorrect
            ? { x: [0, -10, 10, -10, 10, 0] }
            : {}
        }
        transition={{ duration: 0.4 }}
      >
        <div
          style={{
            minHeight: 56,
            borderBottom: `2px solid ${
              isChecked
                ? isCorrect
                  ? colors.semantic.success
                  : colors.semantic.error
                : colors.neutral.border
            }`,
            paddingBottom: 12,
            marginBottom: 8,
          }}
        >
          <Group gap="sm" wrap="wrap" style={{ minHeight: 40 }}>
            {selectedWords.length === 0 ? (
              <Text c={colors.text.muted} size="lg" style={{ fontStyle: 'italic' }}>
                _______________________
              </Text>
            ) : (
              selectedWords.map((word, index) => (
                <motion.div
                  key={`${word}-${index}`}
                  initial={{ scale: 0.8, opacity: 0, y: 10 }}
                  animate={{ scale: 1, opacity: 1, y: 0 }}
                  exit={{ scale: 0.8, opacity: 0 }}
                  transition={{ duration: 0.15, type: 'spring', stiffness: 500 }}
                  layout
                >
                  <Paper
                    p="sm"
                    px="md"
                    radius="lg"
                    onClick={() => !isChecked && onRemoveWord(index)}
                    style={{
                      backgroundColor: isChecked
                        ? isCorrect
                          ? colors.semantic.successLight
                          : colors.semantic.errorLight
                        : colors.neutral.white,
                      border: `2px solid ${
                        isChecked
                          ? isCorrect
                            ? colors.semantic.success
                            : colors.semantic.error
                          : colors.secondary.blue
                      }`,
                      cursor: isChecked ? 'default' : 'pointer',
                      boxShadow: isChecked ? 'none' : '0 2px 0 #1899D6',
                    }}
                  >
                    <Text
                      fw={600}
                      size="lg"
                      style={{
                        color: isChecked
                          ? isCorrect
                            ? colors.semantic.success
                            : colors.semantic.error
                          : colors.text.primary,
                      }}
                    >
                      {word}
                    </Text>
                  </Paper>
                </motion.div>
              ))
            )}
          </Group>
        </div>
      </motion.div>

      {/* Word choices - pill style like Duolingo */}
      <Group gap="sm" wrap="wrap" justify="center">
        {availableWords.map((word, index) => {
          const usedCount = selectedWords.filter(w => w === word).length
          const totalCount = availableWords.filter(w => w === word).length
          const isFullyUsed = usedCount >= totalCount

          return (
            <motion.div
              key={`${word}-${index}`}
              whileTap={{ scale: 0.95 }}
              animate={isFullyUsed ? { opacity: 0.4, y: 0 } : { opacity: 1, y: 0 }}
              layout
            >
              <Paper
                p="sm"
                px="lg"
                radius="xl"
                onClick={() => {
                  if (!isChecked && !isFullyUsed) {
                    playSound('select', 0.3)
                    onSelectWord(word)
                  }
                }}
                style={{
                  backgroundColor: isFullyUsed ? colors.neutral.background : colors.neutral.white,
                  border: `2px solid ${colors.neutral.border}`,
                  cursor: isChecked || isFullyUsed ? 'default' : 'pointer',
                  boxShadow: isFullyUsed ? 'none' : '0 3px 0 #E5E5E5',
                  transition: 'all 0.15s ease',
                }}
              >
                <Text
                  fw={600}
                  size="lg"
                  style={{
                    color: isFullyUsed ? colors.text.muted : colors.text.primary,
                    visibility: isFullyUsed ? 'hidden' : 'visible',
                  }}
                >
                  {word}
                </Text>
              </Paper>
            </motion.div>
          )
        })}
      </Group>

      {/* Show correct answer if wrong */}
      {isChecked && !isCorrect && (
        <motion.div
          initial={{ opacity: 0, y: 10 }}
          animate={{ opacity: 1, y: 0 }}
        >
          <Text size="sm" c={colors.semantic.error} ta="center">
            Correct answer: {exercise.correct_answer}
          </Text>
        </motion.div>
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
      {exercise.choices?.map((choice, index) => {
        const isSelected = selectedAnswer === choice
        const showCorrect = isChecked && choice === exercise.correct_answer
        const showIncorrect = isChecked && isSelected && !isCorrect
        const optionNumber = index + 1

        return (
          <motion.div
            key={choice}
            whileTap={{ scale: 0.98 }}
            animate={
              showIncorrect
                ? { x: [0, -10, 10, -10, 10, 0] }
                : showCorrect
                ? { scale: [1, 1.02, 1] }
                : {}
            }
            transition={{ duration: 0.4 }}
          >
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
              <Group gap="md">
                {/* Number badge like Duolingo */}
                <div
                  style={{
                    width: 32,
                    height: 32,
                    borderRadius: 8,
                    border: `2px solid ${
                      showCorrect
                        ? colors.semantic.success
                        : showIncorrect
                        ? colors.semantic.error
                        : isSelected
                        ? colors.secondary.blue
                        : colors.neutral.border
                    }`,
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    flexShrink: 0,
                    backgroundColor: showCorrect
                      ? colors.semantic.success
                      : showIncorrect
                      ? colors.semantic.error
                      : isSelected
                      ? colors.secondary.blue
                      : 'transparent',
                  }}
                >
                  <Text
                    size="sm"
                    fw={700}
                    style={{
                      color: (showCorrect || showIncorrect || isSelected)
                        ? 'white'
                        : colors.text.secondary,
                    }}
                  >
                    {optionNumber}
                  </Text>
                </div>
                <Text
                  size="lg"
                  fw={600}
                  style={{
                    flex: 1,
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
                {showCorrect && (
                  <motion.div
                    initial={{ scale: 0 }}
                    animate={{ scale: 1 }}
                    transition={{ type: 'spring', stiffness: 500, damping: 15 }}
                  >
                    <IconCheck size={24} style={{ color: colors.semantic.success }} />
                  </motion.div>
                )}
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

// Speaking Exercise Component - Uses Web Speech API for pronunciation practice
function SpeakingExercise({
  exercise,
  onResult,
  isChecked,
  isCorrect,
}: {
  exercise: Exercise
  onResult: (transcript: string) => void
  isChecked: boolean
  isCorrect: boolean
}) {
  const [isListening, setIsListening] = useState(false)
  const [transcript, setTranscript] = useState('')
  const [confidence, setConfidence] = useState(0)
  const recognitionRef = useRef<SpeechRecognition | null>(null)

  // Initialize speech recognition
  useEffect(() => {
    if (typeof window === 'undefined') return

    const SpeechRecognitionCtor = window.SpeechRecognition || window.webkitSpeechRecognition
    if (!SpeechRecognitionCtor) return

    const recognition = new SpeechRecognitionCtor()
    recognition.continuous = false
    recognition.interimResults = true
    recognition.maxAlternatives = 3

    // Try to set language from exercise (e.g., 'ja' for Japanese)
    const langMatch = exercise.audio_url?.match(/[?&]tl=([a-z]{2})/i)
    if (langMatch) {
      const langMap: Record<string, string> = {
        ja: 'ja-JP', zh: 'zh-CN', es: 'es-ES', fr: 'fr-FR', de: 'de-DE',
        ko: 'ko-KR', it: 'it-IT', pt: 'pt-BR', ru: 'ru-RU', ar: 'ar-SA',
      }
      recognition.lang = langMap[langMatch[1]] || 'en-US'
    }

    recognition.onresult = (event: SpeechRecognitionEvent) => {
      const result = event.results[event.results.length - 1]
      const text = result[0].transcript
      const conf = result[0].confidence

      setTranscript(text)
      setConfidence(conf)

      if (result.isFinal) {
        setIsListening(false)
        onResult(text)
      }
    }

    recognition.onerror = () => {
      setIsListening(false)
    }

    recognition.onend = () => {
      setIsListening(false)
    }

    recognitionRef.current = recognition

    return () => {
      recognition.abort()
    }
  }, [exercise.audio_url, onResult])

  const startListening = () => {
    if (!recognitionRef.current || isChecked) return
    setTranscript('')
    setConfidence(0)
    setIsListening(true)
    recognitionRef.current.start()
    sounds.buttonClick()
  }

  const stopListening = () => {
    if (!recognitionRef.current) return
    recognitionRef.current.stop()
    setIsListening(false)
  }

  return (
    <Stack gap="lg" align="center">
      {/* Target phrase display */}
      <Paper
        p="xl"
        radius="lg"
        style={{
          backgroundColor: colors.neutral.background,
          border: `2px solid ${colors.neutral.border}`,
          width: '100%',
        }}
      >
        <Text size="xl" fw={700} ta="center" style={{ color: colors.text.primary }}>
          {exercise.prompt}
        </Text>
        {exercise.hints && exercise.hints[0] && (
          <Text size="sm" c={colors.text.muted} ta="center" mt="xs">
            {exercise.hints[0]}
          </Text>
        )}
      </Paper>

      {/* Microphone button */}
      <motion.div
        whileTap={{ scale: 0.95 }}
        animate={isListening ? { scale: [1, 1.1, 1] } : {}}
        transition={isListening ? { duration: 1, repeat: Infinity } : {}}
      >
        <ActionIcon
          variant="filled"
          color={isListening ? 'red' : 'blue'}
          size={80}
          radius="50%"
          onClick={isListening ? stopListening : startListening}
          disabled={isChecked}
          style={{
            boxShadow: isListening ? '0 0 20px rgba(255, 75, 75, 0.5)' : '0 4px 0 #1899D6',
          }}
        >
          <svg width="40" height="40" viewBox="0 0 24 24" fill="white">
            {isListening ? (
              // Stop icon
              <rect x="6" y="6" width="12" height="12" rx="2" />
            ) : (
              // Microphone icon
              <>
                <path d="M12 1a3 3 0 0 0-3 3v8a3 3 0 0 0 6 0V4a3 3 0 0 0-3-3z" />
                <path d="M19 10v2a7 7 0 0 1-14 0v-2" fill="none" stroke="white" strokeWidth="2" />
                <line x1="12" y1="19" x2="12" y2="23" stroke="white" strokeWidth="2" />
                <line x1="8" y1="23" x2="16" y2="23" stroke="white" strokeWidth="2" />
              </>
            )}
          </svg>
        </ActionIcon>
      </motion.div>

      {/* Listening indicator */}
      {isListening && (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
        >
          <Text size="sm" c={colors.accent.pink} fw={600}>
            Listening... Speak now!
          </Text>
        </motion.div>
      )}

      {/* Transcript display */}
      {transcript && (
        <motion.div
          initial={{ opacity: 0, y: 10 }}
          animate={{ opacity: 1, y: 0 }}
          style={{ width: '100%' }}
        >
          <Paper
            p="lg"
            radius="lg"
            style={{
              backgroundColor: isChecked
                ? isCorrect
                  ? colors.semantic.successLight
                  : colors.semantic.errorLight
                : colors.neutral.white,
              border: `2px solid ${
                isChecked
                  ? isCorrect
                    ? colors.semantic.success
                    : colors.semantic.error
                  : colors.secondary.blue
              }`,
            }}
          >
            <Text size="lg" fw={600} ta="center" style={{ color: colors.text.primary }}>
              "{transcript}"
            </Text>
            {confidence > 0 && (
              <Text size="xs" c={colors.text.muted} ta="center" mt="xs">
                Confidence: {Math.round(confidence * 100)}%
              </Text>
            )}
          </Paper>
        </motion.div>
      )}

      {/* Correct answer on error */}
      {isChecked && !isCorrect && (
        <Text size="sm" c={colors.semantic.error} ta="center">
          Expected: {exercise.correct_answer}
        </Text>
      )}
    </Stack>
  )
}

// Listen and Select Exercise - Listen to audio and select the matching option
function ListenSelectExercise({
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
  const [hasPlayed, setHasPlayed] = useState(false)

  const playAudio = (slow = false) => {
    setHasPlayed(true)
    if (exercise.audio_url) {
      playTTS(exercise.audio_url, undefined, undefined, slow)
    }
  }

  // Auto-play on mount
  useEffect(() => {
    if (exercise.audio_url && !hasPlayed) {
      setTimeout(() => playAudio(), 500)
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  return (
    <Stack gap="lg">
      {/* Audio playback buttons */}
      <Group justify="center" gap="md">
        <motion.div whileTap={{ scale: 0.95 }}>
          <ActionIcon
            variant="filled"
            color="blue"
            size={64}
            radius="50%"
            onClick={() => playAudio(false)}
            style={{ boxShadow: '0 4px 0 #1899D6' }}
          >
            <IconVolume size={32} />
          </ActionIcon>
        </motion.div>
        <motion.div whileTap={{ scale: 0.95 }}>
          <ActionIcon
            variant="light"
            color="blue"
            size={48}
            radius="50%"
            onClick={() => playAudio(true)}
            title="Play slowly"
            style={{ border: `2px solid ${colors.secondary.blue}` }}
          >
            <IconVolume2 size={24} />
          </ActionIcon>
        </motion.div>
      </Group>

      <Text size="sm" c={colors.text.muted} ta="center">
        Listen and select the correct answer
      </Text>

      {/* Options - can be text or audio buttons */}
      <Stack gap="sm">
        {exercise.choices?.map((choice, index) => {
          const isSelected = selectedAnswer === choice
          const showCorrect = isChecked && choice === exercise.correct_answer
          const showIncorrect = isChecked && isSelected && !isCorrect

          return (
            <motion.div
              key={choice}
              whileTap={{ scale: 0.98 }}
              animate={showIncorrect ? { x: [0, -10, 10, -10, 10, 0] } : {}}
              transition={{ duration: 0.4 }}
            >
              <Paper
                p="lg"
                radius="lg"
                onClick={() => !isChecked && onSelect(choice)}
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
                  boxShadow: isChecked ? 'none' : isSelected ? '0 4px 0 #1899D6' : '0 4px 0 #E5E5E5',
                }}
              >
                <Group>
                  <div
                    style={{
                      width: 28,
                      height: 28,
                      borderRadius: 8,
                      border: `2px solid ${
                        showCorrect
                          ? colors.semantic.success
                          : showIncorrect
                          ? colors.semantic.error
                          : isSelected
                          ? colors.secondary.blue
                          : colors.neutral.border
                      }`,
                      backgroundColor: showCorrect || showIncorrect || isSelected
                        ? showCorrect
                          ? colors.semantic.success
                          : showIncorrect
                          ? colors.semantic.error
                          : colors.secondary.blue
                        : 'transparent',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      flexShrink: 0,
                    }}
                  >
                    <Text size="sm" fw={700} style={{ color: isSelected || showCorrect || showIncorrect ? 'white' : colors.text.secondary }}>
                      {index + 1}
                    </Text>
                  </div>
                  <Text fw={600} style={{ color: colors.text.primary, flex: 1 }}>
                    {choice}
                  </Text>
                  {showCorrect && <IconCheck size={24} style={{ color: colors.semantic.success }} />}
                </Group>
              </Paper>
            </motion.div>
          )
        })}
      </Stack>
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
  const [streak, setStreak] = useState(0)
  const [showConfetti, setShowConfetti] = useState(false)
  const [correctMessage, setCorrectMessage] = useState('')
  const streakRef = useRef(0)

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
    } else if (exerciseType === 'speaking') {
      // For speaking, use the transcript stored in typedAnswer
      // Compare loosely (ignore case, trim whitespace, allow partial matches)
      userAnswer = typedAnswer.trim()
    } else if (exerciseType === 'listen_select') {
      if (!selectedAnswer) return
      userAnswer = selectedAnswer
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
      // Award XP with streak bonus
      const baseXP = 3
      const streakBonus = streakRef.current >= 5 ? 2 : streakRef.current >= 3 ? 1 : 0
      setXpEarned((prev) => prev + baseXP + streakBonus)

      const newStreak = streakRef.current + 1
      streakRef.current = newStreak
      setStreak(newStreak)

      // Use streak-aware message selection
      setCorrectMessage(getCorrectMessage(newStreak))

      // Show confetti for streaks of 3 or more
      if (newStreak >= 3) {
        setShowConfetti(true)
        setTimeout(() => setShowConfetti(false), 4000)
        sounds.streakCelebration()
      } else {
        sounds.correctAnswer()
      }

      // Extra celebration for milestone streaks
      if (newStreak === 5 || newStreak === 10) {
        sounds.levelUp()
      }
    } else {
      streakRef.current = 0
      setStreak(0)
      setMistakes((prev) => prev + 1)
      setHearts((prev) => Math.max(0, prev - 1))
      sounds.wrongAnswer()
      sounds.heartLost()
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
    if (exerciseType === 'speaking') {
      return typedAnswer.trim().length > 0  // Speaking stores result in typedAnswer
    }
    if (exerciseType === 'listen_select') {
      return selectedAnswer !== null
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
      case 'listen_select':
        return 'Listen and select'
      case 'word_bank':
        return 'Build the sentence'
      case 'fill_blank':
        return 'Complete the sentence'
      case 'match_pairs':
        return 'Match the pairs'
      case 'speaking':
        return 'Speak this phrase'
      case 'tap_complete':
        return 'Tap to complete'
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
      {/* Confetti overlay */}
      <Confetti show={showConfetti} />

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
          {/* Streak counter - shows above progress bar when streak >= 2 */}
          {streak >= 2 && (
            <motion.div
              initial={{ opacity: 0, y: -20 }}
              animate={{ opacity: 1, y: 0 }}
              style={{ textAlign: 'center', marginBottom: 8 }}
            >
              <Text
                size="xs"
                fw={800}
                tt="uppercase"
                style={{
                  color: colors.accent.orange,
                  letterSpacing: '0.1em',
                }}
                className={streak >= 5 ? 'animate-celebrate' : ''}
              >
                <IconFlame
                  size={14}
                  style={{
                    color: colors.accent.orange,
                    verticalAlign: 'middle',
                    marginRight: 4,
                  }}
                />
                {streak} IN A ROW
              </Text>
            </motion.div>
          )}

          <Group justify="space-between">
            <ActionIcon variant="subtle" size="lg" onClick={handleQuit}>
              <IconX size={24} style={{ color: colors.text.secondary }} />
            </ActionIcon>

            <div style={{ flex: 1, margin: '0 20px' }}>
              <Progress
                value={progress}
                size="lg"
                radius="xl"
                color="green"
                styles={{
                  root: {
                    backgroundColor: colors.neutral.border,
                  },
                  section: {
                    transition: 'width 0.3s ease',
                  },
                }}
              />
            </div>

            <Group gap={4}>
              <motion.div
                animate={hearts < (user?.hearts || 5) ? { scale: [1, 1.3, 1] } : {}}
                transition={{ duration: 0.3 }}
              >
                <IconHeart size={24} style={{ color: colors.accent.pink, fill: colors.accent.pink }} />
              </motion.div>
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
              <Title order={3} ta="center" fw={800} style={{ color: colors.text.primary }}>
                {getExerciseTypeLabel(currentExercise.type)}
              </Title>

              {/* Mascot with prompt in speech bubble */}
              <Group gap="lg" align="flex-start" justify="center">
                <motion.div
                  initial={{ scale: 0.8, opacity: 0 }}
                  animate={{ scale: 1, opacity: 1 }}
                  transition={{ delay: 0.1, type: 'spring', stiffness: 300 }}
                >
                  <Mascot
                    variant={
                      isChecked
                        ? isCorrect
                          ? streak >= 5
                            ? 'excited'
                            : 'happy'
                          : 'sad'
                        : 'neutral'
                    }
                    message={
                      currentExercise.type === 'listening'
                        ? undefined
                        : currentExercise.type !== 'fill_blank'
                        ? currentExercise.prompt
                        : undefined
                    }
                  />
                </motion.div>

                {/* Audio buttons for listening exercises */}
                {currentExercise.audio_url && (
                  <Group gap="xs" style={{ marginTop: 20 }}>
                    <motion.div whileTap={{ scale: 0.95 }}>
                      <ActionIcon
                        variant="filled"
                        color="blue"
                        size="xl"
                        radius="xl"
                        onClick={() => playAudio(currentExercise.audio_url!)}
                        style={{
                          opacity: isPlaying ? 0.7 : 1,
                          boxShadow: '0 4px 0 #1899D6',
                        }}
                      >
                        <IconVolume size={20} />
                      </ActionIcon>
                    </motion.div>
                    <motion.div whileTap={{ scale: 0.95 }}>
                      <ActionIcon
                        variant="light"
                        color="blue"
                        size="lg"
                        radius="xl"
                        onClick={() => playAudio(currentExercise.audio_url!, true)}
                        title="Play slowly"
                        style={{
                          border: `2px solid ${colors.secondary.blue}`,
                        }}
                      >
                        <IconVolume2 size={16} />
                      </ActionIcon>
                    </motion.div>
                  </Group>
                )}
              </Group>

              {/* Listening prompt (separate since mascot doesn't have speech bubble for it) */}
              {currentExercise.type === 'listening' && (
                <Title order={2} ta="center" style={{ color: colors.text.muted }}>
                  Type what you hear
                </Title>
              )}

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
              ) : currentExercise.type === 'speaking' ? (
                <SpeakingExercise
                  exercise={currentExercise}
                  onResult={(transcript) => setTypedAnswer(transcript)}
                  isChecked={isChecked}
                  isCorrect={isCorrect}
                />
              ) : currentExercise.type === 'listen_select' ? (
                <ListenSelectExercise
                  exercise={currentExercise}
                  selectedAnswer={selectedAnswer}
                  onSelect={setSelectedAnswer}
                  isChecked={isChecked}
                  isCorrect={isCorrect}
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
      <AnimatePresence mode="wait">
        <motion.div
          key={isChecked ? 'checked' : 'unchecked'}
          initial={{ y: 20, opacity: 0 }}
          animate={{ y: 0, opacity: 1 }}
          exit={{ y: 20, opacity: 0 }}
          transition={{ duration: 0.2 }}
          style={{
            position: 'fixed',
            bottom: 0,
            left: 0,
            right: 0,
            zIndex: 100,
          }}
        >
          <Paper
            p="lg"
            radius={0}
            style={{
              backgroundColor: isChecked
                ? isCorrect
                  ? colors.semantic.successLight
                  : colors.semantic.errorLight
                : colors.neutral.white,
              borderTop: isChecked ? 'none' : `2px solid ${colors.neutral.border}`,
            }}
          >
            <Container size="sm">
              {isChecked ? (
                <Group justify="space-between" align="center">
                  <Group gap="md">
                    {/* Large checkmark or X icon */}
                    <motion.div
                      initial={{ scale: 0, rotate: -180 }}
                      animate={{ scale: 1, rotate: 0 }}
                      transition={{ type: 'spring', stiffness: 500, damping: 15 }}
                    >
                      <div
                        style={{
                          width: 48,
                          height: 48,
                          borderRadius: '50%',
                          backgroundColor: isCorrect ? colors.semantic.success : colors.semantic.error,
                          display: 'flex',
                          alignItems: 'center',
                          justifyContent: 'center',
                        }}
                      >
                        <IconCheck
                          size={28}
                          style={{
                            color: 'white',
                            transform: isCorrect ? 'none' : 'rotate(45deg)',
                          }}
                        />
                      </div>
                    </motion.div>

                    <div>
                      <Text
                        size="xl"
                        fw={800}
                        style={{ color: isCorrect ? colors.semantic.success : colors.semantic.error }}
                      >
                        {isCorrect ? correctMessage || 'Correct!' : 'Incorrect'}
                      </Text>
                      {isCorrect ? (
                        <Group gap="xs" style={{ marginTop: 4 }}>
                          <IconFlag size={14} style={{ color: colors.semantic.success }} />
                          <Text
                            size="xs"
                            fw={700}
                            tt="uppercase"
                            style={{ color: colors.semantic.success, cursor: 'pointer' }}
                          >
                            Report
                          </Text>
                        </Group>
                      ) : (
                        currentExercise.type !== 'match_pairs' && (
                          <Text size="sm" style={{ color: colors.semantic.error }}>
                            Correct answer: {currentExercise.correct_answer}
                          </Text>
                        )
                      )}
                    </div>
                  </Group>

                  <motion.div
                    initial={{ scale: 0.8 }}
                    animate={{ scale: 1 }}
                    transition={{ delay: 0.1 }}
                  >
                    <Button
                      size="lg"
                      color={isCorrect ? 'green' : 'red'}
                      onClick={handleContinue}
                      style={{
                        fontWeight: 700,
                        textTransform: 'uppercase',
                        boxShadow: isCorrect ? '0 4px 0 #58A700' : '0 4px 0 #EA2B2B',
                        minWidth: 140,
                      }}
                    >
                      Continue
                    </Button>
                  </motion.div>
                </Group>
              ) : (
                <Group justify="space-between">
                  <Button
                    size="lg"
                    variant="outline"
                    color="gray"
                    onClick={handleQuit}
                    style={{
                      fontWeight: 700,
                      textTransform: 'uppercase',
                      borderWidth: 2,
                      borderColor: colors.neutral.border,
                      color: colors.text.secondary,
                      boxShadow: '0 4px 0 #E5E5E5',
                    }}
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
                      minWidth: 140,
                    }}
                  >
                    Check
                  </Button>
                </Group>
              )}
            </Container>
          </Paper>
        </motion.div>
      </AnimatePresence>
    </div>
  )
}
