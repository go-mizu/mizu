import React, { useState } from 'react'
import { View, TextInput, StyleSheet, TouchableOpacity, Text } from 'react-native'
import { useRouter } from 'expo-router'
import { useTheme } from '../theme'

interface SearchBarProps {
  initialQuery?: string
  autoFocus?: boolean
  onSubmit?: (query: string) => void
}

export function SearchBar({ initialQuery = '', autoFocus = false, onSubmit }: SearchBarProps) {
  const [query, setQuery] = useState(initialQuery)
  const router = useRouter()
  const theme = useTheme()

  const handleSubmit = () => {
    const q = query.trim()
    if (!q) return
    if (onSubmit) {
      onSubmit(q)
    } else {
      // Navigate to search
      if (q.startsWith('@') && !q.includes(' ')) {
        router.push(`/${q.slice(1)}`)
      } else {
        router.push(`/search?q=${encodeURIComponent(q)}`)
      }
    }
  }

  return (
    <View style={[styles.container, { backgroundColor: theme.searchBg, borderColor: theme.border }]}>
      <TextInput
        style={[styles.input, { color: theme.text }]}
        placeholder="Search X"
        placeholderTextColor={theme.secondary}
        value={query}
        onChangeText={setQuery}
        onSubmitEditing={handleSubmit}
        returnKeyType="search"
        autoFocus={autoFocus}
        autoCapitalize="none"
        autoCorrect={false}
      />
      {query.length > 0 && (
        <TouchableOpacity onPress={() => setQuery('')} style={styles.clear}>
          <View style={[styles.clearBtn, { backgroundColor: theme.blue }]}>
            <Text style={styles.clearText}>×</Text>
          </View>
        </TouchableOpacity>
      )}
    </View>
  )
}

const styles = StyleSheet.create({
  container: {
    flexDirection: 'row',
    alignItems: 'center',
    borderRadius: 20,
    paddingHorizontal: 16,
    height: 40,
    borderWidth: StyleSheet.hairlineWidth,
  },
  input: {
    flex: 1,
    fontSize: 15,
    paddingVertical: 0,
  },
  clear: {
    marginLeft: 8,
  },
  clearBtn: {
    width: 20,
    height: 20,
    borderRadius: 10,
    alignItems: 'center',
    justifyContent: 'center',
  },
  clearText: {
    color: '#fff',
    fontSize: 14,
    fontWeight: '700',
    lineHeight: 18,
  },
})
