import { useState } from 'react'
import { format, startOfMonth, endOfMonth, subMonths, isSameMonth } from 'date-fns'
import { ptBR } from 'date-fns/locale'
import { ChevronLeft, ChevronRight, TrendingUp, TrendingDown, DollarSign, BarChart3 } from 'lucide-react'
import { StatCard } from '@/components/common'
import { MonthlyAreaChart, CategoryPieChart, MonthlyBarChart } from '@/components/charts/Charts'
import { useMonthlyReport, useCategoryReport, useExpenses } from '@/hooks/useFinance'
import { categoryLabel, formatCurrency } from '@/utils'
import { useAuthStore } from '@/store/authStore'

export default function DashboardPage() {
  const user = useAuthStore((s) => s.user)
  const [selectedDate, setSelectedDate] = useState(new Date())

  const start = format(startOfMonth(selectedDate), 'yyyy-MM-dd')
  const end   = format(endOfMonth(selectedDate), 'yyyy-MM-dd')
  const monthLabel = format(selectedDate, "MMMM 'de' yyyy", { locale: ptBR })
  const isCurrentMonth = isSameMonth(selectedDate, new Date())

  function prevMonth() { setSelectedDate((d) => subMonths(d, 1)) }
  function nextMonth() { setSelectedDate((d) => subMonths(d, -1)) }

  const { data: monthly, isLoading: monthlyLoading } = useMonthlyReport(12)
  const { data: categoryData, isLoading: categoryLoading } = useCategoryReport(start, end)
  const { data: periodExpenses } = useExpenses({ page: 1, page_size: 1, start_date: start, end_date: end })

  // Find stats for the selected month from the monthly report
  const selectedMonthKey = format(selectedDate, 'yyyy-MM')
  const selectedMonthData = monthly?.find((m) => m.month === selectedMonthKey)
  const prevMonthKey = format(subMonths(selectedDate, 1), 'yyyy-MM')
  const prevMonthData = monthly?.find((m) => m.month === prevMonthKey)

  const totalSpent  = selectedMonthData?.total_spent  ?? 0
  const totalEarned = selectedMonthData?.total_earned ?? 0
  const balance     = totalEarned - totalSpent

  function trendLabel(current: number, prev?: number) {
    if (!prev || prev === 0) return undefined
    const diff = ((current - prev) / prev) * 100
    return `${diff > 0 ? '+' : ''}${diff.toFixed(1)}% vs mês anterior`
  }

  return (
    <div className="space-y-6 max-w-7xl">
      {/* Header with month selector */}
      <div className="flex items-start justify-between gap-4 flex-wrap">
        <div>
          <h1 className="text-2xl font-bold text-white">
            Olá, {user?.name?.split(' ')[0]} 👋
          </h1>
          <p className="text-slate-500 text-sm mt-1">Resumo das suas finanças</p>
        </div>

        {/* Month selector */}
        <div className="flex items-center gap-1 bg-surface-card border border-surface-border rounded-xl px-2 py-1.5">
          <button
            onClick={prevMonth}
            className="btn-ghost p-1.5"
            title="Mês anterior"
          >
            <ChevronLeft size={15} />
          </button>
          <span className="text-sm font-medium text-white capitalize w-40 text-center select-none">
            {monthLabel}
          </span>
          <button
            onClick={nextMonth}
            disabled={isCurrentMonth}
            className="btn-ghost p-1.5 disabled:opacity-30"
            title="Próximo mês"
          >
            <ChevronRight size={15} />
          </button>
        </div>
      </div>

      {/* KPI Cards */}
      <div className="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-4 gap-4">
        <StatCard
          label="Receitas do período"
          value={formatCurrency(totalEarned)}
          sub={trendLabel(totalEarned, prevMonthData?.total_earned)}
          icon={<TrendingUp size={18} />}
          trend="up"
        />
        <StatCard
          label="Despesas do período"
          value={formatCurrency(totalSpent)}
          sub={trendLabel(totalSpent, prevMonthData?.total_spent)}
          icon={<TrendingDown color="red" size={18} />}
          trend="down"
        />
        <StatCard
          label="Saldo do período"
          value={formatCurrency(balance)}
          icon={<DollarSign size={18} />}
          trend={balance >= 0 ? 'up' : 'down'}
        />
        <StatCard
          label="Transações no período"
          value={String(periodExpenses?.total ?? 0)}
          icon={<BarChart3 size={18} />}
          trend="neutral"
        />
      </div>

      {/* Charts row */}
      <div className="grid grid-cols-1 xl:grid-cols-3 gap-4">
        <div className="xl:col-span-2 card">
          <h2 className="font-semibold text-white mb-4">Evolução mensal (12 meses)</h2>
          <MonthlyAreaChart data={monthly} isLoading={monthlyLoading} />
        </div>
        <div className="card">
          <h2 className="font-semibold text-white mb-4 capitalize">
            Categorias — {format(selectedDate, 'MMM yyyy', { locale: ptBR })}
          </h2>
          <CategoryPieChart data={categoryData} isLoading={categoryLoading} />
        </div>
      </div>

      {/* Bar chart */}
      <div className="card">
        <h2 className="font-semibold text-white mb-4">Comparativo mensal</h2>
        <MonthlyBarChart data={monthly} isLoading={monthlyLoading} />
      </div>

      {/* Category breakdown */}
      {!!categoryData?.length && (
        <div className="card">
          <h2 className="font-semibold text-white mb-4 capitalize">
            Gastos por categoria — {format(selectedDate, "MMMM 'de' yyyy", { locale: ptBR })}
          </h2>
          <div className="space-y-3">
            {categoryData.slice(0, 8).map((cat, idx) => {
              const maxTotal = categoryData[0]?.total ?? 1
              const pct = (cat.total / maxTotal) * 100
              const colors = ['#22c55e','#3b82f6','#f59e0b','#ef4444','#8b5cf6','#06b6d4','#ec4899','#f97316']
              return (
                <div key={idx} className="flex items-center gap-3">
                  <span className="text-sm text-slate-400 w-32 truncate flex-shrink-0">
                    {categoryLabel(cat)}
                  </span>
                  <div className="flex-1 bg-surface rounded-full h-2 overflow-hidden">
                    <div
                      className="h-full rounded-full transition-all duration-500"
                      style={{ width: `${pct}%`, backgroundColor: colors[idx % colors.length] }}
                    />
                  </div>
                  <span className="text-sm font-mono text-slate-300 w-24 text-right flex-shrink-0">
                    {formatCurrency(cat.total)}
                  </span>
                </div>
              )
            })}
          </div>
        </div>
      )}
    </div>
  )
}