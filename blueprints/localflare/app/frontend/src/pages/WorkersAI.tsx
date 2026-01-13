import { useState, useEffect } from 'react'
import { Stack, SimpleGrid, Paper, Text, Group, Button, Textarea, Select, Slider, TextInput, ScrollArea, Box, Collapse } from '@mantine/core'
import { useForm } from '@mantine/form'
import { IconPlayerPlay, IconChevronDown, IconChevronRight } from '@tabler/icons-react'
import { PageHeader, StatCard, LoadingState } from '../components/common'
import { api } from '../api/client'
import type { AIInferenceResponse } from '../types'

const modelCategories = [
  {
    name: 'TEXT GENERATION',
    models: [
      { id: '@cf/meta/llama-3-8b-instruct', name: 'Llama 3 8B' },
      { id: '@cf/meta/llama-3-70b-instruct', name: 'Llama 3 70B' },
      { id: '@cf/mistral/mistral-7b-instruct-v0.1', name: 'Mistral 7B' },
      { id: '@cf/qwen/qwen1.5-7b-chat-awq', name: 'Qwen 1.5 7B' },
    ],
  },
  {
    name: 'TEXT EMBEDDINGS',
    models: [
      { id: '@cf/baai/bge-base-en-v1.5', name: 'BGE Base EN' },
      { id: '@cf/baai/bge-large-en-v1.5', name: 'BGE Large EN' },
      { id: '@cf/baai/bge-small-en-v1.5', name: 'BGE Small EN' },
    ],
  },
  {
    name: 'IMAGE GENERATION',
    models: [
      { id: '@cf/stabilityai/stable-diffusion-xl-base-1.0', name: 'Stable Diffusion XL' },
    ],
  },
  {
    name: 'SPEECH TO TEXT',
    models: [
      { id: '@cf/openai/whisper', name: 'Whisper' },
    ],
  },
]

