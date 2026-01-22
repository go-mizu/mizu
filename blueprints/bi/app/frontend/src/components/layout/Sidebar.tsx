import { useState } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import {
  Box, Text, TextInput, Group, UnstyledButton, Collapse, ActionIcon,
  Tooltip, Menu, ScrollArea, Divider, Badge, rem
} from '@mantine/core'
import {
  IconHome2, IconLayoutGrid, IconPlus, IconLayoutDashboard, IconSettings2, IconSearch,
  IconDatabase, IconUsers, IconChevronRight, IconChevronDown, IconLogout,
  IconFileAnalytics, IconChartLine, IconClock, IconStar, IconChevronLeft,
  IconTable, IconFolder, IconFolderFilled, IconSparkles, IconPencil
} from '@tabler/icons-react'
import { useCollections, useCurrentUser, useLogout } from '../../api/hooks'
import { useUIStore } from '../../stores/uiStore'
import { useBookmarkStore } from '../../stores/bookmarkStore'
import { sidebarTheme, semanticColors } from '../../theme'

// =============================================================================
// SIDEBAR STYLES - Metabase Light Theme
// =============================================================================

const styles = {
  sidebar: {
    width: 260,
    height: '100vh',
    backgroundColor: sidebarTheme.bg,
    display: 'flex',
    flexDirection: 'column' as const,
    borderRight: `1px solid ${sidebarTheme.border}`,
    transition: 'width 0.2s ease',
  },
  sidebarCollapsed: {
    width: 60,
  },
  header: {
    padding: `${rem(16)} ${rem(16)}`,
    borderBottom: `1px solid ${sidebarTheme.border}`,
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'space-between',
    minHeight: 64,
  },
  logo: {
    display: 'flex',
    alignItems: 'center',
    gap: rem(10),
    cursor: 'pointer',
  },
  logoIcon: {
    width: 32,
    height: 32,
    borderRadius: rem(8),
    background: `linear-gradient(135deg, ${semanticColors.brand} 0%, #3B7DBF 100%)`,
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
  },
  searchInput: {
    padding: `${rem(8)} ${rem(16)}`,
  },
  navSection: {
    padding: `${rem(8)} ${rem(12)}`,
  },
  navItem: {
    display: 'flex',
    alignItems: 'center',
    gap: rem(10),
    padding: `${rem(10)} ${rem(12)}`,
    borderRadius: rem(8),
    color: sidebarTheme.text,
    fontSize: rem(14),
    fontWeight: 500,
    cursor: 'pointer',
    transition: 'all 0.15s ease',
    marginBottom: rem(2),
  },
  navItemActive: {
    backgroundColor: sidebarTheme.bgActive,
    color: sidebarTheme.textActive,
  },
  navItemHover: {
    backgroundColor: sidebarTheme.bgHover,
    color: sidebarTheme.textHover,
  },
  sectionHeader: {
    padding: `${rem(12)} ${rem(12)} ${rem(4)}`,
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'space-between',
  },
  sectionTitle: {
    fontSize: rem(11),
    fontWeight: 700,
    textTransform: 'uppercase' as const,
    letterSpacing: '0.05em',
    color: sidebarTheme.sectionTitle,
  },
  userSection: {
    padding: rem(12),
    borderTop: `1px solid ${sidebarTheme.border}`,
    marginTop: 'auto',
  },
  userButton: {
    display: 'flex',
    alignItems: 'center',
    gap: rem(12),
    padding: rem(8),
    borderRadius: rem(8),
    width: '100%',
  },
  userAvatar: {
    width: 36,
    height: 36,
    borderRadius: '50%',
    backgroundColor: semanticColors.brand,
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    color: '#ffffff',
    fontWeight: 600,
    fontSize: rem(14),
  },
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
  collapsed?: boolean
}

