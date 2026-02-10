import React from 'react'
import { View, Text, TouchableOpacity, StyleSheet } from 'react-native'
import { useTheme } from '../theme'

interface Tab {
  key: string
  label: string
}

interface TabBarProps {
  tabs: Tab[]
  active: string
  onSelect: (key: string) => void
}

export function TabBar({ tabs, active, onSelect }: TabBarProps) {
  const theme = useTheme()
  return (
    <View style={[styles.container, { borderBottomColor: theme.border }]}>
      {tabs.map((tab) => (
        <TouchableOpacity
          key={tab.key}
          style={styles.tab}
          onPress={() => onSelect(tab.key)}
          activeOpacity={0.7}
        >
          <Text style={[
            styles.label,
            { color: active === tab.key ? theme.tabActive : theme.tabInactive },
            active === tab.key && styles.activeLabel,
          ]}>
            {tab.label}
          </Text>
          {active === tab.key && (
            <View style={[styles.indicator, { backgroundColor: theme.blue }]} />
          )}
        </TouchableOpacity>
      ))}
    </View>
  )
}

const styles = StyleSheet.create({
  container: {
    flexDirection: 'row',
    borderBottomWidth: StyleSheet.hairlineWidth,
  },
  tab: {
    flex: 1,
    alignItems: 'center',
    paddingVertical: 14,
    position: 'relative',
  },
  label: {
    fontSize: 15,
    fontWeight: '500',
  },
  activeLabel: {
    fontWeight: '700',
  },
  indicator: {
    position: 'absolute',
    bottom: 0,
    height: 3,
    width: 56,
    borderRadius: 1.5,
  },
})
