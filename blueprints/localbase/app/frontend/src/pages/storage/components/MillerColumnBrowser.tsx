import { Box, ScrollArea } from '@mantine/core';
import { MillerColumn } from './MillerColumn';
import type { ColumnState } from '../hooks/useMillerNavigation';
import type { StorageObject } from '../../../types';

interface MillerColumnBrowserProps {
  columns: ColumnState[];
  onItemSelect: (item: StorageObject, columnIndex: number) => void;
  onBack: (columnIndex: number) => void;
}

export function MillerColumnBrowser({
  columns,
  onItemSelect,
  onBack,
}: MillerColumnBrowserProps) {
  if (columns.length === 0) {
    return null;
  }

  return (
    <ScrollArea
      style={{
        flex: 1,
        backgroundColor: 'var(--supabase-bg)',
      }}
      scrollbarSize={8}
      type="auto"
      offsetScrollbars
    >
      <Box
        style={{
          display: 'flex',
          minHeight: '100%',
        }}
      >
        {columns.map((column, index) => (
          <MillerColumn
            key={`${column.path}-${index}`}
            path={column.path}
            items={column.items}
            selectedItem={column.selectedItem}
            loading={column.loading}
            error={column.error}
            columnIndex={index}
            showBackButton={index > 0}
            onItemSelect={onItemSelect}
            onBack={onBack}
          />
        ))}
      </Box>
    </ScrollArea>
  );
}
