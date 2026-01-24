import { Paper, Text, Image, Group, Anchor, Stack, Divider } from '@mantine/core'
import { IconExternalLink } from '@tabler/icons-react'
import type { KnowledgePanel as KnowledgePanelType } from '../types'

interface KnowledgePanelProps {
  panel: KnowledgePanelType
}

export function KnowledgePanel({ panel }: KnowledgePanelProps) {
  return (
    <Paper className="knowledge-panel" p="lg">
      {panel.image && (
        <Image
          src={panel.image}
          alt={panel.title}
          radius="md"
          mb="md"
          mah={200}
          fit="contain"
        />
      )}

      <Text size="xl" fw={600}>
        {panel.title}
      </Text>

      {panel.subtitle && (
        <Text size="sm" c="dimmed">
          {panel.subtitle}
        </Text>
      )}

      <Text size="sm" mt="md" lineClamp={5}>
        {panel.description}
      </Text>

      {panel.facts && panel.facts.length > 0 && (
        <>
          <Divider my="md" />
          <Stack gap="xs">
            {panel.facts.map((fact) => (
              <Group key={fact.label} justify="space-between" gap="xs">
                <Text size="sm" c="dimmed">
                  {fact.label}
                </Text>
                <Text size="sm" fw={500}>
                  {fact.value}
                </Text>
              </Group>
            ))}
          </Stack>
        </>
      )}

      {panel.links && panel.links.length > 0 && (
        <>
          <Divider my="md" />
          <Stack gap="xs">
            {panel.links.map((link) => (
              <Anchor
                key={link.url}
                href={link.url}
                target="_blank"
                rel="noopener noreferrer"
                size="sm"
              >
                <Group gap={4}>
                  <IconExternalLink size={14} />
                  {link.title}
                </Group>
              </Anchor>
            ))}
          </Stack>
        </>
      )}

      {panel.source && (
        <Text size="xs" c="dimmed" mt="md">
          Source: {panel.source}
        </Text>
      )}
    </Paper>
  )
}
