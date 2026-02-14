import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip as RechartsTooltip,
  ResponsiveContainer,
  PieChart,
  Pie,
  Cell,
  LineChart,
  Line,
  AreaChart,
  Area,
  Legend,
} from "recharts"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { cn } from "@/lib/utils"

const CHART_COLORS = [
  "#10a37f",
  "#6366f1",
  "#f59e0b",
  "#ef4444",
  "#8b5cf6",
  "#06b6d4",
  "#ec4899",
  "#14b8a6",
  "#f97316",
  "#64748b",
]

interface ChartCardProps {
  title: string
  description?: string
  className?: string
  children: React.ReactNode
}

function ChartCard({ title, description, className, children }: ChartCardProps) {
  return (
    <Card className={cn("", className)}>
      <CardHeader className="pb-2">
        <CardTitle className="text-base">{title}</CardTitle>
        {description && (
          <p className="text-xs text-muted-foreground">{description}</p>
        )}
      </CardHeader>
      <CardContent>
        <div className="h-64">{children}</div>
      </CardContent>
    </Card>
  )
}

// ---

interface BarChartCardProps {
  title: string
  description?: string
  data: Array<Record<string, unknown>>
  dataKey: string
  nameKey: string
  color?: string
  className?: string
}

export function BarChartCard({
  title,
  description,
  data,
  dataKey,
  nameKey,
  color = CHART_COLORS[0],
  className,
}: BarChartCardProps) {
  return (
    <ChartCard title={title} description={description} className={className}>
      <ResponsiveContainer width="100%" height="100%">
        <BarChart data={data} margin={{ top: 4, right: 4, bottom: 4, left: 4 }}>
          <CartesianGrid strokeDasharray="3 3" stroke="var(--color-border)" />
          <XAxis
            dataKey={nameKey}
            tick={{ fontSize: 11, fill: "var(--color-muted-foreground)" }}
            tickLine={false}
            axisLine={{ stroke: "var(--color-border)" }}
          />
          <YAxis
            tick={{ fontSize: 11, fill: "var(--color-muted-foreground)" }}
            tickLine={false}
            axisLine={false}
          />
          <RechartsTooltip
            contentStyle={{
              background: "var(--color-popover)",
              border: "1px solid var(--color-border)",
              borderRadius: "var(--radius-md)",
              fontSize: 12,
              color: "var(--color-popover-foreground)",
            }}
          />
          <Bar dataKey={dataKey} fill={color} radius={[4, 4, 0, 0]} />
        </BarChart>
      </ResponsiveContainer>
    </ChartCard>
  )
}

// ---

interface PieChartCardProps {
  title: string
  description?: string
  data: Array<{ name: string; value: number }>
  className?: string
}

export function PieChartCard({
  title,
  description,
  data,
  className,
}: PieChartCardProps) {
  return (
    <ChartCard title={title} description={description} className={className}>
      <ResponsiveContainer width="100%" height="100%">
        <PieChart>
          <Pie
            data={data}
            cx="50%"
            cy="50%"
            innerRadius={50}
            outerRadius={80}
            paddingAngle={2}
            dataKey="value"
            nameKey="name"
            label={({ name, percent }) =>
              `${name} ${(percent * 100).toFixed(0)}%`
            }
            labelLine={false}
          >
            {data.map((_, i) => (
              <Cell
                key={`cell-${i}`}
                fill={CHART_COLORS[i % CHART_COLORS.length]}
              />
            ))}
          </Pie>
          <RechartsTooltip
            contentStyle={{
              background: "var(--color-popover)",
              border: "1px solid var(--color-border)",
              borderRadius: "var(--radius-md)",
              fontSize: 12,
              color: "var(--color-popover-foreground)",
            }}
          />
          <Legend
            wrapperStyle={{ fontSize: 11 }}
            iconType="circle"
            iconSize={8}
          />
        </PieChart>
      </ResponsiveContainer>
    </ChartCard>
  )
}

// ---

