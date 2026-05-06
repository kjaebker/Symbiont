import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { ThumbsUp, ThumbsDown, Star } from 'lucide-react'
import { getDailyPrompt, respondToPrompt } from '@/api/client'
import { qk } from '@/api/queryKeys'
import { cn } from '@/lib/utils'

export function DailyPromptCard() {
  const queryClient = useQueryClient()

  const { data, isLoading } = useQuery({
    queryKey: qk.dailyPrompt.current,
    queryFn: getDailyPrompt,
    staleTime: 5 * 60 * 1000, // recheck every 5 min (crosses noon threshold)
  })

  const respond = useMutation({
    mutationFn: ({ question, response }: { question: string; response: string }) =>
      respondToPrompt(question, response),
    onSuccess: () => {
      queryClient.setQueryData(qk.dailyPrompt.current, { prompt: null })
    },
  })

  const [openText, setOpenText] = useState('')
  const [rating, setRating] = useState(0)
  const [dismissed, setDismissed] = useState(false)

  const prompt = data?.prompt ?? null

  if (isLoading || !prompt || dismissed) return null

  async function submitOpen() {
    if (!openText.trim() || !prompt) return
    await respond.mutateAsync({ question: prompt.question, response: openText.trim() })
  }

  async function submitThumbs(value: boolean) {
    if (!prompt) return
    await respond.mutateAsync({ question: prompt.question, response: value ? 'Yes' : 'No' })
  }

  async function submitRating(value: number) {
    if (!prompt) return
    setRating(value)
    await respond.mutateAsync({ question: prompt.question, response: `${value}/5` })
  }

  const isPending = respond.isPending
  const isDone = respond.isSuccess

  return (
    <div
      className={cn(
        'glass rounded-2xl px-5 py-4 mb-6 transition-all duration-500',
        isDone ? 'opacity-0 pointer-events-none h-0 overflow-hidden py-0 mb-0' : 'opacity-100',
      )}
    >
      <p className="text-xs text-primary uppercase tracking-widest font-semibold mb-2">
        Daily Check-In
      </p>
      <p className="text-on-surface font-medium mb-4 leading-snug">{prompt.question}</p>

      {prompt.type === 'thumbs' && (
        <div className="flex gap-3">
          <button
            onClick={() => submitThumbs(true)}
            disabled={isPending}
            className="flex items-center gap-2 px-5 py-2.5 rounded-xl bg-secondary/10 text-secondary hover:bg-secondary/20 transition-fluid disabled:opacity-50 font-medium text-sm"
          >
            <ThumbsUp className="h-4 w-4" />
            Yes
          </button>
          <button
            onClick={() => submitThumbs(false)}
            disabled={isPending}
            className="flex items-center gap-2 px-5 py-2.5 rounded-xl bg-surface-container-high text-on-surface-dim hover:bg-surface-container-highest transition-fluid disabled:opacity-50 font-medium text-sm"
          >
            <ThumbsDown className="h-4 w-4" />
            No
          </button>
          <button
            onClick={() => setDismissed(true)}
            className="ml-auto text-xs text-on-surface-faint hover:text-on-surface-dim transition-fluid"
          >
            Skip
          </button>
        </div>
      )}

      {prompt.type === 'rating' && (
        <div className="flex items-center gap-3">
          <div className="flex gap-1.5">
            {[1, 2, 3, 4, 5].map((n) => (
              <button
                key={n}
                onClick={() => submitRating(n)}
                disabled={isPending}
                className={cn(
                  'h-9 w-9 rounded-xl transition-fluid disabled:opacity-50 flex items-center justify-center',
                  n <= rating
                    ? 'bg-primary/20 text-primary'
                    : 'bg-surface-container-high text-on-surface-faint hover:bg-primary/10 hover:text-primary',
                )}
              >
                <Star className="h-4 w-4" fill={n <= rating ? 'currentColor' : 'none'} />
              </button>
            ))}
          </div>
          <button
            onClick={() => setDismissed(true)}
            className="ml-auto text-xs text-on-surface-faint hover:text-on-surface-dim transition-fluid"
          >
            Skip
          </button>
        </div>
      )}

      {prompt.type === 'open' && (
        <div className="space-y-3">
          <textarea
            rows={2}
            value={openText}
            onChange={(e) => setOpenText(e.target.value)}
            placeholder="Add a note..."
            className="w-full bg-surface-container-high text-base text-on-surface rounded-xl px-3 py-2.5 outline-none focus:ring-1 focus:ring-primary/40 resize-none placeholder:text-on-surface-faint transition-fluid"
          />
          <div className="flex items-center gap-3">
            <button
              onClick={submitOpen}
              disabled={isPending || !openText.trim()}
              className="px-4 py-2 rounded-xl bg-primary/20 text-primary text-sm font-medium hover:bg-primary/30 transition-fluid disabled:opacity-50"
            >
              {isPending ? 'Saving…' : 'Save'}
            </button>
            <button
              onClick={() => setDismissed(true)}
              className="text-xs text-on-surface-faint hover:text-on-surface-dim transition-fluid"
            >
              Skip
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
