import React from 'react'
import { Text, StyleSheet } from 'react-native'
import { useRouter } from 'expo-router'
import { useTheme } from '../theme'
import { parseText, TextSegment } from '../utils'
import * as Linking from 'expo-linking'

interface RichTextProps {
  text: string
  urls: string[]
  style?: any
}

export function RichText({ text, urls, style }: RichTextProps) {
  const theme = useTheme()
  const router = useRouter()
  const segments = parseText(text, urls)

  const handlePress = (segment: TextSegment) => {
    switch (segment.type) {
      case 'mention':
        router.push(`/${segment.href}`)
        break
      case 'hashtag':
        router.push(`/search?q=${encodeURIComponent('#' + segment.href)}`)
        break
      case 'url':
        if (segment.href) Linking.openURL(segment.href)
        break
    }
  }

  return (
    <Text style={[styles.text, { color: theme.text }, style]}>
      {segments.map((seg, i) => {
        if (seg.type === 'text') {
          return <Text key={i}>{seg.text}</Text>
        }
        return (
          <Text
            key={i}
            style={{ color: theme.blue }}
            onPress={() => handlePress(seg)}
          >
            {seg.text}
          </Text>
        )
      })}
    </Text>
  )
}

const styles = StyleSheet.create({
  text: {
    fontSize: 15,
    lineHeight: 20,
  },
})
