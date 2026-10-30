import { useCallback, useEffect, useMemo, useState } from 'react'
import { Navigate, NavLink, Outlet, useNavigate } from 'react-router'
import { Message } from '../components/Layout.tsx'
import { adminRequest, authFailure, checkSession, logoutPath, type SessionCheck } from './api.ts'
import { AdminAuthContext, type AdminAuth } from './auth.ts'
import { buttonClass, Forbidden } from './ui.tsx'

/**
 * 로그인해야 들어오는 관리자 화면(/admin/dashboard, /admin/api)의 틀.
 * 들어올 때 GET /api/admin/session으로 확인한다. 401이면 로그인 화면으로, 403이면 "허용된 주소가 아닙니다"만 보인다.
 */
export function AdminLayout() {
  const navigate = useNavigate()
  const [check, setCheck] = useState<SessionCheck | null>(null)

  useEffect(() => {
    const ctrl = new AbortController()
    checkSession(ctrl.signal).then((c) => {
      if (!ctrl.signal.aborted) setCheck(c)
    })
    return () => ctrl.abort()
  }, [])

  const handleAuthError = useCallback(
    (err: unknown) => {
      const f = authFailure(err)
      if (f === 'login') navigate('/admin/login', { replace: true })
      if (f === 'forbidden') setCheck({ kind: 'forbidden' })
      return f !== null
    },
    [navigate],
  )

  const session = check?.kind === 'ok' ? check.session : null
  const auth = useMemo<AdminAuth | null>(() => (session ? { session, handleAuthError } : null), [session, handleAuthError])

  if (check === null) {
    return <Message>불러오는 중…</Message>
  }
  if (check.kind === 'login') {
    return <Navigate to="/admin/login" replace />
  }
  if (check.kind === 'forbidden') {
    return <Forbidden />
  }
  if (check.kind === 'error' || !auth) {
    return <Message>로그인 상태를 확인하지 못했습니다. 잠시 뒤 다시 시도해 주세요.</Message>
  }

  return (
    <AdminAuthContext value={auth}>
      <div className="min-h-screen bg-slate-50 text-slate-900">
        <header className="border-b border-slate-200 bg-white">
          <div className="mx-auto flex max-w-6xl flex-wrap items-center gap-x-6 gap-y-2 px-4 py-3">
            <span className="font-bold tracking-tight">KEDP 관리자</span>
            <nav className="flex gap-4 text-sm">
              <MenuLink to="/admin/dashboard">Dashboard 관리</MenuLink>
              <MenuLink to="/admin/api">API 관리</MenuLink>
            </nav>
            <div className="ml-auto flex items-center gap-3 text-sm text-slate-500">
              <span>{auth.session.account}</span>
              <LogoutButton />
            </div>
          </div>
        </header>
        <main className="mx-auto max-w-6xl px-4 py-6">
          <Outlet />
        </main>
      </div>
    </AdminAuthContext>
  )
}

function MenuLink({ to, children }: { to: string; children: string }) {
  return (
    <NavLink
      to={to}
      className={({ isActive }) =>
        isActive ? 'font-semibold text-blue-700' : 'text-slate-600 hover:text-slate-900 hover:underline'
      }
    >
      {children}
    </NavLink>
  )
}

/** 로그아웃(204). 서버가 cookie를 지운다. 실패해도 로그인 화면으로 간다. */
function LogoutButton() {
  const navigate = useNavigate()
  const [busy, setBusy] = useState(false)
  return (
    <button
      type="button"
      disabled={busy}
      className={buttonClass}
      onClick={() => {
        setBusy(true)
        adminRequest<void>('POST', logoutPath)
          .catch(() => undefined)
          .finally(() => navigate('/admin/login', { replace: true }))
      }}
    >
      로그아웃
    </button>
  )
}
