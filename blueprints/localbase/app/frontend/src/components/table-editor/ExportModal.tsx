import { useState } from 'react';
import {
  Modal,
  Stack,
  Text,
  SegmentedControl,
  Group,
  Button,
  Checkbox,
  Badge,
} from '@mantine/core';
import { IconDownload, IconFileSpreadsheet, IconFileCode, IconJson } from '@tabler/icons-react';
import { databaseApi } from '../../api';

interface ExportModalProps {
  opened: boolean;
  onClose: () => void;
  schema: string;
  table: string;
  totalRows: number;
  selectedRows: number;
  filters?: Record<string, string>;
}

export function ExportModal({
  opened,
  onClose,
  schema,
  table,
  totalRows,
  selectedRows,
  filters,
}: ExportModalProps) {
  const [format, setFormat] = useState<'json' | 'csv' | 'sql'>('csv');
  const [exportSelected, setExportSelected] = useState(false);

  const handleExport = () => {
    const exportFilters = exportSelected && selectedRows > 0 ? filters : undefined;
    const url = databaseApi.exportTableData(schema, table, format, exportFilters);

    // Get the service key for auth
    const serviceKey = localStorage.getItem('supabase_service_key') || '';

    // Create a form to submit with auth headers (fetch download workaround)
    const link = document.createElement('a');
    link.href = `${url}&apikey=${encodeURIComponent(serviceKey)}`;
    link.download = `${table}_export.${format}`;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);

    onClose();
  };

  const getFormatIcon = () => {
    switch (format) {
      case 'csv':
        return <IconFileSpreadsheet size={20} />;
      case 'sql':
        return <IconFileCode size={20} />;
      default:
        return <IconJson size={20} />;
    }
  };

  const getFormatDescription = () => {
    switch (format) {
      case 'csv':
        return 'Export as CSV file, compatible with Excel and spreadsheet applications.';
      case 'sql':
        return 'Export as SQL INSERT statements that can be used to recreate the data.';
      default:
        return 'Export as JSON array, ideal for programmatic use and API testing.';
    }
  };

  return (
    <Modal
      opened={opened}
      onClose={onClose}
      title={
        <Group gap="xs">
          <IconDownload size={20} />
          <Text fw={600}>Export Data</Text>
        </Group>
      }
      size="md"
    >
      <Stack gap="lg">
        <Stack gap="xs">
          <Text size="sm" fw={500}>
            Export format
          </Text>
          <SegmentedControl
            value={format}
            onChange={(value) => setFormat(value as 'json' | 'csv' | 'sql')}
            data={[
              { value: 'csv', label: 'CSV' },
              { value: 'json', label: 'JSON' },
              { value: 'sql', label: 'SQL' },
            ]}
            fullWidth
            styles={{
              root: {
                backgroundColor: 'var(--lb-bg-secondary)',
                border: '1px solid var(--lb-border-default)',
              },
              indicator: {
                backgroundColor: 'var(--lb-bg-primary)',
                boxShadow: '0 1px 3px rgba(0, 0, 0, 0.1)',
              },
            }}
          />
          <Text size="xs" style={{ color: 'var(--lb-text-secondary)' }}>
            {getFormatDescription()}
          </Text>
        </Stack>

        <Stack gap="xs">
          <Text size="sm" fw={500}>
            Rows to export
          </Text>
          <Group>
            <Badge variant="light" color="blue">
              {totalRows} total rows
            </Badge>
            {selectedRows > 0 && (
              <Badge variant="light" color="green">
                {selectedRows} selected
              </Badge>
            )}
          </Group>
          {selectedRows > 0 && (
            <Checkbox
              label={`Export only selected rows (${selectedRows})`}
              checked={exportSelected}
              onChange={(e) => setExportSelected(e.currentTarget.checked)}
            />
          )}
        </Stack>

        <Group justify="flex-end" mt="md">
          <Button variant="outline" onClick={onClose}>
            Cancel
          </Button>
          <Button
            leftSection={getFormatIcon()}
            onClick={handleExport}
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
            Export {format.toUpperCase()}
          </Button>
        </Group>
      </Stack>
    </Modal>
  );
}