function CollectionItem({ id, name, color, depth = 0, collections, collapsed }: CollectionItemProps) {
  const navigate = useNavigate()
  const location = useLocation()
  const [expanded, setExpanded] = useState(false)
  const children = collections?.filter(c => c.parent_id === id) || []
  const hasChildren = children.length > 0
  const isActive = location.pathname === `/collection/${id}`

  if (collapsed) return null

  return (
    <Box>
      <UnstyledButton
        onClick={() => navigate(`/collection/${id}`)}
        style={{
          ...styles.navItem,
          ...(isActive ? styles.navItemActive : {}),
          paddingLeft: rem(12 + depth * 16),
        }}
        onMouseEnter={(e) => {
          if (!isActive) {
            e.currentTarget.style.backgroundColor = sidebarTheme.bgHover
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
            style={{ color: sidebarTheme.iconDefault }}
          >
            {expanded ? <IconChevronDown size={14} /> : <IconChevronRight size={14} />}
          </ActionIcon>
        )}
        {!hasChildren && <Box w={18} />}
        {expanded ? (
          <IconFolderFilled size={18} style={{ color: color || sidebarTheme.newCollection, flexShrink: 0 }} />
        ) : (
          <IconFolder size={18} style={{ color: color || sidebarTheme.iconDefault, flexShrink: 0 }} />
        )}
        <Text size="sm" truncate style={{ flex: 1, color: isActive ? sidebarTheme.textActive : sidebarTheme.text }}>
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
  collapsed?: boolean
  iconColor?: string
}

function NavItem({ icon: Icon, label, path, onClick, active, rightSection, collapsed, iconColor }: NavItemProps) {
  const navigate = useNavigate()
  const location = useLocation()
  const isActive = active ?? (path ? location.pathname === path || location.pathname.startsWith(path + '/') : false)

  const handleClick = () => {
    if (onClick) onClick()
    else if (path) navigate(path)
  }

  return (
    <Tooltip label={label} disabled={!collapsed} position="right">
      <UnstyledButton
        onClick={handleClick}
        style={{
          ...styles.navItem,
          ...(isActive ? styles.navItemActive : {}),
          justifyContent: collapsed ? 'center' : 'flex-start',
          padding: collapsed ? rem(12) : `${rem(10)} ${rem(12)}`,
        }}
        onMouseEnter={(e) => {
          if (!isActive) {
            e.currentTarget.style.backgroundColor = sidebarTheme.bgHover
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
            color: isActive ? sidebarTheme.iconActive : (iconColor || sidebarTheme.iconDefault),
            strokeWidth: 1.75
          }}
        />
        {!collapsed && (
          <>
            <Text size="sm" style={{ flex: 1, color: isActive ? sidebarTheme.textActive : sidebarTheme.text }}>
              {label}
            </Text>
            {rightSection}
          </>
        )}
      </UnstyledButton>
    </Tooltip>
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

  const collapsed = sidebarCollapsed

  // Get icon for item type
  const getItemIcon = (type: string) => {
    switch (type) {
      case 'dashboard': return IconLayoutDashboard
      case 'question': return IconChartLine
      case 'collection': return IconFolder
      case 'table': return IconTable
      default: return IconChartLine
    }
  }

  return (
    <Box
      component="aside"
      style={{
        ...styles.sidebar,
        ...(collapsed ? styles.sidebarCollapsed : {}),
      }}
    >
      {/* Header with Logo and New Button */}
      <Box style={styles.header}>
        <UnstyledButton onClick={() => navigate('/')} style={styles.logo}>
          <Box style={styles.logoIcon}>
            <IconSparkles size={18} color="#ffffff" strokeWidth={2} />
          </Box>
          {!collapsed && (
            <Text size="lg" fw={700} style={{ color: sidebarTheme.text }}>
              Metabase
            </Text>
          )}
        </UnstyledButton>

        {!collapsed && (
          <Group gap={4}>
            {/* New Button with Dropdown */}
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
                  leftSection={<IconPencil size={16} color={semanticColors.brand} strokeWidth={1.75} />}
                  onClick={() => navigate('/question/new')}
                >
                  New question
                </Menu.Item>
                <Menu.Item
                  leftSection={<IconLayoutDashboard size={16} color={semanticColors.summarize} strokeWidth={1.75} />}
                  onClick={() => navigate('/dashboard/new')}
                >
                  New dashboard
                </Menu.Item>
                <Menu.Divider />
                <Menu.Item
                  leftSection={<IconFolder size={16} color={sidebarTheme.newCollection} strokeWidth={1.75} />}
                  onClick={() => navigate('/collection/new')}
                >
                  New collection
                </Menu.Item>
              </Menu.Dropdown>
            </Menu>

            {/* Collapse Button */}
            <Tooltip label="Collapse sidebar">
              <ActionIcon
                variant="subtle"
                size="md"
                onClick={toggleSidebar}
                style={{ color: sidebarTheme.iconDefault }}
              >
                <IconChevronLeft size={18} strokeWidth={1.75} />
              </ActionIcon>
            </Tooltip>
          </Group>
        )}

        {collapsed && (
          <Tooltip label="Expand sidebar" position="right">
            <ActionIcon
              variant="subtle"
              size="md"
              onClick={toggleSidebar}
              style={{ color: sidebarTheme.iconDefault, position: 'absolute', top: '50%', left: '50%', transform: 'translate(-50%, -50%)' }}
            >
              <IconChevronRight size={18} strokeWidth={1.75} />
            </ActionIcon>
          </Tooltip>
        )}
      </Box>

      {/* Search */}
      {!collapsed && (
        <Box style={styles.searchInput}>
          <TextInput
            placeholder="Search..."
            leftSection={<IconSearch size={16} color={sidebarTheme.inputPlaceholder} strokeWidth={1.75} />}
            size="sm"
            onClick={openCommandPalette}
            readOnly
            rightSection={
              <Badge size="xs" variant="light" color="gray" style={{ cursor: 'pointer' }}>
                ⌘K
              </Badge>
            }
            styles={{
              input: {
                backgroundColor: sidebarTheme.inputBg,
                border: `1px solid ${sidebarTheme.inputBorder}`,
                color: sidebarTheme.inputText,
                cursor: 'pointer',
                '&::placeholder': {
                  color: sidebarTheme.inputPlaceholder,
                },
                '&:hover': {
                  backgroundColor: '#F0F0F0',
                  borderColor: '#E0E0E0',
                },
              },
            }}
          />
        </Box>
      )}

      {/* Main Navigation */}
      <ScrollArea style={{ flex: 1 }} scrollbarSize={6}>
        <Box style={styles.navSection}>
          {/* Home */}
          <NavItem icon={IconHome2} label="Home" path="/" collapsed={collapsed} />

          {/* Browse with submenu */}
          <NavItem
            icon={IconLayoutGrid}
            label="Browse"
            onClick={() => {
              if (!collapsed) setBrowseExpanded(!browseExpanded)
              else navigate('/browse')
            }}
            active={location.pathname.startsWith('/browse') && location.pathname === '/browse'}
            rightSection={!collapsed && (
              <IconChevronRight
                size={14}
                style={{
                  color: sidebarTheme.iconDefault,
                  transform: browseExpanded ? 'rotate(90deg)' : 'rotate(0deg)',
                  transition: 'transform 0.2s ease'
                }}
              />
            )}
            collapsed={collapsed}
          />

          {!collapsed && (
            <Collapse in={browseExpanded}>
              <Box pl={20}>
                <NavItem icon={IconDatabase} label="Databases" path="/browse/databases" />
                <NavItem icon={IconFileAnalytics} label="Models" path="/browse/models" />
                <NavItem icon={IconChartLine} label="Metrics" path="/browse/metrics" />
              </Box>
            </Collapse>
          )}

          <Divider my="sm" color={sidebarTheme.border} />

          {/* Quick Create */}
          <NavItem
            icon={IconPencil}
            label="New question"
            path="/question/new"
            iconColor={sidebarTheme.newQuestion}
            collapsed={collapsed}
          />
          <NavItem
            icon={IconLayoutDashboard}
            label="New dashboard"
            path="/dashboard/new"
            iconColor={sidebarTheme.newDashboard}
            collapsed={collapsed}
          />
        </Box>

        {/* Bookmarks Section */}
        {!collapsed && bookmarks.length > 0 && (
          <Box style={styles.navSection}>
            <Divider mb="sm" color={sidebarTheme.border} />
            <Box style={styles.sectionHeader}>
              <Group gap="xs">
                <IconStar size={14} style={{ color: sidebarTheme.sectionTitle }} strokeWidth={1.75} />
                <Text style={styles.sectionTitle}>Bookmarks</Text>
              </Group>
              <ActionIcon
                size="xs"
                variant="transparent"
                onClick={() => setBookmarksExpanded(!bookmarksExpanded)}
                style={{ color: sidebarTheme.sectionTitle }}
              >
                <IconChevronRight
                  size={12}
                  style={{
                    transform: bookmarksExpanded ? 'rotate(90deg)' : 'rotate(0deg)',
                    transition: 'transform 0.2s ease'
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
        {!collapsed && recentItems.length > 0 && (
          <Box style={styles.navSection}>
            <Divider mb="sm" color={sidebarTheme.border} />
            <Box style={styles.sectionHeader}>
              <Group gap="xs">
                <IconClock size={14} style={{ color: sidebarTheme.sectionTitle }} strokeWidth={1.75} />
                <Text style={styles.sectionTitle}>Recents</Text>
              </Group>
              <ActionIcon
                size="xs"
                variant="transparent"
                onClick={() => setRecentsExpanded(!recentsExpanded)}
                style={{ color: sidebarTheme.sectionTitle }}
              >
                <IconChevronRight
                  size={12}
                  style={{
                    transform: recentsExpanded ? 'rotate(90deg)' : 'rotate(0deg)',
                    transition: 'transform 0.2s ease'
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
        {!collapsed && rootCollections.length > 0 && (
          <Box style={styles.navSection}>
            <Divider mb="sm" color={sidebarTheme.border} />
            <Box style={styles.sectionHeader}>
              <Group gap="xs">
                <IconFolder size={14} style={{ color: sidebarTheme.sectionTitle }} strokeWidth={1.75} />
                <Text style={styles.sectionTitle}>Collections</Text>
              </Group>
              <Group gap={4}>
                <Tooltip label="New collection">
                  <ActionIcon
                    size="xs"
                    variant="transparent"
                    onClick={() => navigate('/collection/new')}
                    style={{ color: sidebarTheme.sectionTitle }}
                  >
                    <IconPlus size={12} strokeWidth={2} />
                  </ActionIcon>
                </Tooltip>
                <ActionIcon
                  size="xs"
                  variant="transparent"
                  onClick={() => setCollectionsExpanded(!collectionsExpanded)}
                  style={{ color: sidebarTheme.sectionTitle }}
                >
                  <IconChevronRight
                    size={12}
                    style={{
                      transform: collectionsExpanded ? 'rotate(90deg)' : 'rotate(0deg)',
                      transition: 'transform 0.2s ease'
                    }}
                  />
                </ActionIcon>
              </Group>
            </Box>
            <Collapse in={collectionsExpanded}>
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
        )}

        {/* Admin Section */}
        {user?.role === 'admin' && (
          <Box style={styles.navSection}>
            <Divider mb="sm" color={sidebarTheme.border} />
            <NavItem
              icon={IconSettings2}
              label="Admin"
              onClick={() => {
                if (!collapsed) setAdminExpanded(!adminExpanded)
                else navigate('/admin/datamodel')
              }}
              rightSection={!collapsed && (
                <IconChevronRight
                  size={14}
                  style={{
                    color: sidebarTheme.iconDefault,
                    transform: adminExpanded ? 'rotate(90deg)' : 'rotate(0deg)',
                    transition: 'transform 0.2s ease'
                  }}
                />
              )}
              collapsed={collapsed}
            />
            {!collapsed && (
              <Collapse in={adminExpanded}>
                <Box pl={20}>
                  <NavItem icon={IconDatabase} label="Data Model" path="/admin/datamodel" />
                  <NavItem icon={IconUsers} label="People" path="/admin/people" />
                  <NavItem icon={IconSettings2} label="Settings" path="/admin/settings" />
                </Box>
              </Collapse>
            )}
          </Box>
        )}
      </ScrollArea>

      {/* User Section */}
      <Box style={styles.userSection}>
        <Menu position="top-start" width={220} shadow="md">
          <Menu.Target>
            <UnstyledButton
              style={{
                ...styles.userButton,
                justifyContent: collapsed ? 'center' : 'flex-start',
              }}
              onMouseEnter={(e) => {
                e.currentTarget.style.backgroundColor = sidebarTheme.bgHover
              }}
              onMouseLeave={(e) => {
                e.currentTarget.style.backgroundColor = 'transparent'
              }}
            >
              <Box style={styles.userAvatar}>
                {user?.name?.charAt(0).toUpperCase() || 'U'}
              </Box>
              {!collapsed && (
                <Box style={{ flex: 1, overflow: 'hidden' }}>
                  <Text size="sm" truncate fw={500} style={{ color: sidebarTheme.text }}>
                    {user?.name || 'User'}
                  </Text>
                  <Text size="xs" truncate style={{ color: sidebarTheme.textSecondary }}>
                    {user?.email || ''}
                  </Text>
                </Box>
              )}
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
