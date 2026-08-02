import { useState } from 'react'
import { Plus, Filter, X } from 'lucide-react'
import { useExpenses, useCategories } from '@/hooks/useFinance'
import ExpenseTable from '@/components/expenses/ExpenseTable'
import ExpenseFormModal from '@/components/expenses/ExpenseFormModal'
import type { ListExpensesParams, TransactionType } from '@/types'
import { currentMonthRange, UNCATEGORIZED } from '@/utils'

export default function ExpensesPage() {
  const { start, end } = currentMonthRange()
  const [modalOpen, setModalOpen] = useState(false)
  const [showFilters, setShowFilters] = useState(false)
  const [page, setPage] = useState(1)
  const [filters, setFilters] = useState<ListExpensesParams>({
    start_date: start,
    end_date: end,
  })

  const { data, isLoading } = useExpenses({ ...filters, page, page_size: 20 })
  const { data: categories } = useCategories()

  function resetFilters() {
    setFilters({ start_date: start, end_date: end })
    setPage(1)
  }

  const hasActiveFilters = filters.category_id || filters.type ||
    (filters.start_date !== start) || (filters.end_date !== end)

  return (
    <div className="space-y-6 max-w-6xl">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white">Transações</h1>
          <p className="text-slate-500 text-sm mt-1">{data?.total ?? 0} transações encontradas</p>
        </div>
        <div className="flex gap-2">
          <button
            onClick={() => setShowFilters(!showFilters)}
            className={`btn-ghost flex items-center gap-2 ${hasActiveFilters ? 'text-brand-400' : ''}`}
          >
            <Filter size={16} />
            Filtros
            {hasActiveFilters && (
              <span className="w-2 h-2 rounded-full bg-brand-500" />
            )}
          </button>
          <button
            onClick={() => setModalOpen(true)}
            className="btn-primary flex items-center gap-2"
          >
            <Plus size={16} />
            Adicionar
          </button>
        </div>
      </div>

      {/* Filters panel */}
      {showFilters && (
        <div className="card animate-slide-up">
          <div className="flex items-center justify-between mb-4">
            <h3 className="font-medium text-white text-sm">Filtros</h3>
            {hasActiveFilters && (
              <button onClick={resetFilters} className="text-xs text-slate-500 hover:text-slate-300 flex items-center gap-1">
                <X size={12} /> Limpar
              </button>
            )}
          </div>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
            <div>
              <label className="label">Data inicial</label>
              <input
                type="date"
                value={filters.start_date ?? ''}
                onChange={(e) => { setFilters(f => ({ ...f, start_date: e.target.value })); setPage(1) }}
                className="input"
              />
            </div>
            <div>
              <label className="label">Data final</label>
              <input
                type="date"
                value={filters.end_date ?? ''}
                onChange={(e) => { setFilters(f => ({ ...f, end_date: e.target.value })); setPage(1) }}
                className="input"
              />
            </div>
            <div>
              <label className="label">Tipo</label>
              <select
                value={filters.type ?? ''}
                onChange={(e) => { setFilters(f => ({ ...f, type: (e.target.value || undefined) as TransactionType | undefined })); setPage(1) }}
                className="input"
              >
                <option value="">Todos</option>
                <option value="expense">Despesa</option>
                <option value="income">Receita</option>
              </select>
            </div>
            <div>
              <label className="label">Categoria</label>
              <select
                value={filters.category_id ?? ''}
                onChange={(e) => { setFilters(f => ({ ...f, category_id: e.target.value || undefined })); setPage(1) }}
                className="input"
              >
                <option value="">Todas</option>
                <option value={UNCATEGORIZED}>Sem categoria</option>
                {categories?.map((c) => (
                  <option key={c.id} value={c.id}>{c.icon} {c.name}</option>
                ))}
              </select>
            </div>
          </div>
        </div>
      )}

      {/* Table */}
      <div className="card">
        <ExpenseTable
          data={data}
          isLoading={isLoading}
          page={page}
          onPageChange={setPage}
          pageSize={20}
        />
      </div>

      <ExpenseFormModal open={modalOpen} onClose={() => setModalOpen(false)} />
    </div>
  )
}
