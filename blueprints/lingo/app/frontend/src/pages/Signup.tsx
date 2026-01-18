import { useState } from 'react'
import { Container, Title, TextInput, PasswordInput, Button, Text, Anchor, Stack, Alert, Select } from '@mantine/core'
import { IconMail, IconLock, IconUser, IconAlertCircle } from '@tabler/icons-react'
import { useNavigate } from 'react-router-dom'
import { useAuthStore } from '../stores/auth'
import { colors } from '../styles/tokens'

const ageOptions = [
  { value: '13-17', label: '13-17' },
  { value: '18-24', label: '18-24' },
  { value: '25-34', label: '25-34' },
  { value: '35-44', label: '35-44' },
  { value: '45-54', label: '45-54' },
  { value: '55+', label: '55+' },
]

export default function Signup() {
  const navigate = useNavigate()
  const { signup } = useAuthStore()
  const [email, setEmail] = useState('')
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [age, setAge] = useState<string | null>(null)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setLoading(true)

    try {
      await signup(email, username, password)
      navigate('/learn')
    } catch (err: any) {
      setError(err.message || 'Signup failed')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div style={{
      minHeight: '100vh',
      backgroundColor: '#FFFFFF',
      display: 'flex',
      flexDirection: 'column',
      alignItems: 'center',
      paddingTop: 40,
    }}>
      {/* Header */}
      <Text
        fw={800}
        style={{
          color: colors.primary.green,
          fontSize: '1.75rem',
          cursor: 'pointer',
          marginBottom: 40,
        }}
        onClick={() => navigate('/')}
      >
        lingo
      </Text>

      <Container size={400}>
        <Title order={2} ta="center" mb="xl" style={{ color: colors.text.primary }}>
          Create your profile
        </Title>

        {error && (
          <Alert icon={<IconAlertCircle size={16} />} color="red" mb="md" radius="lg">
            {error}
          </Alert>
        )}

        <form onSubmit={handleSubmit}>
          <Stack gap="md">
            <Select
              placeholder="Age"
              data={ageOptions}
              value={age}
              onChange={setAge}
              size="lg"
              radius="xl"
              styles={{
                input: {
                  backgroundColor: '#FFFFFF',
                  border: `2px solid ${colors.neutral.border}`,
                  color: colors.text.primary,
                },
              }}
            />

            <TextInput
              placeholder="Name (optional)"
              leftSection={<IconUser size={18} style={{ color: colors.text.muted }} />}
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              size="lg"
              radius="xl"
              styles={{
                input: {
                  backgroundColor: '#FFFFFF',
                  border: `2px solid ${colors.neutral.border}`,
                  color: colors.text.primary,
                },
              }}
            />

            <TextInput
              placeholder="Email"
              leftSection={<IconMail size={18} style={{ color: colors.text.muted }} />}
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
              size="lg"
              radius="xl"
              styles={{
                input: {
                  backgroundColor: '#FFFFFF',
                  border: `2px solid ${colors.neutral.border}`,
                  color: colors.text.primary,
                },
              }}
            />

            <PasswordInput
              placeholder="Password"
              leftSection={<IconLock size={18} style={{ color: colors.text.muted }} />}
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
              size="lg"
              radius="xl"
              styles={{
                input: {
                  backgroundColor: '#FFFFFF',
                  border: `2px solid ${colors.neutral.border}`,
                  color: colors.text.primary,
                },
              }}
            />

            <Button
              type="submit"
              fullWidth
              color="green"
              radius="xl"
              size="lg"
              loading={loading}
              style={{
                fontWeight: 700,
                textTransform: 'uppercase',
                marginTop: 8,
                boxShadow: '0 4px 0 #58A700',
              }}
            >
              Create account
            </Button>
          </Stack>
        </form>

        <Text size="xs" ta="center" mt="lg" style={{ color: colors.text.muted }}>
          By signing in to Lingo, you agree to our{' '}
          <Anchor style={{ color: colors.text.muted }}>Terms</Anchor> and{' '}
          <Anchor style={{ color: colors.text.muted }}>Privacy Policy</Anchor>.
        </Text>

        <div style={{
          display: 'flex',
          alignItems: 'center',
          gap: 16,
          margin: '32px 0',
        }}>
          <div style={{ flex: 1, height: 1, backgroundColor: colors.neutral.border }} />
          <Text size="sm" style={{ color: colors.text.muted }}>OR</Text>
          <div style={{ flex: 1, height: 1, backgroundColor: colors.neutral.border }} />
        </div>

        <Stack gap="sm">
          <Button
            variant="outline"
            fullWidth
            radius="xl"
            size="lg"
            leftSection={<Text>🔵</Text>}
            style={{
              borderColor: colors.neutral.border,
              borderWidth: 2,
              color: colors.secondary.blue,
              fontWeight: 700,
              textTransform: 'uppercase',
            }}
          >
            Facebook
          </Button>
          <Button
            variant="outline"
            fullWidth
            radius="xl"
            size="lg"
            leftSection={<Text>🔴</Text>}
            style={{
              borderColor: colors.neutral.border,
              borderWidth: 2,
              color: colors.text.primary,
              fontWeight: 700,
              textTransform: 'uppercase',
            }}
          >
            Google
          </Button>
        </Stack>

        <Text ta="center" mt="xl" style={{ color: colors.text.secondary }}>
          Already have an account?{' '}
          <Anchor
            component="button"
            onClick={() => navigate('/login')}
            style={{ color: colors.secondary.blue, fontWeight: 700 }}
          >
            Log in
          </Anchor>
        </Text>
      </Container>
    </div>
  )
}
