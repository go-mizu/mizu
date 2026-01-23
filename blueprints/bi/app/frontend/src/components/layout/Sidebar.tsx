import { useState } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import {
  Box, Text, TextInput, Group, UnstyledButton, Collapse, ActionIcon,
  Tooltip, Menu, ScrollArea, Divider, Badge, rem, Button
} from '@mantine/core'
import {
  IconHome2, IconPlus, IconLayoutDashboard, IconSettings2, IconSearch,
  IconDatabase, IconUsers, IconChevronRight, IconChevronDown, IconLogout,
  IconFileAnalytics, IconClock, IconStar,
  IconFolder, IconFolderFilled, IconSparkles, IconPencil, IconMenu2,
  IconUser, IconDots, IconChartPie
} from '@tabler/icons-react'
import { useCollections, useCurrentUser, useLogout } from '../../api/hooks'
import { useUIStore } from '../../stores/uiStore'
import { useBookmarkStore } from '../../stores/bookmarkStore'

// =============================================================================
// FLOATING HAMBURGER BUTTON (shown when sidebar is hidden)
// =============================================================================

export function FloatingHamburger() {
  const { sidebarCollapsed, toggleSidebar } = useUIStore()

  if (!sidebarCollapsed) return null

  return (
    <Box
      style={{
        position: 'fixed',
        top: rem(12),
        left: rem(12),
        zIndex: 100,
      }}
    >
      <Tooltip label="Show sidebar" position="right">
        <ActionIcon
          variant="default"
          size="lg"
          onClick={toggleSidebar}
          style={{
            backgroundColor: 'var(--color-background)',
            border: '1px solid var(--color-border)',
            boxShadow: 'var(--shadow)',
          }}
        >
          <IconMenu2 size={20} strokeWidth={1.75} />
        </ActionIcon>
      </Tooltip>
    </Box>
  )
}

// =============================================================================
// COLLECTION ITEM COMPONENT
// =============================================================================

interface CollectionItemProps {
  id: string
  name: string
  color?: string
  depth?: number
  collections: any[]
}

function CollectionItem({ id, name, color, depth = 0, collections }: CollectionItemProps) {
  const navigate = useNavigate()
  const location = useLocation()
  const [expanded, setExpanded] = useState(false)
  const children = collections?.filter(c => c.parent_id === id) || []
  const hasChildren = children.length > 0
  const isActive = location.pathname === `/collection/${id}`

  return (
    <Box>
      <UnstyledButton
        onClick={() => navigate(`/collection/${id}`)}
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: rem(10),
          padding: `${rem(8)} ${rem(12)}`,
          paddingLeft: rem(12 + depth * 16),
          borderRadius: 'var(--radius)',
          color: isActive ? 'var(--sidebar-text-active)' : 'var(--sidebar-text)',
          fontSize: rem(14),
          fontWeight: 500,
          cursor: 'pointer',
          transition: 'all var(--transition-fast)',
          marginBottom: rem(2),
          backgroundColor: isActive ? 'var(--sidebar-bg-active)' : 'transparent',
        }}
        onMouseEnter={(e) => {
          if (!isActive) {
            e.currentTarget.style.backgroundColor = 'var(--sidebar-bg-hover)'
          }
        }}
        onMouseLeave={(e) => {
          if (!isActive) {
            e.currentTarget.style.backgroundColor = 'transparent'
          }
        }}
      >
        {hasChildren && (
          <ActionIcon
            size="xs"
            variant="transparent"
            onClick={(e) => {
              e.stopPropagation()
              setExpanded(!expanded)
            }}
            style={{ color: 'var(--sidebar-icon)' }}
          >
            {expanded ? <IconChevronDown size={14} /> : <IconChevronRight size={14} />}
          </ActionIcon>
        )}
        {!hasChildren && <Box w={18} />}
        {expanded ? (
          <IconFolderFilled size={18} style={{ color: color || 'var(--color-warning)', flexShrink: 0 }} />
        ) : (
          <IconFolder size={18} style={{ color: color || 'var(--sidebar-icon)', flexShrink: 0 }} />
        )}
        <Text size="sm" truncate style={{ flex: 1, color: isActive ? 'var(--sidebar-text-active)' : 'var(--sidebar-text)' }}>
          {name}
        </Text>
      </UnstyledButton>

      {hasChildren && (
        <Collapse in={expanded}>
          {children.map(child => (
            <CollectionItem
              key={child.id}
              id={child.id}
              name={child.name}
              color={child.color}
              depth={depth + 1}
              collections={collections}
            />
          ))}
        </Collapse>
      )}
    </Box>
  )
}

