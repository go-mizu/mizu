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
  Tabs,
  Paper,
  Switch,
  SimpleGrid,
  Card,
  ThemeIcon,
  Box,
  Alert,
  NumberInput,
  Textarea,
} from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import {
  IconPlus,
  IconDotsVertical,
  IconTrash,
  IconEdit,
  IconMail,
  IconUsers,
  IconKey,
  IconBrandGoogle,
  IconBrandGithub,
  IconBrandApple,
  IconBrandTwitter,
  IconBrandDiscord,
  IconPhone,
  IconShield,
  IconSettings,
  IconInfoCircle,
  IconExternalLink,
} from '@tabler/icons-react';
import { notifications } from '@mantine/notifications';
import { PageContainer } from '../../components/layout/PageContainer';
import { DataTable, type Column } from '../../components/common/DataTable';
import { EmptyState } from '../../components/common/EmptyState';
import { ConfirmModal } from '../../components/common/ConfirmModal';
import { SearchInput } from '../../components/forms/SearchInput';
import { authApi } from '../../api';
import type { User } from '../../types';

interface AuthProvider {
  id: string;
  name: string;
  icon: typeof IconMail;
  enabled: boolean;
  clientId?: string;
  clientSecret?: string;
  description: string;
  category: 'email' | 'phone' | 'social';
}

const defaultProviders: AuthProvider[] = [
  {
    id: 'email',
    name: 'Email',
    icon: IconMail,
    enabled: true,
    description: 'Allow users to sign up and sign in with email and password',
    category: 'email',
  },
  {
    id: 'phone',
    name: 'Phone',
    icon: IconPhone,
    enabled: false,
    description: 'Allow users to sign up and sign in with phone number and OTP',
    category: 'phone',
  },
  {
    id: 'google',
    name: 'Google',
    icon: IconBrandGoogle,
    enabled: false,
    clientId: '',
    clientSecret: '',
    description: 'Allow users to sign in with their Google account',
    category: 'social',
  },
  {
    id: 'github',
    name: 'GitHub',
    icon: IconBrandGithub,
    enabled: false,
    clientId: '',
    clientSecret: '',
    description: 'Allow users to sign in with their GitHub account',
    category: 'social',
  },
  {
    id: 'apple',
    name: 'Apple',
    icon: IconBrandApple,
    enabled: false,
    clientId: '',
    clientSecret: '',
    description: 'Allow users to sign in with their Apple ID',
    category: 'social',
  },
  {
    id: 'twitter',
    name: 'Twitter',
    icon: IconBrandTwitter,
    enabled: false,
    clientId: '',
    clientSecret: '',
    description: 'Allow users to sign in with their Twitter account',
    category: 'social',
  },
  {
    id: 'discord',
    name: 'Discord',
    icon: IconBrandDiscord,
    enabled: false,
    clientId: '',
    clientSecret: '',
    description: 'Allow users to sign in with their Discord account',
    category: 'social',
  },
];

interface AuthSettings {
  siteUrl: string;
  redirectUrls: string;
  jwtExpiry: number;
  enableSignUp: boolean;
  enableConfirmation: boolean;
  enableDoubleConfirmation: boolean;
  minimumPasswordLength: number;
}

