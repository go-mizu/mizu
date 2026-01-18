import { useState } from 'react';
import {
  Popover,
  Stack,
  Group,
  Text,
  Checkbox,
  Button,
  ActionIcon,
  TextInput,
  Divider,
  Box,
  ScrollArea,
} from '@mantine/core';
import {
  IconColumns,
  IconSearch,
  IconEye,
  IconEyeOff,
} from '@tabler/icons-react';
import type { Column } from '../../types';

interface ColumnManagerProps {
  columns: Column[];
  visibleColumns: Set<string>;
  columnOrder: string[];
  onVisibilityChange: (columnName: string, visible: boolean) => void;
  onToggleAll: (visible: boolean) => void;
  onReorder?: (columnOrder: string[]) => void;
}

export function ColumnManager({
  columns,
  visibleColumns,
  columnOrder,
  onVisibilityChange,
  onToggleAll,
  // onReorder - reserved for future drag-drop reordering
}: ColumnManagerProps) {
  const [opened, setOpened] = useState(false);
  const [search, setSearch] = useState('');

  // Sort columns by order
  const orderedColumns = [...columns].sort((a, b) => {
    const indexA = columnOrder.indexOf(a.name);
    const indexB = columnOrder.indexOf(b.name);
    if (indexA === -1 && indexB === -1) return 0;
    if (indexA === -1) return 1;
    if (indexB === -1) return -1;
    return indexA - indexB;
  });

  // Filter columns by search
  const filteredColumns = orderedColumns.filter((col) =>
    col.name.toLowerCase().includes(search.toLowerCase())
  );

  const allVisible = columns.every((col) => visibleColumns.has(col.name));
  const noneVisible = columns.every((col) => !visibleColumns.has(col.name));

  return (
    <Popover
      opened={opened}
      onClose={() => setOpened(false)}
      position="bottom-end"
      shadow="md"
      width={280}
    >
      <Popover.Target>
        <Button
          size="xs"
          variant="outline"
          leftSection={<IconColumns size={14} />}
          onClick={() => setOpened(!opened)}
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
          Columns
        </Button>
      </Popover.Target>

      <Popover.Dropdown
        p="xs"
        style={{
          backgroundColor: 'var(--supabase-bg)',
          borderColor: 'var(--supabase-border)',
        }}
      >
        <Stack gap="xs">
          <TextInput
            size="xs"
            placeholder="Search columns..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            leftSection={<IconSearch size={14} />}
            styles={{
              input: {
                backgroundColor: 'var(--supabase-bg-surface)',
                borderColor: 'var(--supabase-border)',
              },
            }}
          />

          <Group justify="space-between">
            <Text size="xs" c="dimmed">
              {visibleColumns.size} of {columns.length} visible
            </Text>
            <Group gap={4}>
              <Button
                size="compact-xs"
                variant="subtle"
                onClick={() => onToggleAll(true)}
                disabled={allVisible}
              >
                Show all
              </Button>
              <Button
                size="compact-xs"
                variant="subtle"
                onClick={() => onToggleAll(false)}
                disabled={noneVisible}
              >
                Hide all
              </Button>
            </Group>
          </Group>

          <Divider />

          <ScrollArea style={{ maxHeight: 300 }}>
            <Stack gap={4}>
              {filteredColumns.map((col) => (
                <Group
                  key={col.name}
                  gap="xs"
                  py={4}
                  px={6}
                  wrap="nowrap"
                  style={{
                    borderRadius: 4,
                    cursor: 'pointer',
                    backgroundColor: visibleColumns.has(col.name)
                      ? 'transparent'
                      : 'var(--supabase-bg-surface)',
                  }}
                >
                  <Checkbox
                    size="xs"
                    checked={visibleColumns.has(col.name)}
                    onChange={(e) =>
                      onVisibilityChange(col.name, e.currentTarget.checked)
                    }
                    styles={{
                      input: {
                        cursor: 'pointer',
                      },
                    }}
                  />
                  <Box style={{ flex: 1, minWidth: 0 }}>
                    <Text size="xs" truncate fw={500}>
                      {col.name}
                    </Text>
                    <Text size="xs" c="dimmed" truncate>
                      {col.type}
                    </Text>
                  </Box>
                  <ActionIcon
                    size="xs"
                    variant="subtle"
                    onClick={() =>
                      onVisibilityChange(col.name, !visibleColumns.has(col.name))
                    }
                  >
                    {visibleColumns.has(col.name) ? (
                      <IconEye size={12} />
                    ) : (
                      <IconEyeOff size={12} />
                    )}
                  </ActionIcon>
                </Group>
              ))}
            </Stack>
          </ScrollArea>

          {filteredColumns.length === 0 && (
            <Text size="xs" c="dimmed" ta="center" py="sm">
              No columns match "{search}"
            </Text>
          )}
        </Stack>
      </Popover.Dropdown>
    </Popover>
  );
}
