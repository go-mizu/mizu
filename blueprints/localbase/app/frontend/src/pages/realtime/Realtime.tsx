import { useEffect, useState, useCallback } from 'react';
import {
  Group,
  Text,
  Stack,
  Card,
  SimpleGrid,
  Badge,
  Paper,
  ScrollArea,
  ThemeIcon,
  Center,
  Loader,
  Code,
  ActionIcon,
  Tabs,
  TextInput,
  NumberInput,
  Switch,
  Button,
  Alert,
  Table,
  Tooltip,
  Box,
} from '@mantine/core';
import {
  IconBolt,
  IconUsers,
  IconBroadcast,
  IconClock,
  IconRefresh,
  IconTrash,
  IconSettings,
  IconInfoCircle,
  IconShield,
  IconDatabase,
  IconPlus,
  IconEye,
  IconEyeOff,
} from '@tabler/icons-react';
import { notifications } from '@mantine/notifications';
import { PageContainer } from '../../components/layout/PageContainer';
import { EmptyState } from '../../components/common/EmptyState';
import { realtimeApi, realtimeClient } from '../../api';
import type { RealtimeStats, Channel } from '../../types';

interface RealtimeMessage {
  id: string;
  timestamp: Date;
  type: string;
  channel?: string;
  payload: any;
}

interface RealtimeSettings {
  maxChannelConnections: number;
  maxConcurrentUsers: number;
  maxEventsPerSecond: number;
  maxJoinReferences: number;
  poolSize: number;
  enablePresence: boolean;
  enableBroadcast: boolean;
  enablePostgresChanges: boolean;
  publicChannelAccess: boolean;
}

interface ChannelRestriction {
  id: string;
  pattern: string;
  type: 'allow' | 'deny';
  description: string;
}

