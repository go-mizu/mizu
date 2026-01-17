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
  Drawer,
  SegmentedControl,
  Switch,
  Menu,
} from '@mantine/core';
import { useDisclosure, useInterval } from '@mantine/hooks';
import {
  IconRefresh,
  IconSearch,
  IconDownload,
  IconX,
  IconChevronRight,
  IconAlertCircle,
  IconAlertTriangle,
  IconInfoCircle,
  IconBug,
} from '@tabler/icons-react';
import { notifications } from '@mantine/notifications';
import { PageContainer } from '../../components/layout/PageContainer';
import { EmptyState } from '../../components/common/EmptyState';
import { logsApi } from '../../api';
import type { Log, LogType } from '../../api/logs';

const LOG_LEVEL_CONFIG: Record<string, { color: string; icon: React.ReactNode }> = {
  error: { color: 'red', icon: <IconAlertCircle size={14} /> },
  warning: { color: 'yellow', icon: <IconAlertTriangle size={14} /> },
  info: { color: 'blue', icon: <IconInfoCircle size={14} /> },
  debug: { color: 'gray', icon: <IconBug size={14} /> },
};

export function LogsExplorerPage() {
  const [logs, setLogs] = useState<Log[]>([]);
  const [logTypes, setLogTypes] = useState<LogType[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);

  // Filters
  const [selectedType, setSelectedType] = useState<string | null>(null);
  const [selectedLevel, setSelectedLevel] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [startTime, setStartTime] = useState<Date | null>(null);
  const [endTime, setEndTime] = useState<Date | null>(null);

  // Auto-refresh
  const [autoRefresh, setAutoRefresh] = useState(false);
  const refreshInterval = useInterval(() => fetchLogs(), 5000);

  // Detail drawer
  const [drawerOpened, { open: openDrawer, close: closeDrawer }] = useDisclosure(false);
  const [selectedLog, setSelectedLog] = useState<Log | null>(null);

  // Pagination
  const [page, setPage] = useState(1);
  const [limit] = useState(100);

  const fetchLogTypes = useCallback(async () => {
    try {
      const types = await logsApi.listLogTypes();
      setLogTypes(types);
    } catch (error: any) {
      console.error('Failed to load log types:', error);
    }
  }, []);

  const fetchLogs = useCallback(async () => {
    setLoading(true);
    try {
      const response = await logsApi.searchLogs({
        type: selectedType || undefined,
        levels: selectedLevel ? [selectedLevel] : undefined,
        query: searchQuery || undefined,
        start_time: startTime?.toISOString(),
        end_time: endTime?.toISOString(),
        limit,
        offset: (page - 1) * limit,
      });
      setLogs(response.logs);
      setTotal(response.total);
    } catch (error: any) {
      notifications.show({
        title: 'Error',
        message: error.message || 'Failed to load logs',
        color: 'red',
      });
    } finally {
      setLoading(false);
    }
  }, [selectedType, selectedLevel, searchQuery, startTime, endTime, page, limit]);

  useEffect(() => {
    fetchLogTypes();
  }, [fetchLogTypes]);

  useEffect(() => {
    fetchLogs();
  }, [fetchLogs]);

  useEffect(() => {
    if (autoRefresh) {
      refreshInterval.start();
    } else {
      refreshInterval.stop();
    }
    return () => refreshInterval.stop();
  }, [autoRefresh]);

  const handleExport = async (format: 'json' | 'csv') => {
    try {
      const blob = await logsApi.exportLogs(format, {
        type: selectedType || undefined,
        level: selectedLevel || undefined,
        start_time: startTime?.toISOString(),
        end_time: endTime?.toISOString(),
      });

      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `logs-${new Date().toISOString()}.${format}`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);

      notifications.show({
        title: 'Success',
        message: 'Logs exported successfully',
        color: 'green',
      });
    } catch (error: any) {
      notifications.show({
        title: 'Error',
        message: error.message || 'Failed to export logs',
        color: 'red',
      });
    }
  };

  const openLogDetail = (log: Log) => {
    setSelectedLog(log);
    openDrawer();
  };

  const clearFilters = () => {
    setSelectedType(null);
    setSelectedLevel(null);
    setSearchQuery('');
    setStartTime(null);
    setEndTime(null);
    setPage(1);
  };

  const hasFilters = selectedType || selectedLevel || searchQuery || startTime || endTime;

  const formatTimestamp = (timestamp: string) => {
    const date = new Date(timestamp);
    return date.toLocaleString();
  };

  return (
    <PageContainer
      title="Logs Explorer"
      description="Search and analyze logs from all services"
      fullWidth
    >
      {/* Filters */}
      <Paper shadow="xs" p="md" mb="md" withBorder>
        <Stack gap="md">
          <Group justify="space-between">
            <Group gap="md">
              <Select
                placeholder="All log types"
                value={selectedType}
                onChange={setSelectedType}
                data={logTypes.map((t) => ({ value: t.id, label: t.name }))}
                clearable
                w={180}
              />
              <SegmentedControl
                value={selectedLevel || 'all'}
                onChange={(val) => setSelectedLevel(val === 'all' ? null : val)}
                data={[
                  { value: 'all', label: 'All' },
                  { value: 'error', label: 'Error' },
                  { value: 'warning', label: 'Warning' },
                  { value: 'info', label: 'Info' },
                  { value: 'debug', label: 'Debug' },
                ]}
              />
              <TextInput
                placeholder="Search logs..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                leftSection={<IconSearch size={16} />}
                w={250}
              />
            </Group>
            <Group gap="sm">
              <Switch
                label="Auto-refresh"
                checked={autoRefresh}
                onChange={(e) => setAutoRefresh(e.currentTarget.checked)}
              />
              <ActionIcon variant="subtle" onClick={fetchLogs}>
                <IconRefresh size={18} />
              </ActionIcon>
              <Menu position="bottom-end">
                <Menu.Target>
                  <Button variant="light" leftSection={<IconDownload size={16} />}>
                    Export
                  </Button>
                </Menu.Target>
                <Menu.Dropdown>
                  <Menu.Item onClick={() => handleExport('json')}>Export as JSON</Menu.Item>
                  <Menu.Item onClick={() => handleExport('csv')}>Export as CSV</Menu.Item>
                </Menu.Dropdown>
              </Menu>
            </Group>
          </Group>

          <Group gap="md">
            <TextInput
              type="datetime-local"
              placeholder="Start time"
              value={startTime ? startTime.toISOString().slice(0, 16) : ''}
              onChange={(e) => setStartTime(e.target.value ? new Date(e.target.value) : null)}
              w={200}
            />
            <Text c="dimmed">to</Text>
            <TextInput
              type="datetime-local"
              placeholder="End time"
              value={endTime ? endTime.toISOString().slice(0, 16) : ''}
              onChange={(e) => setEndTime(e.target.value ? new Date(e.target.value) : null)}
              w={200}
            />
            {hasFilters && (
              <Button
                variant="subtle"
                color="gray"
                leftSection={<IconX size={16} />}
                onClick={clearFilters}
              >
                Clear filters
              </Button>
            )}
          </Group>
        </Stack>
      </Paper>

      {/* Stats */}
      <Group mb="md" gap="md">
        <Badge variant="light" color="blue" size="lg">
          {total.toLocaleString()} logs
        </Badge>
        {autoRefresh && (
          <Badge variant="light" color="green" size="lg">
            Auto-refreshing
          </Badge>
        )}
      </Group>

      {/* Logs Table */}
      {loading && logs.length === 0 ? (
        <Center py="xl">
          <Loader size="lg" />
        </Center>
      ) : logs.length === 0 ? (
        <EmptyState
          icon={<IconSearch size={48} />}
          title="No logs found"
          description={hasFilters ? 'Try adjusting your filters' : 'Logs will appear here as they are generated'}
        />
      ) : (
        <Paper shadow="xs" withBorder>
          <ScrollArea h="calc(100vh - 380px)">
            <Table striped highlightOnHover stickyHeader>
              <Table.Thead>
                <Table.Tr>
                  <Table.Th w={180}>Timestamp</Table.Th>
                  <Table.Th w={100}>Type</Table.Th>
                  <Table.Th w={100}>Level</Table.Th>
                  <Table.Th>Message</Table.Th>
                  <Table.Th w={40}></Table.Th>
                </Table.Tr>
              </Table.Thead>
              <Table.Tbody>
                {logs.map((log) => {
                  const levelConfig = LOG_LEVEL_CONFIG[log.level] || LOG_LEVEL_CONFIG.info;
                  return (
                    <Table.Tr
                      key={log.id}
                      style={{ cursor: 'pointer' }}
                      onClick={() => openLogDetail(log)}
                    >
                      <Table.Td>
                        <Text size="sm" c="dimmed">
                          {formatTimestamp(log.timestamp)}
                        </Text>
                      </Table.Td>
                      <Table.Td>
                        <Badge size="sm" variant="outline">
                          {log.type}
                        </Badge>
                      </Table.Td>
                      <Table.Td>
                        <Badge size="sm" color={levelConfig.color} leftSection={levelConfig.icon}>
                          {log.level}
                        </Badge>
                      </Table.Td>
                      <Table.Td>
                        <Text size="sm" truncate style={{ maxWidth: 500 }}>
                          {log.message}
                        </Text>
                      </Table.Td>
                      <Table.Td>
                        <IconChevronRight size={16} color="var(--mantine-color-gray-5)" />
                      </Table.Td>
                    </Table.Tr>
                  );
                })}
              </Table.Tbody>
            </Table>
          </ScrollArea>
        </Paper>
      )}

      {/* Log Detail Drawer */}
      <Drawer
        opened={drawerOpened}
        onClose={closeDrawer}
        title="Log Details"
        position="right"
        size="lg"
      >
        {selectedLog && (
          <Stack gap="md">
            <Box>
              <Text size="sm" c="dimmed" mb="xs">
                Timestamp
              </Text>
              <Text>{formatTimestamp(selectedLog.timestamp)}</Text>
            </Box>

            <Box>
              <Text size="sm" c="dimmed" mb="xs">
                Type
              </Text>
              <Badge variant="outline">{selectedLog.type}</Badge>
            </Box>

            <Box>
              <Text size="sm" c="dimmed" mb="xs">
                Level
              </Text>
              <Badge
                color={LOG_LEVEL_CONFIG[selectedLog.level]?.color || 'gray'}
                leftSection={LOG_LEVEL_CONFIG[selectedLog.level]?.icon}
              >
                {selectedLog.level}
              </Badge>
            </Box>

            <Box>
              <Text size="sm" c="dimmed" mb="xs">
                Message
              </Text>
              <Code block style={{ whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}>
                {selectedLog.message}
              </Code>
            </Box>

            {selectedLog.metadata && Object.keys(selectedLog.metadata).length > 0 && (
              <Box>
                <Text size="sm" c="dimmed" mb="xs">
                  Metadata
                </Text>
                <Code block style={{ whiteSpace: 'pre-wrap' }}>
                  {JSON.stringify(selectedLog.metadata, null, 2)}
                </Code>
              </Box>
            )}

            <Box>
              <Text size="sm" c="dimmed" mb="xs">
                Log ID
              </Text>
              <Code>{selectedLog.id}</Code>
            </Box>
          </Stack>
        )}
      </Drawer>
    </PageContainer>
  );
}