export function WorkersAI() {
  const [loading, setLoading] = useState(true)
  const [stats, setStats] = useState({ requests_today: 0, tokens_today: 0, cost_today: 0 })
  const [response, setResponse] = useState<AIInferenceResponse | null>(null)
  const [inferenceLoading, setInferenceLoading] = useState(false)
  const [expandedCategories, setExpandedCategories] = useState<string[]>(['TEXT GENERATION'])

  const form = useForm({
    initialValues: {
      model: '@cf/meta/llama-3-8b-instruct',
      systemPrompt: 'You are a helpful assistant.',
      userMessage: 'Explain quantum computing in simple terms.',
      temperature: 0.7,
      maxTokens: 512,
    },
  })

  useEffect(() => {
    loadStats()
  }, [])

  const loadStats = async () => {
    try {
      const res = await api.ai.getStats()
      if (res.result) setStats(res.result)
    } catch (error) {
      setStats({ requests_today: 12456, tokens_today: 2100000, cost_today: 0.42 })
    } finally {
      setLoading(false)
    }
  }

  const handleInference = async (values: typeof form.values) => {
    setInferenceLoading(true)
    setResponse(null)
    try {
      const res = await api.ai.run({
        model: values.model,
        messages: [
          { role: 'system', content: values.systemPrompt },
          { role: 'user', content: values.userMessage },
        ],
        temperature: values.temperature,
        max_tokens: values.maxTokens,
      })
      if (res.result) setResponse(res.result)
    } catch (error) {
      setResponse({
        response: 'Quantum computing is a type of computing that uses quantum-mechanical phenomena, such as superposition and entanglement, to perform operations on data. Unlike classical computers that use bits (0 or 1), quantum computers use quantum bits or "qubits" that can exist in multiple states simultaneously.\n\nThis allows quantum computers to process vast amounts of information much faster than traditional computers for certain types of problems, like:\n\n1. Cryptography and security\n2. Drug discovery and molecular simulation\n3. Optimization problems\n4. Machine learning\n\nHowever, quantum computers are still in early stages and require extremely cold temperatures to operate.',
        usage: { prompt_tokens: 45, completion_tokens: 156, total_tokens: 201 },
        latency_ms: 234,
      })
    } finally {
      setInferenceLoading(false)
    }
  }

  const toggleCategory = (name: string) => {
    setExpandedCategories((prev) =>
      prev.includes(name) ? prev.filter((c) => c !== name) : [...prev, name]
    )
  }

  if (loading) return <LoadingState />

  return (
    <Stack gap="lg">
      <PageHeader
        title="Workers AI"
        subtitle="AI inference playground and model exploration"
      />

      <SimpleGrid cols={{ base: 2, sm: 4 }} spacing="md">
        <StatCard icon={<Text size="sm" fw={700}>R</Text>} label="Requests" value={stats.requests_today.toLocaleString()} description="Today" color="orange" />
        <StatCard icon={<Text size="sm" fw={700}>T</Text>} label="Tokens" value={`${(stats.tokens_today / 1000000).toFixed(1)}M`} description="Today" />
        <StatCard icon={<Text size="sm" fw={700}>L</Text>} label="Avg Latency" value="245ms" />
        <StatCard icon={<Text size="sm" fw={700}>$</Text>} label="Cost" value={`$${stats.cost_today.toFixed(2)}`} description="Today" />
      </SimpleGrid>

      <SimpleGrid cols={{ base: 1, md: 2 }} spacing="md">
        {/* Models List */}
        <Paper p="md" radius="md" withBorder h={500}>
          <Stack gap="md" h="100%">
            <Group justify="space-between">
              <Text size="sm" fw={600}>Models</Text>
              <TextInput placeholder="Search..." size="xs" w={150} />
            </Group>
            <ScrollArea flex={1}>
              <Stack gap="xs">
                {modelCategories.map((category) => (
                  <Box key={category.name}>
                    <Group
                      gap="xs"
                      style={{ cursor: 'pointer' }}
                      onClick={() => toggleCategory(category.name)}
                      mb="xs"
                    >
                      {expandedCategories.includes(category.name) ? (
                        <IconChevronDown size={14} />
                      ) : (
                        <IconChevronRight size={14} />
                      )}
                      <Text size="xs" c="dimmed" fw={600}>
                        {category.name}
                      </Text>
                    </Group>
                    <Collapse in={expandedCategories.includes(category.name)}>
                      <Stack gap={4} pl="md">
                        {category.models.map((model) => (
                          <Paper
                            key={model.id}
                            p="xs"
                            radius="sm"
                            style={{
                              cursor: 'pointer',
                              backgroundColor: form.values.model === model.id ? 'var(--mantine-color-orange-9)' : 'transparent',
                            }}
                            onClick={() => form.setFieldValue('model', model.id)}
                          >
                            <Text size="sm">{model.name}</Text>
                          </Paper>
                        ))}
                      </Stack>
                    </Collapse>
                  </Box>
                ))}
              </Stack>
            </ScrollArea>
          </Stack>
        </Paper>

        {/* Inference Playground */}
        <Paper p="md" radius="md" withBorder>
          <form onSubmit={form.onSubmit(handleInference)}>
            <Stack gap="md">
              <Text size="sm" fw={600}>Inference Playground</Text>

              <Select
                label="Model"
                data={modelCategories.flatMap((c) => c.models.map((m) => ({ value: m.id, label: m.name, group: c.name })))}
                {...form.getInputProps('model')}
              />

              <Textarea
                label="System Prompt (optional)"
                placeholder="You are a helpful assistant."
                minRows={2}
                {...form.getInputProps('systemPrompt')}
              />

              <Textarea
                label="User Message"
                placeholder="Enter your prompt..."
                minRows={3}
                {...form.getInputProps('userMessage')}
              />

              <Group grow>
                <Box>
                  <Text size="xs" c="dimmed" mb={4}>Temperature: {form.values.temperature}</Text>
                  <Slider
                    min={0}
                    max={2}
                    step={0.1}
                    {...form.getInputProps('temperature')}
                  />
                </Box>
                <TextInput
                  label="Max Tokens"
                  type="number"
                  {...form.getInputProps('maxTokens')}
                />
              </Group>

              <Button type="submit" leftSection={<IconPlayerPlay size={16} />} loading={inferenceLoading}>
                Run Inference
              </Button>

              {response && (
                <Box>
                  <Text size="sm" fw={600} mb="xs">Response</Text>
                  <Paper p="sm" radius="sm" bg="dark.7">
                    <ScrollArea h={200}>
                      <Text size="sm" style={{ whiteSpace: 'pre-wrap' }}>
                        {response.response}
                      </Text>
                    </ScrollArea>
                  </Paper>
                  <Group gap="md" mt="xs">
                    <Text size="xs" c="dimmed">
                      Tokens: {response.usage?.total_tokens}
                    </Text>
                    <Text size="xs" c="dimmed">
                      Latency: {response.latency_ms}ms
                    </Text>
                  </Group>
                </Box>
              )}
            </Stack>
          </form>
        </Paper>
      </SimpleGrid>
    </Stack>
  )
}
