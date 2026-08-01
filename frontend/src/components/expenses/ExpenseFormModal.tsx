import { useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { format } from 'date-fns'
import { Modal, Alert, Spinner } from '@/components/common'
import { useCategories, useCreateExpense, useUpdateExpense } from '@/hooks/useFinance'
import { getErrorMessage, toDateInputValue } from '@/utils'
import type { Expense } from '@/types'

const schema = z.object({
  description: z.string().min(1, 'Descrição obrigatória').max(255),
  amount: z.coerce.number().positive('Valor deve ser positivo'),
  type: z.enum(['expense', 'income']),
  date: z.string().min(1, 'Data obrigatória'),
  category_id: z.string().optional(),
  tags: z.string().optional(),
})

type FormValues = z.infer<typeof schema>

interface Props {
  open: boolean
  onClose: () => void
  expense?: Expense
}

export default function ExpenseFormModal({ open, onClose, expense }: Props) {
  const { data: categories } = useCategories()
  const createMutation = useCreateExpense()
  const updateMutation = useUpdateExpense(expense?.id ?? '')

  const isEditing = !!expense
  const mutation = isEditing ? updateMutation : createMutation
  const error = mutation.error ? getErrorMessage(mutation.error) : null

  const {
    register,
    handleSubmit,
    reset,
    watch,
    setValue,
    formState: { errors },
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { type: 'expense', date: format(new Date(), 'yyyy-MM-dd') },
  })

  const selectedType = watch('type')

  useEffect(() => {
    if (expense) {
      reset({
        description: expense.description,
        amount: parseFloat(expense.amount),
        type: expense.type,
        date: toDateInputValue(expense.date),
        category_id: expense.category_id ?? '',
        tags: expense.tags ?? '',
      })
    } else {
      reset({ type: 'expense', date: format(new Date(), 'yyyy-MM-dd') })
    }
  }, [expense, reset])

  async function onSubmit(values: FormValues) {
    const payload = { ...values, category_id: values.category_id || undefined }
    if (isEditing) {
      await updateMutation.mutateAsync(payload)
    } else {
      await createMutation.mutateAsync(payload)
    }
    onClose()
    reset()
  }

  return (
    <Modal open={open} onClose={onClose} title={isEditing ? 'Editar Transação' : 'Nova Transação'}>
      <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
        {error && <Alert message={error} />}

        {/* Type toggle */}
        <div>
          <label className="label">Tipo</label>
          <div className="flex rounded-xl overflow-hidden border border-surface-border">
            <button
              type="button"
              onClick={() => setValue('type', 'expense')}
              className={`flex-1 py-2.5 text-sm font-medium transition-all duration-200
                ${selectedType === 'expense'
                  ? 'bg-red-500/15 text-red-400 border-r border-red-500/20'
                  : 'text-slate-500 hover:text-slate-300 border-r border-surface-border'}`}
            >
              💸 Despesa
            </button>
            <button
              type="button"
              onClick={() => setValue('type', 'income')}
              className={`flex-1 py-2.5 text-sm font-medium transition-all duration-200
                ${selectedType === 'income'
                  ? 'bg-brand-500/15 text-brand-400'
                  : 'text-slate-500 hover:text-slate-300'}`}
            >
              💰 Receita
            </button>
          </div>
          {errors.type && <p className="text-xs text-red-400 mt-1">{errors.type.message}</p>}
        </div>

        <div>
          <label className="label">Descrição</label>
          <input {...register('description')} className="input" placeholder="Ex: Almoço no restaurante" />
          {errors.description && <p className="text-xs text-red-400 mt-1">{errors.description.message}</p>}
        </div>

        <div className="grid grid-cols-2 gap-3">
          <div>
            <label className="label">Valor ($)</label>
            <input {...register('amount')} type="number" step="0.01" className="input" placeholder="0.00" />
            {errors.amount && <p className="text-xs text-red-400 mt-1">{errors.amount.message}</p>}
          </div>
          <div>
            <label className="label">Data</label>
            <input {...register('date')} type="date" className="input" />
            {errors.date && <p className="text-xs text-red-400 mt-1">{errors.date.message}</p>}
          </div>
        </div>

        <div>
          <label className="label">Categoria</label>
          <select {...register('category_id')} className="input">
            <option value="">Sem categoria</option>
            {categories?.map((c: any) => (
              <option key={c.id} value={c.id}>
                {c.icon} {c.name}
              </option>
            ))}
          </select>
        </div>

        <div>
          <label className="label">Tags (opcional)</label>
          <input {...register('tags')} className="input" placeholder="Ex: trabalho, pessoal" />
        </div>

        <div className="flex gap-3 pt-2">
          <button type="button" onClick={onClose} className="btn-ghost flex-1">
            Cancelar
          </button>
          <button type="submit" disabled={mutation.isPending} className="btn-primary flex-1 flex items-center justify-center gap-2">
            {mutation.isPending && <Spinner className="w-4 h-4 text-white" />}
            {isEditing ? 'Salvar' : 'Adicionar'}
          </button>
        </div>
      </form>
    </Modal>
  )
}
