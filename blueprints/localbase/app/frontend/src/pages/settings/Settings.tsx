import { useState } from 'react';
import {
  Box,
  Tabs,
  TextInput,
  PasswordInput,
  Button,
  Stack,
  Group,
  Text,
  Card,
  CopyButton,
  ActionIcon,
  Tooltip,
  Code,
  Divider,
} from '@mantine/core';
import { IconCopy, IconCheck, IconKey, IconSettings, IconApi, IconDatabase } from '@tabler/icons-react';
import { notifications } from '@mantine/notifications';
import { PageContainer } from '../../components/layout/PageContainer';
import { useAppStore } from '../../stores/appStore';

// Default Supabase local development keys
const DEFAULT_ANON_KEY =
  'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZS1kZW1vIiwicm9sZSI6ImFub24iLCJleHAiOjE5ODM4MTI5OTZ9.CRXP1A7WOeoJeXxjNni43kdQwgnWNReilDMblYTn_I0';
const DEFAULT_SERVICE_KEY =
  'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZS1kZW1vIiwicm9sZSI6InNlcnZpY2Vfcm9sZSIsImV4cCI6MTk4MzgxMjk5Nn0.EGIM96RAZx35lJzdJsyH-qQwv8Hdp7fsn3W0YpN81IU';

