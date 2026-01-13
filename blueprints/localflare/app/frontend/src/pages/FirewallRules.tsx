import { Container, Title, Text, Card, Table, Button, Group, Badge, TextInput, Modal, Stack, Select, Switch, ActionIcon, Menu, Textarea } from '@mantine/core'
import { IconPlus, IconSearch, IconDotsVertical, IconTrash, IconEdit, IconShield } from '@tabler/icons-react'
import { useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { notifications } from '@mantine/notifications'

interface FirewallRule {
  id: string
  name: string
  expression: string
  action: string
  priority: number
  enabled: boolean
}

const actions = ['block', 'challenge', 'js_challenge', 'managed_challenge', 'allow', 'log', 'bypass']

export function FirewallRules() {
  const { id: zoneId } = useParams<{ id: string }>()
  const [rules, setRules] = useState<FirewallRule[]>([])
  const [, setLoading] = useState(true)
  const [modalOpen, setModalOpen] = useState(false)
  const [editRule, setEditRule] = useState<FirewallRule | null>(null)
  const [search, setSearch] = useState('')
  const [form, setForm] = useState({
    name: '',
    expression: '',
    action: 'block',
    priority: 1,
    enabled: true,
  })

  useEffect(() => {
    fetchRules()
  }, [zoneId])

  const fetchRules = async () => {
    try {
      const res = await fetch(`/api/zones/${zoneId}/firewall/rules`)
      const data = await res.json()
      setRules(data.result || [])
    } catch (error) {
      console.error('Failed to fetch rules:', error)
    } finally {
      setLoading(false)
    }
  }

  const saveRule = async () => {
    const method = editRule ? 'PUT' : 'POST'
    const url = editRule
      ? `/api/zones/${zoneId}/firewall/rules/${editRule.id}`
      : `/api/zones/${zoneId}/firewall/rules`

    try {
      const res = await fetch(url, {
        method,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(form),
      })
      if (res.ok) {
        notifications.show({ title: 'Success', message: 'Rule saved', color: 'green' })
        setModalOpen(false)
        setEditRule(null)
        setForm({ name: '', expression: '', action: 'block', priority: 1, enabled: true })
        fetchRules()
      }
    } catch (error) {
      notifications.show({ title: 'Error', message: 'Failed to save rule', color: 'red' })
    }
  }

  const deleteRule = async (ruleId: string) => {
    try {
      const res = await fetch(`/api/zones/${zoneId}/firewall/rules/${ruleId}`, { method: 'DELETE' })
      if (res.ok) {
        notifications.show({ title: 'Success', message: 'Rule deleted', color: 'green' })
        fetchRules()
      }
    } catch (error) {
      notifications.show({ title: 'Error', message: 'Failed to delete rule', color: 'red' })
    }
  }

  const toggleRule = async (rule: FirewallRule) => {
    try {
      await fetch(`/api/zones/${zoneId}/firewall/rules/${rule.id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ ...rule, enabled: !rule.enabled }),
      })
      fetchRules()
    } catch (error) {
      console.error('Failed to toggle rule:', error)
    }
  }

  const openEditModal = (rule: FirewallRule) => {
    setEditRule(rule)
    setForm({
      name: rule.name,
      expression: rule.expression,
      action: rule.action,
      priority: rule.priority,
      enabled: rule.enabled,
    })
    setModalOpen(true)
  }

  const filteredRules = rules.filter(r =>
    r.name.toLowerCase().includes(search.toLowerCase()) ||
    r.expression.toLowerCase().includes(search.toLowerCase())
  )

  const getActionColor = (action: string) => {
    switch (action) {
      case 'block': return 'red'
      case 'challenge': case 'js_challenge': case 'managed_challenge': return 'yellow'
      case 'allow': return 'green'
      case 'log': return 'blue'
      case 'bypass': return 'gray'
      default: return 'gray'
    }
  }

  return (
    <Container size="xl" py="xl">
      <Group justify="space-between" mb="xl">
        <div>
          <Title order={1}>Firewall Rules</Title>
          <Text c="dimmed" mt="xs">Custom WAF rules for your domain</Text>
        </div>
        <Button leftSection={<IconPlus size={16} />} onClick={() => {
          setEditRule(null)
          setForm({ name: '', expression: '', action: 'block', priority: 1, enabled: true })
          setModalOpen(true)
        }}>
          Create Rule
        </Button>
      </Group>

      <Card withBorder shadow="sm" radius="md">
        <Group mb="md">
          <TextInput
            placeholder="Search rules..."
            leftSection={<IconSearch size={16} />}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            style={{ flex: 1 }}
          />
        </Group>

        <Table striped highlightOnHover>
          <Table.Thead>
            <Table.Tr>
              <Table.Th>Priority</Table.Th>
              <Table.Th>Name</Table.Th>
              <Table.Th>Expression</Table.Th>
              <Table.Th>Action</Table.Th>
              <Table.Th>Status</Table.Th>
              <Table.Th></Table.Th>
            </Table.Tr>
          </Table.Thead>
          <Table.Tbody>
            {filteredRules.map((rule) => (
              <Table.Tr key={rule.id}>
                <Table.Td>
                  <Badge variant="outline">{rule.priority}</Badge>
                </Table.Td>
                <Table.Td>
                  <Group gap="xs">
                    <IconShield size={16} />
                    <Text fw={500}>{rule.name}</Text>
                  </Group>
                </Table.Td>
                <Table.Td>
                  <Text ff="monospace" size="sm" lineClamp={1}>{rule.expression}</Text>
                </Table.Td>
                <Table.Td>
                  <Badge color={getActionColor(rule.action)}>{rule.action}</Badge>
                </Table.Td>
                <Table.Td>
                  <Switch
                    checked={rule.enabled}
                    onChange={() => toggleRule(rule)}
                    size="sm"
                  />
                </Table.Td>
                <Table.Td>
                  <Menu position="bottom-end">
                    <Menu.Target>
                      <ActionIcon variant="subtle">
                        <IconDotsVertical size={16} />
                      </ActionIcon>
                    </Menu.Target>
                    <Menu.Dropdown>
                      <Menu.Item leftSection={<IconEdit size={14} />} onClick={() => openEditModal(rule)}>
                        Edit
                      </Menu.Item>
                      <Menu.Divider />
                      <Menu.Item leftSection={<IconTrash size={14} />} color="red" onClick={() => deleteRule(rule.id)}>
                        Delete
                      </Menu.Item>
                    </Menu.Dropdown>
                  </Menu>
                </Table.Td>
              </Table.Tr>
            ))}
          </Table.Tbody>
        </Table>
      </Card>

      <Modal opened={modalOpen} onClose={() => setModalOpen(false)} title={editRule ? 'Edit Rule' : 'Create Rule'} size="lg">
        <Stack>
          <TextInput
            label="Rule Name"
            placeholder="Block bad bots"
            value={form.name}
            onChange={(e) => setForm({ ...form, name: e.target.value })}
          />
          <Textarea
            label="Expression"
            placeholder='(cf.client.bot) or (http.user_agent contains "BadBot")'
            description="Use Cloudflare-style expression language"
            minRows={3}
            value={form.expression}
            onChange={(e) => setForm({ ...form, expression: e.target.value })}
          />
          <Select
            label="Action"
            data={actions.map(a => ({ value: a, label: a.replace(/_/g, ' ').toUpperCase() }))}
            value={form.action}
            onChange={(v) => setForm({ ...form, action: v || 'block' })}
          />
          <TextInput
            label="Priority"
            type="number"
            value={form.priority}
            onChange={(e) => setForm({ ...form, priority: parseInt(e.target.value) || 1 })}
          />
          <Switch
            label="Enabled"
            checked={form.enabled}
            onChange={(e) => setForm({ ...form, enabled: e.target.checked })}
          />
          <Group justify="flex-end" mt="md">
            <Button variant="subtle" onClick={() => setModalOpen(false)}>Cancel</Button>
            <Button onClick={saveRule}>Save</Button>
          </Group>
        </Stack>
      </Modal>
    </Container>
  )
}
