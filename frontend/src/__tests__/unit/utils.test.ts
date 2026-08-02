import { describe, it, expect } from 'vitest'
import { categoryLabel, formatCurrency, formatDate, formatDateTime, formatMonth, currentMonthRange, getErrorMessage, toDateInputValue } from '@/utils'

describe('formatCurrency', () => {
  it('formats positive numbers as USD', () => {
    const result = formatCurrency(1500)
    expect(result).toContain('1,500')
    expect(result).toContain('$')
  })

  it('formats string numbers', () => {
    const result = formatCurrency('250.99')
    expect(result).toContain('250')
  })

  it('handles zero', () => {
    const result = formatCurrency(0)
    expect(result).toContain('0')
  })
})

describe('formatDate', () => {
  it('formats ISO date strings', () => {
    const result = formatDate('2024-03-15T00:00:00Z')
    expect(result).toMatch(/15\/03\/2024/)
  })

  it('keeps the calendar day in negative UTC offsets', () => {
    // UTC midnight read as local time would render 14/03 in UTC-3.
    expect(formatDate('2024-03-15T00:00:00Z')).toMatch(/15\/03\/2024/)
    expect(formatDate('2024-03-15')).toMatch(/15\/03\/2024/)
  })

  it('returns original string on invalid date', () => {
    const result = formatDate('invalid-date')
    expect(result).toBe('invalid-date')
  })
})

describe('toDateInputValue', () => {
  it('keeps the calendar day of a UTC midnight instant', () => {
    expect(toDateInputValue('2024-03-15T00:00:00Z')).toBe('2024-03-15')
  })

  it('is stable across repeated edit round-trips', () => {
    // Regression: each edit used to shift the date one day back.
    let value = toDateInputValue('2024-03-15T00:00:00Z')
    for (let i = 0; i < 5; i++) {
      value = toDateInputValue(`${value}T00:00:00Z`)
    }
    expect(value).toBe('2024-03-15')
  })

  it('returns an empty string on invalid input', () => {
    expect(toDateInputValue('nope')).toBe('')
  })
})

describe('formatDateTime', () => {
  it('formats real timestamps in local time', () => {
    expect(formatDateTime('2024-03-15T12:00:00Z')).toMatch(/15\/03\/2024/)
  })

  it('returns original string on invalid date', () => {
    expect(formatDateTime('invalid-date')).toBe('invalid-date')
  })
})

describe('formatMonth', () => {
  it('formats YYYY-MM strings to short month', () => {
    const result = formatMonth('2024-01')
    expect(result.toLowerCase()).toContain('jan')
  })

  it('returns original string on invalid format', () => {
    const result = formatMonth('bad')
    expect(result).toBe('bad')
  })
})

describe('currentMonthRange', () => {
  it('returns start and end of current month', () => {
    const { start, end } = currentMonthRange()
    expect(start).toMatch(/^\d{4}-\d{2}-01$/)
    expect(end).toMatch(/^\d{4}-\d{2}-\d{2}$/)
    expect(new Date(start) <= new Date(end)).toBe(true)
  })
})

describe('getErrorMessage', () => {
  it('extracts axios-style error message', () => {
    const err = { response: { data: { message: 'Email already registered' } } }
    expect(getErrorMessage(err)).toBe('Email already registered')
  })

  it('extracts native Error message', () => {
    expect(getErrorMessage(new Error('something went wrong'))).toBe('something went wrong')
  })

  it('returns fallback for unknown errors', () => {
    expect(getErrorMessage(null)).toBe('Ocorreu um erro inesperado')
  })
})

describe('categoryLabel', () => {
  it('uses the category name when the row has a category', () => {
    expect(categoryLabel({ category_id: 'e8b5c1a0-1111-2222-3333-444455556666', category_name: 'Mercado' }))
      .toBe('Mercado')
  })

  it('labels rows with no category in Portuguese', () => {
    // The API sends a null category_id and an English placeholder name here.
    expect(categoryLabel({ category_name: 'Uncategorized' })).toBe('Sem categoria')
  })
})
