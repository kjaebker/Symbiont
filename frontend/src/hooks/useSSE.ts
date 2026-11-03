import { useEffect, useRef } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { getToken } from '@/api/client'
import { qk } from '@/api/queryKeys'
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

      // `exact: true` — refresh only the current-value lists. The server
      // emits these on every poll (10s); a prefix match would also refetch
      // every sparkline and history chart under ['probes'] / ['outlets'],
      // which have their own, much slower refetch cadence.
      es.addEventListener('probe_update', () => {
        queryClient.invalidateQueries({ queryKey: qk.probes.all, exact: true })
      })

      es.addEventListener('outlet_update', () => {
        queryClient.invalidateQueries({ queryKey: qk.outlets.all, exact: true })
      })

      es.addEventListener('alert_fired', (e) => {
        queryClient.invalidateQueries({ queryKey: qk.alerts.all })
        try {
          const data = JSON.parse(e.data)
          addToastRef.current('alert', `Alert: ${data.probe_name ?? 'probe'} ${data.condition ?? 'triggered'} (${data.severity ?? 'warning'})`)
        } catch {
          addToastRef.current('alert', 'An alert has been triggered')
        }
      })

      es.addEventListener('alert_cleared', () => {
        queryClient.invalidateQueries({ queryKey: qk.alerts.all })
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
