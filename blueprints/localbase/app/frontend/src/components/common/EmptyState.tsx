import { Box, Text, Button, ThemeIcon } from '@mantine/core';
import type { ReactNode } from 'react';

interface EmptyStateProps {
  icon: ReactNode;
  title: string;
  description: string;
  action?: {
    label: string;
    onClick: () => void;
  };
}

export function EmptyState({ icon, title, description, action }: EmptyStateProps) {
  return (
    <Box
      className="lb-empty-state"
      style={{
        textAlign: 'center',
        padding: 'var(--lb-space-12) var(--lb-space-6)',
      }}
    >
      <ThemeIcon
        size={64}
        variant="light"
        color="gray"
        radius="xl"
        className="lb-empty-state-icon"
        style={{
          margin: '0 auto',
          marginBottom: 'var(--lb-space-4)',
          backgroundColor: 'var(--lb-bg-secondary)',
        }}
      >
        {icon}
      </ThemeIcon>
      <Text
        className="lb-empty-state-title"
        style={{
          fontSize: 'var(--lb-text-lg)',
          fontWeight: 500,
          color: 'var(--lb-text-primary)',
          marginBottom: 'var(--lb-space-2)',
        }}
      >
        {title}
      </Text>
      <Text
        className="lb-empty-state-description"
        style={{
          fontSize: 'var(--lb-text-md)',
          color: 'var(--lb-text-secondary)',
          marginBottom: 'var(--lb-space-6)',
          maxWidth: 400,
          marginLeft: 'auto',
          marginRight: 'auto',
        }}
      >
        {description}
      </Text>
      {action && (
        <Button onClick={action.onClick} variant="filled">
          {action.label}
        </Button>
      )}
    </Box>
  );
}
