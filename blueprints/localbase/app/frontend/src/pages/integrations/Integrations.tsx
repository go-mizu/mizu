import { useState } from 'react';
import {
  Box,
  Paper,
  Group,
  Text,
  Stack,
  Badge,
  Button,
  Tabs,
  Card,
  ThemeIcon,
  SimpleGrid,
  Modal,
  TextInput,
  Textarea,
  Switch,
  Table,
  ActionIcon,
  Tooltip,
  Alert,
  Code,
} from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import {
  IconClock,
  IconStack2,
  IconPlugConnected,
  IconPlus,
  IconTrash,
  IconPlayerPlay,
  IconPlayerPause,
  IconRefresh,
  IconInfoCircle,
  IconExternalLink,
} from '@tabler/icons-react';
import { notifications } from '@mantine/notifications';
import { PageContainer } from '../../components/layout/PageContainer';

interface CronJob {
  id: string;
  name: string;
  schedule: string;
  command: string;
  enabled: boolean;
  lastRun?: string;
  nextRun?: string;
}

interface Queue {
  id: string;
  name: string;
  pendingMessages: number;
  processingMessages: number;
  completedMessages: number;
  failedMessages: number;
}

interface Extension {
  name: string;
  description: string;
  version: string;
  enabled: boolean;
}

