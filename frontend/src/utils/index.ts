import { clsx, type ClassValue } from 'clsx'
import { twMerge } from 'tailwind-merge'
import { format, parseISO, startOfMonth, endOfMonth } from 'date-fns'
import { ptBR } from 'date-fns/locale'

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

export function formatCurrency(value: number | string): string {
  const num = typeof value === 'string' ? parseFloat(value) : value
  return new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD' }).format(num)
}

// The API returns transaction dates as UTC midnight instants ("2024-03-15T00:00:00Z"),
// but they represent a calendar day, not a point in time. Building a Date from the full
// string and then reading it back in local time shifts the day backwards for every
// negative UTC offset (UTC-3 turns 15/03 into 14/03), so we only ever read the
// YYYY-MM-DD prefix and rebuild the date in local time.
export function parseDateOnly(dateStr: string): Date {
  const datePart = dateStr.slice(0, 10)
  if (!/^\d{4}-\d{2}-\d{2}$/.test(datePart)) {
    throw new RangeError(`invalid date: ${dateStr}`)
  }
  const [year, month, day] = datePart.split('-').map(Number)
  return new Date(year, month - 1, day)
}

// Formats a calendar date (e.g. Expense.date) without any timezone conversion.
export function formatDate(dateStr: string, fmt = 'dd/MM/yyyy'): string {
  try {
    return format(parseDateOnly(dateStr), fmt, { locale: ptBR })
  } catch {
    return dateStr
  }
}

// Formats a real instant (e.g. created_at) in the viewer's local timezone.
export function formatDateTime(dateStr: string, fmt = 'dd/MM/yyyy'): string {
  try {
    return format(parseISO(dateStr), fmt, { locale: ptBR })
  } catch {
    return dateStr
  }
}

// Value for an <input type="date">, taken straight from the calendar day.
export function toDateInputValue(dateStr: string): string {
  try {
    return format(parseDateOnly(dateStr), 'yyyy-MM-dd')
  } catch {
    return ''
  }
}

export function formatMonth(monthStr: string): string {
  // monthStr is "YYYY-MM"
  try {
    const [year, month] = monthStr.split('-').map(Number)
    const date = new Date(year, month - 1, 1)
    return format(date, "MMM 'yy", { locale: ptBR })
  } catch {
    return monthStr
  }
}

export function currentMonthRange(): { start: string; end: string } {
  const now = new Date()
  return {
    start: format(startOfMonth(now), 'yyyy-MM-dd'),
    end: format(endOfMonth(now), 'yyyy-MM-dd'),
  }
}

export function getErrorMessage(err: unknown): string {
  if (err && typeof err === 'object' && 'response' in err) {
    const res = (err as { response?: { data?: { message?: string } } }).response
    if (res?.data?.message) return res.data.message
  }
  if (err instanceof Error) return err.message
  return 'Ocorreu um erro inesperado'
}

export const CATEGORY_COLORS = [
  '#22c55e', '#3b82f6', '#f59e0b', '#ef4444', '#8b5cf6',
  '#06b6d4', '#ec4899', '#f97316', '#84cc16', '#6366f1',
]

export const CATEGORY_ICONS = [
  '🍔', '🚗', '🏠', '🎮', '👕', '💊', '📚', '✈️',
  '🎵', '💡', '🛒', '🏋️', '🐶', '💸', '💼', '🎁',
]
