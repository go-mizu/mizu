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
  Checkbox,
  MultiSelect,
  Tooltip,
} from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import {
  IconPlus,
  IconTrash,
  IconRefresh,
  IconList,
  IconKey,
} from '@tabler/icons-react';
import { notifications } from '@mantine/notifications';
import { PageContainer } from '../../components/layout/PageContainer';
import { EmptyState } from '../../components/common/EmptyState';
import { ConfirmModal } from '../../components/common/ConfirmModal';
import { pgmetaApi, databaseApi } from '../../api';
import type { PGIndex } from '../../api/pgmeta';
import type { Table as TableType, Column } from '../../types';

const INDEX_TYPES = [
  { value: 'btree', label: 'B-tree (default)' },
  { value: 'hash', label: 'Hash' },
  { value: 'gin', label: 'GIN (Generalized Inverted Index)' },
  { value: 'gist', label: 'GiST (Generalized Search Tree)' },
  { value: 'spgist', label: 'SP-GiST' },
  { value: 'brin', label: 'BRIN (Block Range Index)' },
];

export function IndexesPage() {
  const [schema, setSchema] = useState('public');
  const [schemas, setSchemas] = useState<string[]>([]);
  const [indexes, setIndexes] = useState<PGIndex[]>([]);
  const [tables, setTables] = useState<TableType[]>([]);
  const [loading, setLoading] = useState(true);

  const [createModalOpened, { open: openCreateModal, close: closeCreateModal }] =
    useDisclosure(false);
  const [deleteModalOpened, { open: openDeleteModal, close: closeDeleteModal }] =
    useDisclosure(false);

  const [selectedIndex, setSelectedIndex] = useState<PGIndex | null>(null);

  // Form state
  const [indexName, setIndexName] = useState('');
  const [indexTable, setIndexTable] = useState<string | null>(null);
  const [indexColumns, setIndexColumns] = useState<string[]>([]);
  const [indexType, setIndexType] = useState('btree');
  const [indexUnique, setIndexUnique] = useState(false);
  const [indexWhere, setIndexWhere] = useState('');
  const [availableColumns, setAvailableColumns] = useState<Column[]>([]);
  const [formLoading, setFormLoading] = useState(false);

  const fetchSchemas = useCallback(async () => {
    try {
      const data = await databaseApi.listSchemas();
      setSchemas(data ?? []);
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
      const [indexesData, tablesData] = await Promise.all([
        pgmetaApi.listIndexes(schema),
        databaseApi.listTables(schema),
      ]);
      setIndexes(indexesData ?? []);
      setTables(tablesData ?? []);
    } catch (error: any) {
      notifications.show({
        title: 'Error',
        message: error.message || 'Failed to load indexes',
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

  // Fetch columns when table changes
  useEffect(() => {
    if (indexTable) {
      databaseApi.listColumns(schema, indexTable).then(setAvailableColumns).catch(() => {
        setAvailableColumns([]);
      });
    } else {
      setAvailableColumns([]);
    }
  }, [schema, indexTable]);

  const handleCreateIndex = async () => {
    if (!indexName.trim() || !indexTable || indexColumns.length === 0) {
      notifications.show({
        title: 'Validation Error',
        message: 'Index name, table, and at least one column are required',
        color: 'red',
      });
      return;
    }

    setFormLoading(true);
    try {
      await pgmetaApi.createIndex({
        schema,
        table: indexTable,
        name: indexName,
        columns: indexColumns,
        unique: indexUnique,
        using: indexType,
        where: indexWhere || undefined,
      });

      notifications.show({
        title: 'Success',
        message: 'Index created successfully',
        color: 'green',
      });

      closeCreateModal();
      resetForm();
      fetchData();
    } catch (error: any) {
      notifications.show({
        title: 'Error',
        message: error.message || 'Failed to create index',
        color: 'red',
      });
    } finally {
      setFormLoading(false);
    }
  };

  const handleDeleteIndex = async () => {
    if (!selectedIndex) return;

    setFormLoading(true);
    try {
      await pgmetaApi.dropIndex(selectedIndex.schema, selectedIndex.name);

      notifications.show({
        title: 'Success',
        message: 'Index deleted successfully',
        color: 'green',
      });

      closeDeleteModal();
      setSelectedIndex(null);
      fetchData();
    } catch (error: any) {
      notifications.show({
        title: 'Error',
        message: error.message || 'Failed to delete index',
        color: 'red',
      });
    } finally {
      setFormLoading(false);
    }
  };

  const resetForm = () => {
    setIndexName('');
    setIndexTable(null);
    setIndexColumns([]);
    setIndexType('btree');
    setIndexUnique(false);
    setIndexWhere('');
  };

  const openDelete = (index: PGIndex) => {
    setSelectedIndex(index);
    openDeleteModal();
  };

  // Group indexes by table
  const indexesByTable = (indexes ?? []).reduce((acc, index) => {
    const key = index.table;
    if (!acc[key]) {
      acc[key] = [];
    }
    acc[key].push(index);
    return acc;
  }, {} as Record<string, PGIndex[]>);

  return (
    <PageContainer
      title="Indexes"
      description="Manage database indexes for query optimization"
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
            {indexes?.length ?? 0} indexes
          </Badge>
        </Group>
        <Group gap="sm">
          <ActionIcon variant="subtle" onClick={fetchData}>
            <IconRefresh size={18} />
          </ActionIcon>
          <Button leftSection={<IconPlus size={16} />} onClick={openCreateModal}>
            New Index
          </Button>
        </Group>
      </Group>

      {/* Content */}
      {loading ? (
        <Center py="xl">
          <Loader size="lg" />
        </Center>
      ) : !indexes || indexes.length === 0 ? (
        <EmptyState
          icon={<IconList size={48} />}
          title="No indexes found"
          description="Create indexes to optimize query performance"
          action={{
            label: 'Create Index',
            onClick: openCreateModal,
          }}
        />
      ) : (
        <Stack gap="md">
          {Object.entries(indexesByTable).map(([tableName, tableIndexes]) => (
            <Paper key={tableName} shadow="xs" p="md" withBorder>
              <Text fw={500} mb="md">
                {tableName}
              </Text>
              <Table striped highlightOnHover>
                <Table.Thead>
                  <Table.Tr>
                    <Table.Th>Name</Table.Th>
                    <Table.Th>Type</Table.Th>
                    <Table.Th>Columns</Table.Th>
                    <Table.Th>Unique</Table.Th>
                    <Table.Th>Size</Table.Th>
                    <Table.Th w={60}></Table.Th>
                  </Table.Tr>
                </Table.Thead>
                <Table.Tbody>
                  {tableIndexes.map((index) => (
                    <Table.Tr key={index.name}>
                      <Table.Td>
                        <Group gap="xs">
                          {index.is_primary && (
                            <Tooltip label="Primary Key">
                              <IconKey size={14} color="var(--mantine-color-yellow-6)" />
                            </Tooltip>
                          )}
                          <Text size="sm" fw={500}>
                            {index.name}
                          </Text>
                        </Group>
                      </Table.Td>
                      <Table.Td>
                        <Badge size="sm" variant="light">
                          {index.using}
                        </Badge>
                      </Table.Td>
                      <Table.Td>
                        <Group gap={4}>
                          {index.columns.map((col) => (
                            <Badge key={col} size="xs" variant="outline">
                              {col}
                            </Badge>
                          ))}
                        </Group>
                      </Table.Td>
                      <Table.Td>
                        {index.is_unique ? (
                          <Badge size="sm" color="green">
                            Yes
                          </Badge>
                        ) : (
                          <Text size="sm" c="dimmed">
                            No
                          </Text>
                        )}
                      </Table.Td>
                      <Table.Td>
                        <Text size="sm">{index.size || '—'}</Text>
                      </Table.Td>
                      <Table.Td>
                        {!index.is_primary && (
                          <ActionIcon
                            variant="subtle"
                            color="red"
                            onClick={() => openDelete(index)}
                          >
                            <IconTrash size={16} />
                          </ActionIcon>
                        )}
                      </Table.Td>
                    </Table.Tr>
                  ))}
                </Table.Tbody>
              </Table>
            </Paper>
          ))}
        </Stack>
      )}

      {/* Create Index Modal */}
      <Modal
        opened={createModalOpened}
        onClose={closeCreateModal}
        title="Create index"
        size="md"
      >
        <Stack gap="md">
          <TextInput
            label="Index name"
            placeholder="idx_table_column"
            value={indexName}
            onChange={(e) => setIndexName(e.target.value)}
            required
          />

          <Select
            label="Table"
            description="Select the table to create an index on"
            value={indexTable}
            onChange={setIndexTable}
            data={tables.map((t) => ({ value: t.name, label: t.name }))}
            placeholder="Select table"
            searchable
            required
          />

          <MultiSelect
            label="Columns"
            description="Select columns to include in the index"
            value={indexColumns}
            onChange={setIndexColumns}
            data={availableColumns.map((c) => ({ value: c.name, label: `${c.name} (${c.type})` }))}
            placeholder="Select columns"
            disabled={!indexTable}
            required
          />

          <Select
            label="Index type"
            value={indexType}
            onChange={(value) => value && setIndexType(value)}
            data={INDEX_TYPES}
          />

          <Checkbox
            label="Unique index"
            description="Enforce unique values in the indexed columns"
            checked={indexUnique}
            onChange={(e) => setIndexUnique(e.currentTarget.checked)}
          />

          <TextInput
            label="WHERE condition (optional)"
            description="Create a partial index with a condition"
            placeholder="column IS NOT NULL"
            value={indexWhere}
            onChange={(e) => setIndexWhere(e.target.value)}
          />

          <Group justify="flex-end" mt="md">
            <Button variant="outline" onClick={closeCreateModal}>
              Cancel
            </Button>
            <Button onClick={handleCreateIndex} loading={formLoading}>
              Create Index
            </Button>
          </Group>
        </Stack>
      </Modal>

      {/* Delete Confirmation */}
      <ConfirmModal
        opened={deleteModalOpened}
        onClose={closeDeleteModal}
        onConfirm={handleDeleteIndex}
        title="Delete index"
        message={`Are you sure you want to delete the index "${selectedIndex?.name}"? This may affect query performance.`}
        confirmLabel="Delete"
        danger
        loading={formLoading}
      />
    </PageContainer>
  );
}
