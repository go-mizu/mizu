import { useState, useMemo } from 'react';
import {
  Table,
  Checkbox,
  Text,
  Center,
  Loader,
  Group,
  ActionIcon,
  Menu,
  ScrollArea,
} from '@mantine/core';
import {
  IconFolder,
  IconFile,
  IconPhoto,
  IconVideo,
  IconMusic,
  IconFileTypePdf,
  IconBrandJavascript,
  IconBrandPython,
  IconMarkdown,
  IconBraces,
  IconFileCode,
  IconFileText,
  IconDotsVertical,
  IconDownload,
  IconLink,
  IconEdit,
  IconTrash,
  IconArrowUp,
  IconArrowDown,
} from '@tabler/icons-react';
import type { StorageObject } from '../../../types';

function getFileIcon(file: StorageObject) {
  const contentType = file.content_type || '';
  const ext = file.name.split('.').pop()?.toLowerCase();

  if (contentType.startsWith('image/')) return IconPhoto;
  if (contentType.startsWith('video/')) return IconVideo;
  if (contentType.startsWith('audio/')) return IconMusic;
  if (contentType === 'application/pdf') return IconFileTypePdf;
  if (contentType === 'application/json') return IconBraces;

  switch (ext) {
    case 'js':
    case 'jsx':
    case 'ts':
    case 'tsx':
      return IconBrandJavascript;
    case 'py':
      return IconBrandPython;
    case 'md':
    case 'mdx':
      return IconMarkdown;
    case 'json':
      return IconBraces;
    case 'go':
    case 'java':
    case 'c':
    case 'cpp':
      return IconFileCode;
    case 'txt':
    case 'log':
      return IconFileText;
    default:
      return IconFile;
  }
}

function isFolder(obj: StorageObject): boolean {
  return obj.name.endsWith('/') || !obj.content_type;
}

function formatFileSize(bytes: number): string {
  if (bytes === 0) return '-';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

function formatDate(dateString: string): string {
  const date = new Date(dateString);
  return date.toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
  });
}

type SortColumn = 'name' | 'size' | 'type' | 'updated_at';
type SortOrder = 'asc' | 'desc';

interface FileListViewProps {
  items: StorageObject[];
  loading: boolean;
  error: string | null;
  selectedItems: Set<string>;
  onItemSelect: (item: StorageObject) => void;
  onItemCheck: (item: StorageObject, checked: boolean) => void;
  onSelectAll: (checked: boolean) => void;
  onDownload: (item: StorageObject) => void;
  onCopyUrl: (item: StorageObject) => void;
  onRename: (item: StorageObject) => void;
  onDelete: (item: StorageObject) => void;
}