export function IntegrationsPage() {
  const [activeTab, setActiveTab] = useState<string | null>('cron');
  const [cronModalOpened, { open: openCronModal, close: closeCronModal }] =
    useDisclosure(false);

  // Mock data
  const [cronJobs, setCronJobs] = useState<CronJob[]>([
    {
      id: '1',
      name: 'cleanup_old_sessions',
      schedule: '0 0 * * *',
      command: "SELECT delete_old_sessions('30 days')",
      enabled: true,
      lastRun: '2025-01-16 00:00:00',
      nextRun: '2025-01-17 00:00:00',
    },
    {
      id: '2',
      name: 'send_daily_digest',
      schedule: '0 9 * * *',
      command: 'SELECT send_daily_digest()',
      enabled: true,
      lastRun: '2025-01-16 09:00:00',
      nextRun: '2025-01-17 09:00:00',
    },
  ]);

  const [queues] = useState<Queue[]>([
    {
      id: '1',
      name: 'email_notifications',
      pendingMessages: 5,
      processingMessages: 2,
      completedMessages: 1543,
      failedMessages: 3,
    },
    {
      id: '2',
      name: 'image_processing',
      pendingMessages: 12,
      processingMessages: 4,
      completedMessages: 892,
      failedMessages: 1,
    },
  ]);

  const [extensions] = useState<Extension[]>([
    {
      name: 'pg_cron',
      description: 'Job scheduling for PostgreSQL',
      version: '1.6.0',
      enabled: true,
    },
    {
      name: 'pgmq',
      description: 'A lightweight message queue for PostgreSQL',
      version: '1.0.0',
      enabled: true,
    },
    {
      name: 'pg_net',
      description: 'Async HTTP/HTTPS requests',
      version: '0.7.0',
      enabled: true,
    },
    {
      name: 'pg_graphql',
      description: 'GraphQL support for PostgreSQL',
      version: '1.5.0',
      enabled: false,
    },
    {
      name: 'vector',
      description: 'Vector similarity search (pgvector)',
      version: '0.5.0',
      enabled: false,
    },
    {
      name: 'postgis',
      description: 'Geographic objects support',
      version: '3.4.0',
      enabled: false,
    },
  ]);

  // Cron form state
  const [cronForm, setCronForm] = useState({
    name: '',
    schedule: '',
    command: '',
  });

  const handleCreateCronJob = () => {
    if (!cronForm.name || !cronForm.schedule || !cronForm.command) {
      notifications.show({
        title: 'Validation Error',
        message: 'All fields are required',
        color: 'red',
      });
      return;
    }

    const newJob: CronJob = {
      id: crypto.randomUUID(),
      name: cronForm.name,
      schedule: cronForm.schedule,
      command: cronForm.command,
      enabled: true,
    };

    setCronJobs([...cronJobs, newJob]);
    closeCronModal();
    setCronForm({ name: '', schedule: '', command: '' });
    notifications.show({
      title: 'Success',
      message: 'Cron job created successfully',
      color: 'green',
    });
  };

  const toggleCronJob = (id: string) => {
    setCronJobs(
      cronJobs.map((job) =>
        job.id === id ? { ...job, enabled: !job.enabled } : job
      )
    );
  };

  const deleteCronJob = (id: string) => {
    setCronJobs(cronJobs.filter((job) => job.id !== id));
    notifications.show({
      title: 'Deleted',
      message: 'Cron job deleted',
      color: 'green',
    });
  };

  const toggleExtension = (name: string) => {
    notifications.show({
      title: 'Extension Update',
      message: `Extension ${name} would be toggled. This requires running SQL.`,
      color: 'blue',
    });
  };

  return (
    <PageContainer
      title="Integrations"
      description="Manage cron jobs, queues, and database extensions"
    >
      <Tabs value={activeTab} onChange={setActiveTab}>
        <Tabs.List mb="lg">
          <Tabs.Tab value="cron" leftSection={<IconClock size={16} />}>
            Cron Jobs
          </Tabs.Tab>
          <Tabs.Tab value="queues" leftSection={<IconStack2 size={16} />}>
            Queues
          </Tabs.Tab>
          <Tabs.Tab value="extensions" leftSection={<IconPlugConnected size={16} />}>
            Extensions
          </Tabs.Tab>
        </Tabs.List>

        {/* Cron Jobs Tab */}
        <Tabs.Panel value="cron">
          <Stack gap="lg">
            <Alert icon={<IconInfoCircle size={16} />} color="blue">
              Cron jobs use pg_cron to schedule recurring SQL commands. Make sure the
              pg_cron extension is enabled.
            </Alert>

            <Group justify="space-between">
              <Text fw={500}>Scheduled Jobs</Text>
              <Button
                size="xs"
                leftSection={<IconPlus size={14} />}
                onClick={openCronModal}
              >
                Create job
              </Button>
            </Group>

            {cronJobs.length === 0 ? (
              <Paper p="xl" ta="center" withBorder>
                <IconClock size={32} style={{ opacity: 0.3 }} />
                <Text c="dimmed" mt="sm">
                  No cron jobs configured
                </Text>
                <Button
                  size="xs"
                  variant="light"
                  mt="md"
                  onClick={openCronModal}
                >
                  Create your first job
                </Button>
              </Paper>
            ) : (
              <Paper withBorder>
                <Table>
                  <Table.Thead>
                    <Table.Tr>
                      <Table.Th>Name</Table.Th>
                      <Table.Th>Schedule</Table.Th>
                      <Table.Th>Command</Table.Th>
                      <Table.Th>Last Run</Table.Th>
                      <Table.Th>Status</Table.Th>
                      <Table.Th w={100}>Actions</Table.Th>
                    </Table.Tr>
                  </Table.Thead>
                  <Table.Tbody>
                    {cronJobs.map((job) => (
                      <Table.Tr key={job.id}>
                        <Table.Td>
                          <Text size="sm" fw={500}>
                            {job.name}
                          </Text>
                        </Table.Td>
                        <Table.Td>
                          <Code>{job.schedule}</Code>
                        </Table.Td>
                        <Table.Td>
                          <Text size="sm" truncate style={{ maxWidth: 200 }}>
                            {job.command}
                          </Text>
                        </Table.Td>
                        <Table.Td>
                          <Text size="sm" c="dimmed">
                            {job.lastRun || 'Never'}
                          </Text>
                        </Table.Td>
                        <Table.Td>
                          <Badge
                            size="sm"
                            variant="light"
                            color={job.enabled ? 'green' : 'gray'}
                          >
                            {job.enabled ? 'Active' : 'Disabled'}
                          </Badge>
                        </Table.Td>
                        <Table.Td>
                          <Group gap={4}>
                            <Tooltip label={job.enabled ? 'Disable' : 'Enable'}>
                              <ActionIcon
                                size="sm"
                                variant="subtle"
                                onClick={() => toggleCronJob(job.id)}
                              >
                                {job.enabled ? (
                                  <IconPlayerPause size={14} />
                                ) : (
                                  <IconPlayerPlay size={14} />
                                )}
                              </ActionIcon>
                            </Tooltip>
                            <Tooltip label="Delete">
                              <ActionIcon
                                size="sm"
                                variant="subtle"
                                color="red"
                                onClick={() => deleteCronJob(job.id)}
                              >
                                <IconTrash size={14} />
                              </ActionIcon>
                            </Tooltip>
                          </Group>
                        </Table.Td>
                      </Table.Tr>
                    ))}
                  </Table.Tbody>
                </Table>
              </Paper>
            )}

            <Paper p="md" withBorder>
              <Text size="sm" fw={500} mb="sm">
                Common Cron Schedules
              </Text>
              <SimpleGrid cols={{ base: 1, sm: 2, lg: 4 }} spacing="sm">
                <Code block>
                  * * * * * - Every minute{'\n'}
                  */5 * * * * - Every 5 minutes
                </Code>
                <Code block>
                  0 * * * * - Every hour{'\n'}
                  0 0 * * * - Every day at midnight
                </Code>
                <Code block>
                  0 9 * * * - Every day at 9am{'\n'}
                  0 0 * * 0 - Every Sunday
                </Code>
                <Code block>
                  0 0 1 * * - First of month{'\n'}
                  0 0 1 1 * - First of year
                </Code>
              </SimpleGrid>
            </Paper>
          </Stack>
        </Tabs.Panel>

        {/* Queues Tab */}
        <Tabs.Panel value="queues">
          <Stack gap="lg">
            <Alert icon={<IconInfoCircle size={16} />} color="blue">
              Message queues use pgmq (PostgreSQL Message Queue) for async task
              processing.
            </Alert>

            <Group justify="space-between">
              <Text fw={500}>Message Queues</Text>
              <Button size="xs" leftSection={<IconPlus size={14} />}>
                Create queue
              </Button>
            </Group>

            {queues.length === 0 ? (
              <Paper p="xl" ta="center" withBorder>
                <IconStack2 size={32} style={{ opacity: 0.3 }} />
                <Text c="dimmed" mt="sm">
                  No queues configured
                </Text>
              </Paper>
            ) : (
              <SimpleGrid cols={{ base: 1, sm: 2 }} spacing="md">
                {queues.map((queue) => (
                  <Card key={queue.id} padding="md" withBorder>
                    <Group justify="space-between" mb="md">
                      <Group gap="sm">
                        <ThemeIcon size="sm" radius="xl" variant="light" color="blue">
                          <IconStack2 size={14} />
                        </ThemeIcon>
                        <Text fw={500}>{queue.name}</Text>
                      </Group>
                      <Group gap={4}>
                        <Tooltip label="Refresh">
                          <ActionIcon size="sm" variant="subtle">
                            <IconRefresh size={14} />
                          </ActionIcon>
                        </Tooltip>
                        <Tooltip label="Delete">
                          <ActionIcon size="sm" variant="subtle" color="red">
                            <IconTrash size={14} />
                          </ActionIcon>
                        </Tooltip>
                      </Group>
                    </Group>

                    <SimpleGrid cols={2} spacing="sm">
                      <Box>
                        <Text size="xs" c="dimmed">
                          Pending
                        </Text>
                        <Text fw={500}>{queue.pendingMessages}</Text>
                      </Box>
                      <Box>
                        <Text size="xs" c="dimmed">
                          Processing
                        </Text>
                        <Text fw={500}>{queue.processingMessages}</Text>
                      </Box>
                      <Box>
                        <Text size="xs" c="dimmed">
                          Completed
                        </Text>
                        <Text fw={500} c="green">
                          {queue.completedMessages}
                        </Text>
                      </Box>
                      <Box>
                        <Text size="xs" c="dimmed">
                          Failed
                        </Text>
                        <Text fw={500} c={queue.failedMessages > 0 ? 'red' : 'dimmed'}>
                          {queue.failedMessages}
                        </Text>
                      </Box>
                    </SimpleGrid>
                  </Card>
                ))}
              </SimpleGrid>
            )}
          </Stack>
        </Tabs.Panel>

        {/* Extensions Tab */}
        <Tabs.Panel value="extensions">
          <Stack gap="lg">
            <Alert icon={<IconInfoCircle size={16} />} color="blue">
              PostgreSQL extensions add powerful functionality to your database.
              Enable only the extensions you need.
            </Alert>

            <SimpleGrid cols={{ base: 1, sm: 2, lg: 3 }} spacing="md">
              {extensions.map((ext) => (
                <Card key={ext.name} padding="md" withBorder>
                  <Group justify="space-between" mb="sm">
                    <Group gap="sm">
                      <ThemeIcon
                        size="sm"
                        radius="xl"
                        variant="light"
                        color={ext.enabled ? 'green' : 'gray'}
                      >
                        <IconPlugConnected size={14} />
                      </ThemeIcon>
                      <Box>
                        <Text fw={500}>{ext.name}</Text>
                        <Text size="xs" c="dimmed">
                          v{ext.version}
                        </Text>
                      </Box>
                    </Group>
                    <Switch
                      checked={ext.enabled}
                      onChange={() => toggleExtension(ext.name)}
                      size="sm"
                    />
                  </Group>
                  <Text size="sm" c="dimmed">
                    {ext.description}
                  </Text>
                  {ext.name === 'pg_cron' && (
                    <Button
                      size="xs"
                      variant="subtle"
                      mt="sm"
                      rightSection={<IconExternalLink size={12} />}
                      onClick={() => setActiveTab('cron')}
                    >
                      Manage cron jobs
                    </Button>
                  )}
                  {ext.name === 'pgmq' && (
                    <Button
                      size="xs"
                      variant="subtle"
                      mt="sm"
                      rightSection={<IconExternalLink size={12} />}
                      onClick={() => setActiveTab('queues')}
                    >
                      Manage queues
                    </Button>
                  )}
                </Card>
              ))}
            </SimpleGrid>
          </Stack>
        </Tabs.Panel>
      </Tabs>

      {/* Create Cron Job Modal */}
      <Modal
        opened={cronModalOpened}
        onClose={closeCronModal}
        title="Create Cron Job"
        size="md"
      >
        <Stack gap="md">
          <TextInput
            label="Job Name"
            placeholder="cleanup_old_records"
            value={cronForm.name}
            onChange={(e) => setCronForm({ ...cronForm, name: e.target.value })}
            required
          />
          <TextInput
            label="Schedule (Cron Expression)"
            placeholder="0 0 * * *"
            description="Use cron syntax: minute hour day month weekday"
            value={cronForm.schedule}
            onChange={(e) => setCronForm({ ...cronForm, schedule: e.target.value })}
            required
          />
          <Textarea
            label="SQL Command"
            placeholder="SELECT cleanup_old_records();"
            minRows={3}
            value={cronForm.command}
            onChange={(e) => setCronForm({ ...cronForm, command: e.target.value })}
            required
            styles={{ input: { fontFamily: 'monospace' } }}
          />
          <Group justify="flex-end" mt="md">
            <Button variant="outline" onClick={closeCronModal}>
              Cancel
            </Button>
            <Button onClick={handleCreateCronJob}>Create Job</Button>
          </Group>
        </Stack>
      </Modal>
    </PageContainer>
  );
}
