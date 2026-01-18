import { useEffect, useState, useCallback, useRef } from 'react';
import {
  Box,
  Button,
  Group,
  Text,
  Stack,
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
  Drawer,
  SegmentedControl,
  Collapse,
} from '@mantine/core';
import { useDisclosure, useMediaQuery, useHotkeys } from '@mantine/hooks';
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
  IconFilter,
  IconSortAscending,
  IconChevronLeft,
  IconChevronRight,
  IconShield,
  IconBolt,
  IconBroadcast,
  IconSearch,
  IconLayoutSidebar,
  IconDownload,
  IconMaximize,
} from '@tabler/icons-react';
import { notifications } from '@mantine/notifications';
import { PageContainer } from '../../components/layout/PageContainer';
import { EmptyState } from '../../components/common/EmptyState';
import { ConfirmModal } from '../../components/common/ConfirmModal';
import {
  TableTabs,
  type TableTab,
  FilterBuilder,
  type Filter,
  RowDetailDrawer,
  ExportModal,
  ColumnManager,
} from '../../components/table-editor';
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

  // New feature modals
  const [exportOpened, { open: openExport, close: closeExport }] = useDisclosure(false);
  const [rowDetailOpened, { open: openRowDetail, close: closeRowDetail }] = useDisclosure(false);
  const [filterOpened, { open: openFilter, close: closeFilter }] = useDisclosure(false);

  // Table tabs state
  const [openTabs, setOpenTabs] = useState<TableTab[]>([]);
  const [activeTabId, setActiveTabId] = useState<string | null>(null);

  // Filter state
  const [filters, setFilters] = useState<Filter[]>([]);
  const [filterLogic, setFilterLogic] = useState<'AND' | 'OR'>('AND');

  // Column visibility state
  const [visibleColumns, setVisibleColumns] = useState<Set<string>>(new Set());
  const [columnOrder, setColumnOrder] = useState<string[]>([]);

  // Row for detail view
  const [detailRow, setDetailRow] = useState<Record<string, any> | null>(null);

  // Total row count for pagination
  const [totalRowCount, setTotalRowCount] = useState<number>(0);

  // Keyboard navigation state
  const [focusedCell, setFocusedCell] = useState<{ row: number; col: number } | null>(null);

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

  // Tab management functions
  const addTab = useCallback((schema: string, table: string, isTransient = true) => {
    const existingTab = openTabs.find((t) => t.schema === schema && t.table === table);
    if (existingTab) {
      setActiveTabId(existingTab.id);
      return;
    }

    // If there's a transient tab, replace it
    const transientIndex = openTabs.findIndex((t) => t.isTransient);
    const newTab: TableTab = {
      id: crypto.randomUUID(),
      schema,
      table,
      isTransient,
    };

    if (transientIndex !== -1 && isTransient) {
      const newTabs = [...openTabs];
      newTabs[transientIndex] = newTab;
      setOpenTabs(newTabs);
    } else {
      setOpenTabs([...openTabs, newTab]);
    }
    setActiveTabId(newTab.id);
  }, [openTabs]);

  const closeTab = useCallback((tabId: string) => {
    const tabIndex = openTabs.findIndex((t) => t.id === tabId);
    const newTabs = openTabs.filter((t) => t.id !== tabId);
    setOpenTabs(newTabs);

    if (activeTabId === tabId) {
      // Activate the previous tab or next tab
      if (newTabs.length > 0) {
        const newIndex = Math.min(tabIndex, newTabs.length - 1);
        setActiveTabId(newTabs[newIndex].id);
        setSelectedTable(newTabs[newIndex].table);
        if (newTabs[newIndex].schema !== selectedSchema) {
          setSelectedSchema(newTabs[newIndex].schema);
        }
      } else {
        setActiveTabId(null);
        setSelectedTable(null);
      }
    }
  }, [openTabs, activeTabId, selectedSchema, setSelectedSchema, setSelectedTable]);

  const makeTabPermanent = useCallback((tabId: string) => {
    setOpenTabs(openTabs.map((t) => (t.id === tabId ? { ...t, isTransient: false } : t)));
  }, [openTabs]);

  // Update columns visibility when columns change
  useEffect(() => {
    if (columns.length > 0 && visibleColumns.size === 0) {
      setVisibleColumns(new Set(columns.map((c) => c.name)));
      setColumnOrder(columns.map((c) => c.name));
    }
  }, [columns, visibleColumns.size]);

  // Apply filters to fetch
  const buildFilterQuery = useCallback(() => {
    if (filters.length === 0) return {};
    const query: Record<string, string> = {};
    filters.forEach((f) => {
      if (f.column && f.value !== '') {
        query[f.column] = `${f.operator}.${f.value}`;
      }
    });
    return query;
  }, [filters]);

  // Keyboard navigation
  useHotkeys([
    ['ArrowUp', () => {
      if (focusedCell && focusedCell.row > 0) {
        setFocusedCell({ row: focusedCell.row - 1, col: focusedCell.col });
      }
    }],
    ['ArrowDown', () => {
      if (focusedCell && focusedCell.row < tableData.length - 1) {
        setFocusedCell({ row: focusedCell.row + 1, col: focusedCell.col });
      }
    }],
    ['ArrowLeft', () => {
      if (focusedCell && focusedCell.col > 0) {
        setFocusedCell({ row: focusedCell.row, col: focusedCell.col - 1 });
      }
    }],
    ['ArrowRight', () => {
      if (focusedCell && focusedCell.col < columns.length - 1) {
        setFocusedCell({ row: focusedCell.row, col: focusedCell.col + 1 });
      }
    }],
    ['Enter', () => {
      if (focusedCell && !editingCell) {
        const col = columns[focusedCell.col];
        const row = tableData[focusedCell.row];
        if (col && row) {
          startEditingCell(focusedCell.row, col.name, row[col.name]);
        }
      }
    }],
    ['Escape', () => {
      if (editingCell) {
        cancelEditingCell();
      }
      setFocusedCell(null);
    }],
    ['mod+r', () => fetchTableData()],
    ['mod+n', () => { setRowFormData({}); openInsertRow(); }],
    ['mod+f', () => { filterOpened ? closeFilter() : openFilter(); }],
    ['mod+shift+e', () => openExport()],
  ]);

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
      // Build options with sorting and filters
      const options: Parameters<typeof databaseApi.getTableData>[2] = {
        limit: 100,
        includeCount: true,
      };
      if (sortColumn) {
        options.order = `${sortColumn}.${sortDirection}`;
      }
      // Apply active filters
      const filterQuery = buildFilterQuery();
      if (Object.keys(filterQuery).length > 0) {
        options.filters = filterQuery;
      }

      const [columnsData, tableResult] = await Promise.all([
        databaseApi.listColumns(selectedSchema, selectedTable),
        databaseApi.getTableData(selectedSchema, selectedTable, options),
      ]);
      setColumns(columnsData ?? []);
      setTableData(tableResult.data ?? []);
      if (tableResult.totalCount !== undefined) {
        setTotalRowCount(tableResult.totalCount);
      }
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
  }, [selectedSchema, selectedTable, sortColumn, sortDirection, buildFilterQuery]);

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
    if (col.type.includes('uuid')) return 300;
    return MIN_COL_WIDTH;
  };

  // Format UUID from byte array (fallback for unformatted UUIDs)
  const formatUUIDFromBytes = (bytes: number[]): string => {
    const hex = bytes.map(b => b.toString(16).padStart(2, '0')).join('');
    return `${hex.slice(0, 8)}-${hex.slice(8, 12)}-${hex.slice(12, 16)}-${hex.slice(16, 20)}-${hex.slice(20)}`;
  };

  // Check if value looks like a UUID byte array
  const isUUIDByteArray = (value: unknown): value is number[] => {
    return Array.isArray(value) &&
      value.length === 16 &&
      value.every(v => typeof v === 'number' && v >= 0 && v <= 255);
  };

  // Format cell value for display with type awareness
  const formatCellValue = (value: any, columnType?: string): string => {
    if (value === null) return 'NULL';
    if (typeof value === 'boolean') return value ? 'true' : 'false';

    // Handle UUID byte arrays (fallback in case backend doesn't convert)
    if (isUUIDByteArray(value)) {
      return formatUUIDFromBytes(value);
    }

    // Format timestamps nicely
    if (columnType?.includes('timestamp') && typeof value === 'string') {
      try {
        const date = new Date(value);
        if (!isNaN(date.getTime())) {
          return date.toLocaleString('en-US', {
            year: 'numeric',
            month: 'short',
            day: 'numeric',
            hour: '2-digit',
            minute: '2-digit',
            second: '2-digit',
          });
        }
      } catch {
        // Fall through to default
      }
    }

    // Format dates nicely
    if (columnType?.includes('date') && !columnType?.includes('timestamp') && typeof value === 'string') {
      try {
        const date = new Date(value);
        if (!isNaN(date.getTime())) {
          return date.toLocaleDateString('en-US', {
            year: 'numeric',
            month: 'short',
            day: 'numeric',
          });
        }
      } catch {
        // Fall through to default
      }
    }

    // Format JSON with indentation for readability
    if (typeof value === 'object') {
      try {
        return JSON.stringify(value, null, 2);
      } catch {
        return String(value);
      }
    }

    return String(value);
  };

  // Get display style for cell based on column type
  const getCellStyle = (columnType?: string): React.CSSProperties => {
    if (columnType?.includes('json')) {
      return { fontFamily: 'monospace', fontSize: '0.75rem', whiteSpace: 'pre-wrap' };
    }
    if (columnType?.includes('uuid')) {
      return { fontFamily: 'monospace', fontSize: '0.75rem' };
    }
    if (columnType?.includes('numeric') || columnType?.includes('integer') || columnType?.includes('decimal') || columnType?.includes('bigint')) {
      return { fontFamily: 'monospace', textAlign: 'right' };
    }
    return {};
  };

  // Pagination state
  const [currentPage, setCurrentPage] = useState(1);
  const [rowsPerPage, setRowsPerPage] = useState(100);
  const totalPages = Math.ceil((tableData?.length ?? 0) / rowsPerPage);

  // Search state for table list
  const [tableSearchQuery, setTableSearchQuery] = useState('');

  // Tab state (Data vs Definition)
  const [activeTab, setActiveTab] = useState<'data' | 'definition'>('data');

  // Responsive state
  const isMobile = useMediaQuery('(max-width: 768px)');
  const isTablet = useMediaQuery('(max-width: 1024px)');
  const [tableSidebarOpened, { open: openTableSidebar, close: closeTableSidebar }] = useDisclosure(false);

  // Filter tables by search query
  const filteredTables = tables.filter((table) =>
    tableSearchQuery === '' || table.name.toLowerCase().includes(tableSearchQuery.toLowerCase())
  );

  // Sidebar content component for reuse
  const SidebarContent = ({ onTableSelect }: { onTableSelect?: () => void }) => (
    <>
      {/* Sidebar Header */}
      <Box
        p="sm"
        style={{
          borderBottom: '1px solid var(--supabase-border)',
        }}
      >
        <Text
          size="xs"
          fw={600}
          tt="uppercase"
          mb="xs"
          style={{
            color: 'var(--supabase-text-muted)',
            letterSpacing: '0.05em',
            fontSize: '0.6875rem',
          }}
        >
          Table Editor
        </Text>
        <Select
          size="xs"
          value={selectedSchema}
          onChange={(value) => {
            if (value) {
              setSelectedSchema(value);
              setSelectedTable(null);
            }
          }}
          data={(schemas ?? []).map((s) => ({ value: s, label: `schema ${s}` }))}
          placeholder="Select schema"
          leftSection={<Text size="xs" c="dimmed">schema</Text>}
          styles={{
            input: {
              backgroundColor: 'var(--supabase-bg-surface)',
            },
          }}
        />
      </Box>

      {/* New Table Button */}
      <Box px="sm" py="xs">
        <Button
          size="xs"
          variant="outline"
          fullWidth
          leftSection={<IconPlus size={14} />}
          onClick={openCreateTable}
          styles={{
            root: {
              borderColor: 'var(--supabase-border)',
              color: 'var(--supabase-text)',
              justifyContent: 'flex-start',
              fontWeight: 400,
              '&:hover': {
                backgroundColor: 'var(--supabase-bg-surface)',
              },
            },
          }}
        >
          New table
        </Button>
      </Box>

      {/* Search Tables */}
      <Box px="sm" pb="xs">
        <TextInput
          size="xs"
          placeholder="Search tables..."
          leftSection={<IconSearch size={14} />}
          value={tableSearchQuery}
          onChange={(e) => setTableSearchQuery(e.target.value)}
          styles={{
            input: {
              backgroundColor: 'var(--supabase-bg-surface)',
            },
          }}
        />
      </Box>

      {/* Table List */}
      <ScrollArea style={{ flex: 1 }} px="sm" pb="sm">
        {loading ? (
          <Center py="xl">
            <Loader size="sm" />
          </Center>
        ) : !filteredTables || filteredTables.length === 0 ? (
          <Text size="sm" c="dimmed" ta="center" py="xl">
            {tableSearchQuery ? 'No tables match your search' : `No tables in ${selectedSchema}`}
          </Text>
        ) : (
          <Stack gap={2}>
            {filteredTables.map((table) => (
              <Box
                key={table.name}
                py={6}
                px={8}
                style={{
                  cursor: 'pointer',
                  backgroundColor:
                    selectedTable === table.name
                      ? 'var(--supabase-brand-light)'
                      : 'transparent',
                  borderRadius: 4,
                  transition: 'background-color 0.1s ease',
                }}
                onClick={() => {
                  setSelectedTable(table.name);
                  onTableSelect?.();
                }}
                onMouseEnter={(e) => {
                  if (selectedTable !== table.name) {
                    e.currentTarget.style.backgroundColor = 'var(--supabase-bg-surface-hover)';
                  }
                }}
                onMouseLeave={(e) => {
                  if (selectedTable !== table.name) {
                    e.currentTarget.style.backgroundColor = 'transparent';
                  }
                }}
              >
                <Group justify="space-between" wrap="nowrap" gap={8}>
                  <Group gap={8} wrap="nowrap" style={{ minWidth: 0, flex: 1 }}>
                    <IconTable
                      size={16}
                      style={{
                        flexShrink: 0,
                        color: selectedTable === table.name
                          ? 'var(--supabase-brand)'
                          : 'var(--supabase-text-muted)',
                      }}
                    />
                    <Text
                      size="sm"
                      truncate
                      style={{
                        color: selectedTable === table.name
                          ? 'var(--supabase-brand)'
                          : 'var(--supabase-text)',
                      }}
                    >
                      {table.name}
                    </Text>
                  </Group>
                  <Menu position="right-start" shadow="md" withinPortal>
                    <Menu.Target>
                      <ActionIcon
                        size="xs"
                        variant="subtle"
                        onClick={(e) => e.stopPropagation()}
                        style={{ opacity: 0.5 }}
                      >
                        <IconDotsVertical size={14} />
                      </ActionIcon>
                    </Menu.Target>
                    <Menu.Dropdown>
                      <Menu.Item
                        leftSection={<IconEdit size={14} />}
                      >
                        Edit table
                      </Menu.Item>
                      <Menu.Item
                        color="red"
                        leftSection={<IconTrash size={14} />}
                        onClick={(e) => {
                          e.stopPropagation();
                          setSelectedTable(table.name);
                          openDeleteTable();
                        }}
                      >
                        Delete table
                      </Menu.Item>
                    </Menu.Dropdown>
                  </Menu>
                </Group>
              </Box>
            ))}
          </Stack>
        )}
      </ScrollArea>
    </>
  );

  return (
    <PageContainer title="Table Editor" description="View and edit your database tables" fullWidth noPadding noHeader>
      {/* Mobile Drawer for Table List */}
      <Drawer
        opened={tableSidebarOpened}
        onClose={closeTableSidebar}
        title="Tables"
        size="280px"
        padding="0"
        styles={{
          body: {
            padding: 0,
            height: 'calc(100% - 60px)',
            display: 'flex',
            flexDirection: 'column',
          },
          header: {
            borderBottom: '1px solid var(--supabase-border)',
          },
        }}
      >
        <SidebarContent onTableSelect={closeTableSidebar} />
      </Drawer>

      <Box style={{ display: 'flex', height: 'calc(100vh - 96px)' }}>
        {/* Table Sidebar - Hidden on mobile */}
        {!isMobile && (
          <Box
            style={{
              width: isTablet ? 220 : 280,
              minWidth: isTablet ? 220 : 280,
              borderRight: '1px solid var(--supabase-border)',
              display: 'flex',
              flexDirection: 'column',
              backgroundColor: 'var(--supabase-bg)',
            }}
          >
            <SidebarContent />
          </Box>
        )}

        {/* Data Grid */}
        <Box style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0 }}>
          {selectedTable ? (
            <>
              {/* Table Tab Bar */}
              <Box
                style={{
                  borderBottom: '1px solid var(--supabase-border)',
                  backgroundColor: 'var(--supabase-bg)',
                  display: 'flex',
                  alignItems: 'center',
                  paddingLeft: isMobile ? 8 : 0,
                  paddingRight: 8,
                  minHeight: 40,
                  gap: 8,
                }}
              >
                {/* Mobile sidebar toggle */}
                {isMobile && (
                  <Tooltip label="Show tables">
                    <ActionIcon
                      variant="subtle"
                      size="sm"
                      onClick={openTableSidebar}
                    >
                      <IconLayoutSidebar size={16} />
                    </ActionIcon>
                  </Tooltip>
                )}
                {openTabs.length > 0 ? (
                  <TableTabs
                    tabs={openTabs}
                    activeTabId={activeTabId}
                    onTabClick={(tabId) => {
                      const tab = openTabs.find((t) => t.id === tabId);
                      if (tab) {
                        setActiveTabId(tabId);
                        setSelectedTable(tab.table);
                        if (tab.schema !== selectedSchema) {
                          setSelectedSchema(tab.schema);
                        }
                      }
                    }}
                    onTabDoubleClick={makeTabPermanent}
                    onTabClose={closeTab}
                    onAddTab={() => {
                      // Open table selection or show all tables
                    }}
                  />
                ) : (
                  <Group gap={0} style={{ flex: 1, minWidth: 0 }} pl={12}>
                    <Box
                      px="sm"
                      py={8}
                      style={{
                        borderBottom: '2px solid var(--supabase-brand)',
                        marginBottom: -1,
                        minWidth: 0,
                      }}
                    >
                      <Group gap={6} wrap="nowrap">
                        <IconTable size={14} color="var(--supabase-brand)" style={{ flexShrink: 0 }} />
                        <Text size="sm" fw={500} truncate style={{ color: 'var(--supabase-brand)' }}>
                          {selectedTable}
                        </Text>
                      </Group>
                    </Box>
                    {!isMobile && (
                      <ActionIcon
                        variant="subtle"
                        size="sm"
                        ml={4}
                        style={{ opacity: 0.5 }}
                        onClick={() => addTab(selectedSchema, selectedTable!, false)}
                      >
                        <IconPlus size={14} />
                      </ActionIcon>
                    )}
                  </Group>
                )}
              </Box>

              {/* Toolbar */}
              <Box
                px="sm"
                py={8}
                style={{
                  borderBottom: '1px solid var(--supabase-border)',
                  backgroundColor: 'var(--supabase-bg)',
                }}
              >
                <Group justify="space-between" wrap={isMobile ? 'wrap' : 'nowrap'} gap={8}>
                  {/* Left side - Filter, Sort, Columns, Export, Insert */}
                  <Group gap={8} wrap="nowrap">
                    {!isMobile && (
                      <>
                        <Button
                          size="xs"
                          variant={filterOpened ? 'filled' : 'outline'}
                          leftSection={<IconFilter size={14} />}
                          onClick={() => filterOpened ? closeFilter() : openFilter()}
                          rightSection={filters.length > 0 ? (
                            <Badge size="xs" variant="filled" color="green" circle>
                              {filters.length}
                            </Badge>
                          ) : undefined}
                          styles={{
                            root: {
                              borderColor: filterOpened ? 'var(--supabase-brand)' : 'var(--supabase-border)',
                              backgroundColor: filterOpened ? 'var(--supabase-brand)' : 'transparent',
                              color: filterOpened ? 'white' : 'var(--supabase-text)',
                              fontWeight: 400,
                              '&:hover': {
                                backgroundColor: filterOpened ? 'var(--supabase-brand-hover)' : 'var(--supabase-bg-surface)',
                              },
                            },
                          }}
                        >
                          Filter
                        </Button>

                        <Button
                          size="xs"
                          variant="outline"
                          leftSection={<IconSortAscending size={14} />}
                          rightSection={sortColumn ? (
                            <Badge size="xs" variant="light" color="blue">
                              {sortDirection === 'asc' ? 'A-Z' : 'Z-A'}
                            </Badge>
                          ) : undefined}
                          styles={{
                            root: {
                              borderColor: 'var(--supabase-border)',
                              color: 'var(--supabase-text)',
                              fontWeight: 400,
                              '&:hover': {
                                backgroundColor: 'var(--supabase-bg-surface)',
                              },
                            },
                          }}
                        >
                          Sort
                        </Button>

                        <ColumnManager
                          columns={columns}
                          visibleColumns={visibleColumns}
                          columnOrder={columnOrder}
                          onVisibilityChange={(colName, visible) => {
                            const newVisible = new Set(visibleColumns);
                            if (visible) {
                              newVisible.add(colName);
                            } else {
                              newVisible.delete(colName);
                            }
                            setVisibleColumns(newVisible);
                          }}
                          onToggleAll={(visible) => {
                            if (visible) {
                              setVisibleColumns(new Set(columns.map((c) => c.name)));
                            } else {
                              setVisibleColumns(new Set());
                            }
                          }}
                          onReorder={setColumnOrder}
                        />

                        <Button
                          size="xs"
                          variant="outline"
                          leftSection={<IconDownload size={14} />}
                          onClick={openExport}
                          styles={{
                            root: {
                              borderColor: 'var(--supabase-border)',
                              color: 'var(--supabase-text)',
                              fontWeight: 400,
                              '&:hover': {
                                backgroundColor: 'var(--supabase-bg-surface)',
                              },
                            },
                          }}
                        >
                          Export
                        </Button>
                      </>
                    )}

                    <Button
                      size="xs"
                      leftSection={
                        <Box
                          style={{
                            display: 'flex',
                            alignItems: 'center',
                            justifyContent: 'center',
                            width: 16,
                            height: 16,
                            backgroundColor: 'rgba(255, 255, 255, 0.2)',
                            borderRadius: 3,
                          }}
                        >
                          <IconCheck size={10} />
                        </Box>
                      }
                      onClick={() => {
                        setRowFormData({});
                        openInsertRow();
                      }}
                      styles={{
                        root: {
                          backgroundColor: 'var(--supabase-brand)',
                          '&:hover': {
                            backgroundColor: 'var(--supabase-brand-hover)',
                          },
                        },
                      }}
                    >
                      Insert
                    </Button>

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
                        {isMobile ? selectedRows.size : `Delete (${selectedRows.size})`}
                      </Button>
                    )}
                  </Group>

                  {/* Right side - RLS, Index Advisor, Realtime, Role */}
                  <Group gap={8} wrap="nowrap">
                    {!isMobile && !isTablet && (
                      <>
                        <Button
                          size="xs"
                          variant="outline"
                          leftSection={<IconShield size={14} />}
                          rightSection={
                            currentTable?.rls_enabled ? (
                              <Badge size="xs" variant="light" color="green" style={{ marginLeft: 4 }}>
                                Enabled
                              </Badge>
                            ) : undefined
                          }
                          styles={{
                            root: {
                              borderColor: 'var(--supabase-border)',
                              color: 'var(--supabase-text)',
                              fontWeight: 400,
                              '&:hover': {
                                backgroundColor: 'var(--supabase-bg-surface)',
                              },
                            },
                          }}
                        >
                          RLS policies
                        </Button>

                        <Button
                          size="xs"
                          variant="outline"
                          leftSection={<IconBolt size={14} />}
                          styles={{
                            root: {
                              borderColor: 'var(--supabase-border)',
                              color: 'var(--supabase-text)',
                              fontWeight: 400,
                              '&:hover': {
                                backgroundColor: 'var(--supabase-bg-surface)',
                              },
                            },
                          }}
                        >
                          Index Advisor
                        </Button>

                        <Button
                          size="xs"
                          variant="outline"
                          leftSection={<IconBroadcast size={14} />}
                          styles={{
                            root: {
                              borderColor: 'var(--supabase-border)',
                              color: 'var(--supabase-text)',
                              fontWeight: 400,
                              '&:hover': {
                                backgroundColor: 'var(--supabase-bg-surface)',
                              },
                            },
                          }}
                        >
                          Enable Realtime
                        </Button>

                        <Select
                          size="xs"
                          value="postgres"
                          data={[
                            { value: 'postgres', label: 'postgres' },
                            { value: 'anon', label: 'anon' },
                            { value: 'authenticated', label: 'authenticated' },
                          ]}
                          leftSection={<Text size="xs" c="dimmed">Role</Text>}
                          w={150}
                          styles={{
                            input: {
                              backgroundColor: 'var(--supabase-bg)',
                              borderColor: 'var(--supabase-border)',
                            },
                          }}
                        />
                      </>
                    )}

                    {/* Refresh */}
                    <Tooltip label="Refresh">
                      <ActionIcon variant="subtle" onClick={fetchTableData}>
                        <IconRefresh size={18} />
                      </ActionIcon>
                    </Tooltip>

                    {/* More actions */}
                    <Menu position="bottom-end">
                      <Menu.Target>
                        <ActionIcon variant="subtle">
                          <IconDotsVertical size={18} />
                        </ActionIcon>
                      </Menu.Target>
                      <Menu.Dropdown>
                        {(isMobile || isTablet) && (
                          <>
                            <Menu.Item leftSection={<IconFilter size={14} />}>
                              Filter
                            </Menu.Item>
                            <Menu.Item leftSection={<IconSortAscending size={14} />}>
                              Sort
                            </Menu.Item>
                            <Menu.Item leftSection={<IconShield size={14} />}>
                              RLS policies
                            </Menu.Item>
                            <Menu.Item leftSection={<IconBolt size={14} />}>
                              Index Advisor
                            </Menu.Item>
                            <Menu.Item leftSection={<IconBroadcast size={14} />}>
                              Enable Realtime
                            </Menu.Item>
                            <Menu.Divider />
                          </>
                        )}
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

              {/* Filter Builder Panel */}
              <Collapse in={filterOpened}>
                <Box px="md" pb="md">
                  <FilterBuilder
                    columns={columns}
                    filters={filters}
                    logic={filterLogic}
                    onChange={(newFilters, newLogic) => {
                      setFilters(newFilters);
                      setFilterLogic(newLogic);
                    }}
                    onApply={() => {
                      fetchTableData();
                    }}
                    onClear={() => {
                      setFilters([]);
                      fetchTableData();
                    }}
                    onClose={closeFilter}
                  />
                </Box>
              </Collapse>

              {/* Data Table / Definition View */}
              <Box style={{ flex: 1, overflow: 'hidden' }}>
                {dataLoading ? (
                  <Center py="xl">
                    <Loader size="sm" />
                  </Center>
                ) : activeTab === 'definition' ? (
                  /* Definition Tab - Shows table structure */
                  <ScrollArea style={{ height: '100%' }} p="md">
                    <Box style={{ maxWidth: 800 }}>
                      <Text size="sm" fw={600} mb="md" style={{ color: 'var(--supabase-text)' }}>
                        Table Definition: {selectedTable}
                      </Text>
                      <Box
                        style={{
                          border: '1px solid var(--supabase-border)',
                          borderRadius: 'var(--supabase-radius-lg)',
                          overflow: 'hidden',
                        }}
                      >
                        <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 13 }}>
                          <thead>
                            <tr style={{ backgroundColor: 'var(--supabase-bg-surface)' }}>
                              <th style={{ padding: '10px 12px', textAlign: 'left', borderBottom: '1px solid var(--supabase-border)', fontWeight: 600 }}>Column</th>
                              <th style={{ padding: '10px 12px', textAlign: 'left', borderBottom: '1px solid var(--supabase-border)', fontWeight: 600 }}>Type</th>
                              <th style={{ padding: '10px 12px', textAlign: 'left', borderBottom: '1px solid var(--supabase-border)', fontWeight: 600 }}>Default</th>
                              <th style={{ padding: '10px 12px', textAlign: 'center', borderBottom: '1px solid var(--supabase-border)', fontWeight: 600 }}>Nullable</th>
                              <th style={{ padding: '10px 12px', textAlign: 'center', borderBottom: '1px solid var(--supabase-border)', fontWeight: 600 }}>Primary</th>
                              <th style={{ padding: '10px 12px', textAlign: 'center', borderBottom: '1px solid var(--supabase-border)', fontWeight: 600 }}>Unique</th>
                            </tr>
                          </thead>
                          <tbody>
                            {columns.map((col, idx) => (
                              <tr
                                key={col.name}
                                style={{
                                  backgroundColor: idx % 2 === 0 ? 'var(--supabase-bg)' : 'var(--supabase-bg-surface)',
                                }}
                              >
                                <td style={{ padding: '10px 12px', borderBottom: '1px solid var(--supabase-border)' }}>
                                  <Group gap={6}>
                                    {col.is_primary_key && <IconKey size={14} color="var(--supabase-brand)" />}
                                    <Text size="sm" fw={500}>{col.name}</Text>
                                  </Group>
                                </td>
                                <td style={{ padding: '10px 12px', borderBottom: '1px solid var(--supabase-border)' }}>
                                  <Badge size="sm" variant="light" color="gray">
                                    {col.type}
                                  </Badge>
                                </td>
                                <td style={{ padding: '10px 12px', borderBottom: '1px solid var(--supabase-border)', fontFamily: 'monospace', fontSize: 12 }}>
                                  {col.default_value || <Text size="xs" c="dimmed">-</Text>}
                                </td>
                                <td style={{ padding: '10px 12px', borderBottom: '1px solid var(--supabase-border)', textAlign: 'center' }}>
                                  {col.is_nullable ? (
                                    <Badge size="xs" variant="light" color="gray">Yes</Badge>
                                  ) : (
                                    <Badge size="xs" variant="light" color="red">No</Badge>
                                  )}
                                </td>
                                <td style={{ padding: '10px 12px', borderBottom: '1px solid var(--supabase-border)', textAlign: 'center' }}>
                                  {col.is_primary_key && <IconCheck size={16} color="var(--supabase-brand)" />}
                                </td>
                                <td style={{ padding: '10px 12px', borderBottom: '1px solid var(--supabase-border)', textAlign: 'center' }}>
                                  {col.is_unique && <IconCheck size={16} color="var(--supabase-info)" />}
                                </td>
                              </tr>
                            ))}
                          </tbody>
                        </table>
                      </Box>

                      {/* Table Info */}
                      <Box mt="lg">
                        <Text size="sm" fw={600} mb="sm" style={{ color: 'var(--supabase-text)' }}>
                          Table Information
                        </Text>
                        <Stack gap="xs">
                          <Group gap="xs">
                            <Text size="sm" c="dimmed" style={{ width: 120 }}>Schema:</Text>
                            <Text size="sm" fw={500}>{selectedSchema}</Text>
                          </Group>
                          <Group gap="xs">
                            <Text size="sm" c="dimmed" style={{ width: 120 }}>Table name:</Text>
                            <Text size="sm" fw={500}>{selectedTable}</Text>
                          </Group>
                          <Group gap="xs">
                            <Text size="sm" c="dimmed" style={{ width: 120 }}>Columns:</Text>
                            <Text size="sm" fw={500}>{columns.length}</Text>
                          </Group>
                          <Group gap="xs">
                            <Text size="sm" c="dimmed" style={{ width: 120 }}>RLS:</Text>
                            <Badge size="sm" variant="light" color={currentTable?.rls_enabled ? 'green' : 'gray'}>
                              {currentTable?.rls_enabled ? 'Enabled' : 'Disabled'}
                            </Badge>
                          </Group>
                        </Stack>
                      </Box>

                      {/* Actions */}
                      <Group mt="lg" gap="sm">
                        <Button
                          size="xs"
                          variant="outline"
                          leftSection={<IconColumns size={14} />}
                          onClick={openAddColumn}
                        >
                          Add Column
                        </Button>
                      </Group>
                    </Box>
                  </ScrollArea>
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
                                  <Tooltip label="Expand row">
                                    <ActionIcon
                                      size="xs"
                                      variant="subtle"
                                      onClick={() => {
                                        setDetailRow(row);
                                        openRowDetail();
                                      }}
                                    >
                                      <IconMaximize size={14} />
                                    </ActionIcon>
                                  </Tooltip>
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
                                        ...getCellStyle(col.type),
                                      }}
                                    >
                                      {formatCellValue(row[col.name], col.type)}
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

              {/* Pagination Footer */}
              {tableData && tableData.length > 0 && (
                <Box
                  px="sm"
                  py={8}
                  style={{
                    borderTop: '1px solid var(--supabase-border)',
                    backgroundColor: 'var(--supabase-bg)',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'space-between',
                  }}
                >
                  {/* Left side - Page navigation */}
                  <Group gap={8}>
                    <ActionIcon
                      size="sm"
                      variant="outline"
                      disabled={currentPage === 1}
                      onClick={() => setCurrentPage((p) => Math.max(1, p - 1))}
                      styles={{
                        root: {
                          borderColor: 'var(--supabase-border)',
                          '&:hover:not(:disabled)': {
                            backgroundColor: 'var(--supabase-bg-surface)',
                          },
                        },
                      }}
                    >
                      <IconChevronLeft size={14} />
                    </ActionIcon>

                    <Group gap={4}>
                      <Text size="xs" c="dimmed">Page</Text>
                      <TextInput
                        size="xs"
                        value={currentPage}
                        onChange={(e) => {
                          const val = parseInt(e.target.value, 10);
                          if (!isNaN(val) && val >= 1 && val <= totalPages) {
                            setCurrentPage(val);
                          }
                        }}
                        w={40}
                        styles={{
                          input: {
                            textAlign: 'center',
                            padding: '4px',
                          },
                        }}
                      />
                      <Text size="xs" c="dimmed">of {totalPages || 1}</Text>
                    </Group>

                    <ActionIcon
                      size="sm"
                      variant="outline"
                      disabled={currentPage >= totalPages}
                      onClick={() => setCurrentPage((p) => Math.min(totalPages, p + 1))}
                      styles={{
                        root: {
                          borderColor: 'var(--supabase-border)',
                          '&:hover:not(:disabled)': {
                            backgroundColor: 'var(--supabase-bg-surface)',
                          },
                        },
                      }}
                    >
                      <IconChevronRight size={14} />
                    </ActionIcon>
                  </Group>

                  {/* Right side - Row count and tabs */}
                  <Group gap={16}>
                    <Group gap={4}>
                      <Select
                        size="xs"
                        value={String(rowsPerPage)}
                        onChange={(val) => val && setRowsPerPage(parseInt(val, 10))}
                        data={[
                          { value: '50', label: '50 rows' },
                          { value: '100', label: '100 rows' },
                          { value: '200', label: '200 rows' },
                        ]}
                        w={100}
                        styles={{
                          input: {
                            backgroundColor: 'var(--supabase-bg)',
                            borderColor: 'var(--supabase-border)',
                          },
                        }}
                      />
                    </Group>

                    {!isMobile && (
                      <Text size="xs" c="dimmed">
                        {tableData.length} records
                      </Text>
                    )}

                    {/* Data / Definition tabs */}
                    <SegmentedControl
                      size="xs"
                      value={activeTab}
                      onChange={(value) => setActiveTab(value as 'data' | 'definition')}
                      data={[
                        { label: 'Data', value: 'data' },
                        { label: 'Definition', value: 'definition' },
                      ]}
                      styles={{
                        root: {
                          backgroundColor: 'var(--supabase-bg-surface)',
                          border: '1px solid var(--supabase-border)',
                        },
                        indicator: {
                          backgroundColor: 'var(--supabase-bg)',
                          boxShadow: 'var(--supabase-shadow-sm)',
                        },
                      }}
                    />
                  </Group>
                </Box>
              )}
            </>
          ) : (
            <Center style={{ flex: 1 }}>
              <Stack align="center" gap="lg">
                <EmptyState
                  icon={<IconTable size={32} />}
                  title="Select a table"
                  description={isMobile ? "Tap the button below to browse tables" : "Choose a table from the sidebar or create a new one"}
                  action={{
                    label: 'Create table',
                    onClick: openCreateTable,
                  }}
                />
                {isMobile && (
                  <Button
                    variant="outline"
                    leftSection={<IconLayoutSidebar size={16} />}
                    onClick={openTableSidebar}
                    styles={{
                      root: {
                        borderColor: 'var(--supabase-border)',
                        color: 'var(--supabase-text)',
                      },
                    }}
                  >
                    Browse tables
                  </Button>
                )}
              </Stack>
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

      {/* Export Modal */}
      <ExportModal
        opened={exportOpened}
        onClose={closeExport}
        schema={selectedSchema}
        table={selectedTable || ''}
        totalRows={totalRowCount || tableData.length}
        selectedRows={selectedRows.size}
        filters={buildFilterQuery()}
      />

      {/* Row Detail Drawer */}
      <RowDetailDrawer
        opened={rowDetailOpened}
        onClose={closeRowDetail}
        columns={columns}
        row={detailRow}
        onSave={async (data) => {
          if (!detailRow || !selectedTable) return;
          const pkColumn = columns.find((c) => c.is_primary_key);
          if (!pkColumn) return;
          const pkValue = detailRow[pkColumn.name];
          await databaseApi.updateRow(
            selectedTable,
            `${pkColumn.name}=eq.${pkValue}`,
            data
          );
          notifications.show({
            title: 'Success',
            message: 'Row updated successfully',
            color: 'green',
          });
          closeRowDetail();
          fetchTableData();
        }}
      />
    </PageContainer>
  );
}
