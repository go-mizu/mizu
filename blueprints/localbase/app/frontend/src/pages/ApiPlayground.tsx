import { useState, useEffect, useCallback, useMemo } from 'react';
import {
  Box,
  Stack,
  Text,
  Badge,
  Code,
  Group,
  CopyButton,
  ActionIcon,
  Tooltip,
  Select,
  TextInput,
  Button,
  Tabs,
  ScrollArea,
  Loader,
  Alert,
  SegmentedControl,
  Divider,
  Collapse,
  Switch,
  ThemeIcon,
} from '@mantine/core';
import {
  IconCopy,
  IconCheck,
  IconApi,
  IconSend,
  IconClock,
  IconDatabase,
  IconShield,
  IconFolder,
  IconBolt,
  IconTrash,
  IconHistory,
  IconPlus,
  IconChevronDown,
  IconChevronRight,
  IconBrandJavascript,
  IconBrandPython,
  IconBrandGolang,
  IconTerminal,
  IconSearch,
  IconAlertCircle,
  IconCircleCheck,
} from '@tabler/icons-react';
import { PageContainer } from '../components/layout/PageContainer';
import Editor from '@monaco-editor/react';

// Types
type HttpMethod = 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE';
type AuthRole = 'anon' | 'service_role' | 'authenticated';
type CodeLanguage = 'javascript' | 'curl' | 'python' | 'go';

interface Parameter {
  name: string;
  type: string;
  description: string;
  required?: boolean;
  example?: string;
}

interface Endpoint {
  method: HttpMethod;
  path: string;
  description: string;
  category: string;
  parameters?: Parameter[];
  requestBody?: Record<string, any>;
  example?: string;
}

interface EndpointCategory {
  name: string;
  icon: React.ReactNode;
  description: string;
  endpoints: Endpoint[];
}

interface PlaygroundResponse {
  status: number;
  statusText: string;
  headers: Record<string, string>;
  body: any;
  duration_ms: number;
}

interface RequestHistoryEntry {
  id: string;
  method: HttpMethod;
  path: string;
  status: number;
  duration_ms: number;
  timestamp: string;
  request: {
    headers: Record<string, string>;
    query: Record<string, string>;
    body: string | null;
  };
  response: PlaygroundResponse;
}

interface TableInfo {
  schema: string;
  name: string;
  columns: Array<{
    name: string;
    type: string;
    is_nullable: boolean;
    is_primary_key: boolean;
  }>;
  rls_enabled: boolean;
}

