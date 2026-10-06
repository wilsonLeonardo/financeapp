import { describe, expect, it } from 'vitest'
import { suggestionSource, toolLabel } from '@/utils/ai'

describe('toolLabel', () => {
  it('describes known tools while running and when done', () => {
    expect(toolLabel('spending_by_category', false)).toBe('Somando gastos por categoria…')
    expect(toolLabel('spending_by_category', true)).toBe('Somou gastos por categoria')
  })

  it('falls back to the raw tool name', () => {
    expect(toolLabel('new_tool', false)).toBe('Usando new_tool')
    expect(toolLabel('new_tool', true)).toBe('Usou new_tool')
  })
})

describe('suggestionSource', () => {
  it('shows history matches with their similarity', () => {
    expect(suggestionSource('knn', 0.964)).toBe('Histórico · 96%')
  })

  it('labels model and empty suggestions', () => {
    expect(suggestionSource('llm', null)).toBe('IA')
    expect(suggestionSource('none', null)).toBe('Sem sugestão')
  })
})
