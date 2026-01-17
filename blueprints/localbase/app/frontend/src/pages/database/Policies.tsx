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
  Modal,
  TextInput,
  Textarea,
  Loader,
  Center,
  Code,
  MultiSelect,
  Tabs,
  Tooltip,
  Collapse,
} from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import {
  IconPlus,
  IconTrash,
  IconRefresh,
  IconShield,
  IconShieldCheck,
  IconShieldOff,
  IconChevronDown,
  IconChevronRight,
} from '@tabler/icons-react';
import { notifications } from '@mantine/notifications';
import { PageContainer } from '../../components/layout/PageContainer';
import { EmptyState } from '../../components/common/EmptyState';
import { ConfirmModal } from '../../components/common/ConfirmModal';
import { databaseApi } from '../../api';
import type { Table as TableType, Policy } from '../../types';

const POLICY_TEMPLATES = [
  {
    name: 'Enable read access for all users',
    command: 'SELECT',
    definition: 'true',
    description: 'Allow all users to read all rows',
  },
  {
    name: 'Enable insert for authenticated users only',
    command: 'INSERT',
    definition: '(auth.role() = \'authenticated\')',
    description: 'Only authenticated users can insert rows',
  },
  {
    name: 'Enable read access for users based on user_id',
    command: 'SELECT',
    definition: '(auth.uid() = user_id)',
    description: 'Users can only read rows where user_id matches their ID',
  },
  {
    name: 'Enable update for users based on user_id',
    command: 'UPDATE',
    definition: '(auth.uid() = user_id)',
    check_expression: '(auth.uid() = user_id)',
    description: 'Users can only update their own rows',
  },
  {
    name: 'Enable delete for users based on user_id',
    command: 'DELETE',
    definition: '(auth.uid() = user_id)',
    description: 'Users can only delete their own rows',
  },
];

interface TablePolicies {
  table: TableType;
  policies: Policy[];
  expanded: boolean;
}

