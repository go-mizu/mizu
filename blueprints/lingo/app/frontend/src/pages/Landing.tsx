import { Container, Title, Text, Button, Group, Stack, Paper, SimpleGrid } from '@mantine/core'
import { IconBrain, IconDeviceGamepad2, IconChartLine } from '@tabler/icons-react'
import { useNavigate } from 'react-router-dom'
import { motion } from 'framer-motion'
import { colors } from '../styles/tokens'

const languages = [
  { name: 'Spanish', flag: '🇪🇸', learners: '34.2M' },
  { name: 'French', flag: '🇫🇷', learners: '23.1M' },
  { name: 'German', flag: '🇩🇪', learners: '17.8M' },
  { name: 'Japanese', flag: '🇯🇵', learners: '16.4M' },
  { name: 'Korean', flag: '🇰🇷', learners: '13.2M' },
  { name: 'Chinese', flag: '🇨🇳', learners: '11.9M' },
]

const features = [
  {
    icon: IconBrain,
    title: 'Effective and efficient',
    description: 'Our courses effectively teach reading, listening, and speaking skills.',
    color: '#58CC02',
  },
  {
    icon: IconDeviceGamepad2,
    title: 'Personalized learning',
    description: 'Combining the best of AI and language science, lessons are tailored to help you learn at just the right level and pace.',
    color: '#1CB0F6',
  },
  {
    icon: IconChartLine,
    title: 'Stay motivated',
    description: 'We make it easy to form a habit of language learning with game-like features, fun challenges, and reminders.',
    color: '#FF9600',
  },
]

