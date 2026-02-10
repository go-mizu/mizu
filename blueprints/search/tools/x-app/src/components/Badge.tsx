import React from 'react'
import { View, Text, StyleSheet } from 'react-native'

interface BadgeProps {
  verifiedType?: string
  size?: number
}

export function Badge({ verifiedType, size = 16 }: BadgeProps) {
  // Blue for regular, gold for Business, gray for Government
  let color = '#1d9bf0'
  if (verifiedType === 'Business') color = '#e2b719'
  else if (verifiedType === 'Government') color = '#829aab'

  return (
    <View style={[styles.badge, { width: size, height: size, borderRadius: size / 2, backgroundColor: color }]}>
      <Text style={[styles.check, { fontSize: size * 0.6 }]}>✓</Text>
    </View>
  )
}

const styles = StyleSheet.create({
  badge: {
    alignItems: 'center',
    justifyContent: 'center',
    marginLeft: 2,
  },
  check: {
    color: '#ffffff',
    fontWeight: '700',
    lineHeight: 14,
  },
})
