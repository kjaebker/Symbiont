import { Routes, Route, Navigate } from 'react-router-dom'
import { getToken } from '@/api/client'
import Layout from '@/components/Layout'
import Login from '@/pages/Login'
import Dashboard from '@/pages/Dashboard'
import Chemistry from '@/pages/Chemistry'
import MaintenancePage from '@/pages/MaintenancePage'
import HistoryPage from '@/pages/HistoryPage'
import Livestock from '@/pages/Livestock'
import LivestockDetail from '@/pages/LivestockDetail'
import Alerts from '@/pages/Alerts'
import Settings from '@/pages/Settings'

function RequireAuth({ children }: { children: React.ReactNode }) {
  if (!getToken()) {
    return <Navigate to="/login" replace />
  }
  return <>{children}</>
}

export default function App() {
  return (
    <Routes>
      <Route path="/login" element={<Login />} />
      <Route
        element={
          <RequireAuth>
            <Layout />
          </RequireAuth>
        }
      >
        <Route index element={<Dashboard />} />
        <Route path="chemistry" element={<Chemistry />} />
        <Route path="maintenance" element={<MaintenancePage />} />
        <Route path="history" element={<HistoryPage />} />
        <Route path="livestock" element={<Livestock />} />
        <Route path="livestock/:id" element={<LivestockDetail />} />
        <Route path="alerts" element={<Alerts />} />
        <Route path="settings" element={<Settings />} />
        {/* Legacy redirects */}
        <Route path="measurements" element={<Navigate to="/chemistry" replace />} />
        <Route path="journal" element={<Navigate to="/history?tab=journal" replace />} />
        <Route path="control" element={<Navigate to="/maintenance?tab=control" replace />} />
      </Route>
    </Routes>
  )
}
