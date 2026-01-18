import { useState } from 'react'
import { Container, Title, TextInput, PasswordInput, Button, Text, Anchor, Stack, Alert } from '@mantine/core'
import { IconMail, IconLock, IconAlertCircle } from '@tabler/icons-react'
import { useNavigate } from 'react-router-dom'
import { useAuthStore } from '../stores/auth'
import { colors } from '../styles/tokens'

export default function Login() {
  const navigate = useNavigate()
  const { login } = useAuthStore()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setLoading(true)

    try {
      await login(email, password)
      navigate('/learn')
    } catch (err: any) {
      setError(err.message || 'Login failed')
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
          Log in
        </Title>

        {error && (
          <Alert icon={<IconAlertCircle size={16} />} color="red" mb="md" radius="lg">
            {error}
          </Alert>
        )}

        <form onSubmit={handleSubmit}>
          <Stack gap="md">
            <TextInput
              placeholder="Email or username"
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
                  '&:focus': {
                    borderColor: colors.secondary.blue,
                  },
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

            <Anchor
              component="button"
              type="button"
              ta="right"
              size="sm"
              style={{ color: colors.secondary.blue }}
            >
              Forgot password?
            </Anchor>

            <Button
              type="submit"
              fullWidth
              color="blue"
              radius="xl"
              size="lg"
              loading={loading}
              style={{
                fontWeight: 700,
                textTransform: 'uppercase',
                marginTop: 8,
                boxShadow: '0 4px 0 #1899D6',
              }}
            >
              Log in
            </Button>
          </Stack>
        </form>

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
          Don't have an account?{' '}
          <Anchor
            component="button"
            onClick={() => navigate('/signup')}
            style={{ color: colors.secondary.blue, fontWeight: 700 }}
          >
            Sign up
          </Anchor>
        </Text>
      </Container>
    </div>
  )
}
