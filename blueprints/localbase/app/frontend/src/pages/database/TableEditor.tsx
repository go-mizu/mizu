import { useEffect, useState, useCallback } from 'react';
import {
  Box,
  Button,
  Group,
  Text,
  Stack,
  Paper,
  Table,
  ScrollArea,
  Select,
  ActionIcon,
  Menu,
  Badge,
  Modal,
  TextInput,
  Loader,
  Center,
} from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import {
  IconPlus,
  IconTrash,
  IconRefresh,
  IconDotsVertical,
  IconTable,
  IconKey,
} from '@tabler/icons-react';
import { notifications } from '@mantine/notifications';
import { PageContainer } from '../../components/layout/PageContainer';
import { EmptyState } from '../../components/common/EmptyState';
import { ConfirmModal } from '../../components/common/ConfirmModal';
import { databaseApi } from '../../api';
import { useAppStore } from '../../stores/appStore';
import type { Table as TableType, Column } from '../../types';

export function TableEditorPage() {
  const { selectedSchema, setSelectedSchema, selectedTable, setSelectedTable } = useAppStore();

  const [schemas, setSchemas] = useState<string[]>([]);
  const [tables, setTables] = useState<TableType[]>([]);
  const [columns, setColumns] = useState<Column[]>([]);
  const [tableData, setTableData] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [dataLoading, setDataLoading] = useState(false);

  const [createTableOpened, { open: openCreateTable, close: closeCreateTable }] =
    useDisclosure(false);
  const [deleteTableOpened, { open: openDeleteTable, close: closeDeleteTable }] =
    useDisclosure(false);

  const [newTableName, setNewTableName] = useState('');
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

  const fetchTables = useCallback(async () => {
    setLoading(true);
    try {
      const data = await databaseApi.listTables(selectedSchema);
      setTables(data);
      if (data.length > 0 && !selectedTable) {
        setSelectedTable(data[0].name);
      }
    } catch (error: any) {
      notifications.show({
        title: 'Error',
        message: error.message || 'Failed to load tables',
        color: 'red',
      });
    } finally {
      setLoading(false);
    }
  }, [selectedSchema, selectedTable, setSelectedTable]);

  const fetchTableData = useCallback(async () => {
    if (!selectedTable) return;

    setDataLoading(true);
    try {
      const [columnsData, rowsData] = await Promise.all([
        databaseApi.listColumns(selectedSchema, selectedTable),
        databaseApi.selectTable(selectedTable, 'limit=100'),
      ]);
      setColumns(columnsData);
      setTableData(rowsData);
    } catch (error: any) {
      notifications.show({
        title: 'Error',
        message: error.message || 'Failed to load table data',
        color: 'red',
      });
      setColumns([]);
      setTableData([]);
    } finally {
      setDataLoading(false);
    }
  }, [selectedSchema, selectedTable]);

  useEffect(() => {
    fetchSchemas();
  }, [fetchSchemas]);

  useEffect(() => {
    fetchTables();
  }, [fetchTables]);

  useEffect(() => {
    if (selectedTable) {
      fetchTableData();
    }
  }, [selectedTable, fetchTableData]);

  const handleCreateTable = async () => {
    if (!newTableName.trim()) {
      notifications.show({
        title: 'Validation Error',
        message: 'Table name is required',
        color: 'red',
      });
      return;
    }

    setFormLoading(true);
    try {
      await databaseApi.createTable({
        schema: selectedSchema,
        name: newTableName,
        columns: [
          {
            name: 'id',
            type: 'uuid',
            is_primary_key: true,
            is_nullable: false,
            is_unique: true,
            default_value: 'gen_random_uuid()',
          },
          {
            name: 'created_at',
            type: 'timestamptz',
            is_nullable: false,
            is_primary_key: false,
            is_unique: false,
            default_value: 'now()',
          },
        ],
      });
      notifications.show({
        title: 'Success',
        message: 'Table created successfully',
        color: 'green',
      });
      closeCreateTable();
      setNewTableName('');
      fetchTables();
    } catch (error: any) {
      notifications.show({
        title: 'Error',
        message: error.message || 'Failed to create table',
        color: 'red',
      });
    } finally {
      setFormLoading(false);
    }
  };

  const handleDeleteTable = async () => {
    if (!selectedTable) return;

    setFormLoading(true);
    try {
      await databaseApi.dropTable(selectedSchema, selectedTable);
      notifications.show({
        title: 'Success',
        message: 'Table deleted successfully',
        color: 'green',
      });
      closeDeleteTable();
      setSelectedTable(null);
      fetchTables();
    } catch (error: any) {
      notifications.show({
        title: 'Error',
        message: error.message || 'Failed to delete table',
        color: 'red',
      });
    } finally {
      setFormLoading(false);
    }
  };

  const currentTable = tables.find((t) => t.name === selectedTable);

  return (
    <PageContainer title="Table Editor" description="View and edit your database tables" fullWidth noPadding>
      <Box style={{ display: 'flex', height: 'calc(100vh - 140px)' }}>
        {/* Table Sidebar */}
        <Box
          style={{
            width: 280,
            borderRight: '1px solid var(--supabase-border)',
            display: 'flex',
            flexDirection: 'column',
          }}
        >
          <Box p="md" pb="sm">
            <Group justify="space-between" mb="sm">
              <Text fw={600} size="sm">
                Tables
              </Text>
              <ActionIcon variant="subtle" onClick={openCreateTable} size="sm">
                <IconPlus size={16} />
              </ActionIcon>
            </Group>
            <Select
              size="xs"
              value={selectedSchema}
              onChange={(value) => {
                if (value) {
                  setSelectedSchema(value);
                  setSelectedTable(null);
                }
              }}
              data={schemas.map((s) => ({ value: s, label: s }))}
              placeholder="Select schema"
            />
          </Box>

          <Box style={{ flex: 1, overflow: 'auto' }} px="sm" pb="sm">
            {loading ? (
              <Center py="xl">
                <Loader size="sm" />
              </Center>
            ) : tables.length === 0 ? (
              <Text size="sm" c="dimmed" ta="center" py="xl">
                No tables in {selectedSchema}
              </Text>
            ) : (
              <Stack gap={4}>
                {tables.map((table) => (
                  <Paper
                    key={table.name}
                    p="xs"
                    style={{
                      cursor: 'pointer',
                      backgroundColor:
                        selectedTable === table.name
                          ? 'var(--supabase-brand-light)'
                          : 'transparent',
                      borderRadius: 6,
                    }}
                    onClick={() => setSelectedTable(table.name)}
                  >
                    <Group justify="space-between" wrap="nowrap">
                      <Group gap="xs" wrap="nowrap">
                        <IconTable size={16} />
                        <Text size="sm" truncate style={{ maxWidth: 150 }}>
                          {table.name}
                        </Text>
                      </Group>
                      <Badge size="xs" variant="light" color="gray">
                        {table.row_count}
                      </Badge>
                    </Group>
                  </Paper>
                ))}
              </Stack>
            )}
          </Box>
        </Box>

        {/* Data Grid */}
        <Box style={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
          {selectedTable ? (
            <>
              {/* Toolbar */}
              <Box
                p="md"
                style={{ borderBottom: '1px solid var(--supabase-border)' }}
              >
                <Group justify="space-between">
                  <Group gap="xs">
                    <Text fw={500}>{selectedTable}</Text>
                    {currentTable?.rls_enabled && (
                      <Badge size="sm" variant="light" color="green">
                        RLS enabled
                      </Badge>
                    )}
                  </Group>
                  <Group gap="sm">
                    <ActionIcon variant="subtle" onClick={fetchTableData}>
                      <IconRefresh size={18} />
                    </ActionIcon>
                    <Menu position="bottom-end">
                      <Menu.Target>
                        <ActionIcon variant="subtle">
                          <IconDotsVertical size={18} />
                        </ActionIcon>
                      </Menu.Target>
                      <Menu.Dropdown>
                        <Menu.Item
                          color="red"
                          leftSection={<IconTrash size={14} />}
                          onClick={openDeleteTable}
                        >
                          Delete table
                        </Menu.Item>
                      </Menu.Dropdown>
                    </Menu>
                  </Group>
                </Group>
              </Box>

              {/* Column Headers Info */}
              {columns.length > 0 && (
                <Box
                  px="md"
                  py="xs"
                  style={{
                    borderBottom: '1px solid var(--supabase-border)',
                    backgroundColor: 'var(--supabase-bg-surface)',
                  }}
                >
                  <Group gap="xs">
                    <Text size="xs" c="dimmed">
                      {columns.length} columns
                    </Text>
                    <Text size="xs" c="dimmed">
                      •
                    </Text>
                    <Text size="xs" c="dimmed">
                      {tableData.length} rows (showing max 100)
                    </Text>
                  </Group>
                </Box>
              )}

              {/* Data Table */}
              <ScrollArea style={{ flex: 1 }}>
                {dataLoading ? (
                  <Center py="xl">
                    <Loader size="sm" />
                  </Center>
                ) : columns.length === 0 ? (
                  <Box p="xl" ta="center">
                    <Text c="dimmed">No columns in this table</Text>
                  </Box>
                ) : tableData.length === 0 ? (
                  <Box p="xl" ta="center">
                    <Text c="dimmed">No data in this table</Text>
                  </Box>
                ) : (
                  <Table striped highlightOnHover stickyHeader>
                    <Table.Thead>
                      <Table.Tr>
                        {columns.map((col) => (
                          <Table.Th key={col.name}>
                            <Group gap={4} wrap="nowrap">
                              {col.is_primary_key && (
                                <IconKey size={12} color="var(--supabase-brand)" />
                              )}
                              <Text size="xs" fw={500}>
                                {col.name}
                              </Text>
                              <Text size="xs" c="dimmed">
                                {col.type}
                              </Text>
                            </Group>
                          </Table.Th>
                        ))}
                      </Table.Tr>
                    </Table.Thead>
                    <Table.Tbody>
                      {tableData.map((row, rowIndex) => (
                        <Table.Tr key={rowIndex}>
                          {columns.map((col) => (
                            <Table.Td key={col.name}>
                              <Text size="sm" style={{ maxWidth: 300 }} truncate>
                                {row[col.name] === null
                                  ? 'NULL'
                                  : typeof row[col.name] === 'object'
                                    ? JSON.stringify(row[col.name])
                                    : String(row[col.name])}
                              </Text>
                            </Table.Td>
                          ))}
                        </Table.Tr>
                      ))}
                    </Table.Tbody>
                  </Table>
                )}
              </ScrollArea>
            </>
          ) : (
            <Center style={{ flex: 1 }}>
              <EmptyState
                icon={<IconTable size={32} />}
                title="Select a table"
                description="Choose a table from the sidebar or create a new one"
                action={{
                  label: 'Create table',
                  onClick: openCreateTable,
                }}
              />
            </Center>
          )}
        </Box>
      </Box>

      {/* Create Table Modal */}
      <Modal opened={createTableOpened} onClose={closeCreateTable} title="Create table" size="md">
        <Stack gap="md">
          <TextInput
            label="Table name"
            placeholder="my_table"
            value={newTableName}
            onChange={(e) => setNewTableName(e.target.value)}
            required
          />
          <Text size="xs" c="dimmed">
            The table will be created with default columns: id (uuid, primary key) and created_at
            (timestamptz).
          </Text>
          <Group justify="flex-end" mt="md">
            <Button variant="outline" onClick={closeCreateTable}>
              Cancel
            </Button>
            <Button onClick={handleCreateTable} loading={formLoading}>
              Create table
            </Button>
          </Group>
        </Stack>
      </Modal>

      {/* Delete Table Confirmation */}
      <ConfirmModal
        opened={deleteTableOpened}
        onClose={closeDeleteTable}
        onConfirm={handleDeleteTable}
        title="Delete table"
        message={`Are you sure you want to delete "${selectedTable}"? All data in this table will be permanently deleted.`}
        confirmLabel="Delete"
        danger
        loading={formLoading}
      />
    </PageContainer>
  );
}
