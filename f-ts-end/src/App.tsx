import { lazy, Suspense } from 'react'
import { Route, Routes } from 'react-router'
import { Layout, Message, NotFound } from './components/Layout.tsx'
import { Home } from './pages/Home.tsx'

// 상세 화면은 차트(ECharts)가 커서 따로 읽는다. 메인 화면은 차트 없이 먼저 뜬다.
const IndicatorPage = lazy(() => import('./pages/IndicatorPage.tsx').then((m) => ({ default: m.IndicatorPage })))

// 사용자 화면 두 개(50.web 7장). 관리자 화면은 다음 Issue에서 더한다.
export function App() {
  return (
    <Layout>
      <Suspense fallback={<Message>불러오는 중…</Message>}>
        <Routes>
          <Route path="/" element={<Home />} />
          <Route path="/indicator/:id" element={<IndicatorPage />} />
          <Route path="*" element={<NotFound />} />
        </Routes>
      </Suspense>
    </Layout>
  )
}
