import { useState, useEffect } from 'react'
import { Button, Modal, TextInput, Stack, Group, Badge, Text as MantineText } from '@mantine/core'
import { useNavigate } from 'react-router-dom'
import { useForm } from '@mantine/form'
import { notifications } from '@mantine/notifications'
import { IconPlus, IconEye, IconTrash, IconExternalLink, IconGitBranch, IconRocket } from '@tabler/icons-react'
import { PageHeader, DataTable, StatusBadge, type Column } from '../components/common'
import { api } from '../api/client'
import type { PagesProject } from '../types'

export function Pages() {
  const navigate = useNavigate()
  const [projects, setProjects] = useState<PagesProject[]>([])
  const [loading, setLoading] = useState(true)
  const [createModalOpen, setCreateModalOpen] = useState(false)

  const form = useForm({
    initialValues: {
      name: '',
      production_branch: 'main',
    },
    validate: {
      name: (v) => (!v ? 'Name is required' : !/^[a-z0-9-]+$/.test(v) ? 'Use lowercase letters, numbers, and hyphens only' : null),
    },
  })

  useEffect(() => {
    loadProjects()
  }, [])

  const loadProjects = async () => {
    try {
      const res = await api.pages.listProjects()
      if (res.result) setProjects(res.result.projects ?? [])
    } catch (error) {
      console.error('Failed to load Pages projects:', error)
      setProjects([
        {
          name: 'my-blog',
          subdomain: 'my-blog',
          created_at: new Date(Date.now() - 172800000).toISOString(),
          production_branch: 'main',
          latest_deployment: {
            id: 'deploy-1',
            url: 'https://my-blog.pages.dev',
            environment: 'production',
            deployment_trigger: { type: 'push', metadata: { branch: 'main', commit_hash: 'abc123' } },
            created_at: new Date(Date.now() - 3600000).toISOString(),
            status: 'success',
          },
          domains: ['blog.example.com'],
        },
        {
          name: 'docs-site',
          subdomain: 'docs-site',
          created_at: new Date(Date.now() - 604800000).toISOString(),
          production_branch: 'main',
          latest_deployment: {
            id: 'deploy-2',
            url: 'https://docs-site.pages.dev',
            environment: 'production',
            deployment_trigger: { type: 'push', metadata: { branch: 'main', commit_hash: 'def456' } },
            created_at: new Date(Date.now() - 86400000).toISOString(),
            status: 'success',
          },
          domains: ['docs.example.com'],
        },
        {
          name: 'marketing-site',
          subdomain: 'marketing-site',
          created_at: new Date(Date.now() - 259200000).toISOString(),
          production_branch: 'main',
          latest_deployment: {
            id: 'deploy-3',
            url: 'https://marketing-site.pages.dev',
            environment: 'production',
            deployment_trigger: { type: 'push', metadata: { branch: 'feature/hero', commit_hash: 'ghi789' } },
            created_at: new Date(Date.now() - 7200000).toISOString(),
            status: 'building',
          },
          domains: [],
        },
      ])
    } finally {
      setLoading(false)
    }
  }

  const handleCreate = async (values: typeof form.values) => {
    try {
      await api.pages.createProject(values)
      notifications.show({ title: 'Success', message: 'Project created', color: 'green' })
      setCreateModalOpen(false)
      form.reset()
      loadProjects()
    } catch (error) {
      notifications.show({ title: 'Error', message: 'Failed to create project', color: 'red' })
    }
  }

  const handleDelete = async (project: PagesProject) => {
    if (!confirm(`Delete project "${project.name}"? This will delete all deployments.`)) return
    try {
      await api.pages.deleteProject(project.name)
      notifications.show({ title: 'Success', message: 'Project deleted', color: 'green' })
      loadProjects()
    } catch (error) {
      notifications.show({ title: 'Error', message: 'Failed to delete project', color: 'red' })
    }
  }

  const formatDate = (dateStr: string) => {
    const date = new Date(dateStr)
    const now = new Date()
    const diff = now.getTime() - date.getTime()
    const hours = Math.floor(diff / 3600000)
    if (hours < 1) return 'Just now'
    if (hours < 24) return `${hours}h ago`
    const days = Math.floor(hours / 24)
    if (days < 7) return `${days}d ago`
    return date.toLocaleDateString()
  }

  const columns: Column<PagesProject>[] = [
    { key: 'name', label: 'Project', sortable: true },
    {
      key: 'latest_deployment.status',
      label: 'Status',
      render: (row) => {
        const status = row.latest_deployment?.status
        return status ? <StatusBadge status={status} /> : <Badge color="gray">No deployment</Badge>
      },
    },
    {
      key: 'latest_deployment.url',
      label: 'URL',
      render: (row) => row.latest_deployment?.url ? (
        <Group gap={4}>
          <MantineText size="sm" truncate style={{ maxWidth: 200 }}>{row.latest_deployment.url}</MantineText>
          <IconExternalLink size={12} color="var(--mantine-color-dimmed)" />
        </Group>
      ) : '-',
    },
    {
      key: 'production_branch',
      label: 'Branch',
      render: (row) => (
        <Group gap={4}>
          <IconGitBranch size={14} />
          <MantineText size="sm">{row.production_branch}</MantineText>
        </Group>
      ),
    },
    {
      key: 'latest_deployment.created_at',
      label: 'Last Deploy',
      render: (row) => row.latest_deployment?.created_at ? formatDate(row.latest_deployment.created_at) : '-',
    },
  ]

  return (
    <Stack gap="lg">
      <PageHeader
        title="Pages"
        subtitle="Full-stack site hosting with edge functions"
        actions={
          <Button leftSection={<IconPlus size={16} />} onClick={() => setCreateModalOpen(true)}>
            Create Project
          </Button>
        }
      />

      <DataTable
        data={projects}
        columns={columns}
        loading={loading}
        getRowKey={(row) => row.name}
        searchPlaceholder="Search projects..."
        onRowClick={(row) => navigate(`/pages/${row.name}`)}
        actions={[
          { label: 'View', icon: <IconEye size={14} />, onClick: (row) => navigate(`/pages/${row.name}`) },
          { label: 'Deploy', icon: <IconRocket size={14} />, onClick: (row) => navigate(`/pages/${row.name}`) },
          { label: 'Delete', icon: <IconTrash size={14} />, onClick: handleDelete, color: 'red' },
        ]}
        emptyState={{
          title: 'No projects yet',
          description: 'Create your first Pages project to deploy a website',
          action: { label: 'Create Project', onClick: () => setCreateModalOpen(true) },
        }}
      />

      <Modal opened={createModalOpen} onClose={() => setCreateModalOpen(false)} title="Create Pages Project" size="md">
        <form onSubmit={form.onSubmit(handleCreate)}>
          <Stack gap="md">
            <TextInput
              label="Project Name"
              placeholder="my-site"
              description="Use lowercase letters, numbers, and hyphens. This will be your *.pages.dev subdomain."
              required
              {...form.getInputProps('name')}
            />
            <TextInput
              label="Production Branch"
              placeholder="main"
              description="The branch that triggers production deployments"
              {...form.getInputProps('production_branch')}
            />
            <Group justify="flex-end" mt="md">
              <Button variant="default" onClick={() => setCreateModalOpen(false)}>Cancel</Button>
              <Button type="submit">Create Project</Button>
            </Group>
          </Stack>
        </form>
      </Modal>
    </Stack>
  )
}
