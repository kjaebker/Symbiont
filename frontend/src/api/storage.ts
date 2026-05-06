const TOKEN_KEY = 'symbiont_token'
const BUBBLES_KEY = 'ui:bubbles'

/**
 * Derives the thumbnail URL from an image_path using the same naming convention
 * as the API server: replace the file extension with "-thumb.jpg".
 *
 * E.g. "images/livestock-1-123.jpg" → "/images/livestock-1-123-thumb.jpg"
 */
export function thumbUrl(imagePath: string): string {
  const stem = imagePath.replace(/\.[^./]+$/, '')
  return `/${stem}-thumb.jpg`
}

export function originalUrl(imagePath: string): string {
  const stem = imagePath.replace(/\.[^./]+$/, '')
  return `/${stem}-original.jpg`
}

export function getBubblesEnabled(): boolean {
  try { return localStorage.getItem(BUBBLES_KEY) !== 'false' } catch { return true }
}

export function setBubblesEnabled(value: boolean): void {
  try { localStorage.setItem(BUBBLES_KEY, String(value)) } catch {}
}

export function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY)
}

export function setToken(token: string) {
  localStorage.setItem(TOKEN_KEY, token)
}

export function clearToken() {
  localStorage.removeItem(TOKEN_KEY)
}
