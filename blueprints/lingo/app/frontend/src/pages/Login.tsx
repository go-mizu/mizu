import { useState } from 'react'
import { Container, Paper, Title, TextInput, PasswordInput, Button, Text, Anchor, Stack, Alert } from '@mantine/core'
import { IconMail, IconLock, IconAlertCircle } from '@tabler/icons-react'
import { useNavigate } from 'react-router-dom'
import { useAuthStore } from '../stores/auth'

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
      backgroundColor: '#131f24',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
    }}>
      <Container size={420}>
        <Title
          ta="center"
          style={{ color: '#58cc02', fontWeight: 800, fontSize: 42, cursor: 'pointer' }}
          onClick={() => navigate('/')}
          mb="xl"
        >
          Lingo
        </Title>

        <Paper radius="lg" p="xl" withBorder style={{ backgroundColor: '#1a2c33', borderColor: '#3d5a68' }}>
          <Title order={2} ta="center" mb="md" style={{ color: 'white' }}>
            Welcome back!
          </Title>

          {error && (
            <Alert icon={<IconAlertCircle size={16} />} color="red" mb="md">
              {error}
            </Alert>
          )}

          <form onSubmit={handleSubmit}>
            <Stack gap="md">
              <TextInput
                label="Email"
                placeholder="your@email.com"
                leftSection={<IconMail size={16} />}
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                required
                styles={{
                  input: { backgroundColor: '#233a42', borderColor: '#3d5a68', color: 'white' },
                  label: { color: '#8fa8b2' },
                }}
              />

              <PasswordInput
                label="Password"
                placeholder="Your password"
                leftSection={<IconLock size={16} />}
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
                styles={{
                  input: { backgroundColor: '#233a42', borderColor: '#3d5a68', color: 'white' },
                  label: { color: '#8fa8b2' },
                }}
              />

              <Button type="submit" fullWidth color="green" radius="xl" size="md" loading={loading} mt="md">
                Log In
              </Button>
            </Stack>
          </form>

          <Text ta="center" mt="md" style={{ color: '#8fa8b2' }}>
            Don't have an account?{' '}
            <Anchor component="button" onClick={() => navigate('/signup')} style={{ color: '#1cb0f6' }}>
              Sign up
            </Anchor>
          </Text>
        </Paper>
      </Container>
    </div>
  )
}
