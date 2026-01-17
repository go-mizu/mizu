import { useState, useEffect } from 'react';
import {
  Box,
  Title,
  Text,
  Paper,
  Group,
  Stack,
  Tabs,
  Select,
  ActionIcon,
  Tooltip,
  LoadingOverlay,
  Badge,
  SimpleGrid,
  Skeleton,
} from '@mantine/core';
import {
  IconRefresh,
  IconDatabase,
  IconUsers,
  IconFolder,
  IconBroadcast,
  IconCode,
  IconApi,
  IconTrendingUp,
  IconTrendingDown,
} from '@tabler/icons-react';
import {
  LineChart,
  Line,
  AreaChart,
  Area,
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip as RechartsTooltip,
  Legend,
  ResponsiveContainer,
} from 'recharts';
import { reportsApi, Report, ReportType, ChartData, MetricDataPoint } from '../../api/reports';

// Time range options
const timeRangeOptions = [
  { value: '1h', label: 'Last hour' },
  { value: '3h', label: 'Last 3 hours' },
  { value: '6h', label: 'Last 6 hours' },
  { value: '12h', label: 'Last 12 hours' },
  { value: '24h', label: 'Last 24 hours' },
  { value: '7d', label: 'Last 7 days' },
  { value: '30d', label: 'Last 30 days' },
];

// Report type icons
const reportTypeIcons: Record<string, typeof IconDatabase> = {
  database: IconDatabase,
  auth: IconUsers,
  storage: IconFolder,
  realtime: IconBroadcast,
  functions: IconCode,
  api: IconApi,
};

// Chart colors
const chartColors = {
  primary: '#3ECF8E',
  secondary: '#6366F1',
  tertiary: '#F59E0B',
  quaternary: '#EF4444',
  error: '#EF4444',
  success: '#10B981',
};

// Format timestamp for display
function formatTimestamp(timestamp: string): string {
  const date = new Date(timestamp);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffHours = diffMs / (1000 * 60 * 60);

  if (diffHours < 24) {
    return date.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' });
  }
  return date.toLocaleDateString(undefined, { month: 'short', day: 'numeric' });
}

// Format value based on unit
function formatValue(value: number, unit: string): string {
  if (unit === '%') {
    return `${value.toFixed(1)}%`;
  }
  if (unit === 'ms') {
    return `${value.toFixed(0)}ms`;
  }
  if (unit === 'bytes') {
    if (value >= 1024 * 1024) {
      return `${(value / (1024 * 1024)).toFixed(1)} MB`;
    }
    if (value >= 1024) {
      return `${(value / 1024).toFixed(1)} KB`;
    }
    return `${value} B`;
  }
  if (value >= 1000000) {
    return `${(value / 1000000).toFixed(1)}M`;
  }
  if (value >= 1000) {
    return `${(value / 1000).toFixed(1)}K`;
  }
  return value.toFixed(0);
}

// Calculate trend from data points
function calculateTrend(data: MetricDataPoint[]): { change: number; isPositive: boolean } {
  if (data.length < 2) return { change: 0, isPositive: true };

  const midpoint = Math.floor(data.length / 2);
  const firstHalf = data.slice(0, midpoint);
  const secondHalf = data.slice(midpoint);

  const firstAvg = firstHalf.reduce((sum, dp) => sum + dp.value, 0) / firstHalf.length;
  const secondAvg = secondHalf.reduce((sum, dp) => sum + dp.value, 0) / secondHalf.length;

  if (firstAvg === 0) return { change: 0, isPositive: true };

  const change = ((secondAvg - firstAvg) / firstAvg) * 100;
  return { change: Math.abs(change), isPositive: change >= 0 };
}

// Chart component
interface ChartCardProps {
  chart: ChartData;
}

