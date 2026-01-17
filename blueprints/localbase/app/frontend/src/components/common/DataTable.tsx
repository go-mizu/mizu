import {
  Table,
  Checkbox,
  Group,
  Pagination,
  Text,
  Skeleton,
  Box,
} from '@mantine/core';
import { useMemo, type ReactNode } from 'react';

export interface Column<T> {
  key: string;
  header: string;
  render?: (item: T) => ReactNode;
  width?: number | string;
  sortable?: boolean;
}

interface DataTableProps<T> {
  data: T[];
  columns: Column<T>[];
  loading?: boolean;
  emptyState?: ReactNode;
  selectable?: boolean;
  selectedRows?: T[];
  onSelectionChange?: (selected: T[]) => void;
  onRowClick?: (item: T) => void;
  pagination?: {
    page: number;
    totalPages: number;
    total: number;
    onPageChange: (page: number) => void;
  };
  getRowKey: (item: T) => string;
  stickyHeader?: boolean;
  maxHeight?: number | string;
}

export function DataTable<T>({
  data,
  columns,
  loading = false,
  emptyState,
  selectable = false,
  selectedRows = [],
  onSelectionChange,
  onRowClick,
  pagination,
  getRowKey,
  stickyHeader = false,
  maxHeight,
}: DataTableProps<T>) {
  const selectedKeys = useMemo(
    () => new Set(selectedRows.map(getRowKey)),
    [selectedRows, getRowKey]
  );

  const allSelected = data.length > 0 && data.every((item) => selectedKeys.has(getRowKey(item)));
  const someSelected = data.some((item) => selectedKeys.has(getRowKey(item))) && !allSelected;

  const toggleAll = () => {
    if (!onSelectionChange) return;
    if (allSelected) {
      onSelectionChange([]);
    } else {
      onSelectionChange([...data]);
    }
  };

  const toggleRow = (item: T) => {
    if (!onSelectionChange) return;
    const key = getRowKey(item);
    if (selectedKeys.has(key)) {
      onSelectionChange(selectedRows.filter((r) => getRowKey(r) !== key));
    } else {
      onSelectionChange([...selectedRows, item]);
    }
  };

  // Loading skeleton
  if (loading) {
    return (
      <Table>
        <Table.Thead>
          <Table.Tr>
            {selectable && (
              <Table.Th style={{ width: 40 }}>
                <Skeleton height={20} width={20} />
              </Table.Th>
            )}
            {columns.map((col) => (
              <Table.Th key={col.key} style={{ width: col.width }}>
                {col.header}
              </Table.Th>
            ))}
          </Table.Tr>
        </Table.Thead>
        <Table.Tbody>
          {Array.from({ length: 5 }).map((_, i) => (
            <Table.Tr key={i}>
              {selectable && (
                <Table.Td>
                  <Skeleton height={20} width={20} />
                </Table.Td>
              )}
              {columns.map((col) => (
                <Table.Td key={col.key}>
                  <Skeleton height={20} />
                </Table.Td>
              ))}
            </Table.Tr>
          ))}
        </Table.Tbody>
      </Table>
    );
  }

  // Empty state
  if (data.length === 0 && emptyState) {
    return <>{emptyState}</>;
  }

  return (
    <Box>
      <Box
        style={{
          overflow: 'auto',
          maxHeight: maxHeight,
        }}
      >
        <Table
          stickyHeader={stickyHeader}
          highlightOnHover={!!onRowClick}
          style={{ minWidth: '100%' }}
        >
          <Table.Thead>
            <Table.Tr>
              {selectable && (
                <Table.Th style={{ width: 40 }}>
                  <Checkbox
                    checked={allSelected}
                    indeterminate={someSelected}
                    onChange={toggleAll}
                  />
                </Table.Th>
              )}
              {columns.map((col) => (
                <Table.Th key={col.key} style={{ width: col.width }}>
                  {col.header}
                </Table.Th>
              ))}
            </Table.Tr>
          </Table.Thead>
          <Table.Tbody>
            {data.map((item) => {
              const key = getRowKey(item);
              const isSelected = selectedKeys.has(key);

              return (
                <Table.Tr
                  key={key}
                  onClick={() => onRowClick?.(item)}
                  style={{
                    cursor: onRowClick ? 'pointer' : undefined,
                    backgroundColor: isSelected ? 'var(--supabase-brand-light)' : undefined,
                  }}
                >
                  {selectable && (
                    <Table.Td onClick={(e) => e.stopPropagation()}>
                      <Checkbox
                        checked={isSelected}
                        onChange={() => toggleRow(item)}
                      />
                    </Table.Td>
                  )}
                  {columns.map((col) => (
                    <Table.Td key={col.key}>
                      {col.render
                        ? col.render(item)
                        : String((item as any)[col.key] ?? '')}
                    </Table.Td>
                  ))}
                </Table.Tr>
              );
            })}
          </Table.Tbody>
        </Table>
      </Box>

      {pagination && pagination.totalPages > 1 && (
        <Group justify="space-between" mt="md">
          <Text size="sm" c="dimmed">
            Showing {data.length} of {pagination.total} items
          </Text>
          <Pagination
            value={pagination.page}
            onChange={pagination.onPageChange}
            total={pagination.totalPages}
            size="sm"
          />
        </Group>
      )}
    </Box>
  );
}
