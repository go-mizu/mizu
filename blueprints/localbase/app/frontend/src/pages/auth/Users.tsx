import { useEffect, useState, useCallback } from 'react';
import {
  Button,
  Modal,
  TextInput,
  PasswordInput,
  Stack,
  Group,
  Text,
  ActionIcon,
  Menu,
  Badge,
  Tooltip,
} from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import {
  IconPlus,
  IconDotsVertical,
  IconTrash,
  IconEdit,
  IconMail,
  IconUsers,
} from '@tabler/icons-react';
import { notifications } from '@mantine/notifications';
import { PageContainer } from '../../components/layout/PageContainer';
import { DataTable, type Column } from '../../components/common/DataTable';
import { EmptyState } from '../../components/common/EmptyState';
import { ConfirmModal } from '../../components/common/ConfirmModal';
import { SearchInput } from '../../components/forms/SearchInput';
import { authApi } from '../../api';
import type { User } from '../../types';

export function UsersPage() {
  const [users, setUsers] = useState<User[]>([]);
  const [filteredUsers, setFilteredUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedUser, setSelectedUser] = useState<User | null>(null);

  // Modal states
  const [createOpened, { open: openCreate, close: closeCreate }] = useDisclosure(false);
  const [editOpened, { open: openEdit, close: closeEdit }] = useDisclosure(false);
  const [deleteOpened, { open: openDelete, close: closeDelete }] = useDisclosure(false);

  // Form state
  const [formData, setFormData] = useState({
    email: '',
    password: '',
    phone: '',
  });
  const [formLoading, setFormLoading] = useState(false);

  const fetchUsers = useCallback(async () => {
    setLoading(true);
    try {
      const response = await authApi.listUsers(1, 100);
      setUsers(response.users);
      setFilteredUsers(response.users);
    } catch (error) {
      notifications.show({
        title: 'Error',
        message: 'Failed to load users',
        color: 'red',
      });
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchUsers();
  }, [fetchUsers]);

  useEffect(() => {
    if (searchQuery) {
      const query = searchQuery.toLowerCase();
      setFilteredUsers(
        users.filter(
          (user) =>
            user.email?.toLowerCase().includes(query) ||
            user.phone?.toLowerCase().includes(query)
        )
      );
    } else {
      setFilteredUsers(users);
    }
  }, [searchQuery, users]);

  const handleCreate = async () => {
    if (!formData.email || !formData.password) {
      notifications.show({
        title: 'Validation Error',
        message: 'Email and password are required',
        color: 'red',
      });
      return;
    }

    setFormLoading(true);
    try {
      await authApi.createUser({
        email: formData.email,
        password: formData.password,
        phone: formData.phone || undefined,
      });
      notifications.show({
        title: 'Success',
        message: 'User created successfully',
        color: 'green',
      });
      closeCreate();
      setFormData({ email: '', password: '', phone: '' });
      fetchUsers();
    } catch (error: any) {
      notifications.show({
        title: 'Error',
        message: error.message || 'Failed to create user',
        color: 'red',
      });
    } finally {
      setFormLoading(false);
    }
  };

  const handleEdit = async () => {
    if (!selectedUser) return;

    setFormLoading(true);
    try {
      await authApi.updateUser(selectedUser.id, {
        email: formData.email || undefined,
        phone: formData.phone || undefined,
        password: formData.password || undefined,
      });
      notifications.show({
        title: 'Success',
        message: 'User updated successfully',
        color: 'green',
      });
      closeEdit();
      setFormData({ email: '', password: '', phone: '' });
      setSelectedUser(null);
      fetchUsers();
    } catch (error: any) {
      notifications.show({
        title: 'Error',
        message: error.message || 'Failed to update user',
        color: 'red',
      });
    } finally {
      setFormLoading(false);
    }
  };

  const handleDelete = async () => {
    if (!selectedUser) return;

    setFormLoading(true);
    try {
      await authApi.deleteUser(selectedUser.id);
      notifications.show({
        title: 'Success',
        message: 'User deleted successfully',
        color: 'green',
      });
      closeDelete();
      setSelectedUser(null);
      fetchUsers();
    } catch (error: any) {
      notifications.show({
        title: 'Error',
        message: error.message || 'Failed to delete user',
        color: 'red',
      });
    } finally {
      setFormLoading(false);
    }
  };

  const openEditModal = (user: User) => {
    setSelectedUser(user);
    setFormData({
      email: user.email || '',
      password: '',
      phone: user.phone || '',
    });
    openEdit();
  };

  const openDeleteModal = (user: User) => {
    setSelectedUser(user);
    openDelete();
  };

  const formatDate = (dateString: string | undefined) => {
    if (!dateString) return '-';
    return new Date(dateString).toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  const columns: Column<User>[] = [
    {
      key: 'email',
      header: 'Email',
      render: (user) => (
        <Group gap="xs">
          <Text size="sm" fw={500}>
            {user.email || '-'}
          </Text>
          {user.email_confirmed_at && (
            <Tooltip label="Email verified">
              <Badge size="xs" color="green" variant="light">
                Verified
              </Badge>
            </Tooltip>
          )}
        </Group>
      ),
    },
    {
      key: 'phone',
      header: 'Phone',
      render: (user) => <Text size="sm">{user.phone || '-'}</Text>,
    },
    {
      key: 'provider',
      header: 'Provider',
      render: (user) => {
        const provider = user.app_metadata?.provider || 'email';
        return (
          <Badge size="sm" variant="light" color="gray">
            {provider}
          </Badge>
        );
      },
    },
    {
      key: 'created_at',
      header: 'Created',
      render: (user) => (
        <Text size="sm" c="dimmed">
          {formatDate(user.created_at)}
        </Text>
      ),
    },
    {
      key: 'last_sign_in_at',
      header: 'Last Sign In',
      render: (user) => (
        <Text size="sm" c="dimmed">
          {formatDate(user.last_sign_in_at)}
        </Text>
      ),
    },
    {
      key: 'actions',
      header: '',
      width: 50,
      render: (user) => (
        <Menu position="bottom-end" withinPortal>
          <Menu.Target>
            <ActionIcon variant="subtle" color="gray" onClick={(e) => e.stopPropagation()}>
              <IconDotsVertical size={16} />
            </ActionIcon>
          </Menu.Target>
          <Menu.Dropdown>
            <Menu.Item
              leftSection={<IconEdit size={14} />}
              onClick={(e) => {
                e.stopPropagation();
                openEditModal(user);
              }}
            >
              Edit user
            </Menu.Item>
            <Menu.Divider />
            <Menu.Item
              color="red"
              leftSection={<IconTrash size={14} />}
              onClick={(e) => {
                e.stopPropagation();
                openDeleteModal(user);
              }}
            >
              Delete user
            </Menu.Item>
          </Menu.Dropdown>
        </Menu>
      ),
    },
  ];

  return (
    <PageContainer
      title="Users"
      description="Manage authenticated users"
      actions={
        <Button leftSection={<IconPlus size={16} />} onClick={openCreate}>
          Add user
        </Button>
      }
    >
      {/* Search */}
      <Group mb="lg">
        <SearchInput
          value={searchQuery}
          onChange={setSearchQuery}
          placeholder="Search by email or phone..."
        />
        <Text size="sm" c="dimmed">
          {filteredUsers.length} user{filteredUsers.length !== 1 ? 's' : ''}
        </Text>
      </Group>

      {/* Table */}
      <DataTable
        data={filteredUsers}
        columns={columns}
        loading={loading}
        getRowKey={(user) => user.id}
        emptyState={
          <EmptyState
            icon={<IconUsers size={32} />}
            title="No users found"
            description={
              searchQuery
                ? 'No users match your search criteria'
                : 'Get started by creating your first user'
            }
            action={
              !searchQuery
                ? {
                    label: 'Add user',
                    onClick: openCreate,
                  }
                : undefined
            }
          />
        }
      />

      {/* Create User Modal */}
      <Modal opened={createOpened} onClose={closeCreate} title="Create new user" size="md">
        <Stack gap="md">
          <TextInput
            label="Email"
            placeholder="user@example.com"
            value={formData.email}
            onChange={(e) => setFormData({ ...formData, email: e.target.value })}
            required
            leftSection={<IconMail size={16} />}
          />
          <PasswordInput
            label="Password"
            placeholder="Minimum 8 characters"
            value={formData.password}
            onChange={(e) => setFormData({ ...formData, password: e.target.value })}
            required
          />
          <TextInput
            label="Phone (optional)"
            placeholder="+1234567890"
            value={formData.phone}
            onChange={(e) => setFormData({ ...formData, phone: e.target.value })}
          />
          <Group justify="flex-end" mt="md">
            <Button variant="outline" onClick={closeCreate}>
              Cancel
            </Button>
            <Button onClick={handleCreate} loading={formLoading}>
              Create user
            </Button>
          </Group>
        </Stack>
      </Modal>

      {/* Edit User Modal */}
      <Modal opened={editOpened} onClose={closeEdit} title="Edit user" size="md">
        <Stack gap="md">
          <TextInput
            label="Email"
            placeholder="user@example.com"
            value={formData.email}
            onChange={(e) => setFormData({ ...formData, email: e.target.value })}
            leftSection={<IconMail size={16} />}
          />
          <PasswordInput
            label="New Password (leave empty to keep current)"
            placeholder="Minimum 8 characters"
            value={formData.password}
            onChange={(e) => setFormData({ ...formData, password: e.target.value })}
          />
          <TextInput
            label="Phone"
            placeholder="+1234567890"
            value={formData.phone}
            onChange={(e) => setFormData({ ...formData, phone: e.target.value })}
          />
          <Group justify="flex-end" mt="md">
            <Button variant="outline" onClick={closeEdit}>
              Cancel
            </Button>
            <Button onClick={handleEdit} loading={formLoading}>
              Save changes
            </Button>
          </Group>
        </Stack>
      </Modal>

      {/* Delete Confirmation Modal */}
      <ConfirmModal
        opened={deleteOpened}
        onClose={closeDelete}
        onConfirm={handleDelete}
        title="Delete user"
        message={`Are you sure you want to delete ${selectedUser?.email || 'this user'}? This action cannot be undone.`}
        confirmLabel="Delete"
        danger
        loading={formLoading}
      />
    </PageContainer>
  );
}
