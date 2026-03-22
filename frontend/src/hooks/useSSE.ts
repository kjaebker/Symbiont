import { useEffect, useRef } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { getToken } from '@/api/client'

export function useSSE() {
  const queryClient = useQueryClient()
  const retryDelay = useRef(1000)

  useEffect(() => {
    let es: EventSource | null = null
    let mounted = true

    function connect() {
      const token = getToken()
      if (!token || !mounted) return

      es = new EventSource(`/api/stream?token=${encodeURIComponent(token)}`)

      es.addEventListener('probe_update', () => {
        queryClient.invalidateQueries({ queryKey: ['probes'] })
      })

      es.addEventListener('outlet_update', () => {
        queryClient.invalidateQueries({ queryKey: ['outlets'] })
      })

      es.addEventListener('alert_fired', () => {
        queryClient.invalidateQueries({ queryKey: ['alerts'] })
      })

      es.onopen = () => {
        retryDelay.current = 1000
      }

      es.onerror = () => {
        es?.close()
        if (!mounted) return
        setTimeout(() => {
          retryDelay.current = Math.min(retryDelay.current * 2, 30000)
          connect()
        }, retryDelay.current)
      }
    }

    connect()

    return () => {
      mounted = false
      es?.close()
    }
  }, [queryClient])
}
