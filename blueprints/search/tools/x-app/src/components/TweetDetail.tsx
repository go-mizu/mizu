import React from 'react'
import { View, Text, StyleSheet, TouchableOpacity } from 'react-native'
import { Image } from 'expo-image'
import { useRouter } from 'expo-router'
import * as Linking from 'expo-linking'
import { useTheme } from '../theme'
import { Badge } from './Badge'
import { RichText } from './RichText'
import { MediaGrid } from './MediaGrid'
import { QuotedTweet } from './QuotedTweet'
import { fmtNum, fullDate } from '../utils'
import type { Tweet } from '../api/types'

interface TweetDetailProps {
  tweet: Tweet
}

export function TweetDetail({ tweet }: TweetDetailProps) {
  const theme = useTheme()
  const router = useRouter()

  return (
    <View style={[styles.container, { borderBottomColor: theme.border }]}>
      {/* Header */}
      <View style={styles.header}>
        <TouchableOpacity onPress={() => router.push(`/${tweet.username}`)} activeOpacity={0.8}>
          <Image source={{ uri: tweet.avatar }} style={styles.avatar} />
        </TouchableOpacity>
        <View style={styles.headerText}>
          <View style={styles.nameRow}>
            <Text style={[styles.name, { color: theme.text }]}>{tweet.name}</Text>
            {tweet.isBlueVerified && <Badge verifiedType={tweet.verifiedType} />}
          </View>
          <Text style={[styles.handle, { color: theme.secondary }]}>@{tweet.username}</Text>
        </View>
        <TouchableOpacity
          onPress={() => Linking.openURL(tweet.permanentURL)}
          style={styles.viewOnX}
        >
          <Text style={[styles.viewOnXText, { color: theme.blue }]}>View on X</Text>
        </TouchableOpacity>
      </View>

      {/* Reply context */}
      {tweet.isReply && tweet.replyToUser && (
        <Text style={[styles.replyContext, { color: theme.secondary }]}>
          Replying to <Text style={{ color: theme.blue }}>@{tweet.replyToUser}</Text>
        </Text>
      )}

      {/* Tweet text */}
      <RichText text={tweet.text} urls={tweet.urls} style={styles.bodyText} />

      {/* Media */}
      <MediaGrid
        photos={tweet.photos}
        videos={tweet.videos}
        videoThumbnails={tweet.videoThumbnails}
        gifs={tweet.gifs}
      />

      {/* Quoted tweet */}
      {tweet.quotedTweet && <QuotedTweet tweet={tweet.quotedTweet} />}

      {/* Timestamp */}
      <Text style={[styles.timestamp, { color: theme.secondary }]}>
        {fullDate(tweet.postedAt)}
      </Text>

      {/* Stats bar */}
      <View style={[styles.stats, { borderColor: theme.border }]}>
        {tweet.retweets > 0 && (
          <StatItem label="Reposts" count={tweet.retweets} color={theme.text} />
        )}
        {tweet.quotes > 0 && (
          <StatItem label="Quotes" count={tweet.quotes} color={theme.text} />
        )}
        {tweet.likes > 0 && (
          <StatItem label="Likes" count={tweet.likes} color={theme.text} />
        )}
        {tweet.bookmarks > 0 && (
          <StatItem label="Bookmarks" count={tweet.bookmarks} color={theme.text} />
        )}
        {tweet.views > 0 && (
          <StatItem label="Views" count={tweet.views} color={theme.text} />
        )}
      </View>
    </View>
  )
}

function StatItem({ label, count, color }: { label: string; count: number; color: string }) {
  return (
    <View style={statStyles.item}>
      <Text style={[statStyles.count, { color }]}>{fmtNum(count)}</Text>
      <Text style={[statStyles.label, { color: '#536471' }]}> {label}</Text>
    </View>
  )
}

const statStyles = StyleSheet.create({
  item: { flexDirection: 'row', marginRight: 16 },
  count: { fontWeight: '700', fontSize: 14 },
  label: { fontSize: 14 },
})

const styles = StyleSheet.create({
  container: {
    paddingHorizontal: 16,
    paddingVertical: 12,
    borderBottomWidth: StyleSheet.hairlineWidth,
  },
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    marginBottom: 12,
  },
  avatar: {
    width: 48,
    height: 48,
    borderRadius: 24,
    marginRight: 12,
  },
  headerText: { flex: 1 },
  nameRow: {
    flexDirection: 'row',
    alignItems: 'center',
  },
  name: { fontWeight: '700', fontSize: 16 },
  handle: { fontSize: 14 },
  viewOnX: { paddingLeft: 8 },
  viewOnXText: { fontSize: 14, fontWeight: '600' },
  replyContext: { fontSize: 14, marginBottom: 8 },
  bodyText: { fontSize: 17, lineHeight: 24 },
  timestamp: {
    fontSize: 14,
    marginTop: 12,
    paddingTop: 12,
  },
  stats: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    paddingVertical: 12,
    marginTop: 4,
    borderTopWidth: StyleSheet.hairlineWidth,
    borderBottomWidth: StyleSheet.hairlineWidth,
  },
})
