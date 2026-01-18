import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { MantineProvider } from '@mantine/core';

// Mock Monaco Editor
vi.mock('@monaco-editor/react', () => ({
  default: ({ value, onChange, ...props }: any) => (
    <textarea
      data-testid="monaco-editor"
      value={value}
      onChange={(e) => onChange?.(e.target.value)}
      {...props}
    />
  ),
}));

// Mock Recharts
vi.mock('recharts', () => ({
  ResponsiveContainer: ({ children }: any) => <div data-testid="responsive-container">{children}</div>,
  AreaChart: ({ children }: any) => <div data-testid="area-chart">{children}</div>,
  Area: () => null,
  XAxis: () => null,
  YAxis: () => null,
  CartesianGrid: () => null,
  Tooltip: () => null,
}));

// Mock functions API
vi.mock('../../api/functions', () => ({
  functionsApi: {
    listFunctions: vi.fn(),
    getSource: vi.fn(),
    listTemplates: vi.fn(),
    listSecrets: vi.fn(),
    createFunction: vi.fn(),
    deleteFunction: vi.fn(),
    updateFunction: vi.fn(),
    deployFunction: vi.fn(),
    updateSource: vi.fn(),
    getLogs: vi.fn(),
    getMetrics: vi.fn(),
    listDeployments: vi.fn(),
    testFunction: vi.fn(),
    createSecret: vi.fn(),
    deleteSecret: vi.fn(),
    bulkUpdateSecrets: vi.fn(),
    downloadFunction: vi.fn(),
  },
}));

// Import after mocks
import { FunctionsPage } from './Functions';
import { functionsApi } from '../../api/functions';

// Mock crypto.randomUUID
Object.defineProperty(global, 'crypto', {
  value: {
    randomUUID: () => 'test-uuid-' + Math.random().toString(36).substr(2, 9),
  },
});

// Test data
const mockFunction = {
  id: 'func-1',
  name: 'test-function',
  slug: 'test-function',
  status: 'active',
  version: 1,
  verify_jwt: true,
  entrypoint: 'index.ts',
  created_at: '2025-01-15T10:00:00Z',
  updated_at: '2025-01-15T10:00:00Z',
};

const mockFunction2 = {
  id: 'func-2',
  name: 'another-function',
  slug: 'another-function',
  status: 'inactive',
  version: 2,
  verify_jwt: false,
  entrypoint: 'index.ts',
  created_at: '2025-01-14T10:00:00Z',
  updated_at: '2025-01-14T10:00:00Z',
};

const mockSourceCode = `import { serve } from "https://deno.land/std@0.168.0/http/server.ts"

serve(async (req) => {
  const { name } = await req.json()
  return new Response(JSON.stringify({ message: \`Hello \${name}!\` }))
})`;

const mockTemplates = [
  {
    id: 'hello-world',
    name: 'Hello World',
    description: 'Basic HTTP handler',
    icon: 'wave',
    category: 'starter',
  },
  {
    id: 'stripe-webhook',
    name: 'Stripe Webhook',
    description: 'Handle Stripe payment events',
    icon: 'credit-card',
    category: 'integration',
  },
];

const mockSecrets = [
  { id: 'sec-1', name: 'API_KEY', created_at: '2025-01-10T10:00:00Z' },
  { id: 'sec-2', name: 'DATABASE_URL', created_at: '2025-01-10T10:00:00Z' },
];

const mockLogs = [
  {
    id: 'log-1',
    function_id: 'func-1',
    timestamp: '2025-01-15T11:00:00Z',
    level: 'info',
    message: 'Function invoked successfully',
    status_code: 200,
    duration_ms: 45,
  },
];

const mockMetrics = {
  function_id: 'func-1',
  invocations: {
    total: 1500,
    success: 1450,
    error: 50,
    by_hour: [
      { hour: '2025-01-15T10:00:00Z', count: 100 },
      { hour: '2025-01-15T11:00:00Z', count: 150 },
    ],
  },
  latency: {
    avg: 45,
    p50: 40,
    p95: 120,
    p99: 250,
  },
};

