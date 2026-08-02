import { useState } from 'react'
import { Plus, Trash2 } from 'lucide-react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useCategories, useCreateCategory, useDeleteCategory } from '@/hooks/useFinance'
import { ConfirmDialog, PageLoader, EmptyState, useToast } from '@/components/common'
import { CATEGORY_COLORS, CATEGORY_ICONS, getErrorMessage } from '@/utils'

const schema = z.object({
  name: z.string().min(1, 'Nome obrigatório').max(50),
  icon: z.string().default('📁'),
  color: z.string().default('#22c55e'),
})
type FormValues = z.infer<typeof schema>

export default function CategoriesPage() {
  const { data: categories, isLoading } = useCategories()
  const createMutation = useCreateCategory()
  const deleteMutation = useDeleteCategory()
  const [deletingId, setDeletingId] = useState<string | null>(null)
  const { show, ToastContainer } = useToast()
  const [selectedIcon, setSelectedIcon] = useState('📁')
  const [selectedColor, setSelectedColor] = useState(CATEGORY_COLORS[0])

  const { register, handleSubmit, reset, formState: { errors } } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { icon: '📁', color: CATEGORY_COLORS[0] },
  })

  async function onSubmit(values: FormValues) {
    try {
      await createMutation.mutateAsync({ ...values, icon: selectedIcon, color: selectedColor })
      show('Categoria criada com sucesso')
      reset()
      setSelectedIcon('📁')
      setSelectedColor(CATEGORY_COLORS[0])
    } catch (err) {
      show(getErrorMessage(err), 'error')
    }
  }

  async function handleDelete() {
    if (!deletingId) return
    try {
      await deleteMutation.mutateAsync(deletingId)
      show('Categoria removida')
    } catch (err) {
      show(getErrorMessage(err), 'error')
    } finally {
      setDeletingId(null)
    }
  }

  return (
    <div className="max-w-3xl space-y-6">
      <ToastContainer />
      <div>
        <h1 className="text-2xl font-bold text-white">Categorias</h1>
        <p className="text-slate-500 text-sm mt-1">Organize suas transações com categorias personalizadas</p>
      </div>

      {/* Create form */}
      <div className="card">
        <h2 className="font-semibold text-white mb-4">Nova categoria</h2>
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <div>
            <label className="label">Nome</label>
            <input {...register('name')} className="input" placeholder="Ex: Alimentação" />
            {errors.name && <p className="text-xs text-red-400 mt-1">{errors.name.message}</p>}
          </div>

          <div>
            <label className="label">Ícone</label>
            <div className="flex flex-wrap gap-2">
              {CATEGORY_ICONS.map((icon) => (
                <button
                  key={icon}
                  type="button"
                  onClick={() => setSelectedIcon(icon)}
                  className={`w-9 h-9 rounded-lg text-lg transition-all duration-150 flex items-center justify-center
                    ${selectedIcon === icon
                      ? 'bg-brand-500/20 ring-2 ring-brand-500 scale-110'
                      : 'bg-surface hover:bg-surface-hover'}`}
                >
                  {icon}
                </button>
              ))}
            </div>
          </div>

          <div>
            <label className="label">Cor</label>
            <div className="flex flex-wrap gap-2">
              {CATEGORY_COLORS.map((color) => (
                <button
                  key={color}
                  type="button"
                  onClick={() => setSelectedColor(color)}
                  className={`w-7 h-7 rounded-full transition-all duration-150 border-2
                    ${selectedColor === color ? 'border-white scale-110' : 'border-transparent'}`}
                  style={{ backgroundColor: color }}
                />
              ))}
            </div>
          </div>

          <button
            type="submit"
            disabled={createMutation.isPending}
            className="btn-primary flex items-center gap-2"
          >
            <Plus size={16} />
            Criar categoria
          </button>
        </form>
      </div>

      {/* Categories list */}
      <div className="card">
        <h2 className="font-semibold text-white mb-4">Suas categorias ({categories?.length ?? 0})</h2>
        {isLoading ? (
          <PageLoader />
        ) : !categories?.length ? (
          <EmptyState message="Nenhuma categoria criada ainda" icon="🏷️" />
        ) : (
          <div className="space-y-2">
            {categories.map((cat) => (
              <div
                key={cat.id}
                className="flex items-center gap-3 p-3 rounded-xl hover:bg-surface-hover transition-colors group"
              >
                <div
                  className="w-9 h-9 rounded-xl flex items-center justify-center text-lg flex-shrink-0"
                  style={{ backgroundColor: `${cat.color}20` }}
                >
                  {cat.icon}
                </div>
                <span className="flex-1 text-sm font-medium text-slate-200">{cat.name}</span>
                <div
                  className="w-3 h-3 rounded-full flex-shrink-0"
                  style={{ backgroundColor: cat.color }}
                />
                <button
                  onClick={() => setDeletingId(cat.id)}
                  className="btn-danger p-1.5 opacity-0 group-hover:opacity-100 transition-opacity"
                >
                  <Trash2 size={13} />
                </button>
              </div>
            ))}
          </div>
        )}
      </div>

      <ConfirmDialog
        open={!!deletingId}
        title="Remover categoria"
        description="As transações vinculadas a esta categoria não serão afetadas. Deseja continuar?"
        onConfirm={handleDelete}
        onCancel={() => setDeletingId(null)}
        loading={deleteMutation.isPending}
      />
    </div>
  )
}
