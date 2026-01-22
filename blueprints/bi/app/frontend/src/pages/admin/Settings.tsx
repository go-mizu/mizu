import { useEffect, useState } from 'react'
import { Container, Title, Text, Card, Group, Stack, Button, TextInput, Switch, Divider, PasswordInput, Badge } from '@mantine/core'
import { IconDeviceFloppy, IconUser, IconLock, IconPalette } from '@tabler/icons-react'
import { api } from '../../api/client'

interface User {
  id: string
  email: string
  name: string
  role: string
}

interface Settings {
  site_name: string
  enable_public_sharing: boolean
  enable_embedding: boolean
  admin_email: string
}

export default function Settings() {
  const [user, setUser] = useState<User | null>(null)
  const [settings, setSettings] = useState<Settings>({
    site_name: 'BI',
    enable_public_sharing: false,
    enable_embedding: false,
    admin_email: '',
  })
  const [, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)

  // Profile form
  const [name, setName] = useState('')
  const [email, setEmail] = useState('')
  const [currentPassword, setCurrentPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')

  useEffect(() => {
    loadData()
  }, [])

  const loadData = async () => {
    try {
      const [userRes, settingsRes] = await Promise.all([
        api.get<User>('/users/me'),
        api.get<Settings>('/settings'),
      ])
      setUser(userRes)
      setName(userRes?.name || '')
      setEmail(userRes?.email || '')
      if (settingsRes) {
        setSettings(settingsRes)
      }
    } catch (error) {
      console.error('Failed to load settings:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleSaveProfile = async () => {
    setSaving(true)
    try {
      await api.put('/users/me', { name, email })
      alert('Profile updated!')
    } catch (err) {
      alert('Failed to update profile')
    } finally {
      setSaving(false)
    }
  }

  const handleChangePassword = async () => {
    if (newPassword !== confirmPassword) {
      alert('Passwords do not match')
      return
    }
    setSaving(true)
    try {
      await api.post('/users/me/password', {
        current_password: currentPassword,
        new_password: newPassword,
      })
      alert('Password changed!')
      setCurrentPassword('')
      setNewPassword('')
      setConfirmPassword('')
    } catch (err) {
      alert('Failed to change password')
    } finally {
      setSaving(false)
    }
  }

  const handleSaveSettings = async () => {
    setSaving(true)
    try {
      await api.put('/settings', settings)
      alert('Settings saved!')
    } catch (err) {
      alert('Failed to save settings')
    } finally {
      setSaving(false)
    }
  }

  return (
    <Container size="md" py="lg">
      {/* Header */}
      <Group justify="space-between" mb="xl">
        <div>
          <Title order={2}>Settings</Title>
          <Text c="dimmed">Manage your account and application settings</Text>
        </div>
      </Group>

      <Stack gap="lg">
        {/* Profile */}
        <Card withBorder radius="md" padding="lg">
          <Group mb="md">
            <IconUser size={24} color="var(--mantine-color-blue-6)" />
            <Title order={4}>Profile</Title>
            {user?.role && (
              <Badge color={user.role === 'admin' ? 'red' : 'blue'} variant="light">
                {user.role}
              </Badge>
            )}
          </Group>
          <Stack>
            <TextInput
              label="Name"
              placeholder="Your name"
              value={name}
              onChange={(e) => setName(e.target.value)}
            />
            <TextInput
              label="Email"
              placeholder="your@email.com"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
            />
            <Group justify="flex-end">
              <Button
                leftSection={<IconDeviceFloppy size={16} />}
                onClick={handleSaveProfile}
                loading={saving}
              >
                Save Profile
              </Button>
            </Group>
          </Stack>
        </Card>

        {/* Password */}
        <Card withBorder radius="md" padding="lg">
          <Group mb="md">
            <IconLock size={24} color="var(--mantine-color-orange-6)" />
            <Title order={4}>Change Password</Title>
          </Group>
          <Stack>
            <PasswordInput
              label="Current Password"
              placeholder="Enter current password"
              value={currentPassword}
              onChange={(e) => setCurrentPassword(e.target.value)}
            />
            <PasswordInput
              label="New Password"
              placeholder="Enter new password"
              value={newPassword}
              onChange={(e) => setNewPassword(e.target.value)}
            />
            <PasswordInput
              label="Confirm New Password"
              placeholder="Confirm new password"
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
            />
            <Group justify="flex-end">
              <Button
                variant="light"
                onClick={handleChangePassword}
                loading={saving}
                disabled={!currentPassword || !newPassword || !confirmPassword}
              >
                Change Password
              </Button>
            </Group>
          </Stack>
        </Card>

        {/* Application Settings (Admin only) */}
        {user?.role === 'admin' && (
          <Card withBorder radius="md" padding="lg">
            <Group mb="md">
              <IconPalette size={24} color="var(--mantine-color-violet-6)" />
              <Title order={4}>Application Settings</Title>
            </Group>
            <Stack>
              <TextInput
                label="Site Name"
                placeholder="BI"
                value={settings.site_name}
                onChange={(e) => setSettings({ ...settings, site_name: e.target.value })}
              />
              <TextInput
                label="Admin Email"
                placeholder="admin@example.com"
                value={settings.admin_email}
                onChange={(e) => setSettings({ ...settings, admin_email: e.target.value })}
              />
              <Divider my="sm" />
              <Switch
                label="Enable Public Sharing"
                description="Allow users to share dashboards and questions publicly"
                checked={settings.enable_public_sharing}
                onChange={(e) => setSettings({ ...settings, enable_public_sharing: e.currentTarget.checked })}
              />
              <Switch
                label="Enable Embedding"
                description="Allow dashboards and questions to be embedded in other websites"
                checked={settings.enable_embedding}
                onChange={(e) => setSettings({ ...settings, enable_embedding: e.currentTarget.checked })}
              />
              <Group justify="flex-end">
                <Button
                  leftSection={<IconDeviceFloppy size={16} />}
                  onClick={handleSaveSettings}
                  loading={saving}
                >
                  Save Settings
                </Button>
              </Group>
            </Stack>
          </Card>
        )}

        {/* About */}
        <Card withBorder radius="md" padding="lg">
          <Title order={4} mb="md">About</Title>
          <Stack gap="xs">
            <Group>
              <Text size="sm" c="dimmed" w={100}>Version:</Text>
              <Text size="sm">1.0.0</Text>
            </Group>
            <Group>
              <Text size="sm" c="dimmed" w={100}>Built with:</Text>
              <Text size="sm">Mizu Framework</Text>
            </Group>
            <Group>
              <Text size="sm" c="dimmed" w={100}>License:</Text>
              <Text size="sm">MIT</Text>
            </Group>
          </Stack>
        </Card>
      </Stack>
    </Container>
  )
}
