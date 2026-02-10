import React, { useEffect, useState } from 'react'
import { View, FlatList, Text, StyleSheet, ActivityIndicator } from 'react-native'
import { useLocalSearchParams, useRouter, Stack } from 'expo-router'
import { useTheme } from '../src/theme'
import { useSearch } from '../src/hooks/useSearch'
import { SearchBar } from '../src/components/SearchBar'
import { TweetCard } from '../src/components/TweetCard'
import { UserCard } from '../src/components/UserCard'
import { TabBar } from '../src/components/TabBar'
import { addSearchHistory } from '../src/cache/store'
import { SearchTop, SearchLatest, SearchPeople, SearchPhotos } from '../src/api/config'

const searchTabs = [
  { key: SearchTop, label: 'Top' },
  { key: SearchLatest, label: 'Latest' },
  { key: SearchPeople, label: 'People' },
  { key: SearchPhotos, label: 'Media' },
]

export default function SearchScreen() {
  const params = useLocalSearchParams<{ q: string; mode: string }>()
  const theme = useTheme()
  const router = useRouter()
  const query = params.q || ''
  const [mode, setMode] = useState(params.mode || SearchTop)

  const { tweets, users, loading, error, loadMore, refresh, fetchInitial } = useSearch(query, mode)

  useEffect(() => {
    if (query) {
      addSearchHistory(query)
      fetchInitial()
    }
  }, [query, mode])

  const isPeople = mode === SearchPeople

  const handleSearch = (q: string) => {
    if (q.startsWith('@') && !q.includes(' ')) {
      router.push(`/${q.slice(1)}`)
    } else {
      router.setParams({ q })
    }
  }

  return (
    <View style={[styles.container, { backgroundColor: theme.bg }]}>
      <Stack.Screen options={{ title: 'Search', headerShown: false }} />

      <View style={[styles.searchHeader, { backgroundColor: theme.barBg, borderBottomColor: theme.border }]}>
        <View style={styles.searchBarWrapper}>
          <SearchBar initialQuery={query} onSubmit={handleSearch} />
        </View>
      </View>

      <TabBar tabs={searchTabs} active={mode} onSelect={setMode} />

      {isPeople ? (
        <FlatList
          data={users}
          keyExtractor={(item) => item.id}
          renderItem={({ item }) => <UserCard user={item} />}
          onEndReached={loadMore}
          onEndReachedThreshold={0.5}
          ListEmptyComponent={
            !loading ? <Text style={[styles.empty, { color: theme.secondary }]}>No results</Text> : null
          }
          ListFooterComponent={loading ? <ActivityIndicator style={{ padding: 20 }} color={theme.blue} /> : null}
        />
      ) : (
        <FlatList
          data={tweets}
          keyExtractor={(item) => item.id}
          renderItem={({ item }) => <TweetCard tweet={item} />}
          onEndReached={loadMore}
          onEndReachedThreshold={0.5}
          ListEmptyComponent={
            !loading ? <Text style={[styles.empty, { color: theme.secondary }]}>No results</Text> : null
          }
          ListFooterComponent={loading ? <ActivityIndicator style={{ padding: 20 }} color={theme.blue} /> : null}
        />
      )}
    </View>
  )
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  searchHeader: {
    paddingTop: 54,
    paddingBottom: 8,
    paddingHorizontal: 16,
    borderBottomWidth: StyleSheet.hairlineWidth,
  },
  searchBarWrapper: { flexDirection: 'row' },
  empty: { textAlign: 'center', padding: 40, fontSize: 15 },
})
