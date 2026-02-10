import React from 'react'
import { View, Text, StyleSheet, TouchableOpacity, Alert, ScrollView } from 'react-native'
import { Stack } from 'expo-router'
import { useTheme } from '../src/theme'
import { clearCache } from '../src/cache/store'
import { X_AUTH_TOKEN, X_CT0 } from '../src/env'

export default function SettingsScreen() {
  const theme = useTheme()

  const handleClearCache = async () => {
    Alert.alert('Clear Cache', 'Are you sure you want to clear all cached data?', [
      { text: 'Cancel', style: 'cancel' },
      {
        text: 'Clear',
        style: 'destructive',
        onPress: async () => {
          await clearCache()
          Alert.alert('Done', 'Cache cleared')
        },
      },
    ])
  }

  return (
    <ScrollView style={[styles.container, { backgroundColor: theme.bg }]}>
      <Stack.Screen options={{ title: 'Settings' }} />

      <Text style={[styles.section, { color: theme.secondary }]}>CREDENTIALS</Text>
      <View style={[styles.card, { backgroundColor: theme.searchBg, borderColor: theme.border }]}>
        <Text style={[styles.cardLabel, { color: theme.secondary }]}>auth_token</Text>
        <Text style={[styles.cardValue, { color: theme.text }]} numberOfLines={1}>
          {X_AUTH_TOKEN ? X_AUTH_TOKEN.slice(0, 8) + '...' + X_AUTH_TOKEN.slice(-8) : 'Not set'}
        </Text>
        <Text style={[styles.cardLabel, { color: theme.secondary, marginTop: 12 }]}>ct0</Text>
        <Text style={[styles.cardValue, { color: theme.text }]} numberOfLines={1}>
          {X_CT0 ? X_CT0.slice(0, 8) + '...' + X_CT0.slice(-8) : 'Not set'}
        </Text>
      </View>
      <Text style={[styles.hint, { color: theme.secondary }]}>
        Credentials are configured in src/env.ts (prebuilt, like x-viewer's CF environment).
      </Text>

      <Text style={[styles.section, { color: theme.secondary, marginTop: 32 }]}>CACHE</Text>

      <TouchableOpacity
        style={[styles.button, { backgroundColor: '#ff3b30' }]}
        onPress={handleClearCache}
      >
        <Text style={styles.buttonText}>Clear Cache</Text>
      </TouchableOpacity>
    </ScrollView>
  )
}

const styles = StyleSheet.create({
  container: { flex: 1, padding: 16 },
  section: {
    fontSize: 13,
    fontWeight: '600',
    letterSpacing: 0.5,
    marginTop: 16,
    marginBottom: 12,
  },
  card: {
    borderWidth: 1,
    borderRadius: 12,
    padding: 16,
  },
  cardLabel: { fontSize: 12, fontWeight: '600', marginBottom: 4 },
  cardValue: { fontSize: 14, fontFamily: 'monospace' },
  hint: {
    fontSize: 13,
    lineHeight: 18,
    marginTop: 12,
  },
  button: {
    padding: 14,
    borderRadius: 24,
    alignItems: 'center',
    marginTop: 8,
  },
  buttonText: {
    color: '#fff',
    fontWeight: '700',
    fontSize: 16,
  },
})
