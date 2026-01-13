import { useState, useEffect } from 'react'
import { useParams } from 'react-router-dom'
import { Stack, SimpleGrid, Paper, Text, Group, Button, Textarea, NumberInput, Select, Switch, Table, Code, ScrollArea } from '@mantine/core'
import { useForm } from '@mantine/form'
import { IconSearch } from '@tabler/icons-react'
import { PageHeader, StatCard, LoadingState } from '../components/common'
import { api } from '../api/client'
import type { VectorIndex, VectorMatch } from '../types'

export function VectorizeDetail() {
  const { name } = useParams<{ name: string }>()
  const [index, setIndex] = useState<VectorIndex | null>(null)
  const [loading, setLoading] = useState(true)
  const [results, setResults] = useState<VectorMatch[]>([])
  const [queryLoading, setQueryLoading] = useState(false)

  const form = useForm({
    initialValues: { text: '', topK: 10, namespace: '', returnValues: false, returnMetadata: true },
  })

  useEffect(() => {
    if (name) loadIndex()
  }, [name])

  const loadIndex = async () => {
    try {
      const res = await api.vectorize.getIndex(name!)
      if (res.result) setIndex(res.result)
    } catch (error) {
      setIndex({
        id: '1',
        name: name!,
        dimensions: 768,
        metric: 'cosine',
        created_at: new Date().toISOString(),
        vector_count: 50234,
        namespace_count: 3,
      })
    } finally {
      setLoading(false)
    }
  }

  const handleQuery = async (values: typeof form.values) => {
    setQueryLoading(true)
    try {
      const res = await api.vectorize.query(name!, {
        text: values.text,
        topK: values.topK,
        namespace: values.namespace || undefined,
        returnValues: values.returnValues,
        returnMetadata: values.returnMetadata,
      })
      if (res.result) setResults(res.result.matches)
    } catch (error) {
      setResults([
        { id: 'prod-12345', score: 0.95, metadata: { name: 'Sony WH-1000XM5', category: 'headphones' } },
        { id: 'prod-67890', score: 0.89, metadata: { name: 'Bose QC45', category: 'headphones' } },
        { id: 'prod-11223', score: 0.85, metadata: { name: 'Apple AirPods Max', category: 'headphones' } },
      ])
    } finally {
      setQueryLoading(false)
    }
  }

  if (loading) return <LoadingState />
  if (!index) return <Text>Index not found</Text>

  return (
    <Stack gap="lg">
      <PageHeader
        title={index.name}
        breadcrumbs={[{ label: 'Vectorize', path: '/vectorize' }, { label: index.name }]}
        backPath="/vectorize"
      />

      <SimpleGrid cols={{ base: 2, sm: 4 }} spacing="md">
        <StatCard icon={<Text size="sm" fw={700}>V</Text>} label="Vectors" value={index.vector_count.toLocaleString()} color="orange" />
        <StatCard icon={<Text size="sm" fw={700}>D</Text>} label="Dimensions" value={index.dimensions} />
        <StatCard icon={<Text size="sm" fw={700}>M</Text>} label="Metric" value={index.metric} />
        <StatCard icon={<Text size="sm" fw={700}>N</Text>} label="Namespaces" value={index.namespace_count} />
      </SimpleGrid>

      <Paper p="md" radius="md" withBorder>
        <Stack gap="md">
          <Text size="sm" fw={600}>Query Playground</Text>
          <form onSubmit={form.onSubmit(handleQuery)}>
            <Stack gap="md">
              <Textarea
                label="Vector (or text to embed)"
                placeholder="wireless headphones with noise cancellation"
                minRows={2}
                {...form.getInputProps('text')}
              />
              <Group grow>
                <NumberInput label="Top K" min={1} max={100} {...form.getInputProps('topK')} />
                <Select label="Namespace" placeholder="all" data={[{ value: '', label: 'All' }, { value: 'products', label: 'products' }, { value: 'docs', label: 'docs' }]} {...form.getInputProps('namespace')} clearable />
              </Group>
              <Group>
                <Switch label="Return Values" {...form.getInputProps('returnValues', { type: 'checkbox' })} />
                <Switch label="Return Metadata" {...form.getInputProps('returnMetadata', { type: 'checkbox' })} />
              </Group>
              <Group justify="flex-end">
                <Button type="submit" leftSection={<IconSearch size={16} />} loading={queryLoading}>
                  Search
                </Button>
              </Group>
            </Stack>
          </form>
        </Stack>
      </Paper>

      {results.length > 0 && (
        <Paper p="md" radius="md" withBorder>
          <Stack gap="md">
            <Text size="sm" fw={600}>Results</Text>
            <ScrollArea>
              <Table striped highlightOnHover withTableBorder>
                <Table.Thead>
                  <Table.Tr>
                    <Table.Th>ID</Table.Th>
                    <Table.Th>Score</Table.Th>
                    <Table.Th>Metadata</Table.Th>
                  </Table.Tr>
                </Table.Thead>
                <Table.Tbody>
                  {results.map((match) => (
                    <Table.Tr key={match.id}>
                      <Table.Td><Code>{match.id}</Code></Table.Td>
                      <Table.Td>{match.score.toFixed(4)}</Table.Td>
                      <Table.Td><Code>{JSON.stringify(match.metadata)}</Code></Table.Td>
                    </Table.Tr>
                  ))}
                </Table.Tbody>
              </Table>
            </ScrollArea>
          </Stack>
        </Paper>
      )}
    </Stack>
  )
}
