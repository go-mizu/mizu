import { useState, useEffect } from 'react'
import { SimpleGrid, Card, Text, Center, Loader, Stack, Group, Menu, Button, Box } from '@mantine/core'
import { IconChevronDown, IconCheck } from '@tabler/icons-react'
import { useNavigate } from 'react-router-dom'
import { coursesApi, Course, Language } from '../api/client'
import { useAuthStore } from '../stores/auth'
import { colors } from '../styles/tokens'

// Learner counts by language (simulated data like Duolingo)
const LEARNER_COUNTS: Record<string, number> = {
  'es': 48800000,  // Spanish
  'fr': 27200000,  // French
  'ja': 24400000,  // Japanese
  'de': 18900000,  // German
  'ko': 17800000,  // Korean
  'it': 13400000,  // Italian
  'zh': 11800000,  // Chinese
  'hi': 11700000,  // Hindi
  'ru': 9810000,   // Russian
  'ar': 8460000,   // Arabic
  'pt': 6120000,   // Portuguese
  'en': 21900000,  // English
  'tr': 5200000,   // Turkish
  'nl': 4100000,   // Dutch
  'el': 3200000,   // Greek
  'vi': 2800000,   // Vietnamese
}

// Format learner count
function formatLearners(count: number): string {
  if (count >= 1000000) {
    return `${(count / 1000000).toFixed(1)}M learners`
  }
  if (count >= 1000) {
    return `${(count / 1000).toFixed(0)}K learners`
  }
  return `${count} learners`
}

// Course card component
function CourseCard({
  course,
  language,
  isActive,
  onClick,
}: {
  course: Course
  language?: Language
  isActive: boolean
  onClick: () => void
}) {
  const flagEmoji = language?.flag_emoji || '🌐'
  const learners = LEARNER_COUNTS[course.learning_language_id] || 1000000

  return (
    <Card
      shadow="sm"
      radius="lg"
      p="lg"
      withBorder
      onClick={onClick}
      style={{
        cursor: 'pointer',
        border: isActive ? `3px solid ${colors.primary.green}` : '2px solid #E5E5E5',
        backgroundColor: '#FFFFFF',
        transition: 'all 0.2s ease',
        position: 'relative',
        minHeight: 160,
      }}
      styles={{
        root: {
          '&:hover': {
            borderColor: colors.primary.green,
            transform: 'translateY(-2px)',
          },
        },
      }}
    >
      {/* Active checkmark badge */}
      {isActive && (
        <Box
          style={{
            position: 'absolute',
            top: 8,
            right: 8,
            backgroundColor: colors.primary.green,
            borderRadius: '50%',
            width: 24,
            height: 24,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
          }}
        >
          <IconCheck size={14} color="white" strokeWidth={3} />
        </Box>
      )}

      <Stack align="center" gap="sm" justify="center" h="100%">
        {/* Flag emoji */}
        <Text
          style={{
            fontSize: 48,
            lineHeight: 1,
          }}
        >
          {flagEmoji}
        </Text>

        {/* Language name */}
        <Text fw={700} size="md" ta="center" style={{ color: colors.text.primary }}>
          {language?.name || course.title}
        </Text>

        {/* Learner count */}
        <Text size="xs" c="dimmed" ta="center">
          {formatLearners(learners)}
        </Text>
      </Stack>
    </Card>
  )
}

export default function Courses() {
  const navigate = useNavigate()
  const { user, setActiveCourse } = useAuthStore()
  const [courses, setCourses] = useState<Course[]>([])
  const [languages, setLanguages] = useState<Language[]>([])
  const [loading, setLoading] = useState(true)
  const [nativeLanguage, setNativeLanguage] = useState('en')
  const [languageMap, setLanguageMap] = useState<Record<string, Language>>({})

  useEffect(() => {
    loadData()
  }, [nativeLanguage])

  const loadData = async () => {
    try {
      setLoading(true)

      // Load languages first
      const langs = await coursesApi.listLanguages()
      setLanguages(langs)

      // Create language map for easy lookup
      const langMap: Record<string, Language> = {}
      langs.forEach((lang) => {
        langMap[lang.id] = lang
      })
      setLanguageMap(langMap)

      // Load courses for native language
      const courseList = await coursesApi.listCourses(nativeLanguage)
      setCourses(courseList)
    } catch (err) {
      console.error('Failed to load courses:', err)
    } finally {
      setLoading(false)
    }
  }

  const handleCourseSelect = async (course: Course) => {
    try {
      await setActiveCourse(course.id)
      navigate('/learn')
    } catch (err) {
      console.error('Failed to set active course:', err)
    }
  }

  const nativeLang = languageMap[nativeLanguage]

  if (loading) {
    return (
      <Center h={400}>
        <Loader color="green" size="lg" />
      </Center>
    )
  }

  return (
    <Box maw={1000} mx="auto" py="xl">
      {/* Header */}
      <Group justify="space-between" align="center" mb="xl">
        <Text
          fw={800}
          size="xl"
          style={{
            color: colors.text.primary,
            fontFamily: '"Nunito", "DIN Round Pro", sans-serif',
          }}
        >
          Courses for {nativeLang?.name || 'English'} Speakers
        </Text>

        {/* Native language selector */}
        <Menu shadow="md" width={200}>
          <Menu.Target>
            <Button
              variant="subtle"
              color="gray"
              rightSection={<IconChevronDown size={16} />}
              style={{ textTransform: 'uppercase', fontWeight: 600, color: colors.text.secondary }}
            >
              I speak {nativeLang?.name || 'English'}
            </Button>
          </Menu.Target>
          <Menu.Dropdown>
            {languages.filter((l) => l.enabled).map((lang) => (
              <Menu.Item
                key={lang.id}
                onClick={() => setNativeLanguage(lang.id)}
                leftSection={<Text size="lg">{lang.flag_emoji}</Text>}
              >
                {lang.name}
              </Menu.Item>
            ))}
          </Menu.Dropdown>
        </Menu>
      </Group>

      {/* Courses grid */}
      {courses.length === 0 ? (
        <Center h={200}>
          <Stack align="center" gap="md">
            <Text fw={700} size="lg" style={{ color: colors.text.primary }}>
              No courses available
            </Text>
            <Text c="dimmed">
              Try selecting a different native language
            </Text>
          </Stack>
        </Center>
      ) : (
        <SimpleGrid
          cols={{ base: 2, xs: 2, sm: 3, md: 4 }}
          spacing="lg"
        >
          {courses.map((course) => (
            <CourseCard
              key={course.id}
              course={course}
              language={languageMap[course.learning_language_id]}
              isActive={user?.active_course_id === course.id}
              onClick={() => handleCourseSelect(course)}
            />
          ))}
        </SimpleGrid>
      )}
    </Box>
  )
}
