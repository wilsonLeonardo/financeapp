import { useState } from 'react'
import { Loader2, Sparkles } from 'lucide-react'
import { useCategories } from '@/hooks/useFinance'
import { useApplyCategories, useSuggestCategories } from '@/hooks/useAI'
import { Alert, useToast } from '@/components/common'
import { cn, formatCurrency, formatDate, getErrorMessage } from '@/utils'
import { suggestionSource } from '@/utils/ai'
import type { CategorySuggestion } from '@/types'

const SUGGEST_LIMIT = 30

export default function CategorySuggestions() {
  const { data: categories } = useCategories()
  const suggest = useSuggestCategories()
  const apply = useApplyCategories()
  const { show, ToastContainer } = useToast()

  const [suggestions, setSuggestions] = useState<CategorySuggestion[]>([])
  // expense_id -> chosen category_id; '' means "leave uncategorized".
  const [choices, setChoices] = useState<Record<string, string>>({})

  const selected = Object.entries(choices).filter(([, categoryId]) => categoryId)

  function requestSuggestions() {
    suggest.mutate(
      { limit: SUGGEST_LIMIT },
      {
        onSuccess: (res) => {
          setSuggestions(res.suggestions)
          setChoices(Object.fromEntries(res.suggestions.map((s) => [s.expense_id, s.category_id ?? ''])))
          if (!res.suggestions.length) show('Nenhuma transação sem categoria encontrada')
        },
      }
    )
  }

  function applyChoices() {
    const items = selected.map(([expense_id, category_id]) => ({ expense_id, category_id }))
    apply.mutate(items, {
      onSuccess: (res) => {
        show(`${res.applied} transações categorizadas${res.failed.length ? `, ${res.failed.length} falharam` : ''}`)
        setSuggestions([])
        setChoices({})
      },
      onError: (err) => show(getErrorMessage(err), 'error'),
    })
  }

  return (
    <div className="card space-y-4">
      <ToastContainer />
      <div>
        <h2 className="font-semibold text-white flex items-center gap-2">
          <Sparkles size={16} className="text-brand-400" /> Categorizar com IA
        </h2>
        <p className="text-slate-500 text-sm mt-1">
          Sugere categorias a partir das transações que você já categorizou. Nada é salvo sem a sua confirmação.
        </p>
      </div>

      <button onClick={requestSuggestions} disabled={suggest.isPending} className="btn-primary flex items-center gap-2">
        {suggest.isPending ? <Loader2 size={16} className="animate-spin" /> : <Sparkles size={16} />}
        Sugerir categorias para transações sem categoria
      </button>

      {suggest.isPending && (
        <p className="text-slate-500 text-sm">
          Analisando até {SUGGEST_LIMIT} transações. Lançamentos parecidos com o seu histórico saem na hora; os demais
          passam pelo modelo local e levam alguns segundos cada.
        </p>
      )}
      {suggest.isError && <Alert message={getErrorMessage(suggest.error)} onClose={() => suggest.reset()} />}

      {suggestions.length > 0 && (
        <>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="text-left text-slate-500 border-b border-surface-border">
                  <th className="py-2 pr-3 font-medium">Data</th>
                  <th className="py-2 pr-3 font-medium">Descrição</th>
                  <th className="py-2 pr-3 font-medium text-right">Valor</th>
                  <th className="py-2 pr-3 font-medium">Categoria</th>
                  <th className="py-2 font-medium">Origem</th>
                </tr>
              </thead>
              <tbody>
                {suggestions.map((s) => (
                  <tr key={s.expense_id} className="border-b border-surface-border/50">
                    <td className="py-2 pr-3 text-slate-400 whitespace-nowrap">{formatDate(s.date)}</td>
                    <td className="py-2 pr-3 text-slate-200">{s.description}</td>
                    <td className="py-2 pr-3 text-right text-slate-300 whitespace-nowrap">{formatCurrency(s.amount)}</td>
                    <td className="py-2 pr-3">
                      <select
                        value={choices[s.expense_id] ?? ''}
                        onChange={(e) => setChoices((c) => ({ ...c, [s.expense_id]: e.target.value }))}
                        className="input py-1"
                        aria-label={`Categoria para ${s.description}`}
                      >
                        <option value="">Deixar sem categoria</option>
                        {categories?.map((c) => (
                          <option key={c.id} value={c.id}>
                            {c.name}
                          </option>
                        ))}
                      </select>
                    </td>
                    <td
                      className={cn(
                        'py-2 text-xs whitespace-nowrap',
                        s.method === 'knn' ? 'text-brand-400' : s.method === 'llm' ? 'text-blue-400' : 'text-slate-500'
                      )}
                    >
                      {suggestionSource(s.method, s.confidence)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <div className="flex justify-end">
            <button
              onClick={applyChoices}
              disabled={!selected.length || apply.isPending}
              className="btn-primary flex items-center gap-2"
            >
              {apply.isPending && <Loader2 size={16} className="animate-spin" />}
              Aplicar {selected.length} {selected.length === 1 ? 'categoria' : 'categorias'}
            </button>
          </div>
        </>
      )}
    </div>
  )
}
