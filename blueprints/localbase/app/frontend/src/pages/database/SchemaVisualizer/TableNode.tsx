import { memo } from 'react';
import { Handle, Position } from '@xyflow/react';
import { Box, Text, Group, Tooltip, ActionIcon } from '@mantine/core';
import {
  IconKey,
  IconHash,
  IconFingerprint,
  IconDiamond,
  IconDiamondFilled,
  IconExternalLink,
} from '@tabler/icons-react';

interface ColumnData {
  name: string;
  type: string;
  is_nullable: boolean;
  is_primary_key: boolean;
  is_unique: boolean;
  is_identity: boolean;
}

interface TableNodeData {
  label: string;
  schema: string;
  columns: ColumnData[];
}

const ColumnIcon = ({ column }: { column: ColumnData }) => {
  if (column.is_primary_key) {
    return (
      <Tooltip label="Primary Key" withArrow position="left">
        <IconKey size={12} color="var(--supabase-brand)" />
      </Tooltip>
    );
  }
  if (column.is_identity) {
    return (
      <Tooltip label="Identity" withArrow position="left">
        <IconHash size={12} color="#666" />
      </Tooltip>
    );
  }
  if (column.is_unique) {
    return (
      <Tooltip label="Unique" withArrow position="left">
        <IconFingerprint size={12} color="#666" />
      </Tooltip>
    );
  }
  if (column.is_nullable) {
    return (
      <Tooltip label="Nullable" withArrow position="left">
        <IconDiamond size={12} color="#666" />
      </Tooltip>
    );
  }
  return (
    <Tooltip label="Non-Nullable" withArrow position="left">
      <IconDiamondFilled size={12} color="#666" />
    </Tooltip>
  );
};

function TableNode({ data }: { data: TableNodeData }) {
  const ROW_HEIGHT = 28;
  const HEADER_HEIGHT = 36;

  return (
    <Box
      style={{
        background: 'var(--supabase-bg-surface)',
        border: '1px solid var(--supabase-border)',
        borderRadius: 8,
        minWidth: 250,
        maxWidth: 320,
        boxShadow: '0 2px 8px rgba(0, 0, 0, 0.15)',
      }}
    >
      {/* Header */}
      <Group
        justify="space-between"
        px="sm"
        py="xs"
        style={{
          borderBottom: '1px solid var(--supabase-border)',
          borderRadius: '8px 8px 0 0',
          background: 'var(--supabase-bg)',
        }}
      >
        <Group gap={6}>
          <IconKey size={14} color="var(--supabase-brand)" />
          <Text size="sm" fw={600} style={{ color: 'var(--mantine-color-text)' }}>
            {data.label}
          </Text>
        </Group>
        <ActionIcon
          variant="subtle"
          size="xs"
          component="a"
          href={`/table-editor?schema=${data.schema}&table=${data.label}`}
          target="_blank"
        >
          <IconExternalLink size={12} />
        </ActionIcon>
      </Group>

      {/* Columns */}
      <Box py={4}>
        {data.columns.map((column, index) => (
          <Group
            key={column.name}
            px="sm"
            py={4}
            gap={8}
            wrap="nowrap"
            style={{
              height: ROW_HEIGHT,
              position: 'relative',
            }}
          >
            {/* Left handle for incoming relationships */}
            <Handle
              type="target"
              position={Position.Left}
              id={`${column.name}-target`}
              style={{
                background: 'var(--supabase-brand)',
                width: 8,
                height: 8,
                top: HEADER_HEIGHT + index * ROW_HEIGHT + ROW_HEIGHT / 2,
                left: -4,
              }}
            />

            <ColumnIcon column={column} />

            <Text
              size="xs"
              fw={column.is_primary_key ? 600 : 400}
              style={{
                flex: 1,
                color: column.is_primary_key ? 'var(--mantine-color-text)' : 'var(--mantine-color-dimmed)',
              }}
            >
              {column.name}
            </Text>

            <Text size="xs" c="dimmed" style={{ opacity: 0.7 }}>
              {column.type}
            </Text>

            {/* Right handle for outgoing relationships */}
            <Handle
              type="source"
              position={Position.Right}
              id={`${column.name}-source`}
              style={{
                background: 'var(--supabase-brand)',
                width: 8,
                height: 8,
                top: HEADER_HEIGHT + index * ROW_HEIGHT + ROW_HEIGHT / 2,
                right: -4,
              }}
            />
          </Group>
        ))}
      </Box>
    </Box>
  );
}

export default memo(TableNode);
