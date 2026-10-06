import { useEffect, useMemo, useRef, useState } from 'react'
import { useChat } from '@ai-sdk/react'
import { DefaultChatTransport, type UIMessage } from 'ai'
import { Bot, CheckCircle2, Loader2, Send, Sparkles, Square, User, Wrench, XCircle } from 'lucide-react'
import { Alert } from '@/components/common'
import { AI_URL, authHeaders } from '@/services/ai'
import { cn } from '@/utils'
import { toolLabel } from '@/utils/ai'

const STARTERS = [
  'Quanto gastei por categoria este mês?',
  'Como foram meus gastos nos últimos 6 meses?',
  'Quais foram minhas corridas de Uber em setembro?',
  'Tenho transações sem categoria? Sugira categorias.',
]

export default function AssistantPage() {
  // The ai-service speaks the AI SDK stream protocol, so useChat needs no custom parsing.
  const transport = useMemo(
    () => new DefaultChatTransport({ api: `${AI_URL}/chat`, headers: () => authHeaders() }),
    []
  )
  const { messages, sendMessage, status, stop, error, clearError } = useChat({ transport })
  const [input, setInput] = useState('')
  const endRef = useRef<HTMLDivElement>(null)
  const busy = status === 'submitted' || status === 'streaming'

  useEffect(() => {
    endRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages, status])

  function submit(text: string) {
    const question = text.trim()
    if (!question || busy) return
    sendMessage({ text: question })
    setInput('')
  }

  return (
    <div className="max-w-3xl mx-auto flex flex-col h-full">
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-white flex items-center gap-2">
          <Sparkles size={22} className="text-brand-400" /> Assistente
        </h1>
        <p className="text-slate-500 text-sm mt-1">
          Pergunte sobre seus gastos e categorias. Roda 100% local com Ollama: seus dados não saem da sua máquina.
        </p>
      </div>

      <div className="flex-1 space-y-5 pb-4">
        {messages.length === 0 && (
          <div className="grid sm:grid-cols-2 gap-3">
            {STARTERS.map((s) => (
              <button
                key={s}
                onClick={() => submit(s)}
                className="card text-left text-sm text-slate-300 hover:border-brand-500/50 hover:text-white transition-colors"
              >
                {s}
              </button>
            ))}
          </div>
        )}

        {messages.map((m) => (
          <ChatMessage key={m.id} message={m} />
        ))}

        {status === 'submitted' && (
          <div className="flex items-center gap-2 text-slate-500 text-sm pl-11">
            <Loader2 size={14} className="animate-spin" /> Pensando… modelos locais podem levar alguns segundos.
          </div>
        )}

        {error && (
          <Alert
            message={`Não consegui responder: ${error.message}. Verifique se o ai-service e o Ollama estão rodando.`}
            onClose={clearError}
          />
        )}
        <div ref={endRef} />
      </div>

      <form
        onSubmit={(e) => {
          e.preventDefault()
          submit(input)
        }}
        className="sticky bottom-0 flex gap-2 pt-3 bg-surface"
      >
        <input
          value={input}
          onChange={(e) => setInput(e.target.value)}
          placeholder="Ex.: quanto gastei com mercado em setembro?"
          className="input flex-1"
          aria-label="Mensagem para o assistente"
        />
        {busy ? (
          <button type="button" onClick={() => stop()} className="btn-ghost px-4" aria-label="Parar resposta">
            <Square size={16} />
          </button>
        ) : (
          <button type="submit" className="btn-primary px-4" disabled={!input.trim()} aria-label="Enviar">
            <Send size={16} />
          </button>
        )}
      </form>
    </div>
  )
}

function ChatMessage({ message }: { message: UIMessage }) {
  const isUser = message.role === 'user'
  return (
    <div className={cn('flex gap-3', isUser && 'flex-row-reverse')}>
      <div
        className={cn(
          'w-8 h-8 rounded-xl flex items-center justify-center flex-shrink-0',
          isUser ? 'bg-surface-hover' : 'bg-brand-500/10'
        )}
      >
        {isUser ? <User size={16} className="text-slate-300" /> : <Bot size={16} className="text-brand-400" />}
      </div>
      <div className={cn('space-y-2 max-w-[80%]', isUser && 'items-end flex flex-col')}>
        {message.parts.map((part, i) => {
          if (part.type === 'text') {
            return (
              <div
                key={i}
                className={cn(
                  'rounded-2xl px-4 py-2.5 text-sm whitespace-pre-wrap leading-relaxed',
                  isUser ? 'bg-brand-500 text-white' : 'bg-surface-card border border-surface-border text-slate-200'
                )}
              >
                {part.text}
              </div>
            )
          }
          if (part.type === 'dynamic-tool') {
            const done = part.state === 'output-available'
            const failed = part.state === 'output-error'
            return (
              <div key={i} className="flex items-center gap-1.5 text-xs text-slate-500">
                {failed ? (
                  <XCircle size={13} className="text-red-400" />
                ) : done ? (
                  <CheckCircle2 size={13} className="text-brand-400" />
                ) : (
                  <Wrench size={13} className="animate-pulse" />
                )}
                {toolLabel(part.toolName, done || failed)}
              </div>
            )
          }
          return null
        })}
      </div>
    </div>
  )
}
