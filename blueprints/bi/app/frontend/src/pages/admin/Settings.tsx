import { useState } from 'react'
import {
  Card, Group, Stack, Button, TextInput, Switch, Divider,
  PasswordInput, Badge, Tabs, Table, ActionIcon, Modal, Select, Paper,
  ThemeIcon, Menu, Avatar, SimpleGrid, Text, Title
} from '@mantine/core'
import { useDisclosure } from '@mantine/hooks'
import { notifications } from '@mantine/notifications'
import {
  IconDeviceFloppy, IconUser, IconLock, IconPalette, IconUsers, IconShield,
  IconBell, IconMail, IconWorld, IconTrash, IconEdit,
  IconDotsVertical, IconRefresh, IconSettings, IconActivity,
  IconKey, IconUserPlus, IconAt
} from '@tabler/icons-react'
import {
  useCurrentUser, useUpdateProfile, useChangePassword, useSettings,
  useUpdateSettings, useUsers, useCreateUser, useUpdateUser, useDeleteUser,
  useResetUserPassword, useActivityLog
} from '../../api/hooks'
import type { User, Settings as SettingsType } from '../../api/types'
import { PageHeader, PageContainer, LoadingState } from '../../components/ui'

export default function Settings() {
  const [activeTab, setActiveTab] = useState<string | null>('account')

  const { data: user, isLoading: loadingUser } = useCurrentUser()
  const { data: settings, isLoading: loadingSettings } = useSettings()

  if (loadingUser || loadingSettings) {
    return (
      <PageContainer size="lg">
        <LoadingState message="Loading settings..." />
      </PageContainer>
    )
  }

  const isAdmin = user?.role === 'admin'

  return (
    <PageContainer size="lg">
      <PageHeader
        title="Settings"
        subtitle="Manage your account and application settings"
      />

      <Tabs value={activeTab} onChange={setActiveTab}>
        <Tabs.List mb="lg">
          <Tabs.Tab value="account" leftSection={<IconUser size={16} />}>
            Account
          </Tabs.Tab>
          <Tabs.Tab value="security" leftSection={<IconShield size={16} />}>
            Security
          </Tabs.Tab>
          {isAdmin && (
            <>
              <Tabs.Tab value="users" leftSection={<IconUsers size={16} />}>
                Users
              </Tabs.Tab>
              <Tabs.Tab value="application" leftSection={<IconSettings size={16} />}>
                Application
              </Tabs.Tab>
              <Tabs.Tab value="email" leftSection={<IconMail size={16} />}>
                Email
              </Tabs.Tab>
              <Tabs.Tab value="activity" leftSection={<IconActivity size={16} />}>
                Activity
              </Tabs.Tab>
            </>
          )}
        </Tabs.List>

        <Tabs.Panel value="account">
          <AccountPanel user={user!} />
        </Tabs.Panel>

        <Tabs.Panel value="security">
          <SecurityPanel />
        </Tabs.Panel>

        {isAdmin && (
          <>
            <Tabs.Panel value="users">
              <UsersPanel currentUser={user!} />
            </Tabs.Panel>

            <Tabs.Panel value="application">
              <ApplicationPanel settings={settings} />
            </Tabs.Panel>

            <Tabs.Panel value="email">
              <EmailPanel settings={settings} />
            </Tabs.Panel>

            <Tabs.Panel value="activity">
              <ActivityPanel />
            </Tabs.Panel>
          </>
        )}
      </Tabs>

      {/* About section at the bottom */}
      <Card withBorder radius="md" padding="lg" mt="xl">
        <Title order={4} mb="md">About</Title>
        <SimpleGrid cols={{ base: 1, sm: 3 }}>
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
        </SimpleGrid>
      </Card>
    </PageContainer>
  )
}

