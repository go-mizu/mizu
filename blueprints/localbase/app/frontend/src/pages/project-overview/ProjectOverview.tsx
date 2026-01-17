import { useEffect, useState } from 'react';
import {
  Box,
  Paper,
  Group,
  Text,
  SimpleGrid,
  Stack,
  Badge,
  Button,
  Card,
  ThemeIcon,
  Skeleton,
  List,
  Anchor,
} from '@mantine/core';
import {
  IconUsers,
  IconTable,
  IconFolder,
  IconBolt,
  IconDatabase,
  IconShield,
  IconChevronRight,
  IconExternalLink,
  IconCheck,
  IconAlertTriangle,
  IconCircleCheck,
  IconActivity,
  IconArrowUpRight,
} from '@tabler/icons-react';
import { Link } from 'react-router-dom';
import { PageContainer } from '../../components/layout/PageContainer';
import { dashboardApi, authApi, databaseApi, storageApi, functionsApi } from '../../api';

interface Stats {
  users: number;
  tables: number;
  buckets: number;
  functions: number;
}

interface ServiceHealth {
  name: string;
  status: 'healthy' | 'degraded' | 'unhealthy';
  latency?: number;
}

export function ProjectOverviewPage() {
  const [stats, setStats] = useState<Stats>({
    users: 0,
    tables: 0,
    buckets: 0,
    functions: 0,
  });
  const [health, setHealth] = useState<ServiceHealth[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchData = async () => {
      setLoading(true);
      try {
        const [healthData, usersData, tables, buckets, functions] = await Promise.all([
          dashboardApi.getHealth(),
          authApi.listUsers().catch(() => ({ users: [], total: 0 })),
          databaseApi.listTables('public').catch(() => []),
          storageApi.listBuckets().catch(() => []),
          functionsApi.listFunctions().catch(() => []),
        ]);

        // Convert HealthStatus to ServiceHealth[]
        const healthServices: ServiceHealth[] = [
          { name: 'Database', status: healthData.services.database ? 'healthy' : 'unhealthy' },
          { name: 'Auth', status: healthData.services.auth ? 'healthy' : 'unhealthy' },
          { name: 'Storage', status: healthData.services.storage ? 'healthy' : 'unhealthy' },
          { name: 'Realtime', status: healthData.services.realtime ? 'healthy' : 'unhealthy' },
        ];
        setHealth(healthServices);

        setStats({
          users: usersData?.total || usersData?.users?.length || 0,
          tables: tables?.length || 0,
          buckets: buckets?.length || 0,
          functions: functions?.length || 0,
        });
      } catch (error) {
        console.error('Failed to fetch dashboard data:', error);
      } finally {
        setLoading(false);
      }
    };

    fetchData();
  }, []);

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'healthy':
        return 'green';
      case 'degraded':
        return 'yellow';
      case 'unhealthy':
        return 'red';
      default:
        return 'gray';
    }
  };

  const quickLinks = [
    {
      title: 'Table Editor',
      description: 'View and manage your database tables',
      icon: IconTable,
      path: '/table-editor',
      color: 'blue',
    },
    {
      title: 'SQL Editor',
      description: 'Run SQL queries and manage saved queries',
      icon: IconDatabase,
      path: '/sql-editor',
      color: 'green',
    },
    {
      title: 'Authentication',
      description: 'Manage users and auth settings',
      icon: IconUsers,
      path: '/auth/users',
      color: 'violet',
    },
    {
      title: 'Storage',
      description: 'Upload and manage files',
      icon: IconFolder,
      path: '/storage',
      color: 'orange',
    },
  ];

  const recentActivity = [
    { action: 'Table created', target: 'public.profiles', time: '2 minutes ago' },
    { action: 'User signed up', target: 'user@example.com', time: '5 minutes ago' },
    { action: 'Policy added', target: 'profiles_select', time: '10 minutes ago' },
    { action: 'Function deployed', target: 'hello-world', time: '1 hour ago' },
  ];

  return (
    <PageContainer title="Project Overview" description="Welcome to your local Supabase project">
      <Stack gap="lg">
        {/* Stats Cards */}
        <SimpleGrid cols={{ base: 1, sm: 2, lg: 4 }} spacing="md">
          <StatCard
            title="Users"
            value={stats.users}
            icon={IconUsers}
            color="violet"
            loading={loading}
            linkTo="/auth/users"
          />
          <StatCard
            title="Tables"
            value={stats.tables}
            icon={IconTable}
            color="blue"
            loading={loading}
            linkTo="/table-editor"
          />
          <StatCard
            title="Storage Buckets"
            value={stats.buckets}
            icon={IconFolder}
            color="orange"
            loading={loading}
            linkTo="/storage"
          />
          <StatCard
            title="Edge Functions"
            value={stats.functions}
            icon={IconBolt}
            color="green"
            loading={loading}
            linkTo="/functions"
          />
        </SimpleGrid>

        {/* Two Column Layout */}
        <SimpleGrid cols={{ base: 1, lg: 2 }} spacing="lg">
          {/* Service Health */}
          <Paper p="md" radius="md" withBorder>
            <Group justify="space-between" mb="md">
              <Text fw={600}>Service Health</Text>
              <Badge variant="light" color="green" size="sm">
                All systems operational
              </Badge>
            </Group>
            <Stack gap="sm">
              {loading ? (
                <>
                  <Skeleton height={40} />
                  <Skeleton height={40} />
                  <Skeleton height={40} />
                  <Skeleton height={40} />
                </>
              ) : (
                health.map((service) => (
                  <Group key={service.name} justify="space-between">
                    <Group gap="sm">
                      <ThemeIcon
                        size="sm"
                        radius="xl"
                        variant="light"
                        color={getStatusColor(service.status)}
                      >
                        {service.status === 'healthy' ? (
                          <IconCircleCheck size={14} />
                        ) : (
                          <IconAlertTriangle size={14} />
                        )}
                      </ThemeIcon>
                      <Text size="sm">{service.name}</Text>
                    </Group>
                    <Group gap="xs">
                      {service.latency !== undefined && (
                        <Text size="xs" c="dimmed">
                          {service.latency}ms
                        </Text>
                      )}
                      <Badge
                        size="xs"
                        variant="light"
                        color={getStatusColor(service.status)}
                      >
                        {service.status}
                      </Badge>
                    </Group>
                  </Group>
                ))
              )}
            </Stack>
          </Paper>

          {/* Recent Activity */}
          <Paper p="md" radius="md" withBorder>
            <Group justify="space-between" mb="md">
              <Text fw={600}>Recent Activity</Text>
              <Button
                variant="subtle"
                size="xs"
                rightSection={<IconChevronRight size={14} />}
                component={Link}
                to="/logs"
              >
                View logs
              </Button>
            </Group>
            <Stack gap="sm">
              {recentActivity.map((activity, index) => (
                <Group key={index} justify="space-between">
                  <Group gap="sm">
                    <ThemeIcon size="sm" radius="xl" variant="light" color="gray">
                      <IconActivity size={14} />
                    </ThemeIcon>
                    <Box>
                      <Text size="sm">{activity.action}</Text>
                      <Text size="xs" c="dimmed">
                        {activity.target}
                      </Text>
                    </Box>
                  </Group>
                  <Text size="xs" c="dimmed">
                    {activity.time}
                  </Text>
                </Group>
              ))}
            </Stack>
          </Paper>
        </SimpleGrid>

        {/* Quick Links */}
        <Paper p="md" radius="md" withBorder>
          <Text fw={600} mb="md">
            Quick Actions
          </Text>
          <SimpleGrid cols={{ base: 1, sm: 2, lg: 4 }} spacing="md">
            {quickLinks.map((link) => (
              <Card
                key={link.path}
                component={Link}
                to={link.path}
                padding="md"
                radius="md"
                withBorder
                style={{ cursor: 'pointer' }}
              >
                <Group>
                  <ThemeIcon size="lg" radius="md" variant="light" color={link.color}>
                    <link.icon size={20} />
                  </ThemeIcon>
                  <Box style={{ flex: 1 }}>
                    <Text size="sm" fw={500}>
                      {link.title}
                    </Text>
                    <Text size="xs" c="dimmed">
                      {link.description}
                    </Text>
                  </Box>
                  <IconArrowUpRight size={16} style={{ opacity: 0.5 }} />
                </Group>
              </Card>
            ))}
          </SimpleGrid>
        </Paper>

        {/* Getting Started / Resources */}
        <SimpleGrid cols={{ base: 1, lg: 2 }} spacing="lg">
          <Paper p="md" radius="md" withBorder>
            <Text fw={600} mb="md">
              Getting Started
            </Text>
            <List spacing="sm" size="sm">
              <List.Item
                icon={
                  <ThemeIcon size="sm" radius="xl" variant="light" color="green">
                    <IconCheck size={12} />
                  </ThemeIcon>
                }
              >
                <Text size="sm">Set up your local Supabase instance</Text>
              </List.Item>
              <List.Item
                icon={
                  <ThemeIcon size="sm" radius="xl" variant="light" color="blue">
                    <IconDatabase size={12} />
                  </ThemeIcon>
                }
              >
                <Group gap="xs">
                  <Text size="sm">Create your first table</Text>
                  <Button
                    variant="subtle"
                    size="compact-xs"
                    component={Link}
                    to="/table-editor"
                  >
                    Go
                  </Button>
                </Group>
              </List.Item>
              <List.Item
                icon={
                  <ThemeIcon size="sm" radius="xl" variant="light" color="violet">
                    <IconShield size={12} />
                  </ThemeIcon>
                }
              >
                <Group gap="xs">
                  <Text size="sm">Set up Row Level Security</Text>
                  <Button
                    variant="subtle"
                    size="compact-xs"
                    component={Link}
                    to="/database/policies"
                  >
                    Go
                  </Button>
                </Group>
              </List.Item>
              <List.Item
                icon={
                  <ThemeIcon size="sm" radius="xl" variant="light" color="orange">
                    <IconUsers size={12} />
                  </ThemeIcon>
                }
              >
                <Group gap="xs">
                  <Text size="sm">Add authentication to your app</Text>
                  <Button
                    variant="subtle"
                    size="compact-xs"
                    component={Link}
                    to="/auth/users"
                  >
                    Go
                  </Button>
                </Group>
              </List.Item>
            </List>
          </Paper>

          <Paper p="md" radius="md" withBorder>
            <Text fw={600} mb="md">
              Resources
            </Text>
            <Stack gap="sm">
              <Anchor
                href="https://supabase.com/docs"
                target="_blank"
                underline="never"
                c="inherit"
              >
                <Group gap="sm">
                  <ThemeIcon size="sm" radius="xl" variant="light" color="gray">
                    <IconExternalLink size={14} />
                  </ThemeIcon>
                  <Box style={{ flex: 1 }}>
                    <Text size="sm">Documentation</Text>
                    <Text size="xs" c="dimmed">
                      Learn how to use Supabase features
                    </Text>
                  </Box>
                </Group>
              </Anchor>
              <Anchor
                href="https://supabase.com/docs/reference"
                target="_blank"
                underline="never"
                c="inherit"
              >
                <Group gap="sm">
                  <ThemeIcon size="sm" radius="xl" variant="light" color="gray">
                    <IconExternalLink size={14} />
                  </ThemeIcon>
                  <Box style={{ flex: 1 }}>
                    <Text size="sm">API Reference</Text>
                    <Text size="xs" c="dimmed">
                      Complete API documentation
                    </Text>
                  </Box>
                </Group>
              </Anchor>
              <Anchor
                href="https://github.com/supabase/supabase"
                target="_blank"
                underline="never"
                c="inherit"
              >
                <Group gap="sm">
                  <ThemeIcon size="sm" radius="xl" variant="light" color="gray">
                    <IconExternalLink size={14} />
                  </ThemeIcon>
                  <Box style={{ flex: 1 }}>
                    <Text size="sm">GitHub</Text>
                    <Text size="xs" c="dimmed">
                      View source code and contribute
                    </Text>
                  </Box>
                </Group>
              </Anchor>
            </Stack>
          </Paper>
        </SimpleGrid>
      </Stack>
    </PageContainer>
  );
}

function StatCard({
  title,
  value,
  icon: Icon,
  color,
  loading,
  linkTo,
}: {
  title: string;
  value: number;
  icon: typeof IconUsers;
  color: string;
  loading: boolean;
  linkTo: string;
}) {
  return (
    <Card
      component={Link}
      to={linkTo}
      padding="md"
      radius="md"
      withBorder
      style={{ cursor: 'pointer' }}
    >
      <Group justify="space-between">
        <Box>
          <Text size="xs" c="dimmed" tt="uppercase" fw={600}>
            {title}
          </Text>
          {loading ? (
            <Skeleton height={32} width={60} mt={4} />
          ) : (
            <Text size="xl" fw={700} mt={4}>
              {value.toLocaleString()}
            </Text>
          )}
        </Box>
        <ThemeIcon size="xl" radius="md" variant="light" color={color}>
          <Icon size={24} />
        </ThemeIcon>
      </Group>
    </Card>
  );
}
