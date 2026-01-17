import { useEffect, useState, useCallback } from 'react';
import {
  Box,
  Button,
  Group,
  Text,
  Stack,
  Paper,
  Table,
  Select,
  ActionIcon,
  Badge,
  TextInput,
  Loader,
  Center,
  ScrollArea,
  Code,
  Menu,
  NavLink,
  Divider,
  Tooltip,
  Tabs,
  useMantineTheme,
} from '@mantine/core';
import { useDisclosure, useInterval } from '@mantine/hooks';
import {
  IconRefresh,
  IconSearch,
  IconDownload,
  IconChevronRight,
  IconChevronDown,
  IconDatabase,
  IconServer,
  IconShield,
  IconCloud,
  IconBroadcast,
  IconCode,
  IconClock,
  IconX,
  IconChartBar,
  IconPlus,
  IconArrowLeft,
} from '@tabler/icons-react';
import { notifications } from '@mantine/notifications';
import { EmptyState } from '../../components/common/EmptyState';
import { logsApi } from '../../api';
import type { LogEntry, LogSource, LogHistogramBucket, SavedQuery, LogFilter } from '../../api/logs';

// Collection icons mapping
const COLLECTION_ICONS: Record<string, React.ReactNode> = {
  edge: <IconServer size={16} />,
  postgres: <IconDatabase size={16} />,
  postgrest: <IconDatabase size={16} />,
  pooler: <IconDatabase size={16} />,
  auth: <IconShield size={16} />,
  storage: <IconCloud size={16} />,
  realtime: <IconBroadcast size={16} />,
  functions: <IconCode size={16} />,
  cron: <IconClock size={16} />,
};

// Status color mapping
const getStatusColor = (status: number | undefined) => {
  if (!status) return 'gray';
  if (status >= 500) return 'red';
  if (status >= 400) return 'orange';
  if (status >= 300) return 'yellow';
  if (status >= 200) return 'green';
  return 'gray';
};

// Severity color mapping (for Postgres logs)
const getSeverityColor = (severity: string | undefined) => {
  switch (severity) {
    case 'DEBUG': return 'gray';
    case 'INFO': return 'blue';
    case 'NOTICE': return 'cyan';
    case 'WARNING': return 'yellow';
    case 'ERROR': return 'orange';
    case 'FATAL': return 'red';
    case 'PANIC': return 'red';
    default: return 'gray';
  }
};

// Severity levels for filtering
const SEVERITY_LEVELS = ['DEBUG', 'INFO', 'NOTICE', 'WARNING', 'ERROR', 'FATAL', 'PANIC'];

// Format timestamp like Supabase: "17 Jan 26 21:16:26"
const formatTimestamp = (timestamp: string) => {
  const date = new Date(timestamp);
  const day = date.getDate();
  const month = date.toLocaleString('en-US', { month: 'short' });
  const year = String(date.getFullYear()).slice(-2);
  const time = date.toLocaleTimeString('en-US', { hour12: false });
  return `${day} ${month} ${year} ${time}`;
};

// Simple histogram bar component
function HistogramBar({ bucket, maxCount }: { bucket: LogHistogramBucket; maxCount: number }) {
  const theme = useMantineTheme();
  const height = maxCount > 0 ? Math.max(4, (bucket.count / maxCount) * 60) : 0;

  return (
    <Tooltip label={`${bucket.count} events at ${formatTimestamp(bucket.timestamp)}`}>
      <Box
        style={{
          width: 8,
          height: 60,
          display: 'flex',
          alignItems: 'flex-end',
          cursor: 'pointer',
        }}
      >
        <Box
          style={{
            width: '100%',
            height: height,
            backgroundColor: theme.colors.green[5],
            borderRadius: 2,
          }}
        />
      </Box>
    </Tooltip>
  );
}

