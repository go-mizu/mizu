import {
  Box,
  Button,
  Group,
  Select,
  TextInput,
  ActionIcon,
  Text,
  Stack,
  SegmentedControl,
  Paper,
  CloseButton,
} from '@mantine/core';
import { IconPlus, IconTrash, IconFilter } from '@tabler/icons-react';
import type { Column } from '../../types';

export interface Filter {
  id: string;
  column: string;
  operator: string;
  value: string;
}

interface FilterBuilderProps {
  columns: Column[];
  filters: Filter[];
  logic: 'AND' | 'OR';
  onChange: (filters: Filter[], logic: 'AND' | 'OR') => void;
  onApply: () => void;
  onClear: () => void;
  onClose?: () => void;
}

// Operators by column type
const getOperatorsForType = (type: string): { value: string; label: string }[] => {
  const baseOperators = [
    { value: 'eq', label: 'equals' },
    { value: 'neq', label: 'not equals' },
  ];

  const nullOperator = { value: 'is', label: 'is null/true/false' };

  if (type.includes('text') || type.includes('varchar') || type.includes('char')) {
    return [
      ...baseOperators,
      { value: 'like', label: 'contains' },
      { value: 'ilike', label: 'contains (case insensitive)' },
      nullOperator,
    ];
  }

  if (
    type.includes('int') ||
    type.includes('numeric') ||
    type.includes('decimal') ||
    type.includes('real') ||
    type.includes('double') ||
    type.includes('float')
  ) {
    return [
      ...baseOperators,
      { value: 'gt', label: 'greater than' },
      { value: 'gte', label: 'greater than or equal' },
      { value: 'lt', label: 'less than' },
      { value: 'lte', label: 'less than or equal' },
      nullOperator,
    ];
  }

  if (type.includes('timestamp') || type.includes('date') || type.includes('time')) {
    return [
      ...baseOperators,
      { value: 'gt', label: 'after' },
      { value: 'gte', label: 'on or after' },
      { value: 'lt', label: 'before' },
      { value: 'lte', label: 'on or before' },
      nullOperator,
    ];
  }

  if (type === 'boolean') {
    return [
      { value: 'is', label: 'is' },
    ];
  }

  if (type.includes('uuid')) {
    return [
      ...baseOperators,
      nullOperator,
    ];
  }

  if (type.includes('json')) {
    return [
      { value: 'cs', label: 'contains' },
      { value: 'cd', label: 'contained by' },
      nullOperator,
    ];
  }

  return [
    ...baseOperators,
    { value: 'like', label: 'contains' },
    { value: 'ilike', label: 'contains (case insensitive)' },
    nullOperator,
  ];
};

// Get value options for special types
const getValueOptions = (type: string, operator: string): { value: string; label: string }[] | null => {
  if (operator === 'is') {
    return [
      { value: 'null', label: 'NULL' },
      { value: 'true', label: 'TRUE' },
      { value: 'false', label: 'FALSE' },
    ];
  }

  if (type === 'boolean') {
    return [
      { value: 'true', label: 'true' },
      { value: 'false', label: 'false' },
    ];
  }

  return null;
};

