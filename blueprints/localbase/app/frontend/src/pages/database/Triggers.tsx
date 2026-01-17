import { useEffect, useState, useCallback } from 'react';
import {
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
  Loader,
  Center,
  MultiSelect,
  Checkbox,
} from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import {
  IconPlus,
  IconTrash,
  IconRefresh,
  IconBolt,
} from '@tabler/icons-react';
import { notifications } from '@mantine/notifications';
import { PageContainer } from '../../components/layout/PageContainer';
import { EmptyState } from '../../components/common/EmptyState';
import { ConfirmModal } from '../../components/common/ConfirmModal';
import { pgmetaApi, databaseApi } from '../../api';
import type { PGTrigger, PGDBFunction } from '../../api/pgmeta';
import type { Table as TableType } from '../../types';

const TRIGGER_EVENTS = [
  { value: 'INSERT', label: 'INSERT' },
  { value: 'UPDATE', label: 'UPDATE' },
  { value: 'DELETE', label: 'DELETE' },
  { value: 'TRUNCATE', label: 'TRUNCATE' },
];

const TRIGGER_TIMING = [
  { value: 'BEFORE', label: 'BEFORE' },
  { value: 'AFTER', label: 'AFTER' },
  { value: 'INSTEAD OF', label: 'INSTEAD OF' },
];

