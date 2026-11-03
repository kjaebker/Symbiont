import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { TimeRangePicker } from '../TimeRangePicker'

const HOUR = 60 * 60 * 1000
const PRESETS = ['Last 2h', 'Last 6h', 'Last 24h', 'Last 7d', 'Last 30d']

function lastHours(h: number) {
  const now = new Date()
  return { from: new Date(now.getTime() - h * HOUR), to: now }
}

// The picker is a compact dropdown: presets and date inputs only render
// after the trigger button is clicked.
function renderOpen(value = lastHours(2), onChange = vi.fn()) {
  const utils = render(<TimeRangePicker value={value} onChange={onChange} />)
  fireEvent.click(screen.getByRole('button', { name: /last 2h|–/i }))
  return { ...utils, onChange }
}

describe('TimeRangePicker', () => {
  it('shows the active preset on the trigger and starts closed', () => {
    const { container } = render(
      <TimeRangePicker value={lastHours(2)} onChange={() => {}} />,
    )
    expect(screen.getByRole('button', { name: 'Last 2h' })).toBeDefined()
    expect(container.querySelectorAll('input[type="datetime-local"]').length).toBe(0)
    expect(screen.queryByRole('button', { name: 'Last 24h' })).toBeNull()
  })

  it('shows a date range label when no preset matches', () => {
    const to = new Date(2026, 0, 10, 12, 0)
    const from = new Date(2026, 0, 3, 12, 0)
    render(<TimeRangePicker value={{ from, to }} onChange={() => {}} />)
    expect(screen.getByRole('button').textContent).toMatch(/–/)
  })

  it('opens to reveal every preset and both datetime inputs', () => {
    const { container } = renderOpen()
    for (const label of PRESETS.slice(1)) {
      expect(screen.getByRole('button', { name: label })).toBeDefined()
    }
    expect(container.querySelectorAll('input[type="datetime-local"]').length).toBe(2)
  })

  it('applies a preset range and closes when a preset is clicked', () => {
    const { container, onChange } = renderOpen()
    fireEvent.click(screen.getByRole('button', { name: 'Last 24h' }))

    expect(onChange).toHaveBeenCalledTimes(1)
    const { from, to } = onChange.mock.calls[0][0]
    expect(from).toBeInstanceOf(Date)
    expect(to.getTime() - from.getTime()).toBe(24 * HOUR)
    expect(container.querySelectorAll('input[type="datetime-local"]').length).toBe(0)
  })

  it('calls onChange when the from input changes', () => {
    const { container, onChange } = renderOpen()
    const fromInput = container.querySelectorAll(
      'input[type="datetime-local"]',
    )[0] as HTMLInputElement
    fireEvent.change(fromInput, { target: { value: '2026-01-01T00:00' } })

    expect(onChange).toHaveBeenCalledTimes(1)
    expect(onChange.mock.calls[0][0].from).toEqual(new Date('2026-01-01T00:00'))
  })
})
