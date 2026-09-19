import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { categoryService, expenseService, importService, reportService } from '@/services'
import type { CreateExpenseDTO, ListExpensesParams } from '@/types'

// Expenses
export const EXPENSE_KEYS = {
  all: ['expenses'] as const,
  list: (params: ListExpensesParams) => [...EXPENSE_KEYS.all, 'list', params] as const,
  detail: (id: string) => [...EXPENSE_KEYS.all, id] as const,
}

export function useExpenses(params: ListExpensesParams = {}) {
  return useQuery({
    queryKey: EXPENSE_KEYS.list(params),
    queryFn: () => expenseService.list(params),
  })
}

export function useCreateExpense() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: CreateExpenseDTO) => expenseService.create(data),
    onSuccess: () => qc.invalidateQueries({ queryKey: EXPENSE_KEYS.all }),
  })
}

export function useUpdateExpense(id: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: Partial<CreateExpenseDTO>) => expenseService.update(id, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: EXPENSE_KEYS.all }),
  })
}

export function useDeleteExpense() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => expenseService.delete(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: EXPENSE_KEYS.all }),
  })
}

// Categories
export const CATEGORY_KEYS = { all: ['categories'] as const }

export function useCategories() {
  return useQuery({ queryKey: CATEGORY_KEYS.all, queryFn: categoryService.list })
}

export function useCreateCategory() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: categoryService.create,
    onSuccess: () => qc.invalidateQueries({ queryKey: CATEGORY_KEYS.all }),
  })
}

export function useDeleteCategory() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: categoryService.delete,
    onSuccess: () => qc.invalidateQueries({ queryKey: CATEGORY_KEYS.all }),
  })
}

// Reports
export const REPORT_KEYS = {
  monthly: (months: number) => ['reports', 'monthly', months] as const,
  categories: (start?: string, end?: string) => ['reports', 'categories', start, end] as const,
}

export function useMonthlyReport(months = 12) {
  return useQuery({ queryKey: REPORT_KEYS.monthly(months), queryFn: () => reportService.monthly(months) })
}

export function useCategoryReport(start?: string, end?: string) {
  return useQuery({ queryKey: REPORT_KEYS.categories(start, end), queryFn: () => reportService.categories(start, end) })
}

// Imports
export const IMPORT_KEYS = { all: ['imports'] as const }

export function useImports() {
  return useQuery({ queryKey: IMPORT_KEYS.all, queryFn: importService.list })
}

export function useUploadImport() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (file: File) => importService.upload(file),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: IMPORT_KEYS.all })
      qc.invalidateQueries({ queryKey: EXPENSE_KEYS.all })
    },
  })
}

export function useRevertImport() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => importService.revert(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: IMPORT_KEYS.all })
      qc.invalidateQueries({ queryKey: EXPENSE_KEYS.all })
    },
  })
}