import { useState, useEffect } from 'react'
import { Container, Title, Text, Paper, Group, Stack, Badge, ActionIcon, Loader, Center, Tabs, Button } from '@mantine/core'
import { IconVolume, IconArrowLeft, IconBook, IconVocabulary } from '@tabler/icons-react'
import { useNavigate, useParams } from 'react-router-dom'
import { motion } from 'framer-motion'
import { colors } from '../styles/tokens'
import { coursesApi, Unit, Lexeme } from '../api/client'

interface VocabCardProps {
  lexeme: Lexeme
  index: number
}

function VocabCard({ lexeme, index }: VocabCardProps) {
  const playAudio = () => {
    // Play pronunciation audio
    if (lexeme.audio_url) {
      const audio = new Audio(lexeme.audio_url)
      audio.play().catch(console.error)
    } else {
      // Use browser TTS as fallback
      const utterance = new SpeechSynthesisUtterance(lexeme.word)
      utterance.lang = 'zh-CN'
      speechSynthesis.speak(utterance)
    }
  }

  return (
    <motion.div
      initial={{ opacity: 0, y: 10 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ delay: index * 0.03, duration: 0.2 }}
    >
      <Paper
        p="lg"
        radius="lg"
        style={{
          backgroundColor: '#FFFFFF',
          border: `2px solid ${colors.neutral.border}`,
          cursor: 'pointer',
          transition: 'all 0.15s ease',
        }}
        onMouseEnter={(e) => {
          e.currentTarget.style.borderColor = colors.primary.green
          e.currentTarget.style.transform = 'translateY(-2px)'
        }}
        onMouseLeave={(e) => {
          e.currentTarget.style.borderColor = colors.neutral.border
          e.currentTarget.style.transform = 'translateY(0)'
        }}
        onClick={playAudio}
      >
        <Group justify="space-between" wrap="nowrap">
          <div style={{ flex: 1 }}>
            {/* Chinese character */}
            <Text
              size="xl"
              fw={700}
              style={{ color: colors.text.primary, fontSize: '1.5rem' }}
            >
              {lexeme.word}
            </Text>

            {/* Romanization/Pinyin - extracted from hints or generated */}
            <Text size="sm" style={{ color: colors.secondary.blue, fontStyle: 'italic' }} mt={4}>
              {extractPinyin(lexeme)}
            </Text>

            {/* Translation */}
            <Text size="sm" style={{ color: colors.text.secondary }} mt={4}>
              {lexeme.translation}
            </Text>

            {/* Part of speech badge */}
            {lexeme.pos && (
              <Badge size="xs" color="gray" variant="light" mt={8}>
                {lexeme.pos}
              </Badge>
            )}
          </div>

          {/* Audio button */}
          <ActionIcon
            variant="light"
            color="blue"
            size="lg"
            radius="xl"
            onClick={(e) => {
              e.stopPropagation()
              playAudio()
            }}
          >
            <IconVolume size={20} />
          </ActionIcon>
        </Group>
      </Paper>
    </motion.div>
  )
}

// Helper to extract pinyin from word (for Chinese characters, we might have it in example or need to show the word itself)
function extractPinyin(lexeme: Lexeme): string {
  // If we have example sentence with pinyin, try to extract it
  // For now, we'll just show a placeholder or the word itself for non-Chinese
  // In a real app, we'd have pinyin stored separately
  return lexeme.example_sentence || lexeme.word
}

