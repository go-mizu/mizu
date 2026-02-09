const tidKeyword = 'obfiowerehiring'
const tidPairsURL = 'https://raw.githubusercontent.com/fa0311/x-client-transaction-id-pair-dict/refs/heads/main/pair.json'
const tidEpochOffset = 1682924400

interface TIDPair {
  animationKey: string
  verification: string
}

let cachedPairs: TIDPair[] = []
let lastFetch = 0

async function fetchTIDPairs(): Promise<TIDPair[]> {
  const now = Date.now()
  if (cachedPairs.length > 0 && now - lastFetch < 3600000) {
    return cachedPairs
  }

  try {
    const resp = await fetch(tidPairsURL)
    if (!resp.ok) {
      if (cachedPairs.length > 0) return cachedPairs
      throw new Error(`fetch TID pairs: ${resp.status}`)
    }
    const pairs: TIDPair[] = await resp.json()
    if (pairs.length === 0) {
      if (cachedPairs.length > 0) return cachedPairs
      throw new Error('TID pairs empty')
    }
    cachedPairs = pairs
    lastFetch = now
    return pairs
  } catch (e) {
    if (cachedPairs.length > 0) return cachedPairs
    throw e
  }
}

function base64Decode(s: string): Uint8Array {
  let padded = s
  const m = padded.length % 4
  if (m !== 0) padded += '='.repeat(4 - m)
  const binary = atob(padded)
  const bytes = new Uint8Array(binary.length)
  for (let i = 0; i < binary.length; i++) {
    bytes[i] = binary.charCodeAt(i)
  }
  return bytes
}

function base64Encode(bytes: Uint8Array): string {
  let binary = ''
  for (let i = 0; i < bytes.length; i++) {
    binary += String.fromCharCode(bytes[i])
  }
  return btoa(binary).replace(/=+$/, '')
}

export async function generateTID(path: string): Promise<string> {
  const pairs = await fetchTIDPairs()
  const pair = pairs[Math.floor(Math.random() * pairs.length)]

  const timeNow = Math.floor(Date.now() / 1000) - tidEpochOffset
  const timeNowBytes = new Uint8Array([
    timeNow & 0xff,
    (timeNow >> 8) & 0xff,
    (timeNow >> 16) & 0xff,
    (timeNow >> 24) & 0xff,
  ])

  const data = `GET!${path}!${timeNow}${tidKeyword}${pair.animationKey}`
  const encoder = new TextEncoder()
  const hashBuffer = await crypto.subtle.digest('SHA-256', encoder.encode(data))
  const hashBytes = new Uint8Array(hashBuffer)

  const keyBytes = base64Decode(pair.verification)

  const bytesArr = new Uint8Array(keyBytes.length + 4 + 16 + 1)
  bytesArr.set(keyBytes, 0)
  bytesArr.set(timeNowBytes, keyBytes.length)
  bytesArr.set(hashBytes.slice(0, 16), keyBytes.length + 4)
  bytesArr[keyBytes.length + 4 + 16] = 3

  const randomNum = Math.floor(Math.random() * 256)
  const tid = new Uint8Array(1 + bytesArr.length)
  tid[0] = randomNum
  for (let i = 0; i < bytesArr.length; i++) {
    tid[i + 1] = bytesArr[i] ^ randomNum
  }

  return base64Encode(tid)
}