// =============================================================================
// NAV LINK COMPONENT
// =============================================================================

interface NavItemProps {
  icon: typeof IconHome2
  label: string
  path?: string
  onClick?: () => void
  active?: boolean
  rightSection?: React.ReactNode
  iconColor?: string
}

function NavItem({ icon: Icon, label, path, onClick, active, rightSection, iconColor }: NavItemProps) {
  const navigate = useNavigate()
  const location = useLocation()
  const isActive = active ?? (path ? location.pathname === path || location.pathname.startsWith(path + '/') : false)

  const handleClick = () => {
    if (onClick) onClick()
    else if (path) navigate(path)
  }

  return (
    <UnstyledButton
      onClick={handleClick}
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: rem(10),
        padding: `${rem(8)} ${rem(12)}`,
        borderRadius: 'var(--radius)',
        color: isActive ? 'var(--sidebar-text-active)' : 'var(--sidebar-text)',
        fontSize: rem(14),
        fontWeight: 500,
        cursor: 'pointer',
        transition: 'all var(--transition-fast)',
        marginBottom: rem(2),
        backgroundColor: isActive ? 'var(--sidebar-bg-active)' : 'transparent',
        width: '100%',
      }}
      onMouseEnter={(e) => {
        if (!isActive) {
          e.currentTarget.style.backgroundColor = 'var(--sidebar-bg-hover)'
        }
      }}
      onMouseLeave={(e) => {
        if (!isActive) {
          e.currentTarget.style.backgroundColor = 'transparent'
        }
      }}
    >
      <Icon
        size={20}
        style={{
          flexShrink: 0,
          color: isActive ? 'var(--sidebar-icon-active)' : (iconColor || 'var(--sidebar-icon)'),
          strokeWidth: 1.75
        }}
      />
      <Text size="sm" style={{ flex: 1, color: isActive ? 'var(--sidebar-text-active)' : 'var(--sidebar-text)' }}>
        {label}
      </Text>
      {rightSection}
    </UnstyledButton>
  )
}

// =============================================================================
// MAIN SIDEBAR COMPONENT
// =============================================================================