export function UsersPage() {
  const [activeTab, setActiveTab] = useState<string | null>('users');
  const [users, setUsers] = useState<User[]>([]);
  const [filteredUsers, setFilteredUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedUser, setSelectedUser] = useState<User | null>(null);

  // Providers state
  const [providers, setProviders] = useState<AuthProvider[]>(defaultProviders);
  const [selectedProvider, setSelectedProvider] = useState<AuthProvider | null>(null);
  const [providerModalOpened, { open: openProviderModal, close: closeProviderModal }] =
    useDisclosure(false);

  // Auth settings
  const [authSettings, setAuthSettings] = useState<AuthSettings>({
    siteUrl: 'http://localhost:3000',
    redirectUrls: 'http://localhost:3000/**',
    jwtExpiry: 3600,
    enableSignUp: true,
    enableConfirmation: true,
    enableDoubleConfirmation: false,
    minimumPasswordLength: 8,
  });

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

  const toggleProvider = (providerId: string) => {
    setProviders(
      providers.map((p) =>
        p.id === providerId ? { ...p, enabled: !p.enabled } : p
      )
    );
    notifications.show({
      title: 'Provider Updated',
      message: `Provider ${providerId} has been ${providers.find(p => p.id === providerId)?.enabled ? 'disabled' : 'enabled'}`,
      color: 'green',
    });
  };

  const openProviderSettings = (provider: AuthProvider) => {
    setSelectedProvider(provider);
    openProviderModal();
  };

  const handleSaveProvider = () => {
    if (!selectedProvider) return;

    setProviders(
      providers.map((p) =>
        p.id === selectedProvider.id ? selectedProvider : p
      )
    );
    closeProviderModal();
    notifications.show({
      title: 'Provider Saved',
      message: `${selectedProvider.name} settings have been updated`,
      color: 'green',
    });
  };

  const handleSaveSettings = () => {
    notifications.show({
      title: 'Settings Saved',
      message: 'Authentication settings have been updated',
      color: 'green',
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

  const enabledProviders = providers.filter((p) => p.enabled);
  const socialProviders = providers.filter((p) => p.category === 'social');

  return (
    <PageContainer
      title="Authentication"
      description="Manage users, providers, and authentication settings"
    >
      <Tabs value={activeTab} onChange={setActiveTab}>
        <Tabs.List mb="lg">
          <Tabs.Tab value="users" leftSection={<IconUsers size={16} />}>
            Users
          </Tabs.Tab>
          <Tabs.Tab value="providers" leftSection={<IconKey size={16} />}>
            Providers
          </Tabs.Tab>
          <Tabs.Tab value="settings" leftSection={<IconSettings size={16} />}>
            Settings
          </Tabs.Tab>
        </Tabs.List>

        {/* Users Tab */}
        <Tabs.Panel value="users">
          <Group justify="space-between" mb="lg">
            <Group>
              <SearchInput
                value={searchQuery}
                onChange={setSearchQuery}
                placeholder="Search by email or phone..."
              />
              <Text size="sm" c="dimmed">
                {filteredUsers.length} user{filteredUsers.length !== 1 ? 's' : ''}
              </Text>
            </Group>
            <Button leftSection={<IconPlus size={16} />} onClick={openCreate}>
              Add user
            </Button>
          </Group>

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
        </Tabs.Panel>

        {/* Providers Tab */}
        <Tabs.Panel value="providers">
          <Stack gap="lg">
            <Alert icon={<IconInfoCircle size={16} />} color="blue">
              Configure authentication providers for your application. Enable the providers
              you want to allow users to sign in with.
            </Alert>

            {/* Email & Phone */}
            <Box>
              <Text fw={600} mb="md">
                Email & Phone
              </Text>
              <SimpleGrid cols={{ base: 1, sm: 2 }} spacing="md">
                {providers
                  .filter((p) => p.category === 'email' || p.category === 'phone')
                  .map((provider) => (
                    <Card key={provider.id} padding="md" withBorder>
                      <Group justify="space-between">
                        <Group gap="sm">
                          <ThemeIcon
                            size="lg"
                            radius="md"
                            variant="light"
                            color={provider.enabled ? 'green' : 'gray'}
                          >
                            <provider.icon size={20} />
                          </ThemeIcon>
                          <Box>
                            <Text fw={500}>{provider.name}</Text>
                            <Text size="xs" c="dimmed">
                              {provider.description}
                            </Text>
                          </Box>
                        </Group>
                        <Switch
                          checked={provider.enabled}
                          onChange={() => toggleProvider(provider.id)}
                        />
                      </Group>
                    </Card>
                  ))}
              </SimpleGrid>
            </Box>

            {/* Social Providers */}
            <Box>
              <Text fw={600} mb="md">
                Social Providers
              </Text>
              <SimpleGrid cols={{ base: 1, sm: 2, lg: 3 }} spacing="md">
                {socialProviders.map((provider) => (
                  <Card key={provider.id} padding="md" withBorder>
                    <Group justify="space-between" mb="sm">
                      <Group gap="sm">
                        <ThemeIcon
                          size="lg"
                          radius="md"
                          variant="light"
                          color={provider.enabled ? 'green' : 'gray'}
                        >
                          <provider.icon size={20} />
                        </ThemeIcon>
                        <Text fw={500}>{provider.name}</Text>
                      </Group>
                      <Switch
                        checked={provider.enabled}
                        onChange={() => toggleProvider(provider.id)}
                      />
                    </Group>
                    <Text size="xs" c="dimmed" mb="sm">
                      {provider.description}
                    </Text>
                    {provider.enabled && (
                      <Button
                        size="xs"
                        variant="light"
                        fullWidth
                        onClick={() => openProviderSettings(provider)}
                      >
                        Configure
                      </Button>
                    )}
                  </Card>
                ))}
              </SimpleGrid>
            </Box>

            {/* Active Providers Summary */}
            <Paper p="md" withBorder>
              <Text fw={600} mb="sm">
                Active Providers
              </Text>
              {enabledProviders.length === 0 ? (
                <Text size="sm" c="dimmed">
                  No providers enabled. Enable at least one provider to allow users to sign in.
                </Text>
              ) : (
                <Group gap="xs">
                  {enabledProviders.map((provider) => (
                    <Badge
                      key={provider.id}
                      size="lg"
                      variant="light"
                      leftSection={<provider.icon size={14} />}
                    >
                      {provider.name}
                    </Badge>
                  ))}
                </Group>
              )}
            </Paper>
          </Stack>
        </Tabs.Panel>

        {/* Settings Tab */}
        <Tabs.Panel value="settings">
          <Stack gap="lg">
            <SimpleGrid cols={{ base: 1, lg: 2 }} spacing="lg">
              {/* URL Configuration */}
              <Paper p="md" withBorder>
                <Group gap="xs" mb="md">
                  <IconShield size={18} />
                  <Text fw={600}>URL Configuration</Text>
                </Group>
                <Stack gap="md">
                  <TextInput
                    label="Site URL"
                    description="The base URL of your application"
                    value={authSettings.siteUrl}
                    onChange={(e) =>
                      setAuthSettings({ ...authSettings, siteUrl: e.target.value })
                    }
                  />
                  <Textarea
                    label="Redirect URLs"
                    description="Allowed redirect URLs (one per line, supports wildcards)"
                    value={authSettings.redirectUrls}
                    onChange={(e) =>
                      setAuthSettings({ ...authSettings, redirectUrls: e.target.value })
                    }
                    minRows={3}
                  />
                </Stack>
              </Paper>

              {/* Security Settings */}
              <Paper p="md" withBorder>
                <Group gap="xs" mb="md">
                  <IconKey size={18} />
                  <Text fw={600}>Security</Text>
                </Group>
                <Stack gap="md">
                  <NumberInput
                    label="JWT Expiry (seconds)"
                    description="How long access tokens are valid"
                    value={authSettings.jwtExpiry}
                    onChange={(val) =>
                      setAuthSettings({ ...authSettings, jwtExpiry: Number(val) || 3600 })
                    }
                    min={60}
                    max={604800}
                  />
                  <NumberInput
                    label="Minimum Password Length"
                    description="Required minimum characters for passwords"
                    value={authSettings.minimumPasswordLength}
                    onChange={(val) =>
                      setAuthSettings({
                        ...authSettings,
                        minimumPasswordLength: Number(val) || 8,
                      })
                    }
                    min={6}
                    max={128}
                  />
                </Stack>
              </Paper>
            </SimpleGrid>

            {/* Feature Toggles */}
            <Paper p="md" withBorder>
              <Group gap="xs" mb="md">
                <IconSettings size={18} />
                <Text fw={600}>Features</Text>
              </Group>
              <Stack gap="md">
                <Switch
                  label="Enable Sign-ups"
                  description="Allow new users to create accounts"
                  checked={authSettings.enableSignUp}
                  onChange={(e) =>
                    setAuthSettings({
                      ...authSettings,
                      enableSignUp: e.currentTarget.checked,
                    })
                  }
                />
                <Switch
                  label="Email Confirmation"
                  description="Require email verification before allowing sign-in"
                  checked={authSettings.enableConfirmation}
                  onChange={(e) =>
                    setAuthSettings({
                      ...authSettings,
                      enableConfirmation: e.currentTarget.checked,
                    })
                  }
                />
                <Switch
                  label="Double Confirmation"
                  description="Require confirmation when changing email addresses"
                  checked={authSettings.enableDoubleConfirmation}
                  onChange={(e) =>
                    setAuthSettings({
                      ...authSettings,
                      enableDoubleConfirmation: e.currentTarget.checked,
                    })
                  }
                />
              </Stack>
            </Paper>

            <Group justify="flex-end">
              <Button onClick={handleSaveSettings}>Save Settings</Button>
            </Group>
          </Stack>
        </Tabs.Panel>
      </Tabs>

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

      {/* Provider Configuration Modal */}
      <Modal
        opened={providerModalOpened}
        onClose={closeProviderModal}
        title={`Configure ${selectedProvider?.name}`}
        size="md"
      >
        {selectedProvider && (
          <Stack gap="md">
            <Alert icon={<IconInfoCircle size={16} />} color="blue">
              Get your OAuth credentials from the{' '}
              <Text component="a" href="#" c="blue" style={{ textDecoration: 'underline' }}>
                {selectedProvider.name} Developer Console
              </Text>
            </Alert>
            <TextInput
              label="Client ID"
              placeholder={`Your ${selectedProvider.name} Client ID`}
              value={selectedProvider.clientId || ''}
              onChange={(e) =>
                setSelectedProvider({
                  ...selectedProvider,
                  clientId: e.target.value,
                })
              }
            />
            <PasswordInput
              label="Client Secret"
              placeholder={`Your ${selectedProvider.name} Client Secret`}
              value={selectedProvider.clientSecret || ''}
              onChange={(e) =>
                setSelectedProvider({
                  ...selectedProvider,
                  clientSecret: e.target.value,
                })
              }
            />
            <Box>
              <Text size="sm" fw={500} mb={4}>
                Callback URL
              </Text>
              <TextInput
                value={`${authSettings.siteUrl}/auth/v1/callback`}
                readOnly
                styles={{ input: { fontFamily: 'monospace', fontSize: 12 } }}
              />
              <Text size="xs" c="dimmed" mt={4}>
                Add this URL to your {selectedProvider.name} app's authorized redirect URIs
              </Text>
            </Box>
            <Group justify="flex-end" mt="md">
              <Button variant="outline" onClick={closeProviderModal}>
                Cancel
              </Button>
              <Button onClick={handleSaveProvider}>Save</Button>
            </Group>
          </Stack>
        )}
      </Modal>
    </PageContainer>
  );
}
