import { useState } from 'react'
import {
  Card, Group, Stack, Button, TextInput, Table, Text,
  Badge, ActionIcon, Menu, Modal, Select, Tabs, Avatar
} from '@mantine/core'
import { useDisclosure } from '@mantine/hooks'
import { notifications } from '@mantine/notifications'
import {
  IconSearch, IconDots, IconEdit, IconTrash, IconMail,
  IconUserPlus, IconUsers
} from '@tabler/icons-react'
import { useUsers, useCurrentUser } from '../../api/hooks'
import { api } from '../../api/client'
import { PageHeader, PageContainer, EmptyState } from '../../components/ui'

interface InviteData {
  email: string
  name: string
  role: 'admin' | 'user' | 'viewer'
}

export default function People() {
  const { data: users, refetch } = useUsers()
  const { data: currentUser } = useCurrentUser()
  const [search, setSearch] = useState('')
  const [inviteModalOpened, { open: openInviteModal, close: closeInviteModal }] = useDisclosure(false)
  const [inviteData, setInviteData] = useState<InviteData>({ email: '', name: '', role: 'user' })
  const [activeTab, setActiveTab] = useState<string | null>('users')

  const filteredUsers = users?.filter(user =>
    user.name.toLowerCase().includes(search.toLowerCase()) ||
    user.email.toLowerCase().includes(search.toLowerCase())
  ) || []

  const handleInvite = async () => {
    try {
      await api.post('/users', {
        email: inviteData.email,
        name: inviteData.name,
        role: inviteData.role,
        password_hash: '', // In production, would send invite email
      })
      notifications.show({
        title: 'User created',
        message: `${inviteData.email} has been added`,
        color: 'green',
      })
      closeInviteModal()
      setInviteData({ email: '', name: '', role: 'user' })
      refetch()
    } catch (error) {
      notifications.show({
        title: 'Error',
        message: 'Failed to create user',
        color: 'red',
      })
    }
  }

  const handleDelete = async (userId: string) => {
    if (userId === currentUser?.id) {
      notifications.show({
        title: 'Error',
        message: 'You cannot delete yourself',
        color: 'red',
      })
      return
    }
    try {
      await api.delete(`/users/${userId}`)
      notifications.show({
        title: 'User deleted',
        message: 'User has been removed',
        color: 'green',
      })
      refetch()
    } catch (error) {
      notifications.show({
        title: 'Error',
        message: 'Failed to delete user',
        color: 'red',
      })
    }
  }

  const getRoleBadgeColor = (role: string) => {
    switch (role) {
      case 'admin': return 'red'
      case 'user': return 'blue'
      case 'viewer': return 'gray'
      default: return 'gray'
    }
  }

  return (
    <PageContainer>
      <PageHeader
        title="People"
        subtitle="Manage users and permissions"
        actions={
          <Button leftSection={<IconUserPlus size={16} />} onClick={openInviteModal}>
            Invite User
          </Button>
        }
      />

      {/* Tabs */}
      <Tabs value={activeTab} onChange={setActiveTab} mb="lg">
        <Tabs.List>
          <Tabs.Tab value="users" leftSection={<IconUsers size={14} />}>
            Users ({users?.length || 0})
          </Tabs.Tab>
          <Tabs.Tab value="groups" leftSection={<IconUsers size={14} />}>
            Groups
          </Tabs.Tab>
        </Tabs.List>
      </Tabs>

      {activeTab === 'users' && (
        <Card withBorder radius="md">
          {/* Search */}
          <Group mb="md">
            <TextInput
              placeholder="Search users..."
              leftSection={<IconSearch size={16} />}
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              style={{ flex: 1 }}
            />
          </Group>

          {/* Users Table */}
          <Table striped highlightOnHover>
            <Table.Thead>
              <Table.Tr>
                <Table.Th>User</Table.Th>
                <Table.Th>Email</Table.Th>
                <Table.Th>Role</Table.Th>
                <Table.Th>Last Login</Table.Th>
                <Table.Th w={50}></Table.Th>
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>
              {filteredUsers.map(user => (
                <Table.Tr key={user.id}>
                  <Table.Td>
                    <Group gap="sm">
                      <Avatar size="sm" color="brand" radius="xl">
                        {user.name?.charAt(0).toUpperCase() || 'U'}
                      </Avatar>
                      <Text fw={500}>{user.name}</Text>
                    </Group>
                  </Table.Td>
                  <Table.Td>{user.email}</Table.Td>
                  <Table.Td>
                    <Badge color={getRoleBadgeColor(user.role)} variant="light">
                      {user.role}
                    </Badge>
                  </Table.Td>
                  <Table.Td>
                    {user.last_login
                      ? new Date(user.last_login).toLocaleDateString()
                      : 'Never'}
                  </Table.Td>
                  <Table.Td>
                    <Menu position="bottom-end">
                      <Menu.Target>
                        <ActionIcon variant="subtle" color="gray">
                          <IconDots size={16} />
                        </ActionIcon>
                      </Menu.Target>
                      <Menu.Dropdown>
                        <Menu.Item leftSection={<IconEdit size={14} />}>
                          Edit
                        </Menu.Item>
                        <Menu.Item leftSection={<IconMail size={14} />}>
                          Reset Password
                        </Menu.Item>
                        <Menu.Divider />
                        <Menu.Item
                          leftSection={<IconTrash size={14} />}
                          color="red"
                          onClick={() => handleDelete(user.id)}
                          disabled={user.id === currentUser?.id}
                        >
                          Delete
                        </Menu.Item>
                      </Menu.Dropdown>
                    </Menu>
                  </Table.Td>
                </Table.Tr>
              ))}
              {filteredUsers.length === 0 && (
                <Table.Tr>
                  <Table.Td colSpan={5}>
                    <Text ta="center" c="dimmed" py="xl">
                      {search ? 'No users found matching your search' : 'No users yet'}
                    </Text>
                  </Table.Td>
                </Table.Tr>
              )}
            </Table.Tbody>
          </Table>
        </Card>
      )}

      {activeTab === 'groups' && (
        <EmptyState
          icon={<IconUsers size={32} strokeWidth={1.5} />}
          iconColor="var(--color-info)"
          title="Groups coming soon"
          description="Groups help you manage permissions for multiple users at once."
          size="md"
        />
      )}

      {/* Invite Modal */}
      <Modal opened={inviteModalOpened} onClose={closeInviteModal} title="Invite User">
        <Stack>
          <TextInput
            label="Name"
            placeholder="Enter name"
            value={inviteData.name}
            onChange={(e) => setInviteData({ ...inviteData, name: e.target.value })}
            required
          />
          <TextInput
            label="Email"
            placeholder="user@example.com"
            value={inviteData.email}
            onChange={(e) => setInviteData({ ...inviteData, email: e.target.value })}
            required
          />
          <Select
            label="Role"
            data={[
              { value: 'admin', label: 'Admin - Full access' },
              { value: 'user', label: 'User - Can create and edit' },
              { value: 'viewer', label: 'Viewer - Read only' },
            ]}
            value={inviteData.role}
            onChange={(value) => setInviteData({ ...inviteData, role: value as any })}
          />
          <Group justify="flex-end" mt="md">
            <Button variant="light" onClick={closeInviteModal}>Cancel</Button>
            <Button onClick={handleInvite} disabled={!inviteData.email || !inviteData.name}>
              Send Invite
            </Button>
          </Group>
        </Stack>
      </Modal>
    </PageContainer>
  )
}
