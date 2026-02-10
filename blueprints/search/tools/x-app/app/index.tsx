import React, { useState, useEffect } from 'react'
import { View, Text, StyleSheet, TouchableOpacity, ScrollView } from 'react-native'
import { useRouter, Stack } from 'expo-router'
import { useTheme } from '../src/theme'
import { SearchBar } from '../src/components/SearchBar'
import { getSearchHistory } from '../src/cache/store'

const quickLinks = [
  { label: '@karpathy', type: 'user', value: 'karpathy' },
  { label: '@openai', type: 'user', value: 'openai' },
  { label: '#ai', type: 'search', value: '#ai' },
  { label: '#golang', type: 'search', value: '#golang' },
  { label: '@mitchellh', type: 'user', value: 'mitchellh' },
  { label: '#typescript', type: 'search', value: '#typescript' },
]

export default function HomeScreen() {
  const theme = useTheme()
  const router = useRouter()
  const [history, setHistory] = useState<string[]>([])

  useEffect(() => {
    getSearchHistory().then(setHistory)
  }, [])

  return (
    <ScrollView style={[styles.scroll, { backgroundColor: theme.bg }]} contentContainerStyle={styles.content}>
      <Stack.Screen options={{ headerShown: false }} />

      <View style={styles.hero}>
        <Text style={[styles.logo, { color: theme.text }]}>𝕏</Text>
        <Text style={[styles.subtitle, { color: theme.secondary }]}>X/Twitter Viewer</Text>
      </View>

      <View style={styles.searchContainer}>
        <SearchBar autoFocus={false} />
      </View>

      <Text style={[styles.sectionTitle, { color: theme.secondary }]}>Quick Links</Text>
      <View style={styles.pills}>
        {quickLinks.map((link) => (
          <TouchableOpacity
            key={link.label}
            style={[styles.pill, { backgroundColor: theme.searchBg, borderColor: theme.border }]}
            onPress={() => {
              if (link.type === 'user') router.push(`/${link.value}`)
              else router.push(`/search?q=${encodeURIComponent(link.value)}`)
            }}
          >
            <Text style={[styles.pillText, { color: theme.blue }]}>{link.label}</Text>
          </TouchableOpacity>
        ))}
      </View>

      {history.length > 0 && (
        <>
          <Text style={[styles.sectionTitle, { color: theme.secondary }]}>Recent Searches</Text>
          {history.slice(0, 8).map((q) => (
            <TouchableOpacity
              key={q}
              style={styles.historyItem}
              onPress={() => router.push(`/search?q=${encodeURIComponent(q)}`)}
            >
              <Text style={[styles.historyText, { color: theme.text }]}>{q}</Text>
            </TouchableOpacity>
          ))}
        </>
      )}
    </ScrollView>
  )
}

const styles = StyleSheet.create({
  scroll: { flex: 1 },
  content: { paddingBottom: 40 },
  hero: {
    alignItems: 'center',
    paddingTop: 80,
    paddingBottom: 24,
  },
  logo: { fontSize: 64, fontWeight: '800' },
  subtitle: { fontSize: 16, marginTop: 4 },
  searchContainer: {
    paddingHorizontal: 24,
    marginBottom: 24,
  },
  sectionTitle: {
    fontSize: 13,
    fontWeight: '600',
    textTransform: 'uppercase',
    letterSpacing: 0.5,
    paddingHorizontal: 24,
    marginTop: 16,
    marginBottom: 8,
  },
  pills: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    paddingHorizontal: 24,
    gap: 8,
  },
  pill: {
    paddingHorizontal: 14,
    paddingVertical: 8,
    borderRadius: 20,
    borderWidth: 1,
  },
  pillText: { fontSize: 14, fontWeight: '500' },
  historyItem: {
    paddingHorizontal: 24,
    paddingVertical: 10,
  },
  historyText: { fontSize: 15 },
})