export function FilterBuilder({
  columns,
  filters,
  logic,
  onChange,
  onApply,
  onClear,
  onClose,
}: FilterBuilderProps) {
  const addFilter = () => {
    const newFilter: Filter = {
      id: crypto.randomUUID(),
      column: columns[0]?.name || '',
      operator: 'eq',
      value: '',
    };
    onChange([...filters, newFilter], logic);
  };

  const updateFilter = (id: string, updates: Partial<Filter>) => {
    const updatedFilters = filters.map((f) =>
      f.id === id ? { ...f, ...updates } : f
    );
    onChange(updatedFilters, logic);
  };

  const removeFilter = (id: string) => {
    onChange(
      filters.filter((f) => f.id !== id),
      logic
    );
  };

  const getColumnType = (columnName: string): string => {
    const col = columns.find((c) => c.name === columnName);
    return col?.type || 'text';
  };

  return (
    <Paper
      p="md"
      withBorder
      style={{
        backgroundColor: 'var(--lb-bg-secondary)',
        borderColor: 'var(--lb-border-default)',
        borderRadius: 'var(--lb-radius-md)',
      }}
    >
      <Group justify="space-between" mb="sm">
        <Group gap="xs">
          <IconFilter size={16} style={{ color: 'var(--lb-brand)' }} />
          <Text size="sm" fw={600}>
            Filter rows
          </Text>
        </Group>
        {onClose && (
          <CloseButton size="sm" onClick={onClose} />
        )}
      </Group>

      {filters.length > 0 && (
        <Stack gap="xs" mb="sm">
          {filters.map((filter, index) => {
            const columnType = getColumnType(filter.column);
            const operators = getOperatorsForType(columnType);
            const valueOptions = getValueOptions(columnType, filter.operator);

            return (
              <Group key={filter.id} gap="xs" wrap="nowrap" align="flex-start">
                {index > 0 && (
                  <Box w={60}>
                    <SegmentedControl
                      size="xs"
                      value={logic}
                      onChange={(value) => onChange(filters, value as 'AND' | 'OR')}
                      data={[
                        { value: 'AND', label: 'AND' },
                        { value: 'OR', label: 'OR' },
                      ]}
                      styles={{
                        root: {
                          backgroundColor: 'var(--lb-bg-primary)',
                          border: '1px solid var(--lb-border-default)',
                        },
                      }}
                    />
                  </Box>
                )}
                {index === 0 && <Box w={60} />}

                <Select
                  size="xs"
                  value={filter.column}
                  onChange={(value) => value && updateFilter(filter.id, { column: value })}
                  data={columns.map((c) => ({ value: c.name, label: c.name }))}
                  searchable
                  style={{ flex: 1 }}
                  styles={{
                    input: {
                      backgroundColor: 'var(--lb-bg-primary)',
                      borderColor: 'var(--lb-border-default)',
                    },
                  }}
                />

                <Select
                  size="xs"
                  value={filter.operator}
                  onChange={(value) => value && updateFilter(filter.id, { operator: value })}
                  data={operators}
                  w={180}
                  styles={{
                    input: {
                      backgroundColor: 'var(--lb-bg-primary)',
                      borderColor: 'var(--lb-border-default)',
                    },
                  }}
                />

                {valueOptions ? (
                  <Select
                    size="xs"
                    value={filter.value}
                    onChange={(value) => value && updateFilter(filter.id, { value })}
                    data={valueOptions}
                    style={{ flex: 1 }}
                    styles={{
                      input: {
                        backgroundColor: 'var(--lb-bg-primary)',
                        borderColor: 'var(--lb-border-default)',
                      },
                    }}
                  />
                ) : (
                  <TextInput
                    size="xs"
                    value={filter.value}
                    onChange={(e) => updateFilter(filter.id, { value: e.target.value })}
                    placeholder="Enter value..."
                    style={{ flex: 1 }}
                    styles={{
                      input: {
                        backgroundColor: 'var(--lb-bg-primary)',
                        borderColor: 'var(--lb-border-default)',
                      },
                    }}
                  />
                )}

                <ActionIcon
                  size="sm"
                  variant="subtle"
                  color="red"
                  onClick={() => removeFilter(filter.id)}
                >
                  <IconTrash size={14} />
                </ActionIcon>
              </Group>
            );
          })}
        </Stack>
      )}

      <Group justify="space-between">
        <Button
          size="xs"
          variant="subtle"
          leftSection={<IconPlus size={14} />}
          onClick={addFilter}
        >
          Add filter
        </Button>

        <Group gap="xs">
          {filters.length > 0 && (
            <Button size="xs" variant="subtle" color="red" onClick={onClear}>
              Clear all
            </Button>
          )}
          <Button
            size="xs"
            disabled={filters.length === 0}
            onClick={onApply}
            styles={{
              root: {
                backgroundColor: 'var(--lb-brand)',
                transition: 'var(--lb-transition-fast)',
                '&:hover': {
                  backgroundColor: 'var(--lb-brand-hover)',
                },
              },
            }}
          >
            Apply filters
          </Button>
        </Group>
      </Group>
    </Paper>
  );
}