// Default endpoints organized by category
const defaultEndpoints: EndpointCategory[] = [
  {
    name: 'Authentication',
    icon: <IconShield size={16} />,
    description: 'User authentication and session management',
    endpoints: [
      {
        method: 'POST',
        path: '/auth/v1/signup',
        description: 'Register a new user with email and password',
        category: 'Authentication',
        requestBody: { email: 'user@example.com', password: 'password123' },
      },
      {
        method: 'POST',
        path: '/auth/v1/token?grant_type=password',
        description: 'Sign in with email and password',
        category: 'Authentication',
        requestBody: { email: 'user@example.com', password: 'password123' },
      },
      {
        method: 'POST',
        path: '/auth/v1/logout',
        description: 'Sign out the current user',
        category: 'Authentication',
      },
      {
        method: 'GET',
        path: '/auth/v1/user',
        description: 'Get the current authenticated user',
        category: 'Authentication',
      },
      {
        method: 'PUT',
        path: '/auth/v1/user',
        description: 'Update the current user',
        category: 'Authentication',
        requestBody: { data: { display_name: 'John Doe' } },
      },
      {
        method: 'POST',
        path: '/auth/v1/recover',
        description: 'Send password recovery email',
        category: 'Authentication',
        requestBody: { email: 'user@example.com' },
      },
      {
        method: 'GET',
        path: '/auth/v1/admin/users',
        description: 'List all users (requires service_role)',
        category: 'Authentication',
      },
      {
        method: 'POST',
        path: '/auth/v1/admin/users',
        description: 'Create a new user (requires service_role)',
        category: 'Authentication',
        requestBody: { email: 'newuser@example.com', password: 'password123', email_confirm: true },
      },
    ],
  },
  {
    name: 'Database',
    icon: <IconDatabase size={16} />,
    description: 'CRUD operations on database tables',
    endpoints: [
      {
        method: 'GET',
        path: '/rest/v1/{table}',
        description: 'Select rows from a table',
        category: 'Database',
        parameters: [
          { name: 'select', type: 'string', description: 'Columns to return (e.g., *, id,name)', example: '*' },
          { name: 'limit', type: 'integer', description: 'Maximum number of rows to return', example: '10' },
          { name: 'offset', type: 'integer', description: 'Number of rows to skip', example: '0' },
          { name: 'order', type: 'string', description: 'Order by column (e.g., created_at.desc)', example: 'created_at.desc' },
        ],
      },
      {
        method: 'POST',
        path: '/rest/v1/{table}',
        description: 'Insert rows into a table',
        category: 'Database',
        requestBody: { column1: 'value1', column2: 'value2' },
      },
      {
        method: 'PATCH',
        path: '/rest/v1/{table}?id=eq.{id}',
        description: 'Update rows matching filter',
        category: 'Database',
        requestBody: { column1: 'newvalue' },
      },
      {
        method: 'DELETE',
        path: '/rest/v1/{table}?id=eq.{id}',
        description: 'Delete rows matching filter',
        category: 'Database',
      },
      {
        method: 'POST',
        path: '/rest/v1/rpc/{function}',
        description: 'Call a PostgreSQL function',
        category: 'Database',
        requestBody: { param1: 'value1' },
      },
    ],
  },
  {
    name: 'Storage',
    icon: <IconFolder size={16} />,
    description: 'File storage operations',
    endpoints: [
      {
        method: 'GET',
        path: '/storage/v1/bucket',
        description: 'List all buckets',
        category: 'Storage',
      },
      {
        method: 'POST',
        path: '/storage/v1/bucket',
        description: 'Create a new bucket',
        category: 'Storage',
        requestBody: { name: 'my-bucket', public: false },
      },
      {
        method: 'GET',
        path: '/storage/v1/bucket/{bucket_id}',
        description: 'Get bucket details',
        category: 'Storage',
      },
      {
        method: 'DELETE',
        path: '/storage/v1/bucket/{bucket_id}',
        description: 'Delete a bucket',
        category: 'Storage',
      },
      {
        method: 'POST',
        path: '/storage/v1/object/list/{bucket_id}',
        description: 'List objects in a bucket',
        category: 'Storage',
        requestBody: { prefix: '', limit: 100 },
      },
      {
        method: 'GET',
        path: '/storage/v1/object/{bucket_id}/{path}',
        description: 'Download an object',
        category: 'Storage',
      },
      {
        method: 'DELETE',
        path: '/storage/v1/object/{bucket_id}/{path}',
        description: 'Delete an object',
        category: 'Storage',
      },
      {
        method: 'POST',
        path: '/storage/v1/object/sign/{bucket_id}/{path}',
        description: 'Create a signed URL for an object',
        category: 'Storage',
        requestBody: { expiresIn: 3600 },
      },
    ],
  },
  {
    name: 'Edge Functions',
    icon: <IconBolt size={16} />,
    description: 'Serverless function invocation',
    endpoints: [
      {
        method: 'POST',
        path: '/functions/v1/{function_name}',
        description: 'Invoke an edge function',
        category: 'Edge Functions',
        requestBody: { key: 'value' },
      },
      {
        method: 'GET',
        path: '/api/functions',
        description: 'List all functions (requires service_role)',
        category: 'Edge Functions',
      },
      {
        method: 'POST',
        path: '/api/functions',
        description: 'Create a new function (requires service_role)',
        category: 'Edge Functions',
        requestBody: { name: 'my-function', slug: 'my-function' },
      },
    ],
  },
  {
    name: 'Realtime',
    icon: <IconBolt size={16} />,
    description: 'Realtime subscriptions and channels',
    endpoints: [
      {
        method: 'GET',
        path: '/api/realtime/channels',
        description: 'List active realtime channels',
        category: 'Realtime',
      },
      {
        method: 'GET',
        path: '/api/realtime/stats',
        description: 'Get realtime statistics',
        category: 'Realtime',
      },
    ],
  },
];

// Method badge colors
const methodColors: Record<HttpMethod, string> = {
  GET: 'green',
  POST: 'blue',
  PUT: 'orange',
  PATCH: 'violet',
  DELETE: 'red',
};

// Status code colors
const getStatusColor = (status: number): string => {
  if (status >= 200 && status < 300) return 'green';
  if (status >= 300 && status < 400) return 'blue';
  if (status >= 400 && status < 500) return 'orange';
  return 'red';
};

// Code generators
const generateJavaScriptCode = (
  method: HttpMethod,
  path: string,
  headers: Record<string, string>,
  query: Record<string, string>,
  body: string | null
): string => {
  const baseUrl = window.location.origin;
  let url = `${baseUrl}${path}`;

  const queryString = Object.entries(query)
    .filter(([_, v]) => v)
    .map(([k, v]) => `${encodeURIComponent(k)}=${encodeURIComponent(v)}`)
    .join('&');
  if (queryString) url += `?${queryString}`;

  const options: string[] = [];
  options.push(`  method: '${method}'`);

  const headerLines = Object.entries(headers)
    .filter(([_, v]) => v)
    .map(([k, v]) => `    '${k}': '${v}'`);
  if (headerLines.length > 0) {
    options.push(`  headers: {\n${headerLines.join(',\n')}\n  }`);
  }

  if (body && ['POST', 'PUT', 'PATCH'].includes(method)) {
    options.push(`  body: JSON.stringify(${body})`);
  }

  return `const response = await fetch('${url}', {
${options.join(',\n')}
});

const data = await response.json();
console.log(data);`;
};

const generateCurlCode = (
  method: HttpMethod,
  path: string,
  headers: Record<string, string>,
  query: Record<string, string>,
  body: string | null
): string => {
  const baseUrl = window.location.origin;
  let url = `${baseUrl}${path}`;

  const queryString = Object.entries(query)
    .filter(([_, v]) => v)
    .map(([k, v]) => `${encodeURIComponent(k)}=${encodeURIComponent(v)}`)
    .join('&');
  if (queryString) url += `?${queryString}`;

  const parts = [`curl -X ${method} '${url}'`];

  Object.entries(headers)
    .filter(([_, v]) => v)
    .forEach(([k, v]) => {
      parts.push(`  -H "${k}: ${v}"`);
    });

  if (body && ['POST', 'PUT', 'PATCH'].includes(method)) {
    parts.push(`  -d '${body}'`);
  }

  return parts.join(' \\\n');
};