export function RealtimePage() {
  const [activeTab, setActiveTab] = useState<string | null>('inspector');
  const [stats, setStats] = useState<RealtimeStats | null>(null);
  const [channels, setChannels] = useState<Channel[]>([]);
  const [messages, setMessages] = useState<RealtimeMessage[]>([]);
  const [connected, setConnected] = useState(false);
  const [loading, setLoading] = useState(true);

  // Settings state
  const [settings, setSettings] = useState<RealtimeSettings>({
    maxChannelConnections: 100,
    maxConcurrentUsers: 200,
    maxEventsPerSecond: 100,
    maxJoinReferences: 10,
    poolSize: 5,
    enablePresence: true,
    enableBroadcast: true,
    enablePostgresChanges: true,
    publicChannelAccess: false,
  });

  // Channel restrictions
  const [restrictions, setRestrictions] = useState<ChannelRestriction[]>([
    {
      id: '1',
      pattern: 'private:*',
      type: 'deny',
      description: 'Block public access to private channels',
    },
  ]);

  const [newRestrictionPattern, setNewRestrictionPattern] = useState('');

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const [statsData, channelsData] = await Promise.all([
        realtimeApi.getStats(),
        realtimeApi.listChannels(),
      ]);
      setStats(statsData);
      setChannels(channelsData || []);
    } catch (error: any) {
      notifications.show({
        title: 'Error',
        message: error.message || 'Failed to load realtime data',
        color: 'red',
      });
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchData();

    // Set up WebSocket connection
    const unsubscribeConnection = realtimeClient.onConnectionChange((isConnected) => {
      setConnected(isConnected);
      if (isConnected) {
        notifications.show({
          title: 'Connected',
          message: 'WebSocket connection established',
          color: 'green',
        });
      }
    });

    const unsubscribeMessage = realtimeClient.onMessage((message) => {
      setMessages((prev) => [
        {
          id: crypto.randomUUID(),
          timestamp: new Date(),
          type: message.type || 'unknown',
          channel: message.channel,
          payload: message,
        },
        ...prev.slice(0, 99), // Keep last 100 messages
      ]);
    });

    realtimeClient.connect();

    // Refresh stats periodically
    const interval = setInterval(fetchData, 10000);

    return () => {
      unsubscribeConnection();
      unsubscribeMessage();
      clearInterval(interval);
    };
  }, [fetchData]);

  const clearMessages = () => {
    setMessages([]);
  };

  const formatTime = (date: Date) => {
    return date.toLocaleTimeString('en-US', {
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
      hour12: false,
    });
  };

  const handleSaveSettings = () => {
    notifications.show({
      title: 'Settings Saved',
      message: 'Realtime settings have been updated',
      color: 'green',
    });
  };

  const addRestriction = () => {
    if (!newRestrictionPattern.trim()) return;

    setRestrictions([
      ...restrictions,
      {
        id: crypto.randomUUID(),
        pattern: newRestrictionPattern,
        type: 'deny',
        description: '',
      },
    ]);
    setNewRestrictionPattern('');
  };

  const removeRestriction = (id: string) => {
    setRestrictions(restrictions.filter((r) => r.id !== id));
  };

  return (
    <PageContainer
      title="Realtime"
      description="Monitor WebSocket connections, messages, and configure realtime settings"
    >
      {/* Stats Cards */}
      <SimpleGrid cols={{ base: 1, sm: 4 }} spacing="lg" mb="xl">
        <Card className="supabase-stat-card">
          <Group justify="space-between" mb="xs">
            <Text size="sm" c="dimmed" fw={500}>
              Active Connections
            </Text>
            <ThemeIcon size="lg" variant="light" color="blue" radius="md">
              <IconUsers size={20} />
            </ThemeIcon>
          </Group>
          {loading ? (
            <Loader size="sm" />
          ) : (
            <Text className="supabase-stat-value">{stats?.connections ?? 0}</Text>
          )}
        </Card>

        <Card className="supabase-stat-card">
          <Group justify="space-between" mb="xs">
            <Text size="sm" c="dimmed" fw={500}>
              Active Channels
            </Text>
            <ThemeIcon size="lg" variant="light" color="violet" radius="md">
              <IconBroadcast size={20} />
            </ThemeIcon>
          </Group>
          {loading ? (
            <Loader size="sm" />
          ) : (
            <Text className="supabase-stat-value">{stats?.channels ?? 0}</Text>
          )}
        </Card>

        <Card className="supabase-stat-card">
          <Group justify="space-between" mb="xs">
            <Text size="sm" c="dimmed" fw={500}>
              Messages / sec
            </Text>
            <ThemeIcon size="lg" variant="light" color="orange" radius="md">
              <IconBolt size={20} />
            </ThemeIcon>
          </Group>
          {loading ? (
            <Loader size="sm" />
          ) : (
            <Text className="supabase-stat-value">{stats?.messagesPerSecond ?? 0}</Text>
          )}
        </Card>

        <Card className="supabase-stat-card">
          <Group justify="space-between" mb="xs">
            <Text size="sm" c="dimmed" fw={500}>
              Connection Status
            </Text>
            <ThemeIcon
              size="lg"
              variant="light"
              color={connected ? 'green' : 'red'}
              radius="md"
            >
              <IconBolt size={20} />
            </ThemeIcon>
          </Group>
          <Badge size="lg" color={connected ? 'green' : 'red'} variant="light">
            {connected ? 'Connected' : 'Disconnected'}
          </Badge>
        </Card>
      </SimpleGrid>

      <Tabs value={activeTab} onChange={setActiveTab}>
        <Tabs.List mb="lg">
          <Tabs.Tab value="inspector" leftSection={<IconBroadcast size={16} />}>
            Inspector
          </Tabs.Tab>
          <Tabs.Tab value="channels" leftSection={<IconUsers size={16} />}>
            Channels
          </Tabs.Tab>
          <Tabs.Tab value="settings" leftSection={<IconSettings size={16} />}>
            Settings
          </Tabs.Tab>
        </Tabs.List>

        {/* Inspector Tab */}
        <Tabs.Panel value="inspector">
          <SimpleGrid cols={{ base: 1, lg: 2 }} spacing="lg">
            {/* Channels */}
            <Card className="supabase-section">
              <Group justify="space-between" mb="md">
                <Text fw={600}>Active Channels</Text>
                <ActionIcon variant="subtle" onClick={fetchData}>
                  <IconRefresh size={18} />
                </ActionIcon>
              </Group>

              {loading ? (
                <Center py="xl">
                  <Loader size="sm" />
                </Center>
              ) : !channels || channels.length === 0 ? (
                <EmptyState
                  icon={<IconBroadcast size={32} />}
                  title="No active channels"
                  description="Channels will appear here when clients subscribe"
                />
              ) : (
                <Stack gap="xs">
                  {channels.map((channel) => (
                    <Paper
                      key={channel.id}
                      p="sm"
                      style={{
                        border: '1px solid var(--supabase-border)',
                        borderRadius: 6,
                      }}
                    >
                      <Group justify="space-between">
                        <Group gap="xs">
                          <IconBroadcast size={16} />
                          <Text size="sm" fw={500}>
                            {channel.name}
                          </Text>
                        </Group>
                        <Text size="xs" c="dimmed">
                          Created {new Date(channel.inserted_at).toLocaleDateString()}
                        </Text>
                      </Group>
                    </Paper>
                  ))}
                </Stack>
              )}
            </Card>

            {/* Message Inspector */}
            <Card className="supabase-section" style={{ display: 'flex', flexDirection: 'column' }}>
              <Group justify="space-between" mb="md">
                <Group gap="xs">
                  <Text fw={600}>Message Inspector</Text>
                  <Badge size="sm" variant="light" color={connected ? 'green' : 'gray'}>
                    {messages.length} messages
                  </Badge>
                </Group>
                <ActionIcon variant="subtle" onClick={clearMessages}>
                  <IconTrash size={18} />
                </ActionIcon>
              </Group>

              <ScrollArea style={{ flex: 1, minHeight: 300 }}>
                {messages.length === 0 ? (
                  <Center py="xl">
                    <Text c="dimmed" size="sm">
                      {connected
                        ? 'Waiting for messages...'
                        : 'Connect to see live messages'}
                    </Text>
                  </Center>
                ) : (
                  <Stack gap="xs">
                    {messages.map((msg) => (
                      <Paper
                        key={msg.id}
                        p="xs"
                        style={{
                          border: '1px solid var(--supabase-border)',
                          borderRadius: 6,
                          backgroundColor: 'var(--supabase-bg-surface)',
                        }}
                      >
                        <Group justify="space-between" mb="xs">
                          <Group gap="xs">
                            <Badge size="xs" variant="light">
                              {msg.type}
                            </Badge>
                            {msg.channel && (
                              <Badge size="xs" variant="light" color="blue">
                                {msg.channel}
                              </Badge>
                            )}
                          </Group>
                          <Group gap={4}>
                            <IconClock size={12} />
                            <Text size="xs" c="dimmed">
                              {formatTime(msg.timestamp)}
                            </Text>
                          </Group>
                        </Group>
                        <Code block style={{ fontSize: 11 }}>
                          {JSON.stringify(msg.payload, null, 2)}
                        </Code>
                      </Paper>
                    ))}
                  </Stack>
                )}
              </ScrollArea>
            </Card>
          </SimpleGrid>
        </Tabs.Panel>

        {/* Channels Tab */}
        <Tabs.Panel value="channels">
          <Stack gap="lg">
            <Alert icon={<IconInfoCircle size={16} />} color="blue">
              Manage channel subscriptions and monitor channel activity. Channels are created
              automatically when clients subscribe.
            </Alert>

            <Paper withBorder>
              {loading ? (
                <Center py="xl">
                  <Loader size="sm" />
                </Center>
              ) : !channels || channels.length === 0 ? (
                <Box p="xl">
                  <EmptyState
                    icon={<IconBroadcast size={32} />}
                    title="No active channels"
                    description="Channels will appear here when clients subscribe to them"
                  />
                </Box>
              ) : (
                <Table>
                  <Table.Thead>
                    <Table.Tr>
                      <Table.Th>Channel Name</Table.Th>
                      <Table.Th>Type</Table.Th>
                      <Table.Th>Subscribers</Table.Th>
                      <Table.Th>Created</Table.Th>
                      <Table.Th>Status</Table.Th>
                    </Table.Tr>
                  </Table.Thead>
                  <Table.Tbody>
                    {channels.map((channel) => (
                      <Table.Tr key={channel.id}>
                        <Table.Td>
                          <Group gap="xs">
                            <IconBroadcast size={16} />
                            <Text size="sm" fw={500}>
                              {channel.name}
                            </Text>
                          </Group>
                        </Table.Td>
                        <Table.Td>
                          <Badge size="sm" variant="light">
                            broadcast
                          </Badge>
                        </Table.Td>
                        <Table.Td>
                          <Text size="sm">0</Text>
                        </Table.Td>
                        <Table.Td>
                          <Text size="sm" c="dimmed">
                            {new Date(channel.inserted_at).toLocaleString()}
                          </Text>
                        </Table.Td>
                        <Table.Td>
                          <Badge size="sm" variant="light" color="green">
                            Active
                          </Badge>
                        </Table.Td>
                      </Table.Tr>
                    ))}
                  </Table.Tbody>
                </Table>
              )}
            </Paper>
          </Stack>
        </Tabs.Panel>

        {/* Settings Tab */}
        <Tabs.Panel value="settings">
          <Stack gap="lg">
            <Alert icon={<IconInfoCircle size={16} />} color="blue">
              Configure realtime settings including connection limits, channel restrictions,
              and database connection pool size for RLS authorization.
            </Alert>

            <SimpleGrid cols={{ base: 1, lg: 2 }} spacing="lg">
              {/* Connection Settings */}
              <Paper p="md" withBorder>
                <Group gap="xs" mb="md">
                  <IconDatabase size={18} />
                  <Text fw={600}>Connection Settings</Text>
                </Group>

                <Stack gap="md">
                  <NumberInput
                    label="Max Channel Connections"
                    description="Maximum connections per channel"
                    value={settings.maxChannelConnections}
                    onChange={(val) =>
                      setSettings({ ...settings, maxChannelConnections: Number(val) || 100 })
                    }
                    min={1}
                    max={10000}
                  />

                  <NumberInput
                    label="Max Concurrent Users"
                    description="Maximum total concurrent users"
                    value={settings.maxConcurrentUsers}
                    onChange={(val) =>
                      setSettings({ ...settings, maxConcurrentUsers: Number(val) || 200 })
                    }
                    min={1}
                    max={100000}
                  />

                  <NumberInput
                    label="Max Events per Second"
                    description="Rate limit for events per connection"
                    value={settings.maxEventsPerSecond}
                    onChange={(val) =>
                      setSettings({ ...settings, maxEventsPerSecond: Number(val) || 100 })
                    }
                    min={1}
                    max={1000}
                  />

                  <NumberInput
                    label="Database Pool Size"
                    description="Connection pool size for RLS authorization checks"
                    value={settings.poolSize}
                    onChange={(val) =>
                      setSettings({ ...settings, poolSize: Number(val) || 5 })
                    }
                    min={1}
                    max={100}
                  />
                </Stack>
              </Paper>

              {/* Feature Toggles */}
              <Paper p="md" withBorder>
                <Group gap="xs" mb="md">
                  <IconBolt size={18} />
                  <Text fw={600}>Features</Text>
                </Group>

                <Stack gap="md">
                  <Switch
                    label="Enable Presence"
                    description="Allow presence tracking in channels"
                    checked={settings.enablePresence}
                    onChange={(e) =>
                      setSettings({ ...settings, enablePresence: e.currentTarget.checked })
                    }
                  />

                  <Switch
                    label="Enable Broadcast"
                    description="Allow broadcast messages between clients"
                    checked={settings.enableBroadcast}
                    onChange={(e) =>
                      setSettings({ ...settings, enableBroadcast: e.currentTarget.checked })
                    }
                  />

                  <Switch
                    label="Enable Postgres Changes"
                    description="Allow subscribing to database changes"
                    checked={settings.enablePostgresChanges}
                    onChange={(e) =>
                      setSettings({
                        ...settings,
                        enablePostgresChanges: e.currentTarget.checked,
                      })
                    }
                  />

                  <Switch
                    label="Public Channel Access"
                    description="Allow unauthenticated access to public channels"
                    checked={settings.publicChannelAccess}
                    onChange={(e) =>
                      setSettings({
                        ...settings,
                        publicChannelAccess: e.currentTarget.checked,
                      })
                    }
                  />
                </Stack>
              </Paper>
            </SimpleGrid>

            {/* Channel Restrictions */}
            <Paper p="md" withBorder>
              <Group gap="xs" mb="md">
                <IconShield size={18} />
                <Text fw={600}>Channel Restrictions</Text>
              </Group>

              <Text size="sm" c="dimmed" mb="md">
                Define patterns to restrict public access to specific channels. Use wildcards (*)
                for pattern matching.
              </Text>

              <Group mb="md">
                <TextInput
                  placeholder="e.g., private:*, admin:*"
                  value={newRestrictionPattern}
                  onChange={(e) => setNewRestrictionPattern(e.target.value)}
                  style={{ flex: 1 }}
                />
                <Button
                  leftSection={<IconPlus size={14} />}
                  onClick={addRestriction}
                  disabled={!newRestrictionPattern.trim()}
                >
                  Add Restriction
                </Button>
              </Group>

              {restrictions.length === 0 ? (
                <Text size="sm" c="dimmed" ta="center" py="md">
                  No channel restrictions configured
                </Text>
              ) : (
                <Table>
                  <Table.Thead>
                    <Table.Tr>
                      <Table.Th>Pattern</Table.Th>
                      <Table.Th>Type</Table.Th>
                      <Table.Th w={80}>Actions</Table.Th>
                    </Table.Tr>
                  </Table.Thead>
                  <Table.Tbody>
                    {restrictions.map((restriction) => (
                      <Table.Tr key={restriction.id}>
                        <Table.Td>
                          <Code>{restriction.pattern}</Code>
                        </Table.Td>
                        <Table.Td>
                          <Badge
                            size="sm"
                            variant="light"
                            color={restriction.type === 'deny' ? 'red' : 'green'}
                          >
                            {restriction.type}
                          </Badge>
                        </Table.Td>
                        <Table.Td>
                          <Tooltip label="Remove">
                            <ActionIcon
                              size="sm"
                              variant="subtle"
                              color="red"
                              onClick={() => removeRestriction(restriction.id)}
                            >
                              <IconTrash size={14} />
                            </ActionIcon>
                          </Tooltip>
                        </Table.Td>
                      </Table.Tr>
                    ))}
                  </Table.Tbody>
                </Table>
              )}
            </Paper>

            <Group justify="flex-end">
              <Button onClick={handleSaveSettings}>Save Settings</Button>
            </Group>
          </Stack>
        </Tabs.Panel>
      </Tabs>
    </PageContainer>
  );
}
