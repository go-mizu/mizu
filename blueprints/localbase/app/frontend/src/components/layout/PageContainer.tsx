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
        <Box
          mb="lg"
          px={noPadding ? 'lg' : 0}
          pt={noPadding ? 'lg' : 0}
          className="lb-page-header"
          style={{ flexDirection: 'column', alignItems: 'stretch' }}
        >
          <Group justify="space-between" align="flex-start">
            <Box>
              {loading ? (
                <>
                  <Skeleton height={32} width={200} mb="xs" />
                  {description && <Skeleton height={20} width={300} />}
                </>
              ) : (
                <>
                  <Title
                    order={2}
                    className="lb-page-title"
                    style={{
                      fontSize: 'var(--lb-text-2xl)',
                      fontWeight: 600,
                      color: 'var(--lb-text-primary)',
                      lineHeight: 1.25,
                    }}
                  >
                    {title}
                  </Title>
                  {description && (
                    <Text
                      className="lb-page-description"
                      mt={4}
                      style={{
                        color: 'var(--lb-text-secondary)',
                        fontSize: 'var(--lb-text-md)',
                      }}
                    >
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
    <Box
      className="lb-section"
      mb="lg"
      style={{
        backgroundColor: 'var(--lb-bg-primary)',
        border: '1px solid var(--lb-border-default)',
        borderRadius: 'var(--lb-radius-lg)',
        padding: 'var(--lb-space-5)',
      }}
    >
      {(title || actions) && (
        <Group justify="space-between" mb="md">
          <Box>
            {title && (
              <Text
                fw={600}
                style={{
                  fontSize: 'var(--lb-text-md)',
                  color: 'var(--lb-text-primary)',
                }}
              >
                {title}
              </Text>
            )}
            {description && (
              <Text
                size="sm"
                style={{
                  color: 'var(--lb-text-secondary)',
                  fontSize: 'var(--lb-text-sm)',
                  marginTop: 'var(--lb-space-1)',
                }}
              >
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
