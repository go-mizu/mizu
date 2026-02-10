import React from 'react'
import { View, Text, StyleSheet, TouchableOpacity, Dimensions } from 'react-native'
import { Image } from 'expo-image'
import { useRouter } from 'expo-router'
import * as Linking from 'expo-linking'
import { useTheme } from '../theme'
import { Badge } from './Badge'
import { RichText } from './RichText'
import { fmtNum, joinDate } from '../utils'
import type { Profile } from '../api/types'

const { width: SCREEN_WIDTH } = Dimensions.get('window')

interface ProfileHeaderProps {
  profile: Profile
}

export function ProfileHeader({ profile }: ProfileHeaderProps) {
  const theme = useTheme()
  const router = useRouter()

  return (
    <View>
      {/* Banner */}
      {profile.banner ? (
        <Image source={{ uri: profile.banner }} style={styles.banner} contentFit="cover" />
      ) : (
        <View style={[styles.banner, { backgroundColor: theme.blue }]} />
      )}

      {/* Avatar */}
      <View style={[styles.avatarContainer, { borderColor: theme.bg }]}>
        <Image source={{ uri: profile.avatar }} style={styles.avatar} />
      </View>

      <View style={styles.info}>
        {/* Name + Badge */}
        <View style={styles.nameRow}>
          <Text style={[styles.name, { color: theme.text }]}>{profile.name}</Text>
          {profile.isBlueVerified && <Badge verifiedType={profile.verifiedType} size={18} />}
          {profile.isPrivate && (
            <Text style={[styles.lock, { color: theme.secondary }]}> 🔒</Text>
          )}
        </View>

        <Text style={[styles.handle, { color: theme.secondary }]}>@{profile.username}</Text>

        {/* Bio */}
        {profile.biography ? (
          <View style={styles.bio}>
            <RichText text={profile.biography} urls={profile.website ? [profile.website] : []} />
          </View>
        ) : null}

        {/* Meta row */}
        <View style={styles.metaRow}>
          {profile.location ? (
            <Text style={[styles.meta, { color: theme.secondary }]}>📍 {profile.location}  </Text>
          ) : null}
          {profile.website ? (
            <TouchableOpacity onPress={() => Linking.openURL(profile.website)}>
              <Text style={[styles.meta, { color: theme.blue }]}>
                🔗 {profile.website.replace(/^https?:\/\//, '').slice(0, 30)}
              </Text>
            </TouchableOpacity>
          ) : null}
          {profile.joined ? (
            <Text style={[styles.meta, { color: theme.secondary }]}>📅 {joinDate(profile.joined)}</Text>
          ) : null}
        </View>

        {/* Stats */}
        <View style={styles.statsRow}>
          <Text style={[styles.statCount, { color: theme.text }]}>{fmtNum(profile.tweetsCount)}</Text>
          <Text style={[styles.statLabel, { color: theme.secondary }]}> Posts   </Text>

          <TouchableOpacity onPress={() => router.push(`/${profile.username}/following`)}>
            <Text>
              <Text style={[styles.statCount, { color: theme.text }]}>{fmtNum(profile.followingCount)}</Text>
              <Text style={[styles.statLabel, { color: theme.secondary }]}> Following   </Text>
            </Text>
          </TouchableOpacity>

          <TouchableOpacity onPress={() => router.push(`/${profile.username}/followers`)}>
            <Text>
              <Text style={[styles.statCount, { color: theme.text }]}>{fmtNum(profile.followersCount)}</Text>
              <Text style={[styles.statLabel, { color: theme.secondary }]}> Followers</Text>
            </Text>
          </TouchableOpacity>
        </View>
      </View>
    </View>
  )
}

const styles = StyleSheet.create({
  banner: {
    width: SCREEN_WIDTH,
    height: SCREEN_WIDTH / 3,
  },
  avatarContainer: {
    position: 'absolute',
    top: SCREEN_WIDTH / 3 - 44,
    left: 16,
    width: 88,
    height: 88,
    borderRadius: 44,
    borderWidth: 4,
    overflow: 'hidden',
  },
  avatar: {
    width: 80,
    height: 80,
    borderRadius: 40,
  },
  info: {
    paddingHorizontal: 16,
    paddingTop: 52,
  },
  nameRow: {
    flexDirection: 'row',
    alignItems: 'center',
  },
  name: {
    fontSize: 20,
    fontWeight: '800',
  },
  lock: { fontSize: 16 },
  handle: {
    fontSize: 15,
    marginTop: 1,
  },
  bio: {
    marginTop: 12,
  },
  metaRow: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    marginTop: 12,
  },
  meta: {
    fontSize: 14,
  },
  statsRow: {
    flexDirection: 'row',
    marginTop: 12,
    marginBottom: 4,
  },
  statCount: {
    fontWeight: '700',
    fontSize: 14,
  },
  statLabel: {
    fontSize: 14,
  },
})
