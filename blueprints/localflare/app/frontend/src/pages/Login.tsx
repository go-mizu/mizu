import { Container, Paper, Title, TextInput, PasswordInput, Button, Stack, Text, Group, Anchor, Center } from '@mantine/core'
import { IconCloud } from '@tabler/icons-react'
import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { notifications } from '@mantine/notifications'

export function Login() {
  const navigate = useNavigate()
  const [loading, setLoading] = useState(false)
  const [form, setForm] = useState({
    email: '',
    password: '',
  })

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)

    try {
      const res = await fetch('/api/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(form),
      })

      if (res.ok) {
        notifications.show({ title: 'Welcome', message: 'Login successful', color: 'green' })
        navigate('/')
      } else {
        const data = await res.json()
        notifications.show({ title: 'Error', message: data.message || 'Invalid credentials', color: 'red' })
      }
    } catch (error) {
      notifications.show({ title: 'Error', message: 'Failed to login', color: 'red' })
    } finally {
      setLoading(false)
    }
  }

  return (
    <Container size={420} py={80}>
      <Center mb="xl">
        <Group gap="xs">
          <IconCloud size={40} color="var(--mantine-color-orange-6)" />
          <Title order={1} c="orange">Localflare</Title>
        </Group>
      </Center>

      <Paper withBorder shadow="md" p={30} radius="md">
        <Title order={2} ta="center" mb="md">
          Welcome back
        </Title>
        <Text c="dimmed" size="sm" ta="center" mb="xl">
          Sign in to your Localflare account
        </Text>

        <form onSubmit={handleSubmit}>
          <Stack gap="md">
            <TextInput
              label="Email"
              placeholder="you@example.com"
              required
              value={form.email}
              onChange={(e) => setForm({ ...form, email: e.target.value })}
            />
            <PasswordInput
              label="Password"
              placeholder="Your password"
              required
              value={form.password}
              onChange={(e) => setForm({ ...form, password: e.target.value })}
            />
            <Button type="submit" fullWidth loading={loading}>
              Sign in
            </Button>
          </Stack>
        </form>

        <Text c="dimmed" size="sm" ta="center" mt="md">
          Don't have an account?{' '}
          <Anchor component="button" size="sm" onClick={() => navigate('/register')}>
            Create account
          </Anchor>
        </Text>
      </Paper>

      <Text c="dimmed" size="xs" ta="center" mt="xl">
        Localflare - 100% Offline Cloudflare Alternative
      </Text>
    </Container>
  )
}