export default function Landing() {
  const navigate = useNavigate()

  return (
    <div style={{ minHeight: '100vh', backgroundColor: '#FFFFFF' }}>
      {/* Header */}
      <Container size="xl" py="md">
        <Group justify="space-between">
          <Text
            size="xl"
            fw={800}
            style={{
              color: colors.primary.green,
              cursor: 'pointer',
              letterSpacing: '-0.5px',
              fontSize: '1.75rem',
            }}
            onClick={() => navigate('/')}
          >
            lingo
          </Text>
          <Group gap="sm">
            <Button
              variant="subtle"
              color="gray"
              radius="xl"
              onClick={() => navigate('/login')}
              style={{ fontWeight: 700, textTransform: 'uppercase' }}
            >
              Log in
            </Button>
          </Group>
        </Group>
      </Container>

      {/* Hero Section */}
      <Container size="lg" py={80}>
        <Group justify="center" align="center" gap={80}>
          {/* Mascot placeholder */}
          <motion.div
            initial={{ opacity: 0, x: -20 }}
            animate={{ opacity: 1, x: 0 }}
            transition={{ duration: 0.5 }}
          >
            <div style={{
              width: 300,
              height: 300,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
            }}>
              <Text style={{ fontSize: 200 }}>🦉</Text>
            </div>
          </motion.div>

          {/* Hero text */}
          <motion.div
            initial={{ opacity: 0, x: 20 }}
            animate={{ opacity: 1, x: 0 }}
            transition={{ duration: 0.5 }}
          >
            <Stack gap="lg" maw={400}>
              <Title
                order={1}
                style={{
                  color: colors.text.primary,
                  fontSize: '2rem',
                  lineHeight: 1.3,
                }}
              >
                The free, fun, and effective way to learn a language!
              </Title>
              <Stack gap="sm">
                <Button
                  fullWidth
                  size="lg"
                  color="green"
                  radius="xl"
                  onClick={() => navigate('/signup')}
                  style={{
                    fontWeight: 700,
                    textTransform: 'uppercase',
                    fontSize: 15,
                    height: 50,
                    boxShadow: '0 4px 0 #58A700',
                  }}
                >
                  Get started
                </Button>
                <Button
                  fullWidth
                  size="lg"
                  variant="outline"
                  color="blue"
                  radius="xl"
                  onClick={() => navigate('/login')}
                  style={{
                    fontWeight: 700,
                    textTransform: 'uppercase',
                    fontSize: 15,
                    height: 50,
                    borderWidth: 2,
                  }}
                >
                  I already have an account
                </Button>
              </Stack>
            </Stack>
          </motion.div>
        </Group>
      </Container>

      {/* Language Selection */}
      <div style={{ backgroundColor: '#F7F7F7', padding: '60px 0' }}>
        <Container size="lg">
          <Title order={2} ta="center" mb="xl" style={{ color: colors.text.primary }}>
            Choose a language to get started
          </Title>
          <SimpleGrid cols={{ base: 2, sm: 3, md: 6 }} spacing="md">
            {languages.map((lang, index) => (
              <motion.div
                key={lang.name}
                initial={{ opacity: 0, y: 20 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.3, delay: index * 0.05 }}
              >
                <Paper
                  p="lg"
                  radius="lg"
                  style={{
                    backgroundColor: '#FFFFFF',
                    textAlign: 'center',
                    cursor: 'pointer',
                    transition: 'all 0.15s ease',
                    border: '2px solid transparent',
                  }}
                  onClick={() => navigate('/signup')}
                  onMouseEnter={(e) => {
                    e.currentTarget.style.borderColor = colors.primary.green
                    e.currentTarget.style.transform = 'translateY(-4px)'
                  }}
                  onMouseLeave={(e) => {
                    e.currentTarget.style.borderColor = 'transparent'
                    e.currentTarget.style.transform = 'translateY(0)'
                  }}
                >
                  <Text style={{ fontSize: 48 }} mb="xs">{lang.flag}</Text>
                  <Text fw={700} style={{ color: colors.text.primary }}>{lang.name}</Text>
                  <Text size="xs" style={{ color: colors.text.muted }}>{lang.learners} learners</Text>
                </Paper>
              </motion.div>
            ))}
          </SimpleGrid>
        </Container>
      </div>

      {/* Features Section */}
      <Container size="lg" py={80}>
        <SimpleGrid cols={{ base: 1, md: 3 }} spacing="xl">
          {features.map((feature, index) => (
            <motion.div
              key={feature.title}
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.5, delay: 0.2 + index * 0.1 }}
            >
              <Stack align="center" ta="center">
                <div style={{
                  width: 80,
                  height: 80,
                  borderRadius: '50%',
                  backgroundColor: `${feature.color}20`,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                }}>
                  <feature.icon size={40} style={{ color: feature.color }} />
                </div>
                <Title order={3} style={{ color: colors.text.primary }}>
                  {feature.title}
                </Title>
                <Text style={{ color: colors.text.secondary }}>
                  {feature.description}
                </Text>
              </Stack>
            </motion.div>
          ))}
        </SimpleGrid>
      </Container>

      {/* Stats Banner */}
      <div style={{ backgroundColor: colors.primary.green, padding: '40px 0' }}>
        <Container size="lg">
          <Group justify="center" gap={80}>
            {[
              { value: '500M+', label: 'learners' },
              { value: '40+', label: 'languages' },
              { value: '#1', label: 'education app' },
            ].map((stat) => (
              <Stack key={stat.label} gap={0} align="center">
                <Title order={1} style={{ color: 'white', fontSize: 40 }}>{stat.value}</Title>
                <Text fw={600} style={{ color: 'rgba(255,255,255,0.8)', textTransform: 'uppercase' }}>
                  {stat.label}
                </Text>
              </Stack>
            ))}
          </Group>
        </Container>
      </div>

      {/* CTA Section */}
      <Container size="sm" py={80}>
        <Stack align="center" gap="lg">
          <Title order={2} ta="center" style={{ color: colors.text.primary }}>
            Start learning for free today
          </Title>
          <Text ta="center" style={{ color: colors.text.secondary }} maw={400}>
            Join millions of learners mastering new languages with bite-sized lessons and personalized learning.
          </Text>
          <Button
            size="xl"
            color="green"
            radius="xl"
            onClick={() => navigate('/signup')}
            style={{
              fontWeight: 700,
              textTransform: 'uppercase',
              fontSize: 16,
              paddingLeft: 48,
              paddingRight: 48,
              boxShadow: '0 4px 0 #58A700',
            }}
          >
            Get started - It's free
          </Button>
        </Stack>
      </Container>

      {/* Footer */}
      <div style={{ borderTop: '1px solid #E5E5E5', padding: '40px 0' }}>
        <Container size="lg">
          <Group justify="space-between" align="flex-start">
            <div>
              <Text
                fw={800}
                style={{
                  color: colors.primary.green,
                  fontSize: '1.5rem',
                  marginBottom: 8,
                }}
              >
                lingo
              </Text>
              <Text size="sm" style={{ color: colors.text.muted }}>
                Language learning, made fun.
              </Text>
            </div>
            <Group gap={40}>
              <Stack gap="xs">
                <Text fw={700} size="sm" style={{ color: colors.text.primary }}>About</Text>
                <Text size="sm" style={{ color: colors.text.secondary, cursor: 'pointer' }}>Mission</Text>
                <Text size="sm" style={{ color: colors.text.secondary, cursor: 'pointer' }}>Approach</Text>
                <Text size="sm" style={{ color: colors.text.secondary, cursor: 'pointer' }}>Efficacy</Text>
              </Stack>
              <Stack gap="xs">
                <Text fw={700} size="sm" style={{ color: colors.text.primary }}>Products</Text>
                <Text size="sm" style={{ color: colors.text.secondary, cursor: 'pointer' }}>Lingo</Text>
                <Text size="sm" style={{ color: colors.text.secondary, cursor: 'pointer' }}>Lingo for Schools</Text>
                <Text size="sm" style={{ color: colors.text.secondary, cursor: 'pointer' }}>Podcast</Text>
              </Stack>
              <Stack gap="xs">
                <Text fw={700} size="sm" style={{ color: colors.text.primary }}>Help</Text>
                <Text size="sm" style={{ color: colors.text.secondary, cursor: 'pointer' }}>Support</Text>
                <Text size="sm" style={{ color: colors.text.secondary, cursor: 'pointer' }}>Terms</Text>
                <Text size="sm" style={{ color: colors.text.secondary, cursor: 'pointer' }}>Privacy</Text>
              </Stack>
            </Group>
          </Group>
        </Container>
      </div>
    </div>
  )
}
