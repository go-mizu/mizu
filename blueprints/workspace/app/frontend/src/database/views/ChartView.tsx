import { useState, useMemo } from 'react'
import {
  BarChart,
  Bar,
  LineChart,
  Line,
  PieChart,
  Pie,
  Cell,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from 'recharts'
import { BarChart2, TrendingUp, PieChart as PieChartIcon, Settings } from 'lucide-react'
import { DatabaseRow, Property } from '../../api/client'

type ChartType = 'bar' | 'line' | 'pie' | 'donut'

interface ChartViewProps {
  rows: DatabaseRow[]
  properties: Property[]
  groupBy: string | null
  onAddRow: (initialProperties?: Record<string, unknown>) => Promise<DatabaseRow | null>
  onUpdateRow: (rowId: string, updates: Record<string, unknown>) => void
  onDeleteRow: (rowId: string) => void
  onAddProperty: (property: Omit<Property, 'id'>) => void
  onUpdateProperty: (propertyId: string, updates: Partial<Property>) => void
  onDeleteProperty: (propertyId: string) => void
}

const CHART_COLORS = [
  '#2383e2', // blue
  '#6c5ce7', // purple
  '#00b894', // green
  '#fdcb6e', // yellow
  '#e17055', // orange
  '#d63031', // red
  '#0984e3', // light blue
  '#a29bfe', // lavender
  '#55efc4', // mint
  '#fab1a0', // peach
]

export function ChartView({
  rows,
  properties,
  onAddRow,
}: ChartViewProps) {
  const [chartType, setChartType] = useState<ChartType>('bar')
  const [xAxisProperty, setXAxisProperty] = useState<string>('')
  const [yAxisProperty, setYAxisProperty] = useState<string>('')
  const [showSettings, setShowSettings] = useState(false)

  // Get properties suitable for X and Y axes
  const categoricalProperties = properties.filter((p) =>
    ['select', 'multi_select', 'text', 'status'].includes(p.type)
  )
  const numericProperties = properties.filter((p) =>
    ['number', 'formula'].includes(p.type)
  )

  // Auto-select properties if not set
  useMemo(() => {
    if (!xAxisProperty && categoricalProperties.length > 0) {
      setXAxisProperty(categoricalProperties[0].id)
    }
    if (!yAxisProperty && numericProperties.length > 0) {
      setYAxisProperty(numericProperties[0].id)
    }
  }, [categoricalProperties, numericProperties, xAxisProperty, yAxisProperty])

  // Transform data for charts
  const chartData = useMemo(() => {
    if (!xAxisProperty) return []

    const groupedData: Record<string, { name: string; value: number; count: number }> = {}

    rows.forEach((row) => {
      const xValue = String(row.properties[xAxisProperty] || 'Unknown')
      let yValue = 1 // Default to count

      if (yAxisProperty && row.properties[yAxisProperty] !== undefined) {
        const rawValue = row.properties[yAxisProperty]
        yValue = typeof rawValue === 'number' ? rawValue : parseFloat(String(rawValue)) || 0
      }

      if (!groupedData[xValue]) {
        groupedData[xValue] = { name: xValue, value: 0, count: 0 }
      }
      groupedData[xValue].value += yValue
      groupedData[xValue].count += 1
    })

    return Object.values(groupedData)
  }, [rows, xAxisProperty, yAxisProperty])

  const renderChart = () => {
    if (chartData.length === 0) {
      return (
        <div className="chart-empty-state">
          <BarChart2 size={48} />
          <h3>No data to display</h3>
          <p>Add rows to your database or configure chart settings</p>
          <button className="btn-primary" onClick={() => onAddRow()}>
            Add first row
          </button>
        </div>
      )
    }

    switch (chartType) {
      case 'bar':
        return (
          <ResponsiveContainer width="100%" height={400}>
            <BarChart data={chartData}>
              <CartesianGrid strokeDasharray="3 3" stroke="var(--border-color)" />
              <XAxis
                dataKey="name"
                tick={{ fill: 'var(--text-secondary)', fontSize: 12 }}
                axisLine={{ stroke: 'var(--border-color)' }}
              />
              <YAxis
                tick={{ fill: 'var(--text-secondary)', fontSize: 12 }}
                axisLine={{ stroke: 'var(--border-color)' }}
              />
              <Tooltip
                contentStyle={{
                  background: 'var(--bg-primary)',
                  border: '1px solid var(--border-color)',
                  borderRadius: '6px',
                }}
              />
              <Legend />
              <Bar
                dataKey="value"
                name={yAxisProperty ? properties.find((p) => p.id === yAxisProperty)?.name : 'Count'}
                fill={CHART_COLORS[0]}
                radius={[4, 4, 0, 0]}
              />
            </BarChart>
          </ResponsiveContainer>
        )

      case 'line':
        return (
          <ResponsiveContainer width="100%" height={400}>
            <LineChart data={chartData}>
              <CartesianGrid strokeDasharray="3 3" stroke="var(--border-color)" />
              <XAxis
                dataKey="name"
                tick={{ fill: 'var(--text-secondary)', fontSize: 12 }}
                axisLine={{ stroke: 'var(--border-color)' }}
              />
              <YAxis
                tick={{ fill: 'var(--text-secondary)', fontSize: 12 }}
                axisLine={{ stroke: 'var(--border-color)' }}
              />
              <Tooltip
                contentStyle={{
                  background: 'var(--bg-primary)',
                  border: '1px solid var(--border-color)',
                  borderRadius: '6px',
                }}
              />
              <Legend />
              <Line
                type="monotone"
                dataKey="value"
                name={yAxisProperty ? properties.find((p) => p.id === yAxisProperty)?.name : 'Count'}
                stroke={CHART_COLORS[0]}
                strokeWidth={2}
                dot={{ fill: CHART_COLORS[0], strokeWidth: 2 }}
                activeDot={{ r: 6 }}
              />
            </LineChart>
          </ResponsiveContainer>
        )

      case 'pie':
      case 'donut':
        return (
          <ResponsiveContainer width="100%" height={400}>
            <PieChart>
              <Pie
                data={chartData}
                cx="50%"
                cy="50%"
                innerRadius={chartType === 'donut' ? 80 : 0}
                outerRadius={140}
                paddingAngle={2}
                dataKey="value"
                nameKey="name"
                label={({ name, percent }) => `${name} (${(percent * 100).toFixed(0)}%)`}
                labelLine={false}
              >
                {chartData.map((_, index) => (
                  <Cell
                    key={`cell-${index}`}
                    fill={CHART_COLORS[index % CHART_COLORS.length]}
                  />
                ))}
              </Pie>
              <Tooltip
                contentStyle={{
                  background: 'var(--bg-primary)',
                  border: '1px solid var(--border-color)',
                  borderRadius: '6px',
                }}
              />
              <Legend />
            </PieChart>
          </ResponsiveContainer>
        )

      default:
        return null
    }
  }

  return (
    <div className="chart-view">
      <div className="chart-toolbar">
        <div className="chart-type-selector">
          <button
            className={`chart-type-btn ${chartType === 'bar' ? 'active' : ''}`}
            onClick={() => setChartType('bar')}
            title="Bar chart"
          >
            <BarChart2 size={18} />
          </button>
          <button
            className={`chart-type-btn ${chartType === 'line' ? 'active' : ''}`}
            onClick={() => setChartType('line')}
            title="Line chart"
          >
            <TrendingUp size={18} />
          </button>
          <button
            className={`chart-type-btn ${chartType === 'pie' ? 'active' : ''}`}
            onClick={() => setChartType('pie')}
            title="Pie chart"
          >
            <PieChartIcon size={18} />
          </button>
          <button
            className={`chart-type-btn ${chartType === 'donut' ? 'active' : ''}`}
            onClick={() => setChartType('donut')}
            title="Donut chart"
          >
            <div className="donut-icon">
              <PieChartIcon size={18} />
            </div>
          </button>
        </div>

        <button
          className={`settings-btn ${showSettings ? 'active' : ''}`}
          onClick={() => setShowSettings(!showSettings)}
        >
          <Settings size={16} />
          <span>Settings</span>
        </button>
      </div>

      {showSettings && (
        <div className="chart-settings">
          <div className="setting-row">
            <label>X-Axis (Group by)</label>
            <select
              value={xAxisProperty}
              onChange={(e) => setXAxisProperty(e.target.value)}
            >
              <option value="">Select property...</option>
              {categoricalProperties.map((prop) => (
                <option key={prop.id} value={prop.id}>
                  {prop.name}
                </option>
              ))}
            </select>
          </div>
          <div className="setting-row">
            <label>Y-Axis (Value)</label>
            <select
              value={yAxisProperty}
              onChange={(e) => setYAxisProperty(e.target.value)}
            >
              <option value="">Count</option>
              {numericProperties.map((prop) => (
                <option key={prop.id} value={prop.id}>
                  {prop.name}
                </option>
              ))}
            </select>
          </div>
        </div>
      )}

      <div className="chart-container">{renderChart()}</div>
    </div>
  )
}
