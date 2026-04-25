import { createContext, useContext } from 'react'

export type PageKey = 'dashboard' | 'livestock' | 'chemistry' | 'history' | 'alerts' | 'default'

export interface PageTheme {
  accent: string
  sub: string
  page: PageKey
  setTheme: (t: Partial<Pick<PageTheme, 'accent' | 'sub'>>) => void
}

export const routeThemes: Record<string, { accent: string; sub: string; page: PageKey }> = {
  '/':           { accent: '#3adffa', sub: '#6dfe9c', page: 'dashboard' },
  '/livestock':  { accent: '#ff8796', sub: '#3adffa', page: 'livestock' },
  '/chemistry':  { accent: '#6dfe9c', sub: '#3adffa', page: 'chemistry' },
  '/alerts':     { accent: '#ff8796', sub: '#fbbf24', page: 'alerts' },
  '/history':    { accent: '#3adffa', sub: '#5a6080', page: 'history' },
}

export const defaultRouteTheme = { accent: '#3adffa', sub: '#6dfe9c', page: 'default' as PageKey }

export const PageThemeContext = createContext<PageTheme>({
  ...defaultRouteTheme,
  setTheme: () => {},
})

export function usePageTheme() {
  return useContext(PageThemeContext)
}