export default function Guidebook() {
  const navigate = useNavigate()
  const { unitId } = useParams<{ unitId: string }>()
  const [unit, setUnit] = useState<Unit | null>(null)
  const [vocabulary, setVocabulary] = useState<Lexeme[]>([])
  const [loading, setLoading] = useState(true)
  const [activeTab, setActiveTab] = useState<string | null>('phrases')

  useEffect(() => {
    async function loadGuidebookData() {
      if (!unitId) return

      try {
        setLoading(true)

        // Get unit details
        const unitData = await coursesApi.getUnit(unitId)
        setUnit(unitData)

        // Get vocabulary for the course
        if (unitData.course_id) {
          const vocab = await coursesApi.getVocabulary(unitData.course_id)
          setVocabulary(vocab)
        }
      } catch (err) {
        console.error('Failed to load guidebook data:', err)
      } finally {
        setLoading(false)
      }
    }

    loadGuidebookData()
  }, [unitId])

  if (loading) {
    return (
      <Center h="60vh">
        <Stack align="center" gap="md">
          <Loader size="lg" color="green" />
          <Text c={colors.text.secondary}>Loading guidebook...</Text>
        </Stack>
      </Center>
    )
  }

  if (!unit) {
    return (
      <Center h="60vh">
        <Stack align="center" gap="md">
          <Text c={colors.semantic.error} fw={600}>Unit not found</Text>
          <Button onClick={() => navigate('/learn')}>Back to Learn</Button>
        </Stack>
      </Center>
    )
  }

  return (
    <div style={{ minHeight: '100vh', backgroundColor: '#FFFFFF' }}>
      {/* Header */}
      <Paper
        p="md"
        style={{
          backgroundColor: colors.primary.green,
          borderRadius: 0,
        }}
      >
        <Container size="md">
          <Group justify="space-between">
            <Group gap="md">
              <ActionIcon
                variant="transparent"
                onClick={() => navigate('/learn')}
                style={{ color: 'white' }}
              >
                <IconArrowLeft size={24} />
              </ActionIcon>
              <div>
                <Text size="xs" fw={600} style={{ color: 'rgba(255,255,255,0.8)', textTransform: 'uppercase' }}>
                  Unit {unit.position}
                </Text>
                <Text fw={700} style={{ color: 'white', fontSize: '1.25rem' }}>
                  Guidebook
                </Text>
              </div>
            </Group>
            <IconBook size={32} style={{ color: 'white' }} />
          </Group>
        </Container>
      </Paper>

      <Container size="md" py="xl">
        {/* Unit title */}
        <Group gap="md" mb="xl">
          <div style={{
            width: 60,
            height: 60,
            borderRadius: '50%',
            backgroundColor: colors.primary.greenLight,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
          }}>
            <IconBook size={32} style={{ color: colors.primary.green }} />
          </div>
          <div>
            <Title order={2} style={{ color: colors.text.primary }}>
              {unit.title}
            </Title>
            <Text style={{ color: colors.text.secondary }}>
              {unit.description}
            </Text>
          </div>
        </Group>

        {/* Tabs */}
        <Tabs value={activeTab} onChange={setActiveTab} mb="xl">
          <Tabs.List>
            <Tabs.Tab value="phrases" leftSection={<IconVocabulary size={16} />}>
              Key Phrases
            </Tabs.Tab>
            <Tabs.Tab value="tips" leftSection={<IconBook size={16} />}>
              Tips
            </Tabs.Tab>
          </Tabs.List>
        </Tabs>

        {/* Content based on tab */}
        {activeTab === 'phrases' && (
          <>
            {/* Key Phrases Section */}
            <Paper
              p="lg"
              radius="lg"
              mb="lg"
              style={{
                backgroundColor: colors.secondary.blueLight,
                border: `2px solid ${colors.secondary.blue}`,
              }}
            >
              <Group gap="md">
                <IconVocabulary size={24} style={{ color: colors.secondary.blue }} />
                <div>
                  <Text fw={700} style={{ color: colors.secondary.blue }}>Key Phrases</Text>
                  <Text size="sm" style={{ color: colors.text.secondary }}>
                    Tap any card to hear the pronunciation
                  </Text>
                </div>
              </Group>
            </Paper>

            {/* Vocabulary Grid */}
            {vocabulary.length > 0 ? (
              <Stack gap="md">
                {vocabulary.map((lexeme, index) => (
                  <VocabCard key={lexeme.id} lexeme={lexeme} index={index} />
                ))}
              </Stack>
            ) : (
              <Paper p="xl" radius="lg" style={{ backgroundColor: '#F7F7F7', textAlign: 'center' }}>
                <Text style={{ color: colors.text.muted }}>
                  No vocabulary available for this unit yet.
                </Text>
              </Paper>
            )}
          </>
        )}

        {activeTab === 'tips' && (
          <>
            {/* Tips Section */}
            {unit.guidebook_content ? (
              <Paper p="xl" radius="lg" style={{ backgroundColor: '#F7F7F7' }}>
                <Text style={{ color: colors.text.primary, whiteSpace: 'pre-wrap' }}>
                  {unit.guidebook_content}
                </Text>
              </Paper>
            ) : (
              <Paper p="xl" radius="lg" style={{ backgroundColor: '#F7F7F7', textAlign: 'center' }}>
                <Text style={{ color: colors.text.muted }}>
                  No tips available for this unit yet.
                </Text>
              </Paper>
            )}

            {/* Skills in this unit */}
            {unit.skills && unit.skills.length > 0 && (
              <>
                <Title order={4} mt="xl" mb="md" style={{ color: colors.text.primary }}>
                  Skills in this Unit
                </Title>
                <Stack gap="sm">
                  {unit.skills.map((skill) => (
                    <Paper
                      key={skill.id}
                      p="md"
                      radius="lg"
                      style={{
                        backgroundColor: '#FFFFFF',
                        border: `2px solid ${colors.neutral.border}`,
                      }}
                    >
                      <Group justify="space-between">
                        <div>
                          <Text fw={600} style={{ color: colors.text.primary }}>
                            {skill.name}
                          </Text>
                          <Text size="sm" style={{ color: colors.text.secondary }}>
                            {skill.lexemes_count} words • {skill.levels} levels
                          </Text>
                        </div>
                        <Badge color="green" variant="light">
                          {skill.levels} crowns
                        </Badge>
                      </Group>
                    </Paper>
                  ))}
                </Stack>
              </>
            )}
          </>
        )}

        {/* Back to learning button */}
        <Button
          fullWidth
          size="lg"
          color="green"
          radius="xl"
          mt="xl"
          onClick={() => navigate('/learn')}
          style={{
            fontWeight: 700,
            textTransform: 'uppercase',
            boxShadow: '0 4px 0 #58A700',
          }}
        >
          Continue Learning
        </Button>
      </Container>
    </div>
  )
}
