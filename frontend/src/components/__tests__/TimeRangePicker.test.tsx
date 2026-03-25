import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { TimeRangePicker } from '../TimeRangePicker'

describe('TimeRangePicker', () => {
  const now = new Date()
  const twoHoursAgo = new Date(now.getTime() - 2 * 60 * 60 * 1000)
  const defaultRange = { from: twoHoursAgo, to: now }

  it('renders preset buttons', () => {
    render(<TimeRangePicker value={defaultRange} onChange={() => {}} />)
    expect(screen.getByText('2h')).toBeDefined()
    expect(screen.getByText('6h')).toBeDefined()
    expect(screen.getByText('24h')).toBeDefined()
    expect(screen.getByText('7d')).toBeDefined()
    expect(screen.getByText('30d')).toBeDefined()
  })

  it('renders datetime inputs', () => {
    const { container } = render(
      <TimeRangePicker value={defaultRange} onChange={() => {}} />,
    )
    const inputs = container.querySelectorAll('input[type="datetime-local"]')
    expect(inputs.length).toBe(2)
  })

  it('calls onChange when preset is clicked', () => {
    const onChange = vi.fn()
    render(<TimeRangePicker value={defaultRange} onChange={onChange} />)
    fireEvent.click(screen.getByText('24h'))
    expect(onChange).toHaveBeenCalledTimes(1)
    const arg = onChange.mock.calls[0][0]
    expect(arg.from).toBeInstanceOf(Date)
    expect(arg.to).toBeInstanceOf(Date)
  })

  it('calls onChange when from input changes', () => {
    const onChange = vi.fn()
    const { container } = render(
      <TimeRangePicker value={defaultRange} onChange={onChange} />,
    )
    const fromInput = container.querySelectorAll(
      'input[type="datetime-local"]',
    )[0] as HTMLInputElement
    fireEvent.change(fromInput, { target: { value: '2026-01-01T00:00' } })
    expect(onChange).toHaveBeenCalledTimes(1)
  })
})
