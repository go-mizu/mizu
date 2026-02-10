import React, { useState, useCallback } from 'react'
import { View, Text, StyleSheet, TouchableOpacity, ScrollView } from 'react-native'
import { useRouter, Stack, useFocusEffect } from 'expo-router'
import { Image } from 'expo-image'
import { useTheme } from '../src/theme'
import { SearchBar } from '../src/components/SearchBar'
import { OfflineBanner } from '../src/components/OfflineBanner'
import { getSearchHistory } from '../src/cache/store'
import { getPinnedProfiles } from '../src/cache/pins'
import { getBookmarkCount } from '../src/cache/bookmarks'
import type { Profile } from '../src/api/types'

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
  const [pinnedProfiles, setPinnedProfiles] = useState<Profile[]>([])
  const [bookmarkCount, setBookmarkCount] = useState(0)

  useFocusEffect(
    useCallback(() => {
      getSearchHistory().then(setHistory)
      getPinnedProfiles().then(setPinnedProfiles)
      getBookmarkCount().then(setBookmarkCount)
    }, [])
  )

  return (
    <ScrollView style={[styles.scroll, { backgroundColor: theme.bg }]} contentContainerStyle={styles.content}>
      <Stack.Screen options={{ headerShown: false }} />
      <OfflineBanner />

      <View style={styles.hero}>
        <Text style={[styles.logo, { color: theme.text }]}>𝕏</Text>
        <Text style={[styles.subtitle, { color: theme.secondary }]}>X/Twitter Viewer</Text>
      </View>

      <View style={styles.searchContainer}>
        <SearchBar autoFocus={false} />
      </View>

      {/* Nav row */}
      <View style={styles.navRow}>
        <TouchableOpacity
          style={[styles.navButton, { backgroundColor: theme.searchBg, borderColor: theme.border }]}
          onPress={() => router.push('/bookmarks')}
        >
          <Text style={styles.navIcon}>🔖</Text>
          <Text style={[styles.navLabel, { color: theme.text }]}>Bookmarks</Text>
          {bookmarkCount > 0 && (
            <Text style={[styles.navCount, { color: theme.secondary }]}>{bookmarkCount}</Text>
          )}
        </TouchableOpacity>

        <TouchableOpacity
          style={[styles.navButton, { backgroundColor: theme.searchBg, borderColor: theme.border }]}
          onPress={() => router.push('/settings')}
        >
          <Text style={styles.navIcon}>⚙️</Text>
          <Text style={[styles.navLabel, { color: theme.text }]}>Settings</Text>
        </TouchableOpacity>
      </View>

      {/* Pinned Profiles */}
      {pinnedProfiles.length > 0 && (
        <>
          <Text style={[styles.sectionTitle, { color: theme.secondary }]}>Pinned Profiles</Text>
          <ScrollView horizontal showsHorizontalScrollIndicator={false} contentContainerStyle={styles.pinnedRow}>
            {pinnedProfiles.map((p) => (
              <TouchableOpacity
                key={p.username}
                style={[styles.pinnedCard, { backgroundColor: theme.searchBg, borderColor: theme.border }]}
                onPress={() => router.push(`/${p.username}`)}
              >
                <Image source={{ uri: p.avatar }} style={styles.pinnedAvatar} />
                <Text style={[styles.pinnedName, { color: theme.text }]} numberOfLines={1}>{p.name}</Text>
                <Text style={[styles.pinnedHandle, { color: theme.secondary }]} numberOfLines={1}>@{p.username}</Text>
              </TouchableOpacity>
            ))}
          </ScrollView>
        </>
      )}

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
    marginBottom: 16,
  },
  navRow: {
    flexDirection: 'row',
    paddingHorizontal: 24,
    gap: 12,
    marginBottom: 8,
  },
  navButton: {
    flex: 1,
    flexDirection: 'row',
    alignItems: 'center',
    paddingHorizontal: 14,
    paddingVertical: 12,
    borderRadius: 12,
    borderWidth: 1,
    gap: 8,
  },
  navIcon: { fontSize: 18 },
  navLabel: { fontSize: 15, fontWeight: '600' },
  navCount: { fontSize: 13, marginLeft: 'auto' },
  sectionTitle: {
    fontSize: 13,
    fontWeight: '600',
    textTransform: 'uppercase',
    letterSpacing: 0.5,
    paddingHorizontal: 24,
    marginTop: 16,
    marginBottom: 8,
  },
  pinnedRow: {
    paddingHorizontal: 24,
    gap: 12,
  },
  pinnedCard: {
    alignItems: 'center',
    padding: 12,
    borderRadius: 12,
    borderWidth: 1,
    width: 100,
  },
  pinnedAvatar: {
    width: 48,
    height: 48,
    borderRadius: 24,
    marginBottom: 8,
  },
  pinnedName: {
    fontSize: 13,
    fontWeight: '600',
    textAlign: 'center',
  },
  pinnedHandle: {
    fontSize: 12,
    textAlign: 'center',
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