export default function Sidebar() {
  const navigate = useNavigate()
  const location = useLocation()
  const { data: user } = useCurrentUser()
  const { mutate: logout } = useLogout()
  const { data: collections } = useCollections()
  const { openCommandPalette, sidebarCollapsed, toggleSidebar } = useUIStore()
  const { bookmarks, recentItems } = useBookmarkStore()

  const rootCollections = collections?.filter(c => !c.parent_id) || []
  const [browseExpanded, setBrowseExpanded] = useState(location.pathname.startsWith('/browse'))
  const [adminExpanded, setAdminExpanded] = useState(location.pathname.startsWith('/admin'))
  const [bookmarksExpanded, setBookmarksExpanded] = useState(true)
  const [recentsExpanded, setRecentsExpanded] = useState(true)
  const [collectionsExpanded, setCollectionsExpanded] = useState(true)

  // Get icon for item type
  const getItemIcon = (type: string) => {
    switch (type) {
      case 'dashboard': return IconLayoutDashboard
      case 'question': return IconChartPie
      case 'collection': return IconFolder
      default: return IconChartPie
    }
  }

  return (
    <Box
      component="aside"
      style={{
        width: 'var(--sidebar-width)',
        height: '100vh',
        backgroundColor: 'var(--sidebar-bg)',
        display: 'flex',
        flexDirection: 'column',
        borderRight: '1px solid var(--sidebar-border)',
        transition: 'transform var(--transition), opacity var(--transition)',
        position: 'fixed',
        left: 0,
        top: 0,
        zIndex: 200,
        ...(sidebarCollapsed ? {
          transform: 'translateX(-100%)',
          opacity: 0,
          pointerEvents: 'none',
        } : {}),
      }}
    >
      {/* Header with Hamburger, Logo and New Button */}
      <Box
        style={{
          padding: `${rem(12)} ${rem(12)}`,
          borderBottom: '1px solid var(--sidebar-border)',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          minHeight: 56,
        }}
      >
        <Group gap="xs">
          {/* Hamburger Menu */}
          <Tooltip label="Hide sidebar" position="right">
            <ActionIcon
              variant="subtle"
              size="md"
              onClick={toggleSidebar}
              style={{ color: 'var(--sidebar-icon)' }}
            >
              <IconMenu2 size={20} strokeWidth={1.75} />
            </ActionIcon>
          </Tooltip>

          <UnstyledButton
            onClick={() => navigate('/')}
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: rem(8),
              cursor: 'pointer',
            }}
          >
            <Box
              style={{
                width: 28,
                height: 28,
                borderRadius: 'var(--radius)',
                background: 'linear-gradient(135deg, var(--color-primary) 0%, #3B7DBF 100%)',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
              }}
            >
              <IconSparkles size={16} color="#ffffff" strokeWidth={2} />
            </Box>
          </UnstyledButton>
        </Group>

        <Menu position="bottom-start" width={200} shadow="md">
          <Menu.Target>
            <ActionIcon
              variant="filled"
              color="brand"
              size="md"
              radius="md"
            >
              <IconPlus size={16} strokeWidth={2.5} />
            </ActionIcon>
          </Menu.Target>
          <Menu.Dropdown>
            <Menu.Item
              leftSection={<IconPencil size={16} style={{ color: 'var(--color-primary)' }} strokeWidth={1.75} />}
              onClick={() => navigate('/question/new')}
            >
              New question
            </Menu.Item>
            <Menu.Item
              leftSection={<IconLayoutDashboard size={16} style={{ color: 'var(--color-success)' }} strokeWidth={1.75} />}
              onClick={() => navigate('/dashboard/new')}
            >
              New dashboard
            </Menu.Item>
            <Menu.Divider />
            <Menu.Item
              leftSection={<IconFolder size={16} style={{ color: 'var(--color-warning)' }} strokeWidth={1.75} />}
              onClick={() => navigate('/collection/new')}
            >
              New collection
            </Menu.Item>
          </Menu.Dropdown>
        </Menu>
      </Box>

      {/* Main Navigation */}
      <ScrollArea style={{ flex: 1 }} scrollbarSize={6}>
        {/* Search */}
        <Box style={{ padding: `${rem(8)} ${rem(12)}` }}>
          <TextInput
            placeholder="Search..."
            leftSection={<IconSearch size={16} style={{ color: 'var(--sidebar-input-placeholder)' }} strokeWidth={1.75} />}
            size="sm"
            onClick={openCommandPalette}
            readOnly
            rightSection={
              <Badge size="xs" variant="light" color="gray" style={{ cursor: 'pointer' }}>
                K
              </Badge>
            }
            styles={{
              input: {
                backgroundColor: 'var(--sidebar-input-bg)',
                border: '1px solid var(--sidebar-input-border)',
                color: 'var(--sidebar-input-text)',
                cursor: 'pointer',
                '&::placeholder': {
                  color: 'var(--sidebar-input-placeholder)',
                },
                '&:hover': {
                  backgroundColor: 'var(--color-background-subtle)',
                  borderColor: 'var(--color-border-strong)',
                },
              },
            }}
          />
        </Box>

        <Box style={{ padding: `${rem(4)} ${rem(8)}` }}>
          {/* Home */}
          <NavItem icon={IconHome2} label="Home" path="/" />

          {/* Browse Section - Models, Databases */}
          <Box
            style={{
              padding: `${rem(12)} ${rem(12)} ${rem(4)}`,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
            }}
          >
            <Text
              style={{
                fontSize: rem(11),
                fontWeight: 700,
                textTransform: 'uppercase',
                letterSpacing: '0.05em',
                color: 'var(--sidebar-section-title)',
                cursor: 'pointer',
              }}
              onClick={() => setBrowseExpanded(!browseExpanded)}
            >
              BROWSE
            </Text>
            <ActionIcon
              size="xs"
              variant="transparent"
              onClick={() => setBrowseExpanded(!browseExpanded)}
              style={{ color: 'var(--sidebar-section-title)' }}
            >
              <IconChevronRight
                size={12}
                style={{
                  transform: browseExpanded ? 'rotate(90deg)' : 'rotate(0deg)',
                  transition: 'transform var(--transition)',
                }}
              />
            </ActionIcon>
          </Box>

          <Collapse in={browseExpanded}>
            <NavItem icon={IconFileAnalytics} label="Models" path="/browse/models" />
            <NavItem icon={IconDatabase} label="Databases" path="/browse/databases" />
          </Collapse>

          {/* Add your own data button - Metabase style */}
          <Button
            fullWidth
            leftSection={<IconPlus size={16} />}
            variant="filled"
            color="brand"
            size="sm"
            radius="md"
            style={{ margin: '12px 0' }}
            onClick={() => navigate('/admin/datamodel?add=true')}
          >
            Add your own data
          </Button>
        </Box>

        {/* Bookmarks Section */}
        {bookmarks.length > 0 && (
          <Box style={{ padding: `${rem(4)} ${rem(8)}` }}>
            <Box
              style={{
                padding: `${rem(12)} ${rem(12)} ${rem(4)}`,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
              }}
            >
              <Group gap="xs">
                <IconStar size={14} style={{ color: 'var(--sidebar-section-title)' }} strokeWidth={1.75} />
                <Text
                  style={{
                    fontSize: rem(11),
                    fontWeight: 700,
                    textTransform: 'uppercase',
                    letterSpacing: '0.05em',
                    color: 'var(--sidebar-section-title)',
                  }}
                >
                  Bookmarks
                </Text>
              </Group>
              <ActionIcon
                size="xs"
                variant="transparent"
                onClick={() => setBookmarksExpanded(!bookmarksExpanded)}
                style={{ color: 'var(--sidebar-section-title)' }}
              >
                <IconChevronRight
                  size={12}
                  style={{
                    transform: bookmarksExpanded ? 'rotate(90deg)' : 'rotate(0deg)',
                    transition: 'transform var(--transition)',
                  }}
                />
              </ActionIcon>
            </Box>
            <Collapse in={bookmarksExpanded}>
              {bookmarks.slice(0, 5).map((bookmark) => {
                const Icon = getItemIcon(bookmark.type)
                const path = `/${bookmark.type}/${bookmark.id}`
                return (
                  <NavItem
                    key={bookmark.id}
                    icon={Icon}
                    label={bookmark.name}
                    path={path}
                  />
                )
              })}
            </Collapse>
          </Box>
        )}

        {/* Recents Section */}
        {recentItems.length > 0 && (
          <Box style={{ padding: `${rem(4)} ${rem(8)}` }}>
            <Box
              style={{
                padding: `${rem(12)} ${rem(12)} ${rem(4)}`,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
              }}
            >
              <Group gap="xs">
                <IconClock size={14} style={{ color: 'var(--sidebar-section-title)' }} strokeWidth={1.75} />
                <Text
                  style={{
                    fontSize: rem(11),
                    fontWeight: 700,
                    textTransform: 'uppercase',
                    letterSpacing: '0.05em',
                    color: 'var(--sidebar-section-title)',
                  }}
                >
                  Recents
                </Text>
              </Group>
              <ActionIcon
                size="xs"
                variant="transparent"
                onClick={() => setRecentsExpanded(!recentsExpanded)}
                style={{ color: 'var(--sidebar-section-title)' }}
              >
                <IconChevronRight
                  size={12}
                  style={{
                    transform: recentsExpanded ? 'rotate(90deg)' : 'rotate(0deg)',
                    transition: 'transform var(--transition)',
                  }}
                />
              </ActionIcon>
            </Box>
            <Collapse in={recentsExpanded}>
              {recentItems.slice(0, 5).map((item) => {
                const Icon = getItemIcon(item.type)
                const path = item.type === 'table' ? '/browse/databases' : `/${item.type}/${item.id}`
                return (
                  <NavItem
                    key={`${item.type}-${item.id}`}
                    icon={Icon}
                    label={item.name}
                    path={path}
                  />
                )
              })}
            </Collapse>
          </Box>
        )}

        {/* Collections Section */}
        <Box style={{ padding: `${rem(4)} ${rem(8)}` }}>
          <Box
            style={{
              padding: `${rem(12)} ${rem(12)} ${rem(4)}`,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
            }}
          >
            <Text
              style={{
                fontSize: rem(11),
                fontWeight: 700,
                textTransform: 'uppercase',
                letterSpacing: '0.05em',
                color: 'var(--sidebar-section-title)',
              }}
            >
              COLLECTIONS
            </Text>
            <Group gap={4}>
              <Menu position="bottom-end" width={180}>
                <Menu.Target>
                  <ActionIcon
                    size="xs"
                    variant="transparent"
                    style={{ color: 'var(--sidebar-section-title)' }}
                  >
                    <IconDots size={14} strokeWidth={2} />
                  </ActionIcon>
                </Menu.Target>
                <Menu.Dropdown>
                  <Menu.Item
                    leftSection={<IconPlus size={14} />}
                    onClick={() => navigate('/collection/new')}
                  >
                    New collection
                  </Menu.Item>
                  <Menu.Item
                    leftSection={<IconFolder size={14} />}
                    onClick={() => navigate('/browse')}
                  >
                    Browse all
                  </Menu.Item>
                </Menu.Dropdown>
              </Menu>
              <ActionIcon
                size="xs"
                variant="transparent"
                onClick={() => setCollectionsExpanded(!collectionsExpanded)}
                style={{ color: 'var(--sidebar-section-title)' }}
              >
                <IconChevronRight
                  size={12}
                  style={{
                    transform: collectionsExpanded ? 'rotate(90deg)' : 'rotate(0deg)',
                    transition: 'transform var(--transition)',
                  }}
                />
              </ActionIcon>
            </Group>
          </Box>
          <Collapse in={collectionsExpanded}>
            {/* Our analytics (root collection) - Metabase style */}
            <NavItem
              icon={IconFolder}
              label="Our analytics"
              path="/collection/root"
              iconColor="#7172AD"
            />

            {/* Your personal collection - Metabase style */}
            <NavItem
              icon={IconUser}
              label="Your personal collection"
              path="/collection/personal"
              iconColor="#509EE3"
            />

            {/* User-created collections */}
            {rootCollections.map(collection => (
              <CollectionItem
                key={collection.id}
                id={collection.id}
                name={collection.name}
                color={collection.color}
                collections={collections || []}
              />
            ))}
          </Collapse>
        </Box>

        {/* Admin Section */}
        {user?.role === 'admin' && (
          <Box style={{ padding: `${rem(4)} ${rem(8)}` }}>
            <Divider mb="sm" style={{ borderColor: 'var(--sidebar-border)' }} />
            <NavItem
              icon={IconSettings2}
              label="Admin"
              onClick={() => setAdminExpanded(!adminExpanded)}
              rightSection={
                <IconChevronRight
                  size={14}
                  style={{
                    color: 'var(--sidebar-icon)',
                    transform: adminExpanded ? 'rotate(90deg)' : 'rotate(0deg)',
                    transition: 'transform var(--transition)',
                  }}
                />
              }
            />
            <Collapse in={adminExpanded}>
              <Box pl={20}>
                <NavItem icon={IconDatabase} label="Data Model" path="/admin/datamodel" />
                <NavItem icon={IconUsers} label="People" path="/admin/people" />
                <NavItem icon={IconSettings2} label="Settings" path="/admin/settings" />
              </Box>
            </Collapse>
          </Box>
        )}
      </ScrollArea>

      {/* User Section */}
      <Box
        style={{
          padding: rem(12),
          borderTop: '1px solid var(--sidebar-border)',
          marginTop: 'auto',
        }}
      >
        <Menu position="top-start" width={220} shadow="md">
          <Menu.Target>
            <UnstyledButton
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: rem(12),
                padding: rem(8),
                borderRadius: 'var(--radius-md)',
                width: '100%',
                transition: 'background-color var(--transition-fast)',
              }}
              onMouseEnter={(e) => {
                e.currentTarget.style.backgroundColor = 'var(--sidebar-bg-hover)'
              }}
              onMouseLeave={(e) => {
                e.currentTarget.style.backgroundColor = 'transparent'
              }}
            >
              <Box
                style={{
                  width: 36,
                  height: 36,
                  borderRadius: '50%',
                  backgroundColor: 'var(--color-primary)',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  color: '#ffffff',
                  fontWeight: 600,
                  fontSize: rem(14),
                }}
              >
                {user?.name?.charAt(0).toUpperCase() || 'U'}
              </Box>
              <Box style={{ flex: 1, overflow: 'hidden' }}>
                <Text size="sm" truncate fw={500} style={{ color: 'var(--sidebar-text)' }}>
                  {user?.name || 'User'}
                </Text>
                <Text size="xs" truncate style={{ color: 'var(--sidebar-text-secondary)' }}>
                  {user?.email || ''}
                </Text>
              </Box>
            </UnstyledButton>
          </Menu.Target>
          <Menu.Dropdown>
            <Menu.Label>
              <Text size="sm" fw={500}>{user?.name}</Text>
              <Text size="xs" c="dimmed">{user?.email}</Text>
            </Menu.Label>
            <Menu.Divider />
            <Menu.Item
              leftSection={<IconSettings2 size={16} strokeWidth={1.75} />}
              onClick={() => navigate('/admin/settings')}
            >
              Account settings
            </Menu.Item>
            <Menu.Divider />
            <Menu.Item
              leftSection={<IconLogout size={16} strokeWidth={1.75} />}
              color="red"
              onClick={() => logout()}
            >
              Log out
            </Menu.Item>
          </Menu.Dropdown>
        </Menu>
      </Box>
    </Box>
  )
}
