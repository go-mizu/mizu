import { Container, Title, Text, Button, Group, Stack, Paper, Badge } from '@mantine/core'
import { IconWorld, IconTrophy, IconFlame } from '@tabler/icons-react'
import { useNavigate } from 'react-router-dom'
import { motion } from 'framer-motion'

const features = [
  {
    icon: IconWorld,
    title: '40+ Languages',
    description: 'Learn any language with bite-sized lessons',
  },
  {
    icon: IconTrophy,
    title: 'Gamified Learning',
    description: 'Earn XP, compete in leagues, unlock achievements',
  },
  {
    icon: IconFlame,
    title: 'Daily Streaks',
    description: 'Build habits with streak tracking',
  },
]

export default function Landing() {
  const navigate = useNavigate()

  return (
    <div style={{
      minHeight: '100vh',
      backgroundColor: '#131f24',
      overflow: 'hidden',
    }}>
      {/* Hero Section */}
      <Container size="lg" py={100}>
        <Stack align="center" gap="xl">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.5 }}
          >
            <Badge size="xl" variant="filled" color="green" mb="md">
              Free Forever
            </Badge>
          </motion.div>

          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.5, delay: 0.1 }}
          >
            <Title
              order={1}
              size={60}
              ta="center"
              style={{ color: 'white', lineHeight: 1.2 }}
            >
              The free, fun, and effective way to{' '}
              <span style={{ color: '#58cc02' }}>learn a language!</span>
            </Title>
          </motion.div>

          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.5, delay: 0.2 }}
          >
            <Text size="xl" ta="center" style={{ color: '#8fa8b2' }} maw={600}>
              Learning with Lingo is fun, and research shows that it works!
              With quick, bite-sized lessons, you'll earn points and unlock new levels.
            </Text>
          </motion.div>

          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.5, delay: 0.3 }}
          >
            <Group mt="xl">
              <Button
                size="xl"
                color="green"
                radius="xl"
                onClick={() => navigate('/signup')}
                style={{ fontWeight: 700, fontSize: 18 }}
              >
                Get Started
              </Button>
              <Button
                size="xl"
                variant="outline"
                color="gray"
                radius="xl"
                onClick={() => navigate('/login')}
                style={{ fontWeight: 700, fontSize: 18 }}
              >
                I Already Have an Account
              </Button>
            </Group>
          </motion.div>
        </Stack>
      </Container>

      {/* Features Section */}
      <Container size="lg" py={60}>
        <Group justify="center" gap="xl">
          {features.map((feature, index) => (
            <motion.div
              key={feature.title}
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.5, delay: 0.4 + index * 0.1 }}
            >
              <Paper
                p="xl"
                radius="lg"
                style={{
                  backgroundColor: '#1a2c33',
                  border: '2px solid #3d5a68',
                  width: 280,
                  textAlign: 'center',
                }}
              >
                <feature.icon size={48} style={{ color: '#58cc02' }} />
                <Title order={3} mt="md" style={{ color: 'white' }}>
                  {feature.title}
                </Title>
                <Text mt="sm" style={{ color: '#8fa8b2' }}>
                  {feature.description}
                </Text>
              </Paper>
            </motion.div>
          ))}
        </Group>
      </Container>

      {/* Stats Section */}
      <Container size="lg" py={60}>
        <Paper
          p="xl"
          radius="lg"
          style={{
            backgroundColor: '#58cc02',
            textAlign: 'center',
          }}
        >
          <Group justify="center" gap={80}>
            <Stack gap={0}>
              <Title order={1} style={{ color: 'white', fontSize: 48 }}>500M+</Title>
              <Text size="lg" fw={600} style={{ color: 'rgba(255,255,255,0.9)' }}>Users</Text>
            </Stack>
            <Stack gap={0}>
              <Title order={1} style={{ color: 'white', fontSize: 48 }}>40+</Title>
              <Text size="lg" fw={600} style={{ color: 'rgba(255,255,255,0.9)' }}>Languages</Text>
            </Stack>
            <Stack gap={0}>
              <Title order={1} style={{ color: 'white', fontSize: 48 }}>#1</Title>
              <Text size="lg" fw={600} style={{ color: 'rgba(255,255,255,0.9)' }}>Education App</Text>
            </Stack>
          </Group>
        </Paper>
      </Container>

      {/* Footer */}
      <Container size="lg" py={40}>
        <Text ta="center" style={{ color: '#8fa8b2' }}>
          Made with ❤️ using Mizu Framework
        </Text>
      </Container>
    </div>
  )
}
