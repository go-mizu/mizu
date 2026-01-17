import { useState, useCallback, useEffect } from 'react';
import { storageApi } from '../../../api';
import type { StorageObject } from '../../../types';

export interface ColumnState {
  path: string;
  items: StorageObject[];
  selectedItem: string | null;
  loading: boolean;
  error: string | null;
}

export interface UseMillerNavigationOptions {
  bucketId: string | null;
  onFileSelect?: (file: StorageObject | null) => void;
}

export interface UseMillerNavigationReturn {
  columns: ColumnState[];
  selectedFile: StorageObject | null;
  loading: boolean;
  navigateToFolder: (folderPath: string, columnIndex: number) => void;
  selectItem: (item: StorageObject, columnIndex: number) => void;
  navigateBack: (columnIndex: number) => void;
  refreshColumn: (columnIndex: number) => void;
  refreshAll: () => void;
  clearSelection: () => void;
  currentPath: string;
}

export function useMillerNavigation({
  bucketId,
  onFileSelect,
}: UseMillerNavigationOptions): UseMillerNavigationReturn {
  const [columns, setColumns] = useState<ColumnState[]>([]);
  const [selectedFile, setSelectedFile] = useState<StorageObject | null>(null);
  const [loading, setLoading] = useState(false);

  const isFolder = useCallback((obj: StorageObject) => {
    return obj.name.endsWith('/') || !obj.content_type;
  }, []);

  const fetchColumnItems = useCallback(async (path: string): Promise<StorageObject[]> => {
    if (!bucketId) return [];

    try {
      const items = await storageApi.listObjects(bucketId, {
        prefix: path,
        limit: 100,
      });
      return items;
    } catch (error) {
      console.error('Failed to fetch column items:', error);
      return [];
    }
  }, [bucketId]);

  // Initialize root column when bucket changes
  useEffect(() => {
    const initRootColumn = async () => {
      if (!bucketId) {
        setColumns([]);
        setSelectedFile(null);
        return;
      }

      setLoading(true);
      try {
        const items = await fetchColumnItems('');
        setColumns([{
          path: '',
          items,
          selectedItem: null,
          loading: false,
          error: null,
        }]);
        setSelectedFile(null);
      } catch (error: any) {
        setColumns([{
          path: '',
          items: [],
          selectedItem: null,
          loading: false,
          error: error.message || 'Failed to load items',
        }]);
      } finally {
        setLoading(false);
      }
    };

    initRootColumn();
  }, [bucketId, fetchColumnItems]);

  // Notify parent when file selection changes
  useEffect(() => {
    onFileSelect?.(selectedFile);
  }, [selectedFile, onFileSelect]);

  const navigateToFolder = useCallback(async (folderPath: string, columnIndex: number) => {
    if (!bucketId) return;

    // Remove all columns after the clicked one
    setColumns(prev => {
      const newColumns = prev.slice(0, columnIndex + 1);
      // Update selected item in the clicked column
      if (newColumns[columnIndex]) {
        newColumns[columnIndex] = {
          ...newColumns[columnIndex],
          selectedItem: folderPath,
        };
      }
      // Add new loading column
      newColumns.push({
        path: folderPath,
        items: [],
        selectedItem: null,
        loading: true,
        error: null,
      });
      return newColumns;
    });

    // Clear file selection when navigating
    setSelectedFile(null);

    // Fetch items for the new column
    try {
      const items = await fetchColumnItems(folderPath);
      setColumns(prev => {
        const lastIndex = prev.length - 1;
        if (prev[lastIndex]?.path === folderPath) {
          const newColumns = [...prev];
          newColumns[lastIndex] = {
            ...newColumns[lastIndex],
            items,
            loading: false,
          };
          return newColumns;
        }
        return prev;
      });
    } catch (error: any) {
      setColumns(prev => {
        const lastIndex = prev.length - 1;
        if (prev[lastIndex]?.path === folderPath) {
          const newColumns = [...prev];
          newColumns[lastIndex] = {
            ...newColumns[lastIndex],
            loading: false,
            error: error.message || 'Failed to load items',
          };
          return newColumns;
        }
        return prev;
      });
    }
  }, [bucketId, fetchColumnItems]);

  const selectItem = useCallback((item: StorageObject, columnIndex: number) => {
    if (isFolder(item)) {
      // Navigate to folder - path should be the item name without trailing slash
      const folderPath = item.name.replace(/\/$/, '');
      navigateToFolder(folderPath, columnIndex);
    } else {
      // Select file - remove columns after current and show preview
      setColumns(prev => {
        const newColumns = prev.slice(0, columnIndex + 1);
        if (newColumns[columnIndex]) {
          newColumns[columnIndex] = {
            ...newColumns[columnIndex],
            selectedItem: item.name,
          };
        }
        return newColumns;
      });
      setSelectedFile(item);
    }
  }, [isFolder, navigateToFolder]);

  const navigateBack = useCallback((columnIndex: number) => {
    if (columnIndex === 0) {
      // Can't go back from root
      return;
    }

    // Remove this column and all after it
    setColumns(prev => {
      const newColumns = prev.slice(0, columnIndex);
      // Clear selection in the parent column
      if (newColumns.length > 0) {
        const lastIndex = newColumns.length - 1;
        newColumns[lastIndex] = {
          ...newColumns[lastIndex],
          selectedItem: null,
        };
      }
      return newColumns;
    });
    setSelectedFile(null);
  }, []);

  const refreshColumn = useCallback(async (columnIndex: number) => {
    if (!bucketId || !columns[columnIndex]) return;

    const column = columns[columnIndex];

    setColumns(prev => {
      const newColumns = [...prev];
      newColumns[columnIndex] = {
        ...newColumns[columnIndex],
        loading: true,
        error: null,
      };
      return newColumns;
    });

    try {
      const items = await fetchColumnItems(column.path);
      setColumns(prev => {
        const newColumns = [...prev];
        if (newColumns[columnIndex]) {
          newColumns[columnIndex] = {
            ...newColumns[columnIndex],
            items,
            loading: false,
          };
        }
        return newColumns;
      });
    } catch (error: any) {
      setColumns(prev => {
        const newColumns = [...prev];
        if (newColumns[columnIndex]) {
          newColumns[columnIndex] = {
            ...newColumns[columnIndex],
            loading: false,
            error: error.message || 'Failed to refresh',
          };
        }
        return newColumns;
      });
    }
  }, [bucketId, columns, fetchColumnItems]);

  const refreshAll = useCallback(() => {
    columns.forEach((_, index) => {
      refreshColumn(index);
    });
  }, [columns, refreshColumn]);

  const clearSelection = useCallback(() => {
    setSelectedFile(null);
    setColumns(prev => prev.map(col => ({
      ...col,
      selectedItem: null,
    })));
  }, []);

  // Compute current path from columns
  const currentPath = columns.length > 1 ? columns[columns.length - 1].path : '';

  return {
    columns,
    selectedFile,
    loading,
    navigateToFolder,
    selectItem,
    navigateBack,
    refreshColumn,
    refreshAll,
    clearSelection,
    currentPath,
  };
}
