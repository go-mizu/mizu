import { useEffect, useState } from 'react'
import { Container, Title, Text, Card, Group, Stack, Button, SimpleGrid, Badge, Paper } from '@mantine/core'
import { IconChartBar, IconLayoutDashboard, IconFolder, IconDatabase, IconPlus } from '@tabler/icons-react'
import { useNavigate } from 'react-router-dom'
import { api } from '../api/client'

interface Stats {
  questions: number
  dashboards: number
  collections: number
  datasources: number
}

interface Dashboard {
  id: string
  name: string
  description: string
}

interface Question {
  id: string
  name: string
  visualization?: { type: string }
}

export default function Home() {
  const navigate = useNavigate()
  const [stats, setStats] = useState<Stats>({ questions: 0, dashboards: 0, collections: 0, datasources: 0 })
  const [dashboards, setDashboards] = useState<Dashboard[]>([])
  const [questions, setQuestions] = useState<Question[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    loadData()
  }, [])

  const loadData = async () => {
    try {
      const [questionsRes, dashboardsRes, collectionsRes, datasourcesRes] = await Promise.all([
        api.get<Question[]>('/questions'),
        api.get<Dashboard[]>('/dashboards'),
        api.get<{ id: string }[]>('/collections'),
        api.get<{ id: string }[]>('/datasources'),
      ])

      setStats({
        questions: questionsRes?.length || 0,
        dashboards: dashboardsRes?.length || 0,
        collections: collectionsRes?.length || 0,
        datasources: datasourcesRes?.length || 0,
      })

      setDashboards(dashboardsRes?.slice(0, 4) || [])
      setQuestions(questionsRes?.slice(0, 6) || [])
    } catch (error) {
      console.error('Failed to load data:', error)
    } finally {
      setLoading(false)
    }
  }

  const statCards = [
    { label: 'Questions', value: stats.questions, icon: IconChartBar, color: 'blue' },
    { label: 'Dashboards', value: stats.dashboards, icon: IconLayoutDashboard, color: 'green' },
    { label: 'Collections', value: stats.collections, icon: IconFolder, color: 'violet' },
    { label: 'Data Sources', value: stats.datasources, icon: IconDatabase, color: 'orange' },
  ]

  return (
    <Container size="xl" py="lg">
      {/* Header */}
      <Group justify="space-between" mb="xl">
        <div>
          <Title order={2}>Welcome to BI</Title>
          <Text c="dimmed">Your business intelligence platform</Text>
        </div>
        <Group>
          <Button leftSection={<IconPlus size={16} />} onClick={() => navigate('/question/new')}>
            New Question
          </Button>
          <Button variant="light" leftSection={<IconLayoutDashboard size={16} />} onClick={() => navigate('/dashboard/new')}>
            New Dashboard
          </Button>
        </Group>
      </Group>

      {/* Stats */}
      <SimpleGrid cols={{ base: 1, sm: 2, md: 4 }} mb="xl">
        {statCards.map((stat) => (
          <Card key={stat.label} withBorder radius="md" padding="lg">
            <Group justify="space-between">
              <div>
                <Text size="xl" fw={700} c={stat.color}>
                  {stat.value}
                </Text>
                <Text c="dimmed" size="sm" mt={4}>
                  {stat.label}
                </Text>
              </div>
              <stat.icon size={32} color={`var(--mantine-color-${stat.color}-6)`} opacity={0.5} />
            </Group>
          </Card>
        ))}
      </SimpleGrid>

      {/* Recent Dashboards */}
      {dashboards.length > 0 && (
        <>
          <Group justify="space-between" mb="md">
            <Title order={4}>Recent Dashboards</Title>
            <Button variant="subtle" size="sm" onClick={() => navigate('/browse')}>
              View all
            </Button>
          </Group>
          <SimpleGrid cols={{ base: 1, sm: 2, md: 4 }} mb="xl">
            {dashboards.map((dashboard) => (
              <Card
                key={dashboard.id}
                withBorder
                radius="md"
                padding="lg"
                style={{ cursor: 'pointer' }}
                onClick={() => navigate(`/dashboard/${dashboard.id}`)}
              >
                <Group>
                  <IconLayoutDashboard size={24} color="var(--mantine-color-green-6)" />
                  <div>
                    <Text fw={500}>{dashboard.name}</Text>
                    {dashboard.description && (
                      <Text size="sm" c="dimmed" lineClamp={1}>
                        {dashboard.description}
                      </Text>
                    )}
                  </div>
                </Group>
              </Card>
            ))}
          </SimpleGrid>
        </>
      )}

      {/* Recent Questions */}
      {questions.length > 0 && (
        <>
          <Group justify="space-between" mb="md">
            <Title order={4}>Recent Questions</Title>
            <Button variant="subtle" size="sm" onClick={() => navigate('/browse')}>
              View all
            </Button>
          </Group>
          <SimpleGrid cols={{ base: 1, sm: 2, md: 3 }} mb="xl">
            {questions.map((question) => (
              <Card
                key={question.id}
                withBorder
                radius="md"
                padding="lg"
                style={{ cursor: 'pointer' }}
                onClick={() => navigate(`/question/${question.id}`)}
              >
                <Group>
                  <IconChartBar size={24} color="var(--mantine-color-blue-6)" />
                  <div>
                    <Text fw={500}>{question.name}</Text>
                    <Badge size="sm" variant="light" color="blue">
                      {question.visualization?.type || 'table'}
                    </Badge>
                  </div>
                </Group>
              </Card>
            ))}
          </SimpleGrid>
        </>
      )}

      {/* Empty state */}
      {!loading && stats.questions === 0 && stats.dashboards === 0 && (
        <Paper withBorder radius="md" p="xl" ta="center">
          <Stack align="center" gap="md">
            <IconDatabase size={48} color="var(--mantine-color-gray-5)" />
            <Title order={3}>Get started with BI</Title>
            <Text c="dimmed" maw={400}>
              Run 'bi seed' to add sample data, or add a data source to start exploring your data.
            </Text>
            <Group>
              <Button leftSection={<IconPlus size={16} />} onClick={() => navigate('/admin/datamodel')}>
                Add Data Source
              </Button>
            </Group>
          </Stack>
        </Paper>
      )}
    </Container>
  )
}