// Account Panel
function AccountPanel({ user }: { user: User }) {
  const updateProfile = useUpdateProfile()
  const [name, setName] = useState(user.name || '')
  const [email, setEmail] = useState(user.email || '')

  const handleSaveProfile = async () => {
    try {
      await updateProfile.mutateAsync({ name, email })
      notifications.show({
        title: 'Profile updated',
        message: 'Your profile has been saved',
        color: 'green',
      })
    } catch (err: any) {
      notifications.show({
        title: 'Update failed',
        message: err.message || 'Failed to update profile',
        color: 'red',
      })
    }
  }

  return (
    <Stack gap="lg">
      <Card withBorder radius="md" padding="lg">
        <Group mb="md">
          <ThemeIcon size="lg" variant="light" color="blue">
            <IconUser size={20} />
          </ThemeIcon>
          <div>
            <Title order={4}>Profile</Title>
            <Text size="sm" c="dimmed">Your personal information</Text>
          </div>
          <Badge ml="auto" color={user.role === 'admin' ? 'red' : user.role === 'user' ? 'blue' : 'gray'} variant="light">
            {user.role}
          </Badge>
        </Group>

        <Stack gap="md">
          <Group grow>
            <TextInput
              label="Name"
              placeholder="Your name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              leftSection={<IconUser size={16} />}
            />
            <TextInput
              label="Email"
              placeholder="your@email.com"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              leftSection={<IconAt size={16} />}
            />
          </Group>

          <Group gap="xs">
            <Text size="sm" c="dimmed">Member since:</Text>
            <Text size="sm">{new Date(user.created_at).toLocaleDateString()}</Text>
          </Group>

          {user.last_login && (
            <Group gap="xs">
              <Text size="sm" c="dimmed">Last login:</Text>
              <Text size="sm">{new Date(user.last_login).toLocaleString()}</Text>
            </Group>
          )}

          <Group justify="flex-end" mt="md">
            <Button
              leftSection={<IconDeviceFloppy size={16} />}
              onClick={handleSaveProfile}
              loading={updateProfile.isPending}
            >
              Save Profile
            </Button>
          </Group>
        </Stack>
      </Card>

      <Card withBorder radius="md" padding="lg">
        <Group mb="md">
          <ThemeIcon size="lg" variant="light" color="violet">
            <IconBell size={20} />
          </ThemeIcon>
          <div>
            <Title order={4}>Notifications</Title>
            <Text size="sm" c="dimmed">Configure how you receive notifications</Text>
          </div>
        </Group>

        <Stack gap="md">
          <Switch
            label="Email notifications"
            description="Receive email alerts when your questions or dashboards have new results"
            defaultChecked
          />
          <Switch
            label="Alert digests"
            description="Receive a daily summary of all triggered alerts"
          />
          <Switch
            label="Subscription notifications"
            description="Receive scheduled dashboard reports"
            defaultChecked
          />
        </Stack>
      </Card>
    </Stack>
  )
}

