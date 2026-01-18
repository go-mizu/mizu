import { useEffect, useState } from 'react';
import {
  Box,
  Title,
  Text,
  Card,
  Group,
  Badge,
  Button,
  TextInput,
  Stack,
  SimpleGrid,
  Loader,
  Alert,
  Switch,
  Tooltip,
} from '@mantine/core';
import {
  IconPuzzle,
  IconSearch,
  IconAlertCircle,
  IconCheck,
} from '@tabler/icons-react';
import { databaseApi } from '../../api/database';
import type { Extension } from '../../types';

export function ExtensionsPage() {
  const [extensions, setExtensions] = useState<Extension[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [enabling, setEnabling] = useState<string | null>(null);

  const fetchExtensions = async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await databaseApi.listExtensions();
      setExtensions(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load extensions');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchExtensions();
  }, []);

  const handleToggleExtension = async (ext: Extension) => {
    try {
      setEnabling(ext.name);
      if (ext.installed_version) {
        // Can't disable extensions easily, show a message
        alert('To disable an extension, use SQL: DROP EXTENSION IF EXISTS ' + ext.name);
      } else {
        await databaseApi.enableExtension(ext.name);
        await fetchExtensions();
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to toggle extension');
    } finally {
      setEnabling(null);
    }
  };

  const filteredExtensions = extensions.filter((ext) =>
    ext.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
    (ext.comment && ext.comment.toLowerCase().includes(searchQuery.toLowerCase()))
  );

  const installedExtensions = filteredExtensions.filter((ext) => ext.installed_version);
  const availableExtensions = filteredExtensions.filter((ext) => !ext.installed_version);

  if (loading) {
    return (
      <Box p="xl" style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100%' }}>
        <Loader size="lg" />
      </Box>
    );
  }

  return (
    <Box p="md">
      <Group justify="space-between" mb="lg">
        <Group gap="xs">
          <IconPuzzle size={24} style={{ color: 'var(--mantine-color-green-6)' }} />
          <Title order={3}>Extensions</Title>
        </Group>
        <TextInput
          placeholder="Search extensions..."
          leftSection={<IconSearch size={16} />}
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          style={{ width: 300 }}
        />
      </Group>

      {error && (
        <Alert icon={<IconAlertCircle size={16} />} color="red" mb="md">
          {error}
        </Alert>
      )}

      {/* Installed Extensions */}
      <Box mb="xl">
        <Text fw={600} size="sm" mb="sm" c="dimmed">
          INSTALLED ({installedExtensions.length})
        </Text>
        <SimpleGrid cols={{ base: 1, sm: 2, lg: 3 }} spacing="md">
          {installedExtensions.map((ext) => (
            <Card key={ext.name} withBorder padding="md">
              <Group justify="space-between" mb="xs">
                <Group gap="xs">
                  <IconPuzzle size={18} style={{ color: 'var(--mantine-color-green-6)' }} />
                  <Text fw={600} size="sm">{ext.name}</Text>
                </Group>
                <Badge color="green" variant="light" size="sm">
                  v{ext.installed_version}
                </Badge>
              </Group>
              <Text size="xs" c="dimmed" lineClamp={2} mb="sm">
                {ext.comment || 'No description available'}
              </Text>
              <Group justify="space-between">
                <Text size="xs" c="dimmed">
                  Default: v{ext.default_version}
                </Text>
                <Tooltip label="Extension is installed">
                  <IconCheck size={16} style={{ color: 'var(--mantine-color-green-6)' }} />
                </Tooltip>
              </Group>
            </Card>
          ))}
        </SimpleGrid>
        {installedExtensions.length === 0 && (
          <Text size="sm" c="dimmed" ta="center" py="xl">
            No installed extensions found
          </Text>
        )}
      </Box>

      {/* Available Extensions */}
      <Box>
        <Text fw={600} size="sm" mb="sm" c="dimmed">
          AVAILABLE ({availableExtensions.length})
        </Text>
        <SimpleGrid cols={{ base: 1, sm: 2, lg: 3 }} spacing="md">
          {availableExtensions.map((ext) => (
            <Card key={ext.name} withBorder padding="md">
              <Group justify="space-between" mb="xs">
                <Group gap="xs">
                  <IconPuzzle size={18} style={{ color: 'var(--mantine-color-gray-5)' }} />
                  <Text fw={600} size="sm">{ext.name}</Text>
                </Group>
                <Badge color="gray" variant="light" size="sm">
                  v{ext.default_version}
                </Badge>
              </Group>
              <Text size="xs" c="dimmed" lineClamp={2} mb="sm">
                {ext.comment || 'No description available'}
              </Text>
              <Button
                size="xs"
                variant="light"
                onClick={() => handleToggleExtension(ext)}
                loading={enabling === ext.name}
                fullWidth
              >
                Enable
              </Button>
            </Card>
          ))}
        </SimpleGrid>
        {availableExtensions.length === 0 && (
          <Text size="sm" c="dimmed" ta="center" py="xl">
            No additional extensions available
          </Text>
        )}
      </Box>
    </Box>
  );
}
