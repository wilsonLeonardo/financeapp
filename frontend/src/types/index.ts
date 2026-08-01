export type TransactionType = 'expense' | 'income'

export interface User {
  id: string
  name: string
  email: string
  created_at: string
}

export interface AuthResponse {
  token: string
  user: User
}

export interface Category {
  id: string
  user_id: string
  name: string
  icon: string
  color: string
  created_at: string
}

export interface Expense {
  id: string
  user_id: string
  category_id?: string
  category?: Category
  amount: string
  type: TransactionType
  description: string
  date: string
  tags?: string
  import_id?: string
  created_at: string
  updated_at: string
}

export interface PaginatedResponse<T> {
  data: T[]
  total: number
  page: number
  page_size: number
}

export interface MonthlySummary {
  month: string
  total_spent: number
  total_earned: number
}

export interface CategorySummary {
  category_id?: string
  category_name: string
  total: number
  count: number
}

export interface Import {
  id: string
  file_name: string
  file_type: string
  status: 'pending' | 'processing' | 'completed' | 'failed'
  total_rows: number
  imported: number
  errors: number
  error_log?: string
  imported_at?: string
  created_at: string
}

export interface CreateExpenseDTO {
  category_id?: string
  amount: number
  type: TransactionType
  description: string
  date: string
  tags?: string
}

export interface ListExpensesParams {
  category_id?: string
  type?: TransactionType
  start_date?: string
  end_date?: string
  page?: number
  page_size?: number
}
