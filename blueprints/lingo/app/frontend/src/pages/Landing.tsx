import { Container, Title, Text, Group, Stack, Paper, SimpleGrid, Box } from '@mantine/core'
import { IconBrain, IconDeviceGamepad2, IconChartLine, IconChevronDown } from '@tabler/icons-react'
import { useNavigate } from 'react-router-dom'
import { motion } from 'framer-motion'
import { colors } from '../styles/tokens'
import { CharacterGroup } from '../components/mascot/CharacterGroup'
import { DuoButton } from '../components/ui/DuoButton'
import { LanguageCarousel } from '../components/landing/LanguageCarousel'
import { AnimatedCounter } from '../components/ui/AnimatedCounter'

const features = [
  {
    icon: IconBrain,
    title: 'Effective and efficient',
    description: 'Our courses effectively teach reading, listening, and speaking skills. Get the most out of every lesson.',
    color: colors.primary.green,
    bgColor: colors.primary.greenLighter,
  },
  {
    icon: IconDeviceGamepad2,
    title: 'Personalized learning',
    description: 'Combining the best of AI and language science, lessons are tailored to help you learn at just the right level and pace.',
    color: colors.secondary.blue,
    bgColor: colors.secondary.blueLight,
  },
  {
    icon: IconChartLine,
    title: 'Stay motivated',
    description: 'We make it easy to form a habit of language learning with game-like features, fun challenges, and reminders.',
    color: colors.accent.orange,
    bgColor: colors.accent.orangeLight,
  },
]

const stats = [
  { value: '500', suffix: 'M+', label: 'learners worldwide' },
  { value: '40', suffix: '+', label: 'languages to learn' },
  { value: '#1', suffix: '', label: 'education app' },
]