export function PoliciesPage() {
  const [schema, setSchema] = useState('public');
  const [schemas, setSchemas] = useState<string[]>([]);
  const [tablePolicies, setTablePolicies] = useState<TablePolicies[]>([]);
  const [loading, setLoading] = useState(true);

  const [createModalOpened, { open: openCreateModal, close: closeCreateModal }] =
    useDisclosure(false);
  const [deleteModalOpened, { open: openDeleteModal, close: closeDeleteModal }] =
    useDisclosure(false);

  const [selectedTable, setSelectedTable] = useState<TableType | null>(null);
  const [selectedPolicy, setSelectedPolicy] = useState<Policy | null>(null);

  // Form state
  const [policyName, setPolicyName] = useState('');
  const [policyCommand, setPolicyCommand] = useState<string>('SELECT');
  const [policyDefinition, setPolicyDefinition] = useState('');
  const [policyCheck, setPolicyCheck] = useState('');
  const [policyRoles, setPolicyRoles] = useState<string[]>(['public']);
  const [formLoading, setFormLoading] = useState(false);

  const fetchSchemas = useCallback(async () => {
    try {
      const data = await databaseApi.listSchemas();
      setSchemas(data);
    } catch (error: any) {
      notifications.show({
        title: 'Error',
        message: error.message || 'Failed to load schemas',
        color: 'red',
      });
    }
  }, []);

  const fetchPolicies = useCallback(async () => {
    setLoading(true);
    try {
      const tables = await databaseApi.listTables(schema);

      // Fetch policies for each table
      const policiesPromises = tables.map(async (table) => {
        try {
          const policies = await databaseApi.listPolicies(schema, table.name);
          return { table, policies, expanded: false };
        } catch {
          return { table, policies: [], expanded: false };
        }
      });

      const results = await Promise.all(policiesPromises);
      setTablePolicies(results);
    } catch (error: any) {
      notifications.show({
        title: 'Error',
        message: error.message || 'Failed to load policies',
        color: 'red',
      });
    } finally {
      setLoading(false);
    }
  }, [schema]);

  useEffect(() => {
    fetchSchemas();
  }, [fetchSchemas]);

  useEffect(() => {
    fetchPolicies();
  }, [fetchPolicies]);

  const toggleTableExpand = (tableName: string) => {
    setTablePolicies((prev) =>
      prev.map((tp) =>
        tp.table.name === tableName ? { ...tp, expanded: !tp.expanded } : tp
      )
    );
  };

  const handleToggleRLS = async (table: TableType) => {
    try {
      // RLS toggle would need a dedicated API endpoint
      // For now, show the current status
      notifications.show({
        title: 'Info',
        message: `RLS is currently ${table.rls_enabled ? 'enabled' : 'disabled'} for ${table.name}`,
        color: 'blue',
      });
    } catch (error: any) {
      notifications.show({
        title: 'Error',
        message: error.message || 'Failed to toggle RLS',
        color: 'red',
      });
    }
  };

  const handleCreatePolicy = async () => {
    if (!selectedTable || !policyName.trim() || !policyDefinition.trim()) {
      notifications.show({
        title: 'Validation Error',
        message: 'Policy name and definition are required',
        color: 'red',
      });
      return;
    }

    setFormLoading(true);
    try {
      await databaseApi.createPolicy({
        name: policyName,
        schema: schema,
        table: selectedTable.name,
        command: policyCommand as 'ALL' | 'SELECT' | 'INSERT' | 'UPDATE' | 'DELETE',
        definition: policyDefinition,
        check_expression: policyCheck || undefined,
        roles: policyRoles.length > 0 ? policyRoles : undefined,
      });

      notifications.show({
        title: 'Success',
        message: 'Policy created successfully',
        color: 'green',
      });

      closeCreateModal();
      resetForm();
      fetchPolicies();
    } catch (error: any) {
      notifications.show({
        title: 'Error',
        message: error.message || 'Failed to create policy',
        color: 'red',
      });
    } finally {
      setFormLoading(false);
    }
  };

  const handleDeletePolicy = async () => {
    if (!selectedPolicy || !selectedTable) return;

    setFormLoading(true);
    try {
      await databaseApi.dropPolicy(schema, selectedTable.name, selectedPolicy.name);

      notifications.show({
        title: 'Success',
        message: 'Policy deleted successfully',
        color: 'green',
      });

      closeDeleteModal();
      setSelectedPolicy(null);
      fetchPolicies();
    } catch (error: any) {
      notifications.show({
        title: 'Error',
        message: error.message || 'Failed to delete policy',
        color: 'red',
      });
    } finally {
      setFormLoading(false);
    }
  };

  const resetForm = () => {
    setPolicyName('');
    setPolicyCommand('SELECT');
    setPolicyDefinition('');
    setPolicyCheck('');
    setPolicyRoles(['public']);
    setSelectedTable(null);
  };

  const applyTemplate = (template: typeof POLICY_TEMPLATES[0]) => {
    setPolicyName(template.name.toLowerCase().replace(/\s+/g, '_'));
    setPolicyCommand(template.command);
    setPolicyDefinition(template.definition);
    if ('check_expression' in template) {
      setPolicyCheck((template as any).check_expression || '');
    } else {
      setPolicyCheck('');
    }
  };

  const openCreateForTable = (table: TableType) => {
    setSelectedTable(table);
    resetForm();
    openCreateModal();
  };

  const openDeleteForPolicy = (table: TableType, policy: Policy) => {
    setSelectedTable(table);
    setSelectedPolicy(policy);
    openDeleteModal();
  };

  const getCommandColor = (command: string) => {
    switch (command) {
      case 'SELECT':
        return 'blue';
      case 'INSERT':
        return 'green';
      case 'UPDATE':
        return 'yellow';
      case 'DELETE':
        return 'red';
      case 'ALL':
        return 'grape';
      default:
        return 'gray';
    }
  };

  const totalPolicies = tablePolicies.reduce((acc, tp) => acc + tp.policies.length, 0);
  const tablesWithRLS = tablePolicies.filter((tp) => tp.table.rls_enabled).length;

  return (
    <PageContainer
      title="Row Level Security"
      description="Manage access policies for your database tables"
    >
      {/* Header */}
      <Group justify="space-between" mb="lg">
        <Group gap="md">
          <Select
            size="sm"
            value={schema}
            onChange={(value) => value && setSchema(value)}
            data={schemas.map((s) => ({ value: s, label: s }))}
            placeholder="Select schema"
            w={150}
          />
          <Badge variant="light" color="blue" size="lg">
            {totalPolicies} policies
          </Badge>
          <Badge variant="light" color="green" size="lg">
            {tablesWithRLS} tables with RLS
          </Badge>
        </Group>
        <ActionIcon variant="subtle" onClick={fetchPolicies}>
          <IconRefresh size={18} />
        </ActionIcon>
      </Group>

      {/* Content */}
      {loading ? (
        <Center py="xl">
          <Loader size="lg" />
        </Center>
      ) : tablePolicies.length === 0 ? (
        <EmptyState
          icon={<IconShield size={48} />}
          title="No tables found"
          description="Create tables first to manage RLS policies"
        />
      ) : (
        <Stack gap="md">
          {tablePolicies.map(({ table, policies, expanded }) => (
            <Paper key={table.name} shadow="xs" p={0} withBorder>
              {/* Table Header */}
              <Box
                p="md"
                style={{
                  cursor: 'pointer',
                  backgroundColor: expanded ? 'var(--mantine-color-gray-0)' : undefined,
                }}
                onClick={() => toggleTableExpand(table.name)}
              >
                <Group justify="space-between">
                  <Group gap="md">
                    {expanded ? <IconChevronDown size={18} /> : <IconChevronRight size={18} />}
                    <Text fw={500}>{table.name}</Text>
                    <Badge size="sm" variant="light" color={table.rls_enabled ? 'green' : 'gray'}>
                      {table.rls_enabled ? 'RLS Enabled' : 'RLS Disabled'}
                    </Badge>
                    <Badge size="sm" variant="outline">
                      {policies.length} {policies.length === 1 ? 'policy' : 'policies'}
                    </Badge>
                  </Group>
                  <Group gap="sm" onClick={(e) => e.stopPropagation()}>
                    <Tooltip label={table.rls_enabled ? 'Disable RLS' : 'Enable RLS'}>
                      <ActionIcon
                        variant="subtle"
                        color={table.rls_enabled ? 'green' : 'gray'}
                        onClick={() => handleToggleRLS(table)}
                      >
                        {table.rls_enabled ? <IconShieldCheck size={18} /> : <IconShieldOff size={18} />}
                      </ActionIcon>
                    </Tooltip>
                    <Button
                      size="xs"
                      variant="light"
                      leftSection={<IconPlus size={14} />}
                      onClick={() => openCreateForTable(table)}
                    >
                      New Policy
                    </Button>
                  </Group>
                </Group>
              </Box>

              {/* Policies List */}
              <Collapse in={expanded}>
                <Box px="md" pb="md">
                  {policies.length === 0 ? (
                    <Text size="sm" c="dimmed" ta="center" py="md">
                      No policies defined. Add a policy to control access.
                    </Text>
                  ) : (
                    <Table striped highlightOnHover>
                      <Table.Thead>
                        <Table.Tr>
                          <Table.Th>Name</Table.Th>
                          <Table.Th>Command</Table.Th>
                          <Table.Th>Definition (USING)</Table.Th>
                          <Table.Th>Check (WITH CHECK)</Table.Th>
                          <Table.Th>Roles</Table.Th>
                          <Table.Th w={60}></Table.Th>
                        </Table.Tr>
                      </Table.Thead>
                      <Table.Tbody>
                        {policies.map((policy) => (
                          <Table.Tr key={policy.name}>
                            <Table.Td>
                              <Text size="sm" fw={500}>
                                {policy.name}
                              </Text>
                            </Table.Td>
                            <Table.Td>
                              <Badge size="sm" color={getCommandColor(policy.command)}>
                                {policy.command}
                              </Badge>
                            </Table.Td>
                            <Table.Td>
                              <Code block style={{ maxWidth: 300 }}>
                                {policy.definition}
                              </Code>
                            </Table.Td>
                            <Table.Td>
                              {policy.check_expression ? (
                                <Code block style={{ maxWidth: 200 }}>
                                  {policy.check_expression}
                                </Code>
                              ) : (
                                <Text size="sm" c="dimmed">
                                  —
                                </Text>
                              )}
                            </Table.Td>
                            <Table.Td>
                              <Group gap={4}>
                                {policy.roles?.map((role) => (
                                  <Badge key={role} size="xs" variant="outline">
                                    {role}
                                  </Badge>
                                ))}
                              </Group>
                            </Table.Td>
                            <Table.Td>
                              <ActionIcon
                                variant="subtle"
                                color="red"
                                onClick={() => openDeleteForPolicy(table, policy)}
                              >
                                <IconTrash size={16} />
                              </ActionIcon>
                            </Table.Td>
                          </Table.Tr>
                        ))}
                      </Table.Tbody>
                    </Table>
                  )}
                </Box>
              </Collapse>
            </Paper>
          ))}
        </Stack>
      )}

      {/* Create Policy Modal */}
      <Modal
        opened={createModalOpened}
        onClose={closeCreateModal}
        title={`Create policy for ${selectedTable?.name}`}
        size="lg"
      >
        <Tabs defaultValue="custom">
          <Tabs.List mb="md">
            <Tabs.Tab value="custom">Custom Policy</Tabs.Tab>
            <Tabs.Tab value="templates">Templates</Tabs.Tab>
          </Tabs.List>

          <Tabs.Panel value="custom">
            <Stack gap="md">
              <TextInput
                label="Policy name"
                placeholder="policy_name"
                value={policyName}
                onChange={(e) => setPolicyName(e.target.value)}
                required
              />

              <Select
                label="Command"
                description="Which SQL command this policy applies to"
                value={policyCommand}
                onChange={(value) => value && setPolicyCommand(value)}
                data={[
                  { value: 'ALL', label: 'ALL - All commands' },
                  { value: 'SELECT', label: 'SELECT - Read access' },
                  { value: 'INSERT', label: 'INSERT - Create access' },
                  { value: 'UPDATE', label: 'UPDATE - Modify access' },
                  { value: 'DELETE', label: 'DELETE - Delete access' },
                ]}
              />

              <Textarea
                label="USING expression"
                description="This expression is used to determine which rows are visible"
                placeholder="(auth.uid() = user_id)"
                value={policyDefinition}
                onChange={(e) => setPolicyDefinition(e.target.value)}
                minRows={3}
                required
              />

              {(policyCommand === 'INSERT' || policyCommand === 'UPDATE' || policyCommand === 'ALL') && (
                <Textarea
                  label="WITH CHECK expression"
                  description="This expression is used for INSERT and UPDATE to determine which rows can be created/modified"
                  placeholder="(auth.uid() = user_id)"
                  value={policyCheck}
                  onChange={(e) => setPolicyCheck(e.target.value)}
                  minRows={3}
                />
              )}

              <MultiSelect
                label="Roles"
                description="Which roles this policy applies to"
                value={policyRoles}
                onChange={setPolicyRoles}
                data={['public', 'authenticated', 'anon', 'service_role']}
                placeholder="Select roles"
              />

              <Group justify="flex-end" mt="md">
                <Button variant="outline" onClick={closeCreateModal}>
                  Cancel
                </Button>
                <Button onClick={handleCreatePolicy} loading={formLoading}>
                  Create Policy
                </Button>
              </Group>
            </Stack>
          </Tabs.Panel>

          <Tabs.Panel value="templates">
            <Stack gap="md">
              {POLICY_TEMPLATES.map((template, index) => (
                <Paper
                  key={index}
                  p="md"
                  withBorder
                  style={{ cursor: 'pointer' }}
                  onClick={() => applyTemplate(template)}
                >
                  <Group justify="space-between" mb="xs">
                    <Text fw={500}>{template.name}</Text>
                    <Badge size="sm" color={getCommandColor(template.command)}>
                      {template.command}
                    </Badge>
                  </Group>
                  <Text size="sm" c="dimmed" mb="xs">
                    {template.description}
                  </Text>
                  <Code block>{template.definition}</Code>
                </Paper>
              ))}
              <Text size="sm" c="dimmed" ta="center">
                Click a template to apply it to the custom policy form
              </Text>
            </Stack>
          </Tabs.Panel>
        </Tabs>
      </Modal>

      {/* Delete Confirmation */}
      <ConfirmModal
        opened={deleteModalOpened}
        onClose={closeDeleteModal}
        onConfirm={handleDeletePolicy}
        title="Delete policy"
        message={`Are you sure you want to delete the policy "${selectedPolicy?.name}"? This action cannot be undone.`}
        confirmLabel="Delete"
        danger
        loading={formLoading}
      />
    </PageContainer>
  );
}
