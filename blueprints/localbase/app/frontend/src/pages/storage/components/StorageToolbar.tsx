import { useState } from 'react';
import {
  Box,
  Button,
  Group,
  ActionIcon,
  Menu,
  Tooltip,
  Breadcrumbs,
  Anchor,
  TextInput,
  Modal,
} from '@mantine/core';
import {
  IconRefresh,
  IconUpload,
  IconFolderPlus,
  IconDotsVertical,
  IconTrash,
  IconSearch,
  IconLayoutColumns,
  IconList,
  IconSettings,
} from '@tabler/icons-react';

export type ViewMode = 'columns' | 'list';

interface StorageToolbarProps {
  bucketName: string;
  currentPath: string;
  viewMode: ViewMode;
  onRefresh: () => void;
  onUpload: () => void;
  onCreateFolder: () => void;
  onDeleteBucket: () => void;
  onViewModeChange: (mode: ViewMode) => void;
  onSearch: (query: string) => void;
  onNavigateToPath: (path: string) => void;
  uploadLoading?: boolean;
}

export function StorageToolbar({
  bucketName,
  currentPath,
  viewMode,
  onRefresh,
  onUpload,
  onCreateFolder,
  onDeleteBucket,
  onViewModeChange,
  onSearch,
  onNavigateToPath,
  uploadLoading,
}: StorageToolbarProps) {
  const [searchOpened, setSearchOpened] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');

  const breadcrumbParts = currentPath ? currentPath.split('/').filter(Boolean) : [];

  const handleSearch = () => {
    if (searchQuery.trim()) {
      onSearch(searchQuery.trim());
    }
    setSearchOpened(false);
    setSearchQuery('');
  };

  return (
    <Box
      px="sm"
      py="xs"
      style={{
        borderBottom: '1px solid var(--supabase-border)',
        backgroundColor: 'var(--supabase-bg)',
        minHeight: 48,
        display: 'flex',
        alignItems: 'center',
      }}
    >
      <Group justify="space-between" style={{ flex: 1 }}>
        {/* Breadcrumb Navigation */}
        <Breadcrumbs>
          <Anchor
            size="sm"
            onClick={() => onNavigateToPath('')}
            style={{ cursor: 'pointer' }}
          >
            {bucketName}
          </Anchor>
          {breadcrumbParts.map((part, index) => (
            <Anchor
              key={index}
              size="sm"
              onClick={() => {
                const path = breadcrumbParts.slice(0, index + 1).join('/');
                onNavigateToPath(path);
              }}
              style={{ cursor: 'pointer' }}
            >
              {part}
            </Anchor>
          ))}
        </Breadcrumbs>

        {/* Action Buttons */}
        <Group gap="sm">
          <Tooltip label="Reload">
            <ActionIcon variant="subtle" onClick={onRefresh}>
              <IconRefresh size={18} />
            </ActionIcon>
          </Tooltip>

          <Menu position="bottom-end" width={160}>
            <Menu.Target>
              <Tooltip label="View">
                <ActionIcon variant="subtle">
                  {viewMode === 'columns' ? (
                    <IconLayoutColumns size={18} />
                  ) : (
                    <IconList size={18} />
                  )}
                </ActionIcon>
              </Tooltip>
            </Menu.Target>
            <Menu.Dropdown>
              <Menu.Item
                leftSection={<IconLayoutColumns size={14} />}
                onClick={() => onViewModeChange('columns')}
                style={{
                  backgroundColor: viewMode === 'columns' ? 'var(--supabase-brand-light)' : undefined,
                }}
              >
                Column view
              </Menu.Item>
              <Menu.Item
                leftSection={<IconList size={14} />}
                onClick={() => onViewModeChange('list')}
                style={{
                  backgroundColor: viewMode === 'list' ? 'var(--supabase-brand-light)' : undefined,
                }}
              >
                List view
              </Menu.Item>
            </Menu.Dropdown>
          </Menu>

          <Button
            variant="outline"
            leftSection={<IconUpload size={16} />}
            onClick={onUpload}
            loading={uploadLoading}
            size="sm"
          >
            Upload files
          </Button>

          <Button
            variant="outline"
            leftSection={<IconFolderPlus size={16} />}
            onClick={onCreateFolder}
            size="sm"
          >
            Create folder
          </Button>

          <Tooltip label="Search">
            <ActionIcon variant="subtle" onClick={() => setSearchOpened(true)}>
              <IconSearch size={18} />
            </ActionIcon>
          </Tooltip>

          <Menu position="bottom-end">
            <Menu.Target>
              <ActionIcon variant="subtle">
                <IconDotsVertical size={18} />
              </ActionIcon>
            </Menu.Target>
            <Menu.Dropdown>
              <Menu.Item
                leftSection={<IconSettings size={14} />}
                disabled
              >
                Bucket settings
              </Menu.Item>
              <Menu.Divider />
              <Menu.Item
                color="red"
                leftSection={<IconTrash size={14} />}
                onClick={onDeleteBucket}
              >
                Delete bucket
              </Menu.Item>
            </Menu.Dropdown>
          </Menu>
        </Group>
      </Group>

      {/* Search Modal */}
      <Modal
        opened={searchOpened}
        onClose={() => {
          setSearchOpened(false);
          setSearchQuery('');
        }}
        title="Search files"
        size="md"
      >
        <TextInput
          placeholder="Search by file name..."
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter') {
              handleSearch();
            }
          }}
          autoFocus
        />
        <Group justify="flex-end" mt="md">
          <Button variant="outline" onClick={() => setSearchOpened(false)}>
            Cancel
          </Button>
          <Button onClick={handleSearch}>
            Search
          </Button>
        </Group>
      </Modal>
    </Box>
  );
}
