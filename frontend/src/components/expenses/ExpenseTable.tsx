import { useState } from 'react'
import { Pencil, Trash2, ChevronLeft, ChevronRight } from 'lucide-react'
import { formatCurrency, formatDate } from '@/utils'
import { useDeleteExpense } from '@/hooks/useFinance'
import { ConfirmDialog, EmptyState, Spinner, useToast } from '@/components/common'
import ExpenseFormModal from './ExpenseFormModal'
import type { Expense, PaginatedResponse } from '@/types'

interface Props {
  data?: PaginatedResponse<Expense>
  isLoading: boolean
  page: number
  onPageChange: (p: number) => void
  pageSize: number
}

export default function ExpenseTable({ data, isLoading, page, onPageChange, pageSize }: Props) {
  const [editingExpense, setEditingExpense] = useState<Expense | null>(null)
  const [deletingId, setDeletingId] = useState<string | null>(null)
  const deleteMutation = useDeleteExpense()
  const { show, ToastContainer } = useToast()

  async function handleDelete() {
    if (!deletingId) return
    try {
      await deleteMutation.mutateAsync(deletingId)
      show('Transação removida com sucesso')
    } catch {
      show('Erro ao remover transação', 'error')
    } finally {
      setDeletingId(null)
    }
  }

  const totalPages = data ? Math.ceil(data.total / pageSize) : 0

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-48">
        <Spinner className="w-8 h-8" />
      </div>
    )
  }

  if (!data?.data.length) {
    return <EmptyState message="Nenhuma transação encontrada" icon="📭" />
  }

  return (
    <>
      <ToastContainer />

      <div className="overflow-x-auto">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-surface-border">
              {['Data', 'Descrição', 'Categoria', 'Tipo', 'Valor', ''].map((h) => (
                <th key={h} className="text-left text-slate-500 font-medium px-4 py-3 first:pl-0 last:pr-0">
                  {h}
                </th>
              ))}
            </tr>
          </thead>
          <tbody className="divide-y divide-surface-border">
            {data.data.map((expense) => (
              <tr key={expense.id} className="group hover:bg-surface-hover/50 transition-colors">
                <td className="px-4 py-3.5 pl-0 text-slate-400 whitespace-nowrap font-mono text-xs">
                  {formatDate(expense.date)}
                </td>
                <td className="px-4 py-3.5 text-slate-200 max-w-xs truncate">
                  {expense.description}
                  {expense.tags && (
                    <span className="ml-2 text-xs text-slate-500">#{expense.tags}</span>
                  )}
                </td>
                <td className="px-4 py-3.5">
                  {expense.category ? (
                    <span
                      className="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-xs font-medium"
                      style={{
                        backgroundColor: `${expense.category.color}20`,
                        color: expense.category.color,
                      }}
                    >
                      {expense.category.icon} {expense.category.name}
                    </span>
                  ) : (
                    <span className="text-slate-600 text-xs">—</span>
                  )}
                </td>
                <td className="px-4 py-3.5">
                  {expense.type === 'expense' ? (
                    <span className="badge-expense">↓ Despesa</span>
                  ) : (
                    <span className="badge-income">↑ Receita</span>
                  )}
                </td>
                <td className="px-4 py-3.5 font-mono font-semibold whitespace-nowrap">
                  <span className={expense.type === 'expense' ? 'text-red-400' : 'text-brand-400'}>
                    {expense.type === 'expense' ? '-' : '+'}
                    {formatCurrency(expense.amount)}
                  </span>
                </td>
                <td className="px-4 py-3.5 pr-0">
                  <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity justify-end">
                    <button onClick={() => setEditingExpense(expense)} className="btn-ghost p-1.5">
                      <Pencil size={14} />
                    </button>
                    <button onClick={() => setDeletingId(expense.id)} className="btn-danger p-1.5">
                      <Trash2 size={14} />
                    </button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="flex items-center justify-between pt-4 border-t border-surface-border mt-4">
          <p className="text-sm text-slate-500">
            {data.total} transações · página {page} de {totalPages}
          </p>
          <div className="flex gap-2">
            <button
              onClick={() => onPageChange(page - 1)}
              disabled={page <= 1}
              className="btn-ghost p-2 disabled:opacity-30"
            >
              <ChevronLeft size={16} />
            </button>
            <button
              onClick={() => onPageChange(page + 1)}
              disabled={page >= totalPages}
              className="btn-ghost p-2 disabled:opacity-30"
            >
              <ChevronRight size={16} />
            </button>
          </div>
        </div>
      )}

      <ExpenseFormModal
        open={!!editingExpense}
        onClose={() => setEditingExpense(null)}
        expense={editingExpense ?? undefined}
      />

      <ConfirmDialog
        open={!!deletingId}
        title="Remover transação"
        description="Esta ação não pode ser desfeita. Deseja continuar?"
        onConfirm={handleDelete}
        onCancel={() => setDeletingId(null)}
        loading={deleteMutation.isPending}
      />
    </>
  )
}