export function LogsExplorerPage() {
  const theme = useMantineTheme();

  // Data state
  const [logs, setLogs] = useState<LogEntry[]>([]);
  const [sources, setSources] = useState<LogSource[]>([]);
  const [histogram, setHistogram] = useState<LogHistogramBucket[]>([]);
  const [savedQueries, setSavedQueries] = useState<SavedQuery[]>([]);
  const [loading, setLoading] = useState(true);

  // Filter state
  const [selectedSource, setSelectedSource] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [timeRange, setTimeRange] = useState('1h');
  const [statusFilter, setStatusFilter] = useState<string | null>(null);
  const [methodFilter, setMethodFilter] = useState<string | null>(null);
  const [severityFilter, setSeverityFilter] = useState<string | null>(null);

  // UI state
  const [collectionsOpen, { toggle: toggleCollections }] = useDisclosure(true);
  const [queriesOpen, { toggle: toggleQueries }] = useDisclosure(true);
  const [selectedLog, setSelectedLog] = useState<LogEntry | null>(null);
  const [detailTab, setDetailTab] = useState<string | null>('details');
  const [autoRefresh] = useState(false);
  const refreshInterval = useInterval(() => fetchLogs(), 5000);

  // Pagination
  const [offset, setOffset] = useState(0);
  const limit = 25;

  // Build filter object from current state
  const buildFilter = useCallback((): LogFilter => {
    const filter: LogFilter = {
      time_range: timeRange,
      limit,
      offset,
    };
    if (selectedSource) filter.source = selectedSource;
    if (searchQuery) filter.query = searchQuery;
    if (statusFilter) {
      if (statusFilter === '2xx') {
        filter.status_min = 200;
        filter.status_max = 299;
      } else if (statusFilter === '3xx') {
        filter.status_min = 300;
        filter.status_max = 399;
      } else if (statusFilter === '4xx') {
        filter.status_min = 400;
        filter.status_max = 499;
      } else if (statusFilter === '5xx') {
        filter.status_min = 500;
        filter.status_max = 599;
      }
    }
    if (methodFilter) filter.method = methodFilter;
    if (severityFilter) filter.severity = severityFilter as any;
    return filter;
  }, [selectedSource, searchQuery, timeRange, statusFilter, methodFilter, severityFilter, offset]);

  // Fetch functions
  const fetchSources = useCallback(async () => {
    try {
      const data = await logsApi.listSources();
      setSources(data);
    } catch (error: any) {
      console.error('Failed to load sources:', error);
    }
  }, []);

  const fetchLogs = useCallback(async () => {
    setLoading(true);
    try {
      const filter = buildFilter();
      const [logsResponse, histogramResponse] = await Promise.all([
        logsApi.listLogs(filter),
        logsApi.getHistogram(filter, '5m'),
      ]);
      setLogs(logsResponse.logs || []);
      setHistogram(histogramResponse.buckets || []);
    } catch (error: any) {
      notifications.show({
        title: 'Error',
        message: error.message || 'Failed to load logs',
        color: 'red',
      });
    } finally {
      setLoading(false);
    }
  }, [buildFilter]);

  const fetchSavedQueries = useCallback(async () => {
    try {
      const queries = await logsApi.listSavedQueries();
      setSavedQueries(queries || []);
    } catch (error: any) {
      console.error('Failed to load queries:', error);
    }
  }, []);

  // Initial load
  useEffect(() => {
    fetchSources();
    fetchSavedQueries();
  }, [fetchSources, fetchSavedQueries]);

  // Fetch logs when filters change
  useEffect(() => {
    fetchLogs();
  }, [fetchLogs]);

  // Auto-refresh
  useEffect(() => {
    if (autoRefresh) {
      refreshInterval.start();
    } else {
      refreshInterval.stop();
    }
    return () => refreshInterval.stop();
  }, [autoRefresh]);

  // Handle export
  const handleExport = async (format: 'json' | 'csv') => {
    try {
      const blob = await logsApi.exportLogs({
        format,
        source: selectedSource || undefined,
        time_range: timeRange,
      });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `logs-${new Date().toISOString()}.${format}`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
      notifications.show({ title: 'Success', message: 'Logs exported successfully', color: 'green' });
    } catch (error: any) {
      notifications.show({ title: 'Error', message: error.message || 'Failed to export logs', color: 'red' });
    }
  };

  // Clear filters
  const clearFilters = () => {
    setSelectedSource(null);
    setSearchQuery('');
    setStatusFilter(null);
    setMethodFilter(null);
    setSeverityFilter(null);
    setOffset(0);
  };

  // Load older logs
  const loadOlder = () => {
    setOffset((prev) => prev + limit);
  };

  // Calculate histogram max
  const histogramMax = Math.max(...histogram.map((b) => b.count), 1);

  // Check if filters are active
  const hasFilters = selectedSource || searchQuery || statusFilter || methodFilter || severityFilter;

  return (
    <Box style={{ display: 'flex', height: 'calc(100vh - 60px)', overflow: 'hidden' }}>
      {/* Left Sidebar */}
      <Box
        style={{
          width: 240,
          borderRight: `1px solid ${theme.colors.gray[3]}`,
          display: 'flex',
          flexDirection: 'column',
          backgroundColor: theme.white,
        }}
      >
        <Box p="md">
          <Text fw={600} size="lg">Logs & Analytics</Text>
        </Box>

        <ScrollArea style={{ flex: 1 }} p="xs">
          {/* New Logs Coming Soon */}
          <Paper withBorder p="sm" mb="md" style={{ backgroundColor: theme.colors.gray[0] }}>
            <Badge size="xs" mb="xs">COMING SOON</Badge>
            <Text size="sm" fw={500}>New logs</Text>
            <Text size="xs" c="dimmed" mb="xs">Get early access</Text>
            <Button size="xs" variant="outline">Early access</Button>
          </Paper>

          {/* Templates */}
          <Box mb="md">
            <Text size="sm" fw={500} c="dimmed" mb="xs">Templates</Text>
            <TextInput
              placeholder="Search collections..."
              size="xs"
              rightSection={<IconPlus size={14} />}
            />
          </Box>

          <Divider my="sm" />

          {/* Collections */}
          <NavLink
            label="COLLECTIONS"
            childrenOffset={0}
            defaultOpened
            opened={collectionsOpen}
            onClick={toggleCollections}
            rightSection={collectionsOpen ? <IconChevronDown size={14} /> : <IconChevronRight size={14} />}
            styles={{ label: { fontSize: 11, fontWeight: 600, color: theme.colors.gray[6] } }}
          />
          {collectionsOpen && (
            <Box ml="xs">
              {sources.map((source) => (
                <NavLink
                  key={source.id}
                  label={source.name}
                  leftSection={COLLECTION_ICONS[source.id] || <IconServer size={16} />}
                  active={selectedSource === source.id}
                  onClick={() => {
                    setSelectedSource(selectedSource === source.id ? null : source.id);
                    setOffset(0);
                  }}
                  styles={{
                    root: { borderRadius: 4, padding: '6px 8px' },
                    label: { fontSize: 13 },
                  }}
                />
              ))}
            </Box>
          )}

          <Divider my="sm" />

          {/* Database Operations (placeholder) */}
          <NavLink
            label="DATABASE OPERATIONS"
            childrenOffset={0}
            defaultOpened={false}
            styles={{ label: { fontSize: 11, fontWeight: 600, color: theme.colors.gray[6] } }}
          >
            <NavLink
              label="Postgres Version Upgrade"
              leftSection={<IconDatabase size={16} />}
              styles={{ root: { borderRadius: 4 }, label: { fontSize: 13 } }}
            />
          </NavLink>

          <Divider my="sm" />

          {/* Saved Queries */}
          <NavLink
            label="QUERIES"
            childrenOffset={0}
            opened={queriesOpen}
            onClick={toggleQueries}
            rightSection={queriesOpen ? <IconChevronDown size={14} /> : <IconChevronRight size={14} />}
            styles={{ label: { fontSize: 11, fontWeight: 600, color: theme.colors.gray[6] } }}
          />
          {queriesOpen && (
            <Box ml="xs" p="sm">
              {savedQueries.length === 0 ? (
                <Box ta="center" py="md">
                  <Text size="xs" c="dimmed" mb="xs">No queries created yet</Text>
                  <Text size="xs" c="dimmed" mb="sm">Create and save your queries to use them in the explorer</Text>
                  <Button size="xs" variant="outline">Create query</Button>
                </Box>
              ) : (
                savedQueries.map((q) => (
                  <NavLink
                    key={q.id}
                    label={q.name}
                    styles={{ root: { borderRadius: 4 }, label: { fontSize: 13 } }}
                  />
                ))
              )}
            </Box>
          )}
        </ScrollArea>
      </Box>

      {/* Main Content */}
      <Box style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
        {/* Toolbar */}
        <Box p="sm" style={{ borderBottom: `1px solid ${theme.colors.gray[3]}` }}>
          <Group justify="space-between">
            <Group gap="sm">
              <TextInput
                placeholder="Search events"
                value={searchQuery}
                onChange={(e) => {
                  setSearchQuery(e.target.value);
                  setOffset(0);
                }}
                leftSection={<IconSearch size={16} />}
                w={200}
                size="sm"
              />
              <ActionIcon variant="subtle" onClick={fetchLogs}>
                <IconRefresh size={18} />
              </ActionIcon>
              <Select
                value={timeRange}
                onChange={(val) => {
                  setTimeRange(val || '1h');
                  setOffset(0);
                }}
                data={[
                  { value: '1h', label: 'Last hour' },
                  { value: '24h', label: 'Last 24 hours' },
                  { value: '7d', label: 'Last 7 days' },
                  { value: '30d', label: 'Last 30 days' },
                ]}
                size="sm"
                w={130}
              />
              <Select
                placeholder="Status"
                value={statusFilter}
                onChange={(val) => {
                  setStatusFilter(val);
                  setOffset(0);
                }}
                data={[
                  { value: '2xx', label: '2xx Success' },
                  { value: '3xx', label: '3xx Redirect' },
                  { value: '4xx', label: '4xx Client Error' },
                  { value: '5xx', label: '5xx Server Error' },
                ]}
                size="sm"
                w={130}
                clearable
              />
              <Select
                placeholder="Method"
                value={methodFilter}
                onChange={(val) => {
                  setMethodFilter(val);
                  setOffset(0);
                }}
                data={['GET', 'POST', 'PUT', 'PATCH', 'DELETE', 'OPTIONS', 'HEAD']}
                size="sm"
                w={100}
                clearable
              />
              <Select
                placeholder="Severity"
                value={severityFilter}
                onChange={(val) => {
                  setSeverityFilter(val);
                  setOffset(0);
                }}
                data={SEVERITY_LEVELS.map(s => ({ value: s, label: s }))}
                size="sm"
                w={110}
                clearable
              />
              <ActionIcon variant="subtle">
                <IconChartBar size={18} />
              </ActionIcon>
              <Menu position="bottom-end">
                <Menu.Target>
                  <ActionIcon variant="subtle">
                    <IconDownload size={18} />
                  </ActionIcon>
                </Menu.Target>
                <Menu.Dropdown>
                  <Menu.Item onClick={() => handleExport('json')}>Export as JSON</Menu.Item>
                  <Menu.Item onClick={() => handleExport('csv')}>Export as CSV</Menu.Item>
                </Menu.Dropdown>
              </Menu>
            </Group>
            <Group gap="sm">
              <Select
                value="primary"
                data={[{ value: 'primary', label: 'Primary Database' }]}
                size="sm"
                w={150}
              />
            </Group>
          </Group>
        </Box>

        {/* Histogram */}
        <Box px="sm" py="xs" style={{ borderBottom: `1px solid ${theme.colors.gray[2]}` }}>
          <Group gap={2} align="flex-end" style={{ height: 70 }}>
            {histogram.length > 0 ? (
              histogram.map((bucket, idx) => (
                <HistogramBar key={idx} bucket={bucket} maxCount={histogramMax} />
              ))
            ) : (
              <Text size="xs" c="dimmed">No data for histogram</Text>
            )}
          </Group>
          <Group justify="space-between" mt="xs">
            <Text size="xs" c="dimmed">{histogram[0] ? formatTimestamp(histogram[0].timestamp) : ''}</Text>
            <Text size="xs" c="dimmed">{histogram[histogram.length - 1] ? formatTimestamp(histogram[histogram.length - 1].timestamp) : ''}</Text>
          </Group>
        </Box>

        {/* Content Area */}
        <Box style={{ flex: 1, display: 'flex', overflow: 'hidden' }}>
          {/* Logs Table */}
          <Box style={{ flex: 1, overflow: 'hidden', display: 'flex', flexDirection: 'column' }}>
            {loading && logs.length === 0 ? (
              <Center py="xl" style={{ flex: 1 }}>
                <Loader size="lg" />
              </Center>
            ) : logs.length === 0 ? (
              <Center style={{ flex: 1 }}>
                <EmptyState
                  icon={<IconSearch size={48} />}
                  title="No logs found"
                  description={hasFilters ? 'Try adjusting your filters' : 'Logs will appear here as they are generated'}
                  action={hasFilters ? { label: 'Clear filters', onClick: clearFilters } : undefined}
                />
              </Center>
            ) : (
              <>
                <ScrollArea style={{ flex: 1 }}>
                  <Table striped highlightOnHover stickyHeader>
                    <Table.Thead>
                      <Table.Tr>
                        <Table.Th w={160}>Timestamp</Table.Th>
                        <Table.Th w={80}>Severity</Table.Th>
                        <Table.Th w={60}>Status</Table.Th>
                        <Table.Th w={70}>Method</Table.Th>
                        <Table.Th>Path / Message</Table.Th>
                      </Table.Tr>
                    </Table.Thead>
                    <Table.Tbody>
                      {logs.map((log) => (
                        <Table.Tr
                          key={log.id}
                          style={{
                            cursor: 'pointer',
                            backgroundColor: selectedLog?.id === log.id ? theme.colors.blue[0] : undefined,
                          }}
                          onClick={() => setSelectedLog(log)}
                        >
                          <Table.Td>
                            <Text size="sm" c="dimmed" style={{ fontFamily: 'monospace' }}>
                              {formatTimestamp(log.timestamp)}
                            </Text>
                          </Table.Td>
                          <Table.Td>
                            {log.severity ? (
                              <Badge size="xs" color={getSeverityColor(log.severity)} variant="light">
                                {log.severity}
                              </Badge>
                            ) : (
                              <Text size="sm" c="dimmed">-</Text>
                            )}
                          </Table.Td>
                          <Table.Td>
                            <Text size="sm" c={getStatusColor(log.status_code)} fw={500}>
                              {log.status_code || '-'}
                            </Text>
                          </Table.Td>
                          <Table.Td>
                            <Text size="sm">{log.method || '-'}</Text>
                          </Table.Td>
                          <Table.Td>
                            <Text size="sm" truncate style={{ maxWidth: 350 }}>
                              {log.path || log.event_message || '-'}
                            </Text>
                          </Table.Td>
                        </Table.Tr>
                      ))}
                    </Table.Tbody>
                  </Table>
                </ScrollArea>
                <Box p="sm" style={{ borderTop: `1px solid ${theme.colors.gray[3]}` }}>
                  <Group justify="space-between">
                    <Button
                      variant="subtle"
                      size="xs"
                      leftSection={<IconArrowLeft size={14} />}
                      onClick={loadOlder}
                      disabled={logs.length < limit}
                    >
                      Load older
                    </Button>
                    <Text size="xs" c="dimmed">Showing {logs.length} results</Text>
                  </Group>
                </Box>
              </>
            )}
          </Box>

          {/* Detail Panel */}
          {selectedLog && (
            <Box
              style={{
                width: 400,
                borderLeft: `1px solid ${theme.colors.gray[3]}`,
                display: 'flex',
                flexDirection: 'column',
                overflow: 'hidden',
              }}
            >
              <Group justify="space-between" p="sm" style={{ borderBottom: `1px solid ${theme.colors.gray[3]}` }}>
                <Tabs value={detailTab} onChange={setDetailTab}>
                  <Tabs.List>
                    <Tabs.Tab value="details">Details</Tabs.Tab>
                    <Tabs.Tab value="raw">Raw</Tabs.Tab>
                  </Tabs.List>
                </Tabs>
                <ActionIcon variant="subtle" onClick={() => setSelectedLog(null)}>
                  <IconX size={16} />
                </ActionIcon>
              </Group>
              <ScrollArea style={{ flex: 1 }} p="sm">
                {detailTab === 'details' ? (
                  <Stack gap="md">
                    <Box>
                      <Text size="xs" c="dimmed" mb={4}>id</Text>
                      <Code style={{ wordBreak: 'break-all' }}>{selectedLog.id}</Code>
                    </Box>
                    <Box>
                      <Text size="xs" c="dimmed" mb={4}>source</Text>
                      <Badge size="sm" variant="light">{selectedLog.source}</Badge>
                    </Box>
                    {selectedLog.severity && (
                      <Box>
                        <Text size="xs" c="dimmed" mb={4}>severity</Text>
                        <Badge size="sm" color={getSeverityColor(selectedLog.severity)} variant="light">
                          {selectedLog.severity}
                        </Badge>
                      </Box>
                    )}
                    <Box>
                      <Text size="xs" c="dimmed" mb={4}>status</Text>
                      <Text c={getStatusColor(selectedLog.status_code)} fw={500}>
                        {selectedLog.status_code || '-'}
                      </Text>
                    </Box>
                    <Box>
                      <Text size="xs" c="dimmed" mb={4}>timestamp</Text>
                      <Text>{formatTimestamp(selectedLog.timestamp)}</Text>
                    </Box>
                    <Box>
                      <Text size="xs" c="dimmed" mb={4}>method</Text>
                      <Text>{selectedLog.method || '-'}</Text>
                    </Box>
                    <Box>
                      <Text size="xs" c="dimmed" mb={4}>path</Text>
                      <Code style={{ wordBreak: 'break-all' }}>{selectedLog.path || '-'}</Code>
                    </Box>
                    {selectedLog.request_id && (
                      <Box>
                        <Text size="xs" c="dimmed" mb={4}>request_id</Text>
                        <Code style={{ wordBreak: 'break-all' }}>{selectedLog.request_id}</Code>
                      </Box>
                    )}
                    {selectedLog.duration_ms !== undefined && selectedLog.duration_ms > 0 && (
                      <Box>
                        <Text size="xs" c="dimmed" mb={4}>duration_ms</Text>
                        <Text>{selectedLog.duration_ms}ms</Text>
                      </Box>
                    )}
                    <Box>
                      <Text size="xs" c="dimmed" mb={4}>user_agent</Text>
                      <Code style={{ wordBreak: 'break-all' }}>{selectedLog.user_agent || '-'}</Code>
                    </Box>
                    <Box>
                      <Text size="xs" c="dimmed" mb={4}>apikey</Text>
                      <Code style={{ wordBreak: 'break-all' }}>{selectedLog.apikey || '-'}</Code>
                    </Box>
                    <Box>
                      <Text size="xs" c="dimmed" mb={4}>event_message</Text>
                      <Paper withBorder p="sm" style={{ backgroundColor: theme.colors.gray[0] }}>
                        <Code style={{ whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}>
                          {selectedLog.event_message || '-'}
                        </Code>
                      </Paper>
                    </Box>
                    {selectedLog.metadata && Object.keys(selectedLog.metadata).length > 0 && (
                      <Box>
                        <Text size="xs" c="dimmed" mb={4}>metadata</Text>
                        <Paper withBorder p="sm" style={{ backgroundColor: theme.colors.gray[0] }}>
                          <Code style={{ whiteSpace: 'pre-wrap' }}>
                            {JSON.stringify(selectedLog.metadata, null, 2)}
                          </Code>
                        </Paper>
                      </Box>
                    )}
                    {selectedLog.request_headers && Object.keys(selectedLog.request_headers).length > 0 && (
                      <Box>
                        <Text size="xs" c="dimmed" mb={4}>request_headers</Text>
                        <Paper withBorder p="sm" style={{ backgroundColor: theme.colors.gray[0] }}>
                          <Code style={{ whiteSpace: 'pre-wrap' }}>
                            {JSON.stringify(selectedLog.request_headers, null, 2)}
                          </Code>
                        </Paper>
                      </Box>
                    )}
                    <Button variant="subtle" fullWidth>Expand</Button>
                  </Stack>
                ) : (
                  <Code block style={{ whiteSpace: 'pre-wrap' }}>
                    {JSON.stringify(selectedLog, null, 2)}
                  </Code>
                )}
              </ScrollArea>
            </Box>
          )}
        </Box>
      </Box>
    </Box>
  );
}
