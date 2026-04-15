import { Routes, Route, Navigate } from 'react-router-dom'
import { getToken } from '@/api/client'
import Layout from '@/components/Layout'
import Login from '@/pages/Login'
import Dashboard from '@/pages/Dashboard'
import History from '@/pages/History'
import Journal from '@/pages/Journal'
import Measurements from '@/pages/Measurements'
import Livestock from '@/pages/Livestock'
import LivestockDetail from '@/pages/LivestockDetail'
import Control from '@/pages/Control'
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
        <Route path="history" element={<History />} />
        <Route path="measurements" element={<Measurements />} />
        <Route path="journal" element={<Journal />} />
        <Route path="livestock" element={<Livestock />} />
        <Route path="livestock/:id" element={<LivestockDetail />} />
        <Route path="control" element={<Control />} />
        <Route path="alerts" element={<Alerts />} />
        <Route path="settings" element={<Settings />} />
      </Route>
    </Routes>
  )
}
