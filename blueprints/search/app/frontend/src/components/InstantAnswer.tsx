import { Paper, Text, Group, Badge } from '@mantine/core'
import { IconCalculator, IconCurrencyDollar, IconCloud, IconBook, IconClock, IconTransform } from '@tabler/icons-react'
import type { InstantAnswer as InstantAnswerType } from '../types'

interface InstantAnswerProps {
  answer: InstantAnswerType
}

const icons: Record<string, typeof IconCalculator> = {
  calculator: IconCalculator,
  currency: IconCurrencyDollar,
  weather: IconCloud,
  definition: IconBook,
  time: IconClock,
  unit: IconTransform,
}

interface WeatherData {
  humidity: number
  wind_speed: number
  wind_unit: string
  condition: string
}

interface DefinitionData {
  phonetic?: string
  part_of_speech: string
  definitions?: string[]
  synonyms?: string[]
}

interface TimeData {
  date: string
  timezone: string
  offset: string
}

interface CurrencyData {
  from_currency: string
  to_currency: string
  rate: number
  updated_at: string
}

interface UnitData {
  from_value: number
  from_unit: string
  to_value: number
  to_unit: string
  category: string
}

export function InstantAnswer({ answer }: InstantAnswerProps) {
  const Icon = icons[answer.type] || IconCalculator

  return (
    <Paper className="instant-answer" withBorder p="md">
      <Group gap="sm" mb="xs">
        <Icon size={20} className="text-blue-500" />
        <Badge variant="light" size="sm">
          {answer.type.charAt(0).toUpperCase() + answer.type.slice(1)}
        </Badge>
      </Group>

      <Text className="instant-answer-result">{answer.result}</Text>

      {answer.type === 'calculator' && (
        <Text size="sm" c="dimmed" mt="xs">
          {answer.query} =
        </Text>
      )}

      {answer.type === 'weather' && answer.data ? (
        <WeatherDetails data={answer.data as WeatherData} />
      ) : null}

      {answer.type === 'definition' && answer.data ? (
        <DefinitionDetails data={answer.data as DefinitionData} />
      ) : null}

      {answer.type === 'time' && answer.data ? (
        <TimeDetails data={answer.data as TimeData} />
      ) : null}

      {answer.type === 'currency' && answer.data ? (
        <CurrencyDetails data={answer.data as CurrencyData} />
      ) : null}

      {answer.type === 'unit' && answer.data ? (
        <UnitDetails data={answer.data as UnitData} />
      ) : null}
    </Paper>
  )
}

function WeatherDetails({ data }: { data: WeatherData }) {
  return (
    <Group mt="md" gap="xl">
      <div>
        <Text size="xs" c="dimmed">Humidity</Text>
        <Text size="sm">{data.humidity}%</Text>
      </div>
      <div>
        <Text size="xs" c="dimmed">Wind</Text>
        <Text size="sm">{data.wind_speed} {data.wind_unit}</Text>
      </div>
      <div>
        <Text size="xs" c="dimmed">Condition</Text>
        <Text size="sm">{data.condition}</Text>
      </div>
    </Group>
  )
}

function DefinitionDetails({ data }: { data: DefinitionData }) {
  return (
    <div className="mt-3">
      {data.phonetic && (
        <Text size="sm" c="dimmed" fs="italic">{data.phonetic}</Text>
      )}
      <Text size="sm" c="dimmed" mt="xs">{data.part_of_speech}</Text>
      {data.definitions && data.definitions.length > 1 && (
        <Text size="sm" mt="xs" c="dimmed">
          2. {data.definitions[1]}
        </Text>
      )}
      {data.synonyms && data.synonyms.length > 0 && (
        <Text size="xs" c="dimmed" mt="sm">
          Synonyms: {data.synonyms.join(', ')}
        </Text>
      )}
    </div>
  )
}

function TimeDetails({ data }: { data: TimeData }) {
  return (
    <div className="mt-2">
      <Text size="sm" c="dimmed">{data.date}</Text>
      <Text size="xs" c="dimmed">{data.timezone} ({data.offset})</Text>
    </div>
  )
}

function CurrencyDetails({ data }: { data: CurrencyData }) {
  return (
    <div className="mt-2">
      <Text size="sm" c="dimmed">
        1 {data.from_currency} = {data.rate.toFixed(4)} {data.to_currency}
      </Text>
      <Text size="xs" c="dimmed">Rate as of {new Date(data.updated_at).toLocaleDateString()}</Text>
    </div>
  )
}

function UnitDetails({ data }: { data: UnitData }) {
  return (
    <div className="mt-2">
      <Text size="sm" c="dimmed">
        {data.from_value} {data.from_unit} = {data.to_value.toFixed(4)} {data.to_unit}
      </Text>
      <Text size="xs" c="dimmed" tt="capitalize">{data.category}</Text>
    </div>
  )
}