function ChartCard({ chart }: ChartCardProps) {
  const chartData = chart.data.map((dp) => ({
    ...dp,
    timestamp: formatTimestamp(dp.timestamp),
    ...dp.values,
  }));

  // Get current value (last data point)
  const currentValue = chart.data.length > 0 ? chart.data[chart.data.length - 1].value : 0;
  const trend = calculateTrend(chart.data);

  // Determine which keys to use for multi-series charts
  const hasMultipleMetrics = chart.metrics && chart.metrics.length > 0;
  const metricKeys = hasMultipleMetrics
    ? chart.metrics!.map((m) => m.split('.').pop() || m)
    : ['value'];

  // Render chart based on type
  const renderChart = () => {
    switch (chart.type) {
      case 'line':
        return (
          <ResponsiveContainer width="100%" height={200}>
            <LineChart data={chartData}>
              <CartesianGrid strokeDasharray="3 3" stroke="var(--supabase-border)" />
              <XAxis
                dataKey="timestamp"
                stroke="var(--supabase-text-muted)"
                fontSize={11}
                tickLine={false}
              />
              <YAxis
                stroke="var(--supabase-text-muted)"
                fontSize={11}
                tickLine={false}
                width={50}
                tickFormatter={(v) => formatValue(v, chart.unit)}
              />
              <RechartsTooltip
                contentStyle={{
                  backgroundColor: 'var(--supabase-bg-surface)',
                  border: '1px solid var(--supabase-border)',
                  borderRadius: 8,
                }}
                labelStyle={{ color: 'var(--supabase-text)' }}
              />
              {metricKeys.map((key, index) => (
                <Line
                  key={key}
                  type="monotone"
                  dataKey={hasMultipleMetrics ? key : 'value'}
                  stroke={Object.values(chartColors)[index % Object.values(chartColors).length]}
                  strokeWidth={2}
                  dot={false}
                  name={key}
                />
              ))}
            </LineChart>
          </ResponsiveContainer>
        );

      case 'area':
      case 'stacked_area':
        return (
          <ResponsiveContainer width="100%" height={200}>
            <AreaChart data={chartData}>
              <CartesianGrid strokeDasharray="3 3" stroke="var(--supabase-border)" />
              <XAxis
                dataKey="timestamp"
                stroke="var(--supabase-text-muted)"
                fontSize={11}
                tickLine={false}
              />
              <YAxis
                stroke="var(--supabase-text-muted)"
                fontSize={11}
                tickLine={false}
                width={50}
                tickFormatter={(v) => formatValue(v, chart.unit)}
              />
              <RechartsTooltip
                contentStyle={{
                  backgroundColor: 'var(--supabase-bg-surface)',
                  border: '1px solid var(--supabase-border)',
                  borderRadius: 8,
                }}
                labelStyle={{ color: 'var(--supabase-text)' }}
              />
              {metricKeys.map((key, index) => (
                <Area
                  key={key}
                  type="monotone"
                  dataKey={hasMultipleMetrics ? key : 'value'}
                  stackId={chart.type === 'stacked_area' ? '1' : undefined}
                  stroke={Object.values(chartColors)[index % Object.values(chartColors).length]}
                  fill={Object.values(chartColors)[index % Object.values(chartColors).length]}
                  fillOpacity={0.3}
                  name={key}
                />
              ))}
            </AreaChart>
          </ResponsiveContainer>
        );

      case 'bar':
      case 'stacked_bar':
        return (
          <ResponsiveContainer width="100%" height={200}>
            <BarChart data={chartData}>
              <CartesianGrid strokeDasharray="3 3" stroke="var(--supabase-border)" />
              <XAxis
                dataKey="timestamp"
                stroke="var(--supabase-text-muted)"
                fontSize={11}
                tickLine={false}
              />
              <YAxis
                stroke="var(--supabase-text-muted)"
                fontSize={11}
                tickLine={false}
                width={50}
                tickFormatter={(v) => formatValue(v, chart.unit)}
              />
              <RechartsTooltip
                contentStyle={{
                  backgroundColor: 'var(--supabase-bg-surface)',
                  border: '1px solid var(--supabase-border)',
                  borderRadius: 8,
                }}
                labelStyle={{ color: 'var(--supabase-text)' }}
              />
              <Legend />
              {metricKeys.map((key, index) => (
                <Bar
                  key={key}
                  dataKey={hasMultipleMetrics ? key : 'value'}
                  stackId={chart.type === 'stacked_bar' ? '1' : undefined}
                  fill={Object.values(chartColors)[index % Object.values(chartColors).length]}
                  name={key}
                  radius={[4, 4, 0, 0]}
                />
              ))}
            </BarChart>
          </ResponsiveContainer>
        );

      default:
        return (
          <ResponsiveContainer width="100%" height={200}>
            <LineChart data={chartData}>
              <CartesianGrid strokeDasharray="3 3" stroke="var(--supabase-border)" />
              <XAxis dataKey="timestamp" stroke="var(--supabase-text-muted)" fontSize={11} />
              <YAxis stroke="var(--supabase-text-muted)" fontSize={11} width={50} />
              <RechartsTooltip />
              <Line type="monotone" dataKey="value" stroke={chartColors.primary} strokeWidth={2} dot={false} />
            </LineChart>
          </ResponsiveContainer>
        );
    }
  };

  return (
    <Paper
      p="md"
      radius="md"
      style={{
        backgroundColor: 'var(--supabase-bg)',
        border: '1px solid var(--supabase-border)',
      }}
    >
      <Stack gap="md">
        <Group justify="space-between">
          <Text size="sm" fw={500} c="dimmed">
            {chart.title}
          </Text>
          {!hasMultipleMetrics && (
            <Group gap="xs">
              <Text size="lg" fw={600}>
                {formatValue(currentValue, chart.unit)}
              </Text>
              {trend.change > 0.1 && (
                <Badge
                  size="sm"
                  variant="light"
                  color={trend.isPositive ? 'green' : 'red'}
                  leftSection={
                    trend.isPositive ? <IconTrendingUp size={12} /> : <IconTrendingDown size={12} />
                  }
                >
                  {trend.change.toFixed(1)}%
                </Badge>
              )}
            </Group>
          )}
        </Group>
        {renderChart()}
      </Stack>
    </Paper>
  );
}