const generatePythonCode = (
  method: HttpMethod,
  path: string,
  headers: Record<string, string>,
  query: Record<string, string>,
  body: string | null
): string => {
  const baseUrl = window.location.origin;
  let url = `${baseUrl}${path}`;

  const headerDict = Object.entries(headers)
    .filter(([_, v]) => v)
    .map(([k, v]) => `    "${k}": "${v}"`)
    .join(',\n');

  const queryDict = Object.entries(query)
    .filter(([_, v]) => v)
    .map(([k, v]) => `    "${k}": "${v}"`)
    .join(',\n');

  let code = `import requests

url = "${url}"
headers = {
${headerDict}
}`;

  if (queryDict) {
    code += `
params = {
${queryDict}
}`;
  }

  if (body && ['POST', 'PUT', 'PATCH'].includes(method)) {
    code += `
data = ${body}`;
  }

  const args = ['url', 'headers=headers'];
  if (queryDict) args.push('params=params');
  if (body && ['POST', 'PUT', 'PATCH'].includes(method)) args.push('json=data');

  code += `

response = requests.${method.toLowerCase()}(${args.join(', ')})
print(response.json())`;

  return code;
};

const generateGoCode = (
  method: HttpMethod,
  path: string,
  headers: Record<string, string>,
  query: Record<string, string>,
  body: string | null
): string => {
  const baseUrl = window.location.origin;
  let url = `${baseUrl}${path}`;

  const queryString = Object.entries(query)
    .filter(([_, v]) => v)
    .map(([k, v]) => `${encodeURIComponent(k)}=${encodeURIComponent(v)}`)
    .join('&');
  if (queryString) url += `?${queryString}`;

  let bodySetup = '';
  let bodyArg = 'nil';
  if (body && ['POST', 'PUT', 'PATCH'].includes(method)) {
    bodySetup = `
    body := strings.NewReader(\`${body}\`)`;
    bodyArg = 'body';
  }

  const headerLines = Object.entries(headers)
    .filter(([_, v]) => v)
    .map(([k, v]) => `    req.Header.Set("${k}", "${v}")`)
    .join('\n');

  return `package main

import (
    "fmt"
    "io"
    "net/http"
    "strings"
)

func main() {${bodySetup}
    req, _ := http.NewRequest("${method}", "${url}", ${bodyArg})
${headerLines}

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        panic(err)
    }
    defer resp.Body.Close()

    data, _ := io.ReadAll(resp.Body)
    fmt.Println(string(data))
}`;
};

// Component: Method Badge
function MethodBadge({ method }: { method: HttpMethod }) {
  return (
    <Badge
      size="xs"
      variant="filled"
      color={methodColors[method]}
      style={{ fontWeight: 600, minWidth: 55, textAlign: 'center' }}
    >
      {method}
    </Badge>
  );
}

// Component: Endpoint Item
interface EndpointItemProps {
  endpoint: Endpoint;
  isSelected: boolean;
  onClick: () => void;
}

function EndpointItem({ endpoint, isSelected, onClick }: EndpointItemProps) {
  return (
    <Box
      onClick={onClick}
      p="xs"
      style={{
        cursor: 'pointer',
        borderRadius: 6,
        backgroundColor: isSelected ? 'var(--supabase-bg-hover)' : 'transparent',
        borderLeft: isSelected ? '2px solid var(--mantine-color-green-6)' : '2px solid transparent',
        transition: 'all 150ms ease',
      }}
      className="endpoint-item"
    >
      <Group gap="xs" wrap="nowrap">
        <MethodBadge method={endpoint.method} />
        <Text size="xs" style={{ fontFamily: 'monospace' }} truncate>
          {endpoint.path}
        </Text>
      </Group>
    </Box>
  );
}

// Component: Key-Value Editor
interface KeyValuePair {
  key: string;
  value: string;
  enabled: boolean;
}

interface KeyValueEditorProps {
  pairs: KeyValuePair[];
  onChange: (pairs: KeyValuePair[]) => void;
  placeholder?: { key: string; value: string };
}

function KeyValueEditor({ pairs, onChange, placeholder }: KeyValueEditorProps) {
  const addPair = () => {
    onChange([...pairs, { key: '', value: '', enabled: true }]);
  };

  const removePair = (index: number) => {
    onChange(pairs.filter((_, i) => i !== index));
  };

  const updatePair = (index: number, field: 'key' | 'value' | 'enabled', value: string | boolean) => {
    const newPairs = [...pairs];
    newPairs[index] = { ...newPairs[index], [field]: value };
    onChange(newPairs);
  };

  return (
    <Stack gap="xs">
      {pairs.map((pair, index) => (
        <Group key={index} gap="xs" wrap="nowrap">
          <Switch
            size="xs"
            checked={pair.enabled}
            onChange={(e) => updatePair(index, 'enabled', e.target.checked)}
          />
          <TextInput
            size="xs"
            placeholder={placeholder?.key || 'Key'}
            value={pair.key}
            onChange={(e) => updatePair(index, 'key', e.target.value)}
            style={{ flex: 1 }}
            styles={{ input: { fontFamily: 'monospace', fontSize: 12 } }}
          />
          <TextInput
            size="xs"
            placeholder={placeholder?.value || 'Value'}
            value={pair.value}
            onChange={(e) => updatePair(index, 'value', e.target.value)}
            style={{ flex: 1 }}
            styles={{ input: { fontFamily: 'monospace', fontSize: 12 } }}
          />
          <ActionIcon
            size="sm"
            variant="subtle"
            color="red"
            onClick={() => removePair(index)}
          >
            <IconTrash size={14} />
          </ActionIcon>
        </Group>
      ))}
      <Button
        size="xs"
        variant="subtle"
        leftSection={<IconPlus size={14} />}
        onClick={addPair}
        style={{ alignSelf: 'flex-start' }}
      >
        Add
      </Button>
    </Stack>
  );
}