export default function Landing() {
  const navigate = useNavigate()

  return (
    <div style={{ minHeight: '100vh', backgroundColor: '#FFFFFF' }}>
      {/* Header */}
      <motion.header
        initial={{ y: -20, opacity: 0 }}
        animate={{ y: 0, opacity: 1 }}
        transition={{ duration: 0.4 }}
        style={{
          position: 'sticky',
          top: 0,
          zIndex: 100,
          backgroundColor: 'white',
          borderBottom: `1px solid ${colors.neutral.border}`,
        }}
      >
        <Container size="xl" py="md">
          <Group justify="space-between" align="center">
            {/* Logo */}
            <motion.div
              whileHover={{ scale: 1.02 }}
              style={{ cursor: 'pointer' }}
              onClick={() => navigate('/')}
            >
              <Text
                size="xl"
                fw={800}
                style={{
                  color: colors.primary.green,
                  letterSpacing: '-0.5px',
                  fontSize: '1.75rem',
                }}
              >
                lingo
              </Text>
            </motion.div>

            {/* Site language selector */}
            <motion.button
              whileHover={{ backgroundColor: colors.neutral.background }}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 4,
                padding: '8px 12px',
                border: 'none',
                background: 'transparent',
                cursor: 'pointer',
                borderRadius: 8,
                transition: 'background-color 0.15s ease',
              }}
            >
              <Text
                size="sm"
                fw={700}
                style={{
                  color: colors.text.secondary,
                  textTransform: 'uppercase',
                  letterSpacing: '0.5px',
                }}
              >
                Site Language: English
              </Text>
              <IconChevronDown size={16} color={colors.text.secondary} />
            </motion.button>
          </Group>
        </Container>
      </motion.header>

      {/* Hero Section */}
      <Box className="landing-hero" py={60}>
        <Container size="xl">
          <Group justify="center" align="center" gap={60} wrap="wrap">
            {/* Character Group */}
            <motion.div
              className="landing-hero-characters"
              initial={{ opacity: 0, scale: 0.8 }}
              animate={{ opacity: 1, scale: 1 }}
              transition={{ duration: 0.6, delay: 0.2 }}
            >
              <CharacterGroup />
            </motion.div>

            {/* Hero Content */}
            <motion.div
              className="landing-hero-content"
              initial={{ opacity: 0, x: 30 }}
              animate={{ opacity: 1, x: 0 }}
              transition={{ duration: 0.5, delay: 0.3 }}
            >
              <Stack gap="xl" maw={400}>
                <Title
                  order={1}
                  style={{
                    color: colors.text.primary,
                    fontSize: '1.875rem',
                    lineHeight: 1.3,
                    fontWeight: 700,
                  }}
                >
                  The free, fun, and effective way to learn a language!
                </Title>

                <Stack gap="sm">
                  <DuoButton
                    variant="primary"
                    size="lg"
                    fullWidth
                    glow
                    onClick={() => navigate('/signup')}
                  >
                    Get started
                  </DuoButton>

                  <DuoButton
                    variant="outline"
                    size="lg"
                    fullWidth
                    onClick={() => navigate('/login')}
                  >
                    I already have an account
                  </DuoButton>
                </Stack>
              </Stack>
            </motion.div>
          </Group>
        </Container>
      </Box>

      {/* Language Carousel */}
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.5, delay: 0.5 }}
      >
        <LanguageCarousel onSelect={(code) => console.log('Selected:', code)} />
      </motion.div>

      {/* Features Section */}
      <Box py={80} style={{ backgroundColor: colors.neutral.background }}>
        <Container size="lg">
          <motion.div
            initial={{ opacity: 0, y: 30 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true, margin: '-100px' }}
            transition={{ duration: 0.5 }}
          >
            <Title
              order={2}
              ta="center"
              mb={50}
              style={{
                color: colors.text.primary,
                fontSize: '1.75rem',
              }}
            >
              The world's best way to learn a language
            </Title>
          </motion.div>

          <SimpleGrid cols={{ base: 1, md: 3 }} spacing={30}>
            {features.map((feature, index) => (
              <motion.div
                key={feature.title}
                initial={{ opacity: 0, y: 30 }}
                whileInView={{ opacity: 1, y: 0 }}
                viewport={{ once: true, margin: '-50px' }}
                transition={{ duration: 0.5, delay: index * 0.1 }}
              >
                <Paper
                  p="xl"
                  radius="lg"
                  className="feature-card"
                  style={{
                    backgroundColor: 'white',
                    height: '100%',
                    border: `2px solid ${colors.neutral.border}`,
                  }}
                >
                  <Stack align="center" ta="center" gap="md">
                    <motion.div
                      whileHover={{ scale: 1.1, rotate: 5 }}
                      style={{
                        width: 80,
                        height: 80,
                        borderRadius: '50%',
                        backgroundColor: feature.bgColor,
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                      }}
                    >
                      <feature.icon size={40} style={{ color: feature.color }} stroke={2} />
                    </motion.div>

                    <Title
                      order={3}
                      style={{
                        color: colors.text.primary,
                        fontSize: '1.25rem',
                      }}
                    >
                      {feature.title}
                    </Title>

                    <Text
                      size="sm"
                      style={{
                        color: colors.text.secondary,
                        lineHeight: 1.6,
                      }}
                    >
                      {feature.description}
                    </Text>
                  </Stack>
                </Paper>
              </motion.div>
            ))}
          </SimpleGrid>
        </Container>
      </Box>

      {/* Stats Banner */}
      <Box
        className="stats-section"
        py={50}
        style={{
          background: `linear-gradient(135deg, ${colors.primary.green} 0%, ${colors.primary.greenHover} 100%)`,
        }}
      >
        <Container size="lg">
          <Group justify="center" gap={80} wrap="wrap">
            {stats.map((stat, index) => (
              <motion.div
                key={stat.label}
                initial={{ opacity: 0, scale: 0.8 }}
                whileInView={{ opacity: 1, scale: 1 }}
                viewport={{ once: true }}
                transition={{ duration: 0.5, delay: index * 0.1 }}
              >
                <Stack gap={4} align="center">
                  <Title
                    order={1}
                    style={{
                      color: 'white',
                      fontSize: '2.5rem',
                      fontWeight: 800,
                    }}
                  >
                    {stat.value === '#1' ? (
                      stat.value
                    ) : (
                      <AnimatedCounter
                        value={parseFloat(stat.value)}
                        suffix={stat.suffix}
                        style={{ color: 'white' }}
                      />
                    )}
                  </Title>
                  <Text
                    fw={600}
                    size="sm"
                    style={{
                      color: 'rgba(255,255,255,0.9)',
                      textTransform: 'uppercase',
                      letterSpacing: '0.5px',
                    }}
                  >
                    {stat.label}
                  </Text>
                </Stack>
              </motion.div>
            ))}
          </Group>
        </Container>
      </Box>

      {/* Why Lingo Section */}
      <Box py={80}>
        <Container size="md">
          <motion.div
            initial={{ opacity: 0, y: 30 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true, margin: '-100px' }}
            transition={{ duration: 0.5 }}
          >
            <Stack align="center" gap="xl">
              <Title
                order={2}
                ta="center"
                style={{
                  color: colors.text.primary,
                  fontSize: '1.75rem',
                }}
              >
                Learn anytime, anywhere
              </Title>

              <Text
                ta="center"
                size="lg"
                style={{
                  color: colors.text.secondary,
                  maxWidth: 500,
                  lineHeight: 1.7,
                }}
              >
                Make your breaks and commutes more productive with our mobile app.
                Download for free and learn on the go!
              </Text>

              <Group gap="md" mt="md">
                <motion.div whileHover={{ scale: 1.05 }} whileTap={{ scale: 0.98 }}>
                  <Paper
                    p="md"
                    radius="md"
                    style={{
                      backgroundColor: '#000',
                      cursor: 'pointer',
                      display: 'flex',
                      alignItems: 'center',
                      gap: 12,
                    }}
                  >
                    <svg width="24" height="24" viewBox="0 0 24 24" fill="white">
                      <path d="M17.05 20.28c-.98.95-2.05.8-3.08.35-1.09-.46-2.09-.48-3.24 0-1.44.62-2.2.44-3.06-.35C2.79 15.25 3.51 7.59 9.05 7.31c1.35.07 2.29.74 3.08.8 1.18-.24 2.31-.93 3.57-.84 1.51.12 2.65.72 3.4 1.8-3.12 1.87-2.38 5.98.48 7.13-.57 1.5-1.31 2.99-2.54 4.09zM12.03 7.25c-.15-2.23 1.66-4.07 3.74-4.25.29 2.58-2.34 4.5-3.74 4.25z"/>
                    </svg>
                    <Stack gap={0}>
                      <Text size="xs" style={{ color: 'rgba(255,255,255,0.8)' }}>
                        Download on the
                      </Text>
                      <Text fw={600} style={{ color: 'white' }}>
                        App Store
                      </Text>
                    </Stack>
                  </Paper>
                </motion.div>

                <motion.div whileHover={{ scale: 1.05 }} whileTap={{ scale: 0.98 }}>
                  <Paper
                    p="md"
                    radius="md"
                    style={{
                      backgroundColor: '#000',
                      cursor: 'pointer',
                      display: 'flex',
                      alignItems: 'center',
                      gap: 12,
                    }}
                  >
                    <svg width="24" height="24" viewBox="0 0 24 24" fill="white">
                      <path d="M3.609 1.814L13.792 12 3.61 22.186a.996.996 0 0 1-.61-.92V2.734a1 1 0 0 1 .609-.92zm10.89 10.893l2.302 2.302-10.937 6.333 8.635-8.635zm3.199-3.198l2.807 1.626a1 1 0 0 1 0 1.73l-2.808 1.626L15.206 12l2.492-2.491zM5.864 2.658L16.8 8.99l-2.302 2.302-8.634-8.634z"/>
                    </svg>
                    <Stack gap={0}>
                      <Text size="xs" style={{ color: 'rgba(255,255,255,0.8)' }}>
                        Get it on
                      </Text>
                      <Text fw={600} style={{ color: 'white' }}>
                        Google Play
                      </Text>
                    </Stack>
                  </Paper>
                </motion.div>
              </Group>
            </Stack>
          </motion.div>
        </Container>
      </Box>

      {/* Final CTA Section */}
      <Box py={80} style={{ backgroundColor: colors.neutral.background }}>
        <Container size="sm">
          <motion.div
            initial={{ opacity: 0, y: 30 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true }}
            transition={{ duration: 0.5 }}
          >
            <Stack align="center" gap="xl">
              <Title
                order={2}
                ta="center"
                style={{
                  color: colors.text.primary,
                  fontSize: '1.75rem',
                }}
              >
                Start learning for free today
              </Title>

              <Text
                ta="center"
                style={{
                  color: colors.text.secondary,
                  maxWidth: 400,
                  lineHeight: 1.6,
                }}
              >
                Join millions of learners mastering new languages with bite-sized lessons
                and personalized learning paths.
              </Text>

              <DuoButton
                variant="primary"
                size="xl"
                glow
                onClick={() => navigate('/signup')}
                style={{ paddingLeft: 48, paddingRight: 48 }}
              >
                Get started - It's free
              </DuoButton>
            </Stack>
          </motion.div>
        </Container>
      </Box>

      {/* Footer */}
      <Box
        component="footer"
        py={50}
        style={{
          borderTop: `1px solid ${colors.neutral.border}`,
          backgroundColor: 'white',
        }}
      >
        <Container size="lg">
          <Group justify="space-between" align="flex-start" wrap="wrap" gap={40}>
            {/* Logo and tagline */}
            <Stack gap="xs" maw={200}>
              <Text
                fw={800}
                style={{
                  color: colors.primary.green,
                  fontSize: '1.5rem',
                }}
              >
                lingo
              </Text>
              <Text size="sm" style={{ color: colors.text.muted }}>
                Language learning, made fun and effective.
              </Text>
            </Stack>

            {/* Footer links */}
            <Group gap={60} wrap="wrap">
              <Stack gap="sm">
                <Text fw={700} size="sm" style={{ color: colors.text.primary }}>
                  About
                </Text>
                {['Mission', 'Approach', 'Efficacy', 'Team', 'Careers'].map((link) => (
                  <Text
                    key={link}
                    size="sm"
                    component="a"
                    href="#"
                    style={{
                      color: colors.text.secondary,
                      textDecoration: 'none',
                      cursor: 'pointer',
                      transition: 'color 0.15s ease',
                    }}
                    onMouseEnter={(e) => e.currentTarget.style.color = colors.primary.green}
                    onMouseLeave={(e) => e.currentTarget.style.color = colors.text.secondary}
                  >
                    {link}
                  </Text>
                ))}
              </Stack>

              <Stack gap="sm">
                <Text fw={700} size="sm" style={{ color: colors.text.primary }}>
                  Products
                </Text>
                {['Lingo', 'Lingo for Schools', 'Lingo English Test', 'Podcast', 'Stories'].map((link) => (
                  <Text
                    key={link}
                    size="sm"
                    component="a"
                    href="#"
                    style={{
                      color: colors.text.secondary,
                      textDecoration: 'none',
                      cursor: 'pointer',
                      transition: 'color 0.15s ease',
                    }}
                    onMouseEnter={(e) => e.currentTarget.style.color = colors.primary.green}
                    onMouseLeave={(e) => e.currentTarget.style.color = colors.text.secondary}
                  >
                    {link}
                  </Text>
                ))}
              </Stack>

              <Stack gap="sm">
                <Text fw={700} size="sm" style={{ color: colors.text.primary }}>
                  Help
                </Text>
                {['Support', 'Help Center', 'Terms', 'Privacy', 'Community'].map((link) => (
                  <Text
                    key={link}
                    size="sm"
                    component="a"
                    href="#"
                    style={{
                      color: colors.text.secondary,
                      textDecoration: 'none',
                      cursor: 'pointer',
                      transition: 'color 0.15s ease',
                    }}
                    onMouseEnter={(e) => e.currentTarget.style.color = colors.primary.green}
                    onMouseLeave={(e) => e.currentTarget.style.color = colors.text.secondary}
                  >
                    {link}
                  </Text>
                ))}
              </Stack>
            </Group>
          </Group>

          {/* Copyright */}
          <Text
            size="xs"
            ta="center"
            mt={50}
            style={{ color: colors.text.muted }}
          >
            © 2025 Lingo. All rights reserved.
          </Text>
        </Container>
      </Box>
    </div>
  )
}
