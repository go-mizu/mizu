import { useState } from 'react'
import {
  Container,
  Stack,
  Text,
  Paper,
  Group,
  ActionIcon,
  TextInput,
  Button,
  Menu,
} from '@mantine/core'
import {
  IconArrowLeft,
  IconSearch,
  IconTrash,
  IconDotsVertical,
  IconClock,
} from '@tabler/icons-react'
import { Link, useNavigate } from 'react-router-dom'
import { useSearchStore } from '../stores/searchStore'

export default function HistoryPage() {
  const navigate = useNavigate()
  const { recentSearches, removeRecentSearch, clearRecentSearches } = useSearchStore()
  const [filter, setFilter] = useState('')

  const filteredSearches = recentSearches.filter((search) =>
    search.toLowerCase().includes(filter.toLowerCase())
  )

  const handleSearchClick = (query: string) => {
    navigate(`/search?q=${encodeURIComponent(query)}`)
  }

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Header */}
      <header className="bg-white border-b">
        <Container size="md" className="py-4">
          <Group justify="space-between">
            <Group>
              <Link to="/">
                <ActionIcon variant="subtle" color="gray" size="lg">
                  <IconArrowLeft size={20} />
                </ActionIcon>
              </Link>
              <Text size="xl" fw={600}>
                Search History
              </Text>
            </Group>

            {recentSearches.length > 0 && (
              <Button
                variant="subtle"
                color="red"
                size="sm"
                leftSection={<IconTrash size={16} />}
                onClick={clearRecentSearches}
              >
                Clear all
              </Button>
            )}
          </Group>
        </Container>
      </header>

      {/* Main content */}
      <main>
        <Container size="md" className="py-6">
          {recentSearches.length === 0 ? (
            <Paper p="xl" withBorder className="text-center">
              <IconClock size={48} className="mx-auto mb-4 text-gray-400" />
              <Text size="lg" fw={500} mb="xs">
                No search history
              </Text>
              <Text c="dimmed">
                Your recent searches will appear here
              </Text>
              <Button
                component={Link}
                to="/"
                variant="light"
                mt="lg"
              >
                Start searching
              </Button>
            </Paper>
          ) : (
            <Stack gap="md">
              {/* Search filter */}
              <TextInput
                placeholder="Filter history..."
                leftSection={<IconSearch size={16} />}
                value={filter}
                onChange={(e) => setFilter(e.currentTarget.value)}
              />

              {/* History list */}
              <Paper withBorder>
                {filteredSearches.length === 0 ? (
                  <div className="p-8 text-center">
                    <Text c="dimmed">No matches found</Text>
                  </div>
                ) : (
                  <Stack gap={0}>
                    {filteredSearches.map((search, index) => (
                      <div
                        key={`${search}-${index}`}
                        className={`flex items-center justify-between px-4 py-3 hover:bg-gray-50 cursor-pointer ${
                          index > 0 ? 'border-t' : ''
                        }`}
                        onClick={() => handleSearchClick(search)}
                      >
                        <Group gap="sm">
                          <IconClock size={18} className="text-gray-400" />
                          <Text>{search}</Text>
                        </Group>

                        <Menu position="bottom-end" withinPortal>
                          <Menu.Target>
                            <ActionIcon
                              variant="subtle"
                              color="gray"
                              onClick={(e) => e.stopPropagation()}
                            >
                              <IconDotsVertical size={16} />
                            </ActionIcon>
                          </Menu.Target>
                          <Menu.Dropdown>
                            <Menu.Item
                              leftSection={<IconSearch size={14} />}
                              onClick={(e) => {
                                e.stopPropagation()
                                handleSearchClick(search)
                              }}
                            >
                              Search again
                            </Menu.Item>
                            <Menu.Item
                              color="red"
                              leftSection={<IconTrash size={14} />}
                              onClick={(e) => {
                                e.stopPropagation()
                                removeRecentSearch(search)
                              }}
                            >
                              Remove
                            </Menu.Item>
                          </Menu.Dropdown>
                        </Menu>
                      </div>
                    ))}
                  </Stack>
                )}
              </Paper>

              {/* Stats */}
              <Text size="sm" c="dimmed" ta="center">
                {filteredSearches.length} of {recentSearches.length} searches shown
              </Text>
            </Stack>
          )}
        </Container>
      </main>
    </div>
  )
}
