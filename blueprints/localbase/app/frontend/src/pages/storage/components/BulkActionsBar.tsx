import {
  Box,
  Group,
  Button,
  Text,
  ActionIcon,
} from '@mantine/core';
import {
  IconX,
  IconTrash,
  IconDownload,
  IconCopy,
  IconArrowRight,
} from '@tabler/icons-react';

interface BulkActionsBarProps {
  selectedCount: number;
  onClearSelection: () => void;
  onDelete: () => void;
  onDownload: () => void;
  onMove: () => void;
  onCopy: () => void;
}

export function BulkActionsBar({
  selectedCount,
  onClearSelection,
  onDelete,
  onDownload,
  onMove,
  onCopy,
}: BulkActionsBarProps) {
  if (selectedCount === 0) return null;

  return (
    <Box
      px="md"
      py="xs"
      style={{
        backgroundColor: 'var(--supabase-brand-light)',
        borderBottom: '1px solid var(--supabase-border)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
      }}
    >
      <Group gap="sm">
        <ActionIcon
          variant="subtle"
          onClick={onClearSelection}
          size="sm"
        >
          <IconX size={16} />
        </ActionIcon>
        <Text size="sm" fw={500}>
          {selectedCount} item{selectedCount > 1 ? 's' : ''} selected
        </Text>
      </Group>

      <Group gap="xs">
        <Button
          variant="outline"
          size="xs"
          leftSection={<IconDownload size={14} />}
          onClick={onDownload}
        >
          Download
        </Button>
        <Button
          variant="outline"
          size="xs"
          leftSection={<IconArrowRight size={14} />}
          onClick={onMove}
        >
          Move
        </Button>
        <Button
          variant="outline"
          size="xs"
          leftSection={<IconCopy size={14} />}
          onClick={onCopy}
        >
          Copy
        </Button>
        <Button
          variant="outline"
          size="xs"
          color="red"
          leftSection={<IconTrash size={14} />}
          onClick={onDelete}
        >
          Delete
        </Button>
      </Group>
    </Box>
  );
}
