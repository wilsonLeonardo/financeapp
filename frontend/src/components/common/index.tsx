import { cn } from '@/utils'
import { Loader2, AlertCircle, X } from 'lucide-react'
import { useState } from 'react'

// ── Spinner ───────────────────────────────────────────────────────────────────
export function Spinner({ className }: { className?: string }) {
  return <Loader2 className={cn('animate-spin text-brand-500', className)} size={20} />
}

export function PageLoader() {
  return (
    <div className="flex items-center justify-center h-64">
      <Spinner className="w-8 h-8" />
    </div>
  )
}

// ── Empty State ───────────────────────────────────────────────────────────────
export function EmptyState({ message, icon }: { message: string; icon?: React.ReactNode }) {
  return (
    <div className="flex flex-col items-center justify-center h-48 gap-3 text-slate-500">
      {icon && <div className="text-4xl">{icon}</div>}
      <p className="text-sm">{message}</p>
    </div>
  )
}

// ── Alert ─────────────────────────────────────────────────────────────────────
interface AlertProps { message: string; type?: 'error' | 'success' | 'info'; onClose?: () => void }

export function Alert({ message, type = 'error', onClose }: AlertProps) {
  const styles = {
    error: 'bg-red-500/10 border-red-500/20 text-red-400',
    success: 'bg-brand-500/10 border-brand-500/20 text-brand-400',
    info: 'bg-blue-500/10 border-blue-500/20 text-blue-400',
  }
  return (
    <div className={cn('flex items-start gap-3 p-3 rounded-xl border', styles[type])}>
      <AlertCircle size={16} className="flex-shrink-0 mt-0.5" />
      <p className="text-sm flex-1">{message}</p>
      {onClose && (
        <button onClick={onClose} className="flex-shrink-0 hover:opacity-70 transition-opacity">
          <X size={14} />
        </button>
      )}
    </div>
  )
}

// ── Modal ─────────────────────────────────────────────────────────────────────
interface ModalProps { open: boolean; onClose: () => void; title: string; children: React.ReactNode }

export function Modal({ open, onClose, title, children }: ModalProps) {
  if (!open) return null
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={onClose} />
      <div className="relative bg-surface-card border border-surface-border rounded-2xl w-full max-w-md shadow-2xl animate-slide-up">
        <div className="flex items-center justify-between p-6 border-b border-surface-border">
          <h2 className="font-semibold text-white text-lg">{title}</h2>
          <button onClick={onClose} className="btn-ghost p-1.5">
            <X size={16} />
          </button>
        </div>
        <div className="p-6">{children}</div>
      </div>
    </div>
  )
}

// ── Stat Card ─────────────────────────────────────────────────────────────────
interface StatCardProps {
  label: string
  value: string
  sub?: string
  icon: React.ReactNode
  trend?: 'up' | 'down' | 'neutral'
}

export function StatCard({ label, value, sub, icon, trend }: StatCardProps) {
  const trendColor = trend === 'up' ? 'text-brand-400' : trend === 'down' ? 'text-red-400' : 'text-slate-400'
  return (
    <div className="card flex items-start justify-between gap-4">
      <div>
        <p className="text-sm text-slate-500 mb-1">{label}</p>
        <p className="text-2xl font-bold text-white font-mono">{value}</p>
        {sub && <p className={cn('text-xs mt-1', trendColor)}>{sub}</p>}
      </div>
      <div className="w-10 h-10 rounded-xl bg-surface flex items-center justify-center text-brand-400 flex-shrink-0">
        {icon}
      </div>
    </div>
  )
}

// ── Confirm Dialog ────────────────────────────────────────────────────────────
interface ConfirmProps {
  open: boolean
  title: string
  description: string
  onConfirm: () => void
  onCancel: () => void
  loading?: boolean
}

export function ConfirmDialog({ open, title, description, onConfirm, onCancel, loading }: ConfirmProps) {
  return (
    <Modal open={open} onClose={onCancel} title={title}>
      <p className="text-sm text-slate-400 mb-6">{description}</p>
      <div className="flex gap-3 justify-end">
        <button onClick={onCancel} className="btn-ghost">Cancelar</button>
        <button
          onClick={onConfirm}
          disabled={loading}
          className="flex items-center gap-2 bg-red-500 hover:bg-red-600 text-white font-medium
                     px-4 py-2 rounded-xl transition-colors disabled:opacity-50"
        >
          {loading && <Spinner className="w-4 h-4 text-white" />}
          Confirmar
        </button>
      </div>
    </Modal>
  )
}

// ── Toast ─────────────────────────────────────────────────────────────────────
interface ToastState { message: string; type: 'success' | 'error' }

export function useToast() {
  const [toast, setToast] = useState<ToastState | null>(null)

  function show(message: string, type: 'success' | 'error' = 'success') {
    setToast({ message, type })
    setTimeout(() => setToast(null), 3500)
  }

  function ToastContainer() {
    if (!toast) return null
    return (
      <div className="fixed bottom-6 right-6 z-[100] animate-slide-up">
        <Alert message={toast.message} type={toast.type} onClose={() => setToast(null)} />
      </div>
    )
  }

  return { show, ToastContainer }
}
