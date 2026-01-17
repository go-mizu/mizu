import {
  Box,
  Group,
  Text,
  Button,
  ActionIcon,
  Badge,
  Menu,
  Modal,
  Stack,
  TextInput,
  CopyButton,
  Tooltip,
  Kbd,
  Divider,
  UnstyledButton,
} from '@mantine/core';
import {
  IconSearch,
  IconPlug,
  IconHelp,
  IconSettings,
  IconCopy,
  IconCheck,
  IconChevronDown,
  IconMessageCircle,
  IconBell,
  IconUser,
  IconDatabase,
  IconExternalLink,
} from '@tabler/icons-react';
import { useState } from 'react';
import { useAppStore } from '../../stores/appStore';

export function Header() {
  const { projectName } = useAppStore();
  const [connectModalOpened, setConnectModalOpened] = useState(false);

  // Mock connection details
  const connectionDetails = {
    host: 'localhost',
    port: '54322',
    database: 'postgres',
    user: 'postgres',
    password: 'postgres',
    directUrl: 'postgresql://postgres:postgres@localhost:54322/postgres',
    poolerUrl: 'postgresql://postgres:postgres@localhost:54322/postgres?pgbouncer=true',
    anonKey: 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZS1kZW1vIiwicm9sZSI6ImFub24iLCJleHAiOjE5ODM4MTI5OTZ9.CRXP1A7WOeoJeXxjNni43kdQwgnWNReilDMblYTn_I0',
    serviceKey: 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZS1kZW1vIiwicm9sZSI6InNlcnZpY2Vfcm9sZSIsImV4cCI6MTk4MzgxMjk5Nn0.EGIM96RAZx35lJzdJsyH-qQwv8Hdp7fsn3W0YpN81IU',
    projectUrl: 'http://localhost:8000',
  };

  return (
    <>
      <Box
        style={{
          height: 48,
          borderBottom: '1px solid var(--supabase-border)',
          backgroundColor: 'var(--supabase-bg)',
          display: 'flex',
          alignItems: 'center',
          paddingLeft: 16,
          paddingRight: 16,
        }}
      >
        <Group justify="space-between" style={{ width: '100%' }}>
          {/* Left side - Breadcrumb */}
          <Group gap="xs">
            <Menu position="bottom-start">
              <Menu.Target>
                <UnstyledButton>
                  <Group gap={4}>
                    <IconUser size={14} />
                    <Text size="sm" c="dimmed">
                      local
                    </Text>
                    <IconChevronDown size={12} />
                  </Group>
                </UnstyledButton>
              </Menu.Target>
              <Menu.Dropdown>
                <Menu.Label>Organization</Menu.Label>
                <Menu.Item>Local Development</Menu.Item>
              </Menu.Dropdown>
            </Menu>

            <Text c="dimmed">/</Text>

            <Menu position="bottom-start">
              <Menu.Target>
                <UnstyledButton>
                  <Group gap={4}>
                    <IconDatabase size={14} />
                    <Text size="sm" fw={500}>
                      {projectName}
                    </Text>
                    <IconChevronDown size={12} />
                  </Group>
                </UnstyledButton>
              </Menu.Target>
              <Menu.Dropdown>
                <Menu.Label>Project</Menu.Label>
                <Menu.Item>{projectName}</Menu.Item>
              </Menu.Dropdown>
            </Menu>

            <Badge
              size="xs"
              variant="light"
              color="green"
              style={{ textTransform: 'uppercase', fontWeight: 600, letterSpacing: 0.5 }}
            >
              Local
            </Badge>
          </Group>

          {/* Right side - Actions */}
          <Group gap="sm">
            {/* Connect Button */}
            <Button
              size="xs"
              variant="outline"
              leftSection={<IconPlug size={14} />}
              onClick={() => setConnectModalOpened(true)}
            >
              Connect
            </Button>

            {/* Search */}
            <Tooltip
              label={
                <Group gap={4}>
                  <Text size="xs">Search</Text>
                  <Kbd size="xs">⌘K</Kbd>
                </Group>
              }
            >
              <ActionIcon variant="subtle" size="lg">
                <IconSearch size={18} />
              </ActionIcon>
            </Tooltip>

            {/* Feedback */}
            <Tooltip label="Feedback">
              <ActionIcon variant="subtle" size="lg">
                <IconMessageCircle size={18} />
              </ActionIcon>
            </Tooltip>

            {/* Help */}
            <Menu position="bottom-end">
              <Menu.Target>
                <ActionIcon variant="subtle" size="lg">
                  <IconHelp size={18} />
                </ActionIcon>
              </Menu.Target>
              <Menu.Dropdown>
                <Menu.Item
                  leftSection={<IconExternalLink size={14} />}
                  component="a"
                  href="https://supabase.com/docs"
                  target="_blank"
                >
                  Documentation
                </Menu.Item>
                <Menu.Item leftSection={<IconExternalLink size={14} />}>
                  API Reference
                </Menu.Item>
                <Menu.Divider />
                <Menu.Item leftSection={<IconMessageCircle size={14} />}>
                  Support
                </Menu.Item>
              </Menu.Dropdown>
            </Menu>

            {/* Notifications */}
            <ActionIcon variant="subtle" size="lg">
              <IconBell size={18} />
            </ActionIcon>

            {/* Settings */}
            <ActionIcon variant="subtle" size="lg">
              <IconSettings size={18} />
            </ActionIcon>

            {/* User */}
            <ActionIcon variant="subtle" size="lg" radius="xl">
              <IconUser size={18} />
            </ActionIcon>
          </Group>
        </Group>
      </Box>

      {/* Connect Modal */}
      <Modal
        opened={connectModalOpened}
        onClose={() => setConnectModalOpened(false)}
        title="Connect to your project"
        size="lg"
      >
        <Stack gap="md">
          <Text size="sm" c="dimmed">
            Use these credentials to connect to your local Supabase project.
          </Text>

          <Divider label="Connection String" labelPosition="left" />

          <Box>
            <Text size="xs" fw={500} mb={4}>
              Direct connection
            </Text>
            <Text size="xs" c="dimmed" mb={8}>
              Use this for direct database connections (migrations, admin tasks)
            </Text>
            <ConnectionField value={connectionDetails.directUrl} />
          </Box>

          <Box>
            <Text size="xs" fw={500} mb={4}>
              Transaction pooler
            </Text>
            <Text size="xs" c="dimmed" mb={8}>
              Use this for serverless functions (short-lived connections)
            </Text>
            <ConnectionField value={connectionDetails.poolerUrl} />
          </Box>

          <Divider label="Connection Details" labelPosition="left" />

          <Group grow>
            <Box>
              <Text size="xs" fw={500} mb={4}>
                Host
              </Text>
              <ConnectionField value={connectionDetails.host} />
            </Box>
            <Box>
              <Text size="xs" fw={500} mb={4}>
                Port
              </Text>
              <ConnectionField value={connectionDetails.port} />
            </Box>
          </Group>

          <Group grow>
            <Box>
              <Text size="xs" fw={500} mb={4}>
                Database
              </Text>
              <ConnectionField value={connectionDetails.database} />
            </Box>
            <Box>
              <Text size="xs" fw={500} mb={4}>
                User
              </Text>
              <ConnectionField value={connectionDetails.user} />
            </Box>
          </Group>

          <Box>
            <Text size="xs" fw={500} mb={4}>
              Password
            </Text>
            <ConnectionField value={connectionDetails.password} isSecret />
          </Box>

          <Divider label="API Keys" labelPosition="left" />

          <Box>
            <Text size="xs" fw={500} mb={4}>
              Project URL
            </Text>
            <ConnectionField value={connectionDetails.projectUrl} />
          </Box>

          <Box>
            <Text size="xs" fw={500} mb={4}>
              anon key (public)
            </Text>
            <Text size="xs" c="dimmed" mb={8}>
              Safe to use in browsers. Has limited access based on RLS policies.
            </Text>
            <ConnectionField value={connectionDetails.anonKey} />
          </Box>

          <Box>
            <Text size="xs" fw={500} mb={4}>
              service_role key (secret)
            </Text>
            <Text size="xs" c="red" mb={8}>
              Never expose in browsers. Bypasses RLS - use only in secure backend code.
            </Text>
            <ConnectionField value={connectionDetails.serviceKey} isSecret />
          </Box>
        </Stack>
      </Modal>
    </>
  );
}

