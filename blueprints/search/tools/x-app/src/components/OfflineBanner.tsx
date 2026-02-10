import React from 'react'
import { View, Text, StyleSheet } from 'react-native'
import { useNetwork } from '../hooks/useNetwork'

export function OfflineBanner() {
  const { isOnline } = useNetwork()

  if (isOnline) return null

  return (
    <View style={styles.banner}>
      <Text style={styles.text}>You are offline — showing cached data</Text>
    </View>
  )
}

const styles = StyleSheet.create({
  banner: {
    backgroundColor: '#f59e0b',
    paddingVertical: 6,
    paddingHorizontal: 16,
    alignItems: 'center',
  },
  text: {
    color: '#000',
    fontSize: 13,
    fontWeight: '600',
  },
})
