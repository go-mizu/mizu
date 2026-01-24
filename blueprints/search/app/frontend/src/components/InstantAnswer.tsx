import { Calculator, DollarSign, Cloud, BookOpen, Clock, ArrowRightLeft } from 'lucide-react'
import type { InstantAnswer as InstantAnswerType } from '../types'

interface InstantAnswerProps {
  answer: InstantAnswerType
}

const icons: Record<string, typeof Calculator> = {
  calculator: Calculator,
  currency: DollarSign,
  weather: Cloud,
  definition: BookOpen,
  time: Clock,
  unit: ArrowRightLeft,
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
  const Icon = icons[answer.type] || Calculator

  return (
    <div className="bg-white border border-[#dadce0] rounded-lg p-4 mb-4">
      {/* Header */}
      <div className="flex items-center gap-2 mb-2">
        <Icon size={20} className="text-[#1a73e8]" />
        <span className="px-2 py-0.5 text-xs font-medium text-[#1a73e8] bg-[#e8f0fe] rounded">
          {answer.type.charAt(0).toUpperCase() + answer.type.slice(1)}
        </span>
      </div>

      {/* Result */}
      <div className="text-3xl font-semibold text-[#202124]">
        {answer.result}
      </div>

      {answer.type === 'calculator' && (
        <p className="text-sm text-[#70757a] mt-1">
          {answer.query} =
        </p>
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
    </div>
  )
}

function WeatherDetails({ data }: { data: WeatherData }) {
  return (
    <div className="flex gap-8 mt-4">
      <div>
        <p className="text-xs text-[#70757a]">Humidity</p>
        <p className="text-sm text-[#202124]">{data.humidity}%</p>
      </div>
      <div>
        <p className="text-xs text-[#70757a]">Wind</p>
        <p className="text-sm text-[#202124]">{data.wind_speed} {data.wind_unit}</p>
      </div>
      <div>
        <p className="text-xs text-[#70757a]">Condition</p>
        <p className="text-sm text-[#202124]">{data.condition}</p>
      </div>
    </div>
  )
}

function DefinitionDetails({ data }: { data: DefinitionData }) {
  return (
    <div className="mt-3">
      {data.phonetic && (
        <p className="text-sm text-[#70757a] italic">{data.phonetic}</p>
      )}
      <p className="text-sm text-[#70757a] mt-1">{data.part_of_speech}</p>
      {data.definitions && data.definitions.length > 1 && (
        <p className="text-sm text-[#70757a] mt-1">
          2. {data.definitions[1]}
        </p>
      )}
      {data.synonyms && data.synonyms.length > 0 && (
        <p className="text-xs text-[#70757a] mt-2">
          Synonyms: {data.synonyms.join(', ')}
        </p>
      )}
    </div>
  )
}

function TimeDetails({ data }: { data: TimeData }) {
  return (
    <div className="mt-2">
      <p className="text-sm text-[#70757a]">{data.date}</p>
      <p className="text-xs text-[#70757a]">{data.timezone} ({data.offset})</p>
    </div>
  )
}

function CurrencyDetails({ data }: { data: CurrencyData }) {
  return (
    <div className="mt-2">
      <p className="text-sm text-[#70757a]">
        1 {data.from_currency} = {data.rate.toFixed(4)} {data.to_currency}
      </p>
      <p className="text-xs text-[#70757a]">Rate as of {new Date(data.updated_at).toLocaleDateString()}</p>
    </div>
  )
}

function UnitDetails({ data }: { data: UnitData }) {
  return (
    <div className="mt-2">
      <p className="text-sm text-[#70757a]">
        {data.from_value} {data.from_unit} = {data.to_value.toFixed(4)} {data.to_unit}
      </p>
      <p className="text-xs text-[#70757a] capitalize">{data.category}</p>
    </div>
  )
}
