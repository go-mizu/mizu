import { useEffect, useState } from 'react'
import { Container, Title, Text, Card, Group, Stack, SimpleGrid, Badge, TextInput, Tabs, Paper } from '@mantine/core'
import { IconChartBar, IconLayoutDashboard, IconFolder, IconSearch } from '@tabler/icons-react'
import { useNavigate } from 'react-router-dom'
import { api } from '../api/client'

interface Collection {
  id: string
  name: string
  color: string
}

interface Dashboard {
  id: string
  name: string
  description: string
  collection_id?: string
}

interface Question {
  id: string
  name: string
  collection_id?: string
  visualization?: { type: string }
}

export default function Browse() {
  const navigate = useNavigate()
  const [collections, setCollections] = useState<Collection[]>([])
  const [dashboards, setDashboards] = useState<Dashboard[]>([])
  const [questions, setQuestions] = useState<Question[]>([])
  const [search, setSearch] = useState('')
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    loadData()
  }, [])

  const loadData = async () => {
    try {
      const [collectionsRes, dashboardsRes, questionsRes] = await Promise.all([
        api.get<Collection[]>('/collections'),
        api.get<Dashboard[]>('/dashboards'),
        api.get<Question[]>('/questions'),
      ])
      setCollections(collectionsRes || [])
      setDashboards(dashboardsRes || [])
      setQuestions(questionsRes || [])
    } catch (error) {
      console.error('Failed to load data:', error)
    } finally {
      setLoading(false)
    }
  }

  const filteredDashboards = dashboards.filter(d =>
    d.name.toLowerCase().includes(search.toLowerCase())
  )

  const filteredQuestions = questions.filter(q =>
    q.name.toLowerCase().includes(search.toLowerCase())
  )

  const filteredCollections = collections.filter(c =>
    c.name.toLowerCase().includes(search.toLowerCase())
  )

  return (
    <Container size="xl" py="lg">
      {/* Header */}
      <Group justify="space-between" mb="xl">
        <div>
          <Title order={2}>Browse</Title>
          <Text c="dimmed">Explore your data</Text>
        </div>
        <TextInput
          placeholder="Search..."
          leftSection={<IconSearch size={16} />}
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          w={300}
        />
      </Group>

      <Tabs defaultValue="all">
        <Tabs.List mb="lg">
          <Tabs.Tab value="all">All</Tabs.Tab>
          <Tabs.Tab value="dashboards">Dashboards ({dashboards.length})</Tabs.Tab>
          <Tabs.Tab value="questions">Questions ({questions.length})</Tabs.Tab>
          <Tabs.Tab value="collections">Collections ({collections.length})</Tabs.Tab>
        </Tabs.List>

        <Tabs.Panel value="all">
          {/* Collections */}
          {filteredCollections.length > 0 && (
            <>
              <Title order={4} mb="md">Collections</Title>
              <SimpleGrid cols={{ base: 1, sm: 2, md: 4 }} mb="xl">
                {filteredCollections.map((collection) => (
                  <Card
                    key={collection.id}
                    withBorder
                    radius="md"
                    padding="lg"
                    style={{ cursor: 'pointer', borderLeft: `4px solid ${collection.color || '#509EE3'}` }}
                    onClick={() => navigate(`/browse/${collection.id}`)}
                  >
                    <Group>
                      <IconFolder size={24} color={collection.color || '#509EE3'} />
                      <Text fw={500}>{collection.name}</Text>
                    </Group>
                  </Card>
                ))}
              </SimpleGrid>
            </>
          )}

          {/* Dashboards */}
          {filteredDashboards.length > 0 && (
            <>
              <Title order={4} mb="md">Dashboards</Title>
              <SimpleGrid cols={{ base: 1, sm: 2, md: 3 }} mb="xl">
                {filteredDashboards.map((dashboard) => (
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

          {/* Questions */}
          {filteredQuestions.length > 0 && (
            <>
              <Title order={4} mb="md">Questions</Title>
              <SimpleGrid cols={{ base: 1, sm: 2, md: 3 }} mb="xl">
                {filteredQuestions.map((question) => (
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
          {!loading && filteredDashboards.length === 0 && filteredQuestions.length === 0 && filteredCollections.length === 0 && (
            <Paper withBorder radius="md" p="xl" ta="center">
              <Stack align="center" gap="md">
                <IconSearch size={48} color="var(--mantine-color-gray-5)" />
                <Title order={3}>No results found</Title>
                <Text c="dimmed">
                  {search ? 'Try adjusting your search' : 'Run "bi seed" to add sample data'}
                </Text>
              </Stack>
            </Paper>
          )}
        </Tabs.Panel>

        <Tabs.Panel value="dashboards">
          <SimpleGrid cols={{ base: 1, sm: 2, md: 3 }}>
            {filteredDashboards.map((dashboard) => (
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
        </Tabs.Panel>

        <Tabs.Panel value="questions">
          <SimpleGrid cols={{ base: 1, sm: 2, md: 3 }}>
            {filteredQuestions.map((question) => (
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
        </Tabs.Panel>

        <Tabs.Panel value="collections">
          <SimpleGrid cols={{ base: 1, sm: 2, md: 4 }}>
            {filteredCollections.map((collection) => (
              <Card
                key={collection.id}
                withBorder
                radius="md"
                padding="lg"
                style={{ cursor: 'pointer', borderLeft: `4px solid ${collection.color || '#509EE3'}` }}
                onClick={() => navigate(`/browse/${collection.id}`)}
              >
                <Group>
                  <IconFolder size={24} color={collection.color || '#509EE3'} />
                  <Text fw={500}>{collection.name}</Text>
                </Group>
              </Card>
            ))}
          </SimpleGrid>
        </Tabs.Panel>
      </Tabs>
    </Container>
  )
}
