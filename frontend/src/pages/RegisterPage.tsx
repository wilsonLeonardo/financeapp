import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { TrendingUp } from 'lucide-react'
import { authService } from '@/services'
import { useAuthStore } from '@/store/authStore'
import { Alert, Spinner } from '@/components/common'
import { getErrorMessage } from '@/utils'

const schema = z.object({
  name: z.string().min(2, 'Nome deve ter pelo menos 2 caracteres'),
  email: z.string().email('Email inválido'),
  password: z.string().min(8, 'Senha deve ter pelo menos 8 caracteres'),
  confirmPassword: z.string(),
}).refine((d) => d.password === d.confirmPassword, {
  message: 'Senhas não conferem',
  path: ['confirmPassword'],
})
type FormValues = z.infer<typeof schema>

export default function RegisterPage() {
  const navigate = useNavigate()
  const setAuth = useAuthStore((s) => s.setAuth)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  const { register, handleSubmit, formState: { errors } } = useForm<FormValues>({
    resolver: zodResolver(schema),
  })

  async function onSubmit(values: FormValues) {
    setError(null)
    setLoading(true)
    try {
      const resp = await authService.register({
        name: values.name,
        email: values.email,
        password: values.password,
      })
      setAuth(resp.user, resp.token)
      navigate('/')
    } catch (err) {
      setError(getErrorMessage(err))
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center p-4">
      <div className="w-full max-w-sm">
        <div className="flex items-center gap-3 mb-10 justify-center">
          <div className="w-10 h-10 rounded-xl bg-brand-500 flex items-center justify-center">
            <TrendingUp size={20} className="text-white" />
          </div>
          <span className="text-2xl font-bold text-white">FinanceApp</span>
        </div>

        <div className="card">
          <h1 className="text-xl font-semibold text-white mb-1">Criar conta</h1>
          <p className="text-sm text-slate-500 mb-6">Comece a controlar suas finanças hoje</p>

          {error && <div className="mb-4"><Alert message={error} onClose={() => setError(null)} /></div>}

          <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
            <div>
              <label className="label">Nome</label>
              <input {...register('name')} className="input" placeholder="Seu nome" />
              {errors.name && <p className="text-xs text-red-400 mt-1">{errors.name.message}</p>}
            </div>
            <div>
              <label className="label">Email</label>
              <input {...register('email')} type="email" className="input" placeholder="voce@email.com" />
              {errors.email && <p className="text-xs text-red-400 mt-1">{errors.email.message}</p>}
            </div>
            <div>
              <label className="label">Senha</label>
              <input {...register('password')} type="password" className="input" placeholder="Mín. 8 caracteres" />
              {errors.password && <p className="text-xs text-red-400 mt-1">{errors.password.message}</p>}
            </div>
            <div>
              <label className="label">Confirmar senha</label>
              <input {...register('confirmPassword')} type="password" className="input" placeholder="Repita a senha" />
              {errors.confirmPassword && <p className="text-xs text-red-400 mt-1">{errors.confirmPassword.message}</p>}
            </div>

            <button type="submit" disabled={loading} className="btn-primary w-full flex items-center justify-center gap-2 py-3">
              {loading && <Spinner className="w-4 h-4 text-white" />}
              Criar conta
            </button>
          </form>

          <p className="text-sm text-center text-slate-500 mt-6">
            Já tem conta?{' '}
            <Link to="/login" className="text-brand-400 hover:text-brand-300 font-medium">
              Entrar
            </Link>
          </p>
        </div>
      </div>
    </div>
  )
}
