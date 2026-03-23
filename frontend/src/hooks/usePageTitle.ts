import { useEffect } from 'react'

export function usePageTitle(title: string) {
  useEffect(() => {
    document.title = title ? `Symbiont — ${title}` : 'Symbiont'
  }, [title])
}
