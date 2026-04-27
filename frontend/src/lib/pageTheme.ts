import { createContext, useContext } from 'react'

export type PageKey = 'dashboard' | 'livestock' | 'chemistry' | 'history' | 'alerts' | 'default'

export interface PageTheme {
  accent: string
  sub: string
  page: PageKey
  setTheme: (t: Partial<Pick<PageTheme, 'accent' | 'sub'>>) => void
}

export const routeThemes: Record<string, { accent: string; sub: string; page: PageKey }> = {
  '/':            { accent: '#3adffa', sub: '#6dfe9c', page: 'dashboard'  },
  '/livestock':   { accent: '#6dfe9c', sub: '#3adffa', page: 'livestock'  },
  '/chemistry':   { accent: '#a78bfa', sub: '#6dfe9c', page: 'chemistry'  },
  '/maintenance': { accent: '#fbbf24', sub: '#3adffa', page: 'default'    },
  '/history':     { accent: '#60a5fa', sub: '#8a90a8', page: 'history'    },
  '/alerts':      { accent: '#ff8796', sub: '#fbbf24', page: 'alerts'     },
  '/settings':    { accent: '#8a90a8', sub: '#5a6080', page: 'default'    },
}

export const defaultRouteTheme = { accent: '#3adffa', sub: '#6dfe9c', page: 'default' as PageKey }

export const PageThemeContext = createContext<PageTheme>({
  ...defaultRouteTheme,
  setTheme: () => {},
})

export function usePageTheme() {
  return useContext(PageThemeContext)
}
