import { useState } from 'react'
import { Container, Paper, Title, TextInput, PasswordInput, Button, Text, Anchor, Stack, Alert } from '@mantine/core'
import { IconMail, IconLock, IconUser, IconAlertCircle } from '@tabler/icons-react'
import { useNavigate } from 'react-router-dom'
import { useAuthStore } from '../stores/auth'

export default function Signup() {
  const navigate = useNavigate()
  const { signup } = useAuthStore()
  const [email, setEmail] = useState('')
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
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
            Create your account
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

              <TextInput
                label="Username"
                placeholder="your_username"
                leftSection={<IconUser size={16} />}
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                required
                styles={{
                  input: { backgroundColor: '#233a42', borderColor: '#3d5a68', color: 'white' },
                  label: { color: '#8fa8b2' },
                }}
              />

              <PasswordInput
                label="Password"
                placeholder="Create a password"
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
                Create Account
              </Button>
            </Stack>
          </form>

          <Text ta="center" mt="md" style={{ color: '#8fa8b2' }}>
            Already have an account?{' '}
            <Anchor component="button" onClick={() => navigate('/login')} style={{ color: '#1cb0f6' }}>
              Log in
            </Anchor>
          </Text>
        </Paper>
      </Container>
    </div>
  )
}