// Loading skeleton for charts
function ChartSkeleton() {
  return (
    <Paper
      p="md"
      radius="md"
      style={{
        backgroundColor: 'var(--supabase-bg)',
        border: '1px solid var(--supabase-border)',
      }}
    >
      <Stack gap="md">
        <Group justify="space-between">
          <Skeleton height={16} width={120} />
          <Skeleton height={24} width={80} />
        </Group>
        <Skeleton height={200} />
      </Stack>
    </Paper>
  );
}

export function ReportsPage() {
  const [reportTypes, setReportTypes] = useState<ReportType[]>([]);
  const [activeTab, setActiveTab] = useState<string | null>(null);
  const [timeRange, setTimeRange] = useState('24h');
  const [report, setReport] = useState<Report | null>(null);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);

  // Load report types
  useEffect(() => {
    async function loadReportTypes() {
      try {
        const types = await reportsApi.listReportTypes();
        setReportTypes(types);
        if (types.length > 0 && !activeTab) {
          setActiveTab(types[0].id);
        }
      } catch (error) {
        console.error('Failed to load report types:', error);
      }
    }
    loadReportTypes();
  }, []);

  // Load report data when tab or time range changes
  useEffect(() => {
    async function loadReport() {
      if (!activeTab) return;

      setLoading(true);
      try {
        const data = await reportsApi.getReport(activeTab, { time_range: timeRange });
        setReport(data);
      } catch (error) {
        console.error('Failed to load report:', error);
      } finally {
        setLoading(false);
      }
    }
    loadReport();
  }, [activeTab, timeRange]);

  // Refresh data
  const handleRefresh = async () => {
    if (!activeTab) return;

    setRefreshing(true);
    try {
      const data = await reportsApi.getReport(activeTab, { time_range: timeRange });
      setReport(data);
    } catch (error) {
      console.error('Failed to refresh report:', error);
    } finally {
      setRefreshing(false);
    }
  };

  return (
    <Box p="lg">
      <Stack gap="lg">
        {/* Header */}
        <Group justify="space-between">
          <Box>
            <Title order={2} style={{ color: 'var(--supabase-text)' }}>
              Reports
            </Title>
            <Text size="sm" c="dimmed">
              Monitor your project's performance metrics and health
            </Text>
          </Box>
          <Group>
            <Select
              size="sm"
              value={timeRange}
              onChange={(value) => value && setTimeRange(value)}
              data={timeRangeOptions}
              w={150}
              styles={{
                input: {
                  backgroundColor: 'var(--supabase-bg)',
                  borderColor: 'var(--supabase-border)',
                },
              }}
            />
            <Tooltip label="Refresh data">
              <ActionIcon
                variant="subtle"
                onClick={handleRefresh}
                loading={refreshing}
                style={{ color: 'var(--supabase-text-muted)' }}
              >
                <IconRefresh size={18} />
              </ActionIcon>
            </Tooltip>
          </Group>
        </Group>

        {/* Report Type Tabs */}
        <Tabs value={activeTab} onChange={setActiveTab}>
          <Tabs.List
            style={{
              borderBottom: '1px solid var(--supabase-border)',
              gap: 0,
            }}
          >
            {reportTypes.map((type) => {
              const Icon = reportTypeIcons[type.id] || IconDatabase;
              return (
                <Tabs.Tab
                  key={type.id}
                  value={type.id}
                  leftSection={<Icon size={16} />}
                  style={{
                    color:
                      activeTab === type.id
                        ? 'var(--supabase-brand)'
                        : 'var(--supabase-text-muted)',
                    borderBottom:
                      activeTab === type.id ? '2px solid var(--supabase-brand)' : 'none',
                  }}
                >
                  {type.name}
                </Tabs.Tab>
              );
            })}
          </Tabs.List>
        </Tabs>

        {/* Charts Grid */}
        <Box pos="relative">
          <LoadingOverlay visible={loading && !report} />

          {report && (
            <SimpleGrid cols={{ base: 1, md: 2 }} spacing="md">
              {report.charts.map((chart) => (
                <ChartCard key={chart.id} chart={chart} />
              ))}
            </SimpleGrid>
          )}

          {loading && !report && (
            <SimpleGrid cols={{ base: 1, md: 2 }} spacing="md">
              {[1, 2, 3, 4].map((i) => (
                <ChartSkeleton key={i} />
              ))}
            </SimpleGrid>
          )}
        </Box>

        {/* Report Info */}
        {report && (
          <Text size="xs" c="dimmed" ta="center">
            Data from {new Date(report.from).toLocaleString()} to{' '}
            {new Date(report.to).toLocaleString()} (interval: {report.interval})
          </Text>
        )}
      </Stack>
    </Box>
  );
}
