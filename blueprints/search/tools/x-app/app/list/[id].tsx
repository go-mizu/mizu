import React, { useEffect, useState, useCallback, useRef } from 'react'
import { View, FlatList, Text, StyleSheet, ActivityIndicator } from 'react-native'
import { useLocalSearchParams, Stack } from 'expo-router'
import { useTheme } from '../../src/theme'
import { fetchList } from '../../src/api/client'
import { TweetCard } from '../../src/components/TweetCard'
import { UserCard } from '../../src/components/UserCard'
import { TabBar } from '../../src/components/TabBar'
import type { XList, Tweet, Profile } from '../../src/api/types'

const listTabs = [
  { key: 'tweets', label: 'Tweets' },
  { key: 'members', label: 'Members' },
]

export default function ListScreen() {
  const { id } = useLocalSearchParams<{ id: string }>()
  const theme = useTheme()
  const [list, setList] = useState<XList | null>(null)
  const [activeTab, setActiveTab] = useState('tweets')
  const [tweets, setTweets] = useState<Tweet[]>([])
  const [members, setMembers] = useState<Profile[]>([])
  const [loading, setLoading] = useState(true)
  const cursorRef = useRef('')

  const loadData = useCallback(async (tab: string, cursor?: string) => {
    setLoading(true)
    try {
      const result = await fetchList(id, tab, cursor)
      if (result.list) setList(result.list)
      if (tab === 'tweets') {
        if (cursor) {
          setTweets(prev => [...prev, ...(result.tweets || [])])
        } else {
          setTweets(result.tweets || [])
        }
      } else {
        if (cursor) {
          setMembers(prev => [...prev, ...(result.users || [])])
        } else {
          setMembers(result.users || [])
        }
      }
      cursorRef.current = result.cursor
    } catch {}
    setLoading(false)
  }, [id])

  useEffect(() => {
    cursorRef.current = ''
    loadData(activeTab)
  }, [activeTab, loadData])

  const loadMore = () => {
    if (cursorRef.current) {
      loadData(activeTab, cursorRef.current)
    }
  }

  const renderHeader = () => (
    <View>
      {list && (
        <View style={[styles.listHeader, { borderBottomColor: theme.border }]}>
          <Text style={[styles.listName, { color: theme.text }]}>{list.name}</Text>
          {list.description ? (
            <Text style={[styles.listDesc, { color: theme.secondary }]}>{list.description}</Text>
          ) : null}
          <Text style={[styles.listMeta, { color: theme.secondary }]}>
            {list.memberCount} members · by @{list.ownerName}
          </Text>
        </View>
      )}
      <TabBar tabs={listTabs} active={activeTab} onSelect={setActiveTab} />
    </View>
  )

  return (
    <View style={[styles.container, { backgroundColor: theme.bg }]}>
      <Stack.Screen options={{ title: list?.name || 'List' }} />
      {activeTab === 'tweets' ? (
        <FlatList
          data={tweets}
          keyExtractor={(item) => item.id}
          ListHeaderComponent={renderHeader}
          renderItem={({ item }) => <TweetCard tweet={item} />}
          onEndReached={loadMore}
          onEndReachedThreshold={0.5}
          ListFooterComponent={loading ? <ActivityIndicator style={{ padding: 20 }} color={theme.blue} /> : null}
        />
      ) : (
        <FlatList
          data={members}
          keyExtractor={(item) => item.id}
          ListHeaderComponent={renderHeader}
          renderItem={({ item }) => <UserCard user={item} />}
          onEndReached={loadMore}
          onEndReachedThreshold={0.5}
          ListFooterComponent={loading ? <ActivityIndicator style={{ padding: 20 }} color={theme.blue} /> : null}
        />
      )}
    </View>
  )
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  listHeader: {
    padding: 16,
    borderBottomWidth: StyleSheet.hairlineWidth,
  },
  listName: { fontSize: 20, fontWeight: '800' },
  listDesc: { fontSize: 15, marginTop: 4 },
  listMeta: { fontSize: 14, marginTop: 8 },
})
