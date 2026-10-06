import type { SuggestionMethod } from '@/types'

const TOOL_LABELS: Record<string, { running: string; done: string }> = {
  list_categories: { running: 'Consultando suas categorias', done: 'Consultou suas categorias' },
  spending_by_category: { running: 'Somando gastos por categoria', done: 'Somou gastos por categoria' },
  monthly_totals: { running: 'Calculando totais mensais', done: 'Calculou totais mensais' },
  search_transactions: { running: 'Buscando transações', done: 'Buscou transações' },
  suggest_categories: { running: 'Sugerindo categorias', done: 'Sugeriu categorias' },
}

export function toolLabel(name: string, done: boolean): string {
  const label = TOOL_LABELS[name]
  if (!label) return done ? `Usou ${name}` : `Usando ${name}`
  return done ? label.done : `${label.running}…`
}

export function suggestionSource(method: SuggestionMethod, confidence: number | null): string {
  if (method === 'knn') return confidence !== null ? `Histórico · ${Math.round(confidence * 100)}%` : 'Histórico'
  if (method === 'llm') return 'IA'
  return 'Sem sugestão'
}
