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
  Table,
  Loader,
  Alert,
  Code,
  Modal,
  ScrollArea,
  ActionIcon,
  Tooltip,
  Select,
} from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import {
  IconTerminal2,
  IconSearch,
  IconAlertCircle,
  IconEye,
  IconCode,
  IconRefresh,
} from '@tabler/icons-react';
import { pgmetaApi } from '../../api/pgmeta';

interface DatabaseFunction {
  id: string;
  schema: string;
  name: string;
  language: string;
  definition: string;
  return_type: string;
  argument_types: string;
  type: string;
  security_definer: boolean;
}

export function DatabaseFunctionsPage() {
  const [functions, setFunctions] = useState<DatabaseFunction[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [schemaFilter, setSchemaFilter] = useState<string>('public');
  const [selectedFunction, setSelectedFunction] = useState<DatabaseFunction | null>(null);
  const [viewOpened, { open: openView, close: closeView }] = useDisclosure(false);

  const fetchFunctions = async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await pgmetaApi.listDatabaseFunctions();
      setFunctions(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load functions');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchFunctions();
  }, []);

  const handleViewFunction = (fn: DatabaseFunction) => {
    setSelectedFunction(fn);
    openView();
  };

  const schemas = [...new Set(functions.map((f) => f.schema))].sort();

  const filteredFunctions = functions.filter((fn) => {
    const matchesSearch =
      fn.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      fn.definition?.toLowerCase().includes(searchQuery.toLowerCase());
    const matchesSchema = schemaFilter === 'all' || fn.schema === schemaFilter;
    return matchesSearch && matchesSchema;
  });

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
          <IconTerminal2 size={24} style={{ color: 'var(--mantine-color-green-6)' }} />
          <Title order={3}>Database Functions</Title>
          <Badge variant="light" color="gray">{filteredFunctions.length}</Badge>
        </Group>
        <Group>
          <Select
            size="sm"
            placeholder="Schema"
            value={schemaFilter}
            onChange={(value) => setSchemaFilter(value || 'public')}
            data={[
              { value: 'all', label: 'All schemas' },
              ...schemas.map((s) => ({ value: s, label: s })),
            ]}
            style={{ width: 150 }}
          />
          <TextInput
            placeholder="Search functions..."
            leftSection={<IconSearch size={16} />}
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            style={{ width: 300 }}
          />
          <Tooltip label="Refresh">
            <ActionIcon variant="light" onClick={fetchFunctions}>
              <IconRefresh size={18} />
            </ActionIcon>
          </Tooltip>
        </Group>
      </Group>

      {error && (
        <Alert icon={<IconAlertCircle size={16} />} color="red" mb="md">
          {error}
        </Alert>
      )}

      <Card withBorder padding={0}>
        <ScrollArea>
          <Table striped highlightOnHover>
            <Table.Thead>
              <Table.Tr>
                <Table.Th>Name</Table.Th>
                <Table.Th>Schema</Table.Th>
                <Table.Th>Arguments</Table.Th>
                <Table.Th>Returns</Table.Th>
                <Table.Th>Language</Table.Th>
                <Table.Th>Type</Table.Th>
                <Table.Th style={{ width: 80 }}>Actions</Table.Th>
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>
              {filteredFunctions.map((fn) => (
                <Table.Tr key={fn.id}>
                  <Table.Td>
                    <Group gap="xs">
                      <IconTerminal2 size={14} style={{ color: 'var(--mantine-color-gray-5)' }} />
                      <Text size="sm" fw={500}>{fn.name}</Text>
                    </Group>
                  </Table.Td>
                  <Table.Td>
                    <Badge variant="light" color="gray" size="sm">
                      {fn.schema}
                    </Badge>
                  </Table.Td>
                  <Table.Td>
                    <Text size="xs" c="dimmed" lineClamp={1} style={{ maxWidth: 200 }}>
                      {fn.argument_types || 'none'}
                    </Text>
                  </Table.Td>
                  <Table.Td>
                    <Code style={{ fontSize: '11px' }}>{fn.return_type}</Code>
                  </Table.Td>
                  <Table.Td>
                    <Badge
                      variant="light"
                      color={fn.language === 'plpgsql' ? 'blue' : fn.language === 'sql' ? 'green' : 'gray'}
                      size="sm"
                    >
                      {fn.language}
                    </Badge>
                  </Table.Td>
                  <Table.Td>
                    <Badge
                      variant="outline"
                      color={fn.security_definer ? 'orange' : 'gray'}
                      size="xs"
                    >
                      {fn.security_definer ? 'DEFINER' : 'INVOKER'}
                    </Badge>
                  </Table.Td>
                  <Table.Td>
                    <Tooltip label="View definition">
                      <ActionIcon variant="subtle" onClick={() => handleViewFunction(fn)}>
                        <IconEye size={16} />
                      </ActionIcon>
                    </Tooltip>
                  </Table.Td>
                </Table.Tr>
              ))}
              {filteredFunctions.length === 0 && (
                <Table.Tr>
                  <Table.Td colSpan={7}>
                    <Text size="sm" c="dimmed" ta="center" py="xl">
                      No functions found
                    </Text>
                  </Table.Td>
                </Table.Tr>
              )}
            </Table.Tbody>
          </Table>
        </ScrollArea>
      </Card>

      {/* View Function Modal */}
      <Modal
        opened={viewOpened}
        onClose={closeView}
        title={
          <Group gap="xs">
            <IconCode size={18} />
            <Text fw={600}>{selectedFunction?.name}</Text>
          </Group>
        }
        size="xl"
      >
        {selectedFunction && (
          <Stack gap="md">
            <Group>
              <Badge variant="light" color="gray">
                {selectedFunction.schema}
              </Badge>
              <Badge variant="light" color="blue">
                {selectedFunction.language}
              </Badge>
              <Badge variant="outline" color={selectedFunction.security_definer ? 'orange' : 'gray'}>
                {selectedFunction.security_definer ? 'SECURITY DEFINER' : 'SECURITY INVOKER'}
              </Badge>
            </Group>

            <Box>
              <Text size="sm" fw={500} mb="xs">Arguments</Text>
              <Code block style={{ fontSize: '12px' }}>
                {selectedFunction.argument_types || 'none'}
              </Code>
            </Box>

            <Box>
              <Text size="sm" fw={500} mb="xs">Returns</Text>
              <Code block style={{ fontSize: '12px' }}>
                {selectedFunction.return_type}
              </Code>
            </Box>

            <Box>
              <Text size="sm" fw={500} mb="xs">Definition</Text>
              <ScrollArea h={300}>
                <Code block style={{ fontSize: '12px', whiteSpace: 'pre-wrap' }}>
                  {selectedFunction.definition}
                </Code>
              </ScrollArea>
            </Box>
          </Stack>
        )}
      </Modal>
    </Box>
  );
}
