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
} from '@mantine/core';
import {
  IconBolt,
  IconUsers,
  IconBroadcast,
  IconClock,
  IconRefresh,
  IconTrash,
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

export function RealtimePage() {
  const [stats, setStats] = useState<RealtimeStats | null>(null);
  const [channels, setChannels] = useState<Channel[]>([]);
  const [messages, setMessages] = useState<RealtimeMessage[]>([]);
  const [connected, setConnected] = useState(false);
  const [loading, setLoading] = useState(true);

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

  return (
    <PageContainer title="Realtime" description="Monitor WebSocket connections and messages">
      {/* Stats Cards */}
      <SimpleGrid cols={{ base: 1, sm: 3 }} spacing="lg" mb="xl">
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

      <SimpleGrid cols={{ base: 1, lg: 2 }} spacing="lg">
        {/* Channels */}
        <Card className="supabase-section">
          <Group justify="space-between" mb="md">
            <Text fw={600}>Channels</Text>
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
    </PageContainer>
  );
}