export function FileListView({
  items,
  loading,
  error,
  selectedItems,
  onItemSelect,
  onItemCheck,
  onSelectAll,
  onDownload,
  onCopyUrl,
  onRename,
  onDelete,
}: FileListViewProps) {
  const [sortColumn, setSortColumn] = useState<SortColumn>('name');
  const [sortOrder, setSortOrder] = useState<SortOrder>('asc');

  const handleSort = (column: SortColumn) => {
    if (sortColumn === column) {
      setSortOrder((prev) => (prev === 'asc' ? 'desc' : 'asc'));
    } else {
      setSortColumn(column);
      setSortOrder('asc');
    }
  };

  const sortedItems = useMemo(() => {
    return [...items].sort((a, b) => {
      // Folders always come first
      const aIsFolder = isFolder(a);
      const bIsFolder = isFolder(b);
      if (aIsFolder && !bIsFolder) return -1;
      if (!aIsFolder && bIsFolder) return 1;

      let comparison = 0;
      switch (sortColumn) {
        case 'name':
          comparison = a.name.localeCompare(b.name);
          break;
        case 'size':
          comparison = a.size - b.size;
          break;
        case 'type':
          comparison = (a.content_type || '').localeCompare(b.content_type || '');
          break;
        case 'updated_at':
          comparison = new Date(a.updated_at).getTime() - new Date(b.updated_at).getTime();
          break;
      }
      return sortOrder === 'asc' ? comparison : -comparison;
    });
  }, [items, sortColumn, sortOrder]);

  const allSelected = items.length > 0 && selectedItems.size === items.length;
  const someSelected = selectedItems.size > 0 && selectedItems.size < items.length;

  const SortIcon = sortOrder === 'asc' ? IconArrowUp : IconArrowDown;

  if (loading) {
    return (
      <Center style={{ flex: 1 }} py="xl">
        <Loader size="md" />
      </Center>
    );
  }

  if (error) {
    return (
      <Center style={{ flex: 1 }} py="xl">
        <Text c="red">{error}</Text>
      </Center>
    );
  }

  if (items.length === 0) {
    return (
      <Center style={{ flex: 1 }} py="xl">
        <Text c="dimmed">No files in this folder</Text>
      </Center>
    );
  }

  return (
    <ScrollArea style={{ flex: 1 }}>
      <Table striped highlightOnHover>
        <Table.Thead
          style={{
            backgroundColor: 'var(--supabase-bg-surface)',
            position: 'sticky',
            top: 0,
            zIndex: 1,
          }}
        >
          <Table.Tr>
            <Table.Th style={{ width: 40 }}>
              <Checkbox
                checked={allSelected}
                indeterminate={someSelected}
                onChange={(e) => onSelectAll(e.currentTarget.checked)}
                aria-label="Select all"
              />
            </Table.Th>
            <Table.Th
              style={{ cursor: 'pointer' }}
              onClick={() => handleSort('name')}
            >
              <Group gap="xs">
                Name
                {sortColumn === 'name' && <SortIcon size={14} />}
              </Group>
            </Table.Th>
            <Table.Th
              style={{ cursor: 'pointer', width: 100 }}
              onClick={() => handleSort('size')}
            >
              <Group gap="xs">
                Size
                {sortColumn === 'size' && <SortIcon size={14} />}
              </Group>
            </Table.Th>
            <Table.Th
              style={{ cursor: 'pointer', width: 150 }}
              onClick={() => handleSort('type')}
            >
              <Group gap="xs">
                Type
                {sortColumn === 'type' && <SortIcon size={14} />}
              </Group>
            </Table.Th>
            <Table.Th
              style={{ cursor: 'pointer', width: 130 }}
              onClick={() => handleSort('updated_at')}
            >
              <Group gap="xs">
                Modified
                {sortColumn === 'updated_at' && <SortIcon size={14} />}
              </Group>
            </Table.Th>
            <Table.Th style={{ width: 50 }} />
          </Table.Tr>
        </Table.Thead>
        <Table.Tbody>
          {sortedItems.map((item) => {
            const isFolderItem = isFolder(item);
            const ItemIcon = isFolderItem ? IconFolder : getFileIcon(item);
            const displayName = item.name.split('/').filter(Boolean).pop() || item.name;
            const isSelected = selectedItems.has(item.id || item.name);

            return (
              <Table.Tr
                key={item.id || item.name}
                style={{
                  backgroundColor: isSelected ? 'var(--supabase-brand-light)' : undefined,
                  cursor: 'pointer',
                }}
                onClick={() => onItemSelect(item)}
              >
                <Table.Td onClick={(e) => e.stopPropagation()}>
                  <Checkbox
                    checked={isSelected}
                    onChange={(e) => onItemCheck(item, e.currentTarget.checked)}
                    aria-label={`Select ${displayName}`}
                  />
                </Table.Td>
                <Table.Td>
                  <Group gap="xs">
                    <ItemIcon
                      size={18}
                      color={isFolderItem ? 'var(--supabase-brand)' : 'var(--supabase-text-muted)'}
                      stroke={1.5}
                    />
                    <Text size="sm">{displayName}</Text>
                  </Group>
                </Table.Td>
                <Table.Td>
                  <Text size="sm" c="dimmed">
                    {isFolderItem ? '-' : formatFileSize(item.size)}
                  </Text>
                </Table.Td>
                <Table.Td>
                  <Text size="sm" c="dimmed" truncate style={{ maxWidth: 130 }}>
                    {isFolderItem ? 'folder' : (item.content_type || 'Unknown')}
                  </Text>
                </Table.Td>
                <Table.Td>
                  <Text size="sm" c="dimmed">
                    {item.updated_at ? formatDate(item.updated_at) : '-'}
                  </Text>
                </Table.Td>
                <Table.Td onClick={(e) => e.stopPropagation()}>
                  <Menu position="bottom-end">
                    <Menu.Target>
                      <ActionIcon variant="subtle" size="sm">
                        <IconDotsVertical size={16} />
                      </ActionIcon>
                    </Menu.Target>
                    <Menu.Dropdown>
                      {!isFolderItem && (
                        <>
                          <Menu.Item
                            leftSection={<IconDownload size={14} />}
                            onClick={() => onDownload(item)}
                          >
                            Download
                          </Menu.Item>
                          <Menu.Item
                            leftSection={<IconLink size={14} />}
                            onClick={() => onCopyUrl(item)}
                          >
                            Copy URL
                          </Menu.Item>
                          <Menu.Divider />
                        </>
                      )}
                      <Menu.Item
                        leftSection={<IconEdit size={14} />}
                        onClick={() => onRename(item)}
                      >
                        Rename
                      </Menu.Item>
                      <Menu.Item
                        leftSection={<IconTrash size={14} />}
                        color="red"
                        onClick={() => onDelete(item)}
                      >
                        Delete
                      </Menu.Item>
                    </Menu.Dropdown>
                  </Menu>
                </Table.Td>
              </Table.Tr>
            );
          })}
        </Table.Tbody>
      </Table>
    </ScrollArea>
  );
}