export function SettingsPage() {
  const { projectName, setProjectName, serviceKey, setServiceKey } = useAppStore();
  const [localProjectName, setLocalProjectName] = useState(projectName);
  const [localServiceKey, setLocalServiceKey] = useState(serviceKey);

  const handleSaveGeneral = () => {
    setProjectName(localProjectName);
    notifications.show({
      title: 'Saved',
      message: 'Settings saved successfully',
      color: 'green',
    });
  };

  const handleSaveApi = () => {
    setServiceKey(localServiceKey);
    notifications.show({
      title: 'Saved',
      message: 'API key updated successfully',
      color: 'green',
    });
  };

  const apiUrl = window.location.origin;

  return (
    <PageContainer title="Settings" description="Configure your Localbase project">
      <Tabs defaultValue="general">
        <Tabs.List mb="lg">
          <Tabs.Tab value="general" leftSection={<IconSettings size={16} />}>
            General
          </Tabs.Tab>
          <Tabs.Tab value="api" leftSection={<IconApi size={16} />}>
            API Keys
          </Tabs.Tab>
          <Tabs.Tab value="database" leftSection={<IconDatabase size={16} />}>
            Database
          </Tabs.Tab>
        </Tabs.List>

        <Tabs.Panel value="general">
          <Card className="supabase-section">
            <Text fw={600} mb="md">
              Project Settings
            </Text>
            <Stack gap="md">
              <TextInput
                label="Project Name"
                description="Display name for your project"
                value={localProjectName}
                onChange={(e) => setLocalProjectName(e.target.value)}
              />
              <Group justify="flex-end">
                <Button onClick={handleSaveGeneral}>Save changes</Button>
              </Group>
            </Stack>
          </Card>
        </Tabs.Panel>

        <Tabs.Panel value="api">
          <Stack gap="lg">
            <Card className="supabase-section">
              <Text fw={600} mb="md">
                Project URL
              </Text>
              <Text size="sm" c="dimmed" mb="sm">
                Use this URL when initializing your Supabase client.
              </Text>
              <Group gap="sm">
                <TextInput value={apiUrl} readOnly style={{ flex: 1 }} />
                <CopyButton value={apiUrl}>
                  {({ copied, copy }) => (
                    <Tooltip label={copied ? 'Copied!' : 'Copy'}>
                      <ActionIcon variant="subtle" onClick={copy}>
                        {copied ? <IconCheck size={16} /> : <IconCopy size={16} />}
                      </ActionIcon>
                    </Tooltip>
                  )}
                </CopyButton>
              </Group>
            </Card>

            <Card className="supabase-section">
              <Text fw={600} mb="md">
                API Keys
              </Text>
              <Text size="sm" c="dimmed" mb="md">
                These keys are used to authenticate API requests. Keep your service role key secret!
              </Text>

              <Stack gap="lg">
                <Box>
                  <Group gap="xs" mb="xs">
                    <IconKey size={16} />
                    <Text size="sm" fw={500}>
                      anon (public)
                    </Text>
                  </Group>
                  <Text size="xs" c="dimmed" mb="xs">
                    Safe to use in browsers. Has limited access based on RLS policies.
                  </Text>
                  <Group gap="sm">
                    <Code
                      block
                      style={{
                        flex: 1,
                        fontSize: 11,
                        wordBreak: 'break-all',
                        whiteSpace: 'pre-wrap',
                      }}
                    >
                      {DEFAULT_ANON_KEY}
                    </Code>
                    <CopyButton value={DEFAULT_ANON_KEY}>
                      {({ copied, copy }) => (
                        <Tooltip label={copied ? 'Copied!' : 'Copy'}>
                          <ActionIcon variant="subtle" onClick={copy}>
                            {copied ? <IconCheck size={16} /> : <IconCopy size={16} />}
                          </ActionIcon>
                        </Tooltip>
                      )}
                    </CopyButton>
                  </Group>
                </Box>

                <Divider />

                <Box>
                  <Group gap="xs" mb="xs">
                    <IconKey size={16} color="var(--supabase-error)" />
                    <Text size="sm" fw={500} c="red">
                      service_role (secret)
                    </Text>
                  </Group>
                  <Text size="xs" c="dimmed" mb="xs">
                    Has full access to your database. Never expose this key in client-side code!
                  </Text>
                  <Group gap="sm">
                    <Code
                      block
                      style={{
                        flex: 1,
                        fontSize: 11,
                        wordBreak: 'break-all',
                        whiteSpace: 'pre-wrap',
                      }}
                    >
                      {DEFAULT_SERVICE_KEY}
                    </Code>
                    <CopyButton value={DEFAULT_SERVICE_KEY}>
                      {({ copied, copy }) => (
                        <Tooltip label={copied ? 'Copied!' : 'Copy'}>
                          <ActionIcon variant="subtle" onClick={copy}>
                            {copied ? <IconCheck size={16} /> : <IconCopy size={16} />}
                          </ActionIcon>
                        </Tooltip>
                      )}
                    </CopyButton>
                  </Group>
                </Box>
              </Stack>
            </Card>

            <Card className="supabase-section">
              <Text fw={600} mb="md">
                Dashboard API Key
              </Text>
              <Text size="sm" c="dimmed" mb="md">
                The key used by this dashboard to communicate with the API. By default, it uses the
                service role key.
              </Text>
              <Stack gap="md">
                <PasswordInput
                  label="Service Key"
                  description="Used for dashboard API calls"
                  value={localServiceKey}
                  onChange={(e) => setLocalServiceKey(e.target.value)}
                />
                <Group justify="flex-end">
                  <Button onClick={handleSaveApi}>Save changes</Button>
                </Group>
              </Stack>
            </Card>
          </Stack>
        </Tabs.Panel>

        <Tabs.Panel value="database">
          <Card className="supabase-section">
            <Text fw={600} mb="md">
              Database Connection
            </Text>
            <Text size="sm" c="dimmed" mb="md">
              Connection details for your PostgreSQL database.
            </Text>

            <Stack gap="md">
              <TextInput
                label="Host"
                value="localhost"
                readOnly
                rightSection={
                  <CopyButton value="localhost">
                    {({ copied, copy }) => (
                      <ActionIcon variant="subtle" onClick={copy} size="sm">
                        {copied ? <IconCheck size={14} /> : <IconCopy size={14} />}
                      </ActionIcon>
                    )}
                  </CopyButton>
                }
              />
              <TextInput
                label="Port"
                value="5432"
                readOnly
                rightSection={
                  <CopyButton value="5432">
                    {({ copied, copy }) => (
                      <ActionIcon variant="subtle" onClick={copy} size="sm">
                        {copied ? <IconCheck size={14} /> : <IconCopy size={14} />}
                      </ActionIcon>
                    )}
                  </CopyButton>
                }
              />
              <TextInput
                label="Database"
                value="postgres"
                readOnly
                rightSection={
                  <CopyButton value="postgres">
                    {({ copied, copy }) => (
                      <ActionIcon variant="subtle" onClick={copy} size="sm">
                        {copied ? <IconCheck size={14} /> : <IconCopy size={14} />}
                      </ActionIcon>
                    )}
                  </CopyButton>
                }
              />
              <TextInput
                label="User"
                value="postgres"
                readOnly
                rightSection={
                  <CopyButton value="postgres">
                    {({ copied, copy }) => (
                      <ActionIcon variant="subtle" onClick={copy} size="sm">
                        {copied ? <IconCheck size={14} /> : <IconCopy size={14} />}
                      </ActionIcon>
                    )}
                  </CopyButton>
                }
              />

              <Box>
                <Text size="sm" fw={500} mb="xs">
                  Connection String
                </Text>
                <Code
                  block
                  style={{
                    fontSize: 12,
                    wordBreak: 'break-all',
                    whiteSpace: 'pre-wrap',
                  }}
                >
                  postgresql://postgres:postgres@localhost:5432/postgres
                </Code>
              </Box>
            </Stack>
          </Card>
        </Tabs.Panel>
      </Tabs>
    </PageContainer>
  );
}
