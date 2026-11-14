import { createContext, useContext } from 'react'

export interface PageTheme {
  accent: string
  sub: string
  setTheme: (t: Partial<Pick<PageTheme, 'accent' | 'sub'>>) => void
}

export const routeThemes: Record<string, { accent: string; sub: string }> = {
  '/':            { accent: '#3adffa', sub: '#6dfe9c' },
  '/livestock':   { accent: '#6dfe9c', sub: '#3adffa' },
  '/chemistry':   { accent: '#a78bfa', sub: '#6dfe9c' },
  '/tasks':       { accent: '#fbbf24', sub: '#3adffa' },
  '/history':     { accent: '#60a5fa', sub: '#8a90a8' },
  '/settings':    { accent: '#8a90a8', sub: '#5a6080' },
}

export const defaultRouteTheme = { accent: '#3adffa', sub: '#6dfe9c' }

export const PageThemeContext = createContext<PageTheme>({
  ...defaultRouteTheme,
  setTheme: () => {},
})

export function usePageTheme() {
  return useContext(PageThemeContext)
}
