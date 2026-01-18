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
  useMantineColorScheme,
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
  IconBuilding,
  IconSun,
  IconMoon,
} from '@tabler/icons-react';
import { useState } from 'react';
import { useAppStore } from '../../stores/appStore';

export function Header() {
  const { projectName } = useAppStore();
  const [connectModalOpened, setConnectModalOpened] = useState(false);
  const { colorScheme, toggleColorScheme } = useMantineColorScheme();
  const isDark = colorScheme === 'dark';

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
          borderBottom: '1px solid var(--lb-border-default)',
          backgroundColor: 'var(--lb-bg-primary)',
          display: 'flex',
          alignItems: 'center',
          paddingLeft: 16,
          paddingRight: 16,
        }}
      >
        <Group justify="space-between" style={{ width: '100%' }}>
          {/* Left side - Breadcrumb */}
          <Group gap={8}>
            {/* Organization Dropdown */}
            <Menu position="bottom-start" shadow="md">
              <Menu.Target>
                <UnstyledButton
                  className="lb-breadcrumb-item"
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 6,
                    padding: '4px 8px',
                    borderRadius: 'var(--lb-radius-sm)',
                    transition: 'background-color var(--lb-transition-normal)',
                  }}
                >
                  <IconBuilding size={14} color="var(--lb-text-muted)" />
                  <Text size="sm" c="dimmed" style={{ color: 'var(--lb-text-secondary)' }}>
                    local
                  </Text>
                  <IconChevronDown size={12} color="var(--lb-text-muted)" />
                </UnstyledButton>
              </Menu.Target>
              <Menu.Dropdown>
                <Menu.Label>Organization</Menu.Label>
                <Menu.Item leftSection={<IconBuilding size={14} />}>
                  Local Development
                </Menu.Item>
              </Menu.Dropdown>
            </Menu>

            <Text c="dimmed" size="sm" className="lb-breadcrumb-separator">/</Text>

            {/* Project Dropdown */}
            <Menu position="bottom-start" shadow="md">
              <Menu.Target>
                <UnstyledButton
                  className="lb-breadcrumb-item"
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 6,
                    padding: '4px 8px',
                    borderRadius: 'var(--lb-radius-sm)',
                    transition: 'background-color var(--lb-transition-normal)',
                  }}
                >
                  <IconDatabase size={14} color="var(--lb-text-secondary)" />
                  <Text size="sm" fw={500} style={{ color: 'var(--lb-text-primary)' }}>
                    {projectName}
                  </Text>
                  <IconChevronDown size={12} color="var(--lb-text-muted)" />
                </UnstyledButton>
              </Menu.Target>
              <Menu.Dropdown>
                <Menu.Label>Project</Menu.Label>
                <Menu.Item leftSection={<IconDatabase size={14} />}>
                  {projectName}
                </Menu.Item>
              </Menu.Dropdown>
            </Menu>

            {/* Status Badge */}
            <Badge
              size="xs"
              variant="light"
              color="green"
              className="lb-header-badge lb-header-badge-local"
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
              styles={{
                root: {
                  borderColor: 'var(--lb-border-default)',
                  color: 'var(--lb-text-primary)',
                  fontWeight: 500,
                  '&:hover': {
                    backgroundColor: 'var(--lb-bg-secondary)',
                    borderColor: 'var(--lb-border-strong)',
                  },
                },
              }}
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
              <ActionIcon variant="subtle" size="lg" color="gray">
                <IconSearch size={18} />
              </ActionIcon>
            </Tooltip>

            {/* Feedback */}
            <Tooltip label="Feedback">
              <ActionIcon variant="subtle" size="lg" color="gray">
                <IconMessageCircle size={18} />
              </ActionIcon>
            </Tooltip>

            {/* Help */}
            <Menu position="bottom-end" shadow="md">
              <Menu.Target>
                <ActionIcon variant="subtle" size="lg" color="gray">
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

            {/* Theme Toggle */}
            <Tooltip label={isDark ? 'Switch to light mode' : 'Switch to dark mode'}>
              <ActionIcon
                variant="subtle"
                size="lg"
                color="gray"
                onClick={() => toggleColorScheme()}
                aria-label="Toggle color scheme"
              >
                {isDark ? <IconSun size={18} /> : <IconMoon size={18} />}
              </ActionIcon>
            </Tooltip>

            {/* Notifications */}
            <ActionIcon variant="subtle" size="lg" color="gray">
              <IconBell size={18} />
            </ActionIcon>

            {/* Settings */}
            <ActionIcon variant="subtle" size="lg" color="gray">
              <IconSettings size={18} />
            </ActionIcon>

            {/* User */}
            <ActionIcon
              variant="subtle"
              size="lg"
              radius="xl"
              color="gray"
              style={{
                backgroundColor: 'var(--lb-bg-secondary)',
              }}
            >
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
          <Text size="sm" c="dimmed" style={{ color: 'var(--lb-text-secondary)' }}>
            Use these credentials to connect to your local Supabase project.
          </Text>

          <Divider label="Connection String" labelPosition="left" />

          <Box>
            <Text size="xs" fw={500} mb={4} style={{ color: 'var(--lb-text-primary)' }}>
              Direct connection
            </Text>
            <Text size="xs" c="dimmed" mb={8} style={{ color: 'var(--lb-text-tertiary)' }}>
              Use this for direct database connections (migrations, admin tasks)
            </Text>
            <ConnectionField value={connectionDetails.directUrl} />
          </Box>

          <Box>
            <Text size="xs" fw={500} mb={4} style={{ color: 'var(--lb-text-primary)' }}>
              Transaction pooler
            </Text>
            <Text size="xs" c="dimmed" mb={8} style={{ color: 'var(--lb-text-tertiary)' }}>
              Use this for serverless functions (short-lived connections)
            </Text>
            <ConnectionField value={connectionDetails.poolerUrl} />
          </Box>

          <Divider label="Connection Details" labelPosition="left" />

          <Group grow>
            <Box>
              <Text size="xs" fw={500} mb={4} style={{ color: 'var(--lb-text-primary)' }}>
                Host
              </Text>
              <ConnectionField value={connectionDetails.host} />
            </Box>
            <Box>
              <Text size="xs" fw={500} mb={4} style={{ color: 'var(--lb-text-primary)' }}>
                Port
              </Text>
              <ConnectionField value={connectionDetails.port} />
            </Box>
          </Group>

          <Group grow>
            <Box>
              <Text size="xs" fw={500} mb={4} style={{ color: 'var(--lb-text-primary)' }}>
                Database
              </Text>
              <ConnectionField value={connectionDetails.database} />
            </Box>
            <Box>
              <Text size="xs" fw={500} mb={4} style={{ color: 'var(--lb-text-primary)' }}>
                User
              </Text>
              <ConnectionField value={connectionDetails.user} />
            </Box>
          </Group>

          <Box>
            <Text size="xs" fw={500} mb={4} style={{ color: 'var(--lb-text-primary)' }}>
              Password
            </Text>
            <ConnectionField value={connectionDetails.password} isSecret />
          </Box>

          <Divider label="API Keys" labelPosition="left" />

          <Box>
            <Text size="xs" fw={500} mb={4} style={{ color: 'var(--lb-text-primary)' }}>
              Project URL
            </Text>
            <ConnectionField value={connectionDetails.projectUrl} />
          </Box>

          <Box>
            <Text size="xs" fw={500} mb={4} style={{ color: 'var(--lb-text-primary)' }}>
              anon key (public)
            </Text>
            <Text size="xs" c="dimmed" mb={8} style={{ color: 'var(--lb-text-tertiary)' }}>
              Safe to use in browsers. Has limited access based on RLS policies.
            </Text>
            <ConnectionField value={connectionDetails.anonKey} />
          </Box>

          <Box>
            <Text size="xs" fw={500} mb={4} style={{ color: 'var(--lb-text-primary)' }}>
              service_role key (secret)
            </Text>
            <Text size="xs" mb={8} style={{ color: 'var(--lb-error-text)' }}>
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
            fontFamily: 'var(--lb-font-mono)',
            fontSize: 11,
            borderTopRightRadius: 0,
            borderBottomRightRadius: 0,
            backgroundColor: 'var(--lb-bg-secondary)',
            borderColor: 'var(--lb-border-default)',
          },
        }}
      />
      {isSecret && (
        <Button
          size="xs"
          variant="default"
          onClick={() => setRevealed(!revealed)}
          style={{
            borderRadius: 0,
            borderColor: 'var(--lb-border-default)',
          }}
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
                borderColor: 'var(--lb-border-default)',
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
