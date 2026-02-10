import React, { useState, useCallback } from 'react'
import { View, FlatList, Text, StyleSheet } from 'react-native'
import { Stack, useFocusEffect } from 'expo-router'
import { useTheme } from '../src/theme'
import { TweetCard } from '../src/components/TweetCard'
import { OfflineBanner } from '../src/components/OfflineBanner'
import { getBookmarks } from '../src/cache/bookmarks'
import type { Tweet } from '../src/api/types'

export default function BookmarksScreen() {
  const theme = useTheme()
  const [tweets, setTweets] = useState<Tweet[]>([])
  const [loading, setLoading] = useState(true)

  useFocusEffect(
    useCallback(() => {
      setLoading(true)
      getBookmarks().then(t => {
        setTweets(t)
        setLoading(false)
      })
    }, [])
  )

  return (
    <View style={[styles.container, { backgroundColor: theme.bg }]}>
      <Stack.Screen options={{ title: 'Bookmarks' }} />
      <OfflineBanner />
      <FlatList
        data={tweets}
        keyExtractor={(item) => item.id}
        renderItem={({ item }) => <TweetCard tweet={item} />}
        ListEmptyComponent={
          !loading ? (
            <View style={styles.emptyContainer}>
              <Text style={[styles.emptyIcon]}>🔖</Text>
              <Text style={[styles.emptyTitle, { color: theme.text }]}>No bookmarks yet</Text>
              <Text style={[styles.emptyText, { color: theme.secondary }]}>
                Tap the bookmark icon on any tweet to save it for offline reading.
              </Text>
            </View>
          ) : null
        }
      />
    </View>
  )
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  emptyContainer: {
    alignItems: 'center',
    paddingTop: 80,
    paddingHorizontal: 32,
  },
  emptyIcon: { fontSize: 48, marginBottom: 16 },
  emptyTitle: { fontSize: 20, fontWeight: '700', marginBottom: 8 },
  emptyText: { fontSize: 15, textAlign: 'center', lineHeight: 22 },
})