// Security Panel
function SecurityPanel() {
  const changePassword = useChangePassword()
  const [currentPassword, setCurrentPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')

  const handleChangePassword = async () => {
    if (newPassword !== confirmPassword) {
      notifications.show({
        title: 'Passwords do not match',
        message: 'Please make sure your new passwords match',
        color: 'red',
      })
      return
    }

    if (newPassword.length < 8) {
      notifications.show({
        title: 'Password too short',
        message: 'Password must be at least 8 characters',
        color: 'red',
      })
      return
    }

    try {
      await changePassword.mutateAsync({
        current_password: currentPassword,
        new_password: newPassword,
      })
      notifications.show({
        title: 'Password changed',
        message: 'Your password has been updated successfully',
        color: 'green',
      })
      setCurrentPassword('')
      setNewPassword('')
      setConfirmPassword('')
    } catch (err: any) {
      notifications.show({
        title: 'Password change failed',
        message: err.message || 'Failed to change password',
        color: 'red',
      })
    }
  }

  return (
    <Stack gap="lg">
      <Card withBorder radius="md" padding="lg">
        <Group mb="md">
          <ThemeIcon size="lg" variant="light" color="orange">
            <IconLock size={20} />
          </ThemeIcon>
          <div>
            <Title order={4}>Change Password</Title>
            <Text size="sm" c="dimmed">Update your account password</Text>
          </div>
        </Group>

        <Stack gap="md">
          <PasswordInput
            label="Current Password"
            placeholder="Enter current password"
            value={currentPassword}
            onChange={(e) => setCurrentPassword(e.target.value)}
            leftSection={<IconKey size={16} />}
          />
          <Group grow>
            <PasswordInput
              label="New Password"
              placeholder="Enter new password"
              value={newPassword}
              onChange={(e) => setNewPassword(e.target.value)}
              leftSection={<IconLock size={16} />}
            />
            <PasswordInput
              label="Confirm New Password"
              placeholder="Confirm new password"
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
              leftSection={<IconLock size={16} />}
              error={confirmPassword && newPassword !== confirmPassword ? 'Passwords do not match' : null}
            />
          </Group>

          <Group justify="flex-end" mt="md">
            <Button
              variant="light"
              onClick={handleChangePassword}
              loading={changePassword.isPending}
              disabled={!currentPassword || !newPassword || !confirmPassword}
            >
              Change Password
            </Button>
          </Group>
        </Stack>
      </Card>

      <Card withBorder radius="md" padding="lg">
        <Group mb="md">
          <ThemeIcon size="lg" variant="light" color="cyan">
            <IconShield size={20} />
          </ThemeIcon>
          <div>
            <Title order={4}>Security Options</Title>
            <Text size="sm" c="dimmed">Additional security settings</Text>
          </div>
        </Group>

        <Stack gap="md">
          <Switch
            label="Two-factor authentication"
            description="Add an extra layer of security to your account"
          />
          <Switch
            label="Login notifications"
            description="Get notified when your account is accessed from a new device"
            defaultChecked
          />
        </Stack>
      </Card>

      <Card withBorder radius="md" padding="lg">
        <Group mb="md">
          <ThemeIcon size="lg" variant="light" color="gray">
            <IconActivity size={20} />
          </ThemeIcon>
          <div>
            <Title order={4}>Active Sessions</Title>
            <Text size="sm" c="dimmed">Manage your active sessions</Text>
          </div>
        </Group>

        <Paper withBorder p="md" radius="md" bg="gray.0">
          <Group justify="space-between">
            <div>
              <Text size="sm" fw={500}>Current Session</Text>
              <Text size="xs" c="dimmed">This device • Last active just now</Text>
            </div>
            <Badge color="green" variant="light">Active</Badge>
          </Group>
        </Paper>

        <Button variant="subtle" color="red" mt="md" size="sm">
          Sign out of all other sessions
        </Button>
      </Card>
    </Stack>
  )
}

// Users Panel (Admin only)
function UsersPanel({ currentUser }: { currentUser: User }) {
  const { data: users, isLoading } = useUsers()
  const createUser = useCreateUser()
  const updateUser = useUpdateUser()
  const deleteUser = useDeleteUser()
  const resetPassword = useResetUserPassword()

  const [addModalOpened, { open: openAddModal, close: closeAddModal }] = useDisclosure(false)
  const [editingUser, setEditingUser] = useState<User | null>(null)

  // Form state
  const [formName, setFormName] = useState('')
  const [formEmail, setFormEmail] = useState('')
  const [formPassword, setFormPassword] = useState('')
  const [formRole, setFormRole] = useState<string>('user')

  const resetForm = () => {
    setFormName('')
    setFormEmail('')
    setFormPassword('')
    setFormRole('user')
    setEditingUser(null)
  }

  const handleCreateUser = async () => {
    if (!formName || !formEmail || !formPassword) return
    try {
      await createUser.mutateAsync({
        name: formName,
        email: formEmail,
        password: formPassword,
        role: formRole,
      })
      notifications.show({
        title: 'User created',
        message: `${formName} has been added`,
        color: 'green',
      })
      closeAddModal()
      resetForm()
    } catch (err: any) {
      notifications.show({
        title: 'Creation failed',
        message: err.message || 'Failed to create user',
        color: 'red',
      })
    }
  }

  const handleUpdateUser = async () => {
    if (!editingUser) return
    try {
      await updateUser.mutateAsync({
        id: editingUser.id,
        name: formName,
        email: formEmail,
        role: formRole,
      })
      notifications.show({
        title: 'User updated',
        message: 'Changes have been saved',
        color: 'green',
      })
      resetForm()
    } catch (err: any) {
      notifications.show({
        title: 'Update failed',
        message: err.message || 'Failed to update user',
        color: 'red',
      })
    }
  }

  const handleDeleteUser = async (user: User) => {
    if (user.id === currentUser.id) {
      notifications.show({
        title: 'Cannot delete',
        message: 'You cannot delete your own account',
        color: 'red',
      })
      return
    }
    if (!confirm(`Are you sure you want to delete ${user.name}?`)) return
    try {
      await deleteUser.mutateAsync(user.id)
      notifications.show({
        title: 'User deleted',
        message: `${user.name} has been removed`,
        color: 'green',
      })
    } catch (err: any) {
      notifications.show({
        title: 'Delete failed',
        message: err.message || 'Failed to delete user',
        color: 'red',
      })
    }
  }

  const handleResetPassword = async (user: User) => {
    if (!confirm(`Reset password for ${user.name}?`)) return
    try {
      const result = await resetPassword.mutateAsync(user.id)
      notifications.show({
        title: 'Password reset',
        message: `Temporary password: ${result.temporary_password}`,
        color: 'blue',
        autoClose: false,
      })
    } catch (err: any) {
      notifications.show({
        title: 'Reset failed',
        message: err.message || 'Failed to reset password',
        color: 'red',
      })
    }
  }

  const startEdit = (user: User) => {
    setEditingUser(user)
    setFormName(user.name)
    setFormEmail(user.email)
    setFormRole(user.role)
  }

  if (isLoading) {
    return <LoadingState message="Loading users..." />
  }

  return (
    <Stack gap="lg">
      <Card withBorder radius="md" padding="lg">
        <Group justify="space-between" mb="md">
          <Group>
            <ThemeIcon size="lg" variant="light" color="blue">
              <IconUsers size={20} />
            </ThemeIcon>
            <div>
              <Title order={4}>User Management</Title>
              <Text size="sm" c="dimmed">{users?.length || 0} users</Text>
            </div>
          </Group>
          <Button leftSection={<IconUserPlus size={16} />} onClick={openAddModal}>
            Add User
          </Button>
        </Group>

        <Table striped highlightOnHover>
          <Table.Thead>
            <Table.Tr>
              <Table.Th>User</Table.Th>
              <Table.Th>Email</Table.Th>
              <Table.Th>Role</Table.Th>
              <Table.Th>Last Login</Table.Th>
              <Table.Th style={{ width: 80 }}></Table.Th>
            </Table.Tr>
          </Table.Thead>
          <Table.Tbody>
            {users?.map((user) => (
              <Table.Tr key={user.id}>
                <Table.Td>
                  <Group gap="sm">
                    <Avatar size="sm" radius="xl" color="brand">
                      {user.name.charAt(0).toUpperCase()}
                    </Avatar>
                    <div>
                      <Text size="sm" fw={500}>{user.name}</Text>
                      {user.id === currentUser.id && (
                        <Text size="xs" c="dimmed">(you)</Text>
                      )}
                    </div>
                  </Group>
                </Table.Td>
                <Table.Td>
                  <Text size="sm">{user.email}</Text>
                </Table.Td>
                <Table.Td>
                  <Badge
                    size="sm"
                    variant="light"
                    color={user.role === 'admin' ? 'red' : user.role === 'user' ? 'blue' : 'gray'}
                  >
                    {user.role}
                  </Badge>
                </Table.Td>
                <Table.Td>
                  <Text size="sm" c="dimmed">
                    {user.last_login ? new Date(user.last_login).toLocaleDateString() : 'Never'}
                  </Text>
                </Table.Td>
                <Table.Td>
                  <Menu shadow="md" position="bottom-end">
                    <Menu.Target>
                      <ActionIcon variant="subtle" size="sm">
                        <IconDotsVertical size={16} />
                      </ActionIcon>
                    </Menu.Target>
                    <Menu.Dropdown>
                      <Menu.Item leftSection={<IconEdit size={14} />} onClick={() => startEdit(user)}>
                        Edit
                      </Menu.Item>
                      <Menu.Item leftSection={<IconKey size={14} />} onClick={() => handleResetPassword(user)}>
                        Reset Password
                      </Menu.Item>
                      {user.id !== currentUser.id && (
                        <>
                          <Menu.Divider />
                          <Menu.Item
                            leftSection={<IconTrash size={14} />}
                            color="red"
                            onClick={() => handleDeleteUser(user)}
                          >
                            Delete
                          </Menu.Item>
                        </>
                      )}
                    </Menu.Dropdown>
                  </Menu>
                </Table.Td>
              </Table.Tr>
            ))}
          </Table.Tbody>
        </Table>
      </Card>

      {/* Add User Modal */}
      <Modal opened={addModalOpened} onClose={() => { closeAddModal(); resetForm(); }} title="Add User">
        <Stack gap="md">
          <TextInput
            label="Name"
            placeholder="John Doe"
            value={formName}
            onChange={(e) => setFormName(e.target.value)}
            required
          />
          <TextInput
            label="Email"
            placeholder="john@example.com"
            value={formEmail}
            onChange={(e) => setFormEmail(e.target.value)}
            required
          />
          <PasswordInput
            label="Password"
            placeholder="Initial password"
            value={formPassword}
            onChange={(e) => setFormPassword(e.target.value)}
            required
          />
          <Select
            label="Role"
            data={[
              { value: 'viewer', label: 'Viewer - Can view dashboards and questions' },
              { value: 'user', label: 'User - Can create and edit content' },
              { value: 'admin', label: 'Admin - Full access' },
            ]}
            value={formRole}
            onChange={(v) => setFormRole(v || 'user')}
          />
          <Group justify="flex-end" mt="md">
            <Button variant="subtle" onClick={() => { closeAddModal(); resetForm(); }}>
              Cancel
            </Button>
            <Button
              onClick={handleCreateUser}
              loading={createUser.isPending}
              disabled={!formName || !formEmail || !formPassword}
            >
              Create User
            </Button>
          </Group>
        </Stack>
      </Modal>

      {/* Edit User Modal */}
      <Modal opened={!!editingUser} onClose={resetForm} title="Edit User">
        <Stack gap="md">
          <TextInput
            label="Name"
            value={formName}
            onChange={(e) => setFormName(e.target.value)}
            required
          />
          <TextInput
            label="Email"
            value={formEmail}
            onChange={(e) => setFormEmail(e.target.value)}
            required
          />
          <Select
            label="Role"
            data={[
              { value: 'viewer', label: 'Viewer' },
              { value: 'user', label: 'User' },
              { value: 'admin', label: 'Admin' },
            ]}
            value={formRole}
            onChange={(v) => setFormRole(v || 'user')}
            disabled={editingUser?.id === currentUser.id}
          />
          {editingUser?.id === currentUser.id && (
            <Text size="sm" c="dimmed">You cannot change your own role</Text>
          )}
          <Group justify="flex-end" mt="md">
            <Button variant="subtle" onClick={resetForm}>
              Cancel
            </Button>
            <Button onClick={handleUpdateUser} loading={updateUser.isPending}>
              Save Changes
            </Button>
          </Group>
        </Stack>
      </Modal>
    </Stack>
  )
}

// Application Panel (Admin only)
function ApplicationPanel({ settings }: { settings: SettingsType | undefined }) {
  const updateSettings = useUpdateSettings()
  const [siteName, setSiteName] = useState(settings?.site_name || 'BI')
  const [siteUrl, setSiteUrl] = useState(settings?.site_url || '')
  const [enablePublicSharing, setEnablePublicSharing] = useState(settings?.enable_public_sharing || false)
  const [enableEmbedding, setEnableEmbedding] = useState(settings?.enable_embedding || false)
  const [enableAlerts, setEnableAlerts] = useState(settings?.enable_alerts || false)

  const handleSave = async () => {
    try {
      await updateSettings.mutateAsync({
        site_name: siteName,
        site_url: siteUrl,
        enable_public_sharing: enablePublicSharing,
        enable_embedding: enableEmbedding,
        enable_alerts: enableAlerts,
      })
      notifications.show({
        title: 'Settings saved',
        message: 'Application settings have been updated',
        color: 'green',
      })
    } catch (err: any) {
      notifications.show({
        title: 'Save failed',
        message: err.message || 'Failed to save settings',
        color: 'red',
      })
    }
  }

  return (
    <Stack gap="lg">
      <Card withBorder radius="md" padding="lg">
        <Group mb="md">
          <ThemeIcon size="lg" variant="light" color="violet">
            <IconPalette size={20} />
          </ThemeIcon>
          <div>
            <Title order={4}>Branding</Title>
            <Text size="sm" c="dimmed">Customize the appearance of your BI instance</Text>
          </div>
        </Group>

        <Stack gap="md">
          <TextInput
            label="Site Name"
            description="Displayed in the browser tab and emails"
            placeholder="My BI"
            value={siteName}
            onChange={(e) => setSiteName(e.target.value)}
          />
          <TextInput
            label="Site URL"
            description="Public URL for shared links and emails"
            placeholder="https://bi.example.com"
            value={siteUrl}
            onChange={(e) => setSiteUrl(e.target.value)}
            leftSection={<IconWorld size={16} />}
          />
        </Stack>
      </Card>

      <Card withBorder radius="md" padding="lg">
        <Group mb="md">
          <ThemeIcon size="lg" variant="light" color="cyan">
            <IconWorld size={20} />
          </ThemeIcon>
          <div>
            <Title order={4}>Sharing & Embedding</Title>
            <Text size="sm" c="dimmed">Control how content can be shared</Text>
          </div>
        </Group>

        <Stack gap="md">
          <Switch
            label="Enable Public Sharing"
            description="Allow users to create public links to dashboards and questions"
            checked={enablePublicSharing}
            onChange={(e) => setEnablePublicSharing(e.currentTarget.checked)}
          />
          <Switch
            label="Enable Embedding"
            description="Allow dashboards and questions to be embedded in external websites"
            checked={enableEmbedding}
            onChange={(e) => setEnableEmbedding(e.currentTarget.checked)}
          />
        </Stack>
      </Card>

      <Card withBorder radius="md" padding="lg">
        <Group mb="md">
          <ThemeIcon size="lg" variant="light" color="orange">
            <IconBell size={20} />
          </ThemeIcon>
          <div>
            <Title order={4}>Alerts</Title>
            <Text size="sm" c="dimmed">Configure alerting features</Text>
          </div>
        </Group>

        <Stack gap="md">
          <Switch
            label="Enable Alerts"
            description="Allow users to set up alerts on questions"
            checked={enableAlerts}
            onChange={(e) => setEnableAlerts(e.currentTarget.checked)}
          />
        </Stack>
      </Card>

      <Group justify="flex-end">
        <Button
          leftSection={<IconDeviceFloppy size={16} />}
          onClick={handleSave}
          loading={updateSettings.isPending}
        >
          Save Settings
        </Button>
      </Group>
    </Stack>
  )
}

// Email Panel (Admin only)
function EmailPanel({ settings }: { settings: SettingsType | undefined }) {
  const [adminEmail, setAdminEmail] = useState(settings?.admin_email || '')
  const [smtpHost, setSmtpHost] = useState('')
  const [smtpPort, setSmtpPort] = useState('587')
  const [smtpUser, setSmtpUser] = useState('')
  const [smtpPass, setSmtpPass] = useState('')

  return (
    <Stack gap="lg">
      <Card withBorder radius="md" padding="lg">
        <Group mb="md">
          <ThemeIcon size="lg" variant="light" color="blue">
            <IconMail size={20} />
          </ThemeIcon>
          <div>
            <Title order={4}>Email Configuration</Title>
            <Text size="sm" c="dimmed">Configure SMTP settings for sending emails</Text>
          </div>
        </Group>

        <Stack gap="md">
          <TextInput
            label="Admin Email"
            description="Receives system notifications and error reports"
            placeholder="admin@example.com"
            value={adminEmail}
            onChange={(e) => setAdminEmail(e.target.value)}
            leftSection={<IconAt size={16} />}
          />

          <Divider label="SMTP Server" labelPosition="left" />

          <Group grow>
            <TextInput
              label="SMTP Host"
              placeholder="smtp.example.com"
              value={smtpHost}
              onChange={(e) => setSmtpHost(e.target.value)}
            />
            <TextInput
              label="SMTP Port"
              placeholder="587"
              value={smtpPort}
              onChange={(e) => setSmtpPort(e.target.value)}
              style={{ width: 100 }}
            />
          </Group>

          <Group grow>
            <TextInput
              label="SMTP Username"
              placeholder="username"
              value={smtpUser}
              onChange={(e) => setSmtpUser(e.target.value)}
            />
            <PasswordInput
              label="SMTP Password"
              placeholder="••••••••"
              value={smtpPass}
              onChange={(e) => setSmtpPass(e.target.value)}
            />
          </Group>

          <Group justify="space-between" mt="md">
            <Button variant="light" leftSection={<IconMail size={16} />}>
              Send Test Email
            </Button>
            <Button leftSection={<IconDeviceFloppy size={16} />}>
              Save Email Settings
            </Button>
          </Group>
        </Stack>
      </Card>
    </Stack>
  )
}

// Activity Panel (Admin only)
function ActivityPanel() {
  const { data: activityData, isLoading, refetch } = useActivityLog({ limit: 50 })

  if (isLoading) {
    return <LoadingState message="Loading activity..." />
  }

  return (
    <Stack gap="lg">
      <Card withBorder radius="md" padding="lg">
        <Group justify="space-between" mb="md">
          <Group>
            <ThemeIcon size="lg" variant="light" color="gray">
              <IconActivity size={20} />
            </ThemeIcon>
            <div>
              <Title order={4}>Activity Log</Title>
              <Text size="sm" c="dimmed">Recent actions in the system</Text>
            </div>
          </Group>
          <Button
            variant="light"
            size="sm"
            leftSection={<IconRefresh size={14} />}
            onClick={() => refetch()}
          >
            Refresh
          </Button>
        </Group>

        {activityData?.activities && activityData.activities.length > 0 ? (
          <Table striped highlightOnHover>
            <Table.Thead>
              <Table.Tr>
                <Table.Th>User</Table.Th>
                <Table.Th>Action</Table.Th>
                <Table.Th>Description</Table.Th>
                <Table.Th>Time</Table.Th>
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>
              {activityData.activities.map((activity) => (
                <Table.Tr key={activity.id}>
                  <Table.Td>
                    <Group gap="xs">
                      <Avatar size="sm" radius="xl" color="brand">
                        {activity.user_name?.charAt(0).toUpperCase() || '?'}
                      </Avatar>
                      <Text size="sm">{activity.user_name}</Text>
                    </Group>
                  </Table.Td>
                  <Table.Td>
                    <Badge size="sm" variant="light">
                      {activity.type}
                    </Badge>
                  </Table.Td>
                  <Table.Td>
                    <Text size="sm" lineClamp={1}>{activity.description}</Text>
                  </Table.Td>
                  <Table.Td>
                    <Text size="sm" c="dimmed">
                      {new Date(activity.created_at).toLocaleString()}
                    </Text>
                  </Table.Td>
                </Table.Tr>
              ))}
            </Table.Tbody>
          </Table>
        ) : (
          <Paper withBorder p="xl" ta="center" bg="gray.0">
            <Text c="dimmed">No activity recorded yet</Text>
          </Paper>
        )}
      </Card>
    </Stack>
  )
}
