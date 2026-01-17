import { useEffect, useState, useCallback, useRef } from 'react';
import {
  Box,
  Button,
  Group,
  Text,
  Stack,
  Paper,
  ScrollArea,
  Select,
  ActionIcon,
  Menu,
  Badge,
  Modal,
  TextInput,
  Textarea,
  Loader,
  Center,
  Checkbox,
  Tooltip,
  Switch,
} from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import {
  IconPlus,
  IconTrash,
  IconRefresh,
  IconDotsVertical,
  IconTable,
  IconKey,
  IconEdit,
  IconColumns,
  IconRowInsertBottom,
  IconCheck,
  IconX,
  IconChevronDown,
  IconChevronUp,
} from '@tabler/icons-react';
import { notifications } from '@mantine/notifications';
import { PageContainer } from '../../components/layout/PageContainer';
import { EmptyState } from '../../components/common/EmptyState';
import { ConfirmModal } from '../../components/common/ConfirmModal';
import { databaseApi } from '../../api';
import { useAppStore } from '../../stores/appStore';
import type { Table as TableType, Column } from '../../types';

// Column type options for creating/editing columns
const COLUMN_TYPES = [
  { value: 'uuid', label: 'uuid' },
  { value: 'text', label: 'text' },
  { value: 'varchar', label: 'varchar' },
  { value: 'integer', label: 'integer' },
  { value: 'bigint', label: 'bigint' },
  { value: 'smallint', label: 'smallint' },
  { value: 'decimal', label: 'decimal' },
  { value: 'numeric', label: 'numeric' },
  { value: 'real', label: 'real' },
  { value: 'double precision', label: 'double precision' },
  { value: 'boolean', label: 'boolean' },
  { value: 'date', label: 'date' },
  { value: 'time', label: 'time' },
  { value: 'timestamp', label: 'timestamp' },
  { value: 'timestamptz', label: 'timestamptz' },
  { value: 'json', label: 'json' },
  { value: 'jsonb', label: 'jsonb' },
  { value: 'bytea', label: 'bytea' },
  { value: 'uuid[]', label: 'uuid[]' },
  { value: 'text[]', label: 'text[]' },
  { value: 'integer[]', label: 'integer[]' },
];

// Minimum column width
const MIN_COL_WIDTH = 150;
const MAX_COL_WIDTH = 400;

