import { Group, Title, ActionIcon, Text, Stack, Breadcrumbs, Anchor } from '@mantine/core'
import { IconArrowLeft } from '@tabler/icons-react'
import { useNavigate, Link } from 'react-router-dom'
import type { ReactNode } from 'react'

interface Breadcrumb {
  label: string
  path?: string
}

interface PageHeaderProps {
  title: string
  subtitle?: string
  breadcrumbs?: Breadcrumb[]
  backPath?: string
  actions?: ReactNode
}

export function PageHeader({ title, subtitle, breadcrumbs, backPath, actions }: PageHeaderProps) {
  const navigate = useNavigate()

  return (
    <Stack gap="xs" mb="lg">
      {breadcrumbs && breadcrumbs.length > 0 && (
        <Breadcrumbs separator="/">
          {breadcrumbs.map((crumb, idx) => (
            crumb.path ? (
              <Anchor key={idx} component={Link} to={crumb.path} size="sm" c="dimmed">
                {crumb.label}
              </Anchor>
            ) : (
              <Text key={idx} size="sm" c="dimmed">
                {crumb.label}
              </Text>
            )
          ))}
        </Breadcrumbs>
      )}

      <Group justify="space-between" align="flex-start">
        <Group gap="md">
          {backPath && (
            <ActionIcon
              variant="subtle"
              size="lg"
              onClick={() => navigate(backPath)}
              aria-label="Go back"
            >
              <IconArrowLeft size={20} />
            </ActionIcon>
          )}
          <Stack gap={2}>
            <Title order={2}>{title}</Title>
            {subtitle && (
              <Text size="sm" c="dimmed">
                {subtitle}
              </Text>
            )}
          </Stack>
        </Group>
        {actions && <Group gap="sm">{actions}</Group>}
      </Group>
    </Stack>
  )
}
