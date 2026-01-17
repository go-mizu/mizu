import { TextInput } from '@mantine/core';
import { IconSearch } from '@tabler/icons-react';
import { useDebouncedCallback } from '@mantine/hooks';

interface SearchInputProps {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  debounceMs?: number;
}

export function SearchInput({
  value,
  onChange,
  placeholder = 'Search...',
  debounceMs = 300,
}: SearchInputProps) {
  const debouncedChange = useDebouncedCallback((val: string) => {
    onChange(val);
  }, debounceMs);

  return (
    <TextInput
      placeholder={placeholder}
      leftSection={<IconSearch size={16} />}
      defaultValue={value}
      onChange={(e) => debouncedChange(e.target.value)}
      style={{ width: 280 }}
    />
  );
}
