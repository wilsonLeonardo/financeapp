import axios from 'axios'
import type { ApplyResult, SuggestResult } from '@/types'

export const AI_URL = import.meta.env.VITE_AI_URL ?? 'http://localhost:8000'

// Same JWT as the Go API: the ai-service verifies it and calls the API on the user's behalf.
export function authHeaders(): Record<string, string> {
  const token = localStorage.getItem('token')
  return token ? { Authorization: `Bearer ${token}` } : {}
}

export const aiApi = axios.create({
  baseURL: AI_URL,
  headers: { 'Content-Type': 'application/json' },
})

aiApi.interceptors.request.use((config) => {
  Object.assign(config.headers, authHeaders())
  return config
})

export const aiService = {
  suggest: (params: { limit?: number }) =>
    aiApi.post<SuggestResult>('/categorize/suggest', params).then((r) => r.data),

  apply: (items: { expense_id: string; category_id: string }[]) =>
    aiApi.post<ApplyResult>('/categorize/apply', { items }).then((r) => r.data),
}
