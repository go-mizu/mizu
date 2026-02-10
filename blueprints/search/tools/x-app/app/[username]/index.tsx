import React, { useEffect, useState } from 'react'
import { View, FlatList, Text, StyleSheet, ActivityIndicator, TouchableOpacity, Dimensions } from 'react-native'
import { useLocalSearchParams, Stack } from 'expo-router'
import { Image } from 'expo-image'
import { useTheme } from '../../src/theme'
import { useProfile } from '../../src/hooks/useProfile'
import { useTimeline } from '../../src/hooks/useTimeline'
import { ProfileHeader } from '../../src/components/ProfileHeader'
import { TweetCard } from '../../src/components/TweetCard'
import { TabBar } from '../../src/components/TabBar'
import { OfflineBanner } from '../../src/components/OfflineBanner'
import type { Tweet } from '../../src/api/types'

const tabs = [
  { key: 'tweets', label: 'Posts' },
  { key: 'replies', label: 'Replies' },
  { key: 'media', label: 'Media' },
]

const { width: SCREEN_WIDTH } = Dimensions.get('window')
const MEDIA_ITEM_SIZE = (SCREEN_WIDTH - 4) / 3

export default function ProfileScreen() {
  const { username } = useLocalSearchParams<{ username: string }>()
  const theme = useTheme()
  const [activeTab, setActiveTab] = useState('tweets')

  const { profile, loading: profileLoading, error: profileError } = useProfile(username)
  const { tweets, loading, refreshing, error, loadMore, refresh, fetchInitial } = useTimeline(
    username,
    activeTab
  )

  useEffect(() => {
    if (username) {
      fetchInitial()
    }
  }, [username, activeTab])

  const isMedia = activeTab === 'media'

  const renderHeader = () => (
    <View>
      {profile && <ProfileHeader profile={profile} />}
      <TabBar tabs={tabs} active={activeTab} onSelect={setActiveTab} />
    </View>
  )

  const renderMediaItem = ({ item }: { item: Tweet }) => {
    const mediaUrl = item.photos[0] || item.videoThumbnails[0] || ''
    if (!mediaUrl) return null
    return (
      <TouchableOpacity style={{ width: MEDIA_ITEM_SIZE, height: MEDIA_ITEM_SIZE, padding: 1 }}>
        <Image source={{ uri: mediaUrl }} style={{ flex: 1 }} contentFit="cover" />
      </TouchableOpacity>
    )
  }

  if (profileLoading && !profile) {
    return (
      <View style={[styles.center, { backgroundColor: theme.bg }]}>
        <Stack.Screen options={{ title: `@${username}` }} />
        <ActivityIndicator size="large" color={theme.blue} />
      </View>
    )
  }

  if (profileError) {
    return (
      <View style={[styles.center, { backgroundColor: theme.bg }]}>
        <Stack.Screen options={{ title: `@${username}` }} />
        <Text style={[styles.error, { color: theme.secondary }]}>{profileError}</Text>
      </View>
    )
  }

  const mediaTweets = isMedia ? tweets.filter(t => t.photos.length > 0 || t.videoThumbnails.length > 0) : []

  return (
    <View style={[styles.container, { backgroundColor: theme.bg }]}>
      <Stack.Screen options={{ title: `@${username}` }} />
      <OfflineBanner />
      {isMedia ? (
        <FlatList
          data={mediaTweets}
          keyExtractor={(item) => item.id}
          numColumns={3}
          ListHeaderComponent={renderHeader}
          renderItem={renderMediaItem}
          onEndReached={loadMore}
          onEndReachedThreshold={0.5}
          onRefresh={refresh}
          refreshing={refreshing}
        />
      ) : (
        <FlatList
          data={tweets}
          keyExtractor={(item) => item.id}
          ListHeaderComponent={renderHeader}
          renderItem={({ item }) => <TweetCard tweet={item} />}
          onEndReached={loadMore}
          onEndReachedThreshold={0.5}
          onRefresh={refresh}
          refreshing={refreshing}
          ListFooterComponent={loading ? <ActivityIndicator style={{ padding: 20 }} color={theme.blue} /> : null}
        />
      )}
    </View>
  )
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  center: { flex: 1, alignItems: 'center', justifyContent: 'center' },
  error: { fontSize: 16 },
})
