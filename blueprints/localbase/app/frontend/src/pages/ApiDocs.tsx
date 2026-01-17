import {
  Box,
  Stack,
  Text,
  Card,
  Accordion,
  Badge,
  Code,
  Group,
  CopyButton,
  ActionIcon,
  Tooltip,
} from '@mantine/core';
import { IconCopy, IconCheck, IconApi } from '@tabler/icons-react';
import { PageContainer } from '../components/layout/PageContainer';

interface EndpointProps {
  method: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE';
  path: string;
  description: string;
  example?: string;
}

function Endpoint({ method, path, description, example }: EndpointProps) {
  const methodColors: Record<string, string> = {
    GET: 'blue',
    POST: 'green',
    PUT: 'orange',
    PATCH: 'yellow',
    DELETE: 'red',
  };

  return (
    <Box
      p="sm"
      style={{
        borderBottom: '1px solid var(--supabase-border)',
      }}
    >
      <Group gap="sm" mb="xs">
        <Badge size="sm" variant="filled" color={methodColors[method]}>
          {method}
        </Badge>
        <Code style={{ flex: 1 }}>{path}</Code>
        <CopyButton value={path}>
          {({ copied, copy }) => (
            <Tooltip label={copied ? 'Copied!' : 'Copy path'}>
              <ActionIcon variant="subtle" size="sm" onClick={copy}>
                {copied ? <IconCheck size={14} /> : <IconCopy size={14} />}
              </ActionIcon>
            </Tooltip>
          )}
        </CopyButton>
      </Group>
      <Text size="sm" c="dimmed">
        {description}
      </Text>
      {example && (
        <Code block mt="xs" style={{ fontSize: 11 }}>
          {example}
        </Code>
      )}
    </Box>
  );
}

