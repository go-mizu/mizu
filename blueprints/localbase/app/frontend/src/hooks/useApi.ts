import { useState, useCallback } from 'react';
import { notifications } from '@mantine/notifications';
import { ApiClientError } from '../api/client';

interface UseApiState<T> {
  data: T | null;
  loading: boolean;
  error: string | null;
}

interface UseApiResult<T, P extends any[]> extends UseApiState<T> {
  execute: (...params: P) => Promise<T | null>;
  reset: () => void;
  setData: (data: T | null) => void;
}

export function useApi<T, P extends any[] = []>(
  apiFunction: (...params: P) => Promise<T>,
  options?: {
    onSuccess?: (data: T) => void;
    onError?: (error: string) => void;
    showErrorNotification?: boolean;
    showSuccessNotification?: string;
  }
): UseApiResult<T, P> {
  const [state, setState] = useState<UseApiState<T>>({
    data: null,
    loading: false,
    error: null,
  });

  const execute = useCallback(
    async (...params: P): Promise<T | null> => {
      setState((prev) => ({ ...prev, loading: true, error: null }));

      try {
        const data = await apiFunction(...params);
        setState({ data, loading: false, error: null });

        if (options?.onSuccess) {
          options.onSuccess(data);
        }

        if (options?.showSuccessNotification) {
          notifications.show({
            title: 'Success',
            message: options.showSuccessNotification,
            color: 'green',
          });
        }

        return data;
      } catch (err) {
        const errorMessage =
          err instanceof ApiClientError
            ? err.message
            : err instanceof Error
              ? err.message
              : 'An unexpected error occurred';

        setState((prev) => ({ ...prev, loading: false, error: errorMessage }));

        if (options?.onError) {
          options.onError(errorMessage);
        }

        if (options?.showErrorNotification !== false) {
          notifications.show({
            title: 'Error',
            message: errorMessage,
            color: 'red',
          });
        }

        return null;
      }
    },
    [apiFunction, options]
  );

  const reset = useCallback(() => {
    setState({ data: null, loading: false, error: null });
  }, []);

  const setData = useCallback((data: T | null) => {
    setState((prev) => ({ ...prev, data }));
  }, []);

  return {
    ...state,
    execute,
    reset,
    setData,
  };
}

// Hook for paginated data
interface UsePaginatedApiResult<T> {
  data: T[];
  loading: boolean;
  error: string | null;
  page: number;
  totalPages: number;
  total: number;
  setPage: (page: number) => void;
  refresh: () => Promise<void>;
}

export function usePaginatedApi<T>(
  apiFunction: (page: number, perPage: number) => Promise<{ data: T[]; total: number }>,
  perPage = 20
): UsePaginatedApiResult<T> {
  const [data, setData] = useState<T[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [page, setPageState] = useState(1);
  const [total, setTotal] = useState(0);

  const fetchData = useCallback(
    async (targetPage: number) => {
      setLoading(true);
      setError(null);

      try {
        const result = await apiFunction(targetPage, perPage);
        setData(result.data);
        setTotal(result.total);
      } catch (err) {
        const errorMessage =
          err instanceof ApiClientError
            ? err.message
            : err instanceof Error
              ? err.message
              : 'An unexpected error occurred';
        setError(errorMessage);
        notifications.show({
          title: 'Error',
          message: errorMessage,
          color: 'red',
        });
      } finally {
        setLoading(false);
      }
    },
    [apiFunction, perPage]
  );

  const setPage = useCallback(
    (newPage: number) => {
      setPageState(newPage);
      fetchData(newPage);
    },
    [fetchData]
  );

  const refresh = useCallback(async () => {
    await fetchData(page);
  }, [fetchData, page]);

  return {
    data,
    loading,
    error,
    page,
    totalPages: Math.ceil(total / perPage),
    total,
    setPage,
    refresh,
  };
}
