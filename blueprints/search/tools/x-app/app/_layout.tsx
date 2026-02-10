import React from 'react'
import { Stack } from 'expo-router'
import { StatusBar } from 'expo-status-bar'
import { useColorScheme } from 'react-native'

export default function RootLayout() {
  const scheme = useColorScheme()
  const isDark = scheme === 'dark'

  return (
    <>
      <StatusBar style={isDark ? 'light' : 'dark'} />
      <Stack
        screenOptions={{
          headerStyle: {
            backgroundColor: isDark ? '#000000' : '#ffffff',
          },
          headerTintColor: isDark ? '#e7e9ea' : '#0f1419',
        }}
      />
    </>
  )
}
