import React from 'react'
import { View, FlatList, Text, StyleSheet, ActivityIndicator } from 'react-native'
import { useLocalSearchParams, Stack } from 'expo-router'
import { useTheme } from '../../../src/theme'
import { useTweet } from '../../../src/hooks/useTweet'
import { TweetDetail } from '../../../src/components/TweetDetail'
import { TweetCard } from '../../../src/components/TweetCard'
import { OfflineBanner } from '../../../src/components/OfflineBanner'

export default function TweetDetailScreen() {
  const { id, username } = useLocalSearchParams<{ id: string; username: string }>()
  const theme = useTheme()
  const { tweet, replies, loading, error, loadMore } = useTweet(id)

  if (loading && !tweet) {
    return (
      <View style={[styles.center, { backgroundColor: theme.bg }]}>
        <Stack.Screen options={{ title: 'Post' }} />
        <ActivityIndicator size="large" color={theme.blue} />
      </View>
    )
  }

  if (error) {
    return (
      <View style={[styles.center, { backgroundColor: theme.bg }]}>
        <Stack.Screen options={{ title: 'Post' }} />
        <Text style={[styles.error, { color: theme.secondary }]}>{error}</Text>
      </View>
    )
  }

  return (
    <View style={[styles.container, { backgroundColor: theme.bg }]}>
      <Stack.Screen options={{ title: 'Post' }} />
      <OfflineBanner />
      <FlatList
        data={replies}
        keyExtractor={(item) => item.id}
        ListHeaderComponent={tweet ? <TweetDetail tweet={tweet} /> : null}
        renderItem={({ item }) => <TweetCard tweet={item} />}
        onEndReached={loadMore}
        onEndReachedThreshold={0.5}
        ListFooterComponent={loading ? <ActivityIndicator style={{ padding: 20 }} color={theme.blue} /> : null}
      />
    </View>
  )
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  center: { flex: 1, alignItems: 'center', justifyContent: 'center' },
  error: { fontSize: 16 },
})
