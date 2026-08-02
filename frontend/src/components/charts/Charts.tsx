import {
  AreaChart, Area, BarChart, Bar, PieChart, Pie, Cell,
  XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Legend,
} from 'recharts'
import { categoryLabel, formatCurrency, formatMonth } from '@/utils'
import type { CategorySummary, MonthlySummary } from '@/types'
import { PageLoader } from '@/components/common'

const COLORS = [
  '#22c55e', '#3b82f6', '#f59e0b', '#ef4444', '#8b5cf6',
  '#06b6d4', '#ec4899', '#f97316', '#84cc16', '#6366f1',
]

const tooltipStyle = {
  backgroundColor: '#161b27',
  border: '1px solid #1e2738',
  borderRadius: '12px',
  color: '#e2e8f0',
  fontSize: '12px',
}

// Recharts colors each tooltip row with the series color and falls back to black
// (`entry.color || '#000'`) when the series has none. Pie slices ship no color in
// their tooltip payload, so the text rendered black on the dark card. These two
// styles pin the row and label colors instead of relying on that fallback.
const tooltipItemStyle = { color: '#e2e8f0' }
const tooltipLabelStyle = { color: '#94a3b8', marginBottom: '4px' }

// ── Monthly Area Chart ────────────────────────────────────────────────────────
interface MonthlyChartProps { data?: MonthlySummary[]; isLoading: boolean }

export function MonthlyAreaChart({ data, isLoading }: MonthlyChartProps) {
  if (isLoading) return <PageLoader />

  const chartData = (data ?? []).map((d) => ({
    month: formatMonth(d.month),
    Despesas: Number(d.total_spent),
    Receitas: Number(d.total_earned),
  }))

  return (
    <ResponsiveContainer width="100%" height={240}>
      <AreaChart data={chartData} margin={{ top: 5, right: 5, left: 0, bottom: 5 }}>
        <defs>
          <linearGradient id="colorExpense" x1="0" y1="0" x2="0" y2="1">
            <stop offset="5%" stopColor="#ef4444" stopOpacity={0.15} />
            <stop offset="95%" stopColor="#ef4444" stopOpacity={0} />
          </linearGradient>
          <linearGradient id="colorIncome" x1="0" y1="0" x2="0" y2="1">
            <stop offset="5%" stopColor="#22c55e" stopOpacity={0.15} />
            <stop offset="95%" stopColor="#22c55e" stopOpacity={0} />
          </linearGradient>
        </defs>
        <CartesianGrid strokeDasharray="3 3" stroke="#1e2738" vertical={false} />
        <XAxis dataKey="month" tick={{ fill: '#64748b', fontSize: 11 }} axisLine={false} tickLine={false} />
        <YAxis tickFormatter={(v) => `$${(v / 1000).toFixed(0)}k`} tick={{ fill: '#64748b', fontSize: 11 }} axisLine={false} tickLine={false} />
        <Tooltip contentStyle={tooltipStyle} itemStyle={tooltipItemStyle} labelStyle={tooltipLabelStyle} formatter={(v: number) => formatCurrency(v)} />
        <Legend wrapperStyle={{ fontSize: '12px', color: '#94a3b8' }} />
        <Area type="monotone" dataKey="Despesas" stroke="#ef4444" strokeWidth={2} fill="url(#colorExpense)" dot={false} />
        <Area type="monotone" dataKey="Receitas" stroke="#22c55e" strokeWidth={2} fill="url(#colorIncome)" dot={false} />
      </AreaChart>
    </ResponsiveContainer>
  )
}

// ── Category Pie Chart ────────────────────────────────────────────────────────
interface CategoryChartProps { data?: CategorySummary[]; isLoading: boolean }

export function CategoryPieChart({ data, isLoading }: CategoryChartProps) {
  if (isLoading) return <PageLoader />
  if (!data?.length) return <p className="text-slate-500 text-sm text-center py-10">Sem dados para exibir</p>

  const chartData = data.map((row) => ({ ...row, category_name: categoryLabel(row) }))

  return (
    <ResponsiveContainer width="100%" height={240}>
      <PieChart>
        <Pie
          data={chartData}
          dataKey="total"
          nameKey="category_name"
          cx="50%"
          cy="50%"
          outerRadius={90}
          innerRadius={55}
          paddingAngle={3}
        >
          {chartData.map((_, idx) => (
            <Cell key={idx} fill={COLORS[idx % COLORS.length]} />
          ))}
        </Pie>
        <Tooltip contentStyle={tooltipStyle} itemStyle={tooltipItemStyle} labelStyle={tooltipLabelStyle} formatter={(v: number) => formatCurrency(v)} />
        <Legend
          formatter={(value) => <span style={{ color: '#94a3b8', fontSize: 12 }}>{value}</span>}
          iconType="circle"
          iconSize={8}
        />
      </PieChart>
    </ResponsiveContainer>
  )
}

// ── Monthly Bar Chart ─────────────────────────────────────────────────────────
export function MonthlyBarChart({ data, isLoading }: MonthlyChartProps) {
  if (isLoading) return <PageLoader />

  const chartData = (data ?? []).map((d) => ({
    month: formatMonth(d.month),
    Despesas: Number(d.total_spent),
    Receitas: Number(d.total_earned),
    Saldo: Number(d.total_earned) - Number(d.total_spent),
  }))

  return (
    <ResponsiveContainer width="100%" height={240}>
      <BarChart data={chartData} margin={{ top: 5, right: 5, left: 0, bottom: 5 }} barGap={4}>
        <CartesianGrid strokeDasharray="3 3" stroke="#1e2738" vertical={false} />
        <XAxis dataKey="month" tick={{ fill: '#64748b', fontSize: 11 }} axisLine={false} tickLine={false} />
        <YAxis tickFormatter={(v) => `$${(v / 1000).toFixed(0)}k`} tick={{ fill: '#64748b', fontSize: 11 }} axisLine={false} tickLine={false} />
        <Tooltip contentStyle={tooltipStyle} itemStyle={tooltipItemStyle} labelStyle={tooltipLabelStyle} formatter={(v: number) => formatCurrency(v)} />
        <Legend wrapperStyle={{ fontSize: '12px', color: '#94a3b8' }} />
        <Bar dataKey="Despesas" fill="#ef4444" radius={[4, 4, 0, 0]} />
        <Bar dataKey="Receitas" fill="#22c55e" radius={[4, 4, 0, 0]} />
        <Bar dataKey="Saldo" fill="#3b82f6" radius={[4, 4, 0, 0]} />
      </BarChart>
    </ResponsiveContainer>
  )
}
