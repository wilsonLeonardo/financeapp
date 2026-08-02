import { Outlet, NavLink } from 'react-router-dom'
import {
  LayoutDashboard, Receipt, Upload, Tag, LogOut, Menu, X, TrendingUp,
} from 'lucide-react'
import { useAuthStore } from '@/store/authStore'
import { useUIStore } from '@/store/uiStore'
import { authService } from '@/services'
import { cn } from '@/utils'

const NAV = [
  { to: '/', icon: LayoutDashboard, label: 'Dashboard', end: true },
  { to: '/expenses', icon: Receipt, label: 'Transações' },
  { to: '/import', icon: Upload, label: 'Importar' },
  { to: '/categories', icon: Tag, label: 'Categorias' },
]

export default function Layout() {
  const { user, clearAuth } = useAuthStore()
  const { sidebarOpen, toggleSidebar } = useUIStore()

  async function handleLogout() {
    await authService.logout()
    clearAuth()
  }

  return (
    <div className="flex h-screen overflow-hidden">
      {/* Sidebar */}
      <aside
        className={cn(
          'flex flex-col bg-surface-card border-r border-surface-border transition-all duration-300 z-20',
          sidebarOpen ? 'w-60' : 'w-16'
        )}
      >
        {/* Logo */}
        <div className="flex items-center gap-3 px-4 h-16 border-b border-surface-border flex-shrink-0">
          <div className="w-8 h-8 rounded-lg bg-brand-500 flex items-center justify-center flex-shrink-0">
            <TrendingUp size={16} className="text-white" />
          </div>
          {sidebarOpen && (
            <span className="font-bold text-white tracking-tight text-lg">FinanceApp</span>
          )}
        </div>

        {/* Nav */}
        <nav className="flex-1 py-4 px-2 space-y-1">
          {NAV.map(({ to, icon: Icon, label, end }) => (
            <NavLink
              key={to}
              to={to}
              end={end}
              className={({ isActive }) =>
                cn(
                  'flex items-center gap-3 px-3 py-2.5 rounded-xl transition-all duration-200 group',
                  isActive
                    ? 'bg-brand-500/10 text-brand-400'
                    : 'text-slate-400 hover:bg-surface-hover hover:text-slate-200'
                )
              }
            >
              <Icon size={18} className="flex-shrink-0" />
              {sidebarOpen && <span className="text-sm font-medium">{label}</span>}
            </NavLink>
          ))}
        </nav>

        {/* User */}
        <div className="px-2 pb-4 border-t border-surface-border pt-4">
          {sidebarOpen && (
            <div className="px-3 py-2 mb-2">
              <p className="text-sm font-medium text-white truncate">{user?.name}</p>
              <p className="text-xs text-slate-500 truncate">{user?.email}</p>
            </div>
          )}
          <button
            onClick={handleLogout}
            className="flex items-center gap-3 px-3 py-2.5 rounded-xl text-slate-400
                       hover:bg-red-500/10 hover:text-red-400 transition-all duration-200 w-full"
          >
            <LogOut size={18} className="flex-shrink-0" />
            {sidebarOpen && <span className="text-sm font-medium">Sair</span>}
          </button>
        </div>
      </aside>

      {/* Main */}
      <div className="flex-1 flex flex-col overflow-hidden">
        {/* Top bar */}
        <header className="h-16 flex items-center gap-4 px-6 border-b border-surface-border bg-surface-card flex-shrink-0">
          <button onClick={toggleSidebar} className="btn-ghost p-2">
            {sidebarOpen ? <X size={18} /> : <Menu size={18} />}
          </button>
        </header>

        {/* Page content */}
        <main className="flex-1 overflow-y-auto p-6 animate-fade-in">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
