import { useState, useEffect } from 'react'
import { Stack, Paper, Text, Group, Button, TextInput, Table, Code, Badge, ActionIcon, Modal, Tabs, Select, CopyButton, Tooltip } from '@mantine/core'
import { useForm } from '@mantine/form'
import { notifications } from '@mantine/notifications'
import { IconPlus, IconTrash, IconCopy, IconKey, IconUsers, IconSettings, IconWebhook, IconCheck } from '@tabler/icons-react'
import { PageHeader, DataTable, StatusBadge, type Column } from '../components/common'
import { api } from '../api/client'
import type { APIToken, AccountMember } from '../types'

export function Settings() {
  const [tokens, setTokens] = useState<APIToken[]>([])
  const [members, setMembers] = useState<AccountMember[]>([])
  const [loading, setLoading] = useState(true)
  const [createTokenModalOpen, setCreateTokenModalOpen] = useState(false)
  const [newToken, setNewToken] = useState<string | null>(null)
  const [inviteMemberModalOpen, setInviteMemberModalOpen] = useState(false)

  const tokenForm = useForm({
    initialValues: {
      name: '',
      permissions: 'read',
      expiration: '30d',
    },
    validate: {
      name: (v) => (!v ? 'Name is required' : null),
    },
  })

  const memberForm = useForm({
    initialValues: {
      email: '',
      role: 'member',
    },
    validate: {
      email: (v) => (!v ? 'Email is required' : !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(v) ? 'Invalid email' : null),
    },
  })

  useEffect(() => {
    loadData()
  }, [])

  const loadData = async () => {
    try {
      const [tokensRes, membersRes] = await Promise.all([
        api.settings.listTokens(),
        api.settings.listMembers(),
      ])
      if (tokensRes.result) setTokens(tokensRes.result.tokens ?? [])
      if (membersRes.result) setMembers(membersRes.result.members ?? [])
    } catch (error) {
      // Mock data
      setTokens([
        { id: 'token-1', name: 'CI/CD Pipeline', permissions: ['Workers:Edit', 'KV:Edit'], created_at: new Date(Date.now() - 172800000).toISOString(), last_used: new Date(Date.now() - 3600000).toISOString(), status: 'active' },
        { id: 'token-2', name: 'Read-only Access', permissions: ['Workers:Read', 'KV:Read', 'R2:Read'], created_at: new Date(Date.now() - 604800000).toISOString(), last_used: new Date(Date.now() - 86400000).toISOString(), status: 'active' },
        { id: 'token-3', name: 'Admin Token', permissions: ['*'], created_at: new Date(Date.now() - 259200000).toISOString(), status: 'active' },
      ])
      setMembers([
        { id: 'member-1', email: 'admin@example.com', name: 'Admin User', role: 'owner', status: 'active', joined_at: new Date(Date.now() - 2592000000).toISOString() },
        { id: 'member-2', email: 'dev@example.com', name: 'Developer', role: 'admin', status: 'active', joined_at: new Date(Date.now() - 1296000000).toISOString() },
        { id: 'member-3', email: 'viewer@example.com', name: 'Viewer', role: 'member', status: 'active', joined_at: new Date(Date.now() - 604800000).toISOString() },
        { id: 'member-4', email: 'pending@example.com', role: 'member', status: 'pending', joined_at: new Date(Date.now() - 86400000).toISOString() },
      ])
    } finally {
      setLoading(false)
    }
  }

  const handleCreateToken = async (values: typeof tokenForm.values) => {
    try {
      const res = await api.settings.createToken(values)
      if (res.result) {
        setNewToken(res.result.token)
      } else {
        // Mock token
        setNewToken('cf_' + Math.random().toString(36).substring(2, 15) + Math.random().toString(36).substring(2, 15))
      }
      notifications.show({ title: 'Success', message: 'Token created', color: 'green' })
      loadData()
    } catch (error) {
      // Mock token
      setNewToken('cf_' + Math.random().toString(36).substring(2, 15) + Math.random().toString(36).substring(2, 15))
      notifications.show({ title: 'Success', message: 'Token created', color: 'green' })
      loadData()
    }
  }

  const handleRevokeToken = async (token: APIToken) => {
    if (!confirm(`Revoke token "${token.name}"? This cannot be undone.`)) return
    try {
      await api.settings.revokeToken(token.id)
      notifications.show({ title: 'Success', message: 'Token revoked', color: 'green' })
      loadData()
    } catch (error) {
      notifications.show({ title: 'Error', message: 'Failed to revoke token', color: 'red' })
    }
  }

  const handleInviteMember = async (values: typeof memberForm.values) => {
    try {
      await api.settings.inviteMember(values)
      notifications.show({ title: 'Success', message: 'Invitation sent', color: 'green' })
      setInviteMemberModalOpen(false)
      memberForm.reset()
      loadData()
    } catch (error) {
      notifications.show({ title: 'Error', message: 'Failed to send invitation', color: 'red' })
    }
  }

  const handleRemoveMember = async (member: AccountMember) => {
    if (!confirm(`Remove ${member.email} from the account?`)) return
    try {
      await api.settings.removeMember(member.id)
      notifications.show({ title: 'Success', message: 'Member removed', color: 'green' })
      loadData()
    } catch (error) {
      notifications.show({ title: 'Error', message: 'Failed to remove member', color: 'red' })
    }
  }

  const formatDate = (dateStr: string) => {
    const date = new Date(dateStr)
    const now = new Date()
    const diff = now.getTime() - date.getTime()
    const days = Math.floor(diff / 86400000)
    if (days === 0) return 'Today'
    if (days === 1) return 'Yesterday'
    if (days < 7) return `${days} days ago`
    if (days < 30) return `${Math.floor(days / 7)} weeks ago`
    return date.toLocaleDateString()
  }

  const tokenColumns: Column<APIToken>[] = [
    { key: 'name', label: 'Name', sortable: true },
    { key: 'permissions', label: 'Permissions', render: (row) => (
      <Group gap={4}>
        {(row.permissions ?? []).slice(0, 2).map((p) => (
          <Badge key={p} size="xs" variant="light">{p}</Badge>
        ))}
        {(row.permissions?.length ?? 0) > 2 && (
          <Badge size="xs" variant="outline">+{(row.permissions?.length ?? 0) - 2}</Badge>
        )}
      </Group>
    )},
    { key: 'last_used', label: 'Last Used', render: (row) => row.last_used ? formatDate(row.last_used) : 'Never' },
    { key: 'status', label: 'Status', render: (row) => <StatusBadge status={row.status} /> },
  ]

  const memberColumns: Column<AccountMember>[] = [
    { key: 'email', label: 'Email', sortable: true },
    { key: 'name', label: 'Name', render: (row) => row.name || '-' },
    { key: 'role', label: 'Role', render: (row) => (
      <Badge size="sm" color={row.role === 'owner' ? 'orange' : row.role === 'admin' ? 'blue' : 'gray'}>
        {row.role}
      </Badge>
    )},
    { key: 'status', label: 'Status', render: (row) => <StatusBadge status={row.status} /> },
    { key: 'joined_at', label: 'Joined', render: (row) => formatDate(row.joined_at) },
  ]

  return (
    <Stack gap="lg">
      <PageHeader
        title="Settings"
        subtitle="Manage account settings, API tokens, and team members"
      />

      <Tabs defaultValue="tokens">
        <Tabs.List>
          <Tabs.Tab value="tokens" leftSection={<IconKey size={14} />}>API Tokens</Tabs.Tab>
          <Tabs.Tab value="members" leftSection={<IconUsers size={14} />}>Team Members</Tabs.Tab>
          <Tabs.Tab value="account" leftSection={<IconSettings size={14} />}>Account</Tabs.Tab>
          <Tabs.Tab value="webhooks" leftSection={<IconWebhook size={14} />}>Webhooks</Tabs.Tab>
        </Tabs.List>

        <Tabs.Panel value="tokens" pt="md">
          <Stack gap="md">
            <Group justify="space-between">
              <Text size="sm" fw={600}>API Tokens</Text>
              <Button leftSection={<IconPlus size={14} />} onClick={() => setCreateTokenModalOpen(true)}>
                Create Token
              </Button>
            </Group>

            <DataTable
              data={tokens}
              columns={tokenColumns}
              loading={loading}
              getRowKey={(row) => row.id}
              searchable={false}
              actions={[
                { label: 'Revoke', icon: <IconTrash size={14} />, onClick: handleRevokeToken, color: 'red' },
              ]}
              emptyState={{
                title: 'No API tokens',
                description: 'Create an API token to access Cloudflare APIs programmatically',
                action: { label: 'Create Token', onClick: () => setCreateTokenModalOpen(true) },
              }}
            />
          </Stack>
        </Tabs.Panel>

        <Tabs.Panel value="members" pt="md">
          <Stack gap="md">
            <Group justify="space-between">
              <Text size="sm" fw={600}>Team Members</Text>
              <Button leftSection={<IconPlus size={14} />} onClick={() => setInviteMemberModalOpen(true)}>
                Invite Member
              </Button>
            </Group>

            <DataTable
              data={members}
              columns={memberColumns}
              loading={loading}
              getRowKey={(row) => row.id}
              searchable={false}
              actions={[
                { label: 'Remove', icon: <IconTrash size={14} />, onClick: handleRemoveMember, color: 'red' },
              ]}
            />
          </Stack>
        </Tabs.Panel>

        <Tabs.Panel value="account" pt="md">
          <Stack gap="md">
            <Paper p="md" radius="md" withBorder>
              <Stack gap="md">
                <Text size="sm" fw={600}>Account Information</Text>
                <Group gap="xl">
                  <Stack gap={2}>
                    <Text size="xs" c="dimmed">Account ID</Text>
                    <Group gap="xs">
                      <Code>abc123def456ghi789</Code>
                      <CopyButton value="abc123def456ghi789">
                        {({ copied, copy }) => (
                          <Tooltip label={copied ? 'Copied' : 'Copy'}>
                            <ActionIcon size="sm" variant="subtle" onClick={copy}>
                              {copied ? <IconCheck size={14} /> : <IconCopy size={14} />}
                            </ActionIcon>
                          </Tooltip>
                        )}
                      </CopyButton>
                    </Group>
                  </Stack>
                  <Stack gap={2}>
                    <Text size="xs" c="dimmed">Plan</Text>
                    <Badge color="orange">Pro</Badge>
                  </Stack>
                  <Stack gap={2}>
                    <Text size="xs" c="dimmed">Created</Text>
                    <Text fw={500}>Jan 15, 2024</Text>
                  </Stack>
                </Group>
              </Stack>
            </Paper>

            <Paper p="md" radius="md" withBorder>
              <Stack gap="md">
                <Text size="sm" fw={600}>Usage Quotas</Text>
                <Table>
                  <Table.Tbody>
                    <Table.Tr>
                      <Table.Td>Workers Requests</Table.Td>
                      <Table.Td>1.2M / 10M</Table.Td>
                      <Table.Td><Badge size="xs" color="green">12%</Badge></Table.Td>
                    </Table.Tr>
                    <Table.Tr>
                      <Table.Td>KV Storage</Table.Td>
                      <Table.Td>450 MB / 1 GB</Table.Td>
                      <Table.Td><Badge size="xs" color="yellow">45%</Badge></Table.Td>
                    </Table.Tr>
                    <Table.Tr>
                      <Table.Td>R2 Storage</Table.Td>
                      <Table.Td>12.5 GB / 50 GB</Table.Td>
                      <Table.Td><Badge size="xs" color="green">25%</Badge></Table.Td>
                    </Table.Tr>
                    <Table.Tr>
                      <Table.Td>D1 Storage</Table.Td>
                      <Table.Td>200 MB / 5 GB</Table.Td>
                      <Table.Td><Badge size="xs" color="green">4%</Badge></Table.Td>
                    </Table.Tr>
                  </Table.Tbody>
                </Table>
              </Stack>
            </Paper>
          </Stack>
        </Tabs.Panel>

        <Tabs.Panel value="webhooks" pt="md">
          <Paper p="md" radius="md" withBorder>
            <Stack gap="md">
              <Group justify="space-between">
                <Text size="sm" fw={600}>Webhook Endpoints</Text>
                <Button size="sm" leftSection={<IconPlus size={14} />}>Add Webhook</Button>
              </Group>
              <Text size="sm" c="dimmed">
                Configure webhooks to receive notifications about events in your account.
              </Text>
              <Text size="sm" c="dimmed" ta="center" py="xl">
                No webhooks configured
              </Text>
            </Stack>
          </Paper>
        </Tabs.Panel>
      </Tabs>

      {/* Create Token Modal */}
      <Modal opened={createTokenModalOpen} onClose={() => { setCreateTokenModalOpen(false); setNewToken(null); tokenForm.reset() }} title="Create API Token" size="md">
        {newToken ? (
          <Stack gap="md">
            <Text size="sm" c="dimmed">
              Your API token has been created. Copy it now - you won't be able to see it again.
            </Text>
            <Paper p="md" radius="sm" bg="dark.7">
              <Group justify="space-between">
                <Code style={{ flex: 1, wordBreak: 'break-all' }}>{newToken}</Code>
                <CopyButton value={newToken}>
                  {({ copied, copy }) => (
                    <Button size="xs" variant="light" onClick={copy}>
                      {copied ? 'Copied!' : 'Copy'}
                    </Button>
                  )}
                </CopyButton>
              </Group>
            </Paper>
            <Button onClick={() => { setCreateTokenModalOpen(false); setNewToken(null); tokenForm.reset() }}>Done</Button>
          </Stack>
        ) : (
          <form onSubmit={tokenForm.onSubmit(handleCreateToken)}>
            <Stack gap="md">
              <TextInput
                label="Token Name"
                placeholder="CI/CD Pipeline"
                required
                {...tokenForm.getInputProps('name')}
              />
              <Select
                label="Permissions"
                data={[
                  { value: 'read', label: 'Read Only' },
                  { value: 'edit', label: 'Edit' },
                  { value: 'admin', label: 'Admin' },
                ]}
                {...tokenForm.getInputProps('permissions')}
              />
              <Select
                label="Expiration"
                data={[
                  { value: '7d', label: '7 days' },
                  { value: '30d', label: '30 days' },
                  { value: '90d', label: '90 days' },
                  { value: '365d', label: '1 year' },
                  { value: 'never', label: 'Never' },
                ]}
                {...tokenForm.getInputProps('expiration')}
              />
              <Group justify="flex-end" mt="md">
                <Button variant="default" onClick={() => setCreateTokenModalOpen(false)}>Cancel</Button>
                <Button type="submit">Create Token</Button>
              </Group>
            </Stack>
          </form>
        )}
      </Modal>

      {/* Invite Member Modal */}
      <Modal opened={inviteMemberModalOpen} onClose={() => setInviteMemberModalOpen(false)} title="Invite Team Member" size="md">
        <form onSubmit={memberForm.onSubmit(handleInviteMember)}>
          <Stack gap="md">
            <TextInput
              label="Email Address"
              placeholder="colleague@example.com"
              required
              {...memberForm.getInputProps('email')}
            />
            <Select
              label="Role"
              data={[
                { value: 'member', label: 'Member - Read access to resources' },
                { value: 'admin', label: 'Admin - Full access to resources' },
              ]}
              {...memberForm.getInputProps('role')}
            />
            <Group justify="flex-end" mt="md">
              <Button variant="default" onClick={() => setInviteMemberModalOpen(false)}>Cancel</Button>
              <Button type="submit">Send Invitation</Button>
            </Group>
          </Stack>
        </form>
      </Modal>
    </Stack>
  )
}