// Component: Code Snippet Viewer
interface CodeSnippetProps {
  language: CodeLanguage;
  code: string;
}

function CodeSnippet({ language, code }: CodeSnippetProps) {
  const languageMap: Record<CodeLanguage, string> = {
    javascript: 'javascript',
    curl: 'shell',
    python: 'python',
    go: 'go',
  };

  return (
    <Box style={{ position: 'relative' }}>
      <CopyButton value={code}>
        {({ copied, copy }) => (
          <Tooltip label={copied ? 'Copied!' : 'Copy'} position="left">
            <ActionIcon
              variant="subtle"
              size="sm"
              onClick={copy}
              style={{ position: 'absolute', top: 8, right: 8, zIndex: 10 }}
            >
              {copied ? <IconCheck size={14} /> : <IconCopy size={14} />}
            </ActionIcon>
          </Tooltip>
        )}
      </CopyButton>
      <Editor
        height="200px"
        language={languageMap[language]}
        value={code}
        theme="vs-dark"
        options={{
          readOnly: true,
          minimap: { enabled: false },
          scrollBeyondLastLine: false,
          fontSize: 12,
          lineNumbers: 'off',
          folding: false,
          padding: { top: 8, bottom: 8 },
        }}
      />
    </Box>
  );
}

// Default API key
const DEFAULT_SERVICE_KEY = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZS1kZW1vIiwicm9sZSI6InNlcnZpY2Vfcm9sZSIsImV4cCI6MTk4MzgxMjk5Nn0.EGIM96RAZx35lJzdJsyH-qQwv8Hdp7fsn3W0YpN81IU';
const DEFAULT_ANON_KEY = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZS1kZW1vIiwicm9sZSI6ImFub24iLCJleHAiOjE5ODM4MTI5OTZ9.CRXP1A7WOeoJeXxjNni43kdQwgnWNReilDMblYTn_I0';

