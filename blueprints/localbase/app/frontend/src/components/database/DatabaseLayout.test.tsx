import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { MantineProvider } from '@mantine/core';
import { DatabaseLayout } from './DatabaseLayout';
import { databaseApi } from '../../api/database';

// Mock the database API
vi.mock('../../api/database', () => ({
  databaseApi: {
    getOverview: vi.fn(),
    listSchemas: vi.fn(),
  },
}));

const mockOverview = {
  schemas: [
    { name: 'public', table_count: 5, view_count: 2 },
    { name: 'auth', table_count: 3, view_count: 1 },
  ],
  total_tables: 8,
  total_views: 3,
  total_functions: 10,
  total_indexes: 15,
  total_policies: 5,
  database_size: '128 MB',
  connection_count: 3,
};

const mockSchemas = ['public', 'auth', 'storage'];

function renderWithProviders(ui: React.ReactElement, route = '/database/tables') {
  return render(
    <MantineProvider>
      <MemoryRouter initialEntries={[route]}>
        {ui}
      </MemoryRouter>
    </MantineProvider>
  );
}

describe('DatabaseLayout', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    (databaseApi.getOverview as ReturnType<typeof vi.fn>).mockResolvedValue(mockOverview);
    (databaseApi.listSchemas as ReturnType<typeof vi.fn>).mockResolvedValue(mockSchemas);
  });

  it('renders database sidebar', async () => {
    renderWithProviders(<DatabaseLayout />);

    await waitFor(() => {
      expect(screen.getByText('Database')).toBeInTheDocument();
    });
  });

  it('renders navigation items', async () => {
    renderWithProviders(<DatabaseLayout />);

    await waitFor(() => {
      expect(screen.getByText('Tables')).toBeInTheDocument();
      expect(screen.getByText('SQL Editor')).toBeInTheDocument();
      expect(screen.getByText('Schema Visualizer')).toBeInTheDocument();
      expect(screen.getByText('Policies')).toBeInTheDocument();
      expect(screen.getByText('Roles')).toBeInTheDocument();
      expect(screen.getByText('Indexes')).toBeInTheDocument();
      expect(screen.getByText('Views')).toBeInTheDocument();
      expect(screen.getByText('Triggers')).toBeInTheDocument();
      expect(screen.getByText('Functions')).toBeInTheDocument();
      expect(screen.getByText('Extensions')).toBeInTheDocument();
    });
  });

  it('displays database size from overview', async () => {
    renderWithProviders(<DatabaseLayout />);

    await waitFor(() => {
      expect(screen.getByText('128 MB')).toBeInTheDocument();
    });
  });

  it('displays connection count from overview', async () => {
    renderWithProviders(<DatabaseLayout />);

    await waitFor(() => {
      expect(screen.getByText('3')).toBeInTheDocument();
    });
  });

  it('displays table count badge', async () => {
    renderWithProviders(<DatabaseLayout />);

    await waitFor(() => {
      expect(screen.getByText('8')).toBeInTheDocument();
    });
  });

  it('highlights active route', async () => {
    renderWithProviders(<DatabaseLayout />, '/database/tables');

    await waitFor(() => {
      const tablesLink = screen.getByText('Tables').closest('a');
      expect(tablesLink).toHaveAttribute('data-active', 'true');
    });
  });

  it('handles API error gracefully', async () => {
    (databaseApi.getOverview as ReturnType<typeof vi.fn>).mockRejectedValue(new Error('API Error'));

    renderWithProviders(<DatabaseLayout />);

    // Should still render the sidebar even on error
    await waitFor(() => {
      expect(screen.getByText('Database')).toBeInTheDocument();
    });
  });
});