const mockDeployments = [
  {
    id: 'dep-1',
    function_id: 'func-1',
    version: 1,
    status: 'deployed',
    deployed_at: '2025-01-15T10:00:00Z',
  },
];

function renderWithProviders(ui: React.ReactElement) {
  return render(
    <MantineProvider>
      <MemoryRouter>
        {ui}
      </MemoryRouter>
    </MantineProvider>
  );
}

// Cast functionsApi to mocked type
const mockFunctionsApi = functionsApi as {
  [K in keyof typeof functionsApi]: ReturnType<typeof vi.fn>;
};

describe('FunctionsPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockFunctionsApi.listFunctions.mockResolvedValue([mockFunction, mockFunction2]);
    mockFunctionsApi.getSource.mockResolvedValue({ source_code: mockSourceCode, is_draft: false });
    mockFunctionsApi.listTemplates.mockResolvedValue({ templates: mockTemplates });
    mockFunctionsApi.listSecrets.mockResolvedValue(mockSecrets);
    mockFunctionsApi.getLogs.mockResolvedValue({ logs: mockLogs });
    mockFunctionsApi.getMetrics.mockResolvedValue(mockMetrics);
    mockFunctionsApi.listDeployments.mockResolvedValue(mockDeployments);
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('Initial Rendering', () => {
    it('shows loading state initially', async () => {
      mockFunctionsApi.listFunctions.mockImplementation(() => new Promise(() => {}));
      renderWithProviders(<FunctionsPage />);

      expect(screen.getByText('Loading functions...')).toBeInTheDocument();
    });

    it('renders page title', async () => {
      renderWithProviders(<FunctionsPage />);

      await waitFor(() => {
        expect(screen.getByText('Edge Functions')).toBeInTheDocument();
      });
    });

    it('renders create function buttons', async () => {
      renderWithProviders(<FunctionsPage />);

      await waitFor(() => {
        // Use getAllBy since there might be multiple buttons
        const buttons = screen.getAllByRole('button', { name: /create function/i });
        expect(buttons.length).toBeGreaterThan(0);
      });
    });
  });

  describe('Function List', () => {
    it('shows empty state when no functions', async () => {
      mockFunctionsApi.listFunctions.mockResolvedValue([]);
      renderWithProviders(<FunctionsPage />);

      await waitFor(() => {
        expect(screen.getByText('No functions yet')).toBeInTheDocument();
      });
    });

    it('calls listFunctions API on mount', async () => {
      renderWithProviders(<FunctionsPage />);

      await waitFor(() => {
        expect(mockFunctionsApi.listFunctions).toHaveBeenCalled();
      });
    });
  });

  describe('Function Details', () => {
    it('displays function status badge', async () => {
      renderWithProviders(<FunctionsPage />);

      await waitFor(() => {
        expect(screen.getByText('active')).toBeInTheDocument();
      });
    });

    it('renders all tabs', async () => {
      renderWithProviders(<FunctionsPage />);

      await waitFor(() => {
        expect(screen.getByRole('tab', { name: /code/i })).toBeInTheDocument();
        expect(screen.getByRole('tab', { name: /logs/i })).toBeInTheDocument();
        expect(screen.getByRole('tab', { name: /metrics/i })).toBeInTheDocument();
        expect(screen.getByRole('tab', { name: /deployments/i })).toBeInTheDocument();
        expect(screen.getByRole('tab', { name: /settings/i })).toBeInTheDocument();
      });
    });

    it('shows deploy button', async () => {
      renderWithProviders(<FunctionsPage />);

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /deploy/i })).toBeInTheDocument();
      });
    });

    it('shows test button', async () => {
      renderWithProviders(<FunctionsPage />);

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /test/i })).toBeInTheDocument();
      });
    });
  });

  describe('Code Editor', () => {
    it('renders Monaco editor', async () => {
      renderWithProviders(<FunctionsPage />);

      await waitFor(() => {
        expect(screen.getByTestId('monaco-editor')).toBeInTheDocument();
      });
    });

    it('loads source code into editor', async () => {
      renderWithProviders(<FunctionsPage />);

      await waitFor(() => {
        const editor = screen.getByTestId('monaco-editor') as HTMLTextAreaElement;
        expect(editor.value).toContain('serve');
      });
    });

    it('shows unsaved changes badge when code is modified', async () => {
      const user = userEvent.setup();
      renderWithProviders(<FunctionsPage />);

      await waitFor(() => {
        expect(screen.getByTestId('monaco-editor')).toBeInTheDocument();
      });

      const editor = screen.getByTestId('monaco-editor');
      await user.clear(editor);
      await user.type(editor, '// modified code');

      await waitFor(() => {
        expect(screen.getByText('Unsaved changes')).toBeInTheDocument();
      });
    });
  });

  describe('Test Panel', () => {
    it('opens test panel when test button is clicked', async () => {
      const user = userEvent.setup();
      renderWithProviders(<FunctionsPage />);

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /test/i })).toBeInTheDocument();
      });

      await user.click(screen.getByRole('button', { name: /test/i }));

      await waitFor(() => {
        expect(screen.getByText('Test Function')).toBeInTheDocument();
      });
    });

    it('shows run function button in test panel', async () => {
      const user = userEvent.setup();
      renderWithProviders(<FunctionsPage />);

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /test/i })).toBeInTheDocument();
      });

      await user.click(screen.getByRole('button', { name: /test/i }));

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /run function/i })).toBeInTheDocument();
      });
    });

    it('executes test when run button is clicked', async () => {
      const user = userEvent.setup();
      mockFunctionsApi.testFunction.mockResolvedValue({
        status: 200,
        body: { message: 'Hello World!' },
        duration_ms: 50,
      });

      renderWithProviders(<FunctionsPage />);

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /test/i })).toBeInTheDocument();
      });

      await user.click(screen.getByRole('button', { name: /test/i }));

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /run function/i })).toBeInTheDocument();
      });

      await user.click(screen.getByRole('button', { name: /run function/i }));

      await waitFor(() => {
        expect(mockFunctionsApi.testFunction).toHaveBeenCalled();
      });
    });

    it('displays test response', async () => {
      const user = userEvent.setup();
      mockFunctionsApi.testFunction.mockResolvedValue({
        status: 200,
        body: { message: 'Hello World!' },
        duration_ms: 50,
      });

      renderWithProviders(<FunctionsPage />);

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /test/i })).toBeInTheDocument();
      });

      await user.click(screen.getByRole('button', { name: /test/i }));

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /run function/i })).toBeInTheDocument();
      });

      await user.click(screen.getByRole('button', { name: /run function/i }));

      await waitFor(() => {
        expect(screen.getByText('200')).toBeInTheDocument();
        expect(screen.getByText('50ms')).toBeInTheDocument();
      });
    });
  });

  describe('Logs Tab', () => {
    it('loads logs when logs tab is clicked', async () => {
      const user = userEvent.setup();
      renderWithProviders(<FunctionsPage />);

      await waitFor(() => {
        expect(screen.getByRole('tab', { name: /logs/i })).toBeInTheDocument();
      });

      await user.click(screen.getByRole('tab', { name: /logs/i }));

      await waitFor(() => {
        expect(mockFunctionsApi.getLogs).toHaveBeenCalledWith('func-1', { limit: 100 });
      });
    });

    it('displays logs', async () => {
      const user = userEvent.setup();
      renderWithProviders(<FunctionsPage />);

      await waitFor(() => {
        expect(screen.getByRole('tab', { name: /logs/i })).toBeInTheDocument();
      });

      await user.click(screen.getByRole('tab', { name: /logs/i }));

      await waitFor(() => {
        expect(screen.getByText('Function invoked successfully')).toBeInTheDocument();
      });
    });

    it('shows empty state when no logs', async () => {
      const user = userEvent.setup();
      mockFunctionsApi.getLogs.mockResolvedValue({ logs: [] });
      renderWithProviders(<FunctionsPage />);

      await waitFor(() => {
        expect(screen.getByRole('tab', { name: /logs/i })).toBeInTheDocument();
      });

      await user.click(screen.getByRole('tab', { name: /logs/i }));

      await waitFor(() => {
        expect(screen.getByText('No logs yet')).toBeInTheDocument();
      });
    });
  });

  describe('Metrics Tab', () => {
    it('loads metrics when metrics tab is clicked', async () => {
      const user = userEvent.setup();
      renderWithProviders(<FunctionsPage />);

      await waitFor(() => {
        expect(screen.getByRole('tab', { name: /metrics/i })).toBeInTheDocument();
      });

      await user.click(screen.getByRole('tab', { name: /metrics/i }));

      await waitFor(() => {
        expect(mockFunctionsApi.getMetrics).toHaveBeenCalled();
      });
    });

    it('displays total invocations', async () => {
      const user = userEvent.setup();
      renderWithProviders(<FunctionsPage />);

      await waitFor(() => {
        expect(screen.getByRole('tab', { name: /metrics/i })).toBeInTheDocument();
      });

      await user.click(screen.getByRole('tab', { name: /metrics/i }));

      await waitFor(() => {
        expect(screen.getByText('1,500')).toBeInTheDocument();
      });
    });

    it('displays average latency', async () => {
      const user = userEvent.setup();
      renderWithProviders(<FunctionsPage />);

      await waitFor(() => {
        expect(screen.getByRole('tab', { name: /metrics/i })).toBeInTheDocument();
      });

      await user.click(screen.getByRole('tab', { name: /metrics/i }));

      await waitFor(() => {
        expect(screen.getByText('45ms')).toBeInTheDocument();
      });
    });

    it('renders chart', async () => {
      const user = userEvent.setup();
      renderWithProviders(<FunctionsPage />);

      await waitFor(() => {
        expect(screen.getByRole('tab', { name: /metrics/i })).toBeInTheDocument();
      });

      await user.click(screen.getByRole('tab', { name: /metrics/i }));

      await waitFor(() => {
        expect(screen.getByTestId('area-chart')).toBeInTheDocument();
      });
    });
  });

  describe('Deployments Tab', () => {
    it('loads deployments when tab is clicked', async () => {
      const user = userEvent.setup();
      renderWithProviders(<FunctionsPage />);

      await waitFor(() => {
        expect(screen.getByRole('tab', { name: /deployments/i })).toBeInTheDocument();
      });

      await user.click(screen.getByRole('tab', { name: /deployments/i }));

      await waitFor(() => {
        expect(mockFunctionsApi.listDeployments).toHaveBeenCalledWith('func-1');
      });
    });

    it('displays deployment history', async () => {
      const user = userEvent.setup();
      renderWithProviders(<FunctionsPage />);

      await waitFor(() => {
        expect(screen.getByRole('tab', { name: /deployments/i })).toBeInTheDocument();
      });

      await user.click(screen.getByRole('tab', { name: /deployments/i }));

      await waitFor(() => {
        expect(screen.getByText('deployed')).toBeInTheDocument();
        expect(screen.getByText('Version 1')).toBeInTheDocument();
        expect(screen.getByText('Current')).toBeInTheDocument();
      });
    });

    it('shows empty state when no deployments', async () => {
      const user = userEvent.setup();
      mockFunctionsApi.listDeployments.mockResolvedValue([]);
      renderWithProviders(<FunctionsPage />);

      await waitFor(() => {
        expect(screen.getByRole('tab', { name: /deployments/i })).toBeInTheDocument();
      });

      await user.click(screen.getByRole('tab', { name: /deployments/i }));

      await waitFor(() => {
        expect(screen.getByText('No deployments yet')).toBeInTheDocument();
      });
    });
  });

  describe('Settings Tab', () => {
    it('displays function settings', async () => {
      const user = userEvent.setup();
      renderWithProviders(<FunctionsPage />);

      await waitFor(() => {
        expect(screen.getByRole('tab', { name: /settings/i })).toBeInTheDocument();
      });

      await user.click(screen.getByRole('tab', { name: /settings/i }));

      await waitFor(() => {
        expect(screen.getByText('Function Settings')).toBeInTheDocument();
      });
    });

    it('displays endpoint section', async () => {
      const user = userEvent.setup();
      renderWithProviders(<FunctionsPage />);

      await waitFor(() => {
        expect(screen.getByRole('tab', { name: /settings/i })).toBeInTheDocument();
      });

      await user.click(screen.getByRole('tab', { name: /settings/i }));

      await waitFor(() => {
        expect(screen.getByText('Endpoint')).toBeInTheDocument();
      });
    });

    it('shows danger zone', async () => {
      const user = userEvent.setup();
      renderWithProviders(<FunctionsPage />);

      await waitFor(() => {
        expect(screen.getByRole('tab', { name: /settings/i })).toBeInTheDocument();
      });

      await user.click(screen.getByRole('tab', { name: /settings/i }));

      await waitFor(() => {
        expect(screen.getByText('Danger Zone')).toBeInTheDocument();
      });
    });
  });

  describe('Create Function Modal', () => {
    it('opens create modal when button is clicked', async () => {
      const user = userEvent.setup();
      renderWithProviders(<FunctionsPage />);

      await waitFor(() => {
        const buttons = screen.getAllByRole('button', { name: /create function/i });
        expect(buttons.length).toBeGreaterThan(0);
      });

      // Click the first create button
      const createButtons = screen.getAllByRole('button', { name: /create function/i });
      await user.click(createButtons[0]);

      await waitFor(() => {
        expect(screen.getByText('Create Edge Function')).toBeInTheDocument();
      });
    });

    it('displays function name input', async () => {
      const user = userEvent.setup();
      renderWithProviders(<FunctionsPage />);

      await waitFor(() => {
        const buttons = screen.getAllByRole('button', { name: /create function/i });
        expect(buttons.length).toBeGreaterThan(0);
      });

      const createButtons = screen.getAllByRole('button', { name: /create function/i });
      await user.click(createButtons[0]);

      await waitFor(() => {
        expect(screen.getByPlaceholderText('my-function')).toBeInTheDocument();
      });
    });

    it('displays templates', async () => {
      const user = userEvent.setup();
      renderWithProviders(<FunctionsPage />);

      await waitFor(() => {
        const buttons = screen.getAllByRole('button', { name: /create function/i });
        expect(buttons.length).toBeGreaterThan(0);
      });

      const createButtons = screen.getAllByRole('button', { name: /create function/i });
      await user.click(createButtons[0]);

      await waitFor(() => {
        expect(screen.getByText('Hello World')).toBeInTheDocument();
        expect(screen.getByText('Stripe Webhook')).toBeInTheDocument();
      });
    });

  });

  describe('Secrets Modal', () => {
    it('displays existing secrets', async () => {
      const user = userEvent.setup();
      renderWithProviders(<FunctionsPage />);

      await waitFor(() => {
        const buttons = screen.getAllByRole('button', { name: /manage secrets/i });
        expect(buttons.length).toBeGreaterThan(0);
      });

      await user.click(screen.getAllByRole('button', { name: /manage secrets/i })[0]);

      await waitFor(() => {
        expect(screen.getByText('Manage Secrets')).toBeInTheDocument();
        expect(screen.getByText('API_KEY')).toBeInTheDocument();
        expect(screen.getByText('DATABASE_URL')).toBeInTheDocument();
      });
    });

    it('shows add secret inputs', async () => {
      const user = userEvent.setup();
      renderWithProviders(<FunctionsPage />);

      await waitFor(() => {
        const buttons = screen.getAllByRole('button', { name: /manage secrets/i });
        expect(buttons.length).toBeGreaterThan(0);
      });

      await user.click(screen.getAllByRole('button', { name: /manage secrets/i })[0]);

      await waitFor(() => {
        expect(screen.getByPlaceholderText('Secret name')).toBeInTheDocument();
        expect(screen.getByPlaceholderText('Secret value')).toBeInTheDocument();
      });
    });
  });

  describe('Function Actions', () => {
    it('deploys function when deploy button is clicked', async () => {
      const user = userEvent.setup();
      mockFunctionsApi.deployFunction.mockResolvedValue({});
      renderWithProviders(<FunctionsPage />);

      await waitFor(() => {
        expect(screen.getByTestId('monaco-editor')).toBeInTheDocument();
      });

      // Modify code to enable deploy button
      const editor = screen.getByTestId('monaco-editor');
      await user.clear(editor);
      await user.type(editor, '// modified');

      await user.click(screen.getByRole('button', { name: /deploy/i }));

      await waitFor(() => {
        expect(mockFunctionsApi.deployFunction).toHaveBeenCalled();
      });
    });
  });

  describe('Empty State', () => {
    it('shows message to select or create function when none selected', async () => {
      mockFunctionsApi.listFunctions.mockResolvedValue([]);
      renderWithProviders(<FunctionsPage />);

      await waitFor(() => {
        expect(screen.getByText(/select a function or create a new one/i)).toBeInTheDocument();
      });
    });
  });

  describe('Error Handling', () => {
    it('handles API errors gracefully', async () => {
      const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
      mockFunctionsApi.listFunctions.mockRejectedValue(new Error('API Error'));

      renderWithProviders(<FunctionsPage />);

      await waitFor(() => {
        expect(consoleSpy).toHaveBeenCalledWith('Failed to load functions:', expect.any(Error));
      });

      consoleSpy.mockRestore();
    });

    it('handles source code loading errors', async () => {
      const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
      mockFunctionsApi.getSource.mockRejectedValue(new Error('Source Error'));

      renderWithProviders(<FunctionsPage />);

      await waitFor(() => {
        expect(consoleSpy).toHaveBeenCalledWith('Failed to load source:', expect.any(Error));
      });

      consoleSpy.mockRestore();
    });
  });

  describe('Secrets Display in Sidebar', () => {
    it('shows secrets in sidebar', async () => {
      renderWithProviders(<FunctionsPage />);

      await waitFor(() => {
        // Check for secrets section
        expect(screen.getByText('SECRETS')).toBeInTheDocument();
      });
    });
  });

  describe('Accessibility', () => {
    it('has accessible tab navigation', async () => {
      renderWithProviders(<FunctionsPage />);

      await waitFor(() => {
        const tabs = screen.getAllByRole('tab');
        expect(tabs.length).toBeGreaterThan(0);
      });
    });

    it('has accessible buttons', async () => {
      renderWithProviders(<FunctionsPage />);

      await waitFor(() => {
        const buttons = screen.getAllByRole('button');
        expect(buttons.length).toBeGreaterThan(0);
      });
    });

    it('has accessible modals', async () => {
      const user = userEvent.setup();
      renderWithProviders(<FunctionsPage />);

      await waitFor(() => {
        const buttons = screen.getAllByRole('button', { name: /create function/i });
        expect(buttons.length).toBeGreaterThan(0);
      });

      const createButtons = screen.getAllByRole('button', { name: /create function/i });
      await user.click(createButtons[0]);

      await waitFor(() => {
        const dialog = screen.getByRole('dialog');
        expect(dialog).toBeInTheDocument();
      });
    });
  });
});
