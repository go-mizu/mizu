import cssText from './styles.css'

function hash(s: string): string {
  let h = 0
  for (let i = 0; i < s.length; i++) {
    h = ((h << 5) - h + s.charCodeAt(i)) | 0
  }
  return (h >>> 0).toString(36)
}

export const cssHash = hash(cssText)
export const cssURL = `/s/${cssHash}.css`
export { cssText }
