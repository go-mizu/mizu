import { useState, useEffect } from 'react';
import {
  Drawer,
  Box,
  Group,
  Text,
  Stack,
  TextInput,
  Textarea,
  Select,
  Button,
  Badge,
  ActionIcon,
  Tooltip,
  ScrollArea,
  Divider,
  CopyButton,
} from '@mantine/core';
import {
  IconKey,
  IconCopy,
  IconCheck,
} from '@tabler/icons-react';
import type { Column } from '../../types';

interface RowDetailDrawerProps {
  opened: boolean;
  onClose: () => void;
  columns: Column[];
  row: Record<string, any> | null;
  onSave: (data: Record<string, any>) => Promise<void>;
  readOnly?: boolean;
}

export function RowDetailDrawer({
  opened,
  onClose,
  columns,
  row,
  onSave,
  readOnly = false,
}: RowDetailDrawerProps) {
  const [formData, setFormData] = useState<Record<string, any>>({});
  const [saving, setSaving] = useState(false);
  const [hasChanges, setHasChanges] = useState(false);

  // Initialize form data when row changes
  useEffect(() => {
    if (row) {
      const initialData: Record<string, any> = {};
      columns.forEach((col) => {
        const value = row[col.name];
        if (value === null) {
          initialData[col.name] = '';
        } else if (typeof value === 'object') {
          initialData[col.name] = JSON.stringify(value, null, 2);
        } else {
          initialData[col.name] = String(value);
        }
      });
      setFormData(initialData);
      setHasChanges(false);
    }
  }, [row, columns]);

  const handleChange = (column: string, value: any) => {
    setFormData((prev) => ({ ...prev, [column]: value }));
    setHasChanges(true);
  };

  const handleSave = async () => {
    setSaving(true);
    try {
      // Convert form data back to proper types
      const saveData: Record<string, any> = {};
      columns.forEach((col) => {
        const value = formData[col.name];
        if (value === '' && col.is_nullable) {
          saveData[col.name] = null;
        } else if (col.type.includes('json')) {
          try {
            saveData[col.name] = JSON.parse(value);
          } catch {
            saveData[col.name] = value;
          }
        } else if (col.type === 'boolean') {
          saveData[col.name] = value === 'true' || value === true;
        } else if (
          col.type.includes('int') ||
          col.type.includes('numeric') ||
          col.type.includes('decimal')
        ) {
          saveData[col.name] = value === '' ? null : Number(value);
        } else {
          saveData[col.name] = value;
        }
      });

      await onSave(saveData);
      setHasChanges(false);
    } finally {
      setSaving(false);
    }
  };

  const formatDisplayValue = (value: any, type: string): string => {
    if (value === null || value === undefined) return 'NULL';
    if (typeof value === 'boolean') return value ? 'true' : 'false';
    if (typeof value === 'object') return JSON.stringify(value, null, 2);
    if (type.includes('timestamp')) {
      try {
        const date = new Date(value);
        return date.toLocaleString();
      } catch {
        return String(value);
      }
    }
    return String(value);
  };

  const renderField = (col: Column) => {
    const value = formData[col.name] ?? '';

    // Boolean fields
    if (col.type === 'boolean') {
      return (
        <Select
          size="sm"
          value={value}
          onChange={(val) => handleChange(col.name, val)}
          data={[
            { value: '', label: '(null)' },
            { value: 'true', label: 'true' },
            { value: 'false', label: 'false' },
          ]}
          disabled={readOnly || col.is_primary_key}
          styles={{
            input: {
              backgroundColor: 'var(--lb-bg-primary)',
              borderColor: 'var(--lb-border-default)',
            },
          }}
        />
      );
    }

    // JSON fields
    if (col.type.includes('json')) {
      return (
        <Textarea
          size="sm"
          value={value}
          onChange={(e) => handleChange(col.name, e.target.value)}
          minRows={3}
          maxRows={10}
          autosize
          disabled={readOnly}
          styles={{
            input: {
              fontFamily: 'monospace',
              fontSize: '12px',
              backgroundColor: 'var(--lb-bg-primary)',
              borderColor: 'var(--lb-border-default)',
            },
          }}
        />
      );
    }

    // Long text fields
    if (col.type === 'text' && String(value).length > 100) {
      return (
        <Textarea
          size="sm"
          value={value}
          onChange={(e) => handleChange(col.name, e.target.value)}
          minRows={2}
          maxRows={6}
          autosize
          disabled={readOnly}
          styles={{
            input: {
              backgroundColor: 'var(--lb-bg-primary)',
              borderColor: 'var(--lb-border-default)',
            },
          }}
        />
      );
    }

    // Default text input
    return (
      <TextInput
        size="sm"
        value={value}
        onChange={(e) => handleChange(col.name, e.target.value)}
        disabled={readOnly || col.is_primary_key}
        styles={{
          input: {
            fontFamily: col.type.includes('uuid') || col.type.includes('int') ? 'monospace' : undefined,
            backgroundColor: 'var(--lb-bg-primary)',
            borderColor: 'var(--lb-border-default)',
          },
        }}
      />
    );
  };

  return (
    <Drawer
      opened={opened}
      onClose={onClose}
      title={
        <Group gap="xs">
          <Text fw={600}>Row Details</Text>
          {hasChanges && (
            <Badge size="xs" color="yellow">
              Unsaved changes
            </Badge>
          )}
        </Group>
      }
      position="right"
      size="lg"
      padding="md"
      styles={{
        header: {
          borderBottom: '1px solid var(--lb-border-default)',
          paddingBottom: 12,
        },
        body: {
          padding: 0,
        },
      }}
    >
      <ScrollArea style={{ height: 'calc(100vh - 140px)' }} p="md">
        <Stack gap="md">
          {columns.map((col) => (
            <Box key={col.name}>
              <Group gap="xs" mb={4} justify="space-between">
                <Group gap="xs">
                  {col.is_primary_key && (
                    <IconKey size={14} color="var(--lb-brand)" />
                  )}
                  <Text size="sm" fw={500}>
                    {col.name}
                  </Text>
                  <Text size="xs" style={{ color: 'var(--lb-text-secondary)' }}>
                    {col.type}
                  </Text>
                </Group>
                <Group gap={4}>
                  {col.is_primary_key && (
                    <Badge size="xs" color="blue">
                      PK
                    </Badge>
                  )}
                  {!col.is_nullable && (
                    <Badge size="xs" color="red" variant="light">
                      Required
                    </Badge>
                  )}
                  {col.is_unique && (
                    <Badge size="xs" color="violet" variant="light">
                      Unique
                    </Badge>
                  )}
                  <CopyButton value={formatDisplayValue(row?.[col.name], col.type)}>
                    {({ copied, copy }) => (
                      <Tooltip label={copied ? 'Copied!' : 'Copy value'}>
                        <ActionIcon size="xs" variant="subtle" onClick={copy}>
                          {copied ? <IconCheck size={12} /> : <IconCopy size={12} />}
                        </ActionIcon>
                      </Tooltip>
                    )}
                  </CopyButton>
                </Group>
              </Group>
              {renderField(col)}
              {col.default_value && (
                <Text size="xs" style={{ color: 'var(--lb-text-secondary)' }} mt={2}>
                  Default: {col.default_value}
                </Text>
              )}
            </Box>
          ))}
        </Stack>
      </ScrollArea>

      {!readOnly && (
        <>
          <Divider />
          <Group justify="flex-end" p="md">
            <Button variant="outline" onClick={onClose}>
              Cancel
            </Button>
            <Button
              onClick={handleSave}
              loading={saving}
              disabled={!hasChanges}
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
              Save changes
            </Button>
          </Group>
        </>
      )}
    </Drawer>
  );
}
