import { useRef, useState } from 'react'
import { Upload, FileText, CheckCircle2, XCircle, Loader2, RotateCcw } from 'lucide-react'
import { useUploadImport, useImports, useRevertImport } from '@/hooks/useFinance'
import { formatDateTime, cn, getErrorMessage } from '@/utils'
import { Alert, ConfirmDialog, useToast } from '@/components/common'
import type { Import } from '@/types'

export default function ImportUploader() {
  const inputRef = useRef<HTMLInputElement>(null)
  const [dragging, setDragging] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [revertingId, setRevertingId] = useState<string | null>(null)
  const uploadMutation = useUploadImport()
  const revertMutation = useRevertImport()
  const { data: imports, isLoading } = useImports()
  const { show, ToastContainer } = useToast()

  function handleFile(file: File) {
    setError(null)
    const allowed = ['.csv', '.ofx', '.qfx']
    const ext = '.' + file.name.split('.').pop()?.toLowerCase()
    if (!allowed.includes(ext)) {
      setError(`Formato não suportado. Use: ${allowed.join(', ')}`)
      return
    }
    uploadMutation.mutate(file, {
      onSuccess: (imp) => show(`${imp.imported} transações importadas com sucesso`),
      onError: (err) => setError(getErrorMessage(err)),
    })
  }

  function handleDrop(e: React.DragEvent) {
    e.preventDefault()
    setDragging(false)
    const file = e.dataTransfer.files[0]
    if (file) handleFile(file)
  }

  async function handleRevert() {
    if (!revertingId) return
    try {
      await revertMutation.mutateAsync(revertingId)
      show('Importação revertida e transações removidas')
    } catch (err) {
      show(getErrorMessage(err), 'error')
    } finally {
      setRevertingId(null)
    }
  }

  const statusConfig: Record<Import['status'], { icon: React.ReactNode; color: string; label: string }> = {
    pending:    { icon: <Loader2 size={14} className="animate-spin" />, color: 'text-slate-400', label: 'Pendente' },
    processing: { icon: <Loader2 size={14} className="animate-spin" />, color: 'text-blue-400', label: 'Processando' },
    completed:  { icon: <CheckCircle2 size={14} />, color: 'text-brand-400', label: 'Concluído' },
    failed:     { icon: <XCircle size={14} />, color: 'text-red-400', label: 'Falhou' },
  }

  return (
    <div className="space-y-6">
      <ToastContainer />
      {error && <Alert message={error} onClose={() => setError(null)} />}

      {/* Drop zone */}
      <div
        onClick={() => inputRef.current?.click()}
        onDragOver={(e) => { e.preventDefault(); setDragging(true) }}
        onDragLeave={() => setDragging(false)}
        onDrop={handleDrop}
        className={cn(
          'border-2 border-dashed rounded-2xl p-12 flex flex-col items-center gap-4 cursor-pointer transition-all duration-200',
          dragging
            ? 'border-brand-500 bg-brand-500/5'
            : 'border-surface-border hover:border-brand-500/50 hover:bg-surface-hover'
        )}
      >
        <input
          ref={inputRef}
          type="file"
          accept=".csv,.ofx,.qfx"
          className="hidden"
          onChange={(e) => { const f = e.target.files?.[0]; if (f) handleFile(f) }}
        />
        {uploadMutation.isPending ? (
          <>
            <Loader2 size={40} className="text-brand-500 animate-spin" />
            <p className="text-slate-400 text-sm">Importando...</p>
          </>
        ) : (
          <>
            <div className="w-16 h-16 rounded-2xl bg-brand-500/10 flex items-center justify-center">
              <Upload size={28} className="text-brand-400" />
            </div>
            <div className="text-center">
              <p className="text-white font-medium mb-1">Arraste o arquivo ou clique para selecionar</p>
              <p className="text-slate-500 text-sm">Suporta CSV, OFX e QFX — separador automático ( , ou ; )</p>
            </div>
          </>
        )}
      </div>

      {/* Format guide */}
      <div className="card">
        <h3 className="font-medium text-white mb-3 text-sm">Formatos suportados</h3>
        <div className="space-y-2 text-sm text-slate-400">
          <p><span className="text-brand-400 font-mono mr-2">CSV</span>Nubank, Itaú, Bradesco — detecta colunas automaticamente pelo cabeçalho</p>
          <p><span className="text-blue-400 font-mono mr-2">OFX/QFX</span>Exportação padrão da maioria dos bancos brasileiros</p>
        </div>
      </div>

      {/* Import history */}
      <div>
        <h2 className="font-semibold text-white mb-4">Histórico de importações</h2>
        {isLoading ? (
          <p className="text-slate-500 text-sm">Carregando...</p>
        ) : !imports?.length ? (
          <p className="text-slate-500 text-sm">Nenhuma importação realizada.</p>
        ) : (
          <div className="space-y-2">
            {imports.map((imp) => {
              const s = statusConfig[imp.status]
              return (
                <div key={imp.id} className="card flex items-center gap-4 py-4">
                  <div className="w-9 h-9 rounded-xl bg-surface flex items-center justify-center flex-shrink-0">
                    <FileText size={16} className="text-slate-400" />
                  </div>
                  <div className="flex-1 min-w-0">
                    <p className="text-sm text-white font-medium truncate">{imp.file_name}</p>
                    <p className="text-xs text-slate-500 mt-0.5">
                      {imp.imported} importados · {imp.errors} erros · {formatDateTime(imp.created_at)}
                    </p>
                  </div>
                  <div className={cn('flex items-center gap-1.5 text-xs font-medium', s.color)}>
                    {s.icon} {s.label}
                  </div>
                  {imp.status === 'completed' && imp.imported > 0 && (
                    <button
                      onClick={() => setRevertingId(imp.id)}
                      title="Reverter importação"
                      className="btn-danger p-1.5 flex items-center gap-1.5 text-xs"
                    >
                      <RotateCcw size={13} />
                      Reverter
                    </button>
                  )}
                </div>
              )
            })}
          </div>
        )}
      </div>

      <ConfirmDialog
        open={!!revertingId}
        title="Reverter importação"
        description="Todas as transações desta importação serão removidas permanentemente. Esta ação não pode ser desfeita."
        onConfirm={handleRevert}
        onCancel={() => setRevertingId(null)}
        loading={revertMutation.isPending}
      />
    </div>
  )
}