export function TriggersPage() {
  const [schema, setSchema] = useState('public');
  const [schemas, setSchemas] = useState<string[]>([]);
  const [triggers, setTriggers] = useState<PGTrigger[]>([]);
  const [tables, setTables] = useState<TableType[]>([]);
  const [functions, setFunctions] = useState<PGDBFunction[]>([]);
  const [loading, setLoading] = useState(true);

  const [createModalOpened, { open: openCreateModal, close: closeCreateModal }] =
    useDisclosure(false);
  const [deleteModalOpened, { open: openDeleteModal, close: closeDeleteModal }] =
    useDisclosure(false);

  const [selectedTrigger, setSelectedTrigger] = useState<PGTrigger | null>(null);

  // Form state
  const [triggerName, setTriggerName] = useState('');
  const [triggerTable, setTriggerTable] = useState<string | null>(null);
  const [triggerFunction, setTriggerFunction] = useState<string | null>(null);
  const [triggerTiming, setTriggerTiming] = useState('BEFORE');
  const [triggerEvents, setTriggerEvents] = useState<string[]>(['INSERT']);
  const [triggerForEachRow, setTriggerForEachRow] = useState(true);
  const [triggerCondition, setTriggerCondition] = useState('');
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

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const [triggersData, tablesData, functionsData] = await Promise.all([
        pgmetaApi.listTriggers(schema),
        databaseApi.listTables(schema),
        pgmetaApi.listDatabaseFunctions(`${schema},public`),
      ]);
      setTriggers(triggersData);
      setTables(tablesData);
      // Filter to only trigger functions (those returning trigger)
      const triggerFunctions = functionsData.filter(
        (fn) => fn.return_type === 'trigger' || fn.return_type === 'event_trigger'
      );
      setFunctions(triggerFunctions);
    } catch (error: any) {
      notifications.show({
        title: 'Error',
        message: error.message || 'Failed to load triggers',
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
    fetchData();
  }, [fetchData]);

  const handleCreateTrigger = async () => {
    if (!triggerName.trim() || !triggerTable || !triggerFunction || triggerEvents.length === 0) {
      notifications.show({
        title: 'Validation Error',
        message: 'Trigger name, table, function, and at least one event are required',
        color: 'red',
      });
      return;
    }

    setFormLoading(true);
    try {
      await pgmetaApi.createTrigger({
        schema,
        table: triggerTable,
        name: triggerName,
        function_name: triggerFunction,
        function_schema: schema,
        activation: triggerTiming as 'BEFORE' | 'AFTER' | 'INSTEAD OF',
        events: triggerEvents as Array<'INSERT' | 'UPDATE' | 'DELETE' | 'TRUNCATE'>,
        orientation: triggerForEachRow ? 'ROW' : 'STATEMENT',
        condition: triggerCondition || undefined,
      });

      notifications.show({
        title: 'Success',
        message: 'Trigger created successfully',
        color: 'green',
      });

      closeCreateModal();
      resetForm();
      fetchData();
    } catch (error: any) {
      notifications.show({
        title: 'Error',
        message: error.message || 'Failed to create trigger',
        color: 'red',
      });
    } finally {
      setFormLoading(false);
    }
  };

  const handleDeleteTrigger = async () => {
    if (!selectedTrigger) return;

    setFormLoading(true);
    try {
      await pgmetaApi.dropTrigger(selectedTrigger.schema, selectedTrigger.table, selectedTrigger.name);

      notifications.show({
        title: 'Success',
        message: 'Trigger deleted successfully',
        color: 'green',
      });

      closeDeleteModal();
      setSelectedTrigger(null);
      fetchData();
    } catch (error: any) {
      notifications.show({
        title: 'Error',
        message: error.message || 'Failed to delete trigger',
        color: 'red',
      });
    } finally {
      setFormLoading(false);
    }
  };

  const resetForm = () => {
    setTriggerName('');
    setTriggerTable(null);
    setTriggerFunction(null);
    setTriggerTiming('BEFORE');
    setTriggerEvents(['INSERT']);
    setTriggerForEachRow(true);
    setTriggerCondition('');
  };

  const openDelete = (trigger: PGTrigger) => {
    setSelectedTrigger(trigger);
    openDeleteModal();
  };

  const getEventColor = (event: string) => {
    switch (event) {
      case 'INSERT':
        return 'green';
      case 'UPDATE':
        return 'yellow';
      case 'DELETE':
        return 'red';
      case 'TRUNCATE':
        return 'orange';
      default:
        return 'gray';
    }
  };

  // Group triggers by table
  const triggersByTable = triggers.reduce((acc, trigger) => {
    const key = trigger.table;
    if (!acc[key]) {
      acc[key] = [];
    }
    acc[key].push(trigger);
    return acc;
  }, {} as Record<string, PGTrigger[]>);

  return (
    <PageContainer
      title="Triggers"
      description="Manage database triggers for automated actions"
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
            {triggers.length} triggers
          </Badge>
        </Group>
        <Group gap="sm">
          <ActionIcon variant="subtle" onClick={fetchData}>
            <IconRefresh size={18} />
          </ActionIcon>
          <Button leftSection={<IconPlus size={16} />} onClick={openCreateModal}>
            New Trigger
          </Button>
        </Group>
      </Group>

      {/* Content */}
      {loading ? (
        <Center py="xl">
          <Loader size="lg" />
        </Center>
      ) : triggers.length === 0 ? (
        <EmptyState
          icon={<IconBolt size={48} />}
          title="No triggers found"
          description="Create triggers to automate database actions"
          action={{
            label: 'Create Trigger',
            onClick: openCreateModal,
          }}
        />
      ) : (
        <Stack gap="md">
          {Object.entries(triggersByTable).map(([tableName, tableTriggers]) => (
            <Paper key={tableName} shadow="xs" p="md" withBorder>
              <Text fw={500} mb="md">
                {tableName}
              </Text>
              <Table striped highlightOnHover>
                <Table.Thead>
                  <Table.Tr>
                    <Table.Th>Name</Table.Th>
                    <Table.Th>Timing</Table.Th>
                    <Table.Th>Events</Table.Th>
                    <Table.Th>Function</Table.Th>
                    <Table.Th>Level</Table.Th>
                    <Table.Th>Enabled</Table.Th>
                    <Table.Th w={60}></Table.Th>
                  </Table.Tr>
                </Table.Thead>
                <Table.Tbody>
                  {tableTriggers.map((trigger) => (
                    <Table.Tr key={trigger.name}>
                      <Table.Td>
                        <Group gap="xs">
                          <IconBolt size={14} color="var(--mantine-color-yellow-6)" />
                          <Text size="sm" fw={500}>
                            {trigger.name}
                          </Text>
                        </Group>
                      </Table.Td>
                      <Table.Td>
                        <Badge size="sm" variant="light" color="blue">
                          {trigger.activation}
                        </Badge>
                      </Table.Td>
                      <Table.Td>
                        <Group gap={4}>
                          {trigger.events.map((event) => (
                            <Badge key={event} size="xs" color={getEventColor(event)}>
                              {event}
                            </Badge>
                          ))}
                        </Group>
                      </Table.Td>
                      <Table.Td>
                        <Text size="sm">
                          {trigger.function_schema}.{trigger.function_name}
                        </Text>
                      </Table.Td>
                      <Table.Td>
                        <Badge size="sm" variant="outline">
                          {trigger.orientation}
                        </Badge>
                      </Table.Td>
                      <Table.Td>
                        {trigger.enabled ? (
                          <Badge size="sm" color="green">
                            Enabled
                          </Badge>
                        ) : (
                          <Badge size="sm" color="gray">
                            Disabled
                          </Badge>
                        )}
                      </Table.Td>
                      <Table.Td>
                        <ActionIcon
                          variant="subtle"
                          color="red"
                          onClick={() => openDelete(trigger)}
                        >
                          <IconTrash size={16} />
                        </ActionIcon>
                      </Table.Td>
                    </Table.Tr>
                  ))}
                </Table.Tbody>
              </Table>
            </Paper>
          ))}
        </Stack>
      )}

      {/* Create Trigger Modal */}
      <Modal
        opened={createModalOpened}
        onClose={closeCreateModal}
        title="Create trigger"
        size="md"
      >
        <Stack gap="md">
          <TextInput
            label="Trigger name"
            placeholder="my_trigger"
            value={triggerName}
            onChange={(e) => setTriggerName(e.target.value)}
            required
          />

          <Select
            label="Table"
            description="Select the table to create a trigger on"
            value={triggerTable}
            onChange={setTriggerTable}
            data={tables.map((t) => ({ value: t.name, label: t.name }))}
            placeholder="Select table"
            searchable
            required
          />

          <Select
            label="Trigger function"
            description="Select the function to execute"
            value={triggerFunction}
            onChange={setTriggerFunction}
            data={functions.map((fn) => ({
              value: fn.name,
              label: `${fn.schema}.${fn.name}`,
            }))}
            placeholder="Select function"
            searchable
            required
            nothingFoundMessage={
              functions.length === 0
                ? 'No trigger functions found. Create a function that returns TRIGGER first.'
                : 'No matching functions'
            }
          />

          <Select
            label="Timing"
            description="When the trigger should fire"
            value={triggerTiming}
            onChange={(value) => value && setTriggerTiming(value)}
            data={TRIGGER_TIMING}
          />

          <MultiSelect
            label="Events"
            description="Which events trigger this action"
            value={triggerEvents}
            onChange={setTriggerEvents}
            data={TRIGGER_EVENTS}
            required
          />

          <Checkbox
            label="FOR EACH ROW"
            description="Execute trigger for each affected row (vs. once per statement)"
            checked={triggerForEachRow}
            onChange={(e) => setTriggerForEachRow(e.currentTarget.checked)}
          />

          <TextInput
            label="Condition (optional)"
            description="WHEN clause to conditionally fire the trigger"
            placeholder="(NEW.status = 'active')"
            value={triggerCondition}
            onChange={(e) => setTriggerCondition(e.target.value)}
          />

          <Group justify="flex-end" mt="md">
            <Button variant="outline" onClick={closeCreateModal}>
              Cancel
            </Button>
            <Button onClick={handleCreateTrigger} loading={formLoading}>
              Create Trigger
            </Button>
          </Group>
        </Stack>
      </Modal>

      {/* Delete Confirmation */}
      <ConfirmModal
        opened={deleteModalOpened}
        onClose={closeDeleteModal}
        onConfirm={handleDeleteTrigger}
        title="Delete trigger"
        message={`Are you sure you want to delete the trigger "${selectedTrigger?.name}"? This action cannot be undone.`}
        confirmLabel="Delete"
        danger
        loading={formLoading}
      />
    </PageContainer>
  );
}
