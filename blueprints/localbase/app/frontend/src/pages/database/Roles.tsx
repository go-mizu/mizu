import { useEffect, useState, useCallback } from 'react';
import {
  Button,
  Group,
  Text,
  Stack,
  Paper,
  Table,
  ActionIcon,
  Badge,
  Modal,
  TextInput,
  PasswordInput,
  Loader,
  Center,
  Checkbox,
  NumberInput,
  Tooltip,
} from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import {
  IconPlus,
  IconTrash,
  IconRefresh,
  IconUserShield,
  IconShieldCheck,
  IconDatabase,
  IconUserPlus,
} from '@tabler/icons-react';
import { notifications } from '@mantine/notifications';
import { PageContainer } from '../../components/layout/PageContainer';
import { EmptyState } from '../../components/common/EmptyState';
import { ConfirmModal } from '../../components/common/ConfirmModal';
import { pgmetaApi } from '../../api';
import type { PGRole } from '../../api/pgmeta';

// System roles that should not be deleted
const SYSTEM_ROLES = ['postgres', 'pg_database_owner', 'pg_read_all_data', 'pg_write_all_data'];

export function RolesPage() {
  const [roles, setRoles] = useState<PGRole[]>([]);
  const [loading, setLoading] = useState(true);

  const [createModalOpened, { open: openCreateModal, close: closeCreateModal }] =
    useDisclosure(false);
  const [deleteModalOpened, { open: openDeleteModal, close: closeDeleteModal }] =
    useDisclosure(false);

  const [selectedRole, setSelectedRole] = useState<PGRole | null>(null);

  // Form state
  const [roleName, setRoleName] = useState('');
  const [rolePassword, setRolePassword] = useState('');
  const [roleCanLogin, setRoleCanLogin] = useState(true);
  const [roleCanCreateDB, setRoleCanCreateDB] = useState(false);
  const [roleCanCreateRole, setRoleCanCreateRole] = useState(false);
  const [roleIsSuperuser, setRoleIsSuperuser] = useState(false);
  const [roleInherit, setRoleInherit] = useState(true);
  const [roleConnectionLimit, setRoleConnectionLimit] = useState<number | undefined>(undefined);
  const [formLoading, setFormLoading] = useState(false);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const rolesData = await pgmetaApi.listRoles();
      setRoles(rolesData ?? []);
    } catch (error: any) {
      notifications.show({
        title: 'Error',
        message: error.message || 'Failed to load roles',
        color: 'red',
      });
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const handleCreateRole = async () => {
    if (!roleName.trim()) {
      notifications.show({
        title: 'Validation Error',
        message: 'Role name is required',
        color: 'red',
      });
      return;
    }

    setFormLoading(true);
    try {
      await pgmetaApi.createRole({
        name: roleName,
        password: rolePassword || undefined,
        can_login: roleCanLogin,
        can_create_db: roleCanCreateDB,
        can_create_role: roleCanCreateRole,
        is_superuser: roleIsSuperuser,
        inherit: roleInherit,
        connection_limit: roleConnectionLimit,
      });

      notifications.show({
        title: 'Success',
        message: 'Role created successfully',
        color: 'green',
      });

      closeCreateModal();
      resetForm();
      fetchData();
    } catch (error: any) {
      notifications.show({
        title: 'Error',
        message: error.message || 'Failed to create role',
        color: 'red',
      });
    } finally {
      setFormLoading(false);
    }
  };

  const handleDeleteRole = async () => {
    if (!selectedRole) return;

    setFormLoading(true);
    try {
      await pgmetaApi.dropRole(selectedRole.name);

      notifications.show({
        title: 'Success',
        message: 'Role deleted successfully',
        color: 'green',
      });

      closeDeleteModal();
      setSelectedRole(null);
      fetchData();
    } catch (error: any) {
      notifications.show({
        title: 'Error',
        message: error.message || 'Failed to delete role',
        color: 'red',
      });
    } finally {
      setFormLoading(false);
    }
  };

  const resetForm = () => {
    setRoleName('');
    setRolePassword('');
    setRoleCanLogin(true);
    setRoleCanCreateDB(false);
    setRoleCanCreateRole(false);
    setRoleIsSuperuser(false);
    setRoleInherit(true);
    setRoleConnectionLimit(undefined);
  };

  const openDelete = (role: PGRole) => {
    setSelectedRole(role);
    openDeleteModal();
  };

  const isSystemRole = (roleName: string) => {
    return SYSTEM_ROLES.includes(roleName) || roleName.startsWith('pg_');
  };

  // Separate system and custom roles
  const systemRoles = (roles ?? []).filter((r) => isSystemRole(r.name));
  const customRoles = (roles ?? []).filter((r) => !isSystemRole(r.name));

  const renderRoleRow = (role: PGRole, canDelete: boolean) => (
    <Table.Tr key={role.name}>
      <Table.Td>
        <Group gap="xs">
          {role.is_superuser ? (
            <Tooltip label="Superuser">
              <IconShieldCheck size={16} color="var(--mantine-color-red-6)" />
            </Tooltip>
          ) : role.can_login ? (
            <Tooltip label="Login role">
              <IconUserShield size={16} color="var(--mantine-color-blue-6)" />
            </Tooltip>
          ) : (
            <Tooltip label="Group role">
              <IconUserPlus size={16} color="var(--mantine-color-gray-6)" />
            </Tooltip>
          )}
          <Text size="sm" fw={500}>
            {role.name}
          </Text>
        </Group>
      </Table.Td>
      <Table.Td>
        <Group gap={4}>
          {role.is_superuser && (
            <Badge size="xs" color="red">
              Superuser
            </Badge>
          )}
          {role.can_login && (
            <Badge size="xs" color="blue">
              Login
            </Badge>
          )}
          {role.can_create_db && (
            <Badge size="xs" color="green">
              CreateDB
            </Badge>
          )}
          {role.can_create_role && (
            <Badge size="xs" color="grape">
              CreateRole
            </Badge>
          )}
          {role.can_bypass_rls && (
            <Badge size="xs" color="orange">
              BypassRLS
            </Badge>
          )}
          {role.is_replication_role && (
            <Badge size="xs" color="cyan">
              Replication
            </Badge>
          )}
        </Group>
      </Table.Td>
      <Table.Td>
        {role.connection_limit === -1 ? (
          <Text size="sm" c="dimmed">
            Unlimited
          </Text>
        ) : (
          <Text size="sm">{role.connection_limit}</Text>
        )}
      </Table.Td>
      <Table.Td>
        {role.active_connections > 0 ? (
          <Badge size="sm" color="green">
            {role.active_connections} active
          </Badge>
        ) : (
          <Text size="sm" c="dimmed">
            0
          </Text>
        )}
      </Table.Td>
      <Table.Td>
        {role.valid_until ? (
          <Text size="sm">{new Date(role.valid_until).toLocaleDateString()}</Text>
        ) : (
          <Text size="sm" c="dimmed">
            Never
          </Text>
        )}
      </Table.Td>
      <Table.Td>
        {canDelete && (
          <ActionIcon
            variant="subtle"
            color="red"
            onClick={() => openDelete(role)}
          >
            <IconTrash size={16} />
          </ActionIcon>
        )}
      </Table.Td>
    </Table.Tr>
  );

  return (
    <PageContainer
      title="Roles"
      description="Manage database roles and permissions"
    >
      {/* Header */}
      <Group justify="space-between" mb="lg">
        <Group gap="md">
          <Badge variant="light" color="blue" size="lg">
            {roles?.length ?? 0} roles
          </Badge>
          <Badge variant="light" color="green" size="lg">
            {customRoles?.length ?? 0} custom
          </Badge>
        </Group>
        <Group gap="sm">
          <ActionIcon variant="subtle" onClick={fetchData}>
            <IconRefresh size={18} />
          </ActionIcon>
          <Button leftSection={<IconPlus size={16} />} onClick={openCreateModal}>
            New Role
          </Button>
        </Group>
      </Group>

      {/* Content */}
      {loading ? (
        <Center py="xl">
          <Loader size="lg" />
        </Center>
      ) : !roles || roles.length === 0 ? (
        <EmptyState
          icon={<IconUserShield size={48} />}
          title="No roles found"
          description="Create roles to manage database access"
          action={{
            label: 'Create Role',
            onClick: openCreateModal,
          }}
        />
      ) : (
        <Stack gap="lg">
          {/* Custom Roles */}
          <Paper shadow="xs" p="md" withBorder>
            <Group gap="xs" mb="md">
              <IconUserShield size={20} />
              <Text fw={500}>Custom Roles</Text>
              <Badge size="sm" variant="light">
                {customRoles?.length ?? 0}
              </Badge>
            </Group>
            {!customRoles || customRoles.length === 0 ? (
              <Text c="dimmed" size="sm" ta="center" py="md">
                No custom roles. Create one to manage access.
              </Text>
            ) : (
              <Table striped highlightOnHover>
                <Table.Thead>
                  <Table.Tr>
                    <Table.Th>Name</Table.Th>
                    <Table.Th>Attributes</Table.Th>
                    <Table.Th>Connection Limit</Table.Th>
                    <Table.Th>Active Connections</Table.Th>
                    <Table.Th>Valid Until</Table.Th>
                    <Table.Th w={60}></Table.Th>
                  </Table.Tr>
                </Table.Thead>
                <Table.Tbody>
                  {customRoles.map((role) => renderRoleRow(role, true))}
                </Table.Tbody>
              </Table>
            )}
          </Paper>

          {/* System Roles */}
          <Paper shadow="xs" p="md" withBorder>
            <Group gap="xs" mb="md">
              <IconDatabase size={20} />
              <Text fw={500}>System Roles</Text>
              <Badge size="sm" variant="light" color="gray">
                {systemRoles?.length ?? 0}
              </Badge>
            </Group>
            <Text size="sm" c="dimmed" mb="md">
              System roles are managed by PostgreSQL and cannot be deleted.
            </Text>
            <Table striped highlightOnHover>
              <Table.Thead>
                <Table.Tr>
                  <Table.Th>Name</Table.Th>
                  <Table.Th>Attributes</Table.Th>
                  <Table.Th>Connection Limit</Table.Th>
                  <Table.Th>Active Connections</Table.Th>
                  <Table.Th>Valid Until</Table.Th>
                  <Table.Th w={60}></Table.Th>
                </Table.Tr>
              </Table.Thead>
              <Table.Tbody>
                {systemRoles.map((role) => renderRoleRow(role, false))}
              </Table.Tbody>
            </Table>
          </Paper>
        </Stack>
      )}

      {/* Create Role Modal */}
      <Modal
        opened={createModalOpened}
        onClose={closeCreateModal}
        title="Create role"
        size="md"
      >
        <Stack gap="md">
          <TextInput
            label="Role name"
            placeholder="my_role"
            value={roleName}
            onChange={(e) => setRoleName(e.target.value)}
            required
          />

          <PasswordInput
            label="Password (optional)"
            description="Leave empty for roles that don't need to log in"
            placeholder="Enter password"
            value={rolePassword}
            onChange={(e) => setRolePassword(e.target.value)}
          />

          <Text size="sm" fw={500} mt="md">
            Attributes
          </Text>

          <Checkbox
            label="Can login"
            description="Allow this role to connect to the database"
            checked={roleCanLogin}
            onChange={(e) => setRoleCanLogin(e.currentTarget.checked)}
          />

          <Checkbox
            label="Inherit privileges"
            description="Automatically inherit privileges from granted roles"
            checked={roleInherit}
            onChange={(e) => setRoleInherit(e.currentTarget.checked)}
          />

          <Checkbox
            label="Can create databases"
            description="Allow this role to create new databases"
            checked={roleCanCreateDB}
            onChange={(e) => setRoleCanCreateDB(e.currentTarget.checked)}
          />

          <Checkbox
            label="Can create roles"
            description="Allow this role to create and manage other roles"
            checked={roleCanCreateRole}
            onChange={(e) => setRoleCanCreateRole(e.currentTarget.checked)}
          />

          <Checkbox
            label="Superuser"
            description="Grant all privileges (use with caution)"
            checked={roleIsSuperuser}
            onChange={(e) => setRoleIsSuperuser(e.currentTarget.checked)}
            color="red"
          />

          <NumberInput
            label="Connection limit"
            description="Maximum concurrent connections (-1 for unlimited)"
            placeholder="Unlimited"
            value={roleConnectionLimit}
            onChange={(val) => setRoleConnectionLimit(val as number | undefined)}
            min={-1}
          />

          <Group justify="flex-end" mt="md">
            <Button variant="outline" onClick={closeCreateModal}>
              Cancel
            </Button>
            <Button onClick={handleCreateRole} loading={formLoading}>
              Create Role
            </Button>
          </Group>
        </Stack>
      </Modal>

      {/* Delete Confirmation */}
      <ConfirmModal
        opened={deleteModalOpened}
        onClose={closeDeleteModal}
        onConfirm={handleDeleteRole}
        title="Delete role"
        message={`Are you sure you want to delete the role "${selectedRole?.name}"? This action cannot be undone and may affect database access.`}
        confirmLabel="Delete"
        danger
        loading={formLoading}
      />
    </PageContainer>
  );
}
