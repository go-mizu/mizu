import React from 'react'
import { View, Text, StyleSheet, TouchableOpacity } from 'react-native'
import { Image } from 'expo-image'
import { useRouter } from 'expo-router'
import { useTheme } from '../theme'
import { Badge } from './Badge'
import type { Profile } from '../api/types'

interface UserCardProps {
  user: Profile
}

export function UserCard({ user }: UserCardProps) {
  const theme = useTheme()
  const router = useRouter()

  return (
    <TouchableOpacity
      style={[styles.container, { borderBottomColor: theme.border }]}
      onPress={() => router.push(`/${user.username}`)}
      activeOpacity={0.7}
    >
      <Image source={{ uri: user.avatar }} style={styles.avatar} />
      <View style={styles.body}>
        <View style={styles.nameRow}>
          <Text style={[styles.name, { color: theme.text }]} numberOfLines={1}>{user.name}</Text>
          {user.isBlueVerified && <Badge verifiedType={user.verifiedType} size={14} />}
        </View>
        <Text style={[styles.handle, { color: theme.secondary }]}>@{user.username}</Text>
        {user.biography ? (
          <Text style={[styles.bio, { color: theme.text }]} numberOfLines={2}>
            {user.biography}
          </Text>
        ) : null}
      </View>
    </TouchableOpacity>
  )
}

const styles = StyleSheet.create({
  container: {
    flexDirection: 'row',
    padding: 16,
    borderBottomWidth: StyleSheet.hairlineWidth,
  },
  avatar: {
    width: 48,
    height: 48,
    borderRadius: 24,
    marginRight: 12,
  },
  body: {
    flex: 1,
  },
  nameRow: {
    flexDirection: 'row',
    alignItems: 'center',
  },
  name: {
    fontWeight: '700',
    fontSize: 15,
    maxWidth: 200,
  },
  handle: {
    fontSize: 14,
    marginTop: 1,
  },
  bio: {
    fontSize: 14,
    marginTop: 4,
    lineHeight: 19,
  },
})
