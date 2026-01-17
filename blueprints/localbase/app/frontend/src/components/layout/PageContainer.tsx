import { Box, Group, Title, Text, Skeleton } from '@mantine/core';
import type { ReactNode } from 'react';

interface PageContainerProps {
  title: string;
  description?: string;
  actions?: ReactNode;
  children: ReactNode;
  loading?: boolean;
  fullWidth?: boolean;
  noPadding?: boolean;
  noHeader?: boolean;
}

export function PageContainer({
  title,
  description,
  actions,
  children,
  loading = false,
  fullWidth = false,
  noPadding = false,
  noHeader = false,
}: PageContainerProps) {
  return (
    <Box
      p={noPadding ? 0 : 'lg'}
      style={{
        maxWidth: fullWidth ? '100%' : 1400,
        margin: '0 auto',
        height: '100%',
        display: 'flex',
        flexDirection: 'column',
      }}
    >
      {/* Page Header */}
      {!noHeader && (
        <Box mb="lg" px={noPadding ? 'lg' : 0} pt={noPadding ? 'lg' : 0}>
          <Group justify="space-between" align="flex-start">
            <Box>
              {loading ? (
                <>
                  <Skeleton height={32} width={200} mb="xs" />
                  {description && <Skeleton height={20} width={300} />}
                </>
              ) : (
                <>
                  <Title order={2} className="supabase-page-title">
                    {title}
                  </Title>
                  {description && (
                    <Text className="supabase-page-description" mt={4}>
                      {description}
                    </Text>
                  )}
                </>
              )}
            </Box>
            {actions && <Group gap="sm">{actions}</Group>}
          </Group>
        </Box>
      )}

      {/* Page Content */}
      <Box style={{ flex: 1, minHeight: 0 }}>{children}</Box>
    </Box>
  );
}

interface PageSectionProps {
  title?: string;
  description?: string;
  actions?: ReactNode;
  children: ReactNode;
  noPadding?: boolean;
}

export function PageSection({
  title,
  description,
  actions,
  children,
  noPadding = false,
}: PageSectionProps) {
  return (
    <Box className="supabase-section" mb="lg">
      {(title || actions) && (
        <Group justify="space-between" mb="md">
          <Box>
            {title && (
              <Text fw={600} size="sm">
                {title}
              </Text>
            )}
            {description && (
              <Text size="sm" c="dimmed">
                {description}
              </Text>
            )}
          </Box>
          {actions && <Group gap="sm">{actions}</Group>}
        </Group>
      )}
      <Box p={noPadding ? 0 : undefined}>{children}</Box>
    </Box>
  );
}