interface LineChartCardProps {
  title: string
  description?: string
  data: Array<Record<string, unknown>>
  lines: Array<{ dataKey: string; color?: string; name?: string }>
  xDataKey: string
  className?: string
}

export function LineChartCard({
  title,
  description,
  data,
  lines,
  xDataKey,
  className,
}: LineChartCardProps) {
  return (
    <ChartCard title={title} description={description} className={className}>
      <ResponsiveContainer width="100%" height="100%">
        <LineChart
          data={data}
          margin={{ top: 4, right: 4, bottom: 4, left: 4 }}
        >
          <CartesianGrid strokeDasharray="3 3" stroke="var(--color-border)" />
          <XAxis
            dataKey={xDataKey}
            tick={{ fontSize: 11, fill: "var(--color-muted-foreground)" }}
            tickLine={false}
            axisLine={{ stroke: "var(--color-border)" }}
          />
          <YAxis
            tick={{ fontSize: 11, fill: "var(--color-muted-foreground)" }}
            tickLine={false}
            axisLine={false}
          />
          <RechartsTooltip
            contentStyle={{
              background: "var(--color-popover)",
              border: "1px solid var(--color-border)",
              borderRadius: "var(--radius-md)",
              fontSize: 12,
              color: "var(--color-popover-foreground)",
            }}
          />
          {lines.map((line, i) => (
            <Line
              key={line.dataKey}
              type="monotone"
              dataKey={line.dataKey}
              name={line.name || line.dataKey}
              stroke={line.color || CHART_COLORS[i % CHART_COLORS.length]}
              strokeWidth={2}
              dot={false}
              activeDot={{ r: 4 }}
            />
          ))}
          {lines.length > 1 && (
            <Legend
              wrapperStyle={{ fontSize: 11 }}
              iconType="line"
              iconSize={12}
            />
          )}
        </LineChart>
      </ResponsiveContainer>
    </ChartCard>
  )
}

// ---

interface AreaChartCardProps {
  title: string
  description?: string
  data: Array<Record<string, unknown>>
  areas: Array<{ dataKey: string; color?: string; name?: string }>
  xDataKey: string
  className?: string
  stacked?: boolean
}

export function AreaChartCard({
  title,
  description,
  data,
  areas,
  xDataKey,
  className,
  stacked = false,
}: AreaChartCardProps) {
  return (
    <ChartCard title={title} description={description} className={className}>
      <ResponsiveContainer width="100%" height="100%">
        <AreaChart
          data={data}
          margin={{ top: 4, right: 4, bottom: 4, left: 4 }}
        >
          <CartesianGrid strokeDasharray="3 3" stroke="var(--color-border)" />
          <XAxis
            dataKey={xDataKey}
            tick={{ fontSize: 11, fill: "var(--color-muted-foreground)" }}
            tickLine={false}
            axisLine={{ stroke: "var(--color-border)" }}
          />
          <YAxis
            tick={{ fontSize: 11, fill: "var(--color-muted-foreground)" }}
            tickLine={false}
            axisLine={false}
          />
          <RechartsTooltip
            contentStyle={{
              background: "var(--color-popover)",
              border: "1px solid var(--color-border)",
              borderRadius: "var(--radius-md)",
              fontSize: 12,
              color: "var(--color-popover-foreground)",
            }}
          />
          {areas.map((area, i) => {
            const color =
              area.color || CHART_COLORS[i % CHART_COLORS.length]
            return (
              <Area
                key={area.dataKey}
                type="monotone"
                dataKey={area.dataKey}
                name={area.name || area.dataKey}
                stroke={color}
                fill={color}
                fillOpacity={0.15}
                strokeWidth={2}
                stackId={stacked ? "stack" : undefined}
              />
            )
          })}
          {areas.length > 1 && (
            <Legend
              wrapperStyle={{ fontSize: 11 }}
              iconType="circle"
              iconSize={8}
            />
          )}
        </AreaChart>
      </ResponsiveContainer>
    </ChartCard>
  )
}
