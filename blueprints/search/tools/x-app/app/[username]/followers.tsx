import React, { useEffect } from 'react'
import { View, FlatList, Text, StyleSheet, ActivityIndicator } from 'react-native'
import { useLocalSearchParams, Stack } from 'expo-router'
import { useTheme } from '../../src/theme'
import { useFollows } from '../../src/hooks/useFollows'
import { UserCard } from '../../src/components/UserCard'

export default function FollowersScreen() {
  const { username } = useLocalSearchParams<{ username: string }>()
  const theme = useTheme()
  const { users, loading, error, loadMore, refresh, fetchInitial } = useFollows(username, 'followers')

  useEffect(() => {
    if (username) fetchInitial()
  }, [username])

  return (
    <View style={[styles.container, { backgroundColor: theme.bg }]}>
      <Stack.Screen options={{ title: `@${username} - Followers` }} />
      <FlatList
        data={users}
        keyExtractor={(item) => item.id}
        renderItem={({ item }) => <UserCard user={item} />}
        onEndReached={loadMore}
        onEndReachedThreshold={0.5}
        onRefresh={refresh}
        refreshing={false}
        ListFooterComponent={loading ? <ActivityIndicator style={{ padding: 20 }} color={theme.blue} /> : null}
        ListEmptyComponent={
          !loading ? <Text style={[styles.empty, { color: theme.secondary }]}>No followers</Text> : null
        }
      />
    </View>
  )
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  empty: { textAlign: 'center', padding: 40, fontSize: 15 },
})
