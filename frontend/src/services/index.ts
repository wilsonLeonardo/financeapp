import type {
  AuthResponse,
  Category,
  CategorySummary,
  CreateExpenseDTO,
  Expense,
  Import,
  ListExpensesParams,
  MonthlySummary,
  PaginatedResponse,
} from '@/types'
import { api } from './api'

// ── Auth ──────────────────────────────────────────────────────────────────────

export const authService = {
  register: (data: { name: string; email: string; password: string }) =>
    api.post<AuthResponse>('/auth/register', data).then((r) => r.data),

  login: (data: { email: string; password: string }) =>
    api.post<AuthResponse>('/auth/login', data).then((r) => r.data),

  logout: () => api.post('/auth/logout').then(() => localStorage.removeItem('token')),
}

// ── Expenses ──────────────────────────────────────────────────────────────────

export const expenseService = {
  create: (data: CreateExpenseDTO) =>
    api.post<Expense>('/expenses', data).then((r) => r.data),

  list: (params: ListExpensesParams = {}) =>
    api.get<PaginatedResponse<Expense>>('/expenses', { params }).then((r) => r.data),

  getById: (id: string) =>
    api.get<Expense>(`/expenses/${id}`).then((r) => r.data),

  update: (id: string, data: Partial<CreateExpenseDTO>) =>
    api.put<Expense>(`/expenses/${id}`, data).then((r) => r.data),

  delete: (id: string) => api.delete(`/expenses/${id}`),
}

// ── Categories ────────────────────────────────────────────────────────────────

export const categoryService = {
  list: () => api.get<Category[]>('/categories').then((r) => r.data),

  create: (data: { name: string; icon?: string; color?: string }) =>
    api.post<Category>('/categories', data).then((r) => r.data),

  update: (id: string, data: { name: string; icon?: string; color?: string }) =>
    api.put<Category>(`/categories/${id}`, data).then((r) => r.data),

  delete: (id: string) => api.delete(`/categories/${id}`),
}

// ── Reports ───────────────────────────────────────────────────────────────────

export const reportService = {
  monthly: (months = 12) =>
    api.get<MonthlySummary[]>('/reports/monthly', { params: { months } }).then((r) => r.data),

  categories: (start?: string, end?: string) =>
    api
      .get<CategorySummary[]>('/reports/categories', { params: { start_date: start, end_date: end } })
      .then((r) => r.data),
}

// ── Imports ───────────────────────────────────────────────────────────────────

export const importService = {
  upload: (file: File) => {
    const form = new FormData()
    form.append('file', file)
    return api.post<Import>('/imports', form, { headers: { 'Content-Type': 'multipart/form-data' } }).then((r) => r.data)
  },
  list: () => api.get<Import[]>('/imports').then((r) => r.data),
  revert: (id: string) => api.delete(`/imports/${id}`),
}