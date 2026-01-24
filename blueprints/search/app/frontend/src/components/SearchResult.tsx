import { Text, Anchor, Group, ActionIcon, Menu } from '@mantine/core'
import { IconDotsVertical, IconThumbUp, IconThumbDown, IconBan } from '@tabler/icons-react'
import type { SearchResult as SearchResultType } from '../types'
import { searchApi } from '../api/search'
import { useSearchStore } from '../stores/searchStore'

interface SearchResultProps {
  result: SearchResultType
  openInNewTab?: boolean
}

export function SearchResult({ result, openInNewTab }: SearchResultProps) {
  const { settings } = useSearchStore()

  const handlePreference = async (action: 'upvote' | 'downvote' | 'block') => {
    try {
      await searchApi.setPreference(result.domain, action)
    } catch (error) {
      console.error('Failed to set preference:', error)
    }
  }

  // Parse URL for display
  const displayUrl = (() => {
    try {
      const url = new URL(result.url)
      const pathParts = url.pathname.split('/').filter(Boolean)
      return [url.hostname, ...pathParts.slice(0, 2)].join(' > ')
    } catch {
      return result.url
    }
  })()

  return (
    <div className="result-card group">
      <Group justify="space-between" align="flex-start">
        <div className="flex-1">
          {/* URL breadcrumb */}
          <Group gap={4} className="mb-1">
            {result.favicon && (
              <img
                src={result.favicon}
                alt=""
                className="w-4 h-4 rounded"
                onError={(e) => {
                  (e.target as HTMLImageElement).style.display = 'none'
                }}
              />
            )}
            <Text size="xs" c="dimmed" className="truncate max-w-md">
              {displayUrl}
            </Text>
          </Group>

          {/* Title */}
          <Anchor
            href={result.url}
            target={openInNewTab || settings.open_in_new_tab ? '_blank' : '_self'}
            rel="noopener noreferrer"
            className="result-title"
            underline="never"
          >
            <Text size="lg" c="blue" className="hover:underline">
              {result.title}
            </Text>
          </Anchor>

          {/* Snippet */}
          <Text
            size="sm"
            c="dimmed"
            className="mt-1 snippet"
            dangerouslySetInnerHTML={{ __html: result.snippet }}
          />

          {/* Sitelinks */}
          {result.sitelinks && result.sitelinks.length > 0 && (
            <Group gap="md" className="mt-2">
              {result.sitelinks.map((link) => (
                <Anchor
                  key={link.url}
                  href={link.url}
                  size="sm"
                  target={openInNewTab || settings.open_in_new_tab ? '_blank' : '_self'}
                  rel="noopener noreferrer"
                >
                  {link.title}
                </Anchor>
              ))}
            </Group>
          )}
        </div>

        {/* Actions */}
        <Menu position="bottom-end" withArrow>
          <Menu.Target>
            <ActionIcon
              variant="subtle"
              color="gray"
              className="opacity-0 group-hover:opacity-100 transition-opacity"
            >
              <IconDotsVertical size={16} />
            </ActionIcon>
          </Menu.Target>
          <Menu.Dropdown>
            <Menu.Item
              leftSection={<IconThumbUp size={16} />}
              onClick={() => handlePreference('upvote')}
            >
              Raise {result.domain} in rankings
            </Menu.Item>
            <Menu.Item
              leftSection={<IconThumbDown size={16} />}
              onClick={() => handlePreference('downvote')}
            >
              Lower {result.domain} in rankings
            </Menu.Item>
            <Menu.Divider />
            <Menu.Item
              leftSection={<IconBan size={16} />}
              color="red"
              onClick={() => handlePreference('block')}
            >
              Block {result.domain}
            </Menu.Item>
          </Menu.Dropdown>
        </Menu>
      </Group>
    </div>
  )
}
