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
    <Box className="supabase-empty-state">
      <ThemeIcon
        size={64}
        variant="light"
        color="gray"
        radius="xl"
        className="supabase-empty-state-icon"
        style={{ margin: '0 auto' }}
      >
        {icon}
      </ThemeIcon>
      <Text className="supabase-empty-state-title">{title}</Text>
      <Text className="supabase-empty-state-description">{description}</Text>
      {action && (
        <Button onClick={action.onClick} variant="filled">
          {action.label}
        </Button>
      )}
    </Box>
  );
}
