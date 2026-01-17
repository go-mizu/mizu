import { useEffect, useState, useCallback } from 'react';
import {
  Button,
  Modal,
  TextInput,
  Switch,
  Stack,
  Group,
  Text,
  ActionIcon,
  Menu,
  Badge,
  Card,
} from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import {
  IconPlus,
  IconDotsVertical,
  IconTrash,
  IconCode,
  IconRocket,
  IconPlayerPlay,
} from '@tabler/icons-react';
import { notifications } from '@mantine/notifications';
import { PageContainer } from '../../components/layout/PageContainer';
import { DataTable, type Column } from '../../components/common/DataTable';
import { EmptyState } from '../../components/common/EmptyState';
import { ConfirmModal } from '../../components/common/ConfirmModal';
import { StatusBadge } from '../../components/common/StatusBadge';
import { functionsApi } from '../../api';
import type { EdgeFunction } from '../../types';

export function FunctionsPage() {
  const [functions, setFunctions] = useState<EdgeFunction[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedFunction, setSelectedFunction] = useState<EdgeFunction | null>(null);

  const [createOpened, { open: openCreate, close: closeCreate }] = useDisclosure(false);
  const [deleteOpened, { open: openDelete, close: closeDelete }] = useDisclosure(false);

  const [formData, setFormData] = useState({
    name: '',
    verify_jwt: true,
  });
  const [formLoading, setFormLoading] = useState(false);

  const fetchFunctions = useCallback(async () => {
    setLoading(true);
    try {
      const data = await functionsApi.listFunctions();
      setFunctions(data);
    } catch (error: any) {
      notifications.show({
        title: 'Error',
        message: error.message || 'Failed to load functions',
        color: 'red',
      });
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchFunctions();
  }, [fetchFunctions]);

  const handleCreate = async () => {
    if (!formData.name.trim()) {
      notifications.show({
        title: 'Validation Error',
        message: 'Function name is required',
        color: 'red',
      });
      return;
    }

    setFormLoading(true);
    try {
      await functionsApi.createFunction({
        name: formData.name,
        verify_jwt: formData.verify_jwt,
      });
      notifications.show({
        title: 'Success',
        message: 'Function created successfully',
        color: 'green',
      });
      closeCreate();
      setFormData({ name: '', verify_jwt: true });
      fetchFunctions();
    } catch (error: any) {
      notifications.show({
        title: 'Error',
        message: error.message || 'Failed to create function',
        color: 'red',
      });
    } finally {
      setFormLoading(false);
    }
  };

  const handleDelete = async () => {
    if (!selectedFunction) return;

    setFormLoading(true);
    try {
      await functionsApi.deleteFunction(selectedFunction.id);
      notifications.show({
        title: 'Success',
        message: 'Function deleted successfully',
        color: 'green',
      });
      closeDelete();
      setSelectedFunction(null);
      fetchFunctions();
    } catch (error: any) {
      notifications.show({
        title: 'Error',
        message: error.message || 'Failed to delete function',
        color: 'red',
      });
    } finally {
      setFormLoading(false);
    }
  };

  const handleDeploy = async (fn: EdgeFunction) => {
    try {
      await functionsApi.deployFunction(fn.id, {
        source_code: `// Edge function: ${fn.name}
export default async function(req) {
  return new Response(JSON.stringify({ message: "Hello from ${fn.name}!" }), {
    headers: { "Content-Type": "application/json" },
  });
}`,
      });
      notifications.show({
        title: 'Success',
        message: 'Function deployed successfully',
        color: 'green',
      });
      fetchFunctions();
    } catch (error: any) {
      notifications.show({
        title: 'Error',
        message: error.message || 'Failed to deploy function',
        color: 'red',
      });
    }
  };

  const handleInvoke = async (fn: EdgeFunction) => {
    try {
      const result = await functionsApi.invokeFunction(fn.slug || fn.name);
      notifications.show({
        title: 'Function Response',
        message: JSON.stringify(result),
        color: 'blue',
      });
    } catch (error: any) {
      notifications.show({
        title: 'Invocation Error',
        message: error.message || 'Failed to invoke function',
        color: 'red',
      });
    }
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  const columns: Column<EdgeFunction>[] = [
    {
      key: 'name',
      header: 'Name',
      render: (fn) => (
        <Group gap="xs">
          <IconCode size={16} />
          <Text size="sm" fw={500}>
            {fn.name}
          </Text>
        </Group>
      ),
    },
    {
      key: 'slug',
      header: 'Slug',
      render: (fn) => (
        <Text size="sm" c="dimmed">
          {fn.slug || fn.name}
        </Text>
      ),
    },
    {
      key: 'status',
      header: 'Status',
      render: (fn) => <StatusBadge status={fn.status} />,
    },
    {
      key: 'version',
      header: 'Version',
      render: (fn) => (
        <Badge size="sm" variant="light" color="gray">
          v{fn.version}
        </Badge>
      ),
    },
    {
      key: 'verify_jwt',
      header: 'JWT',
      render: (fn) => (
        <Badge size="sm" variant="light" color={fn.verify_jwt ? 'green' : 'yellow'}>
          {fn.verify_jwt ? 'Required' : 'Optional'}
        </Badge>
      ),
    },
    {
      key: 'updated_at',
      header: 'Last Updated',
      render: (fn) => (
        <Text size="sm" c="dimmed">
          {formatDate(fn.updated_at)}
        </Text>
      ),
    },
    {
      key: 'actions',
      header: '',
      width: 100,
      render: (fn) => (
        <Group gap="xs" wrap="nowrap">
          <ActionIcon
            variant="subtle"
            color="blue"
            onClick={(e) => {
              e.stopPropagation();
              handleInvoke(fn);
            }}
            title="Invoke"
          >
            <IconPlayerPlay size={16} />
          </ActionIcon>
          <Menu position="bottom-end" withinPortal>
            <Menu.Target>
              <ActionIcon variant="subtle" color="gray" onClick={(e) => e.stopPropagation()}>
                <IconDotsVertical size={16} />
              </ActionIcon>
            </Menu.Target>
            <Menu.Dropdown>
              <Menu.Item
                leftSection={<IconRocket size={14} />}
                onClick={(e) => {
                  e.stopPropagation();
                  handleDeploy(fn);
                }}
              >
                Deploy
              </Menu.Item>
              <Menu.Divider />
              <Menu.Item
                color="red"
                leftSection={<IconTrash size={14} />}
                onClick={(e) => {
                  e.stopPropagation();
                  setSelectedFunction(fn);
                  openDelete();
                }}
              >
                Delete
              </Menu.Item>
            </Menu.Dropdown>
          </Menu>
        </Group>
      ),
    },
  ];

  return (
    <PageContainer
      title="Edge Functions"
      description="Deploy and manage serverless functions"
      actions={
        <Button leftSection={<IconPlus size={16} />} onClick={openCreate}>
          Create function
        </Button>
      }
    >
      <Card className="supabase-section" p={0}>
        <DataTable
          data={functions}
          columns={columns}
          loading={loading}
          getRowKey={(fn) => fn.id}
          emptyState={
            <EmptyState
              icon={<IconCode size={32} />}
              title="No functions yet"
              description="Create your first edge function to get started"
              action={{
                label: 'Create function',
                onClick: openCreate,
              }}
            />
          }
        />
      </Card>

      {/* Create Function Modal */}
      <Modal opened={createOpened} onClose={closeCreate} title="Create function" size="md">
        <Stack gap="md">
          <TextInput
            label="Function name"
            placeholder="my-function"
            value={formData.name}
            onChange={(e) => setFormData({ ...formData, name: e.target.value })}
            required
          />
          <Switch
            label="Require JWT verification"
            description="Only allow authenticated requests to invoke this function"
            checked={formData.verify_jwt}
            onChange={(e) => setFormData({ ...formData, verify_jwt: e.target.checked })}
          />
          <Group justify="flex-end" mt="md">
            <Button variant="outline" onClick={closeCreate}>
              Cancel
            </Button>
            <Button onClick={handleCreate} loading={formLoading}>
              Create function
            </Button>
          </Group>
        </Stack>
      </Modal>

      {/* Delete Confirmation Modal */}
      <ConfirmModal
        opened={deleteOpened}
        onClose={closeDelete}
        onConfirm={handleDelete}
        title="Delete function"
        message={`Are you sure you want to delete "${selectedFunction?.name}"? This action cannot be undone.`}
        confirmLabel="Delete"
        danger
        loading={formLoading}
      />
    </PageContainer>
  );
}
