import React, { useState, useCallback } from 'react'
import { View, Text, StyleSheet, TouchableOpacity, Alert, ScrollView } from 'react-native'
import { Stack, useFocusEffect } from 'expo-router'
import { useTheme } from '../src/theme'
import { useNetwork } from '../src/hooks/useNetwork'
import { OfflineBanner } from '../src/components/OfflineBanner'
import { clearCache, getCacheStats, CacheStats } from '../src/cache/store'
import { getBookmarkCount, clearBookmarks } from '../src/cache/bookmarks'
import { getPinnedCount, clearPinnedProfiles } from '../src/cache/pins'
import { X_AUTH_TOKEN, X_CT0 } from '../src/env'

export default function SettingsScreen() {
  const theme = useTheme()
  const { isOnline } = useNetwork()
  const [stats, setStats] = useState<CacheStats | null>(null)
  const [bookmarkCount, setBookmarkCount] = useState(0)
  const [pinnedCount, setPinnedCount] = useState(0)

  const loadStats = useCallback(async () => {
    const [s, b, p] = await Promise.all([
      getCacheStats(),
      getBookmarkCount(),
      getPinnedCount(),
    ])
    setStats(s)
    setBookmarkCount(b)
    setPinnedCount(p)
  }, [])

  useFocusEffect(
    useCallback(() => { loadStats() }, [loadStats])
  )

  const handleClearCache = () => {
    Alert.alert('Clear Cache', 'This will clear all cached API responses. Bookmarks and pinned profiles are not affected.', [
      { text: 'Cancel', style: 'cancel' },
      {
        text: 'Clear',
        style: 'destructive',
        onPress: async () => {
          await clearCache()
          await loadStats()
          Alert.alert('Done', 'Cache cleared')
        },
      },
    ])
  }

  const handleClearBookmarks = () => {
    Alert.alert('Clear Bookmarks', `Remove all ${bookmarkCount} bookmarked tweets?`, [
      { text: 'Cancel', style: 'cancel' },
      {
        text: 'Clear',
        style: 'destructive',
        onPress: async () => {
          await clearBookmarks()
          await loadStats()
          Alert.alert('Done', 'Bookmarks cleared')
        },
      },
    ])
  }

  const handleClearPins = () => {
    Alert.alert('Clear Pinned Profiles', `Remove all ${pinnedCount} pinned profiles?`, [
      { text: 'Cancel', style: 'cancel' },
      {
        text: 'Clear',
        style: 'destructive',
        onPress: async () => {
          await clearPinnedProfiles()
          await loadStats()
          Alert.alert('Done', 'Pinned profiles cleared')
        },
      },
    ])
  }

  return (
    <ScrollView style={[styles.container, { backgroundColor: theme.bg }]}>
      <Stack.Screen options={{ title: 'Settings' }} />
      <OfflineBanner />

      {/* Network Status */}
      <Text style={[styles.section, { color: theme.secondary }]}>NETWORK</Text>
      <View style={[styles.card, { backgroundColor: theme.searchBg, borderColor: theme.border }]}>
        <View style={styles.statusRow}>
          <View style={[styles.statusDot, { backgroundColor: isOnline ? '#00ba7c' : '#f59e0b' }]} />
          <Text style={[styles.statusText, { color: theme.text }]}>
            {isOnline ? 'Online' : 'Offline'}
          </Text>
        </View>
      </View>

      {/* Credentials */}
      <Text style={[styles.section, { color: theme.secondary }]}>CREDENTIALS</Text>
      <View style={[styles.card, { backgroundColor: theme.searchBg, borderColor: theme.border }]}>
        <Text style={[styles.cardLabel, { color: theme.secondary }]}>auth_token</Text>
        <Text style={[styles.cardValue, { color: theme.text }]} numberOfLines={1}>
          {X_AUTH_TOKEN ? X_AUTH_TOKEN.slice(0, 8) + '...' + X_AUTH_TOKEN.slice(-8) : 'Not set'}
        </Text>
        <Text style={[styles.cardLabel, { color: theme.secondary, marginTop: 12 }]}>ct0</Text>
        <Text style={[styles.cardValue, { color: theme.text }]} numberOfLines={1}>
          {X_CT0 ? X_CT0.slice(0, 8) + '...' + X_CT0.slice(-8) : 'Not set'}
        </Text>
      </View>
      <Text style={[styles.hint, { color: theme.secondary }]}>
        Credentials are configured in src/env.ts (prebuilt, like x-viewer's CF environment).
      </Text>

      {/* Cache Stats */}
      <Text style={[styles.section, { color: theme.secondary, marginTop: 32 }]}>CACHE</Text>
      <View style={[styles.card, { backgroundColor: theme.searchBg, borderColor: theme.border }]}>
        <View style={styles.statRow}>
          <Text style={[styles.statLabel, { color: theme.secondary }]}>Cached entries</Text>
          <Text style={[styles.statValue, { color: theme.text }]}>{stats?.entryCount ?? 0}</Text>
        </View>
        <View style={[styles.statRow, { marginTop: 8 }]}>
          <Text style={[styles.statLabel, { color: theme.secondary }]}>Cache size</Text>
          <Text style={[styles.statValue, { color: theme.text }]}>{stats ? `~${stats.approximateSizeKB} KB` : '—'}</Text>
        </View>
        <View style={[styles.statRow, { marginTop: 8 }]}>
          <Text style={[styles.statLabel, { color: theme.secondary }]}>Bookmarks</Text>
          <Text style={[styles.statValue, { color: theme.text }]}>{bookmarkCount}</Text>
        </View>
        <View style={[styles.statRow, { marginTop: 8 }]}>
          <Text style={[styles.statLabel, { color: theme.secondary }]}>Pinned profiles</Text>
          <Text style={[styles.statValue, { color: theme.text }]}>{pinnedCount}</Text>
        </View>
      </View>

      {/* Clear buttons */}
      <View style={styles.buttonGroup}>
        <TouchableOpacity
          style={[styles.button, { backgroundColor: '#ff3b30' }]}
          onPress={handleClearCache}
        >
          <Text style={styles.buttonText}>Clear Cache</Text>
        </TouchableOpacity>

        {bookmarkCount > 0 && (
          <TouchableOpacity
            style={[styles.button, { backgroundColor: '#ff9500' }]}
            onPress={handleClearBookmarks}
          >
            <Text style={styles.buttonText}>Clear Bookmarks ({bookmarkCount})</Text>
          </TouchableOpacity>
        )}

        {pinnedCount > 0 && (
          <TouchableOpacity
            style={[styles.button, { backgroundColor: '#ff9500' }]}
            onPress={handleClearPins}
          >
            <Text style={styles.buttonText}>Clear Pinned Profiles ({pinnedCount})</Text>
          </TouchableOpacity>
        )}
      </View>
    </ScrollView>
  )
}

const styles = StyleSheet.create({
  container: { flex: 1, padding: 16 },
  section: {
    fontSize: 13,
    fontWeight: '600',
    letterSpacing: 0.5,
    marginTop: 16,
    marginBottom: 12,
  },
  card: {
    borderWidth: 1,
    borderRadius: 12,
    padding: 16,
  },
  cardLabel: { fontSize: 12, fontWeight: '600', marginBottom: 4 },
  cardValue: { fontSize: 14, fontFamily: 'monospace' },
  hint: {
    fontSize: 13,
    lineHeight: 18,
    marginTop: 12,
  },
  statusRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 8,
  },
  statusDot: {
    width: 10,
    height: 10,
    borderRadius: 5,
  },
  statusText: {
    fontSize: 15,
    fontWeight: '600',
  },
  statRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
  },
  statLabel: { fontSize: 14 },
  statValue: { fontSize: 14, fontWeight: '600' },
  buttonGroup: {
    marginTop: 16,
    gap: 8,
  },
  button: {
    padding: 14,
    borderRadius: 24,
    alignItems: 'center',
  },
  buttonText: {
    color: '#fff',
    fontWeight: '700',
    fontSize: 16,
  },
})
