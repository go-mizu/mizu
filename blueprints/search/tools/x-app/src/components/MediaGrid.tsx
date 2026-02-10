import React, { useState } from 'react'
import { View, StyleSheet, TouchableOpacity, Modal, Dimensions, Text, ScrollView } from 'react-native'
import { Image } from 'expo-image'
import { useTheme } from '../theme'

interface MediaGridProps {
  photos: string[]
  videos: string[]
  videoThumbnails: string[]
  gifs: string[]
}

export function MediaGrid({ photos, videos, videoThumbnails, gifs }: MediaGridProps) {
  const theme = useTheme()
  const [viewerIndex, setViewerIndex] = useState(-1)
  const allMedia: { url: string; type: 'photo' | 'video' | 'gif' }[] = []

  for (const p of photos) allMedia.push({ url: p, type: 'photo' })
  for (let i = 0; i < videos.length; i++) {
    allMedia.push({ url: videoThumbnails[i] || videos[i], type: 'video' })
  }
  for (const g of gifs) allMedia.push({ url: g, type: 'gif' })

  if (allMedia.length === 0) return null

  const renderItem = (item: typeof allMedia[0], index: number, style: any) => (
    <TouchableOpacity
      key={index}
      style={style}
      onPress={() => setViewerIndex(index)}
      activeOpacity={0.9}
    >
      <Image
        source={{ uri: item.url }}
        style={StyleSheet.absoluteFill}
        contentFit="cover"
        transition={200}
      />
      {item.type === 'video' && (
        <View style={styles.playOverlay}>
          <View style={styles.playButton}>
            <Text style={styles.playIcon}>▶</Text>
          </View>
        </View>
      )}
      {item.type === 'gif' && (
        <View style={styles.gifBadge}>
          <Text style={styles.gifText}>GIF</Text>
        </View>
      )}
    </TouchableOpacity>
  )

  const gap = 2

  return (
    <View style={[styles.container, { borderColor: theme.border }]}>
      {allMedia.length === 1 && renderItem(allMedia[0], 0, styles.single)}
      {allMedia.length === 2 && (
        <View style={styles.row}>
          {renderItem(allMedia[0], 0, [styles.half, { marginRight: gap }])}
          {renderItem(allMedia[1], 1, styles.half)}
        </View>
      )}
      {allMedia.length === 3 && (
        <View style={styles.row}>
          {renderItem(allMedia[0], 0, [styles.twoThirds, { marginRight: gap }])}
          <View style={styles.oneThird}>
            {renderItem(allMedia[1], 1, [styles.halfHeight, { marginBottom: gap }])}
            {renderItem(allMedia[2], 2, styles.halfHeight)}
          </View>
        </View>
      )}
      {allMedia.length >= 4 && (
        <View>
          <View style={[styles.row, { marginBottom: gap }]}>
            {renderItem(allMedia[0], 0, [styles.quadrant, { marginRight: gap }])}
            {renderItem(allMedia[1], 1, styles.quadrant)}
          </View>
          <View style={styles.row}>
            {renderItem(allMedia[2], 2, [styles.quadrant, { marginRight: gap }])}
            {renderItem(allMedia[3], 3, styles.quadrant)}
          </View>
        </View>
      )}

      {/* Full screen viewer modal */}
      <Modal visible={viewerIndex >= 0} transparent animationType="fade">
        <View style={styles.modal}>
          <TouchableOpacity style={styles.modalClose} onPress={() => setViewerIndex(-1)}>
            <Text style={styles.modalCloseText}>×</Text>
          </TouchableOpacity>
          {viewerIndex >= 0 && viewerIndex < allMedia.length && (
            <Image
              source={{ uri: allMedia[viewerIndex].url }}
              style={styles.modalImage}
              contentFit="contain"
            />
          )}
        </View>
      </Modal>
    </View>
  )
}

const styles = StyleSheet.create({
  container: {
    marginTop: 10,
    borderRadius: 16,
    overflow: 'hidden',
    borderWidth: StyleSheet.hairlineWidth,
  },
  single: { height: 280, width: '100%' },
  row: { flexDirection: 'row', height: 200 },
  half: { flex: 1, height: 200 },
  twoThirds: { flex: 2, height: 200 },
  oneThird: { flex: 1, height: 200 },
  halfHeight: { flex: 1 },
  quadrant: { flex: 1, height: 150 },
  playOverlay: {
    ...StyleSheet.absoluteFillObject,
    alignItems: 'center',
    justifyContent: 'center',
    backgroundColor: 'rgba(0,0,0,0.2)',
  },
  playButton: {
    width: 48,
    height: 48,
    borderRadius: 24,
    backgroundColor: 'rgba(29,155,240,0.9)',
    alignItems: 'center',
    justifyContent: 'center',
  },
  playIcon: { color: '#fff', fontSize: 20, marginLeft: 3 },
  gifBadge: {
    position: 'absolute',
    bottom: 8,
    left: 8,
    backgroundColor: '#000',
    paddingHorizontal: 6,
    paddingVertical: 2,
    borderRadius: 4,
  },
  gifText: { color: '#fff', fontSize: 12, fontWeight: '700' },
  modal: {
    flex: 1,
    backgroundColor: 'rgba(0,0,0,0.95)',
    alignItems: 'center',
    justifyContent: 'center',
  },
  modalClose: {
    position: 'absolute',
    top: 50,
    left: 16,
    zIndex: 10,
    width: 36,
    height: 36,
    borderRadius: 18,
    backgroundColor: 'rgba(255,255,255,0.15)',
    alignItems: 'center',
    justifyContent: 'center',
  },
  modalCloseText: { color: '#fff', fontSize: 24, lineHeight: 28 },
  modalImage: { width: '100%', height: '80%' },
})
