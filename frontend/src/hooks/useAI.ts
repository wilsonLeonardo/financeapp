import { useMutation, useQueryClient } from '@tanstack/react-query'
import { aiService } from '@/services/ai'
import { EXPENSE_KEYS } from '@/hooks/useFinance'

export function useSuggestCategories() {
  return useMutation({
    mutationFn: (params: { limit?: number }) => aiService.suggest(params),
  })
}

export function useApplyCategories() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: aiService.apply,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: EXPENSE_KEYS.all })
      qc.invalidateQueries({ queryKey: ['reports'] })
    },
  })
}
