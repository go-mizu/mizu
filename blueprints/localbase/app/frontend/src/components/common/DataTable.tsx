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
      <Table
        style={{
          border: '1px solid var(--lb-border-default)',
          borderRadius: 'var(--lb-radius-lg)',
          overflow: 'hidden',
        }}
      >
        <Table.Thead style={{ backgroundColor: 'var(--lb-table-header-bg)' }}>
          <Table.Tr>
            {selectable && (
              <Table.Th style={{ width: 40, padding: '12px 16px' }}>
                <Skeleton height={20} width={20} />
              </Table.Th>
            )}
            {columns.map((col) => (
              <Table.Th
                key={col.key}
                style={{
                  width: col.width,
                  padding: '12px 16px',
                  fontSize: 'var(--lb-text-sm)',
                  fontWeight: 500,
                  color: 'var(--lb-text-secondary)',
                  textTransform: 'uppercase',
                  letterSpacing: '0.05em',
                }}
              >
                {col.header}
              </Table.Th>
            ))}
          </Table.Tr>
        </Table.Thead>
        <Table.Tbody>
          {Array.from({ length: 5 }).map((_, i) => (
            <Table.Tr key={i}>
              {selectable && (
                <Table.Td style={{ padding: '12px 16px' }}>
                  <Skeleton height={20} width={20} />
                </Table.Td>
              )}
              {columns.map((col) => (
                <Table.Td key={col.key} style={{ padding: '12px 16px' }}>
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
          border: '1px solid var(--lb-border-default)',
          borderRadius: 'var(--lb-radius-lg)',
        }}
      >
        <Table
          stickyHeader={stickyHeader}
          highlightOnHover={!!onRowClick}
          style={{ minWidth: '100%' }}
        >
          <Table.Thead style={{ backgroundColor: 'var(--lb-table-header-bg)' }}>
            <Table.Tr>
              {selectable && (
                <Table.Th style={{ width: 40, padding: '12px 16px' }}>
                  <Checkbox
                    checked={allSelected}
                    indeterminate={someSelected}
                    onChange={toggleAll}
                  />
                </Table.Th>
              )}
              {columns.map((col) => (
                <Table.Th
                  key={col.key}
                  style={{
                    width: col.width,
                    padding: '12px 16px',
                    fontSize: 'var(--lb-text-sm)',
                    fontWeight: 500,
                    color: 'var(--lb-text-secondary)',
                    textTransform: 'uppercase',
                    letterSpacing: '0.05em',
                    borderBottom: '1px solid var(--lb-border-default)',
                  }}
                >
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
                    backgroundColor: isSelected ? 'var(--lb-table-row-selected)' : undefined,
                    transition: 'background-color var(--lb-transition-fast)',
                  }}
                >
                  {selectable && (
                    <Table.Td
                      onClick={(e) => e.stopPropagation()}
                      style={{
                        padding: '12px 16px',
                        borderBottom: '1px solid var(--lb-border-muted)',
                      }}
                    >
                      <Checkbox
                        checked={isSelected}
                        onChange={() => toggleRow(item)}
                      />
                    </Table.Td>
                  )}
                  {columns.map((col) => (
                    <Table.Td
                      key={col.key}
                      style={{
                        padding: '12px 16px',
                        fontSize: 'var(--lb-text-md)',
                        color: 'var(--lb-text-primary)',
                        borderBottom: '1px solid var(--lb-border-muted)',
                      }}
                    >
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
          <Text
            size="sm"
            style={{
              color: 'var(--lb-text-secondary)',
              fontSize: 'var(--lb-text-sm)',
            }}
          >
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
