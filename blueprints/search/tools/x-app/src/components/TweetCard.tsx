import React, { useState, useEffect } from 'react'
import { View, Text, StyleSheet, TouchableOpacity } from 'react-native'
import { Image } from 'expo-image'
import { useRouter } from 'expo-router'
import { useTheme } from '../theme'
import { Badge } from './Badge'
import { RichText } from './RichText'
import { MediaGrid } from './MediaGrid'
import { QuotedTweet } from './QuotedTweet'
import { fmtNum, relTime } from '../utils'
import { addBookmark, removeBookmark, isBookmarked } from '../cache/bookmarks'
import type { Tweet } from '../api/types'

interface TweetCardProps {
  tweet: Tweet
  showBookmarkAction?: boolean
}

export function TweetCard({ tweet: rawTweet, showBookmarkAction = true }: TweetCardProps) {
  const theme = useTheme()
  const router = useRouter()
  const [bookmarked, setBookmarked] = useState(false)

  // If retweet, show inner tweet with retweet label
  const isRT = rawTweet.isRetweet && rawTweet.retweetedTweet
  const tweet = isRT ? rawTweet.retweetedTweet! : rawTweet

  useEffect(() => {
    isBookmarked(tweet.id).then(setBookmarked)
  }, [tweet.id])

  const toggleBookmark = async () => {
    if (bookmarked) {
      await removeBookmark(tweet.id)
      setBookmarked(false)
    } else {
      await addBookmark(tweet)
      setBookmarked(true)
    }
  }

  return (
    <TouchableOpacity
      style={[styles.container, { borderBottomColor: theme.border }]}
      onPress={() => router.push(`/${tweet.username}/status/${tweet.id}`)}
      activeOpacity={0.7}
    >
      {/* Retweet label */}
      {isRT && (
        <View style={styles.rtLabel}>
          <Text style={[styles.rtIcon, { color: theme.secondary }]}>↻</Text>
          <Text style={[styles.rtText, { color: theme.secondary }]}>
            {rawTweet.name} reposted
          </Text>
        </View>
      )}

      {/* Pin label */}
      {rawTweet.isPin && !isRT && (
        <View style={styles.rtLabel}>
          <Text style={[styles.rtText, { color: theme.secondary }]}>Pinned</Text>
        </View>
      )}

      <View style={styles.body}>
        {/* Avatar */}
        <TouchableOpacity
          onPress={() => router.push(`/${tweet.username}`)}
          activeOpacity={0.8}
        >
          <Image source={{ uri: tweet.avatar }} style={styles.avatar} />
        </TouchableOpacity>

        <View style={styles.content}>
          {/* Header */}
          <View style={styles.header}>
            <Text style={[styles.name, { color: theme.text }]} numberOfLines={1}>
              {tweet.name}
            </Text>
            {tweet.isBlueVerified && <Badge verifiedType={tweet.verifiedType} />}
            <Text style={[styles.handle, { color: theme.secondary }]} numberOfLines={1}>
              @{tweet.username}
            </Text>
            <Text style={[styles.dot, { color: theme.secondary }]}> · </Text>
            <Text style={[styles.time, { color: theme.secondary }]}>{relTime(tweet.postedAt)}</Text>
          </View>

          {/* Reply context */}
          {tweet.isReply && tweet.replyToUser && (
            <Text style={[styles.replyContext, { color: theme.secondary }]}>
              Replying to <Text style={{ color: theme.blue }}>@{tweet.replyToUser}</Text>
            </Text>
          )}

          {/* Tweet text */}
          <RichText text={tweet.text} urls={tweet.urls} />

          {/* Media */}
          <MediaGrid
            photos={tweet.photos}
            videos={tweet.videos}
            videoThumbnails={tweet.videoThumbnails}
            gifs={tweet.gifs}
          />

          {/* Quoted tweet */}
          {tweet.quotedTweet && <QuotedTweet tweet={tweet.quotedTweet} />}

          {/* Action bar */}
          <View style={styles.actions}>
            <ActionButton icon="💬" count={tweet.replies} color={theme.secondary} />
            <ActionButton icon="↻" count={tweet.retweets} color={theme.retweet} />
            <ActionButton icon="♡" count={tweet.likes} color={theme.like} />
            <ActionButton icon="👁" count={tweet.views} color={theme.secondary} />
            {showBookmarkAction && (
              <TouchableOpacity onPress={toggleBookmark} hitSlop={{ top: 8, bottom: 8, left: 8, right: 8 }}>
                <Text style={[actionStyles.icon, { color: bookmarked ? theme.blue : theme.secondary }]}>
                  {bookmarked ? '🔖' : '🏷'}
                </Text>
              </TouchableOpacity>
            )}
          </View>
        </View>
      </View>
    </TouchableOpacity>
  )
}

function ActionButton({ icon, count, color }: { icon: string; count: number; color: string }) {
  return (
    <View style={actionStyles.container}>
      <Text style={[actionStyles.icon, { color }]}>{icon}</Text>
      {count > 0 && <Text style={[actionStyles.count, { color }]}>{fmtNum(count)}</Text>}
    </View>
  )
}

const actionStyles = StyleSheet.create({
  container: {
    flexDirection: 'row',
    alignItems: 'center',
    flex: 1,
  },
  icon: { fontSize: 14 },
  count: { fontSize: 13, marginLeft: 4 },
})

const styles = StyleSheet.create({
  container: {
    paddingHorizontal: 16,
    paddingVertical: 12,
    borderBottomWidth: StyleSheet.hairlineWidth,
  },
  rtLabel: {
    flexDirection: 'row',
    alignItems: 'center',
    marginLeft: 52,
    marginBottom: 4,
  },
  rtIcon: { fontSize: 14, marginRight: 4 },
  rtText: { fontSize: 13, fontWeight: '600' },
  body: {
    flexDirection: 'row',
  },
  avatar: {
    width: 40,
    height: 40,
    borderRadius: 20,
    marginRight: 12,
  },
  content: {
    flex: 1,
  },
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    flexWrap: 'nowrap',
  },
  name: {
    fontWeight: '700',
    fontSize: 15,
    flexShrink: 1,
    maxWidth: 140,
  },
  handle: {
    fontSize: 15,
    marginLeft: 4,
    flexShrink: 1,
    maxWidth: 100,
  },
  dot: { fontSize: 15 },
  time: { fontSize: 15 },
  replyContext: {
    fontSize: 13,
    marginBottom: 2,
  },
  actions: {
    flexDirection: 'row',
    marginTop: 10,
    alignItems: 'center',
  },
})