export function ApiDocsPage() {
  const baseUrl = window.location.origin;

  return (
    <PageContainer title="API Documentation" description="Reference for Localbase REST APIs">
      <Stack gap="lg">
        {/* Base URL */}
        <Card className="supabase-section">
          <Group gap="xs" mb="sm">
            <IconApi size={20} />
            <Text fw={600}>Base URL</Text>
          </Group>
          <Group gap="sm">
            <Code style={{ flex: 1 }}>{baseUrl}</Code>
            <CopyButton value={baseUrl}>
              {({ copied, copy }) => (
                <Tooltip label={copied ? 'Copied!' : 'Copy'}>
                  <ActionIcon variant="subtle" onClick={copy}>
                    {copied ? <IconCheck size={16} /> : <IconCopy size={16} />}
                  </ActionIcon>
                </Tooltip>
              )}
            </CopyButton>
          </Group>
        </Card>

        {/* Authentication */}
        <Card className="supabase-section">
          <Text fw={600} mb="sm">
            Authentication
          </Text>
          <Text size="sm" c="dimmed" mb="md">
            All API requests require an API key passed via the <Code>apikey</Code> header or{' '}
            <Code>Authorization: Bearer</Code> header.
          </Text>
          <Code block style={{ fontSize: 12 }}>
            {`curl -X GET '${baseUrl}/rest/v1/users' \\
  -H "apikey: YOUR_API_KEY" \\
  -H "Authorization: Bearer YOUR_API_KEY"`}
          </Code>
        </Card>

        {/* API Endpoints */}
        <Accordion variant="separated">
          {/* Auth API */}
          <Accordion.Item value="auth">
            <Accordion.Control>
              <Group gap="xs">
                <Badge variant="light" color="blue">
                  Auth
                </Badge>
                <Text fw={500}>/auth/v1</Text>
              </Group>
            </Accordion.Control>
            <Accordion.Panel>
              <Stack gap={0}>
                <Endpoint
                  method="POST"
                  path="/auth/v1/signup"
                  description="Register a new user"
                  example={`{
  "email": "user@example.com",
  "password": "password123"
}`}
                />
                <Endpoint
                  method="POST"
                  path="/auth/v1/token?grant_type=password"
                  description="Sign in with email and password"
                  example={`{
  "email": "user@example.com",
  "password": "password123"
}`}
                />
                <Endpoint
                  method="POST"
                  path="/auth/v1/logout"
                  description="Sign out the current user"
                />
                <Endpoint
                  method="GET"
                  path="/auth/v1/user"
                  description="Get the current user"
                />
                <Endpoint
                  method="PUT"
                  path="/auth/v1/user"
                  description="Update the current user"
                />
                <Endpoint
                  method="GET"
                  path="/auth/v1/admin/users"
                  description="List all users (requires service_role)"
                />
                <Endpoint
                  method="POST"
                  path="/auth/v1/admin/users"
                  description="Create a new user (requires service_role)"
                />
              </Stack>
            </Accordion.Panel>
          </Accordion.Item>

          {/* REST API */}
          <Accordion.Item value="rest">
            <Accordion.Control>
              <Group gap="xs">
                <Badge variant="light" color="violet">
                  Database
                </Badge>
                <Text fw={500}>/rest/v1</Text>
              </Group>
            </Accordion.Control>
            <Accordion.Panel>
              <Stack gap={0}>
                <Endpoint
                  method="GET"
                  path="/rest/v1/{table}"
                  description="Select rows from a table"
                  example="?select=*&limit=10"
                />
                <Endpoint
                  method="POST"
                  path="/rest/v1/{table}"
                  description="Insert rows into a table"
                  example={`{
  "column1": "value1",
  "column2": "value2"
}`}
                />
                <Endpoint
                  method="PATCH"
                  path="/rest/v1/{table}?{filters}"
                  description="Update rows in a table"
                  example="?id=eq.123"
                />
                <Endpoint
                  method="DELETE"
                  path="/rest/v1/{table}?{filters}"
                  description="Delete rows from a table"
                  example="?id=eq.123"
                />
                <Endpoint
                  method="POST"
                  path="/rest/v1/rpc/{function}"
                  description="Call a PostgreSQL function"
                />
              </Stack>
            </Accordion.Panel>
          </Accordion.Item>

          {/* Storage API */}
          <Accordion.Item value="storage">
            <Accordion.Control>
              <Group gap="xs">
                <Badge variant="light" color="orange">
                  Storage
                </Badge>
                <Text fw={500}>/storage/v1</Text>
              </Group>
            </Accordion.Control>
            <Accordion.Panel>
              <Stack gap={0}>
                <Endpoint
                  method="GET"
                  path="/storage/v1/bucket"
                  description="List all buckets"
                />
                <Endpoint
                  method="POST"
                  path="/storage/v1/bucket"
                  description="Create a new bucket"
                  example={`{
  "name": "my-bucket",
  "public": false
}`}
                />
                <Endpoint
                  method="POST"
                  path="/storage/v1/object/list/{bucket}"
                  description="List objects in a bucket"
                />
                <Endpoint
                  method="POST"
                  path="/storage/v1/object/{bucket}/{path}"
                  description="Upload an object"
                />
                <Endpoint
                  method="GET"
                  path="/storage/v1/object/{bucket}/{path}"
                  description="Download an object"
                />
                <Endpoint
                  method="DELETE"
                  path="/storage/v1/object/{bucket}/{path}"
                  description="Delete an object"
                />
                <Endpoint
                  method="GET"
                  path="/storage/v1/object/public/{bucket}/{path}"
                  description="Get public URL for an object"
                />
              </Stack>
            </Accordion.Panel>
          </Accordion.Item>

          {/* Realtime API */}
          <Accordion.Item value="realtime">
            <Accordion.Control>
              <Group gap="xs">
                <Badge variant="light" color="cyan">
                  Realtime
                </Badge>
                <Text fw={500}>/realtime/v1</Text>
              </Group>
            </Accordion.Control>
            <Accordion.Panel>
              <Stack gap={0}>
                <Endpoint
                  method="GET"
                  path="/realtime/v1/websocket"
                  description="WebSocket endpoint for realtime subscriptions"
                  example="?apikey=YOUR_API_KEY"
                />
                <Endpoint
                  method="GET"
                  path="/api/realtime/channels"
                  description="List active channels (requires service_role)"
                />
                <Endpoint
                  method="GET"
                  path="/api/realtime/stats"
                  description="Get realtime statistics (requires service_role)"
                />
              </Stack>
            </Accordion.Panel>
          </Accordion.Item>

          {/* Functions API */}
          <Accordion.Item value="functions">
            <Accordion.Control>
              <Group gap="xs">
                <Badge variant="light" color="green">
                  Functions
                </Badge>
                <Text fw={500}>/functions/v1</Text>
              </Group>
            </Accordion.Control>
            <Accordion.Panel>
              <Stack gap={0}>
                <Endpoint
                  method="POST"
                  path="/functions/v1/{function_name}"
                  description="Invoke an edge function"
                />
                <Endpoint
                  method="GET"
                  path="/api/functions"
                  description="List all functions (requires service_role)"
                />
                <Endpoint
                  method="POST"
                  path="/api/functions"
                  description="Create a function (requires service_role)"
                />
                <Endpoint
                  method="POST"
                  path="/api/functions/{id}/deploy"
                  description="Deploy a function (requires service_role)"
                />
              </Stack>
            </Accordion.Panel>
          </Accordion.Item>
        </Accordion>
      </Stack>
    </PageContainer>
  );
}