function ConnectionField({
  value,
  isSecret = false,
}: {
  value: string;
  isSecret?: boolean;
}) {
  const [revealed, setRevealed] = useState(false);

  return (
    <Group gap={0}>
      <TextInput
        size="xs"
        value={isSecret && !revealed ? '•'.repeat(Math.min(value.length, 40)) : value}
        readOnly
        style={{ flex: 1 }}
        styles={{
          input: {
            fontFamily: 'monospace',
            fontSize: 11,
            borderTopRightRadius: 0,
            borderBottomRightRadius: 0,
          },
        }}
      />
      {isSecret && (
        <Button
          size="xs"
          variant="default"
          onClick={() => setRevealed(!revealed)}
          style={{ borderRadius: 0 }}
        >
          {revealed ? 'Hide' : 'Reveal'}
        </Button>
      )}
      <CopyButton value={value} timeout={2000}>
        {({ copied, copy }) => (
          <Tooltip label={copied ? 'Copied' : 'Copy'}>
            <Button
              size="xs"
              variant="default"
              onClick={copy}
              style={{
                borderTopLeftRadius: 0,
                borderBottomLeftRadius: 0,
              }}
            >
              {copied ? <IconCheck size={14} /> : <IconCopy size={14} />}
            </Button>
          </Tooltip>
        )}
      </CopyButton>
    </Group>
  );
}
