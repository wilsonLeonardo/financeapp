import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'
import { useAuthStore } from '@/store/authStore'
import Layout from '@/components/layout/Layout'
import LoginPage from '@/pages/LoginPage'
import RegisterPage from '@/pages/RegisterPage'
import DashboardPage from '@/pages/DashboardPage'
import ExpensesPage from '@/pages/ExpensesPage'
import ImportPage from '@/pages/ImportPage'
import CategoriesPage from '@/pages/CategoriesPage'

function RequireAuth({ children }: { children: React.ReactNode }) {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated)
  return isAuthenticated ? <>{children}</> : <Navigate to="/login" replace />
}

function GuestOnly({ children }: { children: React.ReactNode }) {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated)
  return isAuthenticated ? <Navigate to="/" replace /> : <>{children}</>
}

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<GuestOnly><LoginPage /></GuestOnly>} />
        <Route path="/register" element={<GuestOnly><RegisterPage /></GuestOnly>} />
        <Route path="/" element={<RequireAuth><Layout /></RequireAuth>}>
          <Route index element={<DashboardPage />} />
          <Route path="expenses" element={<ExpensesPage />} />
          <Route path="import" element={<ImportPage />} />
          <Route path="categories" element={<CategoriesPage />} />
        </Route>
      </Routes>
    </BrowserRouter>
  )
}
