import { lazy, Suspense } from 'react'
import { Navigate, Outlet, Route, Routes } from 'react-router'
import { Layout, Message, NotFound } from './components/Layout.tsx'
import { Home } from './pages/Home.tsx'

// 상세 화면은 차트(ECharts)가 커서 따로 읽는다. 메인 화면은 차트 없이 먼저 뜬다.
const IndicatorPage = lazy(() => import('./pages/IndicatorPage.tsx').then((m) => ({ default: m.IndicatorPage })))
// 관리자 화면은 사용자 화면과 따로 읽는다.
const AdminLayout = lazy(() => import('./admin/AdminLayout.tsx').then((m) => ({ default: m.AdminLayout })))
const LoginPage = lazy(() => import('./admin/LoginPage.tsx').then((m) => ({ default: m.LoginPage })))
const DashboardPage = lazy(() => import('./admin/DashboardPage.tsx').then((m) => ({ default: m.DashboardPage })))
const ApiPage = lazy(() => import('./admin/ApiPage.tsx').then((m) => ({ default: m.ApiPage })))

// 사이트 구조(50.web 7장). 사용자 화면에는 관리자 링크를 두지 않는다.
export function App() {
  return (
    <Suspense fallback={<Message>불러오는 중…</Message>}>
      <Routes>
        <Route path="/admin">
          <Route index element={<Navigate to="/admin/dashboard" replace />} />
          <Route path="login" element={<LoginPage />} />
          <Route element={<AdminLayout />}>
            <Route path="dashboard" element={<DashboardPage />} />
            <Route path="api" element={<ApiPage />} />
          </Route>
          <Route path="*" element={<Navigate to="/admin/dashboard" replace />} />
        </Route>
        <Route
          element={
            <Layout>
              <Suspense fallback={<Message>불러오는 중…</Message>}>
                <Outlet />
              </Suspense>
            </Layout>
          }
        >
          <Route path="/" element={<Home />} />
          <Route path="/indicator/:id" element={<IndicatorPage />} />
          <Route path="*" element={<NotFound />} />
        </Route>
      </Routes>
    </Suspense>
  )
}
