import React from 'react'
import { View, Text, StyleSheet, TouchableOpacity } from 'react-native'
import { Image } from 'expo-image'
import { useRouter } from 'expo-router'
import { useTheme } from '../theme'
import { Badge } from './Badge'
import { RichText } from './RichText'
import type { Tweet } from '../api/types'
import { relTime } from '../utils'

interface QuotedTweetProps {
  tweet: Tweet
}

export function QuotedTweet({ tweet }: QuotedTweetProps) {
  const theme = useTheme()
  const router = useRouter()

  return (
    <TouchableOpacity
      style={[styles.container, { borderColor: theme.border }]}
      onPress={() => router.push(`/${tweet.username}/status/${tweet.id}`)}
      activeOpacity={0.7}
    >
      <View style={styles.header}>
        <Image source={{ uri: tweet.avatar }} style={styles.avatar} />
        <Text style={[styles.name, { color: theme.text }]} numberOfLines={1}>
          {tweet.name}
        </Text>
        {tweet.isBlueVerified && <Badge verifiedType={tweet.verifiedType} size={14} />}
        <Text style={[styles.handle, { color: theme.secondary }]} numberOfLines={1}>
          @{tweet.username}
        </Text>
        <Text style={[styles.dot, { color: theme.secondary }]}> · </Text>
        <Text style={[styles.time, { color: theme.secondary }]}>{relTime(tweet.postedAt)}</Text>
      </View>
      <RichText text={tweet.text} urls={tweet.urls} style={styles.bodyText} />
      {tweet.photos.length > 0 && (
        <Image
          source={{ uri: tweet.photos[0] }}
          style={[styles.media, { borderColor: theme.border }]}
          contentFit="cover"
        />
      )}
    </TouchableOpacity>
  )
}

const styles = StyleSheet.create({
  container: {
    marginTop: 10,
    borderWidth: 1,
    borderRadius: 16,
    padding: 12,
    overflow: 'hidden',
  },
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    marginBottom: 4,
  },
  avatar: {
    width: 20,
    height: 20,
    borderRadius: 10,
    marginRight: 4,
  },
  name: {
    fontWeight: '700',
    fontSize: 14,
    maxWidth: 120,
  },
  handle: {
    fontSize: 14,
    marginLeft: 4,
    maxWidth: 100,
  },
  dot: { fontSize: 14 },
  time: { fontSize: 14 },
  bodyText: { fontSize: 14, lineHeight: 18 },
  media: {
    height: 150,
    borderRadius: 12,
    marginTop: 8,
    borderWidth: StyleSheet.hairlineWidth,
  },
})
