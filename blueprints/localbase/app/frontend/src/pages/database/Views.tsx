import { useEffect, useState, useCallback } from 'react';
import {
  Box,
  Button,
  Group,
  Text,
  Stack,
  Paper,
  Select,
  ActionIcon,
  Badge,
  Modal,
  TextInput,
  Textarea,
  Loader,
  Center,
  Code,
  Tabs,
  Tooltip,
  Collapse,
} from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import {
  IconPlus,
  IconTrash,
  IconRefresh,
  IconEye,
  IconDatabase,
  IconChevronDown,
  IconChevronRight,
} from '@tabler/icons-react';
import { notifications } from '@mantine/notifications';
import { PageContainer } from '../../components/layout/PageContainer';
import { EmptyState } from '../../components/common/EmptyState';
import { ConfirmModal } from '../../components/common/ConfirmModal';
import { pgmetaApi, databaseApi } from '../../api';
import type { PGView } from '../../api/pgmeta';

export function ViewsPage() {
  const [schema, setSchema] = useState('public');
  const [schemas, setSchemas] = useState<string[]>([]);
  const [views, setViews] = useState<PGView[]>([]);
  const [materializedViews, setMaterializedViews] = useState<PGView[]>([]);
  const [loading, setLoading] = useState(true);
  const [expandedViews, setExpandedViews] = useState<Set<string>>(new Set());

  const [createModalOpened, { open: openCreateModal, close: closeCreateModal }] =
    useDisclosure(false);
  const [deleteModalOpened, { open: openDeleteModal, close: closeDeleteModal }] =
    useDisclosure(false);

  const [selectedView, setSelectedView] = useState<PGView | null>(null);
  const [viewType, setViewType] = useState<'regular' | 'materialized'>('regular');

  // Form state
  const [viewName, setViewName] = useState('');
  const [viewDefinition, setViewDefinition] = useState('');
  const [formLoading, setFormLoading] = useState(false);

  const fetchSchemas = useCallback(async () => {
    try {
      const data = await databaseApi.listSchemas();
      setSchemas(data || []);
    } catch (error: any) {
      notifications.show({
        title: 'Error',
        message: error.message || 'Failed to load schemas',
        color: 'red',
      });
    }
  }, []);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const [viewsData, mvData] = await Promise.all([
        pgmetaApi.listViews(schema),
        pgmetaApi.listMaterializedViews(schema),
      ]);
      setViews(viewsData || []);
      setMaterializedViews(mvData || []);
    } catch (error: any) {
      notifications.show({
        title: 'Error',
        message: error.message || 'Failed to load views',
        color: 'red',
      });
    } finally {
      setLoading(false);
    }
  }, [schema]);

  useEffect(() => {
    fetchSchemas();
  }, [fetchSchemas]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const toggleExpand = (viewName: string) => {
    setExpandedViews((prev) => {
      const newSet = new Set(prev);
      if (newSet.has(viewName)) {
        newSet.delete(viewName);
      } else {
        newSet.add(viewName);
      }
      return newSet;
    });
  };

  const handleCreateView = async () => {
    if (!viewName.trim() || !viewDefinition.trim()) {
      notifications.show({
        title: 'Validation Error',
        message: 'View name and definition are required',
        color: 'red',
      });
      return;
    }

    setFormLoading(true);
    try {
      if (viewType === 'materialized') {
        await pgmetaApi.createMaterializedView({
          schema,
          name: viewName,
          definition: viewDefinition,
        });
      } else {
        await pgmetaApi.createView({
          schema,
          name: viewName,
          definition: viewDefinition,
        });
      }

      notifications.show({
        title: 'Success',
        message: `${viewType === 'materialized' ? 'Materialized view' : 'View'} created successfully`,
        color: 'green',
      });

      closeCreateModal();
      resetForm();
      fetchData();
    } catch (error: any) {
      notifications.show({
        title: 'Error',
        message: error.message || 'Failed to create view',
        color: 'red',
      });
    } finally {
      setFormLoading(false);
    }
  };

  const handleDeleteView = async () => {
    if (!selectedView) return;

    setFormLoading(true);
    try {
      if (selectedView.is_materialized) {
        await pgmetaApi.dropMaterializedView(schema, selectedView.name);
      } else {
        await pgmetaApi.dropView(schema, selectedView.name);
      }

      notifications.show({
        title: 'Success',
        message: 'View deleted successfully',
        color: 'green',
      });

      closeDeleteModal();
      setSelectedView(null);
      fetchData();
    } catch (error: any) {
      notifications.show({
        title: 'Error',
        message: error.message || 'Failed to delete view',
        color: 'red',
      });
    } finally {
      setFormLoading(false);
    }
  };

  const handleRefreshMaterializedView = async (view: PGView) => {
    try {
      await pgmetaApi.refreshMaterializedView(schema, view.name);
      notifications.show({
        title: 'Success',
        message: `Materialized view "${view.name}" refreshed successfully`,
        color: 'green',
      });
    } catch (error: any) {
      notifications.show({
        title: 'Error',
        message: error.message || 'Failed to refresh materialized view',
        color: 'red',
      });
    }
  };

  const resetForm = () => {
    setViewName('');
    setViewDefinition('');
    setViewType('regular');
  };

  const openDelete = (view: PGView) => {
    setSelectedView(view);
    openDeleteModal();
  };

  const renderViewRow = (view: PGView, isMaterialized: boolean) => {
    const isExpanded = expandedViews.has(view.name);

    return (
      <Box key={view.name}>
        <Paper
          p="md"
          withBorder
          style={{ cursor: 'pointer' }}
          onClick={() => toggleExpand(view.name)}
        >
          <Group justify="space-between">
            <Group gap="md">
              {isExpanded ? <IconChevronDown size={18} /> : <IconChevronRight size={18} />}
              <IconEye size={18} color="var(--mantine-color-blue-6)" />
              <Text fw={500}>{view.name}</Text>
              {isMaterialized && (
                <Badge size="sm" variant="light" color="orange">
                  Materialized
                </Badge>
              )}
              <Badge size="sm" variant="outline">
                {view.columns?.length ?? 0} columns
              </Badge>
            </Group>
            <Group gap="sm" onClick={(e) => e.stopPropagation()}>
              {isMaterialized && (
                <Tooltip label="Refresh data">
                  <ActionIcon
                    variant="subtle"
                    color="blue"
                    onClick={() => handleRefreshMaterializedView(view)}
                  >
                    <IconRefresh size={16} />
                  </ActionIcon>
                </Tooltip>
              )}
              <ActionIcon
                variant="subtle"
                color="red"
                onClick={() => openDelete({ ...view, is_materialized: isMaterialized })}
              >
                <IconTrash size={16} />
              </ActionIcon>
            </Group>
          </Group>
        </Paper>

        <Collapse in={isExpanded}>
          <Box p="md" ml="xl" style={{ borderLeft: '2px solid var(--mantine-color-gray-3)' }}>
            <Text size="sm" fw={500} mb="xs">
              Columns
            </Text>
            {view.columns && Array.isArray(view.columns) && view.columns.length > 0 ? (
              <Group gap="xs" mb="md">
                {view.columns.map((col) => (
                  <Badge key={col.name} variant="outline">
                    {col.name}: {col.type}
                  </Badge>
                ))}
              </Group>
            ) : (
              <Text size="sm" c="dimmed" mb="md">
                No column information available
              </Text>
            )}

            <Text size="sm" fw={500} mb="xs">
              Definition
            </Text>
            <Code block style={{ maxHeight: 200, overflow: 'auto' }}>
              {view.definition || 'Definition not available'}
            </Code>
          </Box>
        </Collapse>
      </Box>
    );
  };

  return (
    <PageContainer
      title="Views"
      description="Manage database views and materialized views"
    >
      {/* Header */}
      <Group justify="space-between" mb="lg">
        <Group gap="md">
          <Select
            size="sm"
            value={schema}
            onChange={(value) => value && setSchema(value)}
            data={schemas.map((s) => ({ value: s, label: s }))}
            placeholder="Select schema"
            w={150}
          />
          <Badge variant="light" color="blue" size="lg">
            {views?.length ?? 0} views
          </Badge>
          <Badge variant="light" color="orange" size="lg">
            {materializedViews?.length ?? 0} materialized
          </Badge>
        </Group>
        <Group gap="sm">
          <ActionIcon variant="subtle" onClick={fetchData}>
            <IconRefresh size={18} />
          </ActionIcon>
          <Button leftSection={<IconPlus size={16} />} onClick={openCreateModal}>
            New View
          </Button>
        </Group>
      </Group>

      {/* Content */}
      {loading ? (
        <Center py="xl">
          <Loader size="lg" />
        </Center>
      ) : (!views || views.length === 0) && (!materializedViews || materializedViews.length === 0) ? (
        <EmptyState
          icon={<IconEye size={48} />}
          title="No views found"
          description="Create views to simplify complex queries"
          action={{
            label: 'Create View',
            onClick: openCreateModal,
          }}
        />
      ) : (
        <Tabs defaultValue="regular">
          <Tabs.List mb="md">
            <Tabs.Tab value="regular" leftSection={<IconEye size={16} />}>
              Views ({views?.length ?? 0})
            </Tabs.Tab>
            <Tabs.Tab value="materialized" leftSection={<IconDatabase size={16} />}>
              Materialized Views ({materializedViews?.length ?? 0})
            </Tabs.Tab>
          </Tabs.List>

          <Tabs.Panel value="regular">
            {!views || views.length === 0 ? (
              <Text c="dimmed" ta="center" py="xl">
                No regular views found
              </Text>
            ) : (
              <Stack gap="sm">
                {views.map((view) => renderViewRow(view, false))}
              </Stack>
            )}
          </Tabs.Panel>

          <Tabs.Panel value="materialized">
            {!materializedViews || materializedViews.length === 0 ? (
              <Text c="dimmed" ta="center" py="xl">
                No materialized views found
              </Text>
            ) : (
              <Stack gap="sm">
                {materializedViews.map((view) => renderViewRow(view, true))}
              </Stack>
            )}
          </Tabs.Panel>
        </Tabs>
      )}

      {/* Create View Modal */}
      <Modal
        opened={createModalOpened}
        onClose={closeCreateModal}
        title="Create view"
        size="lg"
      >
        <Stack gap="md">
          <Select
            label="View type"
            value={viewType}
            onChange={(value) => value && setViewType(value as 'regular' | 'materialized')}
            data={[
              { value: 'regular', label: 'Regular View' },
              { value: 'materialized', label: 'Materialized View' },
            ]}
          />

          {viewType === 'materialized' && (
            <Text size="sm" c="dimmed">
              Materialized views store the query result and must be refreshed manually to update data.
            </Text>
          )}

          <TextInput
            label="View name"
            placeholder="my_view"
            value={viewName}
            onChange={(e) => setViewName(e.target.value)}
            required
          />

          <Textarea
            label="SQL Definition"
            description="The SELECT query that defines the view"
            placeholder="SELECT * FROM users WHERE active = true"
            value={viewDefinition}
            onChange={(e) => setViewDefinition(e.target.value)}
            minRows={6}
            required
            styles={{
              input: { fontFamily: 'monospace' },
            }}
          />

          <Group justify="flex-end" mt="md">
            <Button variant="outline" onClick={closeCreateModal}>
              Cancel
            </Button>
            <Button onClick={handleCreateView} loading={formLoading}>
              Create View
            </Button>
          </Group>
        </Stack>
      </Modal>

      {/* Delete Confirmation */}
      <ConfirmModal
        opened={deleteModalOpened}
        onClose={closeDeleteModal}
        onConfirm={handleDeleteView}
        title="Delete view"
        message={`Are you sure you want to delete the ${selectedView?.is_materialized ? 'materialized view' : 'view'} "${selectedView?.name}"? This action cannot be undone.`}
        confirmLabel="Delete"
        danger
        loading={formLoading}
      />
    </PageContainer>
  );
}
