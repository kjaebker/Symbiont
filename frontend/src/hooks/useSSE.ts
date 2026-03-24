import { useEffect, useRef } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { getToken } from '@/api/client'
import { useToast } from '@/components/Toast'

export function useSSE() {
  const queryClient = useQueryClient()
  const { addToast } = useToast()
  const retryDelay = useRef(1000)
  const addToastRef = useRef(addToast)
  addToastRef.current = addToast

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

      es.addEventListener('alert_fired', (e) => {
        queryClient.invalidateQueries({ queryKey: ['alerts'] })
        try {
          const data = JSON.parse(e.data)
          addToastRef.current('alert', `Alert: ${data.probe_name ?? 'probe'} ${data.condition ?? 'triggered'} (${data.severity ?? 'warning'})`)
        } catch {
          addToastRef.current('alert', 'An alert has been triggered')
        }
      })

      es.addEventListener('alert_cleared', () => {
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
