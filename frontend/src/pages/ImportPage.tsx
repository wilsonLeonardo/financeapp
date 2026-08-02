import ImportUploader from '@/components/importer/ImportUploader'

export default function ImportPage() {
  return (
    <div className="max-w-2xl space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-white">Importar extrato</h1>
        <p className="text-slate-500 text-sm mt-1">
          Importe extratos bancários em CSV ou OFX para inserir transações automaticamente
        </p>
      </div>
      <ImportUploader />
    </div>
  )
}