export function TableEditorPage() {
  const { selectedSchema, setSelectedSchema, selectedTable, setSelectedTable } = useAppStore();

  const [schemas, setSchemas] = useState<string[]>([]);
  const [tables, setTables] = useState<TableType[]>([]);
  const [columns, setColumns] = useState<Column[]>([]);
  const [tableData, setTableData] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [dataLoading, setDataLoading] = useState(false);

  // Selection state
  const [selectedRows, setSelectedRows] = useState<Set<number>>(new Set());

  // Editing state
  const [editingCell, setEditingCell] = useState<{ rowIndex: number; column: string } | null>(null);
  const [editValue, setEditValue] = useState<string>('');

  // Sort state
  const [sortColumn, setSortColumn] = useState<string | null>(null);
  const [sortDirection, setSortDirection] = useState<'asc' | 'desc'>('asc');

  // Modals
  const [createTableOpened, { open: openCreateTable, close: closeCreateTable }] =
    useDisclosure(false);
  const [deleteTableOpened, { open: openDeleteTable, close: closeDeleteTable }] =
    useDisclosure(false);
  const [insertRowOpened, { open: openInsertRow, close: closeInsertRow }] =
    useDisclosure(false);
  const [editRowOpened, { open: openEditRow, close: closeEditRow }] =
    useDisclosure(false);
  const [deleteRowOpened, { open: openDeleteRow, close: closeDeleteRow }] =
    useDisclosure(false);
  const [addColumnOpened, { open: openAddColumn, close: closeAddColumn }] =
    useDisclosure(false);

  // Form states
  const [newTableName, setNewTableName] = useState('');
  const [formLoading, setFormLoading] = useState(false);
  const [rowFormData, setRowFormData] = useState<Record<string, any>>({});
  const [selectedRowForEdit, setSelectedRowForEdit] = useState<any>(null);
  const [selectedRowIndex, setSelectedRowIndex] = useState<number | null>(null);

  // Column form state
  const [newColumnName, setNewColumnName] = useState('');
  const [newColumnType, setNewColumnType] = useState('text');
  const [newColumnDefault, setNewColumnDefault] = useState('');
  const [newColumnNullable, setNewColumnNullable] = useState(true);
  const [newColumnUnique, setNewColumnUnique] = useState(false);

  const scrollRef = useRef<HTMLDivElement>(null);

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

  const fetchTables = useCallback(async () => {
    setLoading(true);
    try {
      const data = await databaseApi.listTables(selectedSchema);
      setTables(data ?? []);
      if ((data?.length ?? 0) > 0 && !selectedTable) {
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
      // Build query with sorting
      let query = 'limit=100';
      if (sortColumn) {
        query += `&order=${sortColumn}.${sortDirection}`;
      }

      const [columnsData, rowsData] = await Promise.all([
        databaseApi.listColumns(selectedSchema, selectedTable),
        databaseApi.selectTable(selectedTable, query),
      ]);
      setColumns(columnsData ?? []);
      setTableData(rowsData ?? []);
      setSelectedRows(new Set());
      setEditingCell(null);
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
  }, [selectedSchema, selectedTable, sortColumn, sortDirection]);

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

  // Get primary key column
  const primaryKeyColumn = columns.find((c) => c.is_primary_key);

  // Build row identifier for API calls
  const getRowIdentifier = (row: any): string => {
    if (primaryKeyColumn) {
      const value = row[primaryKeyColumn.name];
      if (typeof value === 'string') {
        return `${primaryKeyColumn.name}=eq.${value}`;
      }
      return `${primaryKeyColumn.name}=eq.${value}`;
    }
    // Fallback: use all columns
    return columns
      .map((col) => `${col.name}=eq.${row[col.name]}`)
      .join('&');
  };

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

  // Row operations
  const handleInsertRow = async () => {
    setFormLoading(true);
    try {
      // Filter out empty values and build insert data
      const insertData: Record<string, any> = {};
      for (const col of columns) {
        const value = rowFormData[col.name];
        if (value !== undefined && value !== '') {
          // Convert types as needed
          if (col.type === 'integer' || col.type === 'bigint' || col.type === 'smallint') {
            insertData[col.name] = parseInt(value, 10);
          } else if (col.type === 'numeric' || col.type === 'decimal' || col.type === 'real' || col.type === 'double precision') {
            insertData[col.name] = parseFloat(value);
          } else if (col.type === 'boolean') {
            insertData[col.name] = value === 'true' || value === true;
          } else if (col.type === 'json' || col.type === 'jsonb') {
            try {
              insertData[col.name] = JSON.parse(value);
            } catch {
              insertData[col.name] = value;
            }
          } else {
            insertData[col.name] = value;
          }
        }
      }

      await databaseApi.insertRow(selectedTable!, insertData);
      notifications.show({
        title: 'Success',
        message: 'Row inserted successfully',
        color: 'green',
      });
      closeInsertRow();
      setRowFormData({});
      fetchTableData();
    } catch (error: any) {
      notifications.show({
        title: 'Error',
        message: error.message || 'Failed to insert row',
        color: 'red',
      });
    } finally {
      setFormLoading(false);
    }
  };

  const handleUpdateRow = async () => {
    if (!selectedRowForEdit) return;

    setFormLoading(true);
    try {
      const identifier = getRowIdentifier(selectedRowForEdit);
      const updateData: Record<string, any> = {};

      for (const col of columns) {
        const value = rowFormData[col.name];
        if (value !== undefined) {
          // Convert types as needed
          if (value === '' || value === null) {
            updateData[col.name] = null;
          } else if (col.type === 'integer' || col.type === 'bigint' || col.type === 'smallint') {
            updateData[col.name] = parseInt(value, 10);
          } else if (col.type === 'numeric' || col.type === 'decimal' || col.type === 'real' || col.type === 'double precision') {
            updateData[col.name] = parseFloat(value);
          } else if (col.type === 'boolean') {
            updateData[col.name] = value === 'true' || value === true;
          } else if (col.type === 'json' || col.type === 'jsonb') {
            try {
              updateData[col.name] = JSON.parse(value);
            } catch {
              updateData[col.name] = value;
            }
          } else {
            updateData[col.name] = value;
          }
        }
      }

      await databaseApi.updateRow(selectedTable!, identifier, updateData);
      notifications.show({
        title: 'Success',
        message: 'Row updated successfully',
        color: 'green',
      });
      closeEditRow();
      setRowFormData({});
      setSelectedRowForEdit(null);
      fetchTableData();
    } catch (error: any) {
      notifications.show({
        title: 'Error',
        message: error.message || 'Failed to update row',
        color: 'red',
      });
    } finally {
      setFormLoading(false);
    }
  };

  const handleDeleteRow = async () => {
    if (selectedRowIndex === null) return;

    setFormLoading(true);
    try {
      const row = tableData[selectedRowIndex];
      const identifier = getRowIdentifier(row);
      await databaseApi.deleteRow(selectedTable!, identifier);
      notifications.show({
        title: 'Success',
        message: 'Row deleted successfully',
        color: 'green',
      });
      closeDeleteRow();
      setSelectedRowIndex(null);
      fetchTableData();
    } catch (error: any) {
      notifications.show({
        title: 'Error',
        message: error.message || 'Failed to delete row',
        color: 'red',
      });
    } finally {
      setFormLoading(false);
    }
  };

  const handleDeleteSelectedRows = async () => {
    if (selectedRows.size === 0) return;

    setFormLoading(true);
    try {
      const deletePromises = Array.from(selectedRows).map((rowIndex) => {
        const row = tableData[rowIndex];
        const identifier = getRowIdentifier(row);
        return databaseApi.deleteRow(selectedTable!, identifier);
      });

      await Promise.all(deletePromises);
      notifications.show({
        title: 'Success',
        message: `${selectedRows.size} row(s) deleted successfully`,
        color: 'green',
      });
      setSelectedRows(new Set());
      fetchTableData();
    } catch (error: any) {
      notifications.show({
        title: 'Error',
        message: error.message || 'Failed to delete rows',
        color: 'red',
      });
    } finally {
      setFormLoading(false);
    }
  };

  // Column operations
  const handleAddColumn = async () => {
    if (!newColumnName.trim()) {
      notifications.show({
        title: 'Validation Error',
        message: 'Column name is required',
        color: 'red',
      });
      return;
    }

    setFormLoading(true);
    try {
      await databaseApi.addColumn(selectedSchema, selectedTable!, {
        name: newColumnName,
        type: newColumnType,
        default_value: newColumnDefault || undefined,
        is_nullable: newColumnNullable,
        is_unique: newColumnUnique,
      });
      notifications.show({
        title: 'Success',
        message: 'Column added successfully',
        color: 'green',
      });
      closeAddColumn();
      resetColumnForm();
      fetchTableData();
    } catch (error: any) {
      notifications.show({
        title: 'Error',
        message: error.message || 'Failed to add column',
        color: 'red',
      });
    } finally {
      setFormLoading(false);
    }
  };

  const resetColumnForm = () => {
    setNewColumnName('');
    setNewColumnType('text');
    setNewColumnDefault('');
    setNewColumnNullable(true);
    setNewColumnUnique(false);
  };

  // Inline cell editing
  const startEditingCell = (rowIndex: number, column: string, currentValue: any) => {
    setEditingCell({ rowIndex, column });
    setEditValue(currentValue === null ? '' : String(currentValue));
  };

  const saveEditingCell = async () => {
    if (!editingCell) return;

    const row = tableData[editingCell.rowIndex];
    const col = columns.find((c) => c.name === editingCell.column);
    if (!col) return;

    try {
      const identifier = getRowIdentifier(row);
      let value: any = editValue;

      // Convert types
      if (editValue === '' && col.is_nullable) {
        value = null;
      } else if (col.type === 'integer' || col.type === 'bigint' || col.type === 'smallint') {
        value = parseInt(editValue, 10);
      } else if (col.type === 'numeric' || col.type === 'decimal' || col.type === 'real' || col.type === 'double precision') {
        value = parseFloat(editValue);
      } else if (col.type === 'boolean') {
        value = editValue === 'true' || editValue === '1';
      }

      await databaseApi.updateRow(selectedTable!, identifier, { [editingCell.column]: value });

      // Update local data
      const newData = [...tableData];
      newData[editingCell.rowIndex] = { ...row, [editingCell.column]: value };
      setTableData(newData);

      setEditingCell(null);
    } catch (error: any) {
      notifications.show({
        title: 'Error',
        message: error.message || 'Failed to update cell',
        color: 'red',
      });
    }
  };

  const cancelEditingCell = () => {
    setEditingCell(null);
    setEditValue('');
  };

  // Row selection
  const toggleRowSelection = (rowIndex: number) => {
    const newSelection = new Set(selectedRows);
    if (newSelection.has(rowIndex)) {
      newSelection.delete(rowIndex);
    } else {
      newSelection.add(rowIndex);
    }
    setSelectedRows(newSelection);
  };

  const toggleAllRows = () => {
    if (selectedRows.size === tableData.length) {
      setSelectedRows(new Set());
    } else {
      setSelectedRows(new Set(tableData.map((_, i) => i)));
    }
  };

  // Open edit row modal
  const openEditRowModal = (row: any) => {
    setSelectedRowForEdit(row);
    const formData: Record<string, any> = {};
    columns.forEach((col) => {
      formData[col.name] = row[col.name] === null ? '' : row[col.name];
    });
    setRowFormData(formData);
    openEditRow();
  };

  // Open delete row confirmation
  const confirmDeleteRow = (rowIndex: number) => {
    setSelectedRowIndex(rowIndex);
    openDeleteRow();
  };

  // Sorting
  const handleSort = (column: string) => {
    if (sortColumn === column) {
      setSortDirection(sortDirection === 'asc' ? 'desc' : 'asc');
    } else {
      setSortColumn(column);
      setSortDirection('asc');
    }
  };

  const currentTable = tables.find((t) => t.name === selectedTable);

  // Calculate column width based on type
  const getColumnWidth = (col: Column): number => {
    if (col.type.includes('text') || col.type.includes('varchar')) return 200;
    if (col.type.includes('json')) return 250;
    if (col.type.includes('timestamp') || col.type.includes('date')) return 200;
    if (col.type.includes('uuid')) return 280;
    return MIN_COL_WIDTH;
  };

  // Format cell value for display
  const formatCellValue = (value: any): string => {
    if (value === null) return 'NULL';
    if (typeof value === 'boolean') return value ? 'true' : 'false';
    if (typeof value === 'object') return JSON.stringify(value);
    return String(value);
  };

  return (
    <PageContainer title="Table Editor" description="View and edit your database tables" fullWidth noPadding>
      <Box style={{ display: 'flex', height: 'calc(100vh - 140px)' }}>
        {/* Table Sidebar */}
        <Box
          style={{
            width: 260,
            minWidth: 260,
            borderRight: '1px solid var(--supabase-border)',
            display: 'flex',
            flexDirection: 'column',
            backgroundColor: 'var(--supabase-bg-surface)',
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
              data={(schemas ?? []).map((s) => ({ value: s, label: s }))}
              placeholder="Select schema"
            />
          </Box>

          <ScrollArea style={{ flex: 1 }} px="sm" pb="sm">
            {loading ? (
              <Center py="xl">
                <Loader size="sm" />
              </Center>
            ) : !tables || tables.length === 0 ? (
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
                        <Text size="sm" truncate style={{ maxWidth: 140 }}>
                          {table.name}
                        </Text>
                      </Group>
                      <Badge size="xs" variant="light" color="gray">
                        {table.row_count ?? 0}
                      </Badge>
                    </Group>
                  </Paper>
                ))}
              </Stack>
            )}
          </ScrollArea>
        </Box>

        {/* Data Grid */}
        <Box style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0 }}>
          {selectedTable ? (
            <>
              {/* Toolbar */}
              <Box
                p="sm"
                style={{ borderBottom: '1px solid var(--supabase-border)', backgroundColor: 'var(--supabase-bg-surface)' }}
              >
                <Group justify="space-between">
                  <Group gap="sm">
                    <Text fw={500}>{selectedTable}</Text>
                    {currentTable?.rls_enabled && (
                      <Badge size="sm" variant="light" color="green">
                        RLS
                      </Badge>
                    )}
                  </Group>
                  <Group gap="xs">
                    {/* Insert Row Button */}
                    <Button
                      size="xs"
                      leftSection={<IconRowInsertBottom size={14} />}
                      onClick={() => {
                        setRowFormData({});
                        openInsertRow();
                      }}
                    >
                      Insert row
                    </Button>

                    {/* Add Column */}
                    <Tooltip label="Add column">
                      <ActionIcon variant="subtle" onClick={openAddColumn}>
                        <IconColumns size={18} />
                      </ActionIcon>
                    </Tooltip>

                    {/* Refresh */}
                    <Tooltip label="Refresh">
                      <ActionIcon variant="subtle" onClick={fetchTableData}>
                        <IconRefresh size={18} />
                      </ActionIcon>
                    </Tooltip>

                    {/* Delete selected rows */}
                    {selectedRows.size > 0 && (
                      <Button
                        size="xs"
                        color="red"
                        variant="light"
                        leftSection={<IconTrash size={14} />}
                        onClick={handleDeleteSelectedRows}
                        loading={formLoading}
                      >
                        Delete ({selectedRows.size})
                      </Button>
                    )}

                    {/* More actions */}
                    <Menu position="bottom-end">
                      <Menu.Target>
                        <ActionIcon variant="subtle">
                          <IconDotsVertical size={18} />
                        </ActionIcon>
                      </Menu.Target>
                      <Menu.Dropdown>
                        <Menu.Item
                          leftSection={<IconColumns size={14} />}
                          onClick={openAddColumn}
                        >
                          Add column
                        </Menu.Item>
                        <Menu.Divider />
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

              {/* Column/Row count info */}
              {columns && columns.length > 0 && (
                <Box
                  px="sm"
                  py="xs"
                  style={{
                    borderBottom: '1px solid var(--supabase-border)',
                    backgroundColor: 'var(--supabase-bg)',
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
                      {tableData?.length ?? 0} rows (max 100)
                    </Text>
                  </Group>
                </Box>
              )}

              {/* Data Table with horizontal scroll */}
              <Box style={{ flex: 1, overflow: 'hidden' }}>
                {dataLoading ? (
                  <Center py="xl">
                    <Loader size="sm" />
                  </Center>
                ) : !columns || columns.length === 0 ? (
                  <Center py="xl">
                    <EmptyState
                      icon={<IconColumns size={32} />}
                      title="No columns"
                      description="Add columns to this table to start storing data"
                      action={{ label: 'Add column', onClick: openAddColumn }}
                    />
                  </Center>
                ) : !tableData || tableData.length === 0 ? (
                  <Center py="xl">
                    <EmptyState
                      icon={<IconRowInsertBottom size={32} />}
                      title="No rows"
                      description="Insert rows to start adding data"
                      action={{ label: 'Insert row', onClick: () => { setRowFormData({}); openInsertRow(); } }}
                    />
                  </Center>
                ) : (
                  <ScrollArea
                    ref={scrollRef}
                    style={{ height: '100%' }}
                    scrollbarSize={8}
                    type="always"
                  >
                    <Box style={{ minWidth: 'max-content' }}>
                      {/* Table */}
                      <table
                        style={{
                          width: '100%',
                          borderCollapse: 'collapse',
                          fontSize: 13,
                        }}
                      >
                        <thead>
                          <tr style={{ backgroundColor: 'var(--supabase-bg-surface)' }}>
                            {/* Selection checkbox */}
                            <th
                              style={{
                                width: 40,
                                minWidth: 40,
                                padding: '8px 12px',
                                borderBottom: '1px solid var(--supabase-border)',
                                position: 'sticky',
                                left: 0,
                                backgroundColor: 'var(--supabase-bg-surface)',
                                zIndex: 2,
                              }}
                            >
                              <Checkbox
                                size="xs"
                                checked={selectedRows.size === tableData.length && tableData.length > 0}
                                indeterminate={selectedRows.size > 0 && selectedRows.size < tableData.length}
                                onChange={toggleAllRows}
                              />
                            </th>
                            {/* Row actions column */}
                            <th
                              style={{
                                width: 60,
                                minWidth: 60,
                                padding: '8px 4px',
                                borderBottom: '1px solid var(--supabase-border)',
                                position: 'sticky',
                                left: 40,
                                backgroundColor: 'var(--supabase-bg-surface)',
                                zIndex: 2,
                              }}
                            />
                            {/* Data columns */}
                            {columns.map((col) => (
                              <th
                                key={col.name}
                                style={{
                                  minWidth: getColumnWidth(col),
                                  maxWidth: MAX_COL_WIDTH,
                                  padding: '8px 12px',
                                  borderBottom: '1px solid var(--supabase-border)',
                                  textAlign: 'left',
                                  cursor: 'pointer',
                                  userSelect: 'none',
                                }}
                                onClick={() => handleSort(col.name)}
                              >
                                <Group gap={4} wrap="nowrap" justify="space-between">
                                  <Group gap={4} wrap="nowrap">
                                    {col.is_primary_key && (
                                      <IconKey size={12} color="var(--supabase-brand)" />
                                    )}
                                    <Text size="xs" fw={600}>
                                      {col.name}
                                    </Text>
                                    <Text size="xs" c="dimmed" fw={400}>
                                      {col.type}
                                    </Text>
                                  </Group>
                                  {sortColumn === col.name && (
                                    sortDirection === 'asc' ? <IconChevronUp size={14} /> : <IconChevronDown size={14} />
                                  )}
                                </Group>
                              </th>
                            ))}
                          </tr>
                        </thead>
                        <tbody>
                          {tableData.map((row, rowIndex) => (
                            <tr
                              key={rowIndex}
                              style={{
                                backgroundColor: selectedRows.has(rowIndex)
                                  ? 'var(--supabase-brand-light)'
                                  : rowIndex % 2 === 0
                                    ? 'var(--supabase-bg)'
                                    : 'var(--supabase-bg-surface)',
                              }}
                            >
                              {/* Selection checkbox */}
                              <td
                                style={{
                                  padding: '8px 12px',
                                  borderBottom: '1px solid var(--supabase-border)',
                                  position: 'sticky',
                                  left: 0,
                                  backgroundColor: 'inherit',
                                  zIndex: 1,
                                }}
                              >
                                <Checkbox
                                  size="xs"
                                  checked={selectedRows.has(rowIndex)}
                                  onChange={() => toggleRowSelection(rowIndex)}
                                />
                              </td>
                              {/* Row actions */}
                              <td
                                style={{
                                  padding: '4px',
                                  borderBottom: '1px solid var(--supabase-border)',
                                  position: 'sticky',
                                  left: 40,
                                  backgroundColor: 'inherit',
                                  zIndex: 1,
                                }}
                              >
                                <Group gap={2} wrap="nowrap">
                                  <Tooltip label="Edit row">
                                    <ActionIcon
                                      size="xs"
                                      variant="subtle"
                                      onClick={() => openEditRowModal(row)}
                                    >
                                      <IconEdit size={14} />
                                    </ActionIcon>
                                  </Tooltip>
                                  <Tooltip label="Delete row">
                                    <ActionIcon
                                      size="xs"
                                      variant="subtle"
                                      color="red"
                                      onClick={() => confirmDeleteRow(rowIndex)}
                                    >
                                      <IconTrash size={14} />
                                    </ActionIcon>
                                  </Tooltip>
                                </Group>
                              </td>
                              {/* Data cells */}
                              {columns.map((col) => (
                                <td
                                  key={col.name}
                                  style={{
                                    padding: '8px 12px',
                                    borderBottom: '1px solid var(--supabase-border)',
                                    maxWidth: MAX_COL_WIDTH,
                                    overflow: 'hidden',
                                    textOverflow: 'ellipsis',
                                    whiteSpace: 'nowrap',
                                    cursor: 'pointer',
                                  }}
                                  onDoubleClick={() => startEditingCell(rowIndex, col.name, row[col.name])}
                                >
                                  {editingCell?.rowIndex === rowIndex && editingCell?.column === col.name ? (
                                    <Group gap={4} wrap="nowrap">
                                      <TextInput
                                        size="xs"
                                        value={editValue}
                                        onChange={(e) => setEditValue(e.target.value)}
                                        onKeyDown={(e) => {
                                          if (e.key === 'Enter') saveEditingCell();
                                          if (e.key === 'Escape') cancelEditingCell();
                                        }}
                                        autoFocus
                                        style={{ flex: 1 }}
                                      />
                                      <ActionIcon size="xs" color="green" onClick={saveEditingCell}>
                                        <IconCheck size={12} />
                                      </ActionIcon>
                                      <ActionIcon size="xs" color="red" onClick={cancelEditingCell}>
                                        <IconX size={12} />
                                      </ActionIcon>
                                    </Group>
                                  ) : (
                                    <Text
                                      size="sm"
                                      c={row[col.name] === null ? 'dimmed' : undefined}
                                      style={{
                                        fontStyle: row[col.name] === null ? 'italic' : 'normal',
                                      }}
                                    >
                                      {formatCellValue(row[col.name])}
                                    </Text>
                                  )}
                                </td>
                              ))}
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </Box>
                  </ScrollArea>
                )}
              </Box>
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

      {/* Insert Row Modal */}
      <Modal
        opened={insertRowOpened}
        onClose={closeInsertRow}
        title="Insert row"
        size="lg"
      >
        <ScrollArea style={{ maxHeight: 'calc(100vh - 300px)' }}>
          <Stack gap="sm">
            {columns.map((col) => (
              <Box key={col.name}>
                <Group gap="xs" mb={4}>
                  <Text size="sm" fw={500}>
                    {col.name}
                  </Text>
                  <Text size="xs" c="dimmed">
                    {col.type}
                  </Text>
                  {col.is_primary_key && (
                    <Badge size="xs" color="blue">
                      Primary Key
                    </Badge>
                  )}
                  {col.default_value && (
                    <Badge size="xs" variant="light">
                      Default: {col.default_value}
                    </Badge>
                  )}
                </Group>
                {col.type === 'boolean' ? (
                  <Select
                    size="sm"
                    value={rowFormData[col.name] ?? ''}
                    onChange={(value) =>
                      setRowFormData({ ...rowFormData, [col.name]: value })
                    }
                    data={[
                      { value: '', label: '(null)' },
                      { value: 'true', label: 'true' },
                      { value: 'false', label: 'false' },
                    ]}
                    placeholder={col.default_value ? `Default: ${col.default_value}` : 'Select value'}
                  />
                ) : col.type === 'json' || col.type === 'jsonb' ? (
                  <Textarea
                    size="sm"
                    value={rowFormData[col.name] ?? ''}
                    onChange={(e) =>
                      setRowFormData({ ...rowFormData, [col.name]: e.target.value })
                    }
                    placeholder={col.default_value ? `Default: ${col.default_value}` : '{}'}
                    minRows={2}
                    styles={{ input: { fontFamily: 'monospace' } }}
                  />
                ) : (
                  <TextInput
                    size="sm"
                    value={rowFormData[col.name] ?? ''}
                    onChange={(e) =>
                      setRowFormData({ ...rowFormData, [col.name]: e.target.value })
                    }
                    placeholder={col.default_value ? `Default: ${col.default_value}` : `Enter ${col.type}`}
                  />
                )}
              </Box>
            ))}
          </Stack>
        </ScrollArea>
        <Group justify="flex-end" mt="md">
          <Button variant="outline" onClick={closeInsertRow}>
            Cancel
          </Button>
          <Button onClick={handleInsertRow} loading={formLoading}>
            Insert
          </Button>
        </Group>
      </Modal>

      {/* Edit Row Modal */}
      <Modal
        opened={editRowOpened}
        onClose={closeEditRow}
        title="Edit row"
        size="lg"
      >
        <ScrollArea style={{ maxHeight: 'calc(100vh - 300px)' }}>
          <Stack gap="sm">
            {columns.map((col) => (
              <Box key={col.name}>
                <Group gap="xs" mb={4}>
                  <Text size="sm" fw={500}>
                    {col.name}
                  </Text>
                  <Text size="xs" c="dimmed">
                    {col.type}
                  </Text>
                  {col.is_primary_key && (
                    <Badge size="xs" color="blue">
                      Primary Key
                    </Badge>
                  )}
                </Group>
                {col.type === 'boolean' ? (
                  <Select
                    size="sm"
                    value={String(rowFormData[col.name] ?? '')}
                    onChange={(value) =>
                      setRowFormData({ ...rowFormData, [col.name]: value })
                    }
                    data={[
                      { value: '', label: '(null)' },
                      { value: 'true', label: 'true' },
                      { value: 'false', label: 'false' },
                    ]}
                  />
                ) : col.type === 'json' || col.type === 'jsonb' ? (
                  <Textarea
                    size="sm"
                    value={
                      typeof rowFormData[col.name] === 'object'
                        ? JSON.stringify(rowFormData[col.name], null, 2)
                        : rowFormData[col.name] ?? ''
                    }
                    onChange={(e) =>
                      setRowFormData({ ...rowFormData, [col.name]: e.target.value })
                    }
                    minRows={2}
                    styles={{ input: { fontFamily: 'monospace' } }}
                  />
                ) : (
                  <TextInput
                    size="sm"
                    value={String(rowFormData[col.name] ?? '')}
                    onChange={(e) =>
                      setRowFormData({ ...rowFormData, [col.name]: e.target.value })
                    }
                    disabled={col.is_primary_key}
                  />
                )}
              </Box>
            ))}
          </Stack>
        </ScrollArea>
        <Group justify="flex-end" mt="md">
          <Button variant="outline" onClick={closeEditRow}>
            Cancel
          </Button>
          <Button onClick={handleUpdateRow} loading={formLoading}>
            Save
          </Button>
        </Group>
      </Modal>

      {/* Delete Row Confirmation */}
      <ConfirmModal
        opened={deleteRowOpened}
        onClose={closeDeleteRow}
        onConfirm={handleDeleteRow}
        title="Delete row"
        message="Are you sure you want to delete this row? This action cannot be undone."
        confirmLabel="Delete"
        danger
        loading={formLoading}
      />

      {/* Add Column Modal */}
      <Modal
        opened={addColumnOpened}
        onClose={() => { closeAddColumn(); resetColumnForm(); }}
        title="Add column"
        size="md"
      >
        <Stack gap="md">
          <TextInput
            label="Column name"
            placeholder="my_column"
            value={newColumnName}
            onChange={(e) => setNewColumnName(e.target.value)}
            required
          />

          <Select
            label="Type"
            value={newColumnType}
            onChange={(value) => value && setNewColumnType(value)}
            data={COLUMN_TYPES}
            searchable
          />

          <TextInput
            label="Default value"
            placeholder="Optional"
            value={newColumnDefault}
            onChange={(e) => setNewColumnDefault(e.target.value)}
          />

          <Group>
            <Switch
              label="Allow NULL"
              checked={newColumnNullable}
              onChange={(e) => setNewColumnNullable(e.currentTarget.checked)}
            />
            <Switch
              label="Unique"
              checked={newColumnUnique}
              onChange={(e) => setNewColumnUnique(e.currentTarget.checked)}
            />
          </Group>

          <Group justify="flex-end" mt="md">
            <Button variant="outline" onClick={() => { closeAddColumn(); resetColumnForm(); }}>
              Cancel
            </Button>
            <Button onClick={handleAddColumn} loading={formLoading}>
              Add column
            </Button>
          </Group>
        </Stack>
      </Modal>
    </PageContainer>
  );
}