// Main Component
export function ApiPlaygroundPage() {
  const baseUrl = window.location.origin;

  // Request state
  const [method, setMethod] = useState<HttpMethod>('GET');
  const [path, setPath] = useState('/rest/v1/');
  const [headers, setHeaders] = useState<KeyValuePair[]>([
    { key: 'Content-Type', value: 'application/json', enabled: true },
  ]);
  const [queryParams, setQueryParams] = useState<KeyValuePair[]>([]);
  const [body, setBody] = useState('');

  // Response state
  const [response, setResponse] = useState<PlaygroundResponse | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // UI state
  const [selectedEndpoint, setSelectedEndpoint] = useState<Endpoint | null>(null);
  const [authRole, setAuthRole] = useState<AuthRole>('service_role');
  const [codeLanguage, setCodeLanguage] = useState<CodeLanguage>('javascript');
  const [expandedCategories, setExpandedCategories] = useState<string[]>(['Authentication', 'Database']);
  const [history, setHistory] = useState<RequestHistoryEntry[]>([]);
  const [showHistory, setShowHistory] = useState(false);
  const [tables, setTables] = useState<TableInfo[]>([]);
  const [searchQuery, setSearchQuery] = useState('');

  // Fetch tables for dynamic endpoints
  useEffect(() => {
    const fetchTables = async () => {
      try {
        const serviceKey = localStorage.getItem('serviceKey') || DEFAULT_SERVICE_KEY;
        const res = await fetch('/api/database/tables', {
          headers: {
            'apikey': serviceKey,
            'Authorization': `Bearer ${serviceKey}`,
          },
        });
        if (res.ok) {
          const data = await res.json();
          setTables(data || []);
        }
      } catch (e) {
        console.error('Failed to fetch tables:', e);
      }
    };
    fetchTables();
  }, []);

  // Build endpoints with dynamic table endpoints
  const endpoints = useMemo(() => {
    const dynamicTableEndpoints: Endpoint[] = tables.map((table) => ({
      method: 'GET' as HttpMethod,
      path: `/rest/v1/${table.name}`,
      description: `Select rows from ${table.name}`,
      category: 'Tables',
      parameters: [
        { name: 'select', type: 'string', description: 'Columns to return', example: '*' },
        { name: 'limit', type: 'integer', description: 'Max rows', example: '10' },
      ],
    }));

    const tableCategory: EndpointCategory = {
      name: 'Tables',
      icon: <IconDatabase size={16} />,
      description: 'Auto-generated endpoints for your tables',
      endpoints: dynamicTableEndpoints,
    };

    return tables.length > 0
      ? [tableCategory, ...defaultEndpoints]
      : defaultEndpoints;
  }, [tables]);

  // Filter endpoints by search
  const filteredEndpoints = useMemo(() => {
    if (!searchQuery) return endpoints;
    return endpoints
      .map((cat) => ({
        ...cat,
        endpoints: cat.endpoints.filter(
          (ep) =>
            ep.path.toLowerCase().includes(searchQuery.toLowerCase()) ||
            ep.description.toLowerCase().includes(searchQuery.toLowerCase())
        ),
      }))
      .filter((cat) => cat.endpoints.length > 0);
  }, [endpoints, searchQuery]);

  // Get current auth key based on role
  const getAuthKey = useCallback(() => {
    switch (authRole) {
      case 'anon':
        return DEFAULT_ANON_KEY;
      case 'service_role':
        return localStorage.getItem('serviceKey') || DEFAULT_SERVICE_KEY;
      case 'authenticated':
        return localStorage.getItem('userToken') || DEFAULT_ANON_KEY;
    }
  }, [authRole]);

  // Build headers object
  const buildHeaders = useCallback((): Record<string, string> => {
    const authKey = getAuthKey();
    const result: Record<string, string> = {
      'apikey': authKey,
      'Authorization': `Bearer ${authKey}`,
    };
    headers
      .filter((h) => h.enabled && h.key)
      .forEach((h) => {
        result[h.key] = h.value;
      });
    return result;
  }, [headers, getAuthKey]);

  // Build query object
  const buildQuery = useCallback((): Record<string, string> => {
    const result: Record<string, string> = {};
    queryParams
      .filter((p) => p.enabled && p.key)
      .forEach((p) => {
        result[p.key] = p.value;
      });
    return result;
  }, [queryParams]);

  // Execute request
  const executeRequest = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    setResponse(null);

    const startTime = performance.now();

    try {
      const reqHeaders = buildHeaders();
      const queryObj = buildQuery();

      let url = path;
      const queryString = Object.entries(queryObj)
        .filter(([_, v]) => v)
        .map(([k, v]) => `${encodeURIComponent(k)}=${encodeURIComponent(v)}`)
        .join('&');
      if (queryString) {
        url += (url.includes('?') ? '&' : '?') + queryString;
      }

      const fetchOptions: RequestInit = {
        method,
        headers: reqHeaders,
      };

      if (body && ['POST', 'PUT', 'PATCH'].includes(method)) {
        fetchOptions.body = body;
      }

      const res = await fetch(url, fetchOptions);
      const endTime = performance.now();
      const duration = Math.round(endTime - startTime);

      // Get response headers
      const responseHeaders: Record<string, string> = {};
      res.headers.forEach((value, key) => {
        responseHeaders[key] = value;
      });

      // Try to parse body
      let responseBody: any = null;
      const contentType = res.headers.get('content-type');
      if (contentType?.includes('application/json')) {
        const text = await res.text();
        if (text) {
          try {
            responseBody = JSON.parse(text);
          } catch {
            responseBody = text;
          }
        }
      } else {
        responseBody = await res.text();
      }

      const playgroundResponse: PlaygroundResponse = {
        status: res.status,
        statusText: res.statusText,
        headers: responseHeaders,
        body: responseBody,
        duration_ms: duration,
      };

      setResponse(playgroundResponse);

      // Add to history
      const historyEntry: RequestHistoryEntry = {
        id: crypto.randomUUID(),
        method,
        path: url,
        status: res.status,
        duration_ms: duration,
        timestamp: new Date().toISOString(),
        request: {
          headers: reqHeaders,
          query: queryObj,
          body: body || null,
        },
        response: playgroundResponse,
      };
      setHistory((prev) => [historyEntry, ...prev.slice(0, 99)]);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Request failed');
    } finally {
      setIsLoading(false);
    }
  }, [method, path, buildHeaders, buildQuery, body]);

  // Select endpoint
  const selectEndpoint = useCallback((endpoint: Endpoint) => {
    setSelectedEndpoint(endpoint);
    setMethod(endpoint.method);
    setPath(endpoint.path);
    if (endpoint.requestBody) {
      setBody(JSON.stringify(endpoint.requestBody, null, 2));
    } else {
      setBody('');
    }
    if (endpoint.parameters) {
      setQueryParams(
        endpoint.parameters.map((p) => ({
          key: p.name,
          value: p.example || '',
          enabled: !!p.example,
        }))
      );
    } else {
      setQueryParams([]);
    }
    setResponse(null);
    setError(null);
  }, []);

  // Load from history
  const loadFromHistory = useCallback((entry: RequestHistoryEntry) => {
    setMethod(entry.method);
    setPath(entry.path.split('?')[0]);
    setHeaders(
      Object.entries(entry.request.headers)
        .filter(([k]) => !['apikey', 'authorization'].includes(k.toLowerCase()))
        .map(([key, value]) => ({ key, value, enabled: true }))
    );
    setQueryParams(
      Object.entries(entry.request.query).map(([key, value]) => ({
        key,
        value,
        enabled: true,
      }))
    );
    setBody(entry.request.body || '');
    setResponse(entry.response);
    setShowHistory(false);
  }, []);

  // Generate code snippets
  const generatedCode = useMemo(() => {
    const h = buildHeaders();
    const q = buildQuery();
    const b = body || null;

    return {
      javascript: generateJavaScriptCode(method, path, h, q, b),
      curl: generateCurlCode(method, path, h, q, b),
      python: generatePythonCode(method, path, h, q, b),
      go: generateGoCode(method, path, h, q, b),
    };
  }, [method, path, buildHeaders, buildQuery, body]);

  // Toggle category expansion
  const toggleCategory = (name: string) => {
    setExpandedCategories((prev) =>
      prev.includes(name)
        ? prev.filter((n) => n !== name)
        : [...prev, name]
    );
  };

  return (
    <PageContainer
      title="API Playground"
      description="Explore and test Localbase REST APIs interactively"
      fullWidth
      noPadding
    >
      <Box
        style={{
          display: 'grid',
          gridTemplateColumns: '280px 1fr 400px',
          height: 'calc(100vh - 120px)',
          gap: 0,
        }}
      >
        {/* Left Panel: Endpoint Explorer */}
        <Box
          style={{
            borderRight: '1px solid var(--supabase-border)',
            display: 'flex',
            flexDirection: 'column',
            overflow: 'hidden',
          }}
        >
          <Box p="sm" style={{ borderBottom: '1px solid var(--supabase-border)' }}>
            <Group justify="space-between" mb="xs">
              <Text size="sm" fw={600}>Endpoints</Text>
              <Tooltip label="Request History">
                <ActionIcon
                  variant={showHistory ? 'filled' : 'subtle'}
                  size="sm"
                  onClick={() => setShowHistory(!showHistory)}
                >
                  <IconHistory size={16} />
                </ActionIcon>
              </Tooltip>
            </Group>
            <TextInput
              size="xs"
              placeholder="Search endpoints..."
              leftSection={<IconSearch size={14} />}
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
            />
          </Box>

          <ScrollArea style={{ flex: 1 }}>
            {showHistory ? (
              <Stack gap={0} p="xs">
                <Text size="xs" c="dimmed" mb="xs" fw={500}>
                  Recent Requests ({history.length})
                </Text>
                {history.length === 0 ? (
                  <Text size="xs" c="dimmed" ta="center" py="lg">
                    No requests yet
                  </Text>
                ) : (
                  history.map((entry) => (
                    <Box
                      key={entry.id}
                      p="xs"
                      onClick={() => loadFromHistory(entry)}
                      style={{
                        cursor: 'pointer',
                        borderRadius: 6,
                        borderBottom: '1px solid var(--supabase-border)',
                      }}
                      className="endpoint-item"
                    >
                      <Group gap="xs" wrap="nowrap" mb={4}>
                        <MethodBadge method={entry.method} />
                        <Badge size="xs" color={getStatusColor(entry.status)}>
                          {entry.status}
                        </Badge>
                        <Text size="xs" c="dimmed">
                          {entry.duration_ms}ms
                        </Text>
                      </Group>
                      <Text size="xs" style={{ fontFamily: 'monospace' }} truncate>
                        {entry.path}
                      </Text>
                      <Text size="xs" c="dimmed">
                        {new Date(entry.timestamp).toLocaleTimeString()}
                      </Text>
                    </Box>
                  ))
                )}
              </Stack>
            ) : (
              <Stack gap={0} p="xs">
                {filteredEndpoints.map((category) => (
                  <Box key={category.name} mb="xs">
                    <Group
                      gap="xs"
                      p="xs"
                      onClick={() => toggleCategory(category.name)}
                      style={{
                        cursor: 'pointer',
                        borderRadius: 6,
                      }}
                      className="endpoint-item"
                    >
                      {expandedCategories.includes(category.name) ? (
                        <IconChevronDown size={14} />
                      ) : (
                        <IconChevronRight size={14} />
                      )}
                      <ThemeIcon size="sm" variant="light" color="gray">
                        {category.icon}
                      </ThemeIcon>
                      <Text size="sm" fw={500}>
                        {category.name}
                      </Text>
                      <Badge size="xs" variant="light" color="gray">
                        {category.endpoints.length}
                      </Badge>
                    </Group>
                    <Collapse in={expandedCategories.includes(category.name)}>
                      <Stack gap={2} pl="xl">
                        {category.endpoints.map((endpoint, index) => (
                          <EndpointItem
                            key={`${endpoint.method}-${endpoint.path}-${index}`}
                            endpoint={endpoint}
                            isSelected={
                              selectedEndpoint?.path === endpoint.path &&
                              selectedEndpoint?.method === endpoint.method
                            }
                            onClick={() => selectEndpoint(endpoint)}
                          />
                        ))}
                      </Stack>
                    </Collapse>
                  </Box>
                ))}
              </Stack>
            )}
          </ScrollArea>
        </Box>

        {/* Center Panel: Request Builder */}
        <Box
          style={{
            display: 'flex',
            flexDirection: 'column',
            overflow: 'hidden',
          }}
        >
          {/* URL Bar */}
          <Box p="sm" style={{ borderBottom: '1px solid var(--supabase-border)' }}>
            <Group gap="sm" wrap="nowrap">
              <Select
                size="sm"
                value={method}
                onChange={(v) => v && setMethod(v as HttpMethod)}
                data={['GET', 'POST', 'PUT', 'PATCH', 'DELETE']}
                style={{ width: 100 }}
                styles={{
                  input: {
                    fontWeight: 600,
                    color: `var(--mantine-color-${methodColors[method]}-6)`,
                  },
                }}
              />
              <TextInput
                size="sm"
                value={path}
                onChange={(e) => setPath(e.target.value)}
                placeholder="/rest/v1/table"
                style={{ flex: 1 }}
                styles={{ input: { fontFamily: 'monospace' } }}
                leftSection={<Text size="xs" c="dimmed">{baseUrl}</Text>}
                leftSectionWidth={140}
              />
              <Select
                size="sm"
                value={authRole}
                onChange={(v) => v && setAuthRole(v as AuthRole)}
                data={[
                  { value: 'anon', label: 'anon' },
                  { value: 'service_role', label: 'service_role' },
                  { value: 'authenticated', label: 'authenticated' },
                ]}
                style={{ width: 140 }}
              />
              <Button
                size="sm"
                leftSection={isLoading ? <Loader size={14} /> : <IconSend size={14} />}
                onClick={executeRequest}
                disabled={isLoading}
                color="green"
              >
                Send
              </Button>
            </Group>
          </Box>

          {/* Request Configuration */}
          <ScrollArea style={{ flex: 1 }}>
            <Box p="sm">
              <Tabs defaultValue="params">
                <Tabs.List>
                  <Tabs.Tab value="params">Query Params</Tabs.Tab>
                  <Tabs.Tab value="headers">Headers</Tabs.Tab>
                  <Tabs.Tab value="body">Body</Tabs.Tab>
                  <Tabs.Tab value="auth">Authentication</Tabs.Tab>
                </Tabs.List>

                <Tabs.Panel value="params" pt="sm">
                  <KeyValueEditor
                    pairs={queryParams}
                    onChange={setQueryParams}
                    placeholder={{ key: 'Parameter', value: 'Value' }}
                  />
                </Tabs.Panel>

                <Tabs.Panel value="headers" pt="sm">
                  <KeyValueEditor
                    pairs={headers}
                    onChange={setHeaders}
                    placeholder={{ key: 'Header', value: 'Value' }}
                  />
                </Tabs.Panel>

                <Tabs.Panel value="body" pt="sm">
                  {['POST', 'PUT', 'PATCH'].includes(method) ? (
                    <Box style={{ border: '1px solid var(--supabase-border)', borderRadius: 6 }}>
                      <Editor
                        height="200px"
                        language="json"
                        value={body}
                        onChange={(v) => setBody(v || '')}
                        theme="vs-dark"
                        options={{
                          minimap: { enabled: false },
                          scrollBeyondLastLine: false,
                          fontSize: 12,
                          lineNumbers: 'on',
                          folding: true,
                          padding: { top: 8, bottom: 8 },
                        }}
                      />
                    </Box>
                  ) : (
                    <Text size="sm" c="dimmed">
                      Body is only available for POST, PUT, and PATCH requests.
                    </Text>
                  )}
                </Tabs.Panel>

                <Tabs.Panel value="auth" pt="sm">
                  <Stack gap="md">
                    <Box>
                      <Text size="sm" fw={500} mb="xs">Authentication Role</Text>
                      <SegmentedControl
                        size="xs"
                        value={authRole}
                        onChange={(v) => setAuthRole(v as AuthRole)}
                        data={[
                          { label: 'Anonymous', value: 'anon' },
                          { label: 'Service Role', value: 'service_role' },
                          { label: 'Authenticated', value: 'authenticated' },
                        ]}
                      />
                    </Box>
                    <Box>
                      <Text size="sm" fw={500} mb="xs">Current API Key</Text>
                      <Code block style={{ fontSize: 11, wordBreak: 'break-all' }}>
                        {getAuthKey().substring(0, 50)}...
                      </Code>
                    </Box>
                    <Alert color="blue" variant="light">
                      <Text size="xs">
                        <strong>anon:</strong> Public key for client-side use with RLS policies<br />
                        <strong>service_role:</strong> Admin key that bypasses RLS<br />
                        <strong>authenticated:</strong> User JWT after login
                      </Text>
                    </Alert>
                  </Stack>
                </Tabs.Panel>
              </Tabs>

              {/* Selected Endpoint Info */}
              {selectedEndpoint && (
                <Box mt="lg">
                  <Divider mb="md" />
                  <Text size="sm" fw={500} mb="xs">
                    {selectedEndpoint.description}
                  </Text>
                  {selectedEndpoint.parameters && selectedEndpoint.parameters.length > 0 && (
                    <Box mt="sm">
                      <Text size="xs" c="dimmed" mb="xs">
                        Available Parameters
                      </Text>
                      <Stack gap="xs">
                        {selectedEndpoint.parameters.map((param) => (
                          <Group key={param.name} gap="xs">
                            <Code style={{ fontSize: 11 }}>{param.name}</Code>
                            <Text size="xs" c="dimmed">
                              ({param.type}) - {param.description}
                            </Text>
                          </Group>
                        ))}
                      </Stack>
                    </Box>
                  )}
                </Box>
              )}
            </Box>
          </ScrollArea>
        </Box>

        {/* Right Panel: Response & Code */}
        <Box
          style={{
            borderLeft: '1px solid var(--supabase-border)',
            display: 'flex',
            flexDirection: 'column',
            overflow: 'hidden',
          }}
        >
          <Tabs defaultValue="response" style={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
            <Tabs.List px="sm">
              <Tabs.Tab value="response">Response</Tabs.Tab>
              <Tabs.Tab value="code">Code</Tabs.Tab>
            </Tabs.List>

            <Tabs.Panel value="response" style={{ flex: 1, overflow: 'hidden' }}>
              <ScrollArea style={{ height: '100%' }}>
                <Box p="sm">
                  {isLoading && (
                    <Group justify="center" py="xl">
                      <Loader size="sm" />
                      <Text size="sm" c="dimmed">
                        Sending request...
                      </Text>
                    </Group>
                  )}

                  {error && (
                    <Alert color="red" icon={<IconAlertCircle size={16} />}>
                      {error}
                    </Alert>
                  )}

                  {response && !isLoading && (
                    <Stack gap="md">
                      {/* Status Bar */}
                      <Group gap="md">
                        <Badge
                          size="lg"
                          color={getStatusColor(response.status)}
                          leftSection={
                            response.status < 400 ? (
                              <IconCircleCheck size={14} />
                            ) : (
                              <IconAlertCircle size={14} />
                            )
                          }
                        >
                          {response.status} {response.statusText}
                        </Badge>
                        <Group gap="xs">
                          <IconClock size={14} />
                          <Text size="sm" c="dimmed">
                            {response.duration_ms}ms
                          </Text>
                        </Group>
                      </Group>

                      {/* Response Body */}
                      <Box>
                        <Group justify="space-between" mb="xs">
                          <Text size="xs" fw={500}>
                            Response Body
                          </Text>
                          <CopyButton
                            value={
                              typeof response.body === 'string'
                                ? response.body
                                : JSON.stringify(response.body, null, 2)
                            }
                          >
                            {({ copied, copy }) => (
                              <ActionIcon size="sm" variant="subtle" onClick={copy}>
                                {copied ? <IconCheck size={14} /> : <IconCopy size={14} />}
                              </ActionIcon>
                            )}
                          </CopyButton>
                        </Group>
                        <Box
                          style={{
                            border: '1px solid var(--supabase-border)',
                            borderRadius: 6,
                            overflow: 'hidden',
                          }}
                        >
                          <Editor
                            height="300px"
                            language="json"
                            value={
                              typeof response.body === 'string'
                                ? response.body
                                : JSON.stringify(response.body, null, 2)
                            }
                            theme="vs-dark"
                            options={{
                              readOnly: true,
                              minimap: { enabled: false },
                              scrollBeyondLastLine: false,
                              fontSize: 12,
                              lineNumbers: 'on',
                              folding: true,
                              padding: { top: 8, bottom: 8 },
                              wordWrap: 'on',
                            }}
                          />
                        </Box>
                      </Box>

                      {/* Response Headers */}
                      <Box>
                        <Text size="xs" fw={500} mb="xs">
                          Response Headers
                        </Text>
                        <Stack gap={4}>
                          {Object.entries(response.headers).map(([key, value]) => (
                            <Group key={key} gap="xs" wrap="nowrap">
                              <Text size="xs" fw={500} style={{ minWidth: 120 }}>
                                {key}:
                              </Text>
                              <Text
                                size="xs"
                                c="dimmed"
                                style={{
                                  fontFamily: 'monospace',
                                  wordBreak: 'break-all',
                                }}
                              >
                                {value}
                              </Text>
                            </Group>
                          ))}
                        </Stack>
                      </Box>
                    </Stack>
                  )}

                  {!response && !isLoading && !error && (
                    <Box ta="center" py="xl">
                      <IconApi size={48} color="var(--mantine-color-gray-5)" />
                      <Text size="sm" c="dimmed" mt="md">
                        Select an endpoint or send a request to see the response
                      </Text>
                    </Box>
                  )}
                </Box>
              </ScrollArea>
            </Tabs.Panel>

            <Tabs.Panel value="code" style={{ flex: 1, overflow: 'hidden' }}>
              <ScrollArea style={{ height: '100%' }}>
                <Box p="sm">
                  <SegmentedControl
                    size="xs"
                    value={codeLanguage}
                    onChange={(v) => setCodeLanguage(v as CodeLanguage)}
                    data={[
                      {
                        label: (
                          <Group gap={4}>
                            <IconBrandJavascript size={14} />
                            <span>JavaScript</span>
                          </Group>
                        ),
                        value: 'javascript',
                      },
                      {
                        label: (
                          <Group gap={4}>
                            <IconTerminal size={14} />
                            <span>cURL</span>
                          </Group>
                        ),
                        value: 'curl',
                      },
                      {
                        label: (
                          <Group gap={4}>
                            <IconBrandPython size={14} />
                            <span>Python</span>
                          </Group>
                        ),
                        value: 'python',
                      },
                      {
                        label: (
                          <Group gap={4}>
                            <IconBrandGolang size={14} />
                            <span>Go</span>
                          </Group>
                        ),
                        value: 'go',
                      },
                    ]}
                    mb="md"
                    fullWidth
                  />
                  <Box
                    style={{
                      border: '1px solid var(--supabase-border)',
                      borderRadius: 6,
                      overflow: 'hidden',
                    }}
                  >
                    <CodeSnippet
                      language={codeLanguage}
                      code={generatedCode[codeLanguage]}
                    />
                  </Box>
                </Box>
              </ScrollArea>
            </Tabs.Panel>
          </Tabs>
        </Box>
      </Box>

      <style>{`
        .endpoint-item:hover {
          background-color: var(--supabase-bg-hover);
        }
      `}</style>
    </PageContainer>
  );
}
