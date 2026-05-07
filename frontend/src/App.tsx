import { lazy, Suspense } from 'react'
import { Routes, Route, Navigate, Outlet } from 'react-router-dom'
import { getToken } from '@/api/client'
import Layout from '@/components/Layout'
import Login from '@/pages/Login'
import Dashboard from '@/pages/Dashboard'

// Off the critical path — each becomes a separate JS chunk.
const Chemistry = lazy(() => import('@/pages/Chemistry'))
const MaintenancePage = lazy(() => import('@/pages/MaintenancePage'))
const HistoryPage = lazy(() => import('@/pages/HistoryPage'))
const Livestock = lazy(() => import('@/pages/Livestock'))
const LivestockDetail = lazy(() => import('@/pages/LivestockDetail'))
const Alerts = lazy(() => import('@/pages/Alerts'))
const Settings = lazy(() => import('@/pages/Settings'))

function RequireAuth({ children }: { children: React.ReactNode }) {
  if (!getToken()) {
    return <Navigate to="/login" replace />
  }
  return <>{children}</>
}

function PageFallback() {
  return <div className="flex-1" />
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
        {/* Eager — on the critical path */}
        <Route index element={<Dashboard />} />

        {/* Lazy — a single Suspense boundary covers all split routes */}
        <Route element={<Suspense fallback={<PageFallback />}><Outlet /></Suspense>}>
          <Route path="chemistry" element={<Chemistry />} />
          <Route path="maintenance" element={<MaintenancePage />} />
          <Route path="history" element={<HistoryPage />} />
          <Route path="livestock" element={<Livestock />} />
          <Route path="livestock/:id" element={<LivestockDetail />} />
          <Route path="alerts" element={<Alerts />} />
          <Route path="settings" element={<Settings />} />
        </Route>

        {/* Legacy redirects — no component, no Suspense needed */}
        <Route path="measurements" element={<Navigate to="/chemistry" replace />} />
        <Route path="journal" element={<Navigate to="/history?tab=journal" replace />} />
        <Route path="control" element={<Navigate to="/maintenance?tab=control" replace />} />
      </Route>
    </Routes>
  )
}